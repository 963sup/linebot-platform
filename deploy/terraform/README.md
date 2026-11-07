# Terraform deployment prerequisites

This module is an opt-in skeleton for an authorized project. `terraform plan`
and `terraform apply` are never run by repository CI. It creates five distinct
service accounts, a bounded Cloud Tasks queue, Firestore composite indexes and
TTL field policies, least-privilege IAM bindings and optional monitoring
resources. The MINI App account receives `roles/datastore.user` and access only
to Attendance Firestore; it receives no Secret Manager, Messaging or
Cloud Tasks role. Existing Messaging secrets, the default Firestore database and
Cloud Run service names are referenced rather than created.

Before applying, an owner must review the plan, confirm billing and APIs, import
any matching resources, approve alert thresholds/channels and provide a remote
state policy. Keep `enable_monitoring=false` until notification channels and
staging fault-injection evidence are approved. Firestore index and TTL resources
must be imported when they already exist; this module never creates or replaces
the default database. `deploy/firestore/indexes.json` remains the runtime query
contract and must stay aligned with these resources.

No tfvars, state, credentials, project secrets or OAuth tokens belong in Git.
