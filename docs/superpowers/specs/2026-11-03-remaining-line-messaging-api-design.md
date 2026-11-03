# Remaining LINE Messaging API foundation

## Intent

Complete the 17 remaining Messaging API endpoints with the repository's
existing hexagonal boundary: explicit outbound ports, application use cases,
official SDK adapters, and operator-only CLI commands. No endpoint is exposed
through the public webhook merely because it is available in the SDK.

## Shape

| Module | Endpoints | Port / use case | Operator boundary |
|---|---|---|---|
| Blob content | getMessageContent, getMessageContentPreview, getMessageContentTranscodingByMessageId | `MessageContentReader` / `contacts.ContentInspector` | `ops content --read --kind … --message-id …`; binary writes only to an explicit output file |
| Rich Menu media | getRichMenuImage, setRichMenuImage | `RichMenuImageReader`, `RichMenuImageManager` / `engagement.RichMenuImageManager` | download requires `--read --out`; upload requires `--apply --confirm upload-image` |
| Conversation actions | leaveGroup, leaveRoom, showLoadingAnimation, markMessagesAsReadByToken, markMessagesAsRead | `ConversationActionManager` / `messaging.ConversationActions` | every remote write requires `--apply --confirm ACTION`; token bodies come from a JSON file, never a CLI argument |
| Account link | issueLinkToken | `AccountLinkManager` / `contacts.AccountLinker` | `--apply --confirm issue-link-token`; token writes only to an explicit 0600 output file |
| Eligibility-gated operations | pushMessagesByPhone, pushTemplatedMessagesByPhone, getPNPMessageStatistics, getPNPTemplatedMessageStatistics, createCoupon, closeCoupon | `EligibilityOperations` / `engagement.EligibilityOperations` | command requires `--eligibility-reviewed` in addition to normal read/apply confirmation; inventory remains `required_eligibility_review` until a real channel is qualified |

## Shared rules

- SDK request shapes are supplied by JSON files when they exceed the existing
  application message model. The application validates JSON syntax and required
  local identifiers; the official SDK owns discriminator decoding.
- Binary data is bounded to 20 MiB and never printed to stdout or logged.
- Every adapter call uses `WithHttpInfo` and `lineWriteError` / `reportResult`
  so response bodies and credentials cannot surface in errors.
- Read-only queries use `Reader`/`Inspector`; writes use `Manager`/`Command`.
- Endpoint inventory changes to `integrated_offline` only after the adapter
  contract test and the application/CLI route both exist.

## Delivery order

1. Finish Blob CLI wiring and Rich Menu media routes.
2. Add conversation actions and account linking.
3. Add eligibility-gated PNP and coupon operations with their extra operator
   acknowledgement.
4. Update the inventory and run the full Go suite once every endpoint has an
   offline contract test.
