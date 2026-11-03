# Remaining LINE Messaging API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate all 17 remaining LINE Messaging API endpoints with consistent operator-only application boundaries.

**Architecture:** Keep SDK types inside LINE adapters. Read endpoints use `Reader` and `Inspector`; mutations use `Manager` and explicit command confirmation. Binary payloads use `BinaryContent`, remain bounded to 20 MiB, and only write to requested local files.

**Tech Stack:** Go 1.27, line-bot-sdk-go/v8, httptest contract tests.

**Spec:** `docs/superpowers/specs/2026-11-03-remaining-line-messaging-api-design.md`

## Global Constraints

- Public webhook exposes none of these operator endpoints.
- Writes require `--apply --confirm ACTION`; reads require `--read`.
- Eligibility endpoints also require `--eligibility-reviewed`.
- Update inventory only with adapter contract and application/CLI route tests.

## Review Focus

- Binary responses above 20 MiB fail without retaining partial data (Task 1).
- CLI never writes binary data absent an explicit output file (Task 1).
- Action confirmation must match the selected action (Tasks 2–4).
- Token bodies must not be command-line arguments (Task 2).
- Eligibility operations reject absent acknowledgement (Task 4).

### Task 1: Blob content and Rich Menu media CLI

**Files:** ports/adapters already created; add `contacts/content_test.go`, `cli/content.go`, extend `cli/manage_menu.go`, update bootstrap and inventory.

- [ ] Write failing inspector and CLI tests for content/preview/transcoding output, Rich Menu image download/upload, and explicit output/apply flags.
- [ ] Implement `ops content --kind content|preview|transcoding --message-id ID --read [--out FILE]` and Rich Menu image actions using `ContentInspector` and `RichMenuImageManager`.
- [ ] Add Blob endpoint contract assertions and mark five inventory endpoints integrated.
- [ ] Run focused Go tests; commit.

### Task 2: Conversation actions and account linking

**Files:** create outbound `conversation_actions.go`, adapter `conversation_actions.go`, use cases, CLI command and tests.

- [ ] Write failing contract and command-validation tests for leave group/room, loading, both mark-read APIs, and link-token output.
- [ ] Implement `ConversationActionManager` and `AccountLinkManager` with SDK calls and sanitized errors.
- [ ] Implement confirmed CLI actions; issue link tokens only to an explicit 0600 output file.
- [ ] Mark six inventory endpoints integrated; run focused tests; commit.

### Task 3: Coupon operations

**Files:** extend coupon port/use case/adapter and coupon CLI tests.

- [ ] Write failing JSON-request and confirmation tests.
- [ ] Implement create/close with JSON file input and `--eligibility-reviewed`.
- [ ] Mark two endpoints integrated; run tests; commit.

### Task 4: Phone Number Platform operations

**Files:** create eligibility port/use case/adapter and CLI tests.

- [ ] Write failing tests for acknowledgement, JSON requests, and statistics reads.
- [ ] Implement two PNP sends and two PNP statistics queries through official SDK calls.
- [ ] Mark four endpoints integrated, preserving `required_eligibility_review`; run full suite; commit and push `main`.
