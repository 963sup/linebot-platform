# 本機開發

## 環境

安裝 Go 1.27.3、Git、make。確認版本後：

```bash
go version
make bootstrap # 首次解析依賴並產生 go.sum
make check
make run-webhook
```

`curl http://localhost:8080/healthz` 應回傳 `ok`。

## LINE 憑證

從安全的環境變數提供 `LINE_CHANNEL_SECRET` 與 `LINE_CHANNEL_ACCESS_TOKEN`，兩者必須一起設定；未設定時只開放 health，webhook 回 503。不要把值寫進 git、PR、公共 log 或公開 issue。

對外 webhook 在本機測試時須透過 HTTPS 端點，並以測試 LINE Channel 驗證。這個專案**沒有自動建立 LINE Channel 或 GCP 資源**。

目前只支援文字事件，輸入 `flex` 或 `卡片` 時回 Flex bubble，其餘文字回固定說明。Webhook 已驗簽但仍是**無持久去重的同步 MVP**。

## Push CLI

```bash
# 缺 --send 時不會對外發送。
go run ./cmd/ops push --to "<recipient>" --text "hello"
# 經明確授權，且已有 LINE_CHANNEL_ACCESS_TOKEN 時可加 --send。
```

若發送結果不確定，後續重試務必提供同一 `--retry-key`、同一接收者與相同 payload。CLI 會在第一次發送前輸出 UUID；務必妥善保存，不要放在公開日誌或 issue。

## SDK 安裝狀態

`go.mod` 只表示模組版本需求。`go.mod`／`go.sum` 已於 GitHub Actions Go 1.27.3 環境解析並提交。後續升級模組時應在有網路的環境執行 `go mod tidy`、`go mod download`，檢查並提交異動；CI 持續保留依賴解析的審查用 artifact。

Workspace、Cloud Services、OpenAI、Gen AI、garr 已宣告但尚未接線；不能只憑目錄／依賴存在就認定功能完成。
