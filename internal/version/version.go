package version

var (
	// Build-time variables set via ldflags
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)
