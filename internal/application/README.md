# Application Layer

Commands、Queries、Usecases、Ports 與 DTO 定義應用層的輸入、輸出和協作流程。只依賴 domain 與應用層自身抽象，不依賴 SDK 或 adapter。Package 隨真實 capability 建立，且必須具備明確 owner、consumer、行為與測試；未實作的 command、query、DTO、port 或 usecase 不以空目錄預留。
