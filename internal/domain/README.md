# Domain Layer

純業務規則：entities、aggregates、policies、specifications、factories、repositories、`value-objects/`（Go package `valueobjects`）、events。這些 tactical building blocks 按真實行為使用，不代表對應 Context 已獲 strategic promotion。

禁止 import application、adapter、infrastructure 或外部 SDK；請閱讀 [AGENTS.md](AGENTS.md)。DDD 分類只位於 strategically implemented Context 或 ADR-authorized candidate tactical model 內，不在 Domain 根目錄預建空分類。

`identity/` 是唯一 strategically implemented Bounded Context，具備 Entity、Aggregate、Value Objects、Policy、Specification、Factory、Repository 讀取契約、Domain Events 與測試。ADR 0003 另授權 `messaging-delivery/`、`audience/`、`experience-presentation/`、`promotion/`、`workspace-automation/` 作為 `candidate/hypothesis` tactical models；它們的程式碼與測試不構成 strategic validation。

每個 Domain model directory 的 Value Object 檔案皆放在 `value-objects/`，Go package 名稱使用 `valueobjects`。
