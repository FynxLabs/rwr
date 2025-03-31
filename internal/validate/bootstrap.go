package validate

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/types"
)

// ValidateBootstrap validates a bootstrap configuration
func ValidateBootstrap(bootstrap types.BootstrapData, blueprintFile string, results *types.ValidationResults) {
	// Validate packages
	if bootstrap.Packages != nil {
		ValidatePackages(bootstrap.Packages, blueprintFile, results)
	}

	// Validate files
	if bootstrap.Files != nil {
		ValidateFiles(bootstrap.Files, blueprintFile, results)
	}

	// Validate directories
	if bootstrap.Directories != nil {
		for i, dir := range bootstrap.Directories {
			if dir.Name == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'directories[%d].name'", i), blueprintFile, 0, "Add name field to directory")
			}
			if dir.Action == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'directories[%d].action'", i), blueprintFile, 0, "Add action field to directory")
			}
		}
	}

	// Validate git repositories
	if bootstrap.Git != nil {
		ValidateGitRepositories(bootstrap.Git, blueprintFile, results)
	}

	// Validate SSH keys
	if bootstrap.SSHKeys != nil {
		ValidateSSHKeys(bootstrap.SSHKeys, blueprintFile, results)
	}

	// Validate services
	if bootstrap.Services != nil {
		ValidateServices(bootstrap.Services, blueprintFile, results)
	}

	// Validate groups
	if bootstrap.Groups != nil {
		for i, group := range bootstrap.Groups {
			if group.Name == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'groups[%d].name'", i), blueprintFile, 0, "Add name field to group")
			}
		}
	}

	// Validate users
	if bootstrap.Users != nil {
		ValidateUsers(bootstrap.Users, blueprintFile, results)
	}
}
