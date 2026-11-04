# Google Calendar／Tasks／Forms：LINE 使用者自行綁定 Google 帳號

## 範圍與狀態

- **已完成離線整合**：Calendar v3、Tasks v1、Forms v1 及 Google OAuth 核心、Firestore 加密憑證與一次性 State 的程式碼及離線測試已進入 `main`。
- **尚未完成／驗證**：一般 LINE 使用者的網頁自助綁定入口與 LINE Login／LIFF 信任鏈、實際 Google OAuth 同意及 Secret Manager／Firestore IAM 驗收、各帳號獨立同步。OAuth Handler 尚未掛載公開路由。
- 此文由 [PR #26](https://github.com/963sup/linebot-platform/pull/26) 的原始自助連結規格整合至 [PR #27](https://github.com/963sup/linebot-platform/pull/27)，作為 P2～P4 的安全／驗收契約，不代表已部署、取得 Google OAuth 審核或擁有真實使用者資料。
- 使用者自己操作並同意授權；平台只代其執行已授權的操作，不取得 Google 密碼，也不提供管理員共用 Google 帳號。

## P2：自助 Google 綁定／解除綁定（核心已離線實作，外部身分尚未接線）

### 身分與流程

1. 使用者在 LINE Bot 私人對話中點選「連結 Google 帳號」。Webhook 先驗證 LINE 原始簽章，取得可信的 Channel／LINE 使用者身分；群組訊息不得直接代表某位成員授權。
2. 產生一次性、短期有效的綁定 Session（隨機 `state`、PKCE verifier／S256 challenge、LINE 使用者關聯、Scope、建立及過期時間）。Firestore 只保留必要的安全資料及 TTL；state 必須原子單次消耗。
3. 打開獨立的 HTTPS 綁定頁面。**連結可能被轉傳**，所以在完成 Google 綁定前，需由伺服器再驗證目前瀏覽器使用者的 LINE Login／LIFF 身分，並確認能與原始 Bot 使用者識別對應。不可相信自行輸入的 `lineUserId`、任意 query parameter 或只有連結持有權；**Start／Callback 的 state cookie 本身不等於持有者已通過 LINE Login 身分校驗**。
4. 綁定者在瀏覽器選擇自己的 Google 帳號並明確同意 Calendar、Tasks 或 Forms 所需最小 Scope；Authorization Code 回到伺服器端固定 Redirect URI；檢查 state、PKCE、一次性 Session、拒絕及過期狀態。
5. 伺服器交換授權碼、驗證 Google ID Token（issuer、audience、expiry、nonce）或經驗證身分流程；以 Google `sub` 當穩定識別，不把可能變動的 Email 當主鍵。核對實際核發的 Scopes。
6. 依 `(LINE Channel／LINE User ID, Google sub)` 保存帳號連結；必須在可信的 `owner` 識別中包含 Channel／租戶邊界（PR #27 現有 Repository 只對呼叫端提供的 owner + sub 做隔離，不能因此假設已完成 Channel 隔離）。先支援每人一個預設 Google 帳號，可保留擴充多帳號選擇的結構。禁止跨使用者查詢或交換 Token。
7. 綁定成功後只在 LINE 傳送安全完成訊息；**不能**把 Authorization Code、Access Token、Refresh Token、ID Token 或完整 Calendar／Tasks／Forms 資料放進 LINE 訊息、URL、日誌或 CI。
8. 使用者可以在自己的管理介面查看連結狀態、重新授權、撤銷／解除綁定。解除綁定應停止其 Sync／Watch、撤銷或停用憑證並依保留政策刪除資料。

### Token Repository 與授權

- 採既有 `golang.org/x/oauth2` 和 `google.golang.org/api`；若確需其他 SDK／KMS，先查證並審核，不預先新增框架。
- 授權需要背景同步時才要求 `access_type=offline`。Refresh Token 可能缺席、被撤銷、過期或被更換；不可假設每次 OAuth callback 都發新 Refresh Token。
- 在 Firestore 儲存經認證加密的 Token、最小必要 Metadata、授權 Scopes、Google sub、有效期限及金鑰版本；加密金鑰由受控 KMS／等效專用金鑰管理機制保護，禁止明文 Token 欄位及機密日誌。
- Token 更新和憑證寫入需具備併發控制，避免兩次刷新交錯覆蓋；撤銷／`invalid_grant` 要標為需使用者重綁，不得無限重試。
- Calendar／Tasks／Forms 可以分別要求最小 Scope，使用者沒有同意的服務必須拒絕呼叫。授權 Scope 擴大時執行 Incremental Authorization，不默默提高權限。
- Operator CLI 的 `--token-file` 僅供隔離本機測試，不是提供一般使用者的產品綁定方式。

## P3：真正的自助綁定與 Staging 驗收

P3 **不是讓管理員代替使用者連結帳號**，而是以經同意的隔離測試使用者實際驗證 P2 的身分、帳號隔離與 Calendar／Tasks／Forms 基本 API。P4 的同步行為另依同步驗收項目確認：

- 不同 LINE 使用者各自選取不同 Google 帳號，驗證不存在跨帳號 Calendar／Tasks／Forms 資料讀取、寫入或快取，並確認未授權 Scope 被拒絕。
- 驗證使用者拒絕同意、少授 Scope、同意過期、Refresh Token 失效、解除綁定／重綁、LINE 登入驗證錯誤、連結轉傳、state 重放及多人同時連結。
- 驗證 Google OAuth Consent Screen、Redirect URI、應用發布狀態、測試使用者資格與實際需要的 Google OAuth 驗證要求。沒有審核通過的 Scope 或授權不得聲稱一般使用者都可使用。
- 真實 Google 資料與 LINE 正式帳號都不是 CI 的測試物件；CI 只使用模擬帳號與 Stub，Staging 需獨立核准並保存不含憑證的驗收證據。

## P4：每位使用者獨立同步

- 每筆工作必須由已驗證的 LINE 使用者／Google sub／資源組合決定；Tasks／Calendar／Forms Adapter 透過該使用者的 TokenSource 呼叫，不能共用一個全域 Google Service。
- Calendar Events、CalendarList 等可用 `watch` 與 `syncToken`：每個使用者、每個 Calendar 資源各自保存 Sync Cursor 以及 Channel ID、Resource ID、Channel Token 驗證資訊、到期時間和更新狀態。
- Watch 通知不攜帶實際事件資料；Webhook 收到後驗證 Channel／Token／Resource，標記需要同步，再以該使用者 OAuth 取得更動。Channel 過期需重新建立；同步 Token 410 時重建完整快照。
- Google Tasks v1 沒有 `watch`／`syncToken`，使用 `updatedMin`、`pageToken`、`showDeleted`、`showHidden` 等查詢，加上週期性完整核對；任務時間僅能按 API 提供的日期資訊處理。
- 所有同步工作保留重入／去重、速率限制、錯誤隔離與使用者取消綁定即停止；避免從同步通知直接引發未經使用者授權的群組／其他帳號操作。
- Forms Watch 通知經 Google Cloud Pub/Sub 收取，使用每個表單各自的資源身分、Watch ID、到期日、通知去重及更新游標；每次通知後重新查詢表單／回應內容，定期 Renew 並能在通知遺失後補同步。
- 以隔離測試帳號驗證 Calendar Channel Renew／410 全量同步、Tasks 週期核對及 Forms Pub/Sub 通知重送、被刪除事項、Google HTTP 429／5xx、部署重啟與取消綁定後停止工作。
- 優先複用既有 Firestore／Cloud Tasks；不新建與現有可靠性架構競爭的 Queue。

## PR #27 實作對照與未完成事項

**已具備程式碼（不代表已在正式 LINE／Firestore 環境驗收）：** `internal/application/usecases/workspace/link.go`、`internal/adapter/inbound/http/google/oauth/handler.go`、`internal/adapter/outbound/google/cloud/firestore/workspace_oauth.go`、`internal/adapter/outbound/google/api/oauth/provider.go`、`internal/infrastructure/google/api/oauth/flow.go`、`internal/infrastructure/google/api/oauth/keyring.go` 及 `internal/bootstrap/workspace_links.go`。

**仍需實作／驗證：** 將現有可信 LINE Login／LIFF／Messaging Account Linking 識別映射至同一個使用者及 Channel；連結被轉傳時在瀏覽器重新確認 LINE 身分；安全掛載 OAuth Start／Callback；正式 Google 同意／撤銷、Firestore Emulator 多執行個體交易、Secret Manager 金鑰權限、使用者取消連結後同步任務終止，以及 Calendar／Tasks／Forms 的持久同步。

新增設定須遵守 `internal/bootstrap/config_contract.go` 單一設定契約；保留既有 SDK 與 Go Idioms，不為未啟用功能建立空實作。

補充驗收：[OAuth Foundation](oauth-foundation.md)、[Workspace Staging Acceptance](staging-acceptance.md)；本規格保留自助綁定的產品／安全不變量。

官方依據：
- https://developers.google.com/identity/protocols/oauth2/web-server
- https://developers.google.com/identity/openid-connect/openid-connect
- https://developers.google.com/workspace/calendar/api/guides/push
- https://developers.google.com/workspace/tasks/reference/rest/v1/tasks/list
- https://developers.google.com/workspace/forms/api/guides/push-notifications
