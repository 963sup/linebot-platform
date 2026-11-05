# Application Agent Rules

- Application 只能依賴 domain 與本層 Port；不 import adapters 或 infrastructure。
- Usecase 負責授權、流程、交易界線；工作流程與外部能力 Port 由唯一消費它的 capability 擁有，並有明確失敗語意。
- 回傳 Domain model 並表達 Domain persistence 語意的 Repository contract 可由 Domain 擁有；不得在 Application 維護重複 contract。
- 以明確 DTO 隔離 LINE／OpenAI／Gemini／Google SDK 的型別。
- 任務、外部發送需要可重試與冪等設計；不要宣稱端到端 exactly-once。
- 尚未建置的 usecase 不要建立假成功回傳或空 implementation。
