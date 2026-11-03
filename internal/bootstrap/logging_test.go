package bootstrap

import (
	"bytes"
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
	secrets := SecretConfig{ChannelSecret: "hidden-secret", ChannelToken: "hidden-token"}
	for _, printed := range []string{fmt.Sprintf("%v", secrets), fmt.Sprintf("%#v", secrets)} {
		for _, value := range []string{"hidden-secret", "hidden-token"} {
			if strings.Contains(printed, value) {
				t.Fatal("secret formatting disclosed credential")
			}
		}
	}
}
