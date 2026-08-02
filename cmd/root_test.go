package cmd

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// The --init-file flag is bound to repository.init-file, so config, env and
// flag all meet at one key.
func TestInitFileFlagBindsDocumentedKey(t *testing.T) {
	if err := rootCmd.PersistentFlags().Set("init-file", "bound.yaml"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}
	defer func() {
		_ = rootCmd.PersistentFlags().Set("init-file", "")
	}()
	if got := viper.GetString("repository.init-file"); got != "bound.yaml" {
		t.Errorf("repository.init-file = %q after setting --init-file, want bound.yaml", got)
	}
}

// The --init-file flag and the config file used to write and read two
// different keys (rwr.init-file vs repository.init-file). Everything now
// resolves through repository.init-file, with the old key honored as a
// deprecated fallback.
func TestConfiguredInitFile(t *testing.T) {
	reset := func() {
		viper.Set("repository.init-file", "")
		viper.Set("rwr.init-file", "")
	}

	t.Run("flag wins over everything", func(t *testing.T) {
		reset()
		viper.Set("repository.init-file", "from-config.yaml")
		if got := configuredInitFile("from-flag.yaml"); got != "from-flag.yaml" {
			t.Errorf("configuredInitFile = %q, want the flag value", got)
		}
	})

	t.Run("repository.init-file is the config key", func(t *testing.T) {
		reset()
		viper.Set("repository.init-file", "from-config.yaml")
		if got := configuredInitFile(""); got != "from-config.yaml" {
			t.Errorf("configuredInitFile = %q, want from-config.yaml", got)
		}
	})

	t.Run("deprecated rwr.init-file still resolves", func(t *testing.T) {
		reset()
		viper.Set("rwr.init-file", "legacy.yaml")
		if got := configuredInitFile(""); got != "legacy.yaml" {
			t.Errorf("configuredInitFile = %q, want the legacy key's value", got)
		}
	})

	t.Run("documented key wins over the deprecated one", func(t *testing.T) {
		reset()
		viper.Set("repository.init-file", "documented.yaml")
		viper.Set("rwr.init-file", "legacy.yaml")
		if got := configuredInitFile(""); got != "documented.yaml" {
			t.Errorf("configuredInitFile = %q, want documented.yaml", got)
		}
	})
}

// rwr.configdir was read and then unconditionally overwritten, so setting it
// could never take effect. It now participates with the right precedence:
// --config wins, then rwr.configdir (from the environment), then the default.
func TestResolveConfigLocation(t *testing.T) {
	home := t.TempDir()

	t.Run("default", func(t *testing.T) {
		loc, file := resolveConfigLocation(home, "", "")
		if want := filepath.Join(home, ".config", "rwr"); loc != want || file != "" {
			t.Errorf("resolveConfigLocation = (%q, %q), want (%q, \"\")", loc, file, want)
		}
	})

	t.Run("rwr.configdir is effective", func(t *testing.T) {
		custom := t.TempDir()
		loc, file := resolveConfigLocation(home, "", custom)
		if loc != custom || file != "" {
			t.Errorf("resolveConfigLocation = (%q, %q), want (%q, \"\")", loc, file, custom)
		}
	})

	t.Run("--config directory wins over rwr.configdir", func(t *testing.T) {
		flagDir := t.TempDir()
		loc, file := resolveConfigLocation(home, flagDir, t.TempDir())
		if loc != flagDir || file != "" {
			t.Errorf("resolveConfigLocation = (%q, %q), want (%q, \"\")", loc, file, flagDir)
		}
	})

	t.Run("--config file keeps its directory as the location", func(t *testing.T) {
		flagDir := t.TempDir()
		cfg := filepath.Join(flagDir, "config.yaml")
		loc, file := resolveConfigLocation(home, cfg, "")
		if loc != flagDir || file != cfg {
			t.Errorf("resolveConfigLocation = (%q, %q), want (%q, %q)", loc, file, flagDir, cfg)
		}
	})
}
