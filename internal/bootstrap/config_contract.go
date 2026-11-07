// Package bootstrap owns the single runtime configuration contract.
// Do not duplicate env-var names in other config parsers or docs.
package bootstrap

import (
	"fmt"
	"strings"
)

type SettingSpec struct {
	Key         string
	Roles       string
	Secret      bool
	Default     string
	Required    string
	Description string
}

const (
	KeyAppEnv              = "APP_ENV"
	KeyPort                = "PORT"
	KeyEventMode           = "LINE_EVENT_MODE"
	KeyLINETimeout         = "LINE_API_TIMEOUT"
	KeyLogLevel            = "LOG_LEVEL"
	KeyLogFormat           = "LOG_FORMAT"
	KeyTelemetryMode       = "TELEMETRY_MODE"
	KeyTraceSampleRatio    = "OTEL_TRACE_SAMPLE_RATIO"
	KeyMetricInterval      = "OTEL_METRIC_INTERVAL"
	KeyChannelSecret       = "LINE_CHANNEL_SECRET"
	KeyChannelToken        = "LINE_CHANNEL_ACCESS_TOKEN"
	KeyBotDestination      = "LINE_BOT_DESTINATION"
	KeyProjectID           = "GCP_PROJECT_ID"
	KeyQueueName           = "TASK_QUEUE_NAME"
	KeyWorkerURL           = "TASK_WORKER_URL"
	KeyOIDCAudience        = "TASK_OIDC_AUDIENCE"
	KeyInvokerEmail        = "TASK_INVOKER_EMAIL"
	KeyCloudRunService     = "K_SERVICE"
	KeyOpsStagingProject   = "LINEBOT_STAGING_PROJECT"
	KeyAttendanceEntryURL  = "ATTENDANCE_ENTRY_URL"
	KeyLIFFID              = "LINE_LIFF_ID"
	KeyLoginChannelID      = "LINE_LOGIN_CHANNEL_ID"
	KeyMiniAppPublicOrigin = "MINIAPP_PUBLIC_ORIGIN"
	KeyFirestoreDatabase   = "FIRESTORE_DATABASE"
	KeyAttendanceKeyID     = "ATTENDANCE_LOCATION_ACTIVE_KEY_ID"
	KeyAttendanceKeyMap    = "ATTENDANCE_LOCATION_KEY_VERSIONS"
)

// settingContract owns the role, default and secret metadata for each key.
// Key constants are shared with the parsers so a renamed setting cannot drift
// between runtime validation and the generated documentation.
var settingContract = []SettingSpec{
	{Key: KeyAppEnv, Roles: "webhook,worker,miniapp", Secret: false, Default: "local", Required: "Cloud Run 必須明確設定 staging/production", Description: "local、staging、production"},
	{Key: KeyPort, Roles: "webhook,worker,miniapp", Secret: false, Default: "8080", Required: "選用", Description: "HTTP 監聽埠，1–65535"},
	{Key: KeyEventMode, Roles: "webhook,worker", Secret: false, Default: "webhook=sync; worker=durable", Required: "正式 Webhook 必須 durable；Worker 僅允許 durable", Description: "本機 Webhook 可 sync；Worker 固定 durable"},
	{Key: KeyLINETimeout, Roles: "webhook,worker,miniapp", Secret: false, Default: "8s", Required: "選用", Description: "1s–12s，單次 LINE HTTP timeout"},
	{Key: KeyLogLevel, Roles: "webhook,worker,miniapp", Secret: false, Default: "info", Required: "選用", Description: "debug、info、warn、error"},
	{Key: KeyLogFormat, Roles: "webhook,worker,miniapp", Secret: false, Default: "local=text; others=json", Required: "選用", Description: "staging/production 強制 json"},
	{Key: KeyTelemetryMode, Roles: "webhook,worker,miniapp", Secret: false, Default: "off", Required: "選用", Description: "off 或 gcp-otlp；gcp-otlp 固定送往 127.0.0.1:4317 collector"},
	{Key: KeyTraceSampleRatio, Roles: "webhook,worker,miniapp", Secret: false, Default: "0.05", Required: "TELEMETRY_MODE=gcp-otlp 時套用", Description: "parent-based trace sample ratio，0–1"},
	{Key: KeyMetricInterval, Roles: "webhook,worker,miniapp", Secret: false, Default: "60s", Required: "TELEMETRY_MODE=gcp-otlp 時套用", Description: "metrics 匯出週期，10s–5m"},
	{Key: KeyChannelSecret, Roles: "webhook", Secret: true, Default: "—", Required: "durable 必需；sync 與 Token 成對", Description: "僅驗證原始 Webhook 簽章，不應傳給 Worker"},
	{Key: KeyChannelToken, Roles: "worker,local-sync,ops-push,ops-profile", Secret: true, Default: "—", Required: "worker 必需；本機 sync 成對", Description: "durable 公開 Webhook 禁止持有"},
	{Key: KeyBotDestination, Roles: "webhook-durable,worker", Secret: false, Default: "—", Required: "durable 必需", Description: "LINE Bot Info API 的 userId；不是 Console 的 Your user ID"},
	{Key: KeyProjectID, Roles: "webhook-durable,worker,miniapp", Secret: false, Default: "—", Required: "durable 與 miniapp 必需", Description: "最小 IAM 的指定 Google Cloud Project"},
	{Key: KeyQueueName, Roles: "webhook-durable,worker", Secret: false, Default: "—", Required: "durable 必需", Description: "六段 Queue 資源名稱；Project 必須等於 GCP_PROJECT_ID；Queue ID 僅限 1–100 字元英數或連字號"},
	{Key: KeyWorkerURL, Roles: "webhook-durable,worker", Secret: false, Default: "—", Required: "durable 必需", Description: "HTTPS Worker /tasks/v2/line-reply"},
	{Key: KeyOIDCAudience, Roles: "webhook-durable,worker", Secret: false, Default: "—", Required: "durable 必需", Description: "必須符合 Worker HTTPS Origin"},
	{Key: KeyInvokerEmail, Roles: "webhook-durable,worker", Secret: false, Default: "—", Required: "durable 必需", Description: "允許呼叫 Worker 的 Service Account Email"},
	{Key: KeyCloudRunService, Roles: "webhook,worker,miniapp", Secret: false, Default: "平台提供", Required: "Cloud Run 自動提供", Description: "防止在 Cloud Run 以 local profile 啟動"},
	{Key: KeyOpsStagingProject, Roles: "ops-events", Secret: false, Default: "—", Required: "events inspect/check 必需", Description: "明確指定只讀操作允許的 Staging Project"},
	{Key: KeyAttendanceEntryURL, Roles: "webhook", Secret: false, Default: "—", Required: "選用；設定後啟用聊天打卡入口", Description: "LIFF 或 MINI App 絕對 URL；正式環境必須 HTTPS"},
	{Key: KeyLIFFID, Roles: "miniapp", Secret: false, Default: "—", Required: "必需", Description: "公開 LIFF ID，不是 Messaging API secret"},
	{Key: KeyLoginChannelID, Roles: "miniapp", Secret: false, Default: "—", Required: "必需", Description: "LINE Login Channel ID"},
	{Key: KeyMiniAppPublicOrigin, Roles: "miniapp", Secret: false, Default: "—", Required: "必需；production 必須 HTTPS", Description: "無 path/query/fragment 的精確 public origin"},
	{Key: KeyFirestoreDatabase, Roles: "miniapp,ops-attendance", Secret: false, Default: "(default)", Required: "選用", Description: "Attendance 使用的明確 Firestore database ID"},
	{Key: KeyAttendanceKeyID, Roles: "miniapp,ops-attendance", Secret: false, Default: "—", Required: "必需", Description: "Attendance 位置 AES-GCM active key ID"},
	{Key: KeyAttendanceKeyMap, Roles: "miniapp,ops-attendance", Secret: false, Default: "—", Required: "必需", Description: "逗號分隔 key ID 與 pinned attendance-location-key Secret version resource；不含 key bytes"},
}

// SettingsContract returns a copy so callers cannot mutate global metadata.
func SettingsContract() []SettingSpec {
	return append([]SettingSpec(nil), settingContract...)
}

func readSetting(getenv func(string) string, key string) string {
	return strings.TrimSpace(getenv(key))
}

// ConfigurationMarkdown generates the canonical public, secret-free docs.
func ConfigurationMarkdown() string {
	var b strings.Builder
	b.WriteString("# 執行期設定契約（自動產生）\n\n本文件由 `internal/bootstrap/config_contract.go` 產生，**禁止直接編輯表格**。請修改 Go 設定契約後執行 `go run ./cmd/configcheck --write-doc`，再以 `make config-check` 驗證。這份清單涵蓋現有 webhook、worker、miniapp、ops-events 與 ops-attendance 使用的專案設定；Google SDK / ADC 自身的環境變數不在此表。\n\n| 環境變數 | 使用者 | Secret | 預設 | 條件 | 說明 |\n|---|---|---|---|---|---|\n")
	for _, spec := range settingContract {
		secret := "否"
		if spec.Secret {
			secret = "是"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n",
			spec.Key, spec.Roles, secret, spec.Default, spec.Required, spec.Description)
	}
	b.WriteString("\n## 使用規則\n\n- **local**：未指定 `APP_ENV` 時的安全本機預設，可使用 `LINE_EVENT_MODE=sync`，但本機 demo 不具持久去重。不提供 LINE 憑證時只開放 `/healthz`。Mini App 只允許 HTTP loopback origin。\n- **staging / production**：必須明確設定 `APP_ENV`，Webhook 必須 `LINE_EVENT_MODE=durable` 且不含 LINE 發送 Token；Worker 必須持有發送 Token但不得取得簽章用 Secret；Mini App 必須使用沒有 explicit port 的 HTTPS exact origin，且不得持有 Messaging Secret。即使誤設 `APP_ENV=local`，Cloud Run 亦會拒絕啟動。\n- **Credential 分離**：上述 Secret 僅由環境（正式環境建議 Secret Manager references）注入。普通設定可以透過部署的非敏感環境變數傳入。應用不讀取 YAML、JSON 設定檔或自動 `source` `.env`。\n- **可觀測性**：local 預設文字日志、staging/production 預設 JSON；Telemetry 預設 `off`；`gcp-otlp` 只連接固定 loopback collector，不接受任意 endpoint 或 exporter credential。日誌、metric 與 trace 永不包含 Token、Channel Secret、LINE user ID、EventKey 或原始 webhook body。\n- **本機啟動**：`make dev` 啟動不需憑證的 Health MVP；需要測試 LINE 時在**目前 shell**安全提供兩項測試憑證。Worker 與 Mini App 為獨立入口。\n- **無配置副作用**：`go run ./cmd/configcheck --role webhook --validate`、`--role worker --validate` 或 `--role miniapp --validate` 只解析並檢查設定，不建立 SDK Client、不連接 Google Cloud、OTLP collector 或 LINE，也不輸出 Secret 值。\n")
	return b.String()
}
