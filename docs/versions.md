# 版本與依賴策略

本 scaffold **沒有安裝任何 Go dependency**，也沒有有效的 `go.mod`／`go.sum`。

實作時請先查閱各專案當下 README／release：
- LINE Go SDK：`github.com/line/line-bot-sdk-go/v8/linebot`，其 README 要求 Go 1.26+。
- OpenAI Go SDK：`github.com/openai/openai-go/v3`，其 README 要求 Go 1.26+，並特別列出 HTTP/2 安全修補的最低 patch。
- Gemini Go SDK：`google.golang.org/genai`。
- Google Cloud：依服務採 `cloud.google.com/go/<service>`。
- Google Workspace／Discovery：`google.golang.org/api`。
- 程序內韌性選配：`go.linecorp.com/garr`。

正式採用前需固定實際版本、CI Go 工具鏈、建置映像與 Actions SHA；不得以浮動 `latest` 作為可重現 build 依據。
