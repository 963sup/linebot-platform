package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestLoggerRespectsProfileAndSecretRedaction(t *testing.T) {
	config, err := LoadConfig(env(map[string]string{
		KeyLogLevel:  "debug",
		KeyLogFormat: "json",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := NewLogger(config.Log, &buf)
	log.Debug("safe event", "role", "webhook")
	if !strings.Contains(buf.String(), "\"safe event\"") {
		t.Fatal("configured debug level was ignored")
	}
	secrets := SecretConfig{MessagingAPIChannelSecret: "hidden-secret", MessagingAPIChannelAccessToken: "hidden-token"}
	for _, printed := range []string{fmt.Sprintf("%v", secrets), fmt.Sprintf("%#v", secrets)} {
		for _, value := range []string{"hidden-secret", "hidden-token"} {
			if strings.Contains(printed, value) {
				t.Fatal("secret formatting disclosed credential")
			}
		}
	}
}

func TestFullConfigDoesNotExposeSecretsInFormattingOrLogging(t *testing.T) {
	cfg := Config{
		Environment: EnvProduction,
		Secrets: SecretConfig{
			MessagingAPIChannelSecret:      "test-private-secret",
			MessagingAPIChannelAccessToken: "test-private-token",
		},
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var logOutput bytes.Buffer
	logger := NewLogger(LogConfig{Level: "info", Format: "json"}, &logOutput)
	logger.Info("configuration diagnostics", "settings", cfg)

	for _, output := range []string{
		fmt.Sprintf("%v", cfg),
		fmt.Sprintf("%+v", cfg),
		fmt.Sprintf("%#v", cfg),
		string(encoded),
		logOutput.String(),
	} {
		for _, secret := range []string{"test-private-secret", "test-private-token"} {
			if strings.Contains(output, secret) {
				t.Fatal("full configuration disclosed credential value")
			}
		}
	}
}
