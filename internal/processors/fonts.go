package processors

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/ulikunitz/xz"
)

// A var so a test can point the release lookup at a server it controls.
var nerdFontRepoAPI = "https://api.github.com/repos/ryanoasis/nerd-fonts/releases/latest"

// maxFontFileBytes caps what a single archive entry may decompress to. The
// largest Nerd Font faces are a few MB; 64MB is comfortably past any real font
// and comfortably short of filling a tmpfs.
const maxFontFileBytes int64 = 64 << 20

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestReleaseURL() (string, error) {
	resp, err := http.Get(nerdFontRepoAPI) // #nosec G107 -- package-level constant URL, a var only so tests can point it at their own server
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://github.com/ryanoasis/nerd-fonts/releases/download/%s/", release.TagName), nil
}

// ProcessFonts downloads and installs Nerd Fonts from the latest GitHub release.
// It extracts font archives to the system font directory and refreshes the font cache.
func ProcessFonts(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var fontsData types.FontsData
	var err error

	log.Debug("Processing fonts from blueprint")

	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeFonts,
		helpers.TreeSchemaVersion(initConfig), &fontsData)
	if err != nil {
		return fmt.Errorf("error unmarshaling fonts blueprint data: %w", err)
	}

	fontsData.Fonts = helpers.FilterByProfiles(fontsData.Fonts, initConfig.Variables.Flags.Profiles)

	log.Debugf("Found %d font entries to process", len(fontsData.Fonts))

	if system.IsDryRun() {
		for _, font := range fontsData.Fonts {
			if len(font.Names) > 0 {
				for _, name := range font.Names {
					log.Infof("[DRY-RUN] Would install font: %s", name)
				}
			} else if font.Name != "" {
				log.Infof("[DRY-RUN] Would install font: %s", font.Name)
			}
		}
		return nil
	}

	if len(fontsData.Fonts) == 0 {
		return nil
	}

	// Resolved after the dry-run and empty-blueprint exits: this is a network call,
	// and `--dry-run` is expected to work offline.
	//
	// GitHub being unreachable fails every font, but not the run: fonts are
	// cosmetic, and the failures reach the exit code through the ledger.
	releaseURL, err := getLatestReleaseURL()
	if err != nil {
		recordFailure("fonts", "nerd-fonts release lookup", err)
		return nil
	}

	// One font failing does not stop the rest: the failure goes to the ledger,
	// which puts it in the run's exit code, and processing continues.
	for _, font := range fontsData.Fonts {
		if len(font.Names) > 0 {
			for _, name := range font.Names {
				fontWithName := font
				fontWithName.Name = name
				if err := processFont(fontWithName, osInfo, releaseURL); err != nil {
					recordFailure("fonts", name, err)
				}
			}
		} else if font.Name != "" {
			if err := processFont(font, osInfo, releaseURL); err != nil {
				recordFailure("fonts", font.Name, err)
			}
		}
	}

	return nil
}

func processFont(font types.Font, osInfo *types.OSInfo, releaseURL string) error {
	log.Debugf("Processing font: %s", font.Name)

	if font.Provider == "" {
		font.Provider = "nerd"
	}

	log.Debugf("Font provider: %s", font.Provider)
	log.Debugf("Font action: %s", font.Action)

	switch font.Action {
	case "install":
		return installFont(font, osInfo, releaseURL)
	case "remove":
		return removeFont(font, osInfo)
	default:
		return fmt.Errorf("unsupported action for font: %s", font.Action)
	}
}

func installFont(font types.Font, osInfo *types.OSInfo, releaseURL string) error {
	log.Infof("Installing font: %s", font.Name)

	if err := validateFontName(font.Name); err != nil {
		return err
	}

	fontURL := getFontURL(font, releaseURL)
	log.Debugf("Font URL: %s", fontURL)

	tempDir, err := os.MkdirTemp("", "font-download-")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	tarballPath := filepath.Join(tempDir, font.Name+".tar.xz")
	err = downloadFontTarball(fontURL, tarballPath)
	if err != nil {
		return fmt.Errorf("error downloading font tarball: %v", err)
	}

	fontDir := getFontDirectory(font.Location, osInfo)
	err = extractFontTarball(tarballPath, fontDir, font.Location == "system", osInfo)
	if err != nil {
		return fmt.Errorf("error extracting font tarball: %v", err)
	}

	if err := updateFontCache(font.Location == "system"); err != nil {
		log.Warnf("Failed to update font cache: %v", err)
	}

	log.Infof("Font %s installed successfully", font.Name)
	return nil
}

func removeFont(font types.Font, osInfo *types.OSInfo) error {
	log.Infof("Removing font: %s", font.Name)

	if err := validateFontName(font.Name); err != nil {
		return err
	}

	fontDir := getFontDirectory(font.Location, osInfo)
	fontPattern := filepath.Join(fontDir, font.Name+"*.ttf")

	matches, err := filepath.Glob(fontPattern)
	if err != nil {
		return fmt.Errorf("error finding font files: %v", err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			log.Warnf("Failed to remove font file %s: %v", match, err)
		}
	}

	if err := updateFontCache(font.Location == "system"); err != nil {
		log.Warnf("Failed to update font cache: %v", err)
	}

	log.Infof("Font %s removed successfully", font.Name)
	return nil
}

// validateFontName rejects names that would escape the release they are meant to
// name. font.Name is blueprint-supplied and is concatenated straight into the
// nerd-fonts download URL and into the local font path, so "../../owner/repo/x"
// would both redirect the download to an attacker-chosen release and, on removal,
// glob outside the font directory.
func validateFontName(name string) error {
	if name == "" {
		return fmt.Errorf("font name is empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid font name %q: must not contain path separators or %q", name, "..")
	}
	return nil
}

func getFontURL(font types.Font, releaseURL string) string {
	return fmt.Sprintf("%s%s.tar.xz", releaseURL, font.Name)
}

func downloadFontTarball(url, filepath string) error {
	if err := system.ValidateDownloadURL(url); err != nil {
		return err
	}
	resp, err := http.Get(url) // #nosec G107 -- scheme restricted by ValidateDownloadURL above
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// Without this the error page body is written out as a .tar.xz and the failure
	// only surfaces later as an unintelligible decompression error.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: unexpected status %s", url, resp.Status)
	}

	out, err := os.Create(filepath) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		return err
	}
	defer out.Close() //nolint:errcheck

	_, err = io.Copy(out, resp.Body)
	return err
}

// resolveTarEntryPath joins a tar entry name onto destDir and refuses anything that
// would land outside it. Archive entry names are attacker-controlled data: a member
// named "../../../etc/cron.d/x" otherwise resolves through filepath.Join to a real
// path outside the font directory, which rwr then writes — as root, for a
// system-scoped install.
func resolveTarEntryPath(destDir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("tar entry has an empty name")
	}
	// Tar always uses forward slashes; check both so a Windows-style name is not
	// waved through on a platform where filepath.Join treats "\" as a separator.
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(cleaned) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("tar entry %q is an absolute path", name)
	}

	targetPath := filepath.Join(destDir, cleaned)
	rel, err := filepath.Rel(destDir, targetPath)
	if err != nil {
		return "", fmt.Errorf("tar entry %q is outside the destination directory: %w", name, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("tar entry %q escapes the destination directory", name)
	}

	return targetPath, nil
}

func extractFontTarball(tarballPath, destDir string, elevated bool, osInfo *types.OSInfo) error {
	file, err := os.Open(tarballPath) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	// Create a new XZ reader
	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzReader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Links are never needed to install a font and are the cheapest way to make a
		// later write land outside destDir, so they are dropped rather than resolved.
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			log.Warnf("Skipping link entry in font archive: %s", header.Name)
			continue
		}

		if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, ".ttf") {
			targetPath, err := resolveTarEntryPath(destDir, header.Name)
			if err != nil {
				return fmt.Errorf("error extracting font archive: %w", err)
			}

			// Create a temporary file for the extracted font
			tempFile, err := os.CreateTemp("", "font-")
			if err != nil {
				return err
			}
			if err := tempFile.Close(); err != nil {
				return fmt.Errorf("error closing temp file: %w", err)
			}

			// Write the font data to the temporary file
			tempFile, err = os.OpenFile(tempFile.Name(), os.O_WRONLY, 0755) // #nosec G302 -- TODO(PR8): create with target mode instead of chmod-after
			if err != nil {
				return err
			}
			// The cap turns a decompression bomb into an error instead of a
			// filled tmpfs: the archive arrives xz-compressed from the network,
			// and a small download can expand to arbitrary size. Real font
			// faces are single-digit MB.
			n, err := io.Copy(tempFile, io.LimitReader(tr, maxFontFileBytes+1))
			if err != nil {
				if closeErr := tempFile.Close(); closeErr != nil {
					return fmt.Errorf("error copying font data: %w (also failed to close: %v)", err, closeErr)
				}
				return err
			}
			if n > maxFontFileBytes {
				if closeErr := tempFile.Close(); closeErr != nil {
					log.Debugf("closing oversized font temp file: %v", closeErr)
				}
				return fmt.Errorf("font file %s decompresses past %d MB; refusing what looks like a decompression bomb", header.Name, maxFontFileBytes>>20)
			}
			if err := tempFile.Close(); err != nil {
				return fmt.Errorf("error closing temp file after copy: %w", err)
			}

			if tempFile == nil || targetPath == "" {
				return fmt.Errorf("invalid arguments: tempFile or targetPath is nil/empty")
			}
			// Elevated only for a system-scoped install; a user font goes under $HOME
			// and needs no privilege.
			err = system.CopyFile(tempFile.Name(), targetPath, elevated, osInfo)
			if err != nil {
				return fmt.Errorf("error copying font file to destination: %v", err)
			}
		}
	}

	return nil
}

func getFontDirectory(location string, osInfo *types.OSInfo) string {
	if location == "system" {
		switch osInfo.System.OS {
		case "linux":
			return "/usr/local/share/fonts"
		case "darwin":
			return "/Library/Fonts"
		case "windows":
			return filepath.Join(os.Getenv("WINDIR"), "Fonts")
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "fonts")
}

func updateFontCache(elevated bool) error {
	cmd := types.Command{
		Exec:     "fc-cache",
		Args:     []string{"-f", "-v"},
		Elevated: elevated,
	}
	return system.RunCommand(cmd, false)
}
