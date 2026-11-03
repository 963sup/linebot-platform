# LINE 功能覆蓋矩陣（部分實作）

| 功能組 | 預計範圍 | 狀態 |
|---|---|---|
| Webhook | 簽章驗證、文字 Message；同步模式沒有持久去重，選用 durable 模式有 Firestore 事件去重與 OIDC Worker。Edit／unsend／媒體與實際 Staging 仍待完成 | Partial (development only) |
| Messaging | Reply、Push 已有基本 SDK adapter；Broadcast、Multicast、Narrowcast 未實作 | Partial |
| Message contents | Text 與簡單 Flex bubble 已實作；其餘類型尚未實作 | Partial |
| Chat / User | Profile 已接入 SDK Adapter、Contacts Use Case 與明確啟動的 Ops CLI；followers、group／room、content retrieval 尚未實作 | Partial (offline contract only) |
| Rich Menu | 建立、綁定、圖片、alias、batch | Planned |
| Audience | 建立、上傳、查詢、刪除、分享 | Planned |
| Insights | 發送、互動、followers、demographics | Planned |
| Coupon / Membership | 視 LINE 官方帳號所在地區與資格啟用 | Planned / Restricted |
| Channel / account link | Bot Info、Webhook Endpoint GET、Quota、發送統計、Aggregation、Narrowcast Progress 已接入唯讀 CLI；Webhook 寫入、Token、Account Linking 尚未接入 | Partial (offline contract only) |

此表只是能力分組，不能代替端點級覆蓋追蹤。參閱 [Endpoint Inventory](endpoint-inventory.yaml)：固定官方 LINE OpenAPI revision，列出 Messaging API、Insights、Audience Group、Channel Access Token 共 102 個操作。狀態 `integrated_offline` 僅表示程式碼接線及離線契約測試，**不代表真實 LINE Channel／地區資格／Production 已驗證**。每次擴充都必須核對 pinned SDK 的真實 Go symbols、更新 inventory 並新增測試；未知帳號權限標記 `review`，受地區／功能資格限制的操作標記 `required_eligibility_review`。
