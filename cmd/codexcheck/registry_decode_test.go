package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDynamicAgentRegistry(t *testing.T) {
	body := `[agents]
enabled = true
max_concurrent_threads_per_session = 9
max_depth = 1
[agents.example]
description = "test role"
config_file = "agents/example.toml"
`
	for _, tc := range []struct {
		name, extra string
		wantError   bool
	}{
		{"supported", "", false},
		{"unknown-field", "unknown_field = true\n", true},
		{"nested-unknown", "[agents.example.extra]\nunknown = true\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(body+tc.extra), 0600); err != nil {
				t.Fatal(err)
			}
			var c config
			err := readTOML(path, &c)
			if (err != nil) != tc.wantError {
				t.Fatalf("readTOML error=%v, wantError=%v", err, tc.wantError)
			}
		})
	}
}
