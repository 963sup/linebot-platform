# Adapter Layer

Inbound adapters 對接 HTTP webhook、Tasks 與 CLI，負責 protocol 驗證、解析、DTO mapping 與呼叫 Application use case，不直接修改 Domain 或資料庫。Outbound adapters 對接 LINE、OpenAI、Google Gen AI（Gemini）、Google Cloud 與 Google API，實作 consumer-owned Application capability ports 或必要的 Domain repository contracts，並處理 SDK mapping、序列化與技術錯誤轉換。只有具備明確 consumer、行為與測試的 adapter package 才建立；尚未接線的 SDK 能力記錄於 `go.mod`、SDK Catalog 與 build-tag dependency manifest，不以空目錄代表實作狀態。
