package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Initialize loads and parses the init configuration file from a local path or URL.
// It resolves template variables, sets up system paths, and returns the fully
// populated InitConfig used to drive all subsequent blueprint processing.
func Initialize(initFilePath string, flags types.Flags) (*types.InitConfig, error) {
	var initConfig types.InitConfig
	var err error
	var fileExt string
	var tempInitFile string

	// Create a temporary directory for downloaded or processed init files
	tempDir, err := os.MkdirTemp("", "rwr-init-")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// The init file decides the run order, the blueprint git remote and which
	// credentials blueprints may read, so anyone who can rewrite it in flight owns
	// the run. Over http:// that is anyone on the path between here and the server.
	if strings.HasPrefix(initFilePath, "http://") {
		return nil, fmt.Errorf("refusing to fetch the init file over http://: it is served in cleartext and drives everything rwr runs; use https:// instead (%s)", initFilePath)
	}

	// Handle URL or local file. GitHub blob URLs and owner/repo shorthands were
	// already rewritten to raw https URLs by helpers.ResolveInitSource.
	if strings.HasPrefix(initFilePath, "https://") {
		log.Debugf("Init File is a Web URL, Downloading %s", initFilePath)

		fileExt = filepath.Ext(initFilePath)
		tempInitFile = filepath.Join(tempDir, "init"+fileExt)

		err = system.DownloadFile(initFilePath, tempInitFile, false)
		if err != nil {
			return nil, fmt.Errorf("error downloading init file: %w", err)
		}
	} else {
		log.Debugf("Init File is local path: %s", initFilePath)
		if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("init file not found at path: %s", initFilePath)
		}
		fileExt = filepath.Ext(initFilePath)
		tempInitFile = initFilePath
		log.Debugf("Using init file: %s", tempInitFile)
	}

	log.Debugf("Reading in temporary Init File: %s", tempInitFile)

	// Read the init file
	initFileData, err := os.ReadFile(tempInitFile) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		return nil, fmt.Errorf("error reading init file %s: %w", tempInitFile, err)
	}

	// Set default variables
	variables, err := setDefaultVariables()
	if err != nil {
		return nil, err
	}

	// Process the init file as a template
	processedInit, err := helpers.ResolveTemplate(initFileData, variables)
	if err != nil {
		return nil, fmt.Errorf("error processing init file as a template: %w", err)
	}
	// Viper does not read TOML the way rwr's decoders do, and cannot read CUE
	// at all: both are pre-converted (the registry decides the format rather
	// than a literal extension compare).
	switch format, formatErr := helpers.FormatForPath(tempInitFile); {
	case formatErr == nil && format == types.FormatTOML:
		processedInit, fileExt, err = convertTomlToYaml(processedInit)
		if err != nil {
			return nil, err
		}
	case formatErr == nil && format == types.FormatCUE:
		processedInit, fileExt, err = convertCueToJSON(processedInit, tempInitFile)
		if err != nil {
			return nil, err
		}
	}

	// Write the processed init file to the temporary directory
	processedInitFile := filepath.Join(tempDir, "init-processed"+fileExt)
	err = os.WriteFile(processedInitFile, processedInit, 0644) // #nosec G306 G703 -- TODO(PR8): create with target mode instead of chmod-after; TODO(PR8): path derived from operator blueprint input; containment added in PR8
	if err != nil {
		return nil, fmt.Errorf("error writing processed init file: %w", err)
	}

	log.Debugf("Processed Init File Path: %s", processedInitFile)

	// Read the processed init file with Viper
	viper.SetConfigFile(processedInitFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading init file into viper: %w", err)
	}

	// Unmarshal the init file into the InitConfig struct
	if err := viper.Unmarshal(&initConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling %s: %w", processedInitFile, err)
	}

	// A tree-wide schema version applies to every blueprint type, so it has to be
	// readable everywhere. Checked here rather than per file: an unreadable tree
	// version is wrong before a single blueprint is opened.
	if err := types.ValidateTreeSchemaVersion(initConfig.Init.SchemaVersion); err != nil {
		return nil, fmt.Errorf("error in %s: %w", initFilePath, err)
	}

	// The runtime halves of Variables (user, system, flags) are computed, not
	// declared, so they are filled in here. userDefined is the one part that comes
	// from the init file, so it has to survive: assigning the whole struct threw
	// away everything the operator wrote under `variables:`.
	declared := initConfig.Variables.UserDefined
	initConfig.Variables = variables
	initConfig.Variables.Flags = flags
	if initConfig.Variables.UserDefined == nil {
		initConfig.Variables.UserDefined = make(map[string]interface{})
	}
	for key, value := range declared {
		initConfig.Variables.UserDefined[key] = value
	}

	// Set the blueprints location
	setBlueprintsLocation(&initConfig, initFilePath)

	// Apply the credential opt-in before anything reads variables or spawns a
	// command, so the choice is in effect for the whole run.
	types.SetExposedCredentials(initConfig.ExposeCredentials)
	if len(initConfig.ExposeCredentials) > 0 {
		log.Warnf("Blueprints in this tree can read these credentials: %v — "+
			"they are readable by any script or template the blueprints run",
			types.ExposedCredentials())
	}

	// Set user-defined variables and environment variables
	if err := setUserDefinedAndEnvVariables(&initConfig); err != nil {
		return nil, fmt.Errorf("error setting variables: %w", err)
	}

	log.Debugf("Initialized initConfig: %v", initConfig)

	return &initConfig, nil
}

// setDefaultVariables is helpers.DefaultVariables, shared with `rwr validate` so
// both render blueprints against the same variables.
func setDefaultVariables() (types.Variables, error) {
	return helpers.DefaultVariables()
}

// convertCueToJSON evaluates a CUE init file to concrete JSON for viper, the
// same shape as the TOML→YAML pre-conversion below.
func convertCueToJSON(data []byte, filename string) ([]byte, string, error) {
	log.Debugf("CUE format detected, evaluating to JSON for viper")
	jsonData, err := helpers.EvalCUEToJSON(data, filename)
	if err != nil {
		return nil, "", err
	}
	return jsonData, ".json", nil
}

func convertTomlToYaml(data []byte) ([]byte, string, error) {
	log.Debugf("TOML Format detected, converting to yaml for viper")
	var tempMap map[string]interface{}
	if _, err := toml.Decode(string(data), &tempMap); err != nil {
		return nil, "", fmt.Errorf("error decoding TOML: %w", err)
	}
	yamlData, err := yaml.Marshal(tempMap)
	if err != nil {
		return nil, "", fmt.Errorf("error converting TOML to YAML: %w", err)
	}
	return yamlData, ".yaml", nil
}

func setBlueprintsLocation(initConfig *types.InitConfig, initFilePath string) {
	// Handle Git target setup if needed
	if initConfig.Init.Git != nil && initConfig.Init.Git.Target != "" {
		resolvedTarget := system.ExpandPath(initConfig.Init.Git.Target)
		if err := os.MkdirAll(resolvedTarget, 0755); err != nil { // #nosec G301 -- TODO(PR8): blueprint-target directory; create with the requested mode
			log.Warnf("Failed to create blueprint directory: %v", err)
		}
	}

	// Set location based on init file rules
	if initConfig.Init.Location == "" || initConfig.Init.Location == "." {
		initConfig.Init.Location = filepath.Dir(initFilePath)
	} else if initConfig.Init.Location == "~" || strings.HasPrefix(initConfig.Init.Location, "~/") {
		homeDir, _ := os.UserHomeDir() //nolint:errcheck
		initConfig.Init.Location = filepath.Join(homeDir, initConfig.Init.Location[2:])
	} else if !filepath.IsAbs(initConfig.Init.Location) {
		initConfig.Init.Location = filepath.Join(filepath.Dir(initFilePath), initConfig.Init.Location)
	}

	log.Debugf("Blueprints location set to: %s", initConfig.Init.Location)
}

func setUserDefinedAndEnvVariables(initConfig *types.InitConfig) error {

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "RWR_") {
			parts := strings.SplitN(env, "=", 2)
			key := strings.TrimPrefix(parts[0], "RWR_")
			initConfig.Variables.UserDefined[key] = parts[1]
		}
	}

	// Export config values as RWR_VAR_* so blueprints and scripts can read them —
	// except the credentials. setupCommandEnvironment copies os.Environ() into
	// every command rwr spawns, so exporting the GitHub token and the base64 SSH
	// private key put both in reach of any script a blueprint chooses to run, and
	// blueprints are cloned from git repositories.
	for _, key := range viper.AllKeys() {
		if types.IsSecretConfigKey(key) && !types.IsCredentialExposed(key) {
			log.Debugf("Not exporting %s to the environment: it holds a credential "+
				"(add it to exposeCredentials in the init file if a script needs it)", key)
			continue
		}

		value := viper.GetString(key)
		envKey := fmt.Sprintf("RWR_VAR_%s", strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
		if err := os.Setenv(envKey, value); err != nil {
			return fmt.Errorf("error setting environment variable %s: %w", envKey, err)
		}
	}
	return nil
}
