package processors

//
//import (
//	"bytes"
//	"fmt"
//	"os"
//	"path/filepath"
//	"text/template"
//
//	"github.com/charmbracelet/log"
//	"github.com/fynxlabs/rwr/internal/types"
//
//	"github.com/fynxlabs/rwr/internal/helpers"
//)
//
//func ProcessTemplates(blueprintData []byte, blueprintFile string, blueprintDir string, initConfig *types.InitConfig) error {
//	var err error
//
//	if len(blueprintData) == 0 && blueprintFile != "" {
//		log.Debugf("Processing templates from file: %s", blueprintFile)
//		blueprintData, err = os.ReadFile(blueprintFile)
//		if err != nil {
//			log.Errorf("error reading blueprint file: %v", err)
//			return fmt.Errorf("error reading blueprint file: %w", err)
//		}
//	} else {
//		log.Debugf("Processing templates from data")
//	}
//
//	_, err = processTemplates(blueprintData, blueprintDir, initConfig)
//	if err != nil {
//		log.Errorf("error processing templates: %v", err)
//		return fmt.Errorf("error processing templates: %w", err)
//	}
//
//	return nil
//}
//
//func ProcessTemplatesFromFile(blueprintFile string, blueprintDir string, initConfig *types.InitConfig) error {
//	return ProcessTemplates(nil, blueprintFile, blueprintDir, initConfig)
//}
//
//func ProcessTemplatesFromData(blueprintData []byte, blueprintDir string, initConfig *types.InitConfig) error {
//	return ProcessTemplates(blueprintData, "", blueprintDir, initConfig)
//}
//
//func processTemplates(blueprintData []byte, blueprintDir string, initConfig *types.InitConfig) ([]byte, error) {
//	var templateData types.TemplateData
//	var templates []types.Template
//
//	// Unmarshal the blueprint data
//	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &templateData)
//	if err != nil {
//		log.Errorf("error unmarshaling template blueprint: %v", err)
//		return nil, err
//	}
//
//	log.Debugf("Unmarshaled templates: %+v", templateData.Templates)
//
//	var renderedTemplates []byte
//
//	templates = templateData.Templates
//
//	// Process the templates
//	for _, tmpl := range templates {
//		log.Debugf("Processing template: %s", tmpl.Source)
//		log.Debugf("Template source path (relative to %s): %s", blueprintDir, filepath.Join(blueprintDir, tmpl.Source))
//
//		renderedTemplate, err := RenderTemplate(filepath.Join(blueprintDir, tmpl.Source), initConfig.Variables, tmpl)
//		if err != nil {
//			log.Errorf("error processing template: %v", err)
//			return nil, err
//		}
//
//		log.Debugf("Rendered template: %s", renderedTemplate)
//
//		// Write the rendered template to a file if the target is specified
//		if tmpl.Target != "" {
//			log.Debugf("Writing rendered template to file: %s", tmpl.Target)
//			targetPath := helpers.ExpandPath(tmpl.Target)
//			err = helpers.WriteToFile(targetPath, string(renderedTemplate), false)
//			if err != nil {
//				log.Errorf("error writing rendered template to file: %v", err)
//				return nil, err
//			}
//		}
//
//		// Append the rendered template data
//		renderedTemplates = append(renderedTemplates, renderedTemplate...)
//	}
//	return renderedTemplates, nil
//}
//
//func RenderTemplate(templateFile string, variables types.Variables, tmpl types.Template) ([]byte, error) {
//	log.Debugf("Rendering template: %s", templateFile)
//
//	// Read the template file
//	tmplContent, err := os.ReadFile(templateFile)
//	if err != nil {
//		return nil, fmt.Errorf("error reading template file: %w", err)
//	}
//
//	return RenderTemplateString(string(tmplContent), variables, tmpl)
//}
//
//func RenderTemplateString(templateString string, variables types.Variables, tmpl types.Template) ([]byte, error) {
//	// Create a new template
//	t, err := template.New("template").Option("missingkey=error").Parse(templateString)
//	if err != nil {
//		return nil, fmt.Errorf("error parsing template: %w", err)
//	}
//
//	// Merge the User, Flags, and UserDefined maps into a single map
//	data := make(map[string]interface{})
//	data["User"] = variables.User.ToMap()
//	data["Flags"] = variables.Flags.ToMap()
//	data["UserDefined"] = make(map[string]interface{})
//
//	// Merge top-level UserDefined variables
//	for k, v := range variables.UserDefined {
//		data["UserDefined"].(map[string]interface{})[k] = v
//	}
//
//	// Add the custom variables from the Template struct to the UserDefined map
//	for k, v := range tmpl.Variables {
//		data["UserDefined"].(map[string]interface{})[k] = v
//	}
//
//	log.Debugf("Data for template execution: %+v", data)
//
//	// Execute the template
//	var renderedTemplate bytes.Buffer
//	err = t.Option("missingkey=invalid").Execute(&renderedTemplate, data)
//	if err != nil {
//		return nil, fmt.Errorf("error executing template: %w", err)
//	}
//
//	log.Debugf("Rendered template: %s", renderedTemplate.String())
//
//	return renderedTemplate.Bytes(), nil
//}
