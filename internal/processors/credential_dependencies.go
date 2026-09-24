package processors

import (
	"fmt"
	"reflect"
	"strings"

	"charm.land/log/v2"
	"github.com/freehold-digital/rwr/internal/credentials"
	"github.com/freehold-digital/rwr/internal/helpers"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func credentialResolver(c *types.InitConfig) (*credentials.Resolver, func()) {
	if resolver, ok := c.CredentialRuntime.(*credentials.Resolver); ok {
		return resolver, func() {}
	}
	r := credentials.NewResolver(c)
	return r, r.Close
}

// prepareCredentialResource is called after profiles/imports, before the one
// resource runs. Its cleanup restores the previous exposure scope even on error.
func prepareCredentialResource(resource any, deps types.CredentialDependencies, c *types.InitConfig, processor, name, action string, track *progress) (func(), bool) {

	names := append([]string{}, deps.RequiresCredentials...)
	walkStrings(reflect.ValueOf(resource), func(value string) string {
		for _, m := range helpers.CredentialTokenPattern.FindAllStringSubmatch(value, -1) {
			names = append(names, m[1])
		}
		return value
	})
	if err := validateCredentialResourceFields(reflect.ValueOf(resource), ""); err != nil {
		recordFailure(processor, name, err)
		track.item("", name, action, types.StatusFailed, err.Error(), 0)
		return func() {}, false
	}
	for _, credentialName := range names {
		if !types.IsManagedCredential(credentialName) {
			err := fmt.Errorf("undeclared credential %q", credentialName)
			recordFailure(processor, name, err)
			track.item("", name, action, types.StatusFailed, err.Error(), 0)
			return func() {}, false
		}
	}
	if deps.OnCredentialUnavailable != "" && deps.OnCredentialUnavailable != "skip" && deps.OnCredentialUnavailable != "fail" {
		err := fmt.Errorf("onCredentialUnavailable must be skip or fail")
		recordFailure(processor, name, err)
		track.item("", name, action, types.StatusFailed, err.Error(), 0)
		return func() {}, false
	}
	if system.IsDryRun() {
		return func() {}, true
	}
	values := map[string]string{}
	r, closeResolver := credentialResolver(c)
	defer closeResolver()
	policy := deps.OnCredentialUnavailable
	if policy == "" {
		policy = c.CredentialPolicy.OnUnavailable
	}
	for _, name := range names {
		if _, ok := values[name]; ok {
			continue
		}
		if !types.IsManagedCredential(name) {
			err := fmt.Errorf("undeclared credential %q", name)
			recordFailure(processor, name, err)
			track.item("", name, action, types.StatusFailed, err.Error(), 0)
			return func() {}, false
		}
		// Explicit scopes are an access boundary, independent of exposure.
		scope := types.CredentialScope(name)
		allowed := len(scope) == 0
		for _, s := range scope {
			if s == processor || processor == types.BlueprintTypeFiles && s == types.CredentialSurfaceTemplates {
				allowed = true
			}
		}
		if !allowed {
			err := fmt.Errorf("credential %q does not allow processor %s", name, processor)
			recordFailure(processor, name, err)
			track.item("", name, action, types.StatusFailed, err.Error(), 0)
			return func() {}, false
		}
		value, err := r.Read(system.RunContext(), name)
		if err != nil {
			reason := fmt.Sprintf("credential %q unavailable; run rwr run credentials when ready", name)
			status := types.StatusSkipped
			if policy == "fail" || !credentials.IsUnavailable(err) {
				status = types.StatusFailed
				recordFailure(processor, name, fmt.Errorf("%s", reason))
			}
			log.Warn(reason)
			track.item("", name, action, status, reason, 0)
			return func() {}, false
		}
		values[name] = value
	}
	if file, ok := resource.(*types.File); ok && helpers.CredentialTokenPattern.MatchString(file.Content) {
		if file.Mode == 0 {
			file.Mode = 0600
		}
		if uint32(file.Mode)&0077 != 0 {
			err := fmt.Errorf("credential-bearing file requires owner-only permissions")
			recordFailure(processor, name, err)
			track.item("", name, action, types.StatusFailed, err.Error(), 0)
			return func() {}, false
		}
	}
	restore := types.ScopeCredentialValues(values)
	var renderErr error
	walkStrings(reflect.ValueOf(resource), func(value string) string {
		return helpers.CredentialTokenPattern.ReplaceAllStringFunc(value, func(token string) string {
			name := helpers.CredentialTokenPattern.FindStringSubmatch(token)[1]
			if _, ok := types.TemplateCredentials()[name]; !ok {
				renderErr = fmt.Errorf("credential %q is not exposed to templates", name)
				return token
			}
			return values[name]
		})
	})
	if renderErr != nil {
		restore()
		recordFailure(processor, name, renderErr)
		track.item("", name, action, types.StatusFailed, renderErr.Error(), 0)
		return func() {}, false
	}
	return restore, true
}

func walkStrings(v reflect.Value, transform func(string) string) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walkStrings(v.Elem(), transform)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			walkStrings(v.Field(i), transform)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			walkStrings(iter.Value(), transform)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkStrings(v.Index(i), transform)
		}
	case reflect.String:
		result := transform(v.String())
		if v.CanSet() && strings.Compare(result, v.String()) != 0 {
			v.SetString(result)
		}
	}
}

func validateCredentialResourceFields(v reflect.Value, field string) error {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return validateCredentialResourceFields(v.Elem(), field)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := validateCredentialResourceFields(v.Field(i), v.Type().Field(i).Name); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := validateCredentialResourceFields(iter.Value(), field); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := validateCredentialResourceFields(v.Index(i), field); err != nil {
				return err
			}
		}
	case reflect.String:
		if helpers.CredentialTokenPattern.MatchString(v.String()) && field != "Content" {
			return fmt.Errorf("credential template is not permitted in %s; use a declared dependency and child environment", field)
		}
	}
	return nil
}
