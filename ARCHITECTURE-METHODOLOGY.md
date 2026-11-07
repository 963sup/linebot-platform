# 架構方法論與統一設計語言

本文件定義本專案使用 Go Idioms、Domain-Driven Design（DDD）、Hexagonal Architecture、KISS、YAGNI 與 DRY 時的共同語言、責任邊界與衝突決策原則。六種方法論解決的問題不同，不應被當成六套互相競爭或必須完整套用的架構模板。

## 一、六大方法論的核心語言

| 方法論 | 核心語言 | 解決的根本問題 |
|---|---|---|
| DDD | Domain、Boundary、Model、Context | 業務如何正確拆分與建模 |
| Hexagonal Architecture | Port、Adapter、Dependency、Isolation | 核心如何與技術細節隔離 |
| Go Idioms | Simplicity、Composition、Interface、Error | 如何寫出符合 Go 慣例的程式 |
| KISS | Simplicity、Clarity、Essential Complexity | 如何避免不必要的複雜度 |
| YAGNI | Necessity、Demand、Evidence、Timing | 哪些功能現在不該實作 |
| DRY | Knowledge、Duplication、Abstraction、Consistency | 如何避免同一知識重複定義 |

關鍵在於它們的責任不同：

- DDD：決定業務邊界。
- Hexagonal Architecture：決定依賴邊界。
- Go Idioms：決定程式表達方式。
- KISS：控制方案複雜度。
- YAGNI：控制實作範圍。
- DRY：控制知識重複。

## 二、各方法論的完整設計語言

### 1. Go Idioms — 程式設計語言

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Simplicity | 簡潔性 | 優先使用直接、容易理解的實作 |
| Readability | 可讀性 | 程式意圖清楚 |
| Composition | 組合 | 透過組合而非繼承建立行為 |
| Encapsulation | 封裝 | 隱藏內部實作細節 |
| Small Interface | 小介面 | 只抽象實際需要的行為 |
| Implicit Interface | 隱式介面 | 不需要顯式宣告實作關係 |
| Accept Interfaces, Return Structs | 接受介面、回傳具體型別 | 常見 Go API 設計經驗法則，非絕對規則 |
| Error as Value | 錯誤即值 | 明確回傳及處理錯誤 |
| Zero Value | 零值 | 優先讓型別零值具有合理用途 |
| Orthogonality | 正交性 | 各元件職責相互獨立 |
| Concurrency | 並行 | 使用 goroutine、channel 等協調工作 |
| Package Cohesion | 套件內聚 | 依責任與行為組織程式碼 |

參考：Go 官方《Effective Go》。這份文件是重要的 Go 慣例參考，但並非涵蓋所有現代 Go 功能的完整指南。

### 2. DDD — 領域戰略與戰術設計語言

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Domain | 領域 | 定義業務問題 |
| Subdomain | 子領域 | 劃分業務能力 |
| Core Domain | 核心領域 | 識別競爭優勢 |
| Ubiquitous Language | 通用語言 | 統一業務語意 |
| Bounded Context | 限界上下文 | 建立模型邊界 |
| Context Map | 上下文映射 | 確立跨邊界關係 |
| Aggregate | 聚合 | 定義一致性邊界 |
| Invariant | 不變條件 | 保護業務規則 |
| Domain Event | 領域事件 | 表達已發生的業務事實 |
| Anti-Corruption Layer | 防腐層 | 避免外部模型污染 |

前六項以戰略設計為主；後四項屬於戰術設計或整合模式。

### 3. Hexagonal Architecture — 架構邊界語言

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Inside / Outside | 內部／外部 | 區分應用核心與外部世界 |
| Port | 埠 | 定義互動契約 |
| Adapter | 適配器 | 連接外部技術 |
| Driving | 驅動端 | 發起對核心的操作 |
| Driven | 被驅動端 | 提供核心需要的外部能力 |
| Dependency Inversion | 依賴反轉 | 防止核心依賴具體技術 |
| Isolation | 隔離 | 保護業務邏輯 |
| Testability | 可測試性 | 不依賴外部設備測試核心行為 |
| Replaceability | 可替換性 | 降低替換外部技術的成本 |
| Technology Independence | 技術獨立性 | 避免業務規則綁定框架 |

參考：Alistair Cockburn 的 Hexagonal Architecture 原始論述。

### 4. KISS — 簡潔性決策語言

KISS 沒有正式的戰略術語體系。以下為實務上可採用的決策詞彙。

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Simplicity | 簡潔性 | 選擇最簡單的有效方案 |
| Clarity | 清晰性 | 降低理解成本 |
| Essential Complexity | 必要複雜度 | 保留問題本身不可避免的複雜性 |
| Accidental Complexity | 偶發複雜度 | 去除實作額外引入的負擔 |
| Minimalism | 最小化 | 減少非必要結構 |
| Cognitive Load | 認知負荷 | 降低閱讀與維護成本 |
| Directness | 直接性 | 減少不必要的間接層 |

### 5. YAGNI — 需求與範圍決策語言

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Necessity | 必要性 | 確認是否真的需要 |
| Current Requirement | 當前需求 | 以已存在的需求決策 |
| Speculation | 推測性需求 | 避免因想像而提前實作 |
| Premature Abstraction | 過早抽象 | 不提前建造抽象層 |
| Premature Optimization | 過早最佳化 | 沒有證據前不增加最佳化成本 |
| Incremental Development | 增量開發 | 隨實際需求演進 |
| Evidence-Based Design | 證據導向設計 | 用可驗證需求支持設計 |
| Reversibility | 可逆性 | 優先保留低成本修改能力 |

### 6. DRY — 知識一致性語言

DRY 的核心不是單純消除相似程式碼，而是避免同一份知識在多處獨立維護。

| 術語 | 中文 | 設計意義 |
|---|---|---|
| Knowledge | 知識 | 系統中需要維護的事實或規則 |
| Duplication | 重複 | 同一知識被多次獨立定義 |
| Single Source of Truth | 單一事實來源 | 避免同一規則的多個權威版本 |
| Abstraction | 抽象 | 集中表達共同知識 |
| Consistency | 一致性 | 防止規則互相矛盾 |
| Coupling | 耦合 | 評估共用實作產生的依賴 |
| Divergence | 分歧 | 辨識未來可能獨立演進的行為 |
| Change Propagation | 變更傳播 | 減少同一規則需要修改的位置 |

注意：兩段程式碼看起來相似，不代表必須合併。不同 Bounded Context 中，即使有重複結構，也可能代表不同的業務知識。

## 三、六大方法論發生衝突時如何決策

| 衝突 | 建議原則 |
|---|---|
| DDD vs KISS | 必要的業務複雜度保留，不必要的技術結構刪除 |
| DDD vs DRY | 不為共用程式碼破壞 Bounded Context |
| Hexagonal vs YAGNI | 只為真實外部依賴建立 Port，不預先設計所有 Adapter |
| Hexagonal vs Go Idioms | 優先使用小介面與組合，不強制建立大量 Java 式抽象 |
| DRY vs KISS | 只有共同知識確定且合併確實降低維護成本時才抽象 |
| YAGNI vs Extensibility | 保留合理的擴充能力，不提前實作未確認的擴充功能 |

## 四、統一成 10 個架構決策語言

用於 Golang、DDD、Hexagonal Architecture 的專案架構審查時，統一使用以下詞彙，而不是把六種方法論當成六套獨立檢查制度。

| 優先 | 統一語言 | 核心審查問題 |
|---:|---|---|
| 1 | Semantics（語意） | 命名與業務含義是否正確？ |
| 2 | Boundary（邊界） | 責任、模型及上下文是否清楚？ |
| 3 | Cohesion（內聚） | 相關行為是否集中？ |
| 4 | Coupling（耦合） | 是否產生不必要的依賴？ |
| 5 | Dependency（依賴） | 依賴方向是否正確？ |
| 6 | Simplicity（簡潔） | 是否有更簡單的有效解法？ |
| 7 | Necessity（必要性） | 這個設計現在是否需要？ |
| 8 | Consistency（一致性） | 業務規則是否一致且可維護？ |
| 9 | Testability（可測試性） | 核心行為是否能獨立驗證？ |
| 10 | Evolvability（可演進性） | 未來變更是否能局部完成？ |

## 五、三種設計語言的責任

1. **戰略設計語言（Strategic Design）**：定義業務領域、模型邊界、團隊關係與系統整合策略。
2. **戰術設計語言（Tactical Design）**：定義領域模型、業務規則、物件與行為。
3. **架構設計語言（Architectural Design）**：定義依賴方向、介面、適配器與外部技術的隔離方式。

### DDD 戰略設計與整合模式

| 英文術語 | 中文語意 | 核心用途 |
|---|---|---|
| Domain | 領域 | 系統要解決的業務問題範圍 |
| Subdomain | 子領域 | 拆分具有不同業務能力的問題空間 |
| Core Domain | 核心領域 | 最具競爭優勢、值得重點投入的領域 |
| Supporting Subdomain | 支援子領域 | 支援核心業務運作的能力 |
| Generic Subdomain | 通用子領域 | 通常可採用現成方案的通用能力 |
| Ubiquitous Language | 通用語言 | 業務與技術人員共同使用的精確語言 |
| Bounded Context | 限界上下文 | 模型與語意保持一致的適用邊界 |
| Context Map | 上下文映射 | 描述不同 Bounded Context 之間的關係 |
| Context Mapping | 上下文映射設計 | 分析與制定上下文之間的協作及整合方式 |
| Shared Kernel | 共享核心 | 多個上下文共同維護一小部分共享模型 |
| Customer–Supplier | 客戶－供應者 | 上下游團隊協作並協調需求 |
| Conformist | 遵奉者 | 下游接受上游模型，不要求其配合 |
| Anti-Corruption Layer | 防腐層 | 轉換外部模型，避免污染自身領域模型 |
| Open Host Service | 開放主機服務 | 上游提供明確、穩定的整合服務 |
| Published Language | 發布語言 | 透過共同約定的格式或契約交換資訊 |

### Hexagonal Architecture 架構設計語言

| 英文術語 | 中文語意 | 核心用途 |
|---|---|---|
| Application Core | 應用核心 | 容納應用行為及其領域模型 |
| Port | 埠 | 定義核心與外界互動的契約 |
| Adapter | 適配器 | 將外部技術與 Port 契約轉換銜接 |
| Primary Port | 主動埠 | 外界用來驅動應用的入口契約 |
| Primary Adapter | 主動適配器 | HTTP Handler、CLI、事件接收器等 |
| Secondary Port | 被動埠 | 核心要求外部提供能力的契約 |
| Secondary Adapter | 被動適配器 | 資料庫、外部 API、訊息發布器等 |
| Driving Adapter | 驅動適配器 | Primary Adapter 的另一種稱呼 |
| Driven Adapter | 被驅動適配器 | Secondary Adapter 的另一種稱呼 |
| Dependency Inversion | 依賴反轉 | 核心不直接依賴外部技術實作 |
| Inbound | 輸入方向 | 從外部呼叫核心的互動 |
| Outbound | 輸出方向 | 核心對外部能力發出的請求 |
| Composition Root | 組裝根 | 集中建立並連接各個實作元件 |

## 六、專案採用原則

1. 不因外部 API 存在，就建立新的 Domain 或 Bounded Context。
2. 不因 DDD 有某個戰術名詞，就建立對應 Package、Interface 或 Wrapper。
3. Port 必須有真實 consumer；Adapter 必須對應真實外部依賴。
4. Interface 由使用者擁有，保持小型，且必須有測試替身、效果隔離或多實作等實際理由。
5. 必要的業務複雜度放入 Domain；技術流程與供應商操作不偽裝成 Domain Model。
6. Candidate、Hypothesis 與未驗證能力不得提前實體化為完整戰術模型。
7. 相同知識只保留一個權威來源；不同 Bounded Context 的相似結構不因外觀相同而強行共用。
8. 架構文件應描述目前系統及已核准方向，不應用未實作藍圖掩蓋產品狀態。
9. 新增抽象或 Package 前，必須以需求、consumer、行為和測試證據證明必要性。
10. 當方法論發生衝突時，以業務語意正確、依賴方向安全、方案簡單且當前必要為優先。
