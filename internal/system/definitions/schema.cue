// Package providers holds rwr's embedded provider definitions as CUE -
// the single source. The binary embeds these files and evaluates them at
// load; there is no exported or committed second representation.
//
// This schema mirrors types.Provider 1:1 and closes it: an invalid
// definition fails evaluation naming the field.
package providers

// #ConditionRef: what a step condition may reference - the predicates rwr
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
// three actions - and staging at a literal /tmp/ path is the pre-creatable
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
	list?:         string
	listExplicit?: string
	search?:       string
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
