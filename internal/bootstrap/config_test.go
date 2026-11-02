package bootstrap

import "testing"

func env(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		wantPort string
		wantLive bool
		wantErr  bool
	}{
		{name: "defaults", values: nil, wantPort: "8080"},
		{name: "custom", values: map[string]string{"PORT": "8081"}, wantPort: "8081"},
		{name: "credentials", values: map[string]string{"LINE_CHANNEL_SECRET": "secret", "LINE_CHANNEL_ACCESS_TOKEN": "token"}, wantPort: "8080", wantLive: true},
		{name: "bad port", values: map[string]string{"PORT": "not-a-port"}, wantErr: true},
		{name: "zero", values: map[string]string{"PORT": "0"}, wantErr: true},
		{name: "port overflow", values: map[string]string{"PORT": "70000"}, wantErr: true},
		{name: "partial credentials", values: map[string]string{"LINE_CHANNEL_SECRET": "secret"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(env(tt.values))
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && (cfg.Port != tt.wantPort || cfg.LiveLINE() != tt.wantLive) {
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
