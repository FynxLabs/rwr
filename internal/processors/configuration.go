package processors

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessConfiguration applies desktop environment settings from blueprint data.
// It supports dconf, gsettings, macOS defaults, and Windows registry operations.
func ProcessConfiguration(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
	var configData types.ConfigData

	err := helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeConfiguration,
		helpers.TreeSchemaVersion(initConfig), &configData)
	if err != nil {
		return fmt.Errorf("error unmarshaling configuration blueprint: %w", err)
	}

	configurations := helpers.FilterByProfiles(configData.Configurations, initConfig.Variables.Flags.Profiles)

	for _, config := range configurations {
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would apply %s configuration: %s", config.Tool, config.Name)
			continue
		}
		// `set` is the only action any of these tools implement. The field was
		// decoded and never read, so `action: banana` applied the setting anyway.
		if config.Action != "" && config.Action != types.ConfigurationActionSet {
			recordFailure("configuration", config.Name,
				fmt.Errorf("unsupported action %q: the only supported action is %q", config.Action, types.ConfigurationActionSet))
			continue
		}

		var err error
		switch config.Tool {
		case "dconf":
			err = processDconf(blueprintDir, config, initConfig)
		case "gsettings":
			err = processGSettings(config)
		case "macos_defaults":
			err = processMacOSDefaults(config, initConfig)
		case "windows_registry":
			err = processWindowsRegistry(config, initConfig)
		default:
			err = fmt.Errorf("unsupported configuration tool: %s", config.Tool)
		}

		if err != nil {
			log.Errorf("Error processing configuration %s: %v", config.Name, err)
			return fmt.Errorf("error processing configuration %s: %w", config.Name, err)
		}
	}

	return nil
}

func processDconf(blueprintDir string, config types.Configuration, initConfig *types.InitConfig) error {
	log.Debugf("Processing Dconf file: %s", config.File)

	// Resolve the file path relative to the blueprint directory
	file := filepath.Join(blueprintDir, config.File)

	log.Debugf("Dconf file set for path: %s", file)

	boostrapfileName := "configuration_" + config.Name + "_bootstrap"

	bootstrapFile := filepath.Join(initConfig.Variables.Flags.RunOnceLocation, boostrapfileName)

	if config.RunOnce {
		log.Debugf("RunOnce Set: Checking for %s to see if already ran", bootstrapFile)
		if _, err := os.Stat(bootstrapFile); err == nil {
			log.Infof("Dconf configuration already applied, skipping")
			return nil
		}
	}

	cmd := types.Command{
		Exec:     "dconf",
		Args:     []string{"load", "/", "<", file},
		Elevated: config.Elevated,
	}

	err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying dconf configuration: %w", err)
	}

	if config.RunOnce {
		log.Debugf("RunOnce Set: Write Bootstrap File %s", bootstrapFile)
		if err := os.WriteFile(bootstrapFile, []byte{}, 0644); err != nil { // #nosec G306 -- TODO(PR8): create with target mode instead of chmod-after
			log.Warnf("Failed to create dconf bootstrap file: %v", err)
		}
	}

	return nil
}

func processGSettings(config types.Configuration) error {
	log.Debugf("Processing gsettings configuration: %s", config.Name)

	for key, value := range config.Settings {
		log.Debugf("Processing key: %s with value: %v", key, value)

		// Check if the key is writable
		checkCmd := exec.Command("gsettings", "writable", config.Schema, key) // #nosec G204 -- argv, not a shell string: schema/key/value are passed as discrete arguments
		output, err := checkCmd.CombinedOutput()
		if err != nil {
			recordFailure("configuration", config.Schema+"."+key,
				fmt.Errorf("checking whether the key is writable: %v (%s)", err, strings.TrimSpace(string(output))))
			continue
		}

		if strings.TrimSpace(string(output)) != "true" {
			recordFailure("configuration", config.Schema+"."+key, errors.New("gsetting is not writable"))
			continue
		}

		// Convert the value to a string and escape it properly
		strValue := formatGSettingsValue(value)
		log.Debugf("Formatted value: %s", strValue)

		args := []string{"set", config.Schema, key, strValue}
		log.Debugf("Executing command: gsettings %s", strings.Join(args, " "))

		cmd := exec.Command("gsettings", args...) // #nosec G204 -- argv, not a shell string: schema/key/value are passed as discrete arguments
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Every failure here used to be logged and then discarded: this
			// function returned nil unconditionally, so a run in which no setting
			// applied still reported success.
			recordFailure("configuration", config.Schema+"."+key,
				fmt.Errorf("applying value %s: %v (%s)", strValue, err, strings.TrimSpace(string(output))))
		} else {
			log.Debugf("Successfully applied gsettings - Schema: %s, Key: %s, Value: %s", config.Schema, key, strValue)
		}
	}

	return nil
}

func formatGSettingsValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// If the string already looks like a formatted gsettings value, return it as-is
		if strings.HasPrefix(v, "[") || strings.HasPrefix(v, "(") {
			return v
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
	case []interface{}:
		var elements []string
		for _, elem := range v {
			elements = append(elements, formatGSettingsValue(elem))
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ","))
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "[]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func processMacOSDefaults(config types.Configuration, initConfig *types.InitConfig) error {
	args := []string{"write"}
	if config.Domain != "" {
		args = append(args, config.Domain)
	} else {
		args = append(args, "NSGlobalDomain")
	}
	args = append(args, config.Key, fmt.Sprintf("-%s", config.Kind), fmt.Sprintf("%v", config.Value))

	cmd := types.Command{
		Exec:     "defaults",
		Args:     args,
		Elevated: config.Elevated,
	}

	err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug)

	if err != nil {
		return fmt.Errorf("error applying macOS defaults configuration: %w", err)
	}

	return nil
}

// Environment variables carrying the registry path, name and value into the
// PowerShell script body. See processWindowsRegistry for why they exist.
const (
	registryPathEnv  = "RWR_REGISTRY_PATH"
	registryNameEnv  = "RWR_REGISTRY_NAME"
	registryValueEnv = "RWR_REGISTRY_VALUE"
)

// registryScripts maps a blueprint registry type to the PowerShell body used to
// write it. Each body is a compile-time constant that only ever reads its inputs
// from $env:, so nothing a blueprint supplies is ever parsed as PowerShell.
var registryScripts = map[string]string{
	"string":       `Set-ItemProperty -LiteralPath $env:` + registryPathEnv + ` -Name $env:` + registryNameEnv + ` -Value $env:` + registryValueEnv + ` -Type String`,
	"expandstring": `Set-ItemProperty -LiteralPath $env:` + registryPathEnv + ` -Name $env:` + registryNameEnv + ` -Value $env:` + registryValueEnv + ` -Type ExpandString`,
	"binary":       `Set-ItemProperty -LiteralPath $env:` + registryPathEnv + ` -Name $env:` + registryNameEnv + ` -Value ([byte[]]($env:` + registryValueEnv + ` -split ',' | ForEach-Object { [byte]$_ })) -Type Binary`,
	"dword":        `Set-ItemProperty -LiteralPath $env:` + registryPathEnv + ` -Name $env:` + registryNameEnv + ` -Value ([int32]$env:` + registryValueEnv + `) -Type DWord`,
	"qword":        `Set-ItemProperty -LiteralPath $env:` + registryPathEnv + ` -Name $env:` + registryNameEnv + ` -Value ([int64]$env:` + registryValueEnv + `) -Type QWord`,
}

// processWindowsRegistry writes a single registry value through PowerShell.
//
// The path, name and value are blueprint-supplied and are passed as environment
// variables rather than interpolated into the -Command string. PowerShell tokenizes
// whatever follows -Command, so a value such as `a'; Remove-Item C:\ -Recurse; #`
// used to close the surrounding quote and run as a second statement; $env: lookups
// are values, never re-parsed as code. For the same reason there is no
// `Start-Process -Verb RunAs -ArgumentList "-Command ..."` branch any more — that
// re-tokenized the already-built string a second time. Windows elevation follows
// the same rule as every other command in rwr: the process must already be
// elevated (see system.buildCommand).
func processWindowsRegistry(config types.Configuration, initConfig *types.InitConfig) error {
	regType := strings.ToLower(config.Type)
	script, ok := registryScripts[regType]
	if !ok {
		return fmt.Errorf("unsupported registry value type: %s", config.Type)
	}

	value, err := registryValueString(regType, config.Value)
	if err != nil {
		return err
	}

	path := `HKLM:\` + config.Path

	if config.Elevated {
		return applyRegistryElevated(regType, path, config.Key, value, initConfig)
	}

	cmd := types.Command{
		Exec:     "powershell",
		Args:     []string{"-NoProfile", "-NonInteractive", "-Command", script},
		Elevated: false,
		Variables: map[string]string{
			registryPathEnv:  path,
			registryNameEnv:  config.Key,
			registryValueEnv: value,
		},
	}

	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error applying Windows registry configuration: %w", err)
	}

	return nil
}

// applyRegistryElevated performs the write through a UAC prompt.
//
// Windows has no sudo, so elevation means Start-Process -Verb RunAs. The obvious
// way to do that — building the inner command as a string and handing it to
// -ArgumentList — is what the injection fix removed: the string is tokenized a
// second time by the elevated shell, so a quote in a blueprint value escapes into
// code running as administrator.
//
// The values cannot travel in the environment either. A UAC-elevated child is
// created through ShellExecute with a fresh token and does not reliably inherit
// the parent's environment block, which is what the non-elevated path relies on.
//
// So they travel as data: a JSON file rwr writes, whose path is generated by rwr
// and never contains blueprint input. The elevated script reads it with
// ConvertFrom-Json and binds the fields as *values*, never as code. The outer
// command is a compile-time constant plus a base64 blob, and base64's alphabet
// contains no quote, space or metacharacter, so neither layer has anything a
// blueprint can influence.
func applyRegistryElevated(regType, path, name, value string, initConfig *types.InitConfig) error {
	payload := struct {
		Type  string `json:"Type"`
		Path  string `json:"Path"`
		Name  string `json:"Name"`
		Value string `json:"Value"`
	}{Type: regType, Path: path, Name: name, Value: value}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error encoding registry payload: %w", err)
	}

	dataFile, err := os.CreateTemp("", "rwr-registry-*.json")
	if err != nil {
		return fmt.Errorf("error creating registry payload file: %w", err)
	}
	dataPath := dataFile.Name()
	defer func() {
		// The elevated script removes this itself; this covers the case where the
		// operator declines the UAC prompt and it never runs.
		if rerr := os.Remove(dataPath); rerr != nil && !os.IsNotExist(rerr) {
			log.Debugf("Removing registry payload file %s: %v", dataPath, rerr)
		}
	}()

	if _, err := dataFile.Write(encoded); err != nil {
		if cerr := dataFile.Close(); cerr != nil {
			log.Debugf("Closing registry payload file after a failed write: %v", cerr)
		}
		return fmt.Errorf("error writing registry payload file: %w", err)
	}
	if err := dataFile.Close(); err != nil {
		return fmt.Errorf("error closing registry payload file: %w", err)
	}

	// os.CreateTemp cannot produce one, but the script below embeds this path in a
	// single-quoted literal and a quote would break out of it. Refuse rather than
	// assume.
	if strings.ContainsAny(dataPath, "'\"\n\r") {
		return fmt.Errorf("refusing to use temporary path containing a quote: %q", dataPath)
	}

	inner := fmt.Sprintf(elevatedRegistryScript, dataPath)
	outer := fmt.Sprintf(
		`$p = Start-Process -FilePath powershell -Verb RunAs -Wait -PassThru -WindowStyle Hidden `+
			`-ArgumentList @('-NoProfile','-NonInteractive','-EncodedCommand','%s'); exit $p.ExitCode`,
		encodePowerShellCommand(inner))

	cmd := types.Command{
		Exec: "powershell",
		Args: []string{"-NoProfile", "-NonInteractive", "-Command", outer},
	}

	if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error applying elevated Windows registry configuration: %w", err)
	}

	return nil
}

// elevatedRegistryScript reads the payload and applies it. It is a compile-time
// constant with exactly one substitution: the payload path, which rwr generates.
const elevatedRegistryScript = `$ErrorActionPreference = 'Stop'
$payloadPath = '%s'
try {
  $d = Get-Content -LiteralPath $payloadPath -Raw | ConvertFrom-Json
  switch ($d.Type) {
    'string'       { Set-ItemProperty -LiteralPath $d.Path -Name $d.Name -Value $d.Value -Type String }
    'expandstring' { Set-ItemProperty -LiteralPath $d.Path -Name $d.Name -Value $d.Value -Type ExpandString }
    'dword'        { Set-ItemProperty -LiteralPath $d.Path -Name $d.Name -Value ([int32]$d.Value) -Type DWord }
    'qword'        { Set-ItemProperty -LiteralPath $d.Path -Name $d.Name -Value ([int64]$d.Value) -Type QWord }
    'binary'       { Set-ItemProperty -LiteralPath $d.Path -Name $d.Name -Value ([byte[]]($d.Value -split ',' | ForEach-Object { [byte]$_ })) -Type Binary }
    default        { throw "unsupported registry value type: $($d.Type)" }
  }
} finally {
  Remove-Item -LiteralPath $payloadPath -Force -ErrorAction SilentlyContinue
}`

// encodePowerShellCommand renders a script for -EncodedCommand: base64 of UTF-16LE,
// which is the only form that survives without the outer shell re-tokenizing it.
func encodePowerShellCommand(script string) string {
	units := utf16.Encode([]rune(script))
	buf := make([]byte, 0, len(units)*2)
	for _, u := range units {
		// Masking to a byte first makes the narrowing exact rather than implied.
		lo := u & 0xFF
		hi := (u >> 8) & 0xFF
		buf = append(buf, byte(lo), byte(hi))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// registryValueString renders a blueprint value for the environment variable the
// PowerShell body reads. Numeric types are validated here so a YAML string lands as
// a named error instead of reaching the registry as the literal text
// "%!d(string=...)", which is what fmt produced when %d was handed a string.
func registryValueString(regType string, value interface{}) (string, error) {
	switch regType {
	case "dword", "qword":
		n, err := registryInt(value)
		if err != nil {
			return "", fmt.Errorf("invalid %s registry value %v: %w", regType, value, err)
		}
		return strconv.FormatInt(n, 10), nil
	case "binary":
		bytes, err := registryBytes(value)
		if err != nil {
			return "", fmt.Errorf("invalid binary registry value: %w", err)
		}
		parts := make([]string, len(bytes))
		for i, b := range bytes {
			parts[i] = strconv.FormatUint(uint64(b), 10)
		}
		return strings.Join(parts, ","), nil
	default:
		if value == nil {
			return "", nil
		}
		s, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("invalid %s registry value: expected a string, got %T", regType, value)
		}
		return s, nil
	}
}

func registryInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, fmt.Errorf("value %d is too large for a registry integer", v)
		}
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d is too large for a registry integer", v)
		}
		return int64(v), nil
	case float64:
		// JSON and some YAML decoders hand back every number as a float64.
		if v != float64(int64(v)) {
			return 0, fmt.Errorf("not a whole number")
		}
		return int64(v), nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 0, 64)
		if err != nil {
			return 0, fmt.Errorf("not an integer")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("expected an integer, got %T", value)
	}
}

func registryBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case []interface{}:
		out := make([]byte, 0, len(v))
		for _, elem := range v {
			n, err := registryInt(elem)
			if err != nil {
				return nil, err
			}
			if n < 0 || n > 255 {
				return nil, fmt.Errorf("byte value %d out of range", n)
			}
			out = append(out, byte(n))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected a byte sequence, got %T", value)
	}
}
