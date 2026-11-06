# Codex Cloud Environment

## Migration status

The mise/bootstrap migration is in progress, not a verified reusable environment.
Shell syntax and the codexcheck unit tests pass. Full clean installation and
`make check` remain unverified. The gcloud installer currently fails when it
inherits the platform's read-only CLOUDSDK_CONFIG. Local MCP commands still
contain legacy absolute paths and must be aligned with the dynamic tool root.
The setup task currently prepares Go modules only. Do not run the legacy
`start.sh` against local work: it still resets and cleans the checkout.
The empty `.agents/plugins/marketplace.json` is a placeholder, not a validated
plugin marketplace manifest.

Codex Cloud scripts are isolated under `scripts/codex-cloud/`. Every entry point first loads the immutable PATH contract from `lib/paths.sh` and runs the read-only `preflight.sh`. The environment lifecycle then runs `install.sh`, `start.sh`, and `verify.sh`; `repair.sh` is an explicit recovery path.

`install.sh` installs the checksum-pinned mise binary, uses repo `mise.toml` to install Go, Node.js and gopls with secret-free local bootstrap defaults, prepares the remaining pinned local tools without authentication, and never writes Host-owned Codex configuration. All mise data and mutable caches stay under `/workspace`. Project trust and loading the repository `.codex/config.toml` belong to the Cloud Environment Host and are verified from a fresh task. `start.sh` intentionally discards local checkout changes and synchronizes exactly to `origin/main` before verification. `repair.sh` is limited to pinned local tools, PATH, caches and approved Skills. These scripts never perform OAuth, IAM, deployment, network expansion or cloud data writes.

Publish/Republish is an environment lifecycle action and must be verified separately from repository tests. New tasks use the published environment version; documentation or configuration alone is not runtime evidence.

