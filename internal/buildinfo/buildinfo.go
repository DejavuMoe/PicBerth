// Package buildinfo carries build-time identity. Version and Commit are set with
// -ldflags "-X picberth/internal/buildinfo.Version=v1.2.3 -X picberth/internal/buildinfo.Commit=abc123".
package buildinfo

import "runtime"

var (
	Version = "v0.0.1-dev"
	Commit  = "unknown"
)

// GoVersion is the toolchain the binary was built with.
func GoVersion() string { return runtime.Version() }
