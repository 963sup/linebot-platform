# Adapter Layer

Inbound adapters 對接 HTTP webhook、Tasks、CLI；Outbound adapters 對接 LINE、OpenAI、Google Gen AI（Gemini）、Google Cloud 與 Google API。將供應商 API 轉成 application ports 規定的資料結構。Google Gen AI 的 outbound adapter 預留於 `outbound/genai/`；目前無已實作 API。
