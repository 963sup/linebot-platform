# Architecture visual assets

`docs/architecture/blueprint.md` 是唯一規範性架構契約；本目錄的圖片只提供非規範性視覺導覽。若圖片、domain manifests、文字契約或 architecture tests 不一致，以可執行契約與測試為準。

- `go-ddd-hexagonal-architecture-overview.svg` 是概覽的可編輯來源。
- `go-ddd-hexagonal-architecture-overview.png` 是該 SVG 的固定尺寸渲染投影。
- `go-ddd-hexagonal-architecture-codex.svg` 與 `go-ddd-hexagonal-architecture-engineering.svg` 是互補的詳細視圖，不是另一份架構契約。

從 repository root 在本機重建 PNG：

```sh
inkscape docs/assets/architecture/go-ddd-hexagonal-architecture-overview.svg \
  --export-type=png \
  --export-filename=docs/assets/architecture/go-ddd-hexagonal-architecture-overview.png \
  --export-width=2400
```

維護環境需使用 Inkscape，並提供 SVG 指定的 `Noto Sans CJK TC`；缺少時會依序 fallback 至 `Microsoft JhengHei` 與 generic `sans-serif`。Inkscape、fontconfig、字型版本或 rasterizer 不同都可能改變文字度量、antialiasing 與 PNG bytes，因此此命令只描述本機維護流程，不宣稱跨作業系統或跨工具版本可產生 byte-for-byte 相同輸出。提交前以實際渲染結果更新 architecture test 中的 hash。

更新任何視覺資產時，必須同步檢查：layer-first 目錄、依賴方向、Port／Repository ownership、production verification wording、SVG 無障礙 metadata，以及 `tests/architecture` 的語意與完整性守門。
