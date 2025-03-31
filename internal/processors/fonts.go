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

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/ulikunitz/xz"
)

const (
	nerdFontRepoAPI = "https://api.github.com/repos/ryanoasis/nerd-fonts/releases/latest"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestReleaseURL() (string, error) {
	resp, err := http.Get(nerdFontRepoAPI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://github.com/ryanoasis/nerd-fonts/releases/download/%s/", release.TagName), nil
}

func ProcessFonts(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var fontsData types.FontsData
	var err error

	log.Debug("Processing fonts from blueprint")

	err = helpers.UnmarshalBlueprint(blueprintData, format, &fontsData)
	if err != nil {
		return fmt.Errorf("error unmarshaling fonts blueprint data: %w", err)
	}

	log.Debugf("Found %d font entries to process", len(fontsData.Fonts))

	releaseURL, err := getLatestReleaseURL()
	if err != nil {
		return fmt.Errorf("error getting latest release URL: %w", err)
	}

	for _, font := range fontsData.Fonts {
		if len(font.Names) > 0 {
			for _, name := range font.Names {
				fontWithName := font
				fontWithName.Name = name
				if err := processFont(fontWithName, osInfo, releaseURL); err != nil {
					return fmt.Errorf("error processing font %s: %w", name, err)
				}
			}
		} else if font.Name != "" {
			if err := processFont(font, osInfo, releaseURL); err != nil {
				return fmt.Errorf("error processing font %s: %w", font.Name, err)
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

	fontURL := getFontURL(font, releaseURL)
	log.Debugf("Font URL: %s", fontURL)

	tempDir, err := os.MkdirTemp("", "font-download-")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tarballPath := filepath.Join(tempDir, font.Name+".tar.xz")
	err = downloadFontTarball(fontURL, tarballPath)
	if err != nil {
		return fmt.Errorf("error downloading font tarball: %v", err)
	}

	fontDir := getFontDirectory(font.Location, osInfo)
	err = extractFontTarball(tarballPath, fontDir, osInfo)
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

func getFontURL(font types.Font, releaseURL string) string {
	return fmt.Sprintf("%s%s.tar.xz", releaseURL, font.Name)
}

func downloadFontTarball(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractFontTarball(tarballPath, destDir string, osInfo *types.OSInfo) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

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

		if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, ".ttf") {
			targetPath := filepath.Join(destDir, header.Name)

			// Create a temporary file for the extracted font
			tempFile, err := os.CreateTemp("", "font-")
			if err != nil {
				return err
			}
			tempFile.Close()

			// Write the font data to the temporary file
			tempFile, err = os.OpenFile(tempFile.Name(), os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tempFile, tr); err != nil {
				tempFile.Close()
				return err
			}
			tempFile.Close()

			if tempFile == nil || targetPath == "" {
				return fmt.Errorf("invalid arguments: tempFile or targetPath is nil/empty")
			}
			err = system.CopyFile(tempFile.Name(), targetPath, true, osInfo)
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
