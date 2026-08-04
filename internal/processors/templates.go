// The templates section of the files processor: rendering a source
// through the template engine into a target, with the credential-aware
// default mode.

package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func processTemplates(templates []types.File, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig, track *progress) error {
	log.Info("Starting to process templates")

	total := 0
	for _, tmpl := range templates {
		if len(tmpl.Names) > 0 {
			total += len(tmpl.Names)
		} else if tmpl.Name != "" {
			total++
		}
	}
	track.expect("", total)

	run := func(tmpl types.File) {
		started := time.Now()
		switch err := processTemplate(tmpl, blueprintDir, osInfo, initConfig); {
		case err != nil:
			recordFailure("templates", tmpl.Name, err)
			track.item("", tmpl.Name, "template", types.StatusFailed, err.Error(), time.Since(started))
		case system.IsDryRun():
			track.item("", tmpl.Name, "template", types.StatusPlanned, "dry-run", 0)
		case tmpl.Source == "" || tmpl.Target == "":
			// processTemplate warns and returns nil for these; mirror that as a
			// skip rather than a success.
			track.item("", tmpl.Name, "template", types.StatusSkipped, "missing required fields", 0)
		default:
			track.itemIdentity("", tmpl.Name, "template", types.StatusOK, "", time.Since(started), fileJournalIdentity(tmpl, blueprintDir))
		}
	}

	for i, tmpl := range templates {
		log.Debugf("Processing template %d: %+v", i, tmpl)
		if tmpl.Name == "" && len(tmpl.Names) == 0 {
			log.Warn("Skipping empty template")
			continue
		}
		if len(tmpl.Names) > 0 {
			log.Debugf("Template has multiple names: %v", tmpl.Names)
			for _, name := range tmpl.Names {
				log.Infof("Processing template with name: %s", name)
				fileWithName := tmpl
				fileWithName.Name = name
				run(fileWithName)
			}
		} else {
			log.Infof("Processing single template: %s", tmpl.Name)
			run(tmpl)
		}
	}
	log.Info("Finished processing all templates")
	return nil
}

func processTemplate(template types.File, blueprintDir string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	log.Infof("Processing template: %s", template.Name)

	if template.Name == "" || template.Source == "" || template.Target == "" {
		log.Warnf("Skipping template with missing required fields: %+v", template)
		return nil
	}

	sourcePath := filepath.Join(blueprintDir, template.Source, template.Name)
	log.Debugf("Full source path: %s", sourcePath)

	content, err := os.ReadFile(sourcePath) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		log.Errorf("Error reading template file %s: %v", sourcePath, err)
		return fmt.Errorf("error reading template file %s: %w", sourcePath, err)
	}
	log.Debugf("Successfully read template file, content length: %d bytes", len(content))

	log.Debug("Resolving template variables")
	// Copying the struct still shares the UserDefined map, so writing template
	// overrides into it mutated initConfig for every later template and
	// blueprint - a per-template variable permanently clobbered a run-wide one
	// of the same name. Merge into a fresh map instead.
	mergedVariables := initConfig.Variables
	mergedVariables.UserDefined = make(map[string]interface{}, len(initConfig.Variables.UserDefined)+len(template.Variables))
	for k, v := range initConfig.Variables.UserDefined {
		mergedVariables.UserDefined[k] = v
	}
	for k, v := range template.Variables {
		mergedVariables.UserDefined[k] = v
	}
	resolvedContent, err := helpers.ResolveTemplate(content, mergedVariables)
	if err != nil {
		log.Errorf("Error resolving template %s: %v", sourcePath, err)
		return fmt.Errorf("error resolving template %s: %w", sourcePath, err)
	}
	log.Debugf("Successfully resolved template, new content length: %d bytes", len(resolvedContent))

	mode := template.Mode
	if !mode.IsSet() {
		mode = types.FileMode(defaultTemplateMode)
	}

	// Create a File struct from the Template
	file := types.File{
		Name:     template.Name,
		Action:   template.Action,
		Content:  string(resolvedContent),
		Target:   template.Target,
		Owner:    template.Owner,
		Group:    template.Group,
		Mode:     mode,
		Elevated: template.Elevated,
	}

	// Process the template as a file
	err = processFile(file, blueprintDir, osInfo)
	if err != nil {
		log.Errorf("Error processing template as file %s: %v", template.Name, err)
		return fmt.Errorf("error processing template as file %s: %w", template.Name, err)
	}

	log.Infof("Template processed successfully: %s", template.Name)
	return nil
}

// processTemplateImports takes the imported file's `templates:` list. Taking its
// `files:` list instead loses every imported template without an error.
func processTemplateImports(items []types.File, blueprintDir string, format string, treeVersion int) ([]types.File, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.File) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.File, error) {
			var d types.FileData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeFiles, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Templates, nil
		}, format)
}
