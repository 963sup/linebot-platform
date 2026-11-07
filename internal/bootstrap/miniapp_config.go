package bootstrap

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	miniAppChannelIDPattern = regexp.MustCompile(`^[0-9]{6,20}$`)
	liffIDPattern           = regexp.MustCompile(`^[0-9]+-[A-Za-z0-9_-]+$`)
	projectIDPattern        = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
	keyIDPattern            = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

type MiniAppConfig struct {
	Environment       Environment
	Role              RuntimeRole
	Port              string
	LINEAPITimeout    time.Duration
	Log               LogConfig
	Telemetry         TelemetryConfig
	LIFFID            string
	ChannelID         string
	LIFFURL           string
	EndpointURL       string
	PublicOrigin      string
	FirestoreDatabase string
}

func LoadMiniAppConfig(getenv func(string) string) (MiniAppConfig, error) {
	if getenv == nil {
		return MiniAppConfig{}, errors.New("environment reader is required")
	}
	environment := Environment(readSetting(getenv, KeyAppEnv))
	if environment == "" {
		environment = EnvLocal
	}
	switch environment {
	case EnvLocal, EnvStaging, EnvProduction:
	default:
		return MiniAppConfig{}, errors.New("APP_ENV must be local, staging or production")
	}
	if readSetting(getenv, KeyCloudRunService) != "" && environment == EnvLocal {
		return MiniAppConfig{}, errors.New("Cloud Run requires explicit APP_ENV=staging or production")
	}
	for _, key := range []string{KeyChannelSecret, KeyChannelToken, KeyBotDestination, KeyQueueName, KeyWorkerURL, KeyOIDCAudience, KeyInvokerEmail} {
		if readSetting(getenv, key) != "" {
			return MiniAppConfig{}, errors.New("miniapp contains a forbidden delivery setting")
		}
	}
	port, err := loadPort(getenv)
	if err != nil {
		return MiniAppConfig{}, err
	}
	timeout, err := loadLINETimeout(getenv)
	if err != nil {
		return MiniAppConfig{}, err
	}
	logging, err := loadLoggingConfig(getenv, environment)
	if err != nil {
		return MiniAppConfig{}, err
	}
	telemetry, err := loadTelemetryConfig(getenv)
	if err != nil {
		return MiniAppConfig{}, err
	}
	channelID := readSetting(getenv, KeyMiniAppChannelID)
	liffURL := readSetting(getenv, KeyMiniAppLIFFURL)
	endpointURL := readSetting(getenv, KeyMiniAppEndpointURL)
	database := readSetting(getenv, KeyFirestoreDatabase)
	if database == "" {
		database = "(default)"
	}
	if database != "(default)" && !keyIDPattern.MatchString(database) {
		return MiniAppConfig{}, errors.New("FIRESTORE_DATABASE has an invalid database ID")
	}
	if !miniAppChannelIDPattern.MatchString(channelID) {
		return MiniAppConfig{}, errors.New("LINE_MINI_APP_CHANNEL_ID must contain decimal digits")
	}
	liffID, err := miniAppLIFFID(liffURL, channelID)
	if err != nil {
		return MiniAppConfig{}, err
	}
	parsed, err := url.Parse(endpointURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "/miniapp/" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.String() != endpointURL {
		return MiniAppConfig{}, errors.New("LINE_MINI_APP_ENDPOINT_URL must be an exact URL ending in /miniapp/")
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if environment == EnvLocal {
		host := parsed.Hostname()
		if parsed.Scheme != "http" || (host != "localhost" && host != "127.0.0.1") {
			return MiniAppConfig{}, errors.New("local LINE_MINI_APP_ENDPOINT_URL must use HTTP loopback")
		}
	} else if parsed.Scheme != "https" || parsed.Port() != "" || parsed.Host != parsed.Hostname() || parsed.Host != strings.ToLower(parsed.Host) {
		return MiniAppConfig{}, errors.New("staging/production LINE_MINI_APP_ENDPOINT_URL must use HTTPS without an explicit port")
	}
	return MiniAppConfig{
		Environment: environment, Role: RoleMiniApp, Port: port,
		LINEAPITimeout: timeout, Log: logging, Telemetry: telemetry,
		LIFFID: liffID, ChannelID: channelID, LIFFURL: liffURL, EndpointURL: endpointURL, PublicOrigin: origin,
		FirestoreDatabase: database,
	}, nil
}

func miniAppLIFFID(rawURL, channelID string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "miniapp.line.me" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("LINE_MINI_APP_LIFF_URL must use https://miniapp.line.me/{liffId}")
	}
	liffID := strings.TrimPrefix(parsed.EscapedPath(), "/")
	if strings.Contains(liffID, "/") || !liffIDPattern.MatchString(liffID) || !strings.HasPrefix(liffID, channelID+"-") {
		return "", errors.New("LINE_MINI_APP_LIFF_URL must contain a LIFF ID owned by LINE_MINI_APP_CHANNEL_ID")
	}
	return liffID, nil
}
