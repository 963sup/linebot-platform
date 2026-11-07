# LINE 功能覆蓋矩陣（部分實作）

| 功能組 | 預計範圍 | 狀態 |
|---|---|---|
| Webhook | 簽章驗證、文字 Message、Get/Set/Test Webhook Endpoint；Set/Test 僅限需明確確認的管理 CLI。同步模式沒有持久去重，durable 具 Firestore + Cloud Tasks；Edit／Unsend／媒體事件與 Staging 驗收仍未完成 | Partial (offline contract only) |
| Messaging | Reply、Push、Broadcast、Multicast、Narrowcast 均有 SDK Adapter；所有 campaign 發送均需明確 CLI --send + UUID v4 Retry Key；Narrowcast 以 JSON request file 保留 recipient、filter、limit 的完整官方結構 | Partial (offline contract only) |
| Message contents | Text／簡單 Flex Bubble；Webhook 文字回覆支援出勤同義詞、班表／狀態導引、功能選單、問候與可操作 fallback，sync／durable 共用同一確定性政策；Reply、Push、Multicast、Narrowcast、Broadcast 五種發送請求的官方 Validation API 已接入 CLI，聊天資料查詢與其他訊息類型尚未實作 | Partial (offline contract only) |
| Chat / User | Profile、Followers IDs、Group／Room 的 Summary／Count／IDs／Member Profile 已接入 CLI；媒體 content 已有 Blob Adapter/CLI；其他事件仍未實作 | Partial (offline contract only) |
| Rich Menu | List、Get、Default ID、Alias List/Get、User Menu ID、Batch Progress 已接入唯讀 CLI；建立、官方驗證、Alias 建立／更新、單一／批次綁定與解除、Batch 寫入及驗證已接入需雙重確認的 Ops CLI；圖片上傳／下載具受控 Operator CLI | Partial (offline contract only) |
| Audience | Audience Data、List、Shared Data、Shared List 已接入唯讀 CLI；建立、追加、上傳、修改、刪除已接入需要明確 --apply 與 --confirm 的 Operator CLI；真實帳號資格仍待驗證 | Partial (offline contract only) |
| Insights | Demographics、Deliveries、Followers、Message Event、Aggregation Unit、Rich Menu Summary/Daily 七項已接入唯讀 CLI | Partial (offline contract only) |
| Coupon / Membership | 會員清單、訂閱、會員 ID、Coupon 列表與詳情已接入唯讀 CLI；建立／關閉 Coupon 已接入需資格確認的 Operator CLI，實際使用仍需確認地區、資格與授權 | Partial / Restricted |
| Channel / account link | Bot Info、Webhook Endpoint GET、Quota、發送統計、Aggregation、Narrowcast Progress 已接入唯讀 CLI；Webhook 寫入及 Account Link Token 已接入受控 Operator CLI；Channel Access Token 8 項已接入獨立 operator-only Runtime，必須由 0600 JSON 檔提供憑證並將結果寫至 0600 輸出檔；真實 OAuth 身分與資格仍未驗證 | Partial (offline contract only) |

目前 repository inventory 追蹤 102/102 項 LINE 操作，並以 `integrated_offline` 表示程式碼接線與離線契約證據；這不是逐項 `operation_verified`，也不是 Staging、真實帳號資格或 Production 驗收。這項基礎設施範圍不等於已擴充 Webhook 全事件、完成所有 DDD 模型、啟用 AI 或部署 Google Cloud。

此表只是能力分組，不能代替端點級覆蓋追蹤。參閱 [Endpoint Inventory](endpoint-inventory.yaml)：固定官方 LINE OpenAPI revision，列出 Messaging API、Insights、Audience Group、Channel Access Token 共 102 個操作。狀態 `integrated_offline` 僅表示程式碼接線及離線契約測試，**不代表真實 LINE Channel／地區資格／Production 已驗證**。每次擴充都必須核對 pinned SDK 的真實 Go symbols、更新 inventory 並新增測試；未知帳號權限標記 `review`，受地區／功能資格限制的操作標記 `required_eligibility_review`。
