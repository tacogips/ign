// Package build provides build-time information for the CLI application.
// Version is read from VERSION file or set via ldflags during build.
package build

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var embeddedVersion string

// version can be overridden via ldflags:
// -X github.com/tacogips/ign/internal/build.version=x.y.z
var version string

// Version returns the application version.
// Priority: ldflags > embedded VERSION file
func Version() string {
	if version != "" {
		return version
	}
	return strings.TrimSpace(embeddedVersion)
}
