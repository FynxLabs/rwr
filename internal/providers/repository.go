package provider

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	provider, ok := providers[repo.PackageManager]
	if !ok {
		return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
	}

	var action RepositoryAction
	if repo.Action == "add" {
		action = provider.Repository.Add
	} else if repo.Action == "remove" {
		action = provider.Repository.Remove
	} else {
		return fmt.Errorf("unsupported action: %s", repo.Action)
	}

	// Create template data
	data := struct {
		Name        string
		URL         string
		KeyURL      string
		Arch        string
		Channel     string
		Component   string
		SourcesPath string
		KeyPath     string
		TempKeyPath string
	}{
		Name:        repo.Name,
		URL:         repo.URL,
		KeyURL:      repo.KeyURL,
		Arch:        repo.Arch,
		Channel:     repo.Channel,
		Component:   repo.Component,
		SourcesPath: provider.Repository.Paths.Sources,
		KeyPath:     filepath.Join(provider.Repository.Paths.Keys, repo.Name+".gpg"),
		TempKeyPath: filepath.Join("/tmp", repo.Name+".gpg"),
	}

	// Execute each step
	for _, step := range action.Steps {
		if err := executeStep(step, data, osInfo, initConfig); err != nil {
			return err
		}
	}

	return nil
}

func executeStep(step ActionStep, data interface{}, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	switch step.Action {
	case "download":
		source, err := executeTemplate(step.Source, data)
		if err != nil {
			return err
		}
		dest, err := executeTemplate(step.Dest, data)
		if err != nil {
			return err
		}
		return helpers.DownloadFile(source, dest, true)

	case "command":
		exec, err := executeTemplate(step.Exec, data)
		if err != nil {
			return err
		}
		args := make([]string, len(step.Args))
		for i, arg := range step.Args {
			args[i], err = executeTemplate(arg, data)
			if err != nil {
				return err
			}
		}
		cmd := types.Command{
			Exec:     exec,
			Args:     args,
			Elevated: true,
		}
		return helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	case "write":
		dest, err := executeTemplate(step.Dest, data)
		if err != nil {
			return err
		}
		content, err := executeTemplate(step.Content, data)
		if err != nil {
			return err
		}
		return helpers.WriteToFile(dest, content, true)

	case "remove":
		path, err := executeTemplate(step.Dest, data)
		if err != nil {
			return err
		}
		return os.Remove(path)

	default:
		return fmt.Errorf("unsupported action: %s", step.Action)
	}
}

func executeTemplate(text string, data interface{}) (string, error) {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
