package main

import "github.com/hardhacker/podwise-cli/cmd"

// set by goreleaser via -ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
