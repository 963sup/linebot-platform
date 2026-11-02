package bootstrap

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
)

type Config struct {
	Port           string
	ChannelSecret  string
	ChannelToken   string
	LINEAPITimeout time.Duration
}

func LoadConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("environment reader is required")
	}
	port := strings.TrimSpace(getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, errors.New("PORT must be a valid TCP port (1–65535)")
	}

	timeout := httpclient.DefaultTimeout
	if value := strings.TrimSpace(getenv("LINE_API_TIMEOUT")); value != "" {
		timeout, err = time.ParseDuration(value)
		if err != nil || timeout < time.Second || timeout > 12*time.Second {
			return Config{}, errors.New("LINE_API_TIMEOUT must be a duration from 1s to 12s")
		}
	}
	cfg := Config{
		Port:           port,
		ChannelSecret:  strings.TrimSpace(getenv("LINE_CHANNEL_SECRET")),
		ChannelToken:   strings.TrimSpace(getenv("LINE_CHANNEL_ACCESS_TOKEN")),
		LINEAPITimeout: timeout,
	}
	// Partial credentials indicate a deployment configuration mistake.
	if (cfg.ChannelSecret == "") != (cfg.ChannelToken == "") {
		return Config{}, errors.New("LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN must both be set or both omitted")
	}
	return cfg, nil
}

func (c Config) LiveLINE() bool {
	return c.ChannelSecret != "" && c.ChannelToken != ""
}
