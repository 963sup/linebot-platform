# LINE 功能覆蓋矩陣（尚未實作）

| 功能組 | 預計範圍 | 狀態 |
|---|---|---|
| Webhook | 簽章、destination、重送、edit／unsend、message／follow／join 等已公開事件 | Planned |
| Messaging | Reply、Push、Broadcast、Multicast、Narrowcast | Planned |
| Message contents | Text、Flex、Template、Image、Video、Audio、Sticker、Location 等 | Planned |
| Chat / User | Profile、followers、group／room、content retrieval | Planned |
| Rich Menu | 建立、綁定、圖片、alias、batch | Planned |
| Audience | 建立、上傳、查詢、刪除、分享 | Planned |
| Insights | 發送、互動、followers、demographics | Planned |
| Coupon / Membership | 視 LINE 官方帳號所在地區與資格啟用 | Planned / Restricted |
| Channel / account link | Token、bot info、webhook settings、account link | Planned |

此表只是能力分組，**不代表已逐 endpoint 驗證或實作**。正式追蹤應依當下官方 API reference 建立 `endpoint-inventory.yaml`，記錄各 endpoint 的權限條件、SDK 版本、測試及完成狀態。
