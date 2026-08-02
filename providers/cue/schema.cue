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

#ActionStep: {
	action:     string
	path?:      string
	match?:     string
	section?:   string
	source?:    string
	dest?:      string
	sha256?:    string
	exec?:      string
	args?: [...string]
	content?:   string
	condition?: string
}

#Detection: {
	binary: string
	files?: [...string]
	distributions?: [...string]
}

#Commands: {
	install?: string
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
	commands?: #Commands
	repository?: #Repository
	corePackages?: {[string]: [...string]}
	alternatives?: {[string]: #Alternatives}
	install?: {steps?: [...#ActionStep]}
	remove?: {steps?: [...#ActionStep]}
	environment?: {[string]: string}
}

// providers is the exported value: one entry per definition file.
providers: {[Name=string]: #Provider & {name: Name}}
