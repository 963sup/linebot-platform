package bootstrap

import (
	"errors"
	"strconv"
	"strings"
)

type Config struct {
	Port          string
	ChannelSecret string
	ChannelToken  string
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
	cfg := Config{
		Port:          port,
		ChannelSecret: strings.TrimSpace(getenv("LINE_CHANNEL_SECRET")),
		ChannelToken:  strings.TrimSpace(getenv("LINE_CHANNEL_ACCESS_TOKEN")),
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
