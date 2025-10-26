package version

// Version information variables injected at build time via -ldflags
// These variables are set by the build script (scripts/version.sh) and Taskfile
var (
	// Version is the semantic version number (tag+commit or date+commit)
	// Examples: "v1.2.3-a1b2c3d", "20251026-a1b2c3d", "dev"
	Version = "dev"

	// BuildTime is the RFC3339 formatted build timestamp
	// Example: "2025-10-26T10:30:00Z"
	BuildTime = "unknown"

	// CommitID is the 7-character short git commit hash
	// Example: "a1b2c3d"
	CommitID = "0000000"
)

// Info represents version information
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	CommitID  string `json:"commit_id"`
}

// GetInfo returns version information as a struct
func GetInfo() Info {
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		CommitID:  CommitID,
	}
}
