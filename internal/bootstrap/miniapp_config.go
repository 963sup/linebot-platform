package bootstrap

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	lineLoginChannelIDPattern = regexp.MustCompile(`^[0-9]{6,20}$`)
	liffIDPattern             = regexp.MustCompile(`^[0-9]+-[A-Za-z0-9_-]+$`)
	projectIDPattern          = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
	keyIDPattern              = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	pinnedAttendanceKey       = regexp.MustCompile(`^projects/([^/]+)/secrets/attendance-location-key/versions/([1-9][0-9]*)$`)
)

type MiniAppConfig struct {
	Environment       Environment
	Role              RuntimeRole
	Port              string
	LINEAPITimeout    time.Duration
	Log               LogConfig
	Telemetry         TelemetryConfig
	LIFFID            string
	LoginChannelID    string
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
	liffID := readSetting(getenv, KeyLIFFID)
	channelID := readSetting(getenv, KeyLoginChannelID)
	origin := readSetting(getenv, KeyMiniAppPublicOrigin)
	database := readSetting(getenv, KeyFirestoreDatabase)
	if database == "" {
		database = "(default)"
	}
	if database != "(default)" && !keyIDPattern.MatchString(database) {
		return MiniAppConfig{}, errors.New("FIRESTORE_DATABASE has an invalid database ID")
	}
	if !liffIDPattern.MatchString(liffID) {
		return MiniAppConfig{}, errors.New("LINE_LIFF_ID must contain a numeric app ID, hyphen and endpoint ID")
	}
	if !lineLoginChannelIDPattern.MatchString(channelID) {
		return MiniAppConfig{}, errors.New("LINE_LOGIN_CHANNEL_ID must contain decimal digits")
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.String() != origin {
		return MiniAppConfig{}, errors.New("MINIAPP_PUBLIC_ORIGIN must be an exact origin without path, query, user info or fragment")
	}
	if environment == EnvLocal {
		host := parsed.Hostname()
		if parsed.Scheme != "http" || (host != "localhost" && host != "127.0.0.1") {
			return MiniAppConfig{}, errors.New("local MINIAPP_PUBLIC_ORIGIN must use http loopback localhost or 127.0.0.1")
		}
	} else if parsed.Scheme != "https" || parsed.Port() != "" || parsed.Host != parsed.Hostname() || parsed.Host != strings.ToLower(parsed.Host) {
		return MiniAppConfig{}, errors.New("staging/production MINIAPP_PUBLIC_ORIGIN must use https without an explicit port")
	}
	return MiniAppConfig{
		Environment: environment, Role: RoleMiniApp, Port: port,
		LINEAPITimeout: timeout, Log: logging, Telemetry: telemetry,
		LIFFID: liffID, LoginChannelID: channelID, PublicOrigin: origin,
		FirestoreDatabase: database,
	}, nil
}

func parseAttendanceKeyVersions(raw, projectID string) (map[string]string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) {
		return nil, errors.New("ATTENDANCE_LOCATION_KEY_VERSIONS must be an exact pinned key mapping")
	}
	versions := make(map[string]string)
	detectedProjectID := projectID
	for _, entry := range strings.Split(raw, ",") {
		if entry == "" || entry != strings.TrimSpace(entry) || strings.Count(entry, "=") != 1 {
			return nil, errors.New("ATTENDANCE_LOCATION_KEY_VERSIONS contains an invalid entry")
		}
		parts := strings.SplitN(entry, "=", 2)
		id, resource := parts[0], parts[1]
		if id == "" || resource == "" || id != strings.TrimSpace(id) || resource != strings.TrimSpace(resource) || !keyIDPattern.MatchString(id) {
			return nil, errors.New("ATTENDANCE_LOCATION_KEY_VERSIONS contains an invalid entry")
		}
		match := pinnedAttendanceKey.FindStringSubmatch(resource)
		if len(match) != 3 {
			return nil, errors.New("Attendance location keys must use pinned versions")
		}
		if detectedProjectID == "" {
			detectedProjectID = match[1]
		} else if match[1] != detectedProjectID {
			return nil, errors.New("Attendance location keys must belong to one project")
		}
		if _, duplicate := versions[id]; duplicate {
			return nil, errors.New("ATTENDANCE_LOCATION_KEY_VERSIONS contains a duplicate key ID")
		}
		versions[id] = resource
	}
	if len(versions) == 0 {
		return nil, errors.New("ATTENDANCE_LOCATION_KEY_VERSIONS is required")
	}
	return versions, nil
}
