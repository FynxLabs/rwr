package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/freehold-digital/rwr/internal/types"
)

// ValidateCredentialDependencies checks references without acquiring values.
// Full validation invokes it on all resources; runtime checks after profiles.
func ValidateCredentialDependencies(data []byte, format string, c *types.InitConfig) error {
	return validateCredentialDependencies(data, format, c, false)
}

// ValidateBootstrapCredentials rejects secret use before providers are ready.
func ValidateBootstrapCredentials(data []byte, format string, c *types.InitConfig) error {
	return validateCredentialDependencies(data, format, c, true)
}

func validateCredentialDependencies(data []byte, format string, c *types.InitConfig, bootstrap bool) error {
	known := map[string]bool{"gh_api_token": true, "ssh_private_key": true, "bw_session": true}
	for _, s := range c.Credentials {
		known[s.Name] = true
	}
	names := ReferencedCredentials(data)
	for _, m := range CredentialTokenPattern.FindAllSubmatch(data, -1) {
		names = append(names, string(m[1]))
	}
	if bootstrap && len(names) > 0 {
		return fmt.Errorf("bootstrap cannot use credentials; move credential-dependent resources to a regular blueprint and run credentials setup separately")
	}
	for _, name := range names {
		if !known[name] {
			return fmt.Errorf("undeclared credential %q", name)
		}
	}
	var document map[string]interface{}
	if err := UnmarshalBlueprint(data, format, &document); err != nil {
		return err
	}
	var walk func(interface{}) error
	walk = func(v interface{}) error {
		switch value := v.(type) {
		case map[string]interface{}:
			for key, item := range value {
				if key == "requiresCredentials" {
					names, ok := item.([]interface{})
					if !ok {
						return fmt.Errorf("requiresCredentials must be a list")
					}
					if bootstrap && len(names) > 0 {
						return fmt.Errorf("bootstrap cannot require credentials; move credential-dependent resources to a regular blueprint and run credentials setup separately")
					}
					for _, raw := range names {
						name, ok := raw.(string)
						if !ok || !known[name] {
							return fmt.Errorf("undeclared credential dependency")
						}
					}
				}
				if key == "onCredentialUnavailable" {
					policy, ok := item.(string)
					if !ok || policy != "skip" && policy != "fail" {
						return fmt.Errorf("onCredentialUnavailable must be skip or fail")
					}
				}
				if err := walk(item); err != nil {
					return err
				}
			}
		case []interface{}:
			for _, item := range value {
				if err := walk(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	// Some decoders return []string for a typed collection; normalize all wire
	// formats through JSON so the traversal sees one structural representation.
	jsonData, err := json.Marshal(document)
	if err != nil {
		return err
	}
	if err := UnmarshalBlueprint(jsonData, types.FormatJSON, &document); err != nil {
		return err
	}
	return walk(document)
}
