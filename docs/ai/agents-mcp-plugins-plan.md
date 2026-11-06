# Agents、MCP 與 Plugins：P0–P5 實施契約

初始基線：`258cd82ff3b5bb915457964ee53380ddf56debf2`。先提交本計畫，再於獨立分支一次交付六階段；不合併 PR、不部署、不建立憑證或改 IAM。這是歷史執行契約；其舊並行政策已由目前的九個 child threads 政策取代。SDK 依賴現由 `go.mod`、Catalog 與 build-tag dependency manifest 保留，不再保護空 scaffold。

## P0：設定與驗證

歷史階段曾修正當時的並行上限、角色 model 合法欄位、gopls 固定 HOME 路徑與 Google Developer Knowledge 的環境 key header。現行上限為九個 child threads，仍由資料驅動註冊與嚴格驗證保護一對一、名稱、未知鍵、路徑及 secrets。

## P1：行為保持重構

歷史階段曾在當時角色集合加入 go_code_refactor，並按專業角色分派局部重構、跨 package 語意變更、架構決策與交付清理。該角色集合已由現行 9 個角色取代；同一寫入路徑同時仍只能有一個 owner。

## P2：自主工具路由

按任務使用 gopls、Context7、GitHub、OpenAI Docs、Google Developer Knowledge，不強制全量呼叫。區分 declared、available、authenticated、verified、blocked；工具缺失明示安全 fallback，不能由 TOML 推論連線。

## P3：受控安裝

預設檢查，授權後才安裝。核對來源、固定 commit、保留個人內容、重跑冪等、不執行下載腳本。cc-skills 的已核對版本是通用 Engineering & Marketing 技能集合，不是 Go／DDD 專用能力。Google marketplace 加入不是 Plugin 安裝；重複 MCP／安裝認證未處理前，使用明示的 skills-only fallback。原生 CLI 能力／listing 格式不能驗證時停止，不以猜測補全。

## P4：治理與 Cloud Setup

同步根 AGENTS、角色矩陣、MCP／Plugins、Cloud 文件、核准安裝選項與驗收表。架構規則已收斂到 `docs/architecture/blueprint.md`；本文件不重建平行規範，也不藉文件更新搬移產品程式。

## P5：驗收與交付

新增註冊、欄位、路徑、安裝來源、archive、重跑、Marketplace 故障測試。執行能在當前環境完成的測試並確認 GitHub CI 的實際 HEAD。驗收規格見 agents-mcp-plugins-acceptance.md；實際結果記錄 PR，CI 或單元測試不能代替 Codex Host／模型／認證／Skills 發現驗收。

## 切片與安全

先計畫，再設定／角色／工具、治理／測試與 CI 修正。每個 commit 可回復；不重設他人的變更。外部技能不取得額外權限，不自行 OAuth、token、IAM、付費服務、production API 或發布。

## 來源與版本邊界

已透過 Context7 `/openai/codex` 查核 Agent loader、MCP 設定、Plugin CLI／marketplace。上游 main 不等於 Host 版本；仍須檢查 codex --version／--help 與實際能力。

- OpenAI：<https://developers.openai.com/codex/config-reference/>、<https://developers.openai.com/codex/mcp>、<https://developers.openai.com/codex/skills>
- Google：<https://developers.google.com/knowledge/mcp>、<https://developers.google.com/knowledge/authentication>
- 上游：<https://github.com/samber/cc-skills>、<https://github.com/google/skills>

沒有代表性實測資料可保證 99.99% 成功率；必要檢查通過、權限不擴大與未驗證事項透明才是交付標準。
