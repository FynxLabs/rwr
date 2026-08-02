package processors

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A real crypt(3) hash: the processor now refuses cleartext, because
// useradd/usermod write the value verbatim into /etc/shadow.
const testCryptHash = "$6$rounds=5000$abcdefgh$0123456789abcdefghijklmnopqrstuvwxyz"

func newTestInitConfig() *types.InitConfig {
	return &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				Debug: false,
			},
		},
	}
}

// createGroup tests

func TestCreateGroup_BasicGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:   "testgroup",
		Action: "create",
	}

	err := createGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("createGroup failed in dry-run mode: %v", err)
	}
}

func TestCreateGroup_WithGID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:   "testgroup",
		GID:    "1500",
		Action: "create",
	}

	err := createGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("createGroup with GID failed: %v", err)
	}
}

func TestCreateGroup_SystemGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:   "sysgroup",
		System: true,
		Action: "create",
	}

	err := createGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("createGroup with System flag failed: %v", err)
	}
}

func TestCreateGroup_WithGIDAndSystem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:   "sysgroup",
		GID:    "999",
		System: true,
		Action: "create",
	}

	err := createGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("createGroup with GID + System failed: %v", err)
	}
}

// modifyGroup tests

func TestModifyGroup_Rename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:    "oldname",
		NewName: "newname",
		Action:  "modify",
	}

	err := modifyGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyGroup rename failed: %v", err)
	}
}

func TestModifyGroup_ChangeGID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:   "mygroup",
		GID:    "2000",
		Action: "modify",
	}

	err := modifyGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyGroup change GID failed: %v", err)
	}
}

func TestModifyGroup_RenameAndChangeGID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	group := types.Group{
		Name:    "oldname",
		NewName: "newname",
		GID:     "2000",
		Action:  "modify",
	}

	err := modifyGroup(group, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyGroup rename + GID failed: %v", err)
	}
}

// createUser tests

func TestCreateUser_Minimal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "testuser",
		Action: "create",
	}

	err := createUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("createUser minimal failed: %v", err)
	}
}

func TestCreateUser_AllOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:     "fulluser",
		UID:      "1500",
		Password: testCryptHash,
		Shell:    "/bin/zsh",
		Home:     "/home/fulluser",
		Comment:  "Full Test User",
		System:   false,
		Expire:   "2025-12-31",
		Groups:   []string{"wheel", "docker"},
		Action:   "create",
	}

	err := createUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("createUser all options failed: %v", err)
	}
}

func TestCreateUser_SystemUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "svcaccount",
		System: true,
		Shell:  "/usr/sbin/nologin",
		Action: "create",
	}

	err := createUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("createUser system user failed: %v", err)
	}
}

func TestCreateUser_WithUID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "uiduser",
		UID:    "5000",
		Action: "create",
	}

	err := createUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("createUser with UID failed: %v", err)
	}
}

// modifyUser tests

func TestModifyUser_Rename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:    "olduser",
		NewName: "newuser",
		Action:  "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser rename failed: %v", err)
	}
}

func TestModifyUser_ChangePassword(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:     "myuser",
		Password: testCryptHash,
		Action:   "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser change password failed: %v", err)
	}
}

func TestModifyUser_ChangeComment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:    "myuser",
		Comment: "Updated GECOS",
		Action:  "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser change comment failed: %v", err)
	}
}

func TestModifyUser_ChangeUID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "myuser",
		UID:    "3000",
		Action: "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser change UID failed: %v", err)
	}
}

func TestModifyUser_SetExpiry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "myuser",
		Expire: "2026-06-15",
		Action: "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser set expiry failed: %v", err)
	}
}

func TestModifyUser_Lock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "myuser",
		Lock:   true,
		Action: "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser lock failed: %v", err)
	}
}

func TestModifyUser_Unlock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "myuser",
		Unlock: true,
		Action: "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser unlock failed: %v", err)
	}
}

func TestModifyUser_AddAndRemoveGroups(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:         "myuser",
		AddGroups:    []string{"docker", "wheel"},
		RemoveGroups: []string{"oldgroup"},
		Action:       "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser add/remove groups failed: %v", err)
	}
}

func TestModifyUser_AllOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:         "myuser",
		NewName:      "renameduser",
		NewHome:      "/home/renameduser",
		NewShell:     "/bin/fish",
		Password:     testCryptHash,
		Comment:      "Updated User",
		UID:          "4000",
		Expire:       "2027-01-01",
		Lock:         false,
		Unlock:       true,
		AddGroups:    []string{"newgroup"},
		RemoveGroups: []string{"oldgroup"},
		Action:       "modify",
	}

	err := modifyUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("modifyUser all options failed: %v", err)
	}
}

// removeUser tests

func TestRemoveUser_Basic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:   "gonuser",
		Action: "remove",
	}

	err := removeUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("removeUser basic failed: %v", err)
	}
}

func TestRemoveUser_WithRemoveHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	user := types.User{
		Name:       "gonuser",
		RemoveHome: true,
		Action:     "remove",
	}

	err := removeUser(user, newTestInitConfig())
	if err != nil {
		t.Errorf("removeUser with RemoveHome failed: %v", err)
	}
}

// processGroups dispatch tests

func TestProcessGroups_UnsupportedAction(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	// Don't enable dry-run here: we're testing action validation,
	// which happens in the switch default case before any command runs.
	groups := []types.Group{
		{Name: "badgroup", Action: "destroy"},
	}

	err := processGroups(groups, newTestInitConfig())
	if err == nil {
		t.Error("Expected error for unsupported group action 'destroy'")
	}
}

func TestProcessUsers_UnsupportedAction(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	// Don't enable dry-run here: we're testing action validation,
	// which happens in the switch default case before any command runs.
	users := []types.User{
		{Name: "baduser", Action: "destroy"},
	}

	err := processUsers(users, newTestInitConfig())
	if err == nil {
		t.Error("Expected error for unsupported user action 'destroy'")
	}
}

func TestProcessGroups_EmptySlice(t *testing.T) {
	err := processGroups([]types.Group{}, newTestInitConfig())
	if err != nil {
		t.Errorf("Expected no error for empty groups, got: %v", err)
	}
}

func TestProcessUsers_EmptySlice(t *testing.T) {
	err := processUsers([]types.User{}, newTestInitConfig())
	if err != nil {
		t.Errorf("Expected no error for empty users, got: %v", err)
	}
}

// Interactive override tests

func boolPtrUser(b bool) *bool {
	return &b
}

func TestCreateUser_InteractiveOverrideTrue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	// Global interactive is false, but blueprint overrides to true
	config := newTestInitConfig()
	config.Variables.Flags.Interactive = false

	user := types.User{
		Name:        "interactiveuser",
		Action:      "create",
		Interactive: boolPtrUser(true),
	}

	err := createUser(user, config)
	if err != nil {
		t.Errorf("createUser with interactive override true failed: %v", err)
	}
}

func TestCreateUser_InteractiveOverrideFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	// Global interactive is true, but blueprint overrides to false
	config := newTestInitConfig()
	config.Variables.Flags.Interactive = true

	user := types.User{
		Name:        "noninteractiveuser",
		Action:      "create",
		Interactive: boolPtrUser(false),
	}

	err := createUser(user, config)
	if err != nil {
		t.Errorf("createUser with interactive override false failed: %v", err)
	}
}

func TestCreateUser_InteractiveNilUsesGlobal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	// Global interactive is true, no override
	config := newTestInitConfig()
	config.Variables.Flags.Interactive = true

	user := types.User{
		Name:        "defaultuser",
		Action:      "create",
		Interactive: nil,
	}

	err := createUser(user, config)
	if err != nil {
		t.Errorf("createUser with nil interactive (global true) failed: %v", err)
	}
}

func TestModifyUser_InteractiveOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	config := newTestInitConfig()
	config.Variables.Flags.Interactive = false

	user := types.User{
		Name:        "myuser",
		NewName:     "renameduser",
		Action:      "modify",
		Interactive: boolPtrUser(true),
	}

	err := modifyUser(user, config)
	if err != nil {
		t.Errorf("modifyUser with interactive override failed: %v", err)
	}
}

// Dry-run mode tests - verify processors skip execution

func TestProcessGroups_DryRunSkipsExecution(t *testing.T) {
	system.SetDryRun(true)
	defer system.SetDryRun(false)

	groups := []types.Group{
		{Name: "testgroup1", Action: "create"},
		{Name: "testgroup2", Action: "modify", NewName: "renamed"},
	}

	err := processGroups(groups, newTestInitConfig())
	if err != nil {
		t.Errorf("processGroups should succeed in dry-run mode, got: %v", err)
	}
}

func TestProcessUsers_DryRunSkipsExecution(t *testing.T) {
	system.SetDryRun(true)
	defer system.SetDryRun(false)

	users := []types.User{
		{Name: "user1", Action: "create"},
		{Name: "user2", Action: "modify", NewName: "renamed"},
		{Name: "user3", Action: "remove"},
	}

	err := processUsers(users, newTestInitConfig())
	if err != nil {
		t.Errorf("processUsers should succeed in dry-run mode, got: %v", err)
	}
}

func TestRemoveUser_InteractiveOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	config := newTestInitConfig()
	config.Variables.Flags.Interactive = true

	user := types.User{
		Name:        "gonuser",
		Action:      "remove",
		Interactive: boolPtrUser(false),
	}

	err := removeUser(user, config)
	if err != nil {
		t.Errorf("removeUser with interactive override false failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// argv assertions
//
// These live in user_test.go rather than user_darwin_test.go on purpose: Go
// applies an implicit build constraint to any file ending in _darwin_test.go, so
// the macOS cases would never compile on the Linux CI runner — which is the only
// place they can run at all.
// ---------------------------------------------------------------------------

// probeExec wraps a Recorder so existence probes can be answered per command.
// Recorder has a single Err covering every call, which cannot express "the user
// is missing but everything else succeeds".
type probeExec struct {
	rec    *exectest.Recorder
	exists bool
	stdout string
}

func isProbe(cmd types.Command) bool {
	switch cmd.Exec {
	case "getent":
		return true
	case "dscl":
		return len(cmd.Args) >= 2 && cmd.Args[1] == "-read"
	case "dseditgroup":
		return len(cmd.Args) >= 2 && cmd.Args[1] == "checkmember"
	}
	return false
}

func (p probeExec) Run(cmd types.Command, debug bool) error {
	_ = p.rec.Run(cmd, debug)
	if isProbe(cmd) && !p.exists {
		return errors.New("exit status 1")
	}
	return nil
}

func (p probeExec) Output(cmd types.Command, debug bool) (string, error) {
	p.rec.Stdout = p.stdout
	return p.rec.Output(cmd, debug)
}

// platform points the processor at a GOOS and a tool set, and records every
// command it builds.
func platform(t *testing.T, goos string, exists bool, haveSysadminctl bool, stdout string) *exectest.Recorder {
	t.Helper()

	prevOS := userGOOS
	userGOOS = goos
	prevExists := commandExists
	commandExists = func(name string) bool {
		if name == "sysadminctl" {
			return haveSysadminctl
		}
		return true
	}

	rec := exectest.New()
	restore := system.SetExecutor(probeExec{rec: rec, exists: exists, stdout: stdout})

	t.Cleanup(func() {
		restore()
		commandExists = prevExists
		userGOOS = prevOS
	})
	return rec
}

func argvs(rec *exectest.Recorder) [][]string {
	out := make([][]string, 0, len(rec.Calls))
	for _, c := range rec.Calls {
		out = append(out, c.Argv())
	}
	return out
}

func assertArgv(t *testing.T, rec *exectest.Recorder, want [][]string) {
	t.Helper()
	got := argvs(rec)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands:\ngot:  %v\nwant: %v", got, want)
	}
	// Everything that changes the directory must be elevated; probes must not be.
	for _, c := range rec.Calls {
		if isProbeCall(c) {
			if c.Elevated {
				t.Errorf("probe %v should not be elevated", c.Argv())
			}
			continue
		}
		if c.Exec == "dscl" && len(c.Args) >= 2 && c.Args[1] == "-list" {
			continue // read-only ID enumeration
		}
		if !c.Elevated {
			t.Errorf("mutating command %v should be elevated", c.Argv())
		}
	}
}

func isProbeCall(c exectest.Call) bool {
	return isProbe(types.Command{Exec: c.Exec, Args: c.Args})
}

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

// ---------------------------------------------------------------------------
// idempotency
// ---------------------------------------------------------------------------

// The second `rwr all` used to abort the entire run here: useradd exits non-zero
// on "user already exists" and the error propagated all the way out.
func TestCreateUser_ExistingUserConvergesOnLinux(t *testing.T) {
	rec := platform(t, "linux", true, false, "")

	user := types.User{
		Name:     "alice",
		UID:      "1500",
		Shell:    "/bin/zsh",
		Home:     "/home/alice",
		Comment:  "Alice",
		Password: testCryptHash,
		Groups:   []string{"docker"},
		Action:   "create",
	}
	if err := createUser(user, newTestInitConfig()); err != nil {
		t.Fatalf("createUser on an existing account must not fail: %v", err)
	}

	assertArgv(t, rec, [][]string{
		{"getent", "passwd", "alice"},
		{"usermod", "--uid", "1500", "--shell", "/bin/zsh",
			"--home", "/home/alice", "--comment", "Alice", "--append", "--groups", "docker", "alice"},
		{"chpasswd", "-e"},
	})
	if calls := rec.Find("useradd"); len(calls) != 0 {
		t.Errorf("useradd should not run for an existing user: %v", calls)
	}
}

func TestCreateUser_AbsentUserUsesUseraddOnLinux(t *testing.T) {
	rec := platform(t, "linux", false, false, "")

	if err := createUser(types.User{Name: "alice", Shell: "/bin/zsh", Action: "create"}, newTestInitConfig()); err != nil {
		t.Fatalf("createUser: %v", err)
	}
	assertArgv(t, rec, [][]string{
		{"getent", "passwd", "alice"},
		{"useradd", "--create-home", "--shell", "/bin/zsh", "alice"},
	})
}

func TestCreateGroup_ExistingGroupConvergesOnLinux(t *testing.T) {
	rec := platform(t, "linux", true, false, "")

	if err := createGroup(types.Group{Name: "devs", GID: "1500", Action: "create"}, newTestInitConfig()); err != nil {
		t.Fatalf("createGroup on an existing group must not fail: %v", err)
	}
	assertArgv(t, rec, [][]string{
		{"getent", "group", "devs"},
		{"groupmod", "devs", "--gid", "1500"},
	})
	if calls := rec.Find("groupadd"); len(calls) != 0 {
		t.Errorf("groupadd should not run for an existing group: %v", calls)
	}
}

// Two runs of the same blueprint must produce the same end state, and the second
// must not error.
func TestProcessUsers_SecondRunDoesNotAbort(t *testing.T) {
	rec := platform(t, "linux", true, false, "")

	users := []types.User{{Name: "alice", Shell: "/bin/zsh", Action: "create"}}
	if err := processUsers(users, newTestInitConfig()); err != nil {
		t.Fatalf("second run aborted: %v", err)
	}
	if len(rec.Calls) == 0 || isProbeCall(rec.Calls[len(rec.Calls)-1]) {
		t.Fatalf("expected a converging usermod, got %v", argvs(rec))
	}
}

// ---------------------------------------------------------------------------
// password handling
// ---------------------------------------------------------------------------

func TestIsCryptHash(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "yescrypt", value: "$y$j9T$abcdef$ghijkl", want: true},
		{name: "sha512", value: testCryptHash, want: true},
		{name: "bcrypt", value: "$2b$10$abcdefghijklmnopqrstuv", want: true},
		{name: "des", value: "ab1234567890X", want: true},
		{name: "locked", value: "!", want: true},
		{name: "no password", value: "*", want: true},
		{name: "cleartext", value: "hunter2"},
		{name: "cleartext with dollar inside", value: "hunter$2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCryptHash(tt.value); got != tt.want {
				t.Errorf("isCryptHash(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// The whole point of routing through chpasswd: the value is readable in `ps` by
// every local user for as long as the command runs if it is an argument, and the
// debug logger prints cmd.Args.
func TestLinuxPasswordNeverReachesArgv(t *testing.T) {
	const secret = "hunter2"

	for _, action := range []string{"create", "modify"} {
		t.Run(action, func(t *testing.T) {
			rec := platform(t, "linux", false, false, "")

			users := []types.User{{Name: "alice", Password: secret, Action: action}}
			if err := processUsers(users, newTestInitConfig()); err != nil {
				t.Fatalf("processUsers: %v", err)
			}

			chpasswd := rec.Find("chpasswd")
			if len(chpasswd) != 1 {
				t.Fatalf("expected exactly one chpasswd call, got %v", argvs(rec))
			}
			if want := "alice:" + secret + "\n"; chpasswd[0].Stdin != want {
				t.Errorf("chpasswd stdin = %q, want %q", chpasswd[0].Stdin, want)
			}

			for _, call := range rec.Calls {
				for _, arg := range call.Argv() {
					if strings.Contains(arg, secret) {
						t.Errorf("password reached argv: %v", call)
					}
				}
			}
		})
	}
}

// A pre-hashed value must be handed to chpasswd with -e, or chpasswd would hash
// the hash and the account would have no working password.
func TestLinuxHashedPasswordUsesEncryptedFlag(t *testing.T) {
	rec := platform(t, "linux", false, false, "")

	users := []types.User{{Name: "alice", Password: testCryptHash, Action: "create"}}
	if err := processUsers(users, newTestInitConfig()); err != nil {
		t.Fatalf("processUsers: %v", err)
	}

	chpasswd := rec.Find("chpasswd")
	if len(chpasswd) != 1 {
		t.Fatalf("expected exactly one chpasswd call, got %v", argvs(rec))
	}
	if len(chpasswd[0].Args) != 1 || chpasswd[0].Args[0] != "-e" {
		t.Errorf("chpasswd args = %v, want [-e]", chpasswd[0].Args)
	}
}
