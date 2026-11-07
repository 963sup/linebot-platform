# LINE MINI App Attendance

The Attendance MINI App lets an enrolled LINE-authenticated participant inspect today's attendance state and submit one clock-in and one clock-out with the device's current GPS fix. Its normative HTTP contract is [the MINI App OpenAPI document](../../api/openapi/miniapp.yaml).

When the webhook role has `LINE_MINI_APP_LIFF_URL` configured, the trimmed text command `打卡` returns a Flex button that opens the MINI App. The conversation is only a distribution channel and never becomes an Attendance identity or authorization scope.

## Boundary and ownership

- Attendance owns Participant enrollment and one `AttendanceDay` per Participant and server-derived `Workday`.
- Application verifies the LINE principal, derives server time and Asia/Taipei workday, and coordinates transactional persistence.
- The HTTP adapter owns strict JSON, status mapping, LIFF assets and exact-origin validation; it does not decide attendance invariants.
- The LINE adapter verifies the LIFF ID token with `LINE_MINI_APP_CHANNEL_ID` as audience. Browser identifiers, query parameters, cookies and page possession are not authenticated principals.
- Browser Geolocation is user-supplied data, not proof of physical presence. The MVP records latitude, longitude and accuracy as private Firestore data and relies on Google Cloud encryption at rest plus a dedicated runtime identity; it has no Attendance Secret Manager keyring. The current project-scoped `roles/datastore.user` binding is broader than collection-level least privilege, so Production remains `Not Verified` until Attendance uses an isolated database or equivalently scoped access boundary.

## Routes

| Method | Path | Authentication | Purpose |
|---|---|---|---|
| `GET` | `/miniapp` | none | Redirect to `/miniapp/` |
| `GET` | `/miniapp/` | none | Attendance UI |
| `GET` | `/miniapp/config.json` | none | Public LIFF ID |
| `GET` | `/miniapp/assets/*` | none | Embedded static assets |
| `POST` | `/miniapp/api/v1/attendance/enrollments` | LIFF ID-token Bearer | Redeem an enrollment code |
| `GET` | `/miniapp/api/v1/attendance/today` | LIFF ID-token Bearer | Read today's clock-in/out state and allowed action |
| `POST` | `/miniapp/api/v1/attendance/punches` | LIFF ID-token Bearer | Submit `clock_in` or `clock_out` plus GPS |

Every Attendance API route requires exactly one `Authorization: Bearer <LIFF ID token>`. Mutating routes additionally require the exact origin derived from `LINE_MINI_APP_ENDPOINT_URL`. The same-origin GET may omit `Origin` only when `Sec-Fetch-Site` is exactly `same-origin`. Responses emit no CORS permission and use `Cache-Control: no-store`. POST JSON rejects unknown fields, trailing values and oversized bodies.

Enrollment returns `201` when created and `200` for an idempotent replay. Punch returns `201` when recorded and `200` when the same participant/workday/kind was already committed. The browser cannot choose Participant, Workday, recorded time or persistence identifiers. Clock-out requires an existing clock-in and must follow it.

Punch retry idempotency is scoped to the server-derived Asia/Taipei Workday. A response-lost retry that crosses midnight targets the next Workday and is not a replay of the previous day's command.

## UI and privacy

The page initializes LIFF, performs LINE Login when needed and obtains `liff.getIDToken()` only after initialization. The token stays in memory. An unenrolled participant sees the enrollment form; an enrolled participant sees today's server-provided status and only the currently allowed clock action.

Geolocation is requested once from a clock button gesture with a ten-second timeout. The UI uses no local/session storage, location watch, maps or analytics. API responses do not expose coordinates or internal record identifiers. Coordinates, ID tokens, participant IDs and enrollment codes must not enter URLs, logs, traces, analytics or error text.

`attendance_days_v2` currently has no approved TTL. Because it contains precise coordinates, Production must not be promoted until a retention period and deletion mechanism are approved and implemented.

## Operator workflow

`ops attendance employee invite` creates an employee invitation and returns its one-time enrollment code; `employee deactivate` disables an employee. Both require explicit `--project`, `--environment` and `--approve-write`. Enrollment codes expire within 24 hours and must be delivered through an approved private channel. Workplace, Shift, Assignment and Review commands are intentionally absent from this MVP.

## Verification limits

Repository tests verify routing, parsing, headers, server-time invariants, idempotency and persistence contracts. They do not prove LINE Console configuration, real ID-token verification, device permission behavior, Cloud Run origin, Firestore IAM or live concurrency. Those remain `Not Verified` until an authorized no-traffic candidate passes staging smoke and is separately promoted.
