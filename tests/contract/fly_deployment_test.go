package contract

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

type flyConfig struct {
	App           string `toml:"app"`
	PrimaryRegion string `toml:"primary_region"`
	Build         struct {
		Dockerfile string `toml:"dockerfile"`
	} `toml:"build"`
	Env         map[string]string `toml:"env"`
	HTTPService struct {
		InternalPort       int    `toml:"internal_port"`
		ForceHTTPS         bool   `toml:"force_https"`
		AutoStopMachines   any    `toml:"auto_stop_machines"`
		MinMachinesRunning int    `toml:"min_machines_running"`
		Checks             []struct {
			Method string `toml:"method"`
			Path   string `toml:"path"`
		} `toml:"checks"`
	} `toml:"http_service"`
	Files []struct {
		GuestPath  string `toml:"guest_path"`
		SecretName string `toml:"secret_name"`
		LocalPath  string `toml:"local_path"`
		RawValue   string `toml:"raw_value"`
	} `toml:"files"`
}

func TestFlyStagingConfigurationContracts(t *testing.T) {
	for _, tc := range []struct {
		file       string
		dockerfile string
		app        string
	}{
		{file: "fly.toml", dockerfile: "Dockerfile", app: "linebot-platform"},
		{file: "fly.worker.toml", dockerfile: "Dockerfile.worker", app: "linebot-worker-staging"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			var cfg flyConfig
			if _, err := toml.DecodeFile(filepath.Join("../..", tc.file), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.App != tc.app || cfg.PrimaryRegion != "sin" {
				t.Fatalf("Fly app name or region differs from the existing configuration: %q", cfg.App)
			}
			if cfg.Build.Dockerfile != tc.dockerfile {
				t.Fatalf("wrong Dockerfile: %q", cfg.Build.Dockerfile)
			}
			if cfg.Env["APP_ENV"] != "staging" || cfg.Env["LINE_EVENT_MODE"] != "durable" ||
				cfg.Env["LOG_FORMAT"] != "json" || cfg.Env["PORT"] != "8080" {
				t.Fatal("staging runtime settings must be safe and explicit")
			}
			for _, secret := range []string{"LINE_CHANNEL_SECRET", "LINE_CHANNEL_ACCESS_TOKEN",
				"GCP_ADC_JSON_B64", "TASK_INVOKER_EMAIL", "TASK_WORKER_URL"} {
				if _, exists := cfg.Env[secret]; exists {
					t.Fatalf("%s must not be hardcoded in [env]", secret)
				}
			}
			// Fly supports both bool false and string "off" for disabling autostop.
			// Never force a renamed Fly app just to satisfy a suffix convention.
			autoStopDisabled := false
			switch value := cfg.HTTPService.AutoStopMachines.(type) {
			case bool:
				autoStopDisabled = !value
			case string:
				autoStopDisabled = value == "off"
			}
			if cfg.HTTPService.InternalPort != 8080 || !cfg.HTTPService.ForceHTTPS ||
				!autoStopDisabled || cfg.HTTPService.MinMachinesRunning != 1 {
				t.Fatal("staging HTTP service must remain on HTTPS with autostop disabled")
			}
			if len(cfg.HTTPService.Checks) != 1 ||
				cfg.HTTPService.Checks[0].Path != "/healthz" ||
				cfg.HTTPService.Checks[0].Method != "GET" {
				t.Fatal("health checks must use GET /healthz")
			}
			if cfg.Env["GOOGLE_APPLICATION_CREDENTIALS"] != "/gcp-credentials.json" ||
				len(cfg.Files) != 1 ||
				cfg.Files[0].GuestPath != "/gcp-credentials.json" ||
				cfg.Files[0].SecretName != "GCP_ADC_JSON_B64" ||
				cfg.Files[0].RawValue != "" || cfg.Files[0].LocalPath != "" {
				t.Fatal("ADC must be injected from an app-scoped Fly secret, not embedded in the image or config")
			}
		})
	}
}
