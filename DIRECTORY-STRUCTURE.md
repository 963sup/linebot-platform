# Directory Structure

```text
linebot-platform/
├── cmd/
│   ├── server/main.go
│   └── worker/main.go
├── internal/
│   ├── domain/
│   │   ├── identity/
│   │   │   ├── entities/
│   │   │   ├── aggregates/
│   │   │   ├── valueobjects/
│   │   │   ├── policies/
│   │   │   ├── specifications/
│   │   │   ├── factories/
│   │   │   ├── repositories/
│   │   │   └── events/
│   │   ├── conversation/
│   │   ├── calendar/
│   │   ├── task/
│   │   ├── form/
│   │   └── audience/
│   ├── application/
│   │   ├── commands/
│   │   ├── queries/
│   │   ├── usecases/
│   │   └── ports/
│   │       ├── inbound/
│   │       └── outbound/
│   ├── adapter/
│   │   ├── inbound/
│   │   │   ├── line/
│   │   │   ├── liff/
│   │   │   ├── http/
│   │   │   └── cloudtasks/
│   │   └── outbound/
│   │       ├── line/
│   │       ├── google/
│   │       ├── firestore/
│   │       └── cloudtasks/
│   ├── infrastructure/
│   │   ├── line/{auth,client,webhook}/
│   │   ├── google/{auth,api}/
│   │   ├── firestore/
│   │   ├── cloudtasks/
│   │   ├── config/
│   │   ├── httpclient/
│   │   └── observability/
│   └── bootstrap/
├── web/liff/
├── .agents/skills/
├── .codex/agents/
├── .github/workflows/
├── .vscode/
├── docs/
│   ├── architecture/
│   ├── integrations/
│   ├── sdk-catalog.md
│   └── deployment.md
├── scripts/
├── tests/{integration,architecture}/
├── AGENTS.md
├── DESIGN.md
├── Dockerfile
├── Makefile
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Responsibilities

- `cmd/`: executable entry points and composition roots.
- `internal/domain/`: domain models, invariants, policies, specifications, factories, repository contracts, and events. It must not depend on SDKs or infrastructure.
- `internal/application/`: use cases and consumer-owned ports. It coordinates domain behavior without depending on concrete adapters.
- `internal/adapter/inbound/`: translates external requests and events into application inputs.
- `internal/adapter/outbound/`: implements application ports and translates application data to external systems.
- `internal/infrastructure/`: SDK clients, transports, authentication, persistence clients, and observability primitives.
- `internal/bootstrap/`: shared dependency assembly only when multiple entry points genuinely reuse it.
- `tests/`: architecture, contract, integration, and end-to-end verification.
- `docs/`: architecture, integration, SDK, and deployment documentation.

This document describes the intended target structure. It does not claim that every listed feature or file is implemented. The actual implementation status is defined by the Go source, tests, and deployment configuration.

