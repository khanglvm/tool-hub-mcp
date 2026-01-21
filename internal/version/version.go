/*
Package version provides version information for tool-hub-mcp.

Version values are set via ldflags during build:
  - Version: git tag (e.g., v1.0.1)
  - Commit: git commit hash (short form)
  - Date: build date in UTC (YYYY-MM-DD)

If not set via ldflags, defaults to "dev" build.
*/
package version

// Version information (set via ldflags during build)
var (
	// Version is the current version (e.g., v1.0.1)
	Version = "dev"
	// Commit is the git commit hash (short form)
	Commit = "none"
	// Date is the build date in UTC (YYYY-MM-DD)
	Date = "unknown"
)

// GetVersion returns version information as a formatted string
func GetVersion() string {
	return FormatVersion(Version, Commit, Date)
}

// FormatVersion formats version components into a display string
func FormatVersion(version, commit, date string) string {
	if version == "dev" {
		return version + " (development build)"
	}
	return version + " (commit: " + commit + ", built: " + date + ")"
}

// GetVersionComponents returns individual version components
func GetVersionComponents() (version, commit, date string) {
	return Version, Commit, Date
}
