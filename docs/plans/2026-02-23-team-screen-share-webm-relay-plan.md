# Team Screen Share (WebM Relay + MSE) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add single-sharer team screen sharing in chat using Chrome/Edge desktop `getDisplayMedia` + `MediaRecorder` (WebM chunks), server-side WebSocket relay, and viewer playback via `MediaSource`/MSE.

**Architecture:** Keep the existing chat WebSocket (`/api/ws/{teamId}`) for control/signaling JSON and add a separate screen-stream WebSocket endpoint for media chunks. The backend maintains an in-memory per-team screen-share session lock/registry (one sharer only), while the frontend adds a screen-share panel in chat and a client pipeline for publisher capture and viewer MSE playback.

**Tech Stack:** Go, Gorilla WebSocket, Gorilla Mux, in-memory session registry (`sync.RWMutex`), existing JWT/team membership checks, React, Vite, browser `getDisplayMedia`, `MediaRecorder`, `MediaSource` (MSE).

---

### Task 1: Define Backend Screen-Share Session Registry (TDD first)

**Files:**
- Create: `websocket_screen_share.go`
- Create: `websocket_screen_share_test.go`

**Step 1: Write failing tests for registry rules**
- Add tests for:
  - acquiring a share lock for an empty team succeeds
  - second sharer acquisition for same team is denied and returns current sharer identity
  - releasing a session clears the lock
  - wrong user cannot attach as publisher to reserved session

**Step 2: Run tests to verify RED**

Run: `go test ./... -run ScreenShareRegistry -v`

Expected: FAIL due to missing registry/session code.

**Step 3: Write minimal registry implementation**
- Add an in-memory session registry with mutex protection
- Implement methods like:
  - reserve session
  - get active/reserved session
  - validate publisher attach
  - release session

**Step 4: Run tests to verify GREEN**

Run: `go test ./... -run ScreenShareRegistry -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add websocket_screen_share.go websocket_screen_share_test.go
git commit -m "feat: add screen share session registry"
```

### Task 2: Add Screen-Share Control Messages to Existing Chat WebSocket (TDD first)

**Files:**
- Modify: `websocket.go`
- Modify: `websocket_screen_share.go`
- Modify: `websocket_screen_share_test.go`

**Step 1: Write failing tests for control flow**
- Add tests covering:
  - `screen_share_request_start` grants when no active sharer
  - `screen_share_request_start` denies when a sharer is already active/reserved
  - deny payload includes current sharer name/user ID
  - `screen_share_stop` from sharer releases session and broadcasts stop event
  - `screen_share_stop` from non-sharer is ignored/rejected safely

**Step 2: Run tests to verify RED**

Run: `go test ./... -run ScreenShareControl -v`

Expected: FAIL due to unknown WS message types / missing handlers.

**Step 3: Implement control message handling**
- Extend existing chat WS message switch in `websocket.go` to handle:
  - `screen_share_request_start`
  - `screen_share_stop`
- Broadcast control events using existing room broadcast path (`WebSocketMessage`)
- Preserve existing chat/typing behavior unchanged

**Step 4: Run tests to verify GREEN**

Run: `go test ./... -run ScreenShareControl -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add websocket.go websocket_screen_share.go websocket_screen_share_test.go
git commit -m "feat: add screen share control events to chat websocket"
```

### Task 3: Add Separate Screen Stream WebSocket Endpoint + Route (TDD first)

**Files:**
- Modify: `main.go`
- Modify: `websocket_screen_share.go`
- Modify: `websocket_screen_share_test.go`

**Step 1: Write failing endpoint tests**
- Cover:
  - unauthorized token rejected
  - non-member rejected
  - viewer connection rejected when no active session exists
  - publisher connection rejected when user/session mismatch

**Step 2: Run tests to verify RED**

Run: `go test ./... -run ScreenShareEndpoint -v`

Expected: FAIL due to missing route/handler.

**Step 3: Implement screen stream WS handler**
- Add `/api/ws/{teamId}/screen` route in `main.go`
- Add new handler in `websocket_screen_share.go` to:
  - validate token
  - validate membership
  - parse `role` and `session_id`
  - attach viewer/publisher to registry session

**Step 4: Run tests to verify GREEN**

Run: `go test ./... -run ScreenShareEndpoint -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add main.go websocket_screen_share.go websocket_screen_share_test.go
git commit -m "feat: add screen stream websocket endpoint"
```

### Task 4: Implement Media Chunk Relay (Publisher -> Viewers) with Guardrails (TDD first)

**Files:**
- Modify: `websocket_screen_share.go`
- Modify: `websocket_screen_share_test.go`

**Step 1: Write failing relay tests**
- Add tests for:
  - publisher `init` metadata accepted and stored
  - viewer receives `init` after joining an active session
  - binary chunks relay to all viewers
  - oversized chunk is rejected/dropped
  - wrong packet ordering/type does not crash handler

**Step 2: Run tests to verify RED**

Run: `go test ./... -run ScreenShareRelay -v`

Expected: FAIL due to missing relay logic.

**Step 3: Implement minimal relay logic**
- Accept publisher `init` JSON (text WS message)
- Store session metadata (mime, dimensions, fps target)
- Relay binary frames/chunks to viewer connections
- Enforce max chunk size and viewer cap
- Add safe write/cleanup paths for broken viewers

**Step 4: Run tests to verify GREEN**

Run: `go test ./... -run ScreenShareRelay -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add websocket_screen_share.go websocket_screen_share_test.go
git commit -m "feat: relay screen share webm chunks to viewers"
```

### Task 5: Implement Session Lease Expiry / Cleanup and Disconnect Handling (TDD first)

**Files:**
- Modify: `websocket.go`
- Modify: `websocket_screen_share.go`
- Modify: `websocket_screen_share_test.go`

**Step 1: Write failing cleanup tests**
- Cover:
  - reserved session expires if publisher never connects
  - active session expires if publisher stops sending chunks
  - publisher disconnect triggers session release and stop broadcast
  - sharer chat disconnect also releases session if policy is tied to chat client

**Step 2: Run tests to verify RED**

Run: `go test ./... -run ScreenShareCleanup -v`

Expected: FAIL due to missing timeout/cleanup hooks.

**Step 3: Implement cleanup/timeouts**
- Add lease timestamps/timeouts in registry
- Add cleanup helper invoked on:
  - publisher WS close
  - chat client disconnect (for current sharer)
  - timer/heartbeat timeout
- Broadcast `screen_share_stopped` once per cleanup path

**Step 4: Run tests to verify GREEN**

Run: `go test ./... -run ScreenShareCleanup -v`

Expected: PASS.

**Step 5: Commit**

```bash
git add websocket.go websocket_screen_share.go websocket_screen_share_test.go
git commit -m "feat: add screen share session cleanup and timeouts"
```

### Task 6: Add Frontend Screen-Share Panel UI and Control State (no media playback yet)

**Files:**
- Modify: `frontend/src/pages/Chat.jsx`
- Create (optional): `frontend/src/components/ChatScreenSharePanel.jsx`

**Step 1: Add chat WS control message handling in frontend**
- Extend `handleWebSocketMessage` in `frontend/src/pages/Chat.jsx` for:
  - `screen_share_start_granted`
  - `screen_share_started`
  - `screen_share_denied`
  - `screen_share_stopped`

**Step 2: Add local screen-share UI state**
- Track:
  - active session metadata
  - blocked/denied reason
  - publishing/viewing status
  - user-facing errors

**Step 3: Render a screen-share panel in chat**
- Place below team header and above message list
- Show:
  - `Share screen` button (enabled when no active sharer)
  - blocked state with current sharer name
  - `Stop sharing` button when current user is sharer

**Step 4: Manual verification (UI state only)**
- Start app and open two browser tabs in same team
- Simulate/inspect control events (or use temporary dev triggers if needed)
- Confirm blocked state shows sharer name

**Step 5: Commit**

```bash
git add frontend/src/pages/Chat.jsx frontend/src/components/ChatScreenSharePanel.jsx
git commit -m "feat: add chat screen share panel and control state"
```

### Task 7: Implement Frontend Publisher Capture + Screen Stream WS Uplink

**Files:**
- Modify: `frontend/src/pages/Chat.jsx`
- Create (optional helper): `frontend/src/utils/screenSharePublisher.js`

**Step 1: Add share-start happy-path implementation**
- On `Share screen` click:
  - send `screen_share_request_start` on chat WS
  - wait for grant event
  - call `navigator.mediaDevices.getDisplayMedia(...)`

**Step 2: Add MediaRecorder setup**
- Select supported MIME in order:
  - VP9 WebM
  - VP8 WebM
  - plain WebM
- Use target timeslice (start at `150ms`)

**Step 3: Add publisher screen WS uplink**
- Connect to `/api/ws/{teamId}/screen?...role=publisher&session_id=...`
- Send `init` JSON first, then binary chunks from `MediaRecorder`

**Step 4: Add cleanup logic**
- Stop recorder and media tracks
- Close publisher WS
- Handle browser picker cancel and `track.onended`
- Send `screen_share_stop` control message when appropriate

**Step 5: Manual verification**
- In Chrome/Edge, start sharing and confirm no client errors
- Stop via browser UI and confirm local cleanup

**Step 6: Commit**

```bash
git add frontend/src/pages/Chat.jsx frontend/src/utils/screenSharePublisher.js
git commit -m "feat: add screen capture publisher pipeline"
```

### Task 8: Implement Frontend Viewer Playback via MSE (MediaSource)

**Files:**
- Modify: `frontend/src/pages/Chat.jsx`
- Create (optional helper): `frontend/src/utils/screenShareViewerMse.js`

**Step 1: Add viewer screen WS connection**
- On `screen_share_started`, connect as `viewer` with `session_id`
- Handle stop/reconnect cleanup paths

**Step 2: Add MSE setup and append queue**
- Create `MediaSource`
- Create `SourceBuffer` after `init` metadata
- Queue and append chunks in order
- Handle `updateend` and append sequencing

**Step 3: Render `<video>` viewer panel**
- Autoplay muted
- Display sharer name and connection state
- Show human-readable error if MSE init fails

**Step 4: Add buffer trimming / latency control (minimal)**
- Prevent unbounded buffer growth
- Keep a small rolling buffered window to stay near v1 latency target

**Step 5: Manual verification**
- Open sharer + viewer tabs in same team
- Confirm viewer renders screen stream
- Confirm stop event tears down video cleanly

**Step 6: Commit**

```bash
git add frontend/src/pages/Chat.jsx frontend/src/utils/screenShareViewerMse.js
git commit -m "feat: add mse viewer for screen share relay"
```

### Task 9: Frontend/Backend Integration Hardening (Errors, Limits, UX polish)

**Files:**
- Modify: `websocket_screen_share.go`
- Modify: `websocket.go`
- Modify: `frontend/src/pages/Chat.jsx`
- Modify (optional): `frontend/src/components/ChatScreenSharePanel.jsx`

**Step 1: Add user-visible error mapping**
- Permission denied / capture canceled
- Share denied (already active sharer)
- Unsupported browser codec/MSE
- Stream disconnected

**Step 2: Add reconnection/cleanup guards**
- Avoid duplicate screen WS connections
- Ensure route change/unmount closes publisher/viewer resources
- Ensure leaving team/chat clears local screen-share state

**Step 3: Add basic logging/metrics hooks (server)**
- Log start/stop/deny events and viewer counts
- Log dropped oversized chunks

**Step 4: Manual verification**
- Negative path: second sharer blocked and name shown
- Cancel picker after grant -> lock is released (or expires quickly)
- Sharer tab close -> viewers see stop state

**Step 5: Commit**

```bash
git add websocket.go websocket_screen_share.go frontend/src/pages/Chat.jsx frontend/src/components/ChatScreenSharePanel.jsx
git commit -m "feat: harden screen share relay errors and cleanup"
```

### Task 10: End-to-End Verification and Documentation Updates

**Files:**
- Modify: `README.md` (if adding feature note / limitations)
- Modify: `CHANGELOG.md` (if project workflow expects changelog updates)
- Verify: `docs/plans/2026-02-23-team-screen-share-webm-relay-design.md`
- Verify: `docs/plans/2026-02-23-team-screen-share-webm-relay-plan.md`

**Step 1: Backend verification**

Run: `go test ./...`

Expected: PASS, including new screen-share tests.

**Step 2: Frontend verification**

Run: `npm run build`

Expected: PASS (if local Node/Vite version satisfies project requirement).

**Step 3: Manual browser verification (Chrome + Edge)**
- Sharer + at least one viewer in same team
- 30fps target appears smooth enough for screen movement
- Text remains readable at chosen bitrate/resolution
- Latency lands within 0.5s-2s target window in local/test environment

**Step 4: Document v1 limitations**
- no audio
- one sharer only
- Chrome/Edge desktop only
- small-scale audience target

**Step 5: Commit**

```bash
git add README.md CHANGELOG.md docs/plans/2026-02-23-team-screen-share-webm-relay-design.md docs/plans/2026-02-23-team-screen-share-webm-relay-plan.md
git commit -m "docs: add screen share relay plan and limitations"
```
