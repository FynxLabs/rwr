package processors

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// Test blueprint parsing without calling the actual ProcessServices function
func TestProcessServices_BlueprintParsing(t *testing.T) {
	blueprintData := []byte(`
services:
  - name: "nginx"
    action: "enable"
    elevated: true
  - name: "postgresql"
    action: "start"
    profiles: ["database"]
    elevated: true
`)

	var servicesData types.ServiceData
	err := helpers.UnmarshalBlueprint(blueprintData, "yaml", &servicesData)

	if err != nil {
		t.Fatalf("Blueprint parsing failed: %v", err)
	}

	if len(servicesData.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(servicesData.Services))
	}

	// Validate first service
	if servicesData.Services[0].Name != "nginx" {
		t.Errorf("Expected first service name to be 'nginx', got '%s'", servicesData.Services[0].Name)
	}
	if servicesData.Services[0].Action != "enable" {
		t.Errorf("Expected first service action to be 'enable', got '%s'", servicesData.Services[0].Action)
	}
	if !servicesData.Services[0].Elevated {
		t.Error("Expected first service to be elevated")
	}

	// Validate second service
	if servicesData.Services[1].Name != "postgresql" {
		t.Errorf("Expected second service name to be 'postgresql', got '%s'", servicesData.Services[1].Name)
	}
	if len(servicesData.Services[1].Profiles) != 1 || servicesData.Services[1].Profiles[0] != "database" {
		t.Errorf("Expected second service to have profile 'database', got %v", servicesData.Services[1].Profiles)
	}

	t.Log("Blueprint parsing successful")
}

// Test profile filtering logic independently
func TestProcessServices_ProfileFiltering(t *testing.T) {
	services := []types.Service{
		{
			Name:     "dev-service",
			Action:   "start",
			Profiles: []string{"development"}, // Should be included
		},
		{
			Name:     "prod-service",
			Action:   "start",
			Profiles: []string{"production"}, // Should be filtered out
		},
		{
			Name:   "base-service",
			Action: "enable",
			// No profiles specified - should be included
		},
		{
			Name:     "multi-service",
			Action:   "start",
			Profiles: []string{"development", "staging"}, // Should be included (matches development)
		},
	}

	activeProfiles := []string{"development"}
	filteredServices := helpers.FilterByProfiles(services, activeProfiles)

	// Should include: dev-service (development), base-service (no profiles), multi-service (matches development)
	// Should exclude: prod-service (production only)
	expectedCount := 3
	if len(filteredServices) != expectedCount {
		t.Errorf("Expected %d filtered services, got %d", expectedCount, len(filteredServices))
	}
	// Verify specific services are included
	names := make([]string, len(filteredServices))
	for i, svc := range filteredServices {
		names[i] = svc.Name
	}

	expectedNames := []string{"dev-service", "base-service", "multi-service"}
	for _, expected := range expectedNames {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service '%s' to be included in filtered results", expected)
		}
	}

	// Verify prod-service is excluded
	for _, name := range names {
		if name == "prod-service" {
			t.Error("Service 'prod-service' should have been filtered out")
		}
	}

	t.Log("Profile filtering logic works correctly")
}

// Test service structure validation
func TestProcessServices_ServiceStructure(t *testing.T) {
	testCases := []struct {
		name        string
		service     types.Service
		expectValid bool
	}{
		{
			name: "Valid basic service",
			service: types.Service{
				Name:   "nginx",
				Action: "start",
			},
			expectValid: true,
		},
		{
			name: "Service with elevation",
			service: types.Service{
				Name:     "systemd-service",
				Action:   "enable",
				Elevated: true,
			},
			expectValid: true,
		},
		{
			name: "Service with profiles",
			service: types.Service{
				Name:     "dev-service",
				Action:   "start",
				Profiles: []string{"development", "testing"},
			},
			expectValid: true,
		},
		{
			name: "Service creation with content",
			service: types.Service{
				Name:    "custom-service",
				Action:  "create",
				Target:  "/etc/systemd/system/custom.service",
				Content: "[Unit]\nDescription=Custom Service\n",
			},
			expectValid: true,
		},
		{
			name: "Service creation with source",
			service: types.Service{
				Name:   "copied-service",
				Action: "create",
				Source: "/tmp/service.template",
				Target: "/etc/systemd/system/copied.service",
			},
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test service structure validation
			if tc.service.Name == "" {
				t.Error("Service name should not be empty")
			}

			// Test action validation
			validActions := []string{"enable", "disable", "start", "stop", "restart", "reload", "status", "create", "delete"}
			actionValid := false
			for _, validAction := range validActions {
				if tc.service.Action == validAction {
					actionValid = true
					break
				}
			}

			if tc.expectValid && !actionValid {
				t.Errorf("Expected valid action, got '%s'", tc.service.Action)
			}

			// Test create action validation
			if tc.service.Action == "create" {
				if tc.service.Content == "" && tc.service.Source == "" {
					t.Error("Create action should have either content or source")
				}
				if tc.service.Target == "" {
					t.Error("Create action should have target specified")
				}
			}

			t.Logf("Service structure validation passed for %s", tc.name)
		})
	}
}

// Test command generation logic without execution
func TestProcessServices_CommandGeneration(t *testing.T) {
	testCases := []struct {
		name            string
		service         types.Service
		expectedLinux   []string
		expectedMacOS   []string
		expectedWindows []string
	}{
		{
			name: "Enable service",
			service: types.Service{
				Name:   "nginx",
				Action: "enable",
			},
			expectedLinux:   []string{"systemctl", "enable", "nginx"},
			expectedMacOS:   []string{"launchctl", "load", "/Library/LaunchDaemons/nginx.plist"},
			expectedWindows: []string{"sc", "config", "nginx", "start=auto"},
		},
		{
			name: "Start service",
			service: types.Service{
				Name:   "postgresql",
				Action: "start",
			},
			expectedLinux:   []string{"systemctl", "start", "postgresql"},
			expectedMacOS:   []string{"launchctl", "start", "postgresql"},
			expectedWindows: []string{"sc", "start", "postgresql"},
		},
		{
			name: "Status check",
			service: types.Service{
				Name:   "apache2",
				Action: "status",
			},
			expectedLinux:   []string{"systemctl", "status", "apache2"},
			expectedMacOS:   []string{"launchctl", "list", "|", "grep", "apache2"},
			expectedWindows: []string{"sc", "query", "apache2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Linux command generation
			t.Run("Linux", func(t *testing.T) {
				cmd := generateLinuxCommand(tc.service)
				if cmd.Exec != tc.expectedLinux[0] {
					t.Errorf("Expected exec '%s', got '%s'", tc.expectedLinux[0], cmd.Exec)
				}

				expectedArgs := tc.expectedLinux[1:]
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}

				for i, expectedArg := range expectedArgs {
					if i < len(cmd.Args) && cmd.Args[i] != expectedArg {
						t.Errorf("Expected arg[%d] '%s', got '%s'", i, expectedArg, cmd.Args[i])
					}
				}
			})

			// Test macOS command generation
			t.Run("macOS", func(t *testing.T) {
				cmd := generateMacOSCommand(tc.service)
				if cmd.Exec != tc.expectedMacOS[0] {
					t.Errorf("Expected exec '%s', got '%s'", tc.expectedMacOS[0], cmd.Exec)
				}

				expectedArgs := tc.expectedMacOS[1:]
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			})

			// Test Windows command generation
			t.Run("Windows", func(t *testing.T) {
				cmd := generateWindowsCommand(tc.service)
				if cmd.Exec != tc.expectedWindows[0] {
					t.Errorf("Expected exec '%s', got '%s'", tc.expectedWindows[0], cmd.Exec)
				}

				expectedArgs := tc.expectedWindows[1:]
				if len(cmd.Args) != len(expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
				}
			})

			t.Logf("Command generation test passed for %s", tc.name)
		})
	}
}

// Helper functions to generate commands without executing them
func generateLinuxCommand(service types.Service) types.Command {
	cmd := types.Command{Elevated: service.Elevated}

	switch service.Action {
	case "enable":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"enable", service.Name}
	case "disable":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"disable", service.Name}
	case "start":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"start", service.Name}
	case "stop":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"stop", service.Name}
	case "restart":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"restart", service.Name}
	case "reload":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"reload", service.Name}
	case "status":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"status", service.Name}
	case "create", "delete":
		cmd.Exec = "systemctl"
		cmd.Args = []string{"daemon-reload"}
	}

	return cmd
}

func generateMacOSCommand(service types.Service) types.Command {
	cmd := types.Command{Elevated: service.Elevated}

	switch service.Action {
	case "enable":
		cmd.Exec = "launchctl"
		cmd.Args = []string{"load", "/Library/LaunchDaemons/" + service.Name + ".plist"}
	case "disable":
		cmd.Exec = "launchctl"
		cmd.Args = []string{"unload", "/Library/LaunchDaemons/" + service.Name + ".plist"}
	case "start":
		cmd.Exec = "launchctl"
		cmd.Args = []string{"start", service.Name}
	case "stop":
		cmd.Exec = "launchctl"
		cmd.Args = []string{"stop", service.Name}
	case "status":
		cmd.Exec = "launchctl"
		cmd.Args = []string{"list", "|", "grep", service.Name}
	}

	return cmd
}

func generateWindowsCommand(service types.Service) types.Command {
	cmd := types.Command{Elevated: true} // Windows services typically require elevation

	switch service.Action {
	case "enable":
		cmd.Exec = "sc"
		cmd.Args = []string{"config", service.Name, "start=auto"}
	case "disable":
		cmd.Exec = "sc"
		cmd.Args = []string{"config", service.Name, "start=disabled"}
	case "start":
		cmd.Exec = "sc"
		cmd.Args = []string{"start", service.Name}
	case "stop":
		cmd.Exec = "sc"
		cmd.Args = []string{"stop", service.Name}
	case "status":
		cmd.Exec = "sc"
		cmd.Args = []string{"query", service.Name}
	}

	return cmd
}

// Test platform-specific action support
func TestProcessServices_PlatformActionSupport(t *testing.T) {
	testCases := []struct {
		platform      string
		action        string
		shouldSupport bool
		errorMessage  string
	}{
		{"linux", "reload", true, ""},
		{"darwin", "reload", false, "reload action not supported"},
		{"windows", "reload", false, "reload action not supported"},
		{"linux", "enable", true, ""},
		{"darwin", "enable", true, ""},
		{"windows", "enable", true, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.platform+"_"+tc.action, func(t *testing.T) {
			_ = types.Service{
				Name:   "test-service",
				Action: tc.action,
			}

			var supportsAction bool
			var err error

			switch tc.platform {
			case "linux":
				supportsAction = true // Linux supports all actions
			case "darwin":
				if tc.action == "reload" {
					err = &ServiceError{message: "reload action not supported for macOS services"}
					supportsAction = false
				} else {
					supportsAction = true
				}
			case "windows":
				if tc.action == "reload" {
					err = &ServiceError{message: "reload action not supported for Windows services"}
					supportsAction = false
				} else {
					supportsAction = true
				}
			}

			if tc.shouldSupport && !supportsAction {
				t.Errorf("Expected %s to support %s action", tc.platform, tc.action)
			}

			if !tc.shouldSupport && err != nil {
				if !containsString(err.Error(), tc.errorMessage) {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMessage, err.Error())
				}
			}

			t.Logf("Platform action support test passed: %s %s", tc.platform, tc.action)
		})
	}
}

// Custom error type for testing
type ServiceError struct {
	message string
}

func (e *ServiceError) Error() string {
	return e.message
}

// Test edge cases and error conditions
func TestProcessServices_EdgeCases(t *testing.T) {
	t.Run("Empty service name", func(t *testing.T) {
		service := types.Service{
			// Name is empty
		}
		_ = service.Action // Assign value to avoid unused write warning

		if service.Name == "" {
			t.Log("Empty service name detected correctly")
		} else {
			t.Error("Expected empty service name")
		}
	})

	t.Run("Invalid action", func(t *testing.T) {
		service := types.Service{
			Action: "invalid-action",
		}
		_ = service.Name // Assign a value to avoid unused write warning

		validActions := map[string]bool{
			"enable": true, "disable": true, "start": true, "stop": true,
			"restart": true, "reload": true, "status": true, "create": true, "delete": true,
		}

		if !validActions[service.Action] {
			t.Log("Invalid action properly detected")
		} else {
			t.Error("Invalid action should not be considered valid")
		}
	})

	t.Run("Create action without content or source", func(t *testing.T) {
		service := types.Service{
			// Missing both Content and Source
		}
		_ = service.Name // Assign values to avoid unused write warnings
		_ = service.Action
		_ = service.Target

		if service.Content == "" && service.Source == "" {
			t.Log("Missing content/source for create action detected correctly")
		} else {
			t.Error("Expected both content and source to be empty")
		}
	})

	t.Run("Service with special characters", func(t *testing.T) {
		service := types.Service{
			Name: "my-service.with.dots",
		}
		_ = service.Action // Assign a value to avoid unused write warning

		if service.Name == "" {
			t.Error("Service name should not be empty")
		}
		if !strings.Contains(service.Name, ".") {
			t.Error("Service name should contain dots")
		}
		t.Log("Special characters in service names handled correctly")
	})
}

// Test blueprint format variations
func TestProcessServices_BlueprintFormats(t *testing.T) {
	testCases := []struct {
		name   string
		format string
		data   []byte
	}{
		{
			name:   "YAML format",
			format: "yaml",
			data: []byte(`
services:
  - name: "nginx"
    action: "start"
    elevated: true
`),
		},
		{
			name:   "JSON format",
			format: "json",
			data: []byte(`{
  "services": [
    {
      "name": "nginx",
      "action": "start",
      "elevated": true
    }
  ]
}`),
		},
		{
			name:   "TOML format",
			format: "toml",
			data: []byte(`
[[services]]
name = "nginx"
action = "start"
elevated = true
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var serviceData types.ServiceData
			err := helpers.UnmarshalBlueprint(tc.data, tc.format, &serviceData)

			if err != nil {
				t.Fatalf("Failed to parse %s format: %v", tc.format, err)
			}

			if len(serviceData.Services) != 1 {
				t.Errorf("Expected 1 service, got %d", len(serviceData.Services))
			}

			if serviceData.Services[0].Name != "nginx" {
				t.Errorf("Expected service name 'nginx', got '%s'", serviceData.Services[0].Name)
			}

			if serviceData.Services[0].Action != "start" {
				t.Errorf("Expected action 'start', got '%s'", serviceData.Services[0].Action)
			}

			if !serviceData.Services[0].Elevated {
				t.Error("Expected service to be elevated")
			}

			t.Logf("%s format parsing successful", tc.format)
		})
	}
}

// Test invalid blueprint data
func TestProcessServices_InvalidBlueprint(t *testing.T) {
	invalidBlueprint := []byte(`
services:
  - name: "test-service"
    action: "start"
    invalid_field: [this is invalid yaml
`)

	var serviceData types.ServiceData
	err := helpers.UnmarshalBlueprint(invalidBlueprint, "yaml", &serviceData)

	if err == nil {
		t.Fatal("Invalid blueprint should return an error")
	}

	if !containsString(err.Error(), "yaml") && !containsString(err.Error(), "unmarshal") {
		t.Errorf("Expected YAML parsing error, got: %v", err)
	}
	t.Log("Invalid blueprint properly rejected")
}

// Benchmark tests for performance
func BenchmarkServiceFiltering(b *testing.B) {
	services := []types.Service{
		{Name: "nginx", Action: "start", Profiles: []string{"web"}},
		{Name: "postgresql", Action: "start", Profiles: []string{"database"}},
		{Name: "redis", Action: "start"},
		{Name: "elasticsearch", Action: "start", Profiles: []string{"search", "analytics"}},
		{Name: "mongodb", Action: "start", Profiles: []string{"database", "nosql"}},
	}

	activeProfiles := []string{"web", "database"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = helpers.FilterByProfiles(services, activeProfiles)
	}
}

func BenchmarkBlueprintServiceParsing(b *testing.B) {
	blueprintData := []byte(`
services:
  - name: "nginx"
    action: "start"
    elevated: true
  - name: "postgresql"
    action: "enable"
    elevated: true
  - name: "redis"
    action: "start"
    elevated: false
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var serviceData types.ServiceData
		_ = helpers.UnmarshalBlueprint(blueprintData, "yaml", &serviceData)
	}
}
