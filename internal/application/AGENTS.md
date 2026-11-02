# Application Agent Rules

- Application 只能依賴 domain 與本層 Port；不 import adapters 或 infrastructure。
- Usecase 負責授權、流程、交易界線；Repository Port 與事件分發應有明確失敗語意。
- 以明確 DTO 隔離 LINE／OpenAI／Gemini／Google SDK 的型別。
- 任務、外部發送需要可重試與冪等設計；不要宣稱端到端 exactly-once。
- 尚未建置的 usecase 不要建立假成功回傳或空 implementation。
