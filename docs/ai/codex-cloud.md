# Codex Cloud：三個設定檔

環境設定方式以 [Cloud environment](../cloud-environment.md) 為準。

1. `mise.toml` 固定工具版本與 tasks。
2. `bootstrap.sh` 安裝 mise 並執行 `mise install`／tasks。
3. `.codex/config.toml` 定義 MCP 啟動命令與工具權限。

Install script 使用 `bash /workspace/linebot-platform/bootstrap.sh setup`。
MCP host 載入受信任的專案設定後，會自行執行 stdio 命令；不需要維持背景
server，也不需要重複安裝或在新任務開始時 reset／clean Git。

設定檔不能強迫 Codex Cloud host 載入工具。本對話的可用工具中沒有 Repomix；
不能用 CLI 列表、安裝成功或本地 handshake 推論它已出現在對話中。
新任務必須真的找到並呼叫 Repomix `pack_codebase` 才算完成驗收。

本地資料、mise state 和 cache 位於 `/workspace`。腳本不覆寫個人 Codex
設定、不建立憑證、不做 OAuth、IAM 或部署。遠端 MCP 仍需要 agent phase
可連線的網路及各自憑證。Google／LINE／Docker／OpenAPI 整合預設停用，
依主文件準備前置條件後再啟用，避免開局出現沒有工具可用的失敗 server。
