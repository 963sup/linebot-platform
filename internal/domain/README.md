# Domain Layer

純業務規則：entities、aggregates、policies、specifications、factories、repositories、`value-objects/`（Go package `valueobjects`）、events。

禁止 import application、adapter、infrastructure 或外部 SDK；請閱讀 [AGENTS.md](AGENTS.md)。根目錄的各種 DDD 分類目錄仍屬預留；`identity/` 已開始實作 Aggregate、Value Objects、Factory、Policy、Repository 契約與測試。各 Bounded Context 的 Value Object 檔案皆放在該領域的 `value-objects/`，Go package 名稱使用 `valueobjects`。
