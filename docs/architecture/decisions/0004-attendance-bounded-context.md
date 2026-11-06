# ADR 0004: Attendance bounded context

- Status: Accepted
- Date: 2026-11-05
- Strategic state: Implemented

## Context

The MINI App location flow is a delivery mechanism for workforce clock-in and clock-out, not merely location validation. Attendance has product-owned language, a consistency boundary per participant and shift assignment, deterministic duplicate semantics, reviewable anomalies, and an auditable correction lifecycle. These rules are independent from LINE, HTTP, Firestore, and browser geolocation types.

ADR 0003 authorized tactical models for five candidate Contexts only. It did not authorize Attendance, so this decision independently validates and implements the `attendance` Context under the `workforce-attendance` Supporting Subdomain.

## Decision

Attendance owns one Aggregate per `ParticipantID` and `ShiftAssignmentID`. The Aggregate owns at most one effective clock-in and one effective clock-out. Server timestamps are authoritative; clock-out requires and follows clock-in. The hard submission window rejects submissions outside its bounds. Schedule and location anomalies are accepted but marked `required` for review.

Review state is `clear`, `required`, `confirmed`, `corrected`, or `voided`. Confirm, correct, and void actions append immutable reviewer claim, reason, state, provenance, server time, and before/after coordinate-free decision state. A manual correction must retain the Punch evidence reference and preserve effective clock-in/clock-out ordering. One assignment/kind pair is the idempotency slot: after its first successful submission, retries return that existing Punch without reinterpreting a later payload as a second business submission.

The Domain repository contract requires atomic compare-and-swap persistence of an Attendance Aggregate together with separately encrypted, expiring raw location evidence and punch-ID/submission uniqueness in the same transaction. The long-lived Punch retains only distance, accuracy, anomaly reasons, evidence reference, and policy version, so expiry of raw latitude/longitude does not prevent aggregate restoration. Review-only saves append history without creating new evidence. Provider documents and transport DTOs remain Adapter concerns. No Integration Event or outbox contract is introduced by this decision.

An administrator creates an employee record identified by `EmployeeID` and a separate enrollment invitation that expires within 24 hours. Atomic invitation consumption binds the verified `ParticipantID`. A retry by the same participant is idempotent; reuse by another participant, or an attempt to bind a participant already owned by another employee, conflicts.

Attendance does not own:

- HR, payroll, employment, or employee master-data lifecycle.
- LINE authentication or token verification.
- physical presence proof for a person or device at the submitted coordinates.
- notifications or message delivery.

## Consequences

- `internal/domain/attendance` is pure Go and is the implemented strategic model for this boundary.
- Participant, Workplace, ShiftInstance, ShiftAssignment, Punch, LocationAssessment, review history, and repository contracts use provider-independent types.
- Browser location and identity verification must be translated by Application and Adapters before invoking Attendance.
- Persistence and Application wiring are separate delivery work; repository implementation, deployment, and external environment behavior remain `Not Verified` until evidenced.
- Raw encrypted coordinate evidence has a 90 days retention target while the coordinate-free Attendance record has a five years retention target; expiry of the former cannot make the latter unrestorable.
- Adding reporting, payroll export, notifications, or cross-Context publication requires an explicit contract and, for reliable publication, a transactional outbox or equivalent mechanism.
