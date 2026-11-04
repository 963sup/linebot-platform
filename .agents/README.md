# Agent Skills

`.agents/skills/` 下放置本專案專用 `SKILL.md`。這些是開發規範與檢查流程，**不是自動具備外部 API 權限的工具**。

目前規劃的 skills：
- `linebot-domain-change`
- `linebot-messaging-feature`
- `linebot-webhook-safety`
- `linebot-cloudrun-release`
- `linebot-api-upgrade`

使用者仍需授權工具連線與任何外部副作用；不得自動安裝未經審查的 skills。參見 [Codex 指南](../docs/ai/codex.md)。
