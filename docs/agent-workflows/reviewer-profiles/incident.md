# Reviewer Profile: incident

## Activate when

Cloud Run, Cloud Tasks, Firestore, LINE, OAuth, queue backlog, retry storm, secret rollover, or authentication failures are reported.

## Required checks

- Build a timestamped timeline from revisions, request IDs, queue state, errors, and changes.
- Separate symptoms, observations, hypotheses, root cause, and confidence.
- Check retry amplification, idempotency, inbox/outbox, OIDC, IAM, secrets, and OAuth state.
- Propose the smallest reversible mitigation with an authorization gate.
- Define follow-up tests, monitoring, rollback, and postmortem evidence.

## Evidence and restrictions

Every conclusion identifies its source and timestamp. Do not resend real messages, change traffic, rollback, alter IAM, rotate secrets, deploy, or inspect sensitive data without authorization. Missing live evidence is `Not Verified`.
