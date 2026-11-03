// Package cloudtasks constructs the native Google Cloud Tasks client and
// validates resource names without connecting to Google Cloud.
package cloudtasks

import "strings"

// QueueProjectID validates a Cloud Tasks queue resource name and extracts its
// project ID. Structural validation is shared by preflight and the adapter;
// existence, IAM and location are checked only in an authorized environment.
func QueueProjectID(name string) (string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "queues" {
		return "", false
	}
	// The Cloud Tasks API permits letters, digits and hyphens in queue IDs
	// (up to 100 characters). Project IDs additionally permit dots/colons.
	// Location IDs use canonical codes; their existence is checked in staging.
	if !validResourceID(parts[1], true) || !validResourceID(parts[3], false) ||
		!validResourceID(parts[5], false) || len(parts[5]) > 100 {
		return "", false
	}
	return parts[1], true
}

func validResourceID(value string, project bool) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || (project && (c == '.' || c == ':')) {
			continue
		}
		return false
	}
	return true
}
