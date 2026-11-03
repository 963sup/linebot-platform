package cloudtasks

import "testing"

func TestQueueProjectID(t *testing.T) {
	for _, tt := range []struct {
		name, queue, project string
	}{
		{"valid", "projects/staging-project/locations/asia-east1/queues/line-queue", "staging-project"},
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
