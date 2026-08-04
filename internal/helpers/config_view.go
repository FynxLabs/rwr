package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// ConfigFilePath is the config file rwr reads: viper's resolved file when one
// loaded, otherwise config.yaml under the config directory.
func ConfigFilePath() string {
	if used := viper.ConfigFileUsed(); used != "" {
		return used
	}
	configDir := viper.GetString("rwr.configdir")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(homeDir, ".config", "rwr")
	}
	return filepath.Join(configDir, "config.yaml")
}

// ConfigView renders the effective merged configuration, one key per line,
// annotated with where each value comes from (config file, environment, or
// the running process - flags and defaults). Secrets render as the redaction
// placeholder unless showSecrets is set: `rwr config view` output lands in
// terminals, scrollback, and pasted issues.
func ConfigView(showSecrets bool) (string, error) {
	fileEntries, err := configFileEntries()
	if err != nil {
		return "", err
	}

	// Union of what the process knows and what the file declares: a key set
	// only in the file must still show up when the process has not read it.
	keySet := map[string]bool{}
	for _, key := range viper.AllKeys() {
		keySet[key] = true
	}
	for key := range fileEntries {
		keySet[key] = true
	}
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "# effective configuration (file: %s)\n", ConfigFilePath())
	for _, key := range keys {
		value := viper.GetString(key)
		if fileValue, inFile := fileEntries[key]; inFile && value == "" {
			value = fileValue
		}
		if types.IsSecretConfigKey(key) && !showSecrets {
			if value != "" {
				value = types.RedactedPlaceholder
			}
		}
		fmt.Fprintf(&b, "%-32s = %-24q (%s)\n", key, value, configKeySource(key, fileEntries))
	}
	return b.String(), nil
}

// configFileEntries reads the config file on its own, so the view can tell
// file-set keys apart from flag- and default-set ones; a missing file is an
// empty set, not an error.
func configFileEntries() (map[string]string, error) {
	path := ConfigFilePath()
	entries := map[string]string{}
	if path == "" {
		return entries, nil
	}
	if _, err := os.Stat(path); err != nil {
		return entries, nil // no config file: every value is flag, env, or default
	}
	raw := viper.New()
	raw.SetConfigFile(path)
	if err := raw.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	for _, key := range raw.AllKeys() {
		entries[key] = raw.GetString(key)
	}
	return entries, nil
}

// configKeySource labels where a key's effective value comes from.
func configKeySource(key string, fileEntries map[string]string) string {
	envName := "RWR_" + strings.NewReplacer(".", "_", "-", "_").Replace(strings.ToUpper(key))
	if _, ok := os.LookupEnv(envName); ok {
		return "env " + envName
	}
	if _, inFile := fileEntries[key]; inFile {
		return "config"
	}
	return "flag/default"
}
