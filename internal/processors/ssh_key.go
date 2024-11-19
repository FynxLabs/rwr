package processors

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/google/go-github/v66/github"
	"github.com/spf13/viper"
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

	err = processSSHKeys(sshKeyData.SSHKeys, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing SSH Keys: %v", err)
		return fmt.Errorf("error processing SSH Keys: %w", err)
	}

	return nil
}

func processSSHKeys(sshKeys []types.SSHKey, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Process the SSH keys
	for _, sshKey := range sshKeys {
		// Ensure required packages are installed
		err := ensureSSHPackages(osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error ensuring SSH packages: %v", err)
		}

		// Generate SSH key
		keyPath, err := generateSSHKey(sshKey)
		if err != nil {
			log.Errorf("Error generating SSH key %s: %v", sshKey.Name, err)
			continue // Continue with the next key instead of returning
		}

		// Copy public key to GitHub if requested
		if sshKey.CopyToGitHub {
			err = copySSHKeyToGitHub(sshKey, initConfig)
			if err != nil {
				log.Errorf("Error copying SSH key %s to GitHub: %v", sshKey.Name, err)
				continue // Continue with the next key instead of returning
			}
		}

		// Set as RWR SSH Key if requested
		if sshKey.SetAsRWRSSHKey {
			err = setAsRWRSSHKey(keyPath)
			if err != nil {
				log.Errorf("Error setting SSH key %s as RWR SSH Key: %v", sshKey.Name, err)
				continue // Continue with the next key instead of returning
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

func generateSSHKey(sshKey types.SSHKey) (string, error) {
	sshPath := filepath.Join(sshKey.Path, sshKey.Name)

	// Check if the SSH key already exists
	if _, err := os.Stat(sshPath); err == nil {
		log.Warnf("SSH key %s already exists. Skipping generation.", sshPath)
		return sshPath, nil
	}

	args := []string{
		"-t", sshKey.Type,
		"-C", sshKey.Comment,
		"-f", sshPath,
	}

	if sshKey.NoPassphrase {
		args = append(args, "-N", "")
	}

	cmd := types.Command{
		Exec: "ssh-keygen",
		Args: args,
	}

	err := helpers.RunCommand(cmd, true)
	if err != nil {
		return "", fmt.Errorf("error generating SSH key: %v", err)
	}

	log.Infof("SSH key generated: %s", sshPath)
	return sshPath, nil
}

func setAsRWRSSHKey(keyPath string) error {
	// Read the private key file
	privateKey, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("error reading private key file: %v", err)
	}

	// Encode the private key as base64
	encodedKey := base64.StdEncoding.EncodeToString(privateKey)

	// Set the encoded key in Viper configuration
	viper.Set("repository.ssh_private_key", encodedKey)

	// Write the updated configuration to file
	err = viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing updated configuration: %v", err)
	}

	log.Infof("SSH key %s set as RWR SSH Key", keyPath)
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
	tc := oauth2.NewClient(context.TODO(), ts)
	client := github.NewClient(tc)

	sshPath := filepath.Join(sshKey.Path, sshKey.Name)

	publicKeyPath := sshPath + ".pub"
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("error reading public key file: %v", err)
	}

	key := string(publicKeyBytes)

	var title string
	if sshKey.GithubTitle == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("error getting hostname: %v", err)
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter GitHub SSH key title (default: %s): ", hostname)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading user input: %v", err)
		}

		title = strings.TrimSpace(input)
		if title == "" {
			title = hostname
		}
	} else {
		title = sshKey.GithubTitle
	}

	_, _, err = client.Users.CreateKey(context.TODO(), &github.Key{
		Title: &title,
		Key:   &key,
	})
	if err != nil {
		return fmt.Errorf("error adding SSH key to GitHub: %v", err)
	}

	log.Infof("SSH public key added to GitHub: %s", title)
	return nil
}
