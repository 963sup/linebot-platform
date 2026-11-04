# Adapter Layer

Inbound adapters 對接 HTTP webhook、Tasks、CLI；Outbound adapters 對接 LINE、OpenAI、Google Gen AI（Gemini）、Google Cloud 與 Google API。它們將供應商 API 轉成 application ports 規定的資料結構。只有具備明確 consumer、行為與測試的 adapter package 才建立；尚未接線的 SDK 能力記錄於 `go.mod`、SDK Catalog 與 build-tag dependency manifest，不以空目錄代表實作狀態。
