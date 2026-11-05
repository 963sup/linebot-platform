# Agents／MCP／Plugins 驗收規格

本檔是可重複執行的驗收規格，不是成功紀錄。實際結果記錄於 PR，並附受測 commit、Host／CLI／Go 版本、命令、結果與未驗證原因。禁止填入 token、API key、私訊或 production 資料。

## 儲存庫與離線測試

| 項目 | 通過條件 |
| --- | --- |
| 註冊 | 固定 8 個角色與 config 一對一；無重複／未知鍵／越界路徑 |
| 並行上限 | Validator 接受 1／4／8／9，拒絕 0／10；深度只接受 1；此單元測試不是 Host 排程證明 |
| MCP | 保留 10 宣告、9 啟用、1 個服務維持停用；無明文 secrets |
| 安裝來源 | 僅核准來源、精確 commit、實際 FETCH_HEAD 一致 |
| 安全安裝 | 不能越界、跟隨 symlink、覆寫個人內容；失敗不發布半成品；重跑冪等 |
| Marketplace | CLI 不支援即 blocked；既有來源 pin 不符不覆寫；認證失敗不繞過 |
| Go／Shell | codexcheck、相關單元／race、gofmt、vet、build、shell -n、make check |
| SDK | go.mod／go.sum、Catalog 與 build-tag dependency manifest 一致；不要求空 scaffold |

## 必須在實際 Host 執行

| 能力 | 最小驗收 | 未具備環境時 |
| --- | --- | --- |
| 8 個角色 | 在信任專案確認載入，啟動受控唯讀／獨立測試任務；確認模型有效 | 未驗證，不能由 TOML 推論 |
| gopls | 已儲存 Go 檔的符號／references／diagnostics | Go CLI fallback 並註明 |
| Context7 | 查詢目前使用 SDK 的一段版本文件 | 官方文件 fallback |
| GitHub | 唯讀取得指定 repo／branch HEAD | 不宣稱遠端 CI 或 SHA |
| OpenAI Docs | 查詢 Codex 設定官方文件 | 官方網站 fallback |
| Google Knowledge | 查詢 Google API 文件，確認不是 401／403／錯誤 payload | 缺 key／網路則 blocked，不能自行建立憑證 |
| Skills | 新工作階段找到安裝後 SKILL.md，並按需讀取 | 檔案存在只算 installed，host unverified |
| Google Marketplace | listing 顯示來源／固定 revision；Plugin 狀態另記 | 不把 marketplace added 當 plugin installed |

## Agent 路由與故障注入

依 agents.md 的任務案例檢查 owner、寫入範圍、必要 MCP 與交接。停用 gopls／模擬 Context7 網路失敗／移除測試用憑證時，應明示 fallback，不修改個人權限。九個 child threads 滿載時，第十個必須被明確拒絕；兩個寫入任務有共用檔案必須串行。沒有部署授權時不執行 Cloud Run／LINE 發送。

實作角色、獨立 reviewer、最後整合檢查分開記錄；只有自我檢查不能聲稱完成獨立審查。涉及 production／OAuth／IAM 的整合驗證不屬此開發工具 PR 自動操作範圍。

## 回復

程式／設定依 commit revert；不以重設整個分支回復其他人的修改。技能更新不覆寫，舊目錄由使用者確認後另行備份處理。Marketplace 是個人 Host 狀態，任何撤銷須依該 Host 官方命令與當次授權，不能盲目刪除 ~/.codex 或 ~/.agents。

所有必要檢查通過才可標記對應層次完成；CI 排隊／缺少 Host 或認證時保留 pending／blocked。沒有長期代表性實測，不能宣稱 99.99% 整體成功率。
