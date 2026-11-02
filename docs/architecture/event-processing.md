# Webhook 事件處理（目前與目標）

## 現有開發用流程

1. POST /webhook/line，限制 1 MiB body。
2. 使用 LINE Go SDK webhook.ParseRequest 檢查 X-Line-Signature 並轉換事件。
3. 目前只處理文字 MessageEvent：普通文字回應固定說明；輸入 flex／卡片回覆簡單 Flex bubble。
4. Reply 同步呼叫 LINE Messaging API；只有呼叫成功時回 200。
5. 沒有憑證時 webhook 回 503；不支持的事件會被正常 ACK，不記錄敏感內容。

## 限制（上 production 之前必須解決）

- 未寫入 Firestore Inbox/Outbox；沒有跨 Cloud Run instance 去重。
- 一次 webhook 可能包含多個事件，部分回覆成功後發生錯誤並重送時，可能重複 Reply。
- ReplyToken 有時效；不能把 Reply 放進長時間 AI 任務。
- 對 LINE 送出請求時若網路逾時，即使 HTTP 502，遠端也可能已接受請求。
- 如需可靠長任務，必須增加持久事件工作流程：Firestore Inbox+Outbox 交易、Cloud Tasks OIDC、worker lease、reconcile、發送結果 unknown 處理、unsend／edit。

目前的入口僅供本機功能驗證，**不建議直接使用於正式 LINE 官方帳號**。
