# Domain Agent Rules

- 保持 domain 為純 Go 業務邏輯，不 import LINE、OpenAI、Gemini、Google API 或基礎設施。
- 只有業務不變量與聚合操作；不得呼叫網路、檔案或環境變數。
- Entity / Aggregate 以行為和一致性邊界為主，不把 SDK DTO 當成 domain 模型。
- Value Object 目錄在每個 strategically implemented Context 或 ADR-authorized candidate tactical model 內一律使用 `value-objects/`（例如 `internal/domain/audience/value-objects/`）。Go package 名稱仍為 `valueobjects`。Import path 可含 `-`，例如 `import valueobjects "github.com/963sup/linebot-platform/internal/domain/identity/value-objects"`。禁止新增名為 `valueobjects/` 或 `value_objects/` 的替代目錄；新增前檢查 `tests/architecture/value_objects_directory_test.go`。
- 先寫業務測試再實作；沒有真實不變量、owner 與 consumer 時，不建立分類 Package 或空目錄。

- `messaging-delivery/`、`audience/`、`experience-presentation/`、`promotion/` 是 ADR 0003 授權的 `candidate/hypothesis` tactical models；不得因已有程式碼、Adapter 或測試而宣稱 strategic promotion。

- Implemented Bounded Context 只建立實際需要的分類 Package；不為填滿圖而增加空 Policy、Factory、Specification 或 Event。
- 禁止以新 Domain 包裝既有 DTO 卻仍在 Application 重複判斷 Scope、綁定狀態與授權規則。必須在每次重構檢查重複模型與舊呼叫點，並明確記錄尚待退役的技術契約。
