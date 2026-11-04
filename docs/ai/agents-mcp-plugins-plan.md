# Agents、MCP 與 Plugins：P0–P5 實施契約

基線：`258cd82ff3b5bb915457964ee53380ddf56debf2`。本計畫涵蓋一次交付的六個階段；只在獨立分支實作並提出 PR，不合併、不部署、不建立憑證或變更 IAM。既有 27 個角色、16 個 MCP、SDK 依賴及預留骨架不得任意刪除。

## P0：修復設定與驗證

接受最多五個並行 thread；角色模型欄位合法但不猜測或改寫模型 ID。將角色註冊改為資料驅動，檢查註冊與檔案一對一、名稱、未知鍵、路徑穿越與憑證。統一 gopls 與 setup 安裝位置；Google Developer Knowledge 使用環境變數映射 `X-Goog-Api-Key`，不保存金鑰。

## P1：新增行為保持重構

新增 `go_code_refactor`；其餘 27 個角色保留。局部去重與控制流程簡化由新角色負責，跨 package 語意變更仍交 `gopls_go_refactor`，架構決策交 `domain_architect`／`architecture_alignment_engineer`，驗收後清理交既有交付角色。同一寫入路徑同時只有一個 owner。

## P2：自主工具路由

按任務使用 gopls、Context7、GitHub、OpenAI Developer Docs、Google Developer Knowledge；不強迫所有任務呼叫全部工具。區分 declared、available、authenticated、verified 與 blocked。未掛載時說明原因並使用安全 fallback；禁止把設定存在當成實際連線證據。

## P3：受控安裝

提供預設唯讀的 preflight 及明確安裝入口。只接受核准上游、固定 commit 和實際存在的 CLI 功能；重跑不得重複安裝或覆寫既有內容。`samber/cc-skills` 是文件技能集合，不能宣稱涵蓋 Go。Google marketplace 加入不等於 plugin 安裝；只安裝已選定且支援的項目。CLI 不支援時才走已核准的 pinned skills fallback；認證失敗不可以其他方法繞過。禁止自動更新浮動分支、OAuth、token 建立或 production 操作。

## P4：治理與文件

同步根 AGENTS、AI 操作文件、Codex Cloud setup 與目錄命名。DESIGN 為架構權威，config 為連線設定權威，角色檔定義職責。允許經本計畫核准的工具／skills 安裝，仍禁止執行未審查的第三方腳本。不得降低 DDD、Hexagonal 或 Go 契約。

## P5：驗收與交付

新增設定、註冊、邊界與安裝故障測試；執行可用的本機測試、shell 語法檢查及 GitHub CI。檢查遠端 HEAD 和 PR diff。實際 Codex CLI／Cloud 的角色載入、五個 MCP 連線、Plugin 發現與隔離 staging 驗證須分別保留證據；缺少 host／認證時標記未驗證，不能由單元測試或 CI 推論成功。

## 執行順序與回復

先建立本計畫，再以設定／角色、安裝／文件、測試修正切片提交。每個切片可獨立 revert；安裝既有內容不覆寫，無自動清除個人設定。變更僅限 Codex 開發工具與規範，不變更產品 runtime／資料契約。

## 來源與版本邊界

已透過 Context7 查詢 `/openai/codex` 的 Agent loader、MCP 設定和 Plugin CLI。上游 `main` 文件不等於目標 host 的已發布版本；以 `codex --version` 和各命令 `--help` 驗證能力後才執行。

- OpenAI：<https://developers.openai.com/codex/config-reference/>、<https://developers.openai.com/codex/mcp>、<https://developers.openai.com/codex/skills>
- Google：<https://developers.google.com/knowledge/mcp>、<https://developers.google.com/knowledge/authentication>
- Plugin 來源：<https://github.com/samber/cc-skills>、<https://github.com/google/skills>

沒有實測資料可支持 99.99% 成功率；完成標準是必要檢查通過、權限不擴大、未驗證事項明確揭露。
