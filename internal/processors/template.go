package processors

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"text/template"

	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessTemplatesFromFile(blueprintFile string, initConfig *types.InitConfig) error {
	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("error reading blueprint file: %v", err)
		return err
	}

	_, err = processTemplates(blueprintData, initConfig)
	if err != nil {
		log.Errorf("error processing templates: %v", err)
		return err
	}

	return nil
}

func ProcessTemplatesFromData(blueprintData []byte, initConfig *types.InitConfig) error {
	_, err := processTemplates(blueprintData, initConfig)
	if err != nil {
		log.Errorf("error processing templates: %v", err)
		return err
	}

	return nil
}

func processTemplates(blueprintData []byte, initConfig *types.InitConfig) ([]byte, error) {
	var templates []types.Template

	// Unmarshal the blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &templates)
	if err != nil {
		log.Errorf("error unmarshaling template blueprint: %v", err)
		return nil, err
	}

	var renderedTemplates []byte

	// Process the templates
	for _, tmpl := range templates {
		renderedTemplate, err := RenderTemplate(tmpl.Source, initConfig.Variables)
		if err != nil {
			log.Errorf("error processing template: %v", err)
			return nil, err
		}

		// Write the rendered template to a file if the target is specified
		if tmpl.Target != "" {
			err = os.WriteFile(tmpl.Target, renderedTemplate, os.FileMode(tmpl.Mode))
			if err != nil {
				log.Errorf("error writing rendered template to file: %v", err)
				return nil, err
			}
		}

		// Append the rendered template data
		renderedTemplates = append(renderedTemplates, renderedTemplate...)
	}
	return renderedTemplates, nil
}

func RenderTemplate(templateFile string, variables types.Variables) ([]byte, error) {
	// Read the template file
	tmplContent, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("error reading template file: %w", err)
	}

	return RenderTemplateString(string(tmplContent), variables)
}

func RenderTemplateString(templateString string, variables types.Variables) ([]byte, error) {
	// Create a new template
	t, err := template.New("template").Parse(templateString)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	// Merge the User, Flags, and UserDefined maps into a single map
	data := make(map[string]interface{})
	for k, v := range variables.User.ToMap() {
		log.Debugf("User: %s: %s", k, v)
		data[k] = v
	}
	for k, v := range variables.Flags.ToMap() {
		log.Debugf("Flags: %s: %s", k, v)
		data[k] = v
	}
	for k, v := range variables.UserDefined {
		log.Debugf("UserDefined: %s: %s", k, v)
		data[k] = v
	}

	// Execute the template
	var renderedTemplate bytes.Buffer
	err = t.Execute(&renderedTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return renderedTemplate.Bytes(), nil
}
