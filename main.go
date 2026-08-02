package main

import "github.com/fynxlabs/rwr/cmd"

// Build information. These are populated at release time by goreleaser via
// -ldflags "-X main.version=... -X main.commit=... -X main.date=...
// -X main.builtBy=... -X main.treeState=..." (see .goreleaser.yaml). The names
// and the "main" package path must stay in sync with the ldflags there, or the
// injection silently does nothing.
var (
	version   = "dev"
	commit    = ""
	date      = ""
	builtBy   = ""
	treeState = ""
)

func main() {
	cmd.SetVersionInfo(cmd.BuildInfo{
		Version:   version,
		Commit:    commit,
		Date:      date,
		BuiltBy:   builtBy,
		TreeState: treeState,
	})
	cmd.Execute()
}
