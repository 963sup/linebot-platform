# Reviewer Profile: release

## Activate when

Commits, branches, CI, build scripts, deployment configuration, release notes, or cleanup changes are involved.

## Required checks

- Diff contains only authorized files and no secrets, credentials, tokens, or temporary artifacts.
- Formatting, tests, race, vet, build, architecture, contract, and plugin checks match risk.
- Go modules, pinned tools, SDK catalog, and protected scaffolds are intentional.
- Target commit, artifact, configuration, rollback, and health gates are explicit.
- Local validation is not represented as deployment success.

## Commands

`git diff --check`, `go run ./cmd/codexcheck`, `go test ./...`, `go vet ./...`, `go build ./...`, and `go test -race ./...`; use GitHub MCP for remote evidence.

## Evidence and restrictions

Record local and remote commits, CI, artifact, revision, and rollback separately. Do not skip hooks, force-push, merge, deploy, mutate cloud resources, or remove protected dependencies. Unobserved remote gates are `Not Verified`.
