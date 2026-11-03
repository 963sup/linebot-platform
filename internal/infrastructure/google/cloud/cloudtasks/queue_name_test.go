package cloudtasks

import (
	"strings"
	"testing"
)

func TestQueueProjectID(t *testing.T) {
	for _, tt := range []struct {
		name, queue, project string
	}{
		{"valid", "projects/staging-project/locations/asia-east1/queues/line-queue", "staging-project"},
		{"domain-scoped project", "projects/domain.com:project/locations/us-central1/queues/queue-9", "domain.com:project"},
		{"maximum queue ID length", "projects/staging-project/locations/asia-east1/queues/" + strings.Repeat("a", 100), "staging-project"},
		{"invalid project character", "projects/staging_project/locations/asia-east1/queues/line-queue", ""},
		{"invalid location character", "projects/staging-project/locations/asia_east1/queues/line-queue", ""},
		{"invalid queue underscore", "projects/staging-project/locations/asia-east1/queues/line_queue", ""},
		{"invalid queue unicode", "projects/staging-project/locations/asia-east1/queues/測試", ""},
		{"queue ID too long", "projects/staging-project/locations/asia-east1/queues/" + strings.Repeat("a", 101), ""},
		{"wrong prefix", "project/staging-project/locations/asia-east1/queues/line-queue", ""},
		{"extra segment", "projects/staging-project/locations/asia-east1/queues/line-queue/extra", ""},
		{"missing queue", "projects/staging-project/locations/asia-east1/queues/", ""},
		{"missing location", "projects/staging-project/locations//queues/line-queue", ""},
		{"parent traversal", "projects/staging-project/locations/../queues/line-queue", ""},
		{"wrong resource", "projects/staging-project/locations/asia-east1/tasks/line-queue", ""},
		{"empty", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project, ok := QueueProjectID(tt.queue)
			if project != tt.project || ok != (tt.project != "") {
				t.Fatalf("QueueProjectID(%q) = %q, %t; want %q, %t",
					tt.queue, project, ok, tt.project, tt.project != "")
			}
		})
	}
}
