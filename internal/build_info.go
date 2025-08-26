// Package internal provides internal utilities and shared functionality
// for the monitoring application.
package internal

import (
	"fmt"
)

// PrintBuildInfo prints build version, date and commit information to stdout.
// This function is used to display application build metadata during startup.
//
// Parameters:
//   - buildVersion: the application version string (e.g., "v1.0.0")
//   - buildDate: the build timestamp in string format
//   - buildCommit: the git commit hash used for the build
//
// If any parameter is an empty string, it will be displayed as "N/A".
//
// Example output:
//
//	Build version: v1.0.0
//	Build date: 2024-01-15_14:30:25
//	Build commit: a1b2c3d
//
// Usage:
//
//	PrintBuildInfo(version, date, commit)
func PrintBuildInfo(buildVersion string, buildDate string, buildCommit string) {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
