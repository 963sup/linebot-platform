# SDK verification evidence

## Minimum proof for new SDK work

- `go.mod`: exact module and resolved version (never infer from README or catalog).
- `go list` / `go doc` / gopls: exact import, exported symbol and method signature in the pinned dependency.
- Official upstream documentation: request semantics, auth method, quota, cancellation, retry/idempotency, account or region restrictions.
- `internal/infrastructure/<provider>`: typed client construction and explicit resource ownership; respect existing client-side retry and transport behavior.
- `internal/adapter/outbound/<provider>`: native SDK calls and conversion to application-owned types; business use cases must not import provider SDKs.
- `internal/bootstrap`: only initialize enabled services for that process, with cleanup when construction partially fails.
- Tests: failure and cancellation cases, safe fake/contract tests, `go run ./cmd/codexcheck`, `make check`.

## Status definitions

- **declared**: direct requirement exists in `go.mod`; client and permissions may not exist. Optional build-tagged imports keep the module pinned.
- **integrated**: runtime source exists for a service and its outbound adapter. The validator checks source presence but does not prove live credentials or end-to-end correctness. For utilities/tooling, actual usage is reviewed in code.
- **production-ready** is never a catalog status: it requires separate reliability, deployment, IAM, monitoring and authorized staging evidence.

## Fallback when tools are unavailable

1. Try local `go list`, `go doc` and module source first.
2. If the SDK is not downloaded, request an allowed Go proxy/SDK download in the development environment; do not substitute a similar newer SDK.
3. If remote documentation is unavailable, stop short of claiming verified semantics. A compiling unit test does not validate live API permissions.

## Delivery report template

State changed files, the SDK/module version, method references used, commands and pass/fail results, security/idempotency implications, and any steps blocked by missing credentials, network, or staging approval.
