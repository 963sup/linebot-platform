package bootstrap

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Port           string
	ChannelSecret  string
	ChannelToken   string
	LINEAPITimeout time.Duration
	EventMode      string
	Durable        DurableConfig
}

func LoadConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("environment reader is required")
	}
	port, err := loadPort(getenv)
	if err != nil {
		return Config{}, err
	}
	timeout, err := loadLINETimeout(getenv)
	if err != nil {
		return Config{}, err
	}
	mode := strings.TrimSpace(getenv("LINE_EVENT_MODE"))
	if mode == "" {
		mode = "sync"
	}
	if mode != "sync" && mode != "durable" {
		return Config{}, errors.New("LINE_EVENT_MODE must be sync or durable")
	}
	cfg := Config{
		Port:           port,
		ChannelSecret:  strings.TrimSpace(getenv("LINE_CHANNEL_SECRET")),
		ChannelToken:   strings.TrimSpace(getenv("LINE_CHANNEL_ACCESS_TOKEN")),
		LINEAPITimeout: timeout,
		EventMode:      mode,
	}
	if (cfg.ChannelSecret == "") != (cfg.ChannelToken == "") {
		return Config{}, errors.New("LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN must both be set or both omitted")
	}
	if mode == "durable" {
		if !cfg.LiveLINE() {
			return Config{}, errors.New("durable mode requires both LINE credentials")
		}
		cfg.Durable, err = loadDurable(getenv)
		if err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func (c Config) LiveLINE() bool {
	return c.ChannelSecret != "" && c.ChannelToken != ""
}
