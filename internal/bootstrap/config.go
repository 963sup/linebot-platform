package bootstrap

import (
	"errors"
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
)

// SecretConfig is deliberately separate from ordinary runtime settings.
// Both fmt printing and %#v redact its values; never log or marshal this type.
type SecretConfig struct {
	ChannelSecret string
	ChannelToken  string
}

func (SecretConfig) String() string   { return "[redacted secrets]" }
func (SecretConfig) GoString() string { return "bootstrap.SecretConfig{redacted}" }

// LogConfig contains safe, non-sensitive logging settings.
type LogConfig struct {
	Level  string
	Format string
}

// Config holds ordinary settings and explicitly isolated credentials.
// Existing callers must use the typed loader for role-specific validation.
type Config struct {
	Environment    Environment
	Role           RuntimeRole
	Port           string
	EventMode      string
	LINEAPITimeout time.Duration
	Durable        DurableConfig
	Log            LogConfig
	Secrets        SecretConfig `json:"-"`
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
	cfg := Config{
		Role: role, Environment: environment, Port: port,
		LINEAPITimeout: timeout, Log: logging,
		Secrets: SecretConfig{
			ChannelSecret: readSetting(getenv, KeyChannelSecret),
			ChannelToken:  readSetting(getenv, KeyChannelToken),
		},
	}
	if role == RoleWorker {
		cfg.EventMode = "durable"
		if cfg.Secrets.ChannelSecret != "" {
			return Config{}, errors.New("worker must not hold LINE_CHANNEL_SECRET")
		}
		if cfg.Secrets.ChannelToken == "" {
			return Config{}, errors.New("worker requires LINE_CHANNEL_ACCESS_TOKEN")
		}
		cfg.Durable, err = loadDurable(getenv)
		if err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
	cfg.EventMode = readSetting(getenv, KeyEventMode)
	if cfg.EventMode == "" {
		cfg.EventMode = "sync"
	}
	if cfg.EventMode != "sync" && cfg.EventMode != "durable" {
		return Config{}, errors.New("LINE_EVENT_MODE must be sync or durable")
	}
	if cfg.EventMode == "durable" {
		if cfg.Secrets.ChannelSecret == "" {
			return Config{}, errors.New("durable webhook requires LINE_CHANNEL_SECRET")
		}
		if cfg.Secrets.ChannelToken != "" {
			return Config{}, errors.New("public durable webhook must not hold LINE_CHANNEL_ACCESS_TOKEN")
		}
		cfg.Durable, err = loadDurable(getenv)
		if err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
	if environment != EnvLocal {
		return Config{}, errors.New("staging/production webhook must use durable mode")
	}
	if (cfg.Secrets.ChannelSecret == "") != (cfg.Secrets.ChannelToken == "") {
		return Config{}, errors.New("sync mode requires both LINE credentials or neither")
	}
	return cfg, nil
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
	return c.Secrets.ChannelSecret != "" &&
		(c.EventMode == "durable" || c.Secrets.ChannelToken != "")
}
