// Package version provides TR-069 protocol version management.
package version

import (
	"fmt"
	"strings"
)

// Version represents a TR-069 protocol version.
type Version struct {
	Major int
	Minor int
}

// String returns the string representation of the version.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// SupportedVersions returns a list of supported TR-069 versions.
var SupportedVersions = []Version{
	{1, 0}, // Original TR-069
	{1, 1}, // Amendment 1
	{1, 2}, // Amendment 2
	{1, 3}, // Amendment 3
	{1, 4}, // Amendment 4
	{1, 5}, // Amendment 5
	{1, 6}, // Amendment 6
}

// DetectVersion detects the TR-069 version from XML namespace.
func DetectVersion(xmlData []byte) Version {
	// Default to original version
	version := Version{1, 0}
	
	// Check for newer versions in namespace declarations
	xmlStr := string(xmlData)
	
	// Check for Amendment 6 (1.4)
	if strings.Contains(xmlStr, "cwmp-1-4") {
		version = Version{1, 4}
	}
	
	// Check for Amendment 5 (1.3)
	if strings.Contains(xmlStr, "cwmp-1-3") {
		version = Version{1, 3}
	}
	
	// Check for Amendment 4 (1.2)
	if strings.Contains(xmlStr, "cwmp-1-2") {
		version = Version{1, 2}
	}
	
	// Check for Amendment 3 (1.1)
	if strings.Contains(xmlStr, "cwmp-1-1") {
		version = Version{1, 1}
	}
	
	// Check for Amendment 2 (1.0) - already default
	
	return version
}

// IsVersionSupported checks if a version is supported.
func IsVersionSupported(version Version) bool {
	for _, supported := range SupportedVersions {
		if supported.Major == version.Major && supported.Minor == version.Minor {
			return true
		}
	}
	return false
}