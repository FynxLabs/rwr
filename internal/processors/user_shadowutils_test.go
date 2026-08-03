// Tests for the shadow-utils backend: argv construction for groupadd,
// usermod and friends, convergence, and password handling.

package processors

import (
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

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

	err := processGroups(groups, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
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

	err := processUsers(users, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
	if err == nil {
		t.Error("Expected error for unsupported user action 'destroy'")
	}
}

func TestProcessGroups_EmptySlice(t *testing.T) {
	err := processGroups([]types.Group{}, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
	if err != nil {
		t.Errorf("Expected no error for empty groups, got: %v", err)
	}
}

func TestProcessUsers_EmptySlice(t *testing.T) {
	err := processUsers([]types.User{}, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
	if err != nil {
		t.Errorf("Expected no error for empty users, got: %v", err)
	}
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

	err := processGroups(groups, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
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

	err := processUsers(users, newTestInitConfig(), newProgress(types.BlueprintTypeUsers))
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
			if err := processUsers(users, newTestInitConfig(), newProgress(types.BlueprintTypeUsers)); err != nil {
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
	if err := processUsers(users, newTestInitConfig(), newProgress(types.BlueprintTypeUsers)); err != nil {
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
