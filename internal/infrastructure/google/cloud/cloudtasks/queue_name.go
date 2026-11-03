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
	for _, index := range [...]int{1, 3, 5} {
		if parts[index] == "" || parts[index] == "." || parts[index] == ".." {
			return "", false
		}
	}
	return parts[1], true
}
