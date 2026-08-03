package helpers

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"text/template"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// ResolveTemplate renders a Go text template with the provided variables.
// It exposes User, Flags, System, and UserDefined variable maps to the template.
// A reference to a variable that does not exist is an error.
func ResolveTemplate(templateData []byte, variables types.Variables) ([]byte, error) {
	return resolveTemplate(templateData, variables, "error")
}

// ResolveTemplateForValidation renders a template with missing keys left empty.
//
// `rwr validate` cannot know which RWR_* variables the operator will export when
// they run, so a reference to one is not a defect it can report. A run has no such
// excuse — by then the value really is absent and would be written to disk — which
// is why ResolveTemplate is strict and this is not.
func ResolveTemplateForValidation(templateData []byte, variables types.Variables) ([]byte, error) {
	rendered, err := resolveTemplate(templateData, variables, "zero")
	if err != nil {
		return nil, err
	}
	// missingkey=zero on a map[string]interface{} yields a nil interface, which
	// text/template prints as "<no value>". Validation is checking structure, and
	// "<no value>" in the middle of a path breaks nothing here — but leaving it in
	// would put the very string this change exists to eliminate back into the
	// document being checked.
	return bytes.ReplaceAll(rendered, []byte("<no value>"), nil), nil
}

func resolveTemplate(templateData []byte, variables types.Variables, missingKey string) ([]byte, error) {
	templateString := string(templateData)
	t, err := template.New("template").Option("missingkey=" + missingKey).Parse(templateString)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	data := make(map[string]interface{})
	data["User"] = variables.User.ToMap()
	data["Flags"] = variables.Flags.ToMap()
	data["System"] = variables.System.ToMap()
	data["UserDefined"] = variables.UserDefined

	// data["Flags"] no longer carries credentials (see types.Flags.ToMap), so this
	// dump is safe; it is the reason they must stay out of that map. Credentials
	// is added after the dump: it holds the exposeCredentials-gated values, so
	// only the names may reach the log.
	log.Debugf("Template variables: %+v (credentials in scope: %v)", data, types.TemplateCredentialNames())
	data["Credentials"] = types.TemplateCredentials()

	// Execute with the same option the template was parsed with.
	//
	// This used to parse with missingkey=error and then execute with
	// missingkey=invalid, so a reference to a variable that does not exist rendered
	// the literal string "<no value>" and the run carried on. That string then
	// became a file path, a package name, or a service name — rwr wrote
	// "<no value>/.vimrc" rather than telling anybody the variable was missing.
	var renderedTemplate bytes.Buffer
	err = t.Option("missingkey="+missingKey).Execute(&renderedTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return renderedTemplate.Bytes(), nil
}

// builtinRefPattern matches simple references into the fixed template
// namespaces — {{ .User.home }}, {{ if .System.os }} — whose keys are known at
// validate time, unlike UserDefined's.
var builtinRefPattern = regexp.MustCompile(`\.(User|System|Flags)\.([A-Za-z0-9_]+)`)

// UnknownTemplateReferences reports references into the User, System, and
// Flags namespaces that name no existing key. UserDefined is deliberately
// exempt: its values vary per machine, so a reference to one is not a defect
// validate can report. The fixed namespaces have no such excuse — validate
// used to render every namespace with missingkey=zero, so a typo like
// {{ .User.hoem }} validated clean and failed at run time.
func UnknownTemplateReferences(templateData []byte, variables types.Variables) []string {
	namespaces := map[string]map[string]interface{}{
		"User":   variables.User.ToMap(),
		"System": variables.System.ToMap(),
		"Flags":  variables.Flags.ToMap(),
	}

	var unknown []string
	seen := map[string]bool{}
	for _, match := range builtinRefPattern.FindAllSubmatch(templateData, -1) {
		namespace, key := string(match[1]), string(match[2])
		ref := "." + namespace + "." + key
		if seen[ref] {
			continue
		}
		seen[ref] = true
		if _, ok := namespaces[namespace][key]; !ok {
			unknown = append(unknown, ref)
		}
	}
	sort.Strings(unknown)
	return unknown
}
