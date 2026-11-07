package bootstrap

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"time"

	queueinfra "github.com/963sup/linebot-platform/internal/infrastructure/google/cloud/cloudtasks"
	"github.com/963sup/linebot-platform/internal/infrastructure/httpclient"
)

var messagingAPIBotUserIDPattern = regexp.MustCompile(`^U[0-9a-f]{32}$`)

// DurableConfig is only used when LINE_EVENT_MODE=durable. Configuration is
// explicit and independently validated before any GCP clients are created.
type DurableConfig struct {
	ProjectID    string
	BotUserID    string
	QueueName    string
	WorkerURL    string
	OIDCAudience string
	InvokerEmail string
}

func loadPort(getenv func(string) string) (string, error) {
	port := readSetting(getenv, KeyPort)
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
	if value := readSetting(getenv, KeyLINETimeout); value != "" {
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
		ProjectID:    readSetting(getenv, KeyProjectID),
		BotUserID:    readSetting(getenv, KeyMessagingAPIBotUserID),
		QueueName:    readSetting(getenv, KeyQueueName),
		WorkerURL:    readSetting(getenv, KeyWorkerURL),
		OIDCAudience: readSetting(getenv, KeyOIDCAudience),
		InvokerEmail: readSetting(getenv, KeyInvokerEmail),
	}
	if d.ProjectID == "" || d.BotUserID == "" || d.QueueName == "" || d.WorkerURL == "" ||
		d.OIDCAudience == "" || d.InvokerEmail == "" {
		return DurableConfig{}, errors.New("durable event mode requires GCP project, LINE bot destination, queue, worker URL and OIDC identity")
	}
	if !messagingAPIBotUserIDPattern.MatchString(d.BotUserID) {
		return DurableConfig{}, errors.New("LINE_MESSAGING_API_BOT_USER_ID must be U followed by 32 lowercase hexadecimal characters")
	}
	// The preflight must reject the same malformed names as the Cloud Tasks
	// adapter, before creating Firestore or Cloud Tasks SDK clients.
	queueProject, ok := queueinfra.QueueProjectID(d.QueueName)
	if !ok || queueProject != d.ProjectID {
		return DurableConfig{}, errors.New("TASK_QUEUE_NAME must be a valid six-segment queue in GCP_PROJECT_ID")
	}
	u, err := url.Parse(d.WorkerURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "/tasks/v2/line-reply" || u.RawQuery != "" || u.Fragment != "" {
		return DurableConfig{}, errors.New("TASK_WORKER_URL must be an HTTPS /tasks/v2/line-reply endpoint")
	}
	audience, err := url.Parse(d.OIDCAudience)
	if err != nil || audience.Scheme != "https" || audience.Host != u.Host ||
		audience.User != nil || audience.RawQuery != "" || audience.Fragment != "" ||
		audience.Path != "" && audience.Path != "/" {
		return DurableConfig{}, errors.New("TASK_OIDC_AUDIENCE must match the HTTPS worker service origin")
	}
	return d, nil
}
