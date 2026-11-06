# Domain Layer

純業務規則：entities、aggregates、policies、specifications、factories、repositories、`value-objects/`（Go package `valueobjects`）、events。這些 tactical building blocks 按真實行為使用，不代表對應 Context 已獲 strategic promotion。

禁止 import application、adapter、infrastructure 或外部 SDK；請閱讀 [AGENTS.md](AGENTS.md)。DDD 分類只位於 strategically implemented Context 或 ADR-authorized candidate tactical model 內，不在 Domain 根目錄預建空分類。

`identity/` 與 ADR 0004 核准的 `attendance/` 是 strategically implemented Bounded Context；Attendance 擁有打卡配對、異常覆核、修正歷史與原子保存契約，但不擁有 HR／薪資、LINE 驗證、實體在場證明或通知。ADR 0003 另授權 `messaging-delivery/`、`audience/`、`experience-presentation/`、`promotion/`、`workspace-automation/` 作為 `candidate/hypothesis` tactical models；不得因已有程式碼、Adapter 或測試而宣稱 strategic promotion。

每個 Domain model directory 的 Value Object 檔案皆放在 `value-objects/`，Go package 名稱使用 `valueobjects`。
