package bootstrap

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Environment selects security defaults and the accepted runtime profile.
type Environment string

const (
	EnvLocal      Environment = "local"
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
)

type RuntimeRole string

const (
	RoleWebhook RuntimeRole = "webhook"
	RoleWorker  RuntimeRole = "worker"
	RoleMiniApp RuntimeRole = "miniapp"
)

// SecretConfig is deliberately separate from ordinary runtime settings.
// Both fmt printing and %#v redact its values; never log or marshal this type.
type SecretConfig struct {
	MessagingAPIChannelSecret      string
	MessagingAPIChannelAccessToken string
}

func (SecretConfig) String() string   { return "[redacted secrets]" }
func (SecretConfig) GoString() string { return "bootstrap.SecretConfig{redacted}" }

// LogConfig contains safe, non-sensitive logging settings.
type LogConfig struct {
	Level  string
	Format string
}

// TelemetryConfig selects the optional local OTLP export path. Infrastructure
// fixes the endpoint so configuration cannot redirect telemetry or attach
// exporter credentials.
type TelemetryConfig struct {
	Mode             string
	TraceSampleRatio float64
	MetricInterval   time.Duration
}

// Config holds ordinary settings and explicitly isolated credentials.
// Existing callers must use the typed loader for role-specific validation.
type Config struct {
	Environment     Environment
	Role            RuntimeRole
	Port            string
	EventMode       string
	LINEAPITimeout  time.Duration
	MiniAppEntryURL string
	Durable         DurableConfig
	Log             LogConfig
	Telemetry       TelemetryConfig
	Secrets         SecretConfig `json:"-"`
}

// WorkerConfig shares the same validated config contract as webhook.
type WorkerConfig = Config

func LoadConfig(getenv func(string) string) (Config, error) {
	return loadRuntimeConfig(RoleWebhook, getenv)
}

func LoadWorkerConfig(getenv func(string) string) (WorkerConfig, error) {
	return loadRuntimeConfig(RoleWorker, getenv)
}

func loadRuntimeConfig(role RuntimeRole, getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("environment reader is required")
	}
	if role != RoleWebhook && role != RoleWorker {
		return Config{}, errors.New("unsupported runtime role")
	}
	environment := Environment(readSetting(getenv, KeyAppEnv))
	if environment == "" {
		environment = EnvLocal
	}
	switch environment {
	case EnvLocal, EnvStaging, EnvProduction:
	default:
		return Config{}, errors.New("APP_ENV must be local, staging or production")
	}
	// Cloud Run's metadata cannot be overridden by APP_ENV=local.
	if readSetting(getenv, KeyCloudRunService) != "" && environment == EnvLocal {
		return Config{}, errors.New("Cloud Run requires explicit APP_ENV=staging or production")
	}
	port, err := loadPort(getenv)
	if err != nil {
		return Config{}, err
	}
	timeout, err := loadLINETimeout(getenv)
	if err != nil {
		return Config{}, err
	}
	logging, err := loadLoggingConfig(getenv, environment)
	if err != nil {
		return Config{}, err
	}
	telemetry, err := loadTelemetryConfig(getenv)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Role: role, Environment: environment, Port: port,
		LINEAPITimeout: timeout, Log: logging, Telemetry: telemetry,
		Secrets: SecretConfig{
			MessagingAPIChannelSecret:      readSetting(getenv, KeyMessagingAPIChannelSecret),
			MessagingAPIChannelAccessToken: readSetting(getenv, KeyMessagingAPIChannelAccessToken),
		},
	}
	if role == RoleWebhook {
		cfg.MiniAppEntryURL, err = loadMiniAppEntryURL(getenv, environment)
		if err != nil {
			return Config{}, err
		}
	}
	if mode := readSetting(getenv, KeyEventMode); mode != "" && mode != "durable" {
		return Config{}, errors.New("LINE_EVENT_MODE only accepts durable; synchronous webhook was removed")
	}
	cfg.EventMode = "durable"
	if role == RoleWorker {
		if cfg.Secrets.MessagingAPIChannelSecret != "" {
			return Config{}, errors.New("worker must not hold LINE_MESSAGING_API_CHANNEL_SECRET")
		}
		if cfg.Secrets.MessagingAPIChannelAccessToken == "" {
			return Config{}, errors.New("worker requires LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN")
		}
		cfg.Durable, err = loadDurable(getenv, role)
		if err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
	if cfg.Secrets.MessagingAPIChannelAccessToken != "" {
		return Config{}, errors.New("public webhook must not hold LINE_MESSAGING_API_CHANNEL_ACCESS_TOKEN")
	}
	// A credential-free local process serves health endpoints, never messages.
	if environment == EnvLocal && cfg.Secrets.MessagingAPIChannelSecret == "" {
		return cfg, nil
	}
	if cfg.Secrets.MessagingAPIChannelSecret == "" {
		return Config{}, errors.New("webhook requires LINE_MESSAGING_API_CHANNEL_SECRET")
	}
	cfg.Durable, err = loadDurable(getenv, role)
	return cfg, err
}

func loadMiniAppEntryURL(getenv func(string) string, environment Environment) (string, error) {
	value := readSetting(getenv, KeyMiniAppLIFFURL)
	if value == "" {
		return "", nil
	}
	if len(value) > 1000 {
		return "", errors.New("LINE_MINI_APP_LIFF_URL must not exceed 1000 bytes")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || strings.ContainsAny(value, "?#") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("LINE_MINI_APP_LIFF_URL must be an absolute URL without userinfo, query or fragment")
	}
	if parsed.Scheme == "https" {
		return value, nil
	}
	localLoopback := environment == EnvLocal && parsed.Scheme == "http" &&
		(parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1")
	if !localLoopback {
		return "", errors.New("LINE_MINI_APP_LIFF_URL must use HTTPS except for local loopback HTTP")
	}
	return value, nil
}

func loadTelemetryConfig(getenv func(string) string) (TelemetryConfig, error) {
	mode := strings.ToLower(readSetting(getenv, KeyTelemetryMode))
	if mode == "" {
		mode = "off"
	}
	if mode != "off" && mode != "gcp-otlp" {
		return TelemetryConfig{}, errors.New("TELEMETRY_MODE must be off or gcp-otlp")
	}
	ratio := 0.05
	if value := readSetting(getenv, KeyTraceSampleRatio); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed < 0 || parsed > 1 {
			return TelemetryConfig{}, errors.New("OTEL_TRACE_SAMPLE_RATIO must be a number from 0 to 1")
		}
		ratio = parsed
	}
	interval := time.Minute
	if value := readSetting(getenv, KeyMetricInterval); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed < 10*time.Second || parsed > 5*time.Minute {
			return TelemetryConfig{}, errors.New("OTEL_METRIC_INTERVAL must be a duration from 10s to 5m")
		}
		interval = parsed
	}
	return TelemetryConfig{Mode: mode, TraceSampleRatio: ratio, MetricInterval: interval}, nil
}

func loadLoggingConfig(getenv func(string) string, environment Environment) (LogConfig, error) {
	level := strings.ToLower(readSetting(getenv, KeyLogLevel))
	if level == "" {
		level = "info"
	}
	switch level {
	case "debug", "info", "warn", "error":
	default:
		return LogConfig{}, errors.New("LOG_LEVEL must be debug, info, warn or error")
	}
	format := strings.ToLower(readSetting(getenv, KeyLogFormat))
	if format == "" {
		format = "json"
		if environment == EnvLocal {
			format = "text"
		}
	}
	if format != "text" && format != "json" {
		return LogConfig{}, errors.New("LOG_FORMAT must be text or json")
	}
	if environment != EnvLocal && format != "json" {
		return LogConfig{}, errors.New("staging/production LOG_FORMAT must be json")
	}
	return LogConfig{Level: level, Format: format}, nil
}

func (c Config) LiveLINE() bool {
	return c.Secrets.MessagingAPIChannelSecret != ""
}
