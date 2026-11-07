# ADR 0006: Workday Attendance MVP

- Status: Accepted
- Date: 2026-11-07
- Supersedes: ADR 0004

## Context

The first Attendance delivery must prove one complete LINE MINI App journey:
an enrolled LINE participant records a GPS-backed clock-in and clock-out, using
server time, and can safely retry. The former model made workplace policy,
shifts, assignments, review workflows, evidence encryption and key rotation
prerequisites for this basic journey without deployed evidence that those
policies are currently required.

## Decision

Attendance owns one `AttendanceDay` Aggregate per `ParticipantID` and
`Workday`. For the MVP, `Workday` is derived exclusively from server time in the
fixed `Asia/Taipei` timezone. A client cannot choose the workday or recorded
time.

An AttendanceDay owns at most one clock-in and one clock-out. Clock-out requires
and follows clock-in. The idempotency slot is the participant, workday and punch
kind; after the first committed punch, retries return that Punch without
reinterpreting later GPS input. Persistence uses compare-and-swap revision
checks.

Each Punch records the submitted latitude, longitude and accuracy. These values
are private persistence data and are not returned in MINI App responses. The MVP
relies on Google Cloud and Firestore encryption at rest and least-privilege IAM;
it does not implement application-level coordinate encryption, evidence
decryption or a keyring.

Enrollment invitations still bind an Attendance-owned Employee projection to a
verified opaque LINE ParticipantID. Attendance does not own LINE authentication,
HR or payroll lifecycle, workplace policy, scheduling, physical-presence proof,
review workflow or message delivery.

## Consequences

- Workplace, ShiftInstance, ShiftAssignment, location assessment, review state,
  evidence retention and key-version configuration leave the current model.
- The MINI App exposes today's status and clock-in/out actions without a shift
  selector.
- Persistence uses a new workday schema; legacy Attendance collections are not
  silently migrated or deleted.
- Workplace rules, scheduling, review, coordinate retention or application-level
  encryption require new evidence and a later explicit decision.
