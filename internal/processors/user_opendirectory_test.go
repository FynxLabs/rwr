// Tests for the Open Directory backend: dscl argv construction and the
// macOS-specific field handling.

package processors

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestDarwinGroupArgv(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
		stdout string
		run    func(*types.InitConfig) error
		want   [][]string
	}{
		{
			name: "create allocates a free GID above 500",
			run: func(c *types.InitConfig) error {
				return createGroup(types.Group{Name: "devs", Action: "create"}, c)
			},
			stdout: "staff 20\nadmin 80\ndevops 501\n",
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
				{"dscl", ".", "-list", "/Groups", "PrimaryGroupID"},
				{"dscl", ".", "-create", "/Groups/devs"},
				{"dscl", ".", "-create", "/Groups/devs", "PrimaryGroupID", "502"},
				{"dscl", ".", "-create", "/Groups/devs", "RealName", "devs"},
			},
		},
		{
			name: "create honours an explicit GID",
			run: func(c *types.InitConfig) error {
				return createGroup(types.Group{Name: "devs", GID: "888", Action: "create"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
				{"dscl", ".", "-create", "/Groups/devs"},
				{"dscl", ".", "-create", "/Groups/devs", "PrimaryGroupID", "888"},
				{"dscl", ".", "-create", "/Groups/devs", "RealName", "devs"},
			},
		},
		{
			name:   "create on an existing group converges instead of failing",
			exists: true,
			run: func(c *types.InitConfig) error {
				return createGroup(types.Group{Name: "devs", GID: "888", Action: "create"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
				{"dscl", ".", "-create", "/Groups/devs", "PrimaryGroupID", "888"},
			},
		},
		{
			name:   "create on an existing group with no attributes runs nothing",
			exists: true,
			run: func(c *types.InitConfig) error {
				return createGroup(types.Group{Name: "devs", Action: "create"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
			},
		},
		{
			name: "modify renames after changing attributes",
			run: func(c *types.InitConfig) error {
				return modifyGroup(types.Group{Name: "devs", NewName: "eng", GID: "777", Action: "modify"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-create", "/Groups/devs", "PrimaryGroupID", "777"},
				{"dscl", ".", "-change", "/Groups/devs", "RecordName", "devs", "eng"},
			},
		},
		{
			name:   "remove deletes the record",
			exists: true,
			run: func(c *types.InitConfig) error {
				return removeGroup(types.Group{Name: "devs", Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
				{"dscl", ".", "-delete", "/Groups/devs"},
			},
		},
		{
			name: "remove of a missing group is a no-op",
			run: func(c *types.InitConfig) error {
				return removeGroup(types.Group{Name: "devs", Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Groups/devs"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := platform(t, "darwin", tt.exists, true, tt.stdout)
			if err := tt.run(newTestInitConfig()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertArgv(t, rec, tt.want)
		})
	}
}

func TestDarwinUserArgv(t *testing.T) {
	tests := []struct {
		name        string
		exists      bool
		sysadminctl bool
		stdout      string
		run         func(*types.InitConfig) error
		want        [][]string
	}{
		{
			name:        "create via sysadminctl",
			sysadminctl: true,
			run: func(c *types.InitConfig) error {
				return createUser(types.User{
					Name:    "alice",
					UID:     "600",
					Shell:   "/bin/zsh",
					Home:    "/Users/alice",
					Comment: "Alice Example",
					Groups:  []string{"devs"},
					Action:  "create",
				}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"sysadminctl", "-addUser", "alice", "-fullName", "Alice Example", "-UID", "600", "-shell", "/bin/zsh", "-home", "/Users/alice"},
				{"createhomedir", "-c", "-u", "alice"},
				{"dseditgroup", "-o", "checkmember", "-m", "alice", "devs"},
				{"dseditgroup", "-o", "edit", "-a", "alice", "-t", "user", "devs"},
			},
		},
		{
			name:   "create falls back to dscl records and allocates a UID",
			stdout: "root 0\ndaemon 1\nbob 501\n",
			run: func(c *types.InitConfig) error {
				return createUser(types.User{Name: "carol", Action: "create"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/carol"},
				{"dscl", ".", "-list", "/Users", "UniqueID"},
				{"dscl", ".", "-create", "/Users/carol"},
				{"dscl", ".", "-create", "/Users/carol", "UserShell", "/bin/zsh"},
				{"dscl", ".", "-create", "/Users/carol", "RealName", "carol"},
				{"dscl", ".", "-create", "/Users/carol", "UniqueID", "502"},
				{"dscl", ".", "-create", "/Users/carol", "PrimaryGroupID", "20"},
				{"dscl", ".", "-create", "/Users/carol", "NFSHomeDirectory", "/Users/carol"},
				{"createhomedir", "-c", "-u", "carol"},
			},
		},
		{
			name:        "create sets the password through dscl, never sysadminctl",
			sysadminctl: true,
			run: func(c *types.InitConfig) error {
				return createUser(types.User{Name: "alice", UID: "600", Password: "s3cret", Action: "create"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"sysadminctl", "-addUser", "alice", "-fullName", "alice", "-UID", "600", "-shell", "/bin/zsh", "-home", "/Users/alice"},
				{"createhomedir", "-c", "-u", "alice"},
				{"dscl", ".", "-passwd", "/Users/alice", "s3cret"},
			},
		},
		{
			name:        "create on an existing user converges instead of failing",
			exists:      true,
			sysadminctl: true,
			run: func(c *types.InitConfig) error {
				return createUser(types.User{
					Name:    "alice",
					Shell:   "/bin/bash",
					Comment: "Alice",
					UID:     "600",
					Home:    "/Users/alice",
					Groups:  []string{"devs"},
					Action:  "create",
				}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"dscl", ".", "-create", "/Users/alice", "UserShell", "/bin/bash"},
				{"dscl", ".", "-create", "/Users/alice", "RealName", "Alice"},
				{"dscl", ".", "-create", "/Users/alice", "UniqueID", "600"},
				{"dscl", ".", "-create", "/Users/alice", "NFSHomeDirectory", "/Users/alice"},
				// already a member, so no dseditgroup edit
				{"dseditgroup", "-o", "checkmember", "-m", "alice", "devs"},
			},
		},
		{
			name: "modify maps every supported field and renames last",
			run: func(c *types.InitConfig) error {
				return modifyUser(types.User{
					Name:         "alice",
					NewName:      "alicia",
					NewShell:     "/bin/bash",
					NewHome:      "/Users/alicia",
					Comment:      "Alicia",
					UID:          "601",
					Lock:         true,
					AddGroups:    []string{"devs"},
					RemoveGroups: []string{"ops"},
					Action:       "modify",
				}, c)
			},
			want: [][]string{
				{"dscl", ".", "-create", "/Users/alice", "UserShell", "/bin/bash"},
				{"dscl", ".", "-create", "/Users/alice", "RealName", "Alicia"},
				{"dscl", ".", "-create", "/Users/alice", "UniqueID", "601"},
				{"dscl", ".", "-create", "/Users/alice", "NFSHomeDirectory", "/Users/alicia"},
				{"pwpolicy", "-u", "alice", "-disableuser"},
				{"dseditgroup", "-o", "checkmember", "-m", "alice", "devs"},
				{"dseditgroup", "-o", "edit", "-a", "alice", "-t", "user", "devs"},
				{"dseditgroup", "-o", "edit", "-d", "alice", "-t", "user", "ops"},
				{"dscl", ".", "-change", "/Users/alice", "RecordName", "alice", "alicia"},
			},
		},
		{
			name: "unlock uses pwpolicy",
			run: func(c *types.InitConfig) error {
				return modifyUser(types.User{Name: "alice", Unlock: true, Action: "modify"}, c)
			},
			want: [][]string{
				{"pwpolicy", "-u", "alice", "-enableuser"},
			},
		},
		{
			name:        "remove keeps the home directory unless asked",
			exists:      true,
			sysadminctl: true,
			run: func(c *types.InitConfig) error {
				return removeUser(types.User{Name: "alice", Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"sysadminctl", "-deleteUser", "alice", "-keepHome"},
			},
		},
		{
			name:        "remove_home deletes the home directory",
			exists:      true,
			sysadminctl: true,
			run: func(c *types.InitConfig) error {
				return removeUser(types.User{Name: "alice", RemoveHome: true, Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"sysadminctl", "-deleteUser", "alice"},
			},
		},
		{
			name:   "remove falls back to dscl",
			exists: true,
			run: func(c *types.InitConfig) error {
				return removeUser(types.User{Name: "alice", Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
				{"dscl", ".", "-delete", "/Users/alice"},
			},
		},
		{
			name: "remove of a missing user is a no-op",
			run: func(c *types.InitConfig) error {
				return removeUser(types.User{Name: "alice", Action: "remove"}, c)
			},
			want: [][]string{
				{"dscl", ".", "-read", "/Users/alice"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := platform(t, "darwin", tt.exists, tt.sysadminctl, tt.stdout)
			if err := tt.run(newTestInitConfig()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertArgv(t, rec, tt.want)
		})
	}
}

// macOS accepts no crypt hash, so the cleartext rule must not leak onto it.
func TestDarwinPasswordAcceptsCleartext(t *testing.T) {
	rec := platform(t, "darwin", true, false, "")
	if err := modifyUser(types.User{Name: "alice", Password: "plain text", Action: "modify"}, newTestInitConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertArgv(t, rec, [][]string{
		{"dscl", ".", "-passwd", "/Users/alice", "plain text"},
	})
}

// Fields with no Open Directory equivalent must not abort the run.
func TestDarwinUnsupportedFieldsDoNotFail(t *testing.T) {
	rec := platform(t, "darwin", false, true, "")
	user := types.User{Name: "svc", UID: "700", System: true, Expire: "2030-01-01", Action: "create"}
	if err := createUser(user, newTestInitConfig()); err != nil {
		t.Fatalf("system/expire should be warned about, not fatal: %v", err)
	}
	for _, c := range rec.Calls {
		for _, a := range c.Args {
			if strings.Contains(a, "2030-01-01") {
				t.Errorf("expire leaked into argv: %v", c.Argv())
			}
		}
	}
}
