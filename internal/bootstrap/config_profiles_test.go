package bootstrap

import (
	"testing"
)

func durableSettings() map[string]string {
	return map[string]string{
		KeyEventMode:      "durable",
		KeyChannelSecret:  "test-secret",
		KeyBotDestination: "U0123456789abcdef0123456789abcdef",
		KeyProjectID:      "test-project",
		KeyQueueName:      "projects/test-project/locations/asia-east1/queues/line-queue",
		KeyWorkerURL:      "https://worker.example.test/tasks/line-reply",
		KeyOIDCAudience:   "https://worker.example.test",
		KeyInvokerEmail:   "invoker@test-project.iam.gserviceaccount.com",
	}
}

func TestEnvironmentProfilesAndRoleSecretIsolation(t *testing.T) {
	local, err := LoadConfig(env(nil))
	if err != nil || local.Environment != EnvLocal || local.EventMode != "sync" ||
		local.Log.Level != "info" || local.Log.Format != "text" {
		t.Fatalf("invalid safe local defaults: config=%#v err=%v", local, err)
	}
	cloud := durableSettings()
	cloud[KeyAppEnv] = "staging"
	cloud[KeyCloudRunService] = "linebot-webhook"
	webhook, err := LoadConfig(env(cloud))
	if err != nil || webhook.Environment != EnvStaging || webhook.Log.Format != "json" ||
		webhook.Secrets.ChannelToken != "" || !webhook.LiveLINE() {
		t.Fatalf("invalid tokenless cloud webhook: config=%#v err=%v", webhook, err)
	}
	workerEnv := durableSettings()
	delete(workerEnv, KeyChannelSecret)
	workerEnv[KeyAppEnv] = "production"
	workerEnv[KeyChannelToken] = "private-send-token"
	workerEnv[KeyCloudRunService] = "linebot-worker"
	worker, err := LoadWorkerConfig(env(workerEnv))
	if err != nil || worker.Secrets.ChannelToken == "" ||
		worker.Secrets.ChannelSecret != "" || worker.EventMode != "durable" {
		t.Fatalf("invalid private worker: config=%#v err=%v", worker, err)
	}
}

func TestEnvironmentFailsClosedWithoutSecretDisclosure(t *testing.T) {
	for _, tt := range []struct {
		name   string
		config map[string]string
		worker bool
	}{
		{"invalid profile", map[string]string{KeyAppEnv: "prod"}, false},
		{"production sync rejected", map[string]string{KeyAppEnv: "production"}, false},
		{"cloud metadata with local profile", map[string]string{KeyCloudRunService: "service"}, false},
		{"bad logging level", map[string]string{KeyLogLevel: "trace"}, false},
		{"staging text logs rejected", map[string]string{KeyAppEnv: "staging", KeyLogFormat: "text"}, false},
		{"worker missing token", durableSettings(), true},
		{"durable webhook send token", func() map[string]string {
			v := durableSettings()
			v[KeyChannelToken] = "do-not-leak-secret"
			return v
		}(), false},
		{"worker with signing secret", func() map[string]string {
			v := durableSettings()
			v[KeyChannelToken] = "worker-token"
			return v
		}(), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.worker {
				_, err = LoadWorkerConfig(env(tt.config))
			} else {
				_, err = LoadConfig(env(tt.config))
			}
			if err == nil {
				t.Fatal("expected unsafe configuration to be rejected")
			}
			if tt.config[KeyChannelToken] != "" && tt.config[KeyChannelToken] == err.Error() {
				t.Fatal("credential appeared in error")
			}
		})
	}
}

func TestSettingContractUniqueAndSecretClassified(t *testing.T) {
	seen := make(map[string]bool)
	for _, spec := range SettingsContract() {
		if seen[spec.Key] {
			t.Fatalf("duplicate setting key: %s", spec.Key)
		}
		if spec.Key == "" || spec.Roles == "" || spec.Required == "" {
			t.Fatal("incomplete setting contract")
		}
		seen[spec.Key] = true
	}
	for _, secret := range []string{KeyChannelSecret, KeyChannelToken} {
		found := false
		for _, spec := range SettingsContract() {
			if spec.Key == secret {
				found = spec.Secret
			}
		}
		if !found {
			t.Fatalf("secret classification missing: %s", secret)
		}
	}
}

func TestWorkerRejectsIncompatibleWebhookMode(t *testing.T) {
	values := durableSettings()
	delete(values, KeyChannelSecret)
	values[KeyChannelToken] = "private-worker-token"
	values[KeyEventMode] = "sync"
	if _, err := LoadWorkerConfig(env(values)); err == nil {
		t.Fatal("worker accepted explicit sync mode instead of failing closed")
	}
	values[KeyEventMode] = "not-a-mode"
	if _, err := LoadWorkerConfig(env(values)); err == nil {
		t.Fatal("worker silently ignored unknown event mode")
	}
	values[KeyEventMode] = "durable"
	if _, err := LoadWorkerConfig(env(values)); err != nil {
		t.Fatalf("valid shared durable setting rejected: %v", err)
	}
	delete(values, KeyEventMode)
	if _, err := LoadWorkerConfig(env(values)); err != nil {
		t.Fatalf("worker should use implicit durable mode: %v", err)
	}
}
