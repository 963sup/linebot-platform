# Codex Cloud environment

The normal workflow has three files:

- `mise.toml` owns tool versions and tasks.
- `bootstrap.sh` installs checksum-pinned mise, then runs mise. All mutable tool state stays under `/workspace`.
- `.codex/config.toml` declares MCP commands and tool permissions for a host that loads trusted project configuration.

Run once during environment setup, from the checkout:

```bash
bash /workspace/linebot-platform/bootstrap.sh setup
```

This installs Go, Node, gopls and Repomix, then downloads and verifies Go modules.
MCP startup uses `bootstrap.sh exec -- …` with automatic installation disabled;
startup therefore fails clearly if setup was not completed. It does not download
tools, change Git branches, clean files, or modify user-level Codex settings.
There is no daemon to preserve between tasks: the MCP host starts the configured
stdio processes when it opens the session. The checkout must be mounted at
`/workspace/linebot-platform`, matching `.codex/config.toml`.

## What counts as working

Installation, an MCP handshake, and a tool exposed to the assistant are separate
checks. In a new task, the host must load the project configuration, list Repomix
in the assistant's available tools, and successfully call `pack_codebase` on a
small scope such as `mise.toml,bootstrap.sh`. `required = true` makes a Repomix
startup failure visible on a host that honors this option; it cannot make a host
load ignored configuration. Neither a setup script nor `codex mcp list` in a
separate CLI process can prove that this chat has those tools.

This session did not expose Repomix MCP tools. Do not report Cloud MCP readiness
until that fresh-session tool call succeeds. Publishing the environment snapshot
is a separate user action and is not performed by these files.

## Optional integrations

Context7 and OpenAI Docs are public remote MCPs. GitHub needs the host-provided
`GITHUB_MCP_TOKEN`. Google, LINE, Terraform and LINE OpenAPI integrations are
configured but disabled until their credentials, runtime or schema are prepared.
Firestore remains disabled. Their existing tool allowlists and approval policies
still apply when enabled.

`bash bootstrap.sh run install-integrations` explicitly installs the optional
mise-managed gcloud and npm tools. It gives the gcloud installer a writable local
configuration directory without changing the credentials used by later commands.
Terraform still needs Docker and its pinned image; LINE OpenAPI still needs the
reviewed schema at the configured path. Enabling either entry is a separate,
explicit configuration change after those prerequisites are satisfied.

For local checks use `bash bootstrap.sh run mcp-check`. This validates repository
configuration and local tools; it does not certify host registration or remote
credentials. The smoke probe requires Python 3.11+ and uses only its standard library.
Run its regression tests with `python3 -m unittest discover -s scripts/codex-cloud -p "test_*.py"`.
Legacy lifecycle wrappers and their unused helpers have been removed; use bootstrap
and mise tasks directly. No startup entrypoint resets or cleans the checkout.
