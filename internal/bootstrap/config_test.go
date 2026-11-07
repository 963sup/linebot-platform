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
		{name: "credentials", values: map[string]string{"LINE_MESSAGING_API_CHANNEL_SECRET": "secret", "LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN": "token"}, wantPort: "8080", wantLive: true, wantTimeout: 8 * time.Second},
		{name: "bad port", values: map[string]string{"PORT": "not-a-port"}, wantErr: true},
		{name: "zero", values: map[string]string{"PORT": "0"}, wantErr: true},
		{name: "port overflow", values: map[string]string{"PORT": "70000"}, wantErr: true},
		{name: "partial credentials", values: map[string]string{"LINE_MESSAGING_API_CHANNEL_SECRET": "secret"}, wantErr: true},
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
		"LINE_EVENT_MODE":                   "durable",
		"LINE_MESSAGING_API_CHANNEL_SECRET": "secret",
		"LINE_MESSAGING_API_BOT_USER_ID":    "U0123456789abcdef0123456789abcdef",
		"GCP_PROJECT_ID":                    "test-project",
		"TASK_QUEUE_NAME":                   "projects/test-project/locations/asia-east1/queues/line-queue",
		"TASK_WORKER_URL":                   "https://worker.example.test/tasks/v2/line-reply",
		"TASK_OIDC_AUDIENCE":                "https://worker.example.test",
		"TASK_INVOKER_EMAIL":                "invoker@test-project.iam.gserviceaccount.com",
	}
	cfg, err := LoadConfig(env(fields))
	if err != nil || cfg.EventMode != "durable" || cfg.Durable.ProjectID != "test-project" {
		t.Fatalf("valid durable configuration rejected: %#v %v", cfg, err)
	}
	if !cfg.LiveLINE() || cfg.Secrets.ChannelToken != "" {
		t.Fatal("durable webhook must not use a send token")
	}
	withToken := make(map[string]string, len(fields)+1)
	for key, value := range fields {
		withToken[key] = value
	}
	withToken["LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN"] = "token"
	if _, err := LoadConfig(env(withToken)); err == nil {
		t.Fatal("durable public webhook accepted LINE sending token")
	}
	if _, err := LoadConfig(env(map[string]string{"LINE_EVENT_MODE": "sync", "K_SERVICE": "webhook"})); err == nil {
		t.Fatal("Cloud Run must reject sync mode")
	}
	for _, missing := range []string{"GCP_PROJECT_ID", "TASK_QUEUE_NAME", "TASK_INVOKER_EMAIL", "LINE_MESSAGING_API_BOT_USER_ID"} {
		copy := make(map[string]string, len(fields))
		for k, v := range fields {
			copy[k] = v
		}
		delete(copy, missing)
		if _, err := LoadConfig(env(copy)); err == nil {
			t.Fatalf("durable config accepted missing %s", missing)
		}
	}
	for _, destination := range []string{"typo", "u0123456789abcdef0123456789abcdef", "U0123456789ABCDEF0123456789ABCDEF", "U0123"} {
		copy := make(map[string]string, len(fields))
		for key, value := range fields {
			copy[key] = value
		}
		copy[KeyMessagingAPIBotUserID] = destination
		if _, err := LoadConfig(env(copy)); err == nil {
			t.Fatalf("durable config accepted malformed LINE destination %q", destination)
		}
	}
	workerEnv := make(map[string]string, len(withToken))
	for key, value := range withToken {
		if key != KeyMessagingAPIChannelSecret {
			workerEnv[key] = value
		}
	}
	worker, err := LoadWorkerConfig(env(workerEnv))
	if err != nil || worker.Secrets.ChannelToken != "token" || worker.Durable.ProjectID != "test-project" {
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

func TestMiniAppEntryURLValidation(t *testing.T) {
	valid := []struct {
		name string
		env  string
		url  string
	}{
		{"local loopback", "local", "http://127.0.0.1:8080/miniapp/"},
		{"local hostname", "local", "http://localhost:8080/miniapp/"},
		{"local HTTPS", "local", "https://miniapp.line.me/2000000000-example"},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := LoadConfig(env(map[string]string{KeyAppEnv: test.env, KeyMiniAppLIFFURL: test.url}))
			if err != nil || cfg.MiniAppEntryURL != test.url {
				t.Fatalf("config=%#v err=%v", cfg, err)
			}
		})
	}
	invalid := []string{
		"http://example.test/miniapp/",
		"javascript:alert(1)",
		"https://user@example.test/miniapp/",
		"https://example.test/miniapp/?source=group",
		"https://example.test/miniapp/#fragment",
		"/miniapp/",
	}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			if _, err := LoadConfig(env(map[string]string{KeyMiniAppLIFFURL: value})); err == nil {
				t.Fatalf("invalid MINI App entry URL accepted: %q", value)
			}
		})
	}

	production := map[string]string{
		KeyAppEnv: "production", KeyEventMode: "durable", KeyMessagingAPIChannelSecret: "secret",
		KeyMiniAppLIFFURL:        "http://127.0.0.1:8080/miniapp/",
		KeyMessagingAPIBotUserID: "U0123456789abcdef0123456789abcdef", KeyProjectID: "test-project",
		KeyQueueName:    "projects/test-project/locations/asia-east1/queues/line-queue",
		KeyWorkerURL:    "https://worker.example.test/tasks/v2/line-reply",
		KeyOIDCAudience: "https://worker.example.test", KeyInvokerEmail: "invoker@test-project.iam.gserviceaccount.com",
	}
	if _, err := LoadConfig(env(production)); err == nil {
		t.Fatal("production accepted loopback HTTP MINI App entry URL")
	}
}
