# Adapter Agent Rules

- Inbound：先驗證、轉換，再交給 application usecase，勿在 handler 寫業務決策。
- LINE webhook 原始 body 必須先驗簽；未知事件須安全處理。
- Outbound：所有 SDK request / response 僅在 adapter / infrastructure 層存在。
- 對外呼叫須有 context、timeout、明確錯誤型別及必要的契約測試。
- CLI 必須將 root parent context 傳入 handler 與 usecase，command deadline 必須從 parent 派生；Adapter 擁有的 runtime 必須在命令結束時明確 `Close`。Bootstrap 只組裝依賴與資源，不擁有 CLI presentation contract。
- 特別防範重送、部分成功、外部操作結果未知、私訊洩漏與多 Channel token 混用。
- LINE MINI App 只接受經 LINE 官方端點驗證的 LIFF ID token；Origin／Fetch Metadata 是瀏覽器守門而非身分來源。Geolocation 是一次性裝置讀值，不是實體在場證明；API response、log、trace 與 browser storage 禁止不必要的 raw coordinates。
