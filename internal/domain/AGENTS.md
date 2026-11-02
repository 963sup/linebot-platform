# Domain Agent Rules

- 保持 domain 為純 Go 業務邏輯，不 import LINE、OpenAI、Gemini、Google API 或基礎設施。
- 只有業務不變量與聚合操作；不得呼叫網路、檔案或環境變數。
- Entity / Aggregate 以行為和一致性邊界為主，不把 SDK DTO 當成 domain 模型。
- Value Object 的預留位置是 `valueobjects/`，Go package 使用 `valueobjects`；不使用連字號目錄 `value-objects/`。
- 先寫業務測試再實作；新增真實程式後移除該位置的 `.gitkeep`。
