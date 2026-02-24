# Team Screen Share (WebM Relay + MSE) Design

**Date:** 2026-02-23

## Goal

Add screen sharing to team chat where exactly one team member can share at a time, using a server-relayed stream (no WebRTC), with Chrome/Edge desktop support and target latency of roughly 0.5s-2s.

## Agreed Requirements

- Browser support: Chrome/Edge desktop only (v1)
- Media: screen only (no system audio)
- Quality/perf target: 30fps with readable text
- Latency target: 0.5s-2s (prefer closer to 0.5s)
- Scale target: small teams / small audience (about <=10 viewers)
- Permissions: any team member can start sharing
- Concurrency: only one active sharer per team
- Contention behavior: block new share attempts and show who is currently sharing

## Chosen Approach

Use `MediaRecorder` in the browser to capture screen video as WebM chunks, relay those chunks through the server to viewers, and render on viewers with `MediaSource` / `SourceBuffer` (MSE).

Why this approach:

- Meets the no-WebRTC requirement
- Lower complexity/risk than a WebCodecs custom codec pipeline
- Better quality/bandwidth tradeoff than MJPEG/JPEG frame streaming
- Fits the small-scale v1 target

## Alternatives Considered

### 1) MJPEG / JPEG frames over WebSocket

- Pros: simplest conceptually
- Cons: high bandwidth/CPU at 30fps, poor scalability, weaker readability unless bitrate is very high
- Result: rejected for v1

### 2) MediaRecorder WebM chunks + relay + MSE (selected)

- Pros: practical implementation effort, decent quality, no P2P/WebRTC
- Cons: latency is usually closer to ~0.8s-2s than near-instant
- Result: selected for v1

### 3) WebCodecs encode/decode + binary relay

- Pros: best latency control, strongest chance of staying near ~0.5s
- Cons: significantly higher complexity and implementation risk
- Result: possible v2 upgrade path if MediaRecorder latency is not sufficient

## High-Level Architecture

### Transport split

- Keep existing chat WebSocket (`/api/ws/{teamId}`) for control/signaling JSON messages
- Add a separate screen stream WebSocket endpoint for media chunks (publisher/viewer roles)

This avoids mixing high-volume media data with chat JSON and preserves current chat message behavior.

### Team-level screen share session

Maintain in-memory per-team screen share state on the server:

- active sharer identity
- sharer display name
- session ID
- session status (`reserved` vs `active`)
- timestamps / last-seen
- publisher connection
- viewer connections
- stream metadata (`mimeType`, etc.)

Only one active or reserved session may exist per team at a time.

### Publisher (sharer) flow

1. User clicks `Share screen` in chat.
2. Client requests a share lock on the existing chat WS.
3. Server grants or denies based on team active session state.
4. If granted, client runs `getDisplayMedia(...)`.
5. Client starts `MediaRecorder` and opens screen stream WS as `publisher`.
6. Client sends `init` metadata followed by WebM chunks.
7. Server relays chunks to current viewers.
8. On stop/disconnect, server clears session and broadcasts stop event.

### Viewer flow

1. Client receives `screen_share_started` on chat WS.
2. UI shows active screen-share panel and sharer name.
3. Client opens screen stream WS as `viewer` for the session.
4. Client receives `init` metadata and media chunks.
5. Client appends chunks to `MediaSource` / `SourceBuffer`.
6. On `screen_share_stopped`, client resets viewer state and UI.

## Protocol Design

### Control messages (chat WS JSON)

Client -> server:

- `screen_share_request_start`
- `screen_share_stop` (from current sharer)

Server -> client/team:

- `screen_share_start_granted` (to requester only)
  - includes `session_id`, optional lease timeout metadata
- `screen_share_denied` (to requester only)
  - includes current sharer name and user ID
- `screen_share_started` (broadcast to team)
  - includes `session_id`, `sharer_user_id`, `sharer_name`, timestamp
- `screen_share_stopped` (broadcast to team)
  - includes `session_id`, `sharer_user_id`, reason (optional)
- `screen_share_error` (to requester/publisher, optional)

Notes:

- Existing chat WS remains JSON-only.
- These messages are lightweight and integrate with current `WebSocketMessage` handling.

### Screen stream WebSocket endpoint

Proposed endpoint:

- `/api/ws/{teamId}/screen?token=<jwt>&role=<publisher|viewer>&session_id=<id>`

Server validates:

- token
- team membership
- session ID validity
- role permissions (publisher must match granted user/session)

### Screen WS payloads

Publisher -> server:

- `init` JSON frame first (text WS message):
  - `type: "init"`
  - `mime_type`
  - `width`, `height`
  - `fps_target`
- then binary WS messages with raw WebM chunks

Server -> viewers:

- relay `init` first (or immediately on viewer join if already known)
- relay binary chunk messages in order

Viewer handling:

- use `MediaSource` + `SourceBuffer`
- append chunks in order
- maintain a small queue for backpressure and append sequencing

## Media Settings (v1)

### Capture

Use `navigator.mediaDevices.getDisplayMedia` with:

- `video.frameRate.ideal = 30`
- `video.frameRate.max = 30`
- `audio = false`

### Encoder / container

Prefer WebM MIME types in this order (browser capability checked with `MediaRecorder.isTypeSupported`):

1. `video/webm;codecs=vp9`
2. `video/webm;codecs=vp8`
3. `video/webm`

### Chunk cadence

- Start with `timeslice = 150ms`
- Allow tuning in code/config after initial testing (e.g. 100-250ms)

### Bitrate

- Start around `2.5-6 Mbps` depending on capture resolution and readability
- Favor readable text over smooth motion if tradeoffs are required
- Optionally cap capture render resolution for stability (e.g. 1280x720 @ 30fps)

### Expected latency

- Best-case (local/good network): ~0.5s-1s
- Typical: ~1s-2s

This meets the v1 target range.

## Backend Design (Go)

## Server state

Add a dedicated in-memory screen-share registry (team -> active session) protected by mutexes.

Suggested session fields:

- `SessionID`
- `TeamID`
- `SharerUserID`
- `SharerName`
- `Status` (`reserved`, `active`)
- `StartedAt`
- `ReservedAt`
- `LastChunkAt`
- `PublisherConn`
- `Viewers`
- `MimeType`
- `Width`, `Height`
- `FPSTarget`

## Routing

Add a new WebSocket route for screen streaming, separate from existing chat WS route:

- existing: `/api/ws/{teamId}` (chat control/messages)
- new: `/api/ws/{teamId}/screen` (media streaming)

## Enforcement rules

- Single sharer lock per team (atomic check/set)
- Publisher may connect only if:
  - session exists
  - session is `reserved` or `active`
  - publisher user matches reserved sharer
- Viewer may connect only if:
  - session exists and is active (or reserved with init pending, if supported)
  - viewer is a team member
- Max viewers cap enforced (v1 target ~10)

## Lifecycle and cleanup

- `reserved` lease expires quickly (e.g. 10s) if publisher never connects/streams
- `active` session expires if publisher disconnects or no chunks arrive within timeout (e.g. 3-5s)
- On cleanup:
  - close viewer connections
  - clear session state
  - broadcast `screen_share_stopped`

## Limits and hardening

Separate limits from chat WS:

- chat WS keeps small JSON read limits (existing behavior)
- screen WS gets larger limits appropriate for media chunks

Validate:

- `init` message size and fields
- allowed roles
- chunk max size
- session/team/user ownership

Drop malformed packets and wrong-session traffic.

## Frontend Design (React / Chat UI)

## UI placement

Add a screen-share panel in `Chat` near the top of the chat content (below team header, above messages), with states:

- no active share (share button)
- another user sharing (show sharer and viewer panel)
- current user sharing (status + stop button + local preview optional)

## Client state

Control/session state:

- `activeScreenShare` (`sessionId`, `sharerUserId`, `sharerName`, status)

Publisher state:

- `mediaStream`
- `mediaRecorder`
- `screenPublishWs`
- `isPublishing`

Viewer state:

- `screenViewerWs`
- `mediaSource`
- `sourceBuffer`
- `appendQueue`
- `isViewing`
- `viewerError`

## Publisher client flow

1. Send `screen_share_request_start` over current chat WS.
2. On grant, call `getDisplayMedia`.
3. Open screen WS as publisher with `session_id`.
4. Send `init` metadata.
5. Start `MediaRecorder` and send binary chunks every timeslice.
6. Handle `track.onended` to stop sharing and cleanup.

If capture is canceled before publishing begins, client releases the reserved session via control message or server lease timeout.

## Viewer client flow

1. Receive `screen_share_started`.
2. Show screen-share viewer UI.
3. Connect screen WS as viewer.
4. Initialize MSE after `init`.
5. Append chunk queue in order.
6. Handle stop/error and cleanup.

## UX requirements

- Block local share attempt if another user is sharing and show the sharer name
- Clear status and error messages for:
  - browser permission denied / picker canceled
  - unsupported MIME/MSE
  - active share already exists
  - stream ended
- Auto cleanup on route change/unmount
- Auto cleanup when user leaves team/chat

## Reliability, Security, and Testing

## Reliability

- Reserved session lease timeout (~10s)
- Active session idle timeout (~3-5s without chunks)
- Cleanup on publisher disconnect and server restart
- Viewers are live-only (no replay)

## Security

- Reuse JWT token validation and team membership checks for both WS channels
- Enforce publisher session ownership (granted user must be the publisher)
- Enforce single active sharer atomically
- Validate metadata and chunk sizes
- Cap viewers and drop malformed traffic

## Performance guardrails (v1)

- Small team target only
- In-memory relay only (no recording)
- Prefer dropping stale chunks over unbounded buffering if a viewer falls behind
- Add basic logs/metrics:
  - started/stopped sessions
  - denied attempts
  - viewer counts
  - oversized/dropped chunks

## Testing / Verification

### Backend tests

- lock acquisition and deny second sharer
- show current sharer identity in deny response/event
- valid publisher attach only for granted session/user
- viewer attach only for active session
- cleanup on publisher disconnect/timeout
- malformed init / oversized chunk rejection

### Frontend verification

- share start/stop happy path
- browser picker cancel path
- viewer joins active share and video renders
- blocked attempt shows sharer name
- page unmount cleans media tracks and sockets
- route/team leave cleanup

### Browser matrix

- Chrome desktop
- Edge desktop

## Out of Scope (v1)

- WebRTC / peer-to-peer transport
- System audio
- Recording/replay/history
- Multi-sharer sessions
- Approval-based takeover requests
- Safari / mobile support
- Advanced adaptive bitrate controls
