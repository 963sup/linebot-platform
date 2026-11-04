# Google Workspace OAuth 核心（離線實作，尚未開啟）

本次新增 `internal/infrastructure/google/api/oauth/flow.go` 作為 **Google Authorization Code + PKCE + OIDC** 的無狀態 SDK 輔助元件；Calendar 38、Tasks 14、Forms 10 共用一套 OAuth 設定。僅使用既有直接依賴 `golang.org/x/oauth2 v0.37.0` 與 `google.golang.org/api v0.310.0`，不自行實作 Token HTTP 交換。

## 已完成的技術元件

- 使用 `google.Endpoint`，固定 HTTPS Callback URL。
- `RandomState` 產生密碼學安全的 state/nonce；`AuthorizationURL` 使用 S256 PKCE、offline access、OIDC nonce 與有限 Scope allowlist。
- `ExchangeAndVerify` 使用 `Config.Exchange(..., oauth2.VerifierOption(...))`；取得 ID Token 後使用 `idtoken.Validate` 驗證 Google 簽章／Audience，額外檢查 Issuer、Subject、Expiry 與原始 nonce，僅在 Email 經驗證時提供 Email。
- 在離線測試使用 Fake Google Token Endpoint 與 Fake ID Token Verifier；不使用任何 Google 真實帳號、Key、Token 或服務。
- Token 與 provider 原始錯誤不記錄、不轉成 CLI 文字；受控流程應直接處理 Result，而不是輸出到 stdout。

## 尚未實作，不能直接對公提供登入

- 驗證 LINE 使用者及持有的帳號連結意圖，建立單次且短時有效的 State（綁定 ownerLineUserID、verifier、nonce、scope）。
- Firestore durable `Put/Consume`：只能原子消耗一次 state，跨 Cloud Run 執行個體仍防 CSRF、重播與搶先使用；state 應以安全雜湊作索引，不可放入日誌。
- 已驗證 Google 帳號（ID Token 的 `sub`）與 LINE 使用者的存取政策；沒有 Google ID Token 或 nonce 不一致必須拒絕。
- 加密 Refresh Token 的 Token Repository、金鑰輪替、憑證撤銷、Scope 增減與 Google 連線解除；注意更新 Access Token 時 Google 可能不回傳新的 Refresh Token，不能覆蓋現有有效值。
- TokenSource 跨程序刷新與持久化的版本檢查／分散式單次更新策略；Token refresh 結果未知時不得覆蓋較新 Token。
- OAuth Start／Callback HTTP 入口、Redirect URI 與 Cookie policy、登入期限及安全 header、敏感異常監控、隔離 Staging 真實授權測試。
- Credential Store 無可審查的加密／授權實作前，不可接入公開 LINE webhook、Cloud Run 或將本元件標為 production-ready。

## 最小流程

1. 外層驗證 LINE user → 建立 nonce/state/verifier 與按需 scopes。
2. 原子儲存一次性 state，向使用者提供 Google 同意 URL。
3. Callback 驗證 state 並原子消耗，拒絕未知、過期或重複狀態。
4. 使用本核心 Exchange 授權碼並驗證 ID Token 的 Google `sub`。
5. **先**完成憑證加密保存與同一使用者關聯交易，才標示已連結。
6. 每次建立 Google Calendar／Tasks／Forms Client 時，從受授權的 TokenSource 構造；以最低 Scope 運作。

警告：`oauth2/v2` 的 Tokeninfo／Userinfo 不是 OAuth Token 交換、Refresh Token 或身分登入的主要機制；不為這三個 API 額外建立大規模抽象。

來源：
- https://pkg.go.dev/golang.org/x/oauth2
- https://pkg.go.dev/golang.org/x/oauth2/google
- https://pkg.go.dev/google.golang.org/api/idtoken
- https://developers.google.com/identity/protocols/oauth2/web-server


## P2 implementation status (PR #27, offline only)

- Google OAuth flow / PKCE / nonce / ID Token signature+issuer+audience validation and requested-vs-granted scope verification: implemented and covered by offline unit tests.
- `internal/application/usecases/workspace/link.go`: orchestrates Begin / one-time Complete / Disconnect; requires a **trusted** already-authenticated LINE owner.
- `internal/adapter/outbound/google/cloud/firestore/workspace_oauth.go`: transactional single-use State, encrypted connection repository, versioned token updates and disconnected tombstones.
- `internal/infrastructure/google/api/oauth/keyring.go`: AES-256-GCM with document-associated data and key version support. `internal/infrastructure/google/cloud/secretmanager/oauth_keys.go` loads pinned key versions from an injected authorized Secret Manager client.
- `internal/adapter/outbound/google/api/oauth/provider.go`: Google OAuth provider adapter, fixed revocation endpoint and persist-before-use TokenSource. Across replicas, stale CAS updates are rejected and latest persisted credentials are reread; **this is not proof of zero token-refresh races during a real Google outage**.
- `internal/adapter/inbound/http/google/oauth/handler.go`: requires injected trusted LINE principal resolver, HTTPS Google authorization endpoint, secure host-only CSRF Cookie and one-time callback state; currently **not registered** on the public webhook.
- `internal/bootstrap/workspace_links.go`: explicit opt-in DI using injected Firestore client, keyring, Google OAuth client credentials and owned HTTP transport; no credentials automatically read from Git or an unsafe public endpoint.
- Local Go CI checks are separate from `tests/integration/google_oauth_firestore_test.go` which requires `FIRESTORE_EMULATOR_HOST` and additional runtime verification.

### Prerequisites remaining before public use

1. Verify the LINE principal with Messaging Account Linking or LINE Login before issuing authorization URLs; never accept a user ID copied from HTTP parameters.
2. Provision dedicated GCP staging project, restricted Firestore permissions, TTL policy and pinned Secret Manager encryption versions; validate revocation and key rotation.
3. Demonstrate Firestore Emulator transaction races, secret-manager IAM and real Google OAuth consent/refresh/revoke on a separate test Google account. Define a reauthorization fallback for concurrent or revoked refresh tokens.
4. Bind the HTTPS start/callback routes to a reviewed identity gate; enforce session and consent policy, use staging OAuth client IDs, and record redacted audit and recovery evidence.
5. Forms Watches require actual Pub/Sub topic permissions and renewal schedule. Calendar syncToken 410 invalidation and Tasks updatedMin full scan need independent durable checkpoints.
6. Do not mark P3/P4 or production readiness complete merely because offline CI succeeds.
