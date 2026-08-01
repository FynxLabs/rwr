package types

import "fmt"

// A blueprint's wire format and the struct the processors consume are two
// different things, and versioning is what separates them.
//
// Each supported version of a blueprint type is its own struct describing exactly
// what that version accepts on disk. Decoding picks the struct by resolved
// version and then converts it to the canonical type the processors use. So a v2
// of packages is a new struct plus a conversion — nothing downstream of the
// decoder learns that more than one version exists, and no other blueprint type
// is touched.
//
// Adding a version:
//
//  1. define the struct for the new wire format, e.g. packagesV2
//  2. give it a Canonical() that maps it onto PackagesData
//  3. register it under {BlueprintTypePackages: {2: ...}}
//  4. add 2 to supportedSchemaVersions for that type
//
// Nothing else changes, which is the point: a breaking change to one resource
// stays one resource wide instead of moving the whole schema to the next number.

// SchemaVariant is one versioned wire format of a blueprint type.
type SchemaVariant interface {
	// Target returns the value to decode the file into.
	Target() interface{}
	// Canonical returns the decoded content as the type processors consume.
	// For the current version this is the identity.
	Canonical() interface{}
	// DeclaredSchemaVersion reports the version the file declared, or 0.
	DeclaredSchemaVersion() int
}

type variantFactory func() SchemaVariant

// schemaRegistry maps a blueprint type and version to the struct that reads it.
var schemaRegistry = map[string]map[int]variantFactory{
	BlueprintTypePackages:      {1: func() SchemaVariant { return &packagesV1{} }},
	BlueprintTypeRepositories:  {1: func() SchemaVariant { return &repositoriesV1{} }},
	BlueprintTypeFiles:         {1: func() SchemaVariant { return &filesV1{} }},
	BlueprintTypeServices:      {1: func() SchemaVariant { return &servicesV1{} }},
	BlueprintTypeGit:           {1: func() SchemaVariant { return &gitV1{} }},
	BlueprintTypeScripts:       {1: func() SchemaVariant { return &scriptsV1{} }},
	BlueprintTypeSSHKeys:       {1: func() SchemaVariant { return &sshKeysV1{} }},
	BlueprintTypeFonts:         {1: func() SchemaVariant { return &fontsV1{} }},
	BlueprintTypeUsers:         {1: func() SchemaVariant { return &usersV1{} }},
	BlueprintTypeConfiguration: {1: func() SchemaVariant { return &configurationV1{} }},
	BlueprintTypeBootstrap:     {1: func() SchemaVariant { return &bootstrapV1{} }},
}

// NewSchemaVariant returns the struct that reads blueprintType at version.
func NewSchemaVariant(blueprintType string, version int) (SchemaVariant, error) {
	versions, known := schemaRegistry[blueprintType]
	if !known {
		return nil, fmt.Errorf("no schema registered for blueprint type %q", blueprintType)
	}
	factory, ok := versions[version]
	if !ok {
		return nil, ValidateSchemaVersion(blueprintType, version)
	}
	return factory(), nil
}

// HasSchemaVariant reports whether a decoder exists for this type and version.
func HasSchemaVariant(blueprintType string, version int) bool {
	_, ok := schemaRegistry[blueprintType][version]
	return ok
}

// v1 variants. Each wraps the canonical struct, because v1 *is* the current
// shape — the indirection only starts paying off at v2, where the wire struct
// and the canonical struct diverge and Canonical() does real mapping work.

type packagesV1 struct{ PackagesData }

func (v *packagesV1) Target() interface{}        { return &v.PackagesData }
func (v *packagesV1) Canonical() interface{}     { return &v.PackagesData }
func (v *packagesV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type repositoriesV1 struct{ RepositoriesData }

func (v *repositoriesV1) Target() interface{}        { return &v.RepositoriesData }
func (v *repositoriesV1) Canonical() interface{}     { return &v.RepositoriesData }
func (v *repositoriesV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type filesV1 struct{ FileData }

func (v *filesV1) Target() interface{}        { return &v.FileData }
func (v *filesV1) Canonical() interface{}     { return &v.FileData }
func (v *filesV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type servicesV1 struct{ ServiceData }

func (v *servicesV1) Target() interface{}        { return &v.ServiceData }
func (v *servicesV1) Canonical() interface{}     { return &v.ServiceData }
func (v *servicesV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type gitV1 struct{ GitData }

func (v *gitV1) Target() interface{}        { return &v.GitData }
func (v *gitV1) Canonical() interface{}     { return &v.GitData }
func (v *gitV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type scriptsV1 struct{ ScriptData }

func (v *scriptsV1) Target() interface{}        { return &v.ScriptData }
func (v *scriptsV1) Canonical() interface{}     { return &v.ScriptData }
func (v *scriptsV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type sshKeysV1 struct{ SSHKeyData }

func (v *sshKeysV1) Target() interface{}        { return &v.SSHKeyData }
func (v *sshKeysV1) Canonical() interface{}     { return &v.SSHKeyData }
func (v *sshKeysV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type fontsV1 struct{ FontsData }

func (v *fontsV1) Target() interface{}        { return &v.FontsData }
func (v *fontsV1) Canonical() interface{}     { return &v.FontsData }
func (v *fontsV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type usersV1 struct{ UsersData }

func (v *usersV1) Target() interface{}        { return &v.UsersData }
func (v *usersV1) Canonical() interface{}     { return &v.UsersData }
func (v *usersV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type bootstrapV1 struct{ BootstrapData }

func (v *bootstrapV1) Target() interface{}        { return &v.BootstrapData }
func (v *bootstrapV1) Canonical() interface{}     { return &v.BootstrapData }
func (v *bootstrapV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }

type configurationV1 struct{ ConfigData }

func (v *configurationV1) Target() interface{}        { return &v.ConfigData }
func (v *configurationV1) Canonical() interface{}     { return &v.ConfigData }
func (v *configurationV1) DeclaredSchemaVersion() int { return v.DeclaredVersion() }
