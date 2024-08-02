package processors

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessFonts(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var fontsData types.FontsData
	var err error

	log.Debug("Processing fonts from blueprint")

	err = helpers.UnmarshalBlueprint(blueprintData, format, &fontsData)
	if err != nil {
		return fmt.Errorf("error unmarshaling fonts blueprint data: %w", err)
	}

	for _, font := range fontsData.Fonts {
		if len(font.Names) > 0 {
			for _, name := range font.Names {
				fontWithName := font
				fontWithName.Name = name
				if err := processFont(fontWithName); err != nil {
					return fmt.Errorf("error processing font %s: %w", name, err)
				}
			}
		} else {
			if err := processFont(font); err != nil {
				return fmt.Errorf("error processing font %s: %w", font.Name, err)
			}
		}
	}

	return nil
}

func processFont(font types.Font) error {
	log.Infof("Processing font: %s", font.Name)

	if font.Provider == "" {
		font.Provider = "nerd"
	}

	switch font.Action {
	case "install":
		return installFont(font)
	case "remove":
		return removeFont(font)
	default:
		return fmt.Errorf("unsupported action for font: %s", font.Action)
	}
}

func installFont(font types.Font) error {
	log.Infof("Installing font: %s", font.Name)

	fontUrl := getFontUrl(font)
	fontData, err := downloadFont(fontUrl)
	if err != nil {
		return fmt.Errorf("error downloading font %s: %v", font.Name, err)
	}

	fontDir := getFontDirectory(font.Location)
	fontPath := filepath.Join(fontDir, font.Name+".ttf")

	if font.Location == "system" {
		if runtime.GOOS == "windows" {
			err = installFontWindowsElevated(fontPath, fontData)
		} else {
			err = installFontUnixElevated(fontPath, fontData)
		}
	} else {
		err = os.MkdirAll(filepath.Dir(fontPath), 0755)
		if err != nil {
			return fmt.Errorf("error creating font directory: %v", err)
		}
		err = os.WriteFile(fontPath, fontData, 0644)
	}

	if err != nil {
		return fmt.Errorf("error writing font file: %v", err)
	}

	if runtime.GOOS == "windows" {
		err = registerFontWindows(fontPath, font.Location == "system")
	} else {
		err = updateFontCache(font.Location == "system")
	}

	if err != nil {
		return fmt.Errorf("error finalizing font installation: %v", err)
	}

	log.Infof("Font %s installed successfully", font.Name)
	return nil
}

func removeFont(font types.Font) error {
	log.Infof("Removing font: %s", font.Name)

	fontDir := getFontDirectory(font.Location)
	fontPath := filepath.Join(fontDir, font.Name+".ttf")

	if runtime.GOOS == "windows" {
		err := unregisterFontWindows(fontPath, font.Location == "system")
		if err != nil {
			return fmt.Errorf("error unregistering font: %v", err)
		}
	}

	if font.Location == "system" {
		if runtime.GOOS == "windows" {
			err := removeFontWindowsElevated(fontPath)
			if err != nil {
				return fmt.Errorf("error removing font file: %v", err)
			}
		} else {
			cmd := exec.Command("sudo", "rm", fontPath)
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("error removing font file: %v", err)
			}
		}
	} else {
		err := os.Remove(fontPath)
		if err != nil {
			return fmt.Errorf("error removing font file: %v", err)
		}
	}

	if runtime.GOOS != "windows" {
		err := updateFontCache(font.Location == "system")
		if err != nil {
			return fmt.Errorf("error updating font cache: %v", err)
		}
	}

	log.Infof("Font %s removed successfully", font.Name)
	return nil
}

func getFontUrl(font types.Font) string {
	baseUrl := "https://github.com/ryanoasis/nerd-fonts/raw/master/patched-fonts"
	fontName := strings.ReplaceAll(font.Name, " ", "%20")
	return fmt.Sprintf("%s/%s/%s.ttf", baseUrl, fontName, fontName)
}

func downloadFont(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func getFontDirectory(location string) string {
	if runtime.GOOS == "windows" {
		if location == "system" {
			return filepath.Join(os.Getenv("WINDIR"), "Fonts")
		}
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts")
	} else if runtime.GOOS == "darwin" {
		if location == "system" {
			return "/Library/Fonts"
		}
		return filepath.Join(os.Getenv("HOME"), "Library", "Fonts")
	} else {
		if location == "system" {
			return "/usr/local/share/fonts"
		}
		return filepath.Join(os.Getenv("HOME"), ".local", "share", "fonts")
	}
}

func updateFontCache(elevated bool) error {
	var cmd *exec.Cmd
	if elevated {
		cmd = exec.Command("sudo", "fc-cache", "-f", "-v")
	} else {
		cmd = exec.Command("fc-cache", "-f", "-v")
	}
	return cmd.Run()
}

func installFontWindowsElevated(fontPath string, fontData []byte) error {
	tempFile, err := os.CreateTemp("", "font-*.ttf")
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(fontData); err != nil {
		return fmt.Errorf("error writing to temp file: %v", err)
	}
	tempFile.Close()

	psCommand := fmt.Sprintf(`
        $fontFile = "%s"
        $destFile = "%s"
        Copy-Item -Path $fontFile -Destination $destFile -Force
    `, tempFile.Name(), fontPath)

	cmd := exec.Command("powershell", "-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList", fmt.Sprintf("-Command %s", psCommand))
	return cmd.Run()
}

func installFontUnixElevated(fontPath string, fontData []byte) error {
	tempFile, err := os.CreateTemp("", "font-*.ttf")
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(fontData); err != nil {
		return fmt.Errorf("error writing to temp file: %v", err)
	}
	tempFile.Close()

	cmd := exec.Command("sudo", "cp", tempFile.Name(), fontPath)
	return cmd.Run()
}

func registerFontWindows(fontPath string, elevated bool) error {
	psCommand := fmt.Sprintf(`
        $fontFamilyName = [System.Drawing.FontFamily]::new("%s").Name
        $shellApp = New-Object -ComObject shell.application
        $fonts = $shellApp.Namespace(0x14)
        $fonts.CopyHere("%s")
        [System.Runtime.Interopservices.Marshal]::ReleaseComObject($shellApp) | Out-Null
        [System.GC]::Collect()
        [System.GC]::WaitForPendingFinalizers()
    `, filepath.Base(fontPath), fontPath)

	var cmd *exec.Cmd
	if elevated {
		cmd = exec.Command("powershell", "-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList", fmt.Sprintf("-Command %s", psCommand))
	} else {
		cmd = exec.Command("powershell", "-Command", psCommand)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error registering font: %v, output: %s", err, string(output))
	}

	return nil
}

func unregisterFontWindows(fontPath string, elevated bool) error {
	psCommand := fmt.Sprintf(`
        $fontFileName = "%s"
        $fontName = [System.IO.Path]::GetFileNameWithoutExtension($fontFileName)
        $fontRegistryPath = "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts"

        # Remove the font from the registry
        Remove-ItemProperty -Path $fontRegistryPath -Name "$fontName (TrueType)" -ErrorAction SilentlyContinue

        # Notify the system about the font change
        $HWND_BROADCAST = [IntPtr]0xffff
        $WM_FONTCHANGE = 0x001D
        $SMTO_ABORTIFHUNG = 0x0002
        Add-Type -TypeDefinition @'
        using System;
        using System.Runtime.InteropServices;
        public class Win32 {
            [DllImport("user32.dll", CharSet = CharSet.Auto)]
            public static extern IntPtr SendMessageTimeout(
                IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam,
                uint fuFlags, uint uTimeout, out IntPtr lpdwResult);
        }
'@
        [IntPtr]$result = [IntPtr]::Zero
        [Win32]::SendMessageTimeout($HWND_BROADCAST, $WM_FONTCHANGE, [IntPtr]::Zero, [IntPtr]::Zero, $SMTO_ABORTIFHUNG, 1000, [ref]$result) | Out-Null
    `, fontPath)

	var cmd *exec.Cmd
	if elevated {
		cmd = exec.Command("powershell", "-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList", fmt.Sprintf("-Command %s", psCommand))
	} else {
		cmd = exec.Command("powershell", "-Command", psCommand)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error unregistering font: %v, output: %s", err, string(output))
	}

	return nil
}

func removeFontWindowsElevated(fontPath string) error {
	psCommand := fmt.Sprintf(`Remove-Item -Path "%s" -Force`, fontPath)
	cmd := exec.Command("powershell", "-Command", "Start-Process", "powershell", "-Verb", "RunAs", "-ArgumentList", fmt.Sprintf("-Command %s", psCommand))
	return cmd.Run()
}
