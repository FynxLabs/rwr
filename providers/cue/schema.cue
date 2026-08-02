// Package providers holds rwr's embedded provider definitions as CUE.
//
// CUE is a build-time tool only: these sources are exported to JSON
// (mise run providers:export), the JSON is committed under
// internal/system/definitions/providers/ and embedded into the binary.
// cuelang.org/go never ships in rwr for providers.
//
// This schema mirrors types.Provider 1:1; the strict round-trip test in
// internal/system enforces that every exported provider decodes into
// types.Provider with no unknown keys. Contracts (required fields, action
// enums, predicate names) tighten here in the follow-up task; task 1 is the
// mechanical shape.
package providers

// #ConditionRef: what a step condition may reference — the predicates rwr
// derives (repositoryPredicates; a Go test asserts this list equals its keys)
// plus the two step-data fields shipped conditions use. A condition naming
// anything else decoded into nothing historically, so every step ran.
#ConditionRef: "HasKey" | "HasInterfaces" | "HasSlot" | "HasProxy" |
	"HasToken" | "HasAuthentication" | "RequiresAuth" | "IsCustomRegistry" |
	"IsOverlay" | "IsMainRepo" | "IsLocalFile" | "IsLocalSnap" |
	"IsSnapStore" | "UserMode" |
	"ResetSettings" | "URL" // step-data fields, not predicates

#conditionPattern: "\\{\\{.*\\.(HasKey|HasInterfaces|HasSlot|HasProxy|HasToken|HasAuthentication|RequiresAuth|IsCustomRegistry|IsOverlay|IsMainRepo|IsLocalFile|IsLocalSnap|IsSnapStore|UserMode|ResetSettings|URL).*\\}\\}"

// Repository action steps: the enum is exactly what the repository processor
// implements; anything else stops the run there, so it fails export here.
#ActionStep: {
	action:     "exec" | "command" | "download" | "write" | "copy" | "append" | "remove_line" | "remove_section" | "remove"
	path?:      string
	match?:     string
	section?:   string
	source?:    string
	dest?:      string
	sha256?:    string
	exec?:      string
	args?: [...string]
	content?:   string
	condition?: string & =~#conditionPattern
}

// Install/remove steps: the package-manager processor implements only these
// three actions — and staging at a literal /tmp/ path is the pre-creatable
// world-known name {{ .TempDir }} exists to eliminate, so it cannot export.
#InstallStep: {
	action:  "command" | "download" | "write"
	source?: string & !~"/tmp/"
	dest?:   string & !~"/tmp/"
	sha256?: string
	exec?:   string & !~"/tmp/"
	args?: [...string & !~"/tmp/"]
	content?:   string & !~"/tmp/"
	condition?: string & =~#conditionPattern
}

#Detection: {
	binary: string
	files?: [...string]
	distributions?: [...string]
}

#Commands: {
	install: string
	update?:  string
	remove?:  string
	list?:    string
	search?:  string
	clean?:   string
}

#RepositoryPaths: {
	sources?: string
	keys?:    string
	config?:  string
}

#RepositoryAction: {
	steps?: [...#ActionStep]
}

#Repository: {
	paths?:  #RepositoryPaths
	add?:    #RepositoryAction
	remove?: #RepositoryAction
}

#Alternatives: {
	corePackages?: {[string]: [...string]}
}

#Provider: {
	name:      string
	elevated?: bool
	detection: #Detection
	commands: #Commands
	repository?: #Repository
	corePackages?: {[string]: [...string]}
	alternatives?: {[string]: #Alternatives}
	install?: {steps?: [...#InstallStep]}
	remove?: {steps?: [...#InstallStep]}
	environment?: {[string]: string}
}

// providers is the exported value: one entry per definition file.
providers: {[Name=string]: #Provider & {name: Name}}
