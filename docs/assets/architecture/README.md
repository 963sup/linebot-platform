# Architecture visual assets

`docs/architecture/blueprint.md` 是唯一規範性架構契約；本目錄的圖片只提供非規範性視覺導覽。若圖片、domain manifests、文字契約或 architecture tests 不一致，以可執行契約與測試為準。

- `go-ddd-hexagonal-architecture-overview.svg` 是概覽的可編輯來源。
- `go-ddd-hexagonal-architecture-overview.png` 是該 SVG 的固定尺寸渲染投影。
- `go-ddd-hexagonal-architecture-codex.svg` 與 `go-ddd-hexagonal-architecture-engineering.svg` 是互補的詳細視圖，不是另一份架構契約。

從 repository root 重建 PNG：

```sh
inkscape docs/assets/architecture/go-ddd-hexagonal-architecture-overview.svg \
  --export-type=png \
  --export-filename=docs/assets/architecture/go-ddd-hexagonal-architecture-overview.png \
  --export-width=2400
```

更新任何視覺資產時，必須同步檢查：layer-first 目錄、依賴方向、Port／Repository ownership、production verification wording、SVG 無障礙 metadata，以及 `tests/architecture` 的語意與完整性守門。
