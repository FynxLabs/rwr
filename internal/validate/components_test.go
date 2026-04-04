package validate

import (
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestValidatePackages(t *testing.T) {
	tests := []struct {
		name       string
		packages   []types.Package
		wantErrors int
	}{
		{
			"valid package",
			[]types.Package{{Name: "vim", Action: "install", PackageManager: "apt", Names: []string{"vim"}}},
			0,
		},
		{
			"missing name",
			[]types.Package{{Action: "install", PackageManager: "apt", Names: []string{"vim"}}},
			1,
		},
		{
			"invalid action",
			[]types.Package{{Name: "vim", Action: "destroy", PackageManager: "apt", Names: []string{"vim"}}},
			1,
		},
		{
			"empty packages slice",
			[]types.Package{},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidatePackages(tt.packages, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateRepositories(t *testing.T) {
	tests := []struct {
		name       string
		repos      []types.Repository
		wantErrors int
	}{
		{
			"valid repository",
			[]types.Repository{{Name: "test", PackageManager: "apt", Action: "add", URL: "http://example.com"}},
			0,
		},
		{
			"missing name",
			[]types.Repository{{PackageManager: "apt", Action: "add", URL: "http://example.com"}},
			1,
		},
		{
			"invalid action",
			[]types.Repository{{Name: "test", PackageManager: "apt", Action: "destroy"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateRepositories(tt.repos, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateFiles(t *testing.T) {
	tests := []struct {
		name       string
		files      []types.File
		wantErrors int
	}{
		{
			"valid file",
			[]types.File{{Target: "/tmp/test", Action: "create", Content: "hello"}},
			0,
		},
		{
			"missing target",
			[]types.File{{Action: "create", Content: "hello"}},
			1,
		},
		{
			"invalid action",
			[]types.File{{Target: "/tmp/test", Action: "destroy"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateFiles(tt.files, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateGitRepositories(t *testing.T) {
	tests := []struct {
		name       string
		repos      []types.Git
		wantErrors int
	}{
		{
			"valid git repo",
			[]types.Git{{URL: "https://github.com/test/repo", Path: "/home/user/repo"}},
			0,
		},
		{
			"missing url",
			[]types.Git{{Path: "/home/user/repo"}},
			1,
		},
		{
			"missing path",
			[]types.Git{{URL: "https://github.com/test/repo"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateGitRepositories(tt.repos, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateScripts(t *testing.T) {
	tests := []struct {
		name       string
		scripts    []types.Script
		wantErrors int
	}{
		{
			"valid script with exec",
			[]types.Script{{Name: "setup", Exec: "echo hello"}},
			0,
		},
		{
			"valid script with content",
			[]types.Script{{Name: "setup", Content: "#!/bin/bash\necho hello"}},
			0,
		},
		{
			"missing name",
			[]types.Script{{Exec: "echo hello"}},
			1,
		},
		{
			"missing exec and content",
			[]types.Script{{Name: "setup"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateScripts(tt.scripts, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateServices(t *testing.T) {
	tests := []struct {
		name       string
		services   []types.Service
		wantErrors int
	}{
		{
			"valid service",
			[]types.Service{{Name: "nginx", Action: "enable"}},
			0,
		},
		{
			"missing name",
			[]types.Service{{Action: "enable"}},
			1,
		},
		{
			"invalid action",
			[]types.Service{{Name: "nginx", Action: "destroy"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateServices(tt.services, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateSSHKeys(t *testing.T) {
	tests := []struct {
		name       string
		keys       []types.SSHKey
		wantErrors int
	}{
		{
			"valid ssh key",
			[]types.SSHKey{{Name: "mykey", Type: "ed25519", Path: "~/.ssh"}},
			0,
		},
		{
			"missing name",
			[]types.SSHKey{{Type: "ed25519", Path: "~/.ssh"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateSSHKeys(tt.keys, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func TestValidateUsers(t *testing.T) {
	tests := []struct {
		name       string
		users      []types.User
		wantErrors int
	}{
		{
			"valid user",
			[]types.User{{Name: "testuser", Action: "create"}},
			0,
		},
		{
			"missing name",
			[]types.User{{Action: "create"}},
			1,
		},
		{
			"invalid action",
			[]types.User{{Name: "testuser", Action: "destroy"}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			ValidateUsers(tt.users, "/test.yaml", results)
			errors := countErrors(results)
			if errors != tt.wantErrors {
				t.Errorf("got %d errors, want %d; issues: %+v", errors, tt.wantErrors, results.Issues)
			}
		})
	}
}

func countErrors(results *types.ValidationResults) int {
	count := 0
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			count++
		}
	}
	return count
}
