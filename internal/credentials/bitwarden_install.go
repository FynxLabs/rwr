package credentials

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
)

type bitwardenSetup struct {
	skip      bool
	attempted bool
}

func (b *bitwardenSetup) prepare(interactive bool) {
	if b.attempted {
		b.skip = true
		return
	}
	b.attempted = true
	b.skip = true
	if system.IsDryRun() || !interactive || !stdinIsTerminal() {
		log.Warn("Bitwarden is missing; skipping vault credentials for this run")
		return
	}
	if !offerBitwardenInstall() {
		return
	}
	if err := installBitwarden(); err != nil {
		log.Warnf("Could not install Bitwarden: %v; continuing without vault credentials", err)
		return
	}
	if os.Getenv("BW_SESSION") == "" {
		log.Info("Bitwarden installed. Vault credentials are skipped until you log in and unlock this CLI, then export BW_SESSION")
		return
	}
	b.skip = false
}

var offerBitwardenInstall = func() bool {
	install := false
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm().
		Title("Bitwarden isn't installed. Install it now?").
		Description("RWR downloads the official CLI directly. Skip to continue without vault credentials.").
		Affirmative("Install").Negative("Skip").Value(&install)))
	return reporting.WithTerminal(form.Run) == nil && install
}

func bitwardenBinDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "rwr", "bin"), nil
}

// The managed binary is available to both credential resolution and scripts,
// including on later runs; no shell configuration or external installer is needed.
func addBitwardenPath() error {
	dir, err := bitwardenBinDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == dir {
			return nil
		}
	}
	path := os.Getenv("PATH")
	if path != "" {
		path += string(os.PathListSeparator)
	}
	return os.Setenv("PATH", path+dir)
}

type bitwardenRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		Digest string `json:"digest"`
	} `json:"assets"`
}

func bitwardenAssetName(goos, arch, version string) (string, error) {
	platform := map[string]string{"linux": "linux", "darwin": "macos", "windows": "windows"}[goos]
	if platform == "" || (arch != "amd64" && arch != "arm64") || (goos == "windows" && arch != "amd64") {
		return "", fmt.Errorf("no standalone Bitwarden build for %s/%s", goos, arch)
	}
	if arch == "arm64" {
		platform += "-arm64"
	}
	return "bw-" + platform + "-" + version + ".zip", nil
}

// installBitwarden uses Go's HTTP and ZIP support, without curl, unzip, npm,
// or a package manager. GitHub's release digest verifies the downloaded archive.
var installBitwarden = func() error {
	client := system.NewHTTPClient(30 * time.Second)
	resp, err := client.Get("https://api.github.com/repos/bitwarden/clients/releases?per_page=100")
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bitwarden release lookup: %s", resp.Status)
	}
	var releases []bitwardenRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&releases); err != nil {
		return err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease || !strings.HasPrefix(release.Tag, "cli-v") {
			continue
		}
		name, err := bitwardenAssetName(runtime.GOOS, runtime.GOARCH, strings.TrimPrefix(release.Tag, "cli-v"))
		if err != nil {
			return err
		}
		for _, asset := range release.Assets {
			if asset.Name != name {
				continue
			}
			if !strings.HasPrefix(asset.URL, "https://github.com/bitwarden/clients/releases/download/") || !strings.HasPrefix(asset.Digest, "sha256:") {
				return fmt.Errorf("bitwarden release lacks a trusted download URL or SHA-256 digest")
			}
			dir, err := bitwardenBinDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			stage, err := os.MkdirTemp(dir, ".bitwarden-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(stage) //nolint:errcheck
			archive := filepath.Join(stage, "bw.zip")
			log.Info("Downloading Bitwarden CLI")
			if err := system.DownloadFileWithChecksum(asset.URL, archive, false, strings.TrimPrefix(asset.Digest, "sha256:")); err != nil {
				return err
			}
			binary := "bw"
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			if err := extractBitwarden(archive, filepath.Join(stage, binary), binary); err != nil {
				return err
			}
			if err := os.Rename(filepath.Join(stage, binary), filepath.Join(dir, binary)); err != nil {
				return err
			}
			log.Infof("Installed Bitwarden at %s", filepath.Join(dir, binary))
			return addBitwardenPath()
		}
		return fmt.Errorf("bitwarden release %s has no %s", release.Tag, name)
	}
	return fmt.Errorf("no stable Bitwarden CLI release found")
}

func extractBitwarden(archive, destination, binary string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close() //nolint:errcheck
	for _, file := range z.File {
		if file.Name != binary || !file.Mode().IsRegular() {
			continue
		}
		const maxSize = 256 << 20
		if file.UncompressedSize64 > maxSize {
			return fmt.Errorf("bitwarden executable exceeds size limit")
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()                                                              //nolint:errcheck
		dst, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700) // #nosec G304 G302 -- fixed executable name in a private staging directory, never an archive-supplied path; owner execute permission is required
		if err != nil {
			return err
		}
		written, copyErr := io.Copy(dst, io.LimitReader(src, maxSize+1))
		closeErr := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if written > maxSize {
			return fmt.Errorf("bitwarden executable exceeds size limit")
		}
		return closeErr
	}
	return fmt.Errorf("bitwarden archive has no %s executable", binary)
}
