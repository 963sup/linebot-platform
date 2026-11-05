# Codex Cloud Environment

Codex Cloud scripts are isolated under `scripts/codex-cloud/`. The environment lifecycle order is `install.sh`, `start.sh`, then `verify.sh`; `repair.sh` is an explicit recovery path. All phases source the single PATH contract in `lib/paths.sh`.

`install.sh` prepares pinned local tools without authentication. `start.sh` intentionally discards local checkout changes and synchronizes exactly to `origin/main` before verification. `repair.sh` is limited to pinned local tools, PATH, caches and approved Skills. These scripts never perform OAuth, IAM, deployment, network expansion or cloud data writes.

Publish/Republish is an environment lifecycle action and must be verified separately from repository tests. New tasks use the published environment version; documentation or configuration alone is not runtime evidence.

