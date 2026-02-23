# Chat File Attachments Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add arbitrary file attachments to chat (icon picker + drag/drop) using the existing upload endpoint, with a shared backend upload size flag `-max-upload-mb` defaulting to 25 MB.

**Architecture:** Reuse the current upload storage/auth flow and chat message delivery (HTTP + WebSocket). Backend changes are limited to upload configuration and file-type acceptance; frontend extends the existing pending image/upload flow into a generic attachment flow with image/file rendering branches.

**Tech Stack:** Go, Gorilla Mux, BoltDB uploads, React, Vite, existing Axios API client.

---

### Task 1: Backend TDD for Upload Config and Any-Type Acceptance

**Files:**
- Modify: `handlers.go`
- Modify/Create: upload handler tests in root package (e.g. `handlers_upload_test.go`)
- Modify: `main.go`

**Step 1: Write failing tests**
- Upload handler accepts non-image file content type
- Upload handler enforces configured max upload bytes (using test-configurable value)

**Step 2: Run tests to verify RED**
- Run: `go test ./... -run Upload -v`
- Expected: FAIL for missing config hook / image-only restriction.

### Task 2: Backend Implementation (GREEN)

**Files:**
- Modify: `main.go`
- Modify: `handlers.go`

**Step 1: Add flag**
- `-max-upload-mb` default `25`, validate `> 0`
- store runtime bytes in package-level variable/config used by upload handler

**Step 2: Update upload handler**
- Replace hardcoded `8<<20` limits with configured bytes
- Remove image-only MIME restriction
- Keep auth/team checks and response shape unchanged

**Step 3: Run tests**
- Run: `go test ./... -run Upload -v`
- Expected: PASS.

### Task 3: Frontend Attachment Composer UI (Picker + Drag/Drop)

**Files:**
- Modify: `frontend/src/pages/Chat.jsx`

**Step 1: Generalize pending attachment state**
- Replace image-only pending state with file-capable state
- Keep preview for images

**Step 2: Add attachment icon + hidden file input**
- Select any file
- Reuse same send/upload flow

**Step 3: Add drag/drop UX**
- Drag-over highlight on composer area
- Drop selects pending file

### Task 4: Frontend Upload/Send Flow + File Rendering

**Files:**
- Modify: `frontend/src/pages/Chat.jsx`

**Step 1: Upload/send logic**
- Determine message type from file MIME (`image/*` vs other => `file`)
- Send via WS when connected, HTTP fallback otherwise

**Step 2: Attachment rendering**
- Preserve image rendering
- Add file attachment card/link rendering (`type === 'file'`)
- Reuse blob cache for protected downloads
- Track attachment metadata (filename/contentType/size) when loading blobs for file cards

**Step 3: Oversize and upload error messaging**
- Surface backend max-size rejection in UI (basic error text is enough)

### Task 5: Verification + Cleanup

**Files:**
- Modify only if needed for cleanup

**Step 1: Backend verification**
- Run: `go test ./...`

**Step 2: Frontend verification**
- Run: `npm run build`

**Step 3: Cleanup**
- Remove any temporary patch backup files if created

**Step 4: Commit (optional)**
```bash
git add main.go handlers.go frontend/src/pages/Chat.jsx docs/plans/2026-02-22-chat-file-attachments-design.md docs/plans/2026-02-22-chat-file-attachments-plan.md
git commit -m "feat: add file attachments to chat"
```
