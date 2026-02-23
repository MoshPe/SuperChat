# File Attachments in Chat Design

**Date:** 2026-02-22

## Scope and Behavior

Add arbitrary file attachments (not just images) to chat using the existing upload infrastructure.

User interactions:

- Click an attachment icon to choose a file
- Drag and drop a file into the chat input area
- Upload through the existing protected `/api/upload` flow
- Send a chat message referencing the uploaded file

Rendering:

- Existing image preview behavior remains for image attachments
- Non-image attachments render as file cards/links (filename + size if available + open/download)

Backend behavior:

- Existing `/api/upload` accepts any file type
- New backend flag `-max-upload-mb` (default `25`) applies to both images and files
- Existing auth/team checks remain unchanged

## Backend Changes

### Config

Add `-max-upload-mb` flag (default `25`) and validate `> 0`. Convert to bytes for runtime use by `handleUpload`.

### Upload handler

Replace hardcoded `8<<20` upload limits in `handleUpload` with the configured byte limit for:

- `http.MaxBytesReader(...)`
- `ParseMultipartForm(...)`

Remove image-only MIME restrictions so any type is accepted. Preserve existing response shape and auth checks.

### Metadata

Reuse existing `UploadMeta` fields (`filename`, `content_type`) and existing storage. No schema migration.

### Chat message type

Use message `type` to distinguish rendering:

- `image` for images
- `file` for non-images

## Frontend Changes

### Composer UX

- Add attachment icon button with hidden file input
- Add drag/drop file handling on chat input area/container
- Keep paste-image behavior unchanged

### Pending attachment state

Generalize the current pending image state to support any file:

- image: show thumbnail preview
- non-image: show file card (filename/size) and clear button

### Upload + send flow

Reuse existing `/upload?team_id=...` endpoint, then send a chat message with:

- `type: image` for image MIME types
- `type: file` otherwise

Use existing WS send path when connected, HTTP fallback otherwise.

### Message rendering

- Keep image preview rendering
- Add file message renderer for `type === "file"`
- Continue protected blob fetch + object URL caching for attachments
- Cache file metadata (filename/content-type/size) for rendering file cards

## Testing / Verification

- Backend tests for configurable upload limit and non-image acceptance
- Frontend build verification (`npm run build`)
- Manual smoke for icon attach, drag/drop attach, oversize error, cross-user open/download

## Boundaries

Not included in this pass:

- virus scanning
- resumable/chunked uploads
- upload progress bars
- file-type allowlist
- storage backend migration
