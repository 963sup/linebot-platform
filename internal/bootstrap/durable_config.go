package bootstrap

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
)

// DurableConfig is only used when LINE_EVENT_MODE=durable. Configuration is
// explicit and independently validated before any GCP clients are created.
type DurableConfig struct {
	ProjectID        string
	BotDestination   string
	QueueName        string
	WorkerURL        string
	OIDCAudience     string
	InvokerEmail     string
}

type WorkerConfig struct {
	Port             string
	ChannelToken     string
	LINEAPITimeout   time.Duration
	Durable          DurableConfig
}

func loadPort(getenv func(string) string) (string, error) {
	port := strings.TrimSpace(getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("PORT must be a valid TCP port (1–65535)")
	}
	return port, nil
}

func loadLINETimeout(getenv func(string) string) (time.Duration, error) {
	timeout := httpclient.DefaultTimeout
	if value := strings.TrimSpace(getenv("LINE_API_TIMEOUT")); value != "" {
		var err error
		timeout, err = time.ParseDuration(value)
		if err != nil || timeout < time.Second || timeout > 12*time.Second {
			return 0, errors.New("LINE_API_TIMEOUT must be a duration from 1s to 12s")
		}
	}
	return timeout, nil
}

func loadDurable(getenv func(string) string) (DurableConfig, error) {
	d := DurableConfig{
		ProjectID: strings.TrimSpace(getenv("GCP_PROJECT_ID")),
		BotDestination: strings.TrimSpace(getenv("LINE_BOT_DESTINATION")),
		QueueName: strings.TrimSpace(getenv("TASK_QUEUE_NAME")),
		WorkerURL: strings.TrimSpace(getenv("TASK_WORKER_URL")),
		OIDCAudience: strings.TrimSpace(getenv("TASK_OIDC_AUDIENCE")),
		InvokerEmail: strings.TrimSpace(getenv("TASK_INVOKER_EMAIL")),
	}
	if d.ProjectID == "" || d.BotDestination == "" || d.QueueName == "" || d.WorkerURL == "" ||
		d.OIDCAudience == "" || d.InvokerEmail == "" {
		return DurableConfig{}, errors.New("durable event mode requires GCP project, LINE bot destination, queue, worker URL and OIDC identity")
	}
	prefix := "projects/" + d.ProjectID + "/locations/"
	if !strings.HasPrefix(d.QueueName, prefix) || !strings.Contains(d.QueueName[len(prefix):], "/queues/") {
		return DurableConfig{}, errors.New("TASK_QUEUE_NAME must belong to configured GCP_PROJECT_ID")
	}
	u, err := url.Parse(d.WorkerURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "/tasks/line-reply" || u.RawQuery != "" || u.Fragment != "" {
		return DurableConfig{}, errors.New("TASK_WORKER_URL must be an HTTPS /tasks/line-reply endpoint")
	}
	audience, err := url.Parse(d.OIDCAudience)
	if err != nil || audience.Scheme != "https" || audience.Host != u.Host ||
		audience.User != nil || audience.RawQuery != "" || audience.Fragment != "" ||
		audience.Path != "" && audience.Path != "/" {
		return DurableConfig{}, errors.New("TASK_OIDC_AUDIENCE must match the HTTPS worker service origin")
	}
	return d, nil
}

func LoadWorkerConfig(getenv func(string) string) (WorkerConfig, error) {
	if getenv == nil {
		return WorkerConfig{}, errors.New("environment reader is required")
	}
	port, err := loadPort(getenv)
	if err != nil {
		return WorkerConfig{}, err
	}
	timeout, err := loadLINETimeout(getenv)
	if err != nil {
		return WorkerConfig{}, err
	}
	d, err := loadDurable(getenv)
	if err != nil {
		return WorkerConfig{}, err
	}
	token := strings.TrimSpace(getenv("LINE_CHANNEL_ACCESS_TOKEN"))
	if token == "" {
		return WorkerConfig{}, errors.New("worker requires LINE_CHANNEL_ACCESS_TOKEN")
	}
	return WorkerConfig{Port: port, ChannelToken: token, LINEAPITimeout: timeout, Durable: d}, nil
}
