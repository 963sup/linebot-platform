# MCP 與 Codex Plugins

## 決策

優先採用使用者已連線的 GitHub 工具，不預設 GitHub MCP token。外部 MCP、plugin 都先視為**尚未安裝**；只在確有需求、確認合法性與最低權限後啟用。

- **MCP**：Codex 開發時取用 GitHub、官方文件或其他外部資料。
- **Plugin**：將選定 MCP 與 skills 打包成可管理的安裝單位。
- **Runtime SDK**：Cloud Run 上的 bot 呼叫 LINE、OpenAI、Gemini、Google Cloud；**不透過 Codex plugin 執行正式業務請求**。

`.codex/config.toml.example` 只有註解範例。未完成 schema 審查、版本鎖定與使用者授權前，不建立看似可用的 `plugin.json`、`mcp.json` 或 marketplace manifest。

探索外部技能可參考 [skills.sh](https://www.skills.sh/)；須查核來源、執行腳本、license、權限與版本後才匯入。
