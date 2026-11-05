# 目錄配置清單

本文件說明目錄治理方式，不保存空目錄快照。實際目錄與 Go package 以受測 commit 的 Git tree、`go list ./...` 與 [架構藍圖](blueprint.md) 為準；目錄存在不代表功能已部署或外部服務已授權。

## 建立條件

- `internal/domain/<context>/`：只有已確認業務語言、不變量與 Aggregate owner 時建立；DDD 分類留在 Bounded Context 內。
- `internal/application/usecases/<capability>/`：只有具體 command／query、consumer 與測試時建立。
- `internal/application/usecases/<capability>/`：outbound port 必須與唯一 consumer 同 package；禁止重新建立全域 ports bucket。
- `internal/adapter/`：只有 protocol／provider mapping 或 I/O 行為已實作時建立。
- `internal/infrastructure/`：只有共用 client factory、transport、認證、觀測或 lifecycle primitive 已實作時建立。

不得為未排程能力預建 SDK、Domain 或 Application 空目錄，也不得以 `.gitkeep` 作為架構、版本或完成度證據。空 package 應刪除；未實作能力記錄於 Roadmap 或 SDK Catalog。

## 穩定命名

- Value Object 目錄使用 `<context>/value-objects/`，Go package 名稱使用 `valueobjects`。
- Adapter 使用 `internal/adapter` 單數；測試集合使用 `tests` 複數。
- `identity/` 的 `aggregates`、`entities`、`events`、`factories`、`policies`、`repositories`、`specifications`、`value-objects` 是已實作結構，不是全域 scaffold 範本。

## 依賴保留

SDK 版本與能力不由目錄保留：

1. `go.mod`／`go.sum` 是版本與解析結果來源。
2. `docs/sdk/catalog.toml` 記錄 module、用途、接線狀態與預期邊界。
3. `internal/infrastructure/dependencies/modules.go` 以 `dependencies` build tag 保留尚未 runtime 啟用的核准 import。
4. `go run ./cmd/codexcheck` 驗證三者的一致性，不驗證空目錄。

## 查核命令

```bash
git ls-tree -dr --name-only HEAD
go list ./...
find internal -name .gitkeep -print
go run ./cmd/codexcheck
```

`find` 有輸出時應判斷其是否為待清理歷史檔，不得據此推論該 capability 已保留或已完成。
