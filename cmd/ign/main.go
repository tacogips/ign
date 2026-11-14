package main

import (
	"github.com/tacogips/ign/internal/cli"
)

// Version information (set via ldflags during build)
var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func main() {
	// Set version info from build-time variables
	cli.Version = version
	cli.GitCommit = gitCommit
	cli.BuildDate = buildDate

	// Execute the root command
	cli.Execute()
}
