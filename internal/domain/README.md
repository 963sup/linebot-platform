# Domain Layer

純業務規則：entities、aggregates、policies、specifications、factories、repositories、`value-objects/`（Go package `valueobjects`）、events。

禁止 import application、adapter、infrastructure 或外部 SDK；請閱讀 [AGENTS.md](AGENTS.md)。DDD 分類位於具備真實業務不變量的 Bounded Context 內，不在 Domain 根目錄預建空分類。`identity/` 已實作 Entity、Aggregate、Value Objects、Policy、Specification、Factory、Repository 讀取契約、Domain Events 與測試。各 Bounded Context 的 Value Object 檔案皆放在該領域的 `value-objects/`，Go package 名稱使用 `valueobjects`。
