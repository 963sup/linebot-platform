package bootstrap

import (
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		values      map[string]string
		wantPort    string
		wantLive    bool
		wantTimeout time.Duration
		wantErr     bool
	}{
		{name: "defaults", values: nil, wantPort: "8080", wantTimeout: 8 * time.Second},
		{name: "custom", values: map[string]string{"PORT": "8081", "LINE_API_TIMEOUT": "6s"}, wantPort: "8081", wantTimeout: 6 * time.Second},
		{name: "credentials", values: map[string]string{"LINE_CHANNEL_SECRET": "secret", "LINE_CHANNEL_ACCESS_TOKEN": "token"}, wantPort: "8080", wantLive: true, wantTimeout: 8 * time.Second},
		{name: "bad port", values: map[string]string{"PORT": "not-a-port"}, wantErr: true},
		{name: "zero", values: map[string]string{"PORT": "0"}, wantErr: true},
		{name: "port overflow", values: map[string]string{"PORT": "70000"}, wantErr: true},
		{name: "partial credentials", values: map[string]string{"LINE_CHANNEL_SECRET": "secret"}, wantErr: true},
		{name: "invalid duration", values: map[string]string{"LINE_API_TIMEOUT": "bad"}, wantErr: true},
		{name: "zero duration", values: map[string]string{"LINE_API_TIMEOUT": "0s"}, wantErr: true},
		{name: "too short", values: map[string]string{"LINE_API_TIMEOUT": "500ms"}, wantErr: true},
		{name: "too long", values: map[string]string{"LINE_API_TIMEOUT": "13s"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(env(tt.values))
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && (cfg.Port != tt.wantPort || cfg.LiveLINE() != tt.wantLive || cfg.LINEAPITimeout != tt.wantTimeout) {
				t.Fatalf("unexpected config: %#v", cfg)
			}
		})
	}
}

func TestLoadConfigNilEnv(t *testing.T) {
	if _, err := LoadConfig(nil); err == nil {
		t.Fatal("expected error for nil environment reader")
	}
}

func TestDurableConfigRequiresAllCredentialsAndEndpoints(t *testing.T) {
	fields := map[string]string{
		"LINE_EVENT_MODE": "durable",
		"LINE_CHANNEL_SECRET": "secret",
		"LINE_CHANNEL_ACCESS_TOKEN": "token",
		"LINE_BOT_DESTINATION": "U0123456789abcdef0123456789abcdef",
		"GCP_PROJECT_ID": "test-project",
		"TASK_QUEUE_NAME": "projects/test-project/locations/asia-east1/queues/line-queue",
		"TASK_WORKER_URL": "https://worker.example.test/tasks/line-reply",
		"TASK_OIDC_AUDIENCE": "https://worker.example.test",
		"TASK_INVOKER_EMAIL": "invoker@test-project.iam.gserviceaccount.com",
	}
	cfg, err := LoadConfig(env(fields))
	if err != nil || cfg.EventMode != "durable" || cfg.Durable.ProjectID != "test-project" {
		t.Fatalf("valid durable configuration rejected: %#v %v", cfg, err)
	}
	if _, err := NewWebhookRouter(cfg, nil); err == nil {
		t.Fatal("durable router must require explicit runtime ownership")
	}
	for _, missing := range []string{"GCP_PROJECT_ID", "TASK_QUEUE_NAME", "TASK_INVOKER_EMAIL", "LINE_BOT_DESTINATION"} {
		copy := make(map[string]string, len(fields))
		for k, v := range fields { copy[k] = v }
		delete(copy, missing)
		if _, err := LoadConfig(env(copy)); err == nil {
			t.Fatalf("durable config accepted missing %s", missing)
		}
	}
	worker, err := LoadWorkerConfig(env(fields))
	if err != nil || worker.ChannelToken != "token" || worker.Durable.ProjectID != "test-project" {
		t.Fatalf("worker config rejected: %#v %v", worker, err)
	}
	if _, err := LoadWorkerConfig(env(map[string]string{})); err == nil {
		t.Fatal("worker must never start with empty configuration")
	}
}

func TestInvalidEventModeRejected(t *testing.T) {
	if _, err := LoadConfig(env(map[string]string{"LINE_EVENT_MODE": "unknown"})); err == nil {
		t.Fatal("invalid event mode must fail")
	}
}
