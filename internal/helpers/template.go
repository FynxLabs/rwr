package helpers

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// ResolveTemplate renders a Go text template with the provided variables.
// It exposes User, Flags, System, and UserDefined variable maps to the template.
// Returns an error if the template is invalid or references missing keys.
func ResolveTemplate(templateData []byte, variables types.Variables) ([]byte, error) {
	templateString := string(templateData)
	t, err := template.New("template").Option("missingkey=error").Parse(templateString)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	data := make(map[string]interface{})
	data["User"] = variables.User.ToMap()
	data["Flags"] = variables.Flags.ToMap()
	data["System"] = variables.System.ToMap()
	data["UserDefined"] = variables.UserDefined

	log.Debugf("Template variables: %+v", data)

	var renderedTemplate bytes.Buffer
	err = t.Option("missingkey=invalid").Execute(&renderedTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return renderedTemplate.Bytes(), nil
}
