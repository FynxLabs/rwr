package processors

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

func ProcessSSHKeys(blueprintData []byte, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var sshKeyData types.SSHKeyData
	var err error

	log.Debugf("Processing SSH keys from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &sshKeyData)
	if err != nil {
		return fmt.Errorf("error unmarshaling SSH key blueprint: %v", err)
	}

	// Process the SSH keys
	for _, sshKey := range sshKeyData.SSHKeys {
		// Ensure required packages are installed
		err := ensureSSHPackages(osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error ensuring SSH packages: %v", err)
		}

		// Generate SSH key
		err = generateSSHKey(sshKey)
		if err != nil {
			return fmt.Errorf("error generating SSH key: %v", err)
		}

		// Copy public key to GitHub if requested
		if sshKey.CopyToGitHub {
			err = copySSHKeyToGitHub(sshKey, initConfig)
			if err != nil {
				return fmt.Errorf("error copying SSH key to GitHub: %v", err)
			}
		}
	}

	return nil
}

func ensureSSHPackages(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var packages []types.Package

	switch runtime.GOOS {
	case "windows":
		packages = []types.Package{
			{Name: "openssh", Action: "install", PackageManager: "chocolatey"},
		}
	case "darwin":
		packages = []types.Package{
			{Name: "openssh", Action: "install", PackageManager: "brew"},
		}
	default:
		// For Linux, OpenSSH is typically pre-installed
		return nil
	}

	for _, pkg := range packages {
		err := ProcessPackage(pkg, osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error installing SSH package %s: %v", pkg.Name, err)
		}
	}

	return nil
}

func generateSSHKey(sshKey types.SSHKey) error {
	args := []string{
		"-t", sshKey.Type,
		"-C", sshKey.Comment,
		"-f", sshKey.Path,
	}

	if sshKey.NoPassphrase {
		args = append(args, "-N", "")
	}

	cmd := exec.Command("ssh-keygen", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error generating SSH key: %v", err)
	}

	log.Infof("SSH key generated: %s", sshKey.Path)
	return nil
}

func copySSHKeyToGitHub(sshKey types.SSHKey, initConfig *types.InitConfig) error {
	token := initConfig.Variables.Flags.GHAPIToken
	if token == "" {
		return fmt.Errorf("GitHub API token not found")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	publicKeyPath := sshKey.Path + ".pub"
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("error reading public key file: %v", err)
	}

	key := string(publicKeyBytes)
	title := filepath.Base(sshKey.Path)

	_, _, err = client.Users.CreateKey(oauth2.NoContext, &github.Key{
		Title: &title,
		Key:   &key,
	})
	if err != nil {
		return fmt.Errorf("error adding SSH key to GitHub: %v", err)
	}

	log.Infof("SSH public key added to GitHub: %s", title)
	return nil
}
