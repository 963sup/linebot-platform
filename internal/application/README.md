# Application Layer

Commands、Queries、Usecases、DTO 與 capability-local Ports 定義應用層的輸入、輸出和協作流程。Usecase 擁有流程編排、交易界線與 application-level error mapping；工作流程或外部能力 Port 與唯一消費它的 capability 同 package，不建立全域 ports bucket。回傳 Domain model 並表達 Domain persistence 語意的 Repository contract 是例外，可由 Domain 擁有，且不與 Application port 重複。Application 只依賴 Domain 與自身抽象，不依賴 SDK 或 Adapter。Package 隨真實 capability 建立，且必須具備明確 owner、consumer、行為與測試；未實作的 command、query、DTO、port 或 usecase 不以空目錄預留。
