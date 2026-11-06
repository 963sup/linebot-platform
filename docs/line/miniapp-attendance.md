# LINE MINI App Attendance

The Attendance MINI App lets a LINE-authenticated participant enroll, view only assigned current shift/workplace options, and submit a clock-in or clock-out. Its normative HTTP contract is [the MINI App OpenAPI document](../../api/openapi/miniapp.yaml).

## Boundary and ownership

- Attendance Domain owns participants, workplaces, shift instances, assignments, punches, review state and their invariants.
- The Attendance Application use case verifies the principal, redeems enrollment, lists participant-scoped current options and records idempotent punches through consumer-owned capabilities and Domain repository contracts.
- The inbound adapter owns HTTP, strict JSON mapping, LIFF UI and safe error/status mapping. It does not decide assignment, time-window, location-review or replay rules.
- The LINE outbound adapter verifies the LIFF ID token. Browser identifiers, query parameters, cookies and page possession are not authenticated principals.
- Browser geolocation is user-supplied evidence, not proof of physical presence. Domain/Application policy decides whether evidence is clear or requires review.

## Routes

| Method | Path | Authentication | Purpose |
|---|---|---|---|
| `GET` | `/miniapp` | none | Redirect to `/miniapp/` |
| `GET` | `/miniapp/` | none | Attendance UI |
| `GET` | `/miniapp/config.json` | none | Public LIFF ID |
| `GET` | `/miniapp/assets/app.css` | none | Stylesheet |
| `GET` | `/miniapp/assets/app.js` | none | Browser application |
| `POST` | `/miniapp/api/v1/attendance/enrollments` | LIFF ID-token Bearer | Redeem an enrollment code |
| `GET` | `/miniapp/api/v1/attendance/options` | LIFF ID-token Bearer | List assigned current shift/workplace options |
| `POST` | `/miniapp/api/v1/attendance/punches` | LIFF ID-token Bearer | Submit clock-in or clock-out evidence |

All Attendance API routes require one `Authorization: Bearer <LIFF ID token>`. Mutating routes also require one exact configured `Origin`. Because browsers may omit `Origin` on a same-origin GET, the options route accepts that omission only when `Sec-Fetch-Site` is exactly `same-origin`; when `Origin` is present it must match exactly. They emit no CORS headers and all responses use `Cache-Control: no-store`. POST bodies are strict JSON, reject unknown/trailing values and are limited to 4096 bytes.

Enrollment returns `201` when created and `200` for an idempotent existing enrollment. Punch submission returns `201` when recorded and `200` for a replay. Options contain only the authenticated participant's assigned, currently applicable shift/workplace choices; an arbitrary workplace or participant supplied by the browser is never accepted.

Errors use `{ "error": { "code": "...", "message": "..." } }`. Codes and messages are safe for presentation and never contain ID tokens, provider subjects, enrollment codes or coordinates.

## UI and privacy

The page initializes LIFF, redirects unauthenticated users to LINE Login using the fixed `/miniapp/` URL, and obtains the ID token through `liff.getIDToken()` only after initialization. It holds the token only in memory. When enrollment is required it displays the enrollment-code form. Once enrolled it renders the server-provided assigned shift/workplace options and only the allowed clock-in/out actions.

Geolocation is requested once, directly from a clock-in/out button gesture, with `enableHighAccuracy: false`, `maximumAge: 0` and a ten-second timeout. The UI uses no local/session storage, location watch, maps or analytics. Attendance responses never return latitude, longitude or accuracy; they expose application identifiers, server record time, outcome and review state/reasons. The LINE-derived `ParticipantID`, generated `ShiftAssignmentID`, Punch ID and evidence references are opaque; operator-managed Shift IDs are application references and must not contain personal data. In particular, punch responses never return latitude or any other coordinate field.

Coordinates and identity material must not appear in logs, traces, error text or browser storage. Persistence retention, reviewer access and deletion remain Domain/deployment policy, not an HTTP adapter decision.

## Operator workflow

`ops attendance` manages the server-owned data that the browser is never allowed to choose: employee invitations/deactivation, workplace definitions, shift instances, employee assignments, and review actions. Every command requires explicit `--project` and `--environment`; writes additionally require `--approve-write`. Enrollment codes expire within 24 hours, are returned once by `employee invite`, and must be delivered through an approved private channel.

Workplace upsert requires an operator-facing `--display-name`, and shift creation requires an operator-facing `--label` plus explicit clock-in, clock-out, and hard submission windows. The Mini App combines those labels with the shift time range in its assigned-options dropdown. IDs remain opaque persistence and command references rather than user-facing names; the separate windows determine the independent time-anomaly flag.

Review supports `list`, `show`, `confirm`, `correct`, and `void`. `review show` returns coordinate-free evidence by default. Raw coordinates are decrypted only with `--include-location`, `--approve-write`, and non-empty `--reviewer` and `--reason`; the reviewer value is an operator-supplied claim, while Cloud Audit Logs remain the authoritative workload identity evidence. The Firestore adapter writes a five-year immutable access record before reading the 90-day evidence. Manual correction retains the Punch evidence reference and appends provenance plus before/after coordinate-free decision state; it does not create new raw evidence or claim to rerun system location assessment.

## Verification limits

Repository tests can verify routing, parsing, headers, status mapping and browser source constraints. They do not prove LIFF configuration, LINE identity verification, Firestore IAM, assignment data, public HTTPS origin, browser permission behavior or deployed retention controls. Those remain `Not Verified` until authorized staging evidence exists.
