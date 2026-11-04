# Codex MCP、Plugins 與受控 Skills 安裝

`.codex/config.toml` 是開發工具連線的唯一設定來源；LINE Bot runtime 仍使用 Go SDK。MCP 設定、Host 載入、認證、最小工具呼叫、雲端部署必須分別驗證。

## MCP 清單與路由

| MCP | 儲存庫設定 | 用途／前置條件 |
| --- | --- | --- |
| `gopls` | 啟用 | setup 固定安裝到 `$HOME/.local/bin/gopls`；Go 語意分析 |
| `context7` | 啟用 | 函式庫／SDK 版本文件；匿名額度可能受限 |
| `github` | 啟用 | Host 提供 GITHUB_MCP_TOKEN；寫入仍需任務授權 |
| `openaiDeveloperDocs` | 啟用 | OpenAI／Codex 官方文件，不需 OpenAI API key |
| `google_developer_knowledge` | 啟用 | 官方文件；Host 提供 DEVELOPERKNOWLEDGE_API_KEY |
| `firestore_toolbox` | 停用 | 固定版本與五個唯讀工具；隔離 staging／最小 ADC |
| `google_cloud_run` | 停用 | 服務、revision 與明確授權的環境調查 |
| `google_firestore` | 停用 | 隔離 Firestore；不與 Toolbox 無目的重複啟用 |
| `google_cloud_cli` | 停用 | 需最小權限 Google 身分及明確任務授權 |
| `google_cloud_logging` | 停用 | 授權環境日誌，仍需去敏與範圍限制 |
| `google_cloud_monitoring` | 停用 | 授權環境監控查詢 |
| `google_calendar` | 停用 | 使用者／Workspace 資料權限另行驗證 |
| `google_drive` | 停用 | 使用者／Workspace 資料權限另行驗證 |
| `google_sheets` | 停用 | 使用者／Workspace 資料權限另行驗證 |
| `google_docs` | 停用 | 使用者／Workspace 資料權限另行驗證 |
| `google_gmail` | 停用 | 使用者／Workspace 資料權限另行驗證 |

共 16 個宣告，5 個啟用；停用項目是保留配置，不代表 endpoint 已在此 Host 驗證。所有 server 仍為 optional。工具失敗使用根 AGENTS 的安全 fallback，不以安裝更多 MCP 取代診斷。

`gopls` 以 HOME 明確路徑啟動，不依賴 setup shell 的 PATH 能延續到 agent phase。此契約適用 POSIX／WSL／Cloud；原生 Windows 須以 Host 相容路徑另外驗證，不能聲稱此 Bash 設定通用。Detached gopls 讀取已儲存檔案，分析時可能下載 Go modules。

Google Developer Knowledge 使用 `env_http_headers` 將 `X-Goog-Api-Key` 對應到環境變數；不要把 API key、ADC 路徑或 OAuth secret 放進 repo。缺 key 不是已認證，也不能為補齊工具而自行啟用 API／建立 key。此文件 MCP 不能代替 Firestore／Calendar 的資料授權。

GitHub 使用 `writes` approval policy，不保證 token 唯讀；權限仍由 token 與 Host 控制。本設定未加入 X-MCP-Readonly 或 X-MCP-Toolsets header，不在文件宣稱不存在的 allowlist。遠端寫入使用 expected HEAD 並驗證實際 CI。其他 Google 服務維持停用與 prompt。

## Plugin 與 Skills：可重現且不擴權

`.codex/plugins.lock.json` 鎖定兩個已核對來源：

| 來源 | 固定 commit | 本工具安裝內容 |
| --- | --- | --- |
| samber/cc-skills | `50a33836b20f690c668af557bd9ef9d07a8c7ef2` | 通用 Engineering & Marketing Skills；不是 Go／DDD 專用規範 |
| google/skills | `3ae864a38b66642c761c8b3dad6b72e38ca3cd43` | retrieving-developer-knowledge 與其 references／授權文件 |

安裝方式明確標記為 **skills-only fallback**，不宣稱原生 Plugin 已安裝。它避免依賴跨版本不一致的原生 Plugin 語法、浮動版本，以及 Google Plugin 自帶的重複 MCP／ON_INSTALL 認證。新工作階段仍需確認 Host 真的發現 SKILL.md。

```bash
# 預設檢查：不安裝、不認證、不呼叫遠端服務。
bash scripts/check-codex-plugins.sh

# 已授權的固定來源檔案安裝；不執行下載內容。
bash scripts/setup-codex-plugins.sh

# 額外註冊 Google marketplace；只有 CLI 支援 --ref 與可驗證 listing 才執行。
bash scripts/setup-codex-plugins.sh --register-google-marketplace
```

Wrapper 以本機 Go 建置暫存執行檔，因此編譯器可能使用 Go cache／工具鏈；「檢查不修改」指不修改技能、個人 Codex 設定、認證或外部資料。結束碼：0 表示技能檔案符合鎖定收據，2 表示缺少／阻塞，1 表示錯誤。**0 不代表 MCP 或 Host 已通過驗收。**

安裝器使用 bare Git fetch 的精確 commit、核對 FETCH_HEAD，再讀取 archive。拒絕路徑穿越、symlink、特殊檔與超量 archive；保留上游 LICENSE。下載內容不執行、不 checkout hooks；安裝收據記錄來源與內容 digest。重跑相同內容不重裝，不覆寫無收據、被修改或不同版本目錄。升級需先審查新的鎖定 commit 與差異，不自動追蹤 main。

cc-skills 既有原生 Plugin 使用者應先確認 Host 是否已提供相同 Skills；主 Agent 不應再重複安裝 fallback。檔案檢查器無法完整推論各版本原生 Plugin inventory，因此未匹配收據的原生安裝不得宣稱 missing 能力或逕行覆寫。

上游 README 的 `codex plugin add github:samber/cc-skills`、`codex plugin update cc-skills` 是上游示例，不保證每版 CLI 支援。先查本機 `codex plugin --help`／`add --help`；本專案安裝器不以猜測的指令自動更新。需要切換原生安裝時，先停止重複的 fallback 載入並依當前官方文件執行。

Google 官方入口是 `codex plugin marketplace add google/skills`；本工具在支援的 Host 額外固定 `--ref`。Marketplace 加入不等於 Plugin 安裝。在 `/plugins` 選擇項目前核對其 MCP、Hooks、認證與 scope。`google-cloud-developer` 自帶 Developer Knowledge；本專案已有同用途 MCP，所以不自動啟用第二份。Firestore 等會操作資料的 Plugin 也不因 Marketplace 已加入就取得授權。

## 安全界線

缺少技能且符合當次已核准開發範圍時，主 Agent 可自行執行 setup；唯讀角色只交接。這不授權新增憑證、OAuth、IAM、付費服務、production API 或更動個人模型／approval。第三方 Skill 仍是未信任參考內容；禁止其 credential fallback 自動選用其他帳號、把 key 放進 URL 或執行未審查腳本。

## 驗證層次

`go run ./cmd/codexcheck` 是離線契約檢查。`cmd/codexplugins` 檢查技能收據／檔案、工具與憑證存在性，永遠不把此結果稱為 Host verified。實際驗收另依 [驗收表](agents-mcp-plugins-acceptance.md)，在已信任且載入此分支的 Codex Host 完成角色、Skills 發現與五個 MCP 的最小唯讀呼叫。

官方參考：[Codex MCP](https://developers.openai.com/codex/mcp)、[Codex Skills](https://developers.openai.com/codex/skills)、[Config Reference](https://developers.openai.com/codex/config-reference/)、[Google Developer Knowledge MCP](https://developers.google.com/knowledge/mcp)、[Google Skills](https://github.com/google/skills)、[cc-skills](https://github.com/samber/cc-skills)。上游 main／Context7 索引不取代實際 CLI 版本及 --help 的相容性驗證。
