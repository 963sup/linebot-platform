# Architecture visual assets

`docs/architecture/blueprint.md` 是唯一規範性架構契約；本目錄的圖片只提供非規範性視覺導覽。若圖片、domain manifests、文字契約或 architecture tests 不一致，以可執行契約與測試為準。

- `go-ddd-hexagonal-architecture-overview.svg` 是概覽的可編輯來源。
- `go-ddd-hexagonal-architecture-overview.png` 是該 SVG 的固定尺寸渲染投影。
- `go-ddd-hexagonal-architecture-codex.svg` 與 `go-ddd-hexagonal-architecture-engineering.svg` 是互補的詳細視圖，不是另一份架構契約。
- `mermaid/conceptual-*.mmd` 是概念性 Mermaid 原始圖，包含尚未由 `docs/domain/model.yaml` 與 ADR 驗證的候選模型；不得視為已實作的 Domain 或資料契約。每張 Mermaid 都以可見標題與來源註解標示 hypothesis；其中 Account／Team／Access／Timekeeping 與替代 Identity Session 模型沒有因圖而獲得實作授權。現行 Attendance 的 Participant／ShiftAssignment／Punch、成功重試回傳原 Punch、審核歷史與獨立位置證據保留，以 ADR 0004 為準；圖中的未實作事件須先另立發布與 outbox 契約。

一般編輯預覽可在 repository root 以本機 Inkscape 重建 PNG：

```sh
inkscape docs/assets/architecture/go-ddd-hexagonal-architecture-overview.svg \
  --export-type=png \
  --export-filename=docs/assets/architecture/go-ddd-hexagonal-architecture-overview.png \
  --export-width=2400
```

本機預覽需使用 Inkscape，並提供 SVG 指定的 `Noto Sans CJK TC`；缺少時會依序 fallback 至 `Microsoft JhengHei` 與 generic `sans-serif`。一般本機渲染不宣稱 byte-for-byte reproducibility。

提交與 CI 使用 `make architecture-assets-check`。該 target 依 `tools/architecture-assets/render.env` 建置 digest-pinned container，使用固定 Inkscape／Noto CJK 套件版本渲染至暫存目錄，再以 byte comparison 驗證已提交 PNG。`tools/architecture-assets/packages.lock` 另鎖定並逐項比對完整 installed package set，使 transitive dependency 漂移直接令 image build 失敗。更新來源 SVG 後，使用 `scripts/render-architecture-assets.sh --write` 由同一受控 container 寫回 PNG；不得手工更新衍生圖。

Ubuntu snapshot service 在維護環境中不可用時，base digest 與完整 resolved package manifest 可防止靜默漂移，但不能保證套件從 archive 永久可取得；archive 若移除被鎖版本，受控 image 會明確 build 失敗，需由維護者審核並更新 manifest。

更新任何視覺資產時，必須同步檢查：layer-first 目錄、依賴方向、Port／Repository ownership、production verification wording、SVG 無障礙 metadata，以及 `tests/architecture` 的語意與完整性守門。
