# Codex Cloud Environment

The Cloud Install script prepares pinned local tools; the Start skill checks readiness. Run `bash scripts/cloud-bootstrap.sh` for idempotent preparation and `bash scripts/cloud-verify.sh` for evidence. `cloud-repair.sh` is limited to local pinned tools, PATH, caches and approved Skills. It never performs OAuth, IAM, deployment, network expansion or cloud data writes.

Publish/Republish is an environment lifecycle action and must be verified separately from repository tests. New tasks use the published environment version; documentation or configuration alone is not runtime evidence.

