# Profile Rename + Ownership Transfer Bugfix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix realtime display-name propagation, stable message ordering after refresh, and owner ownership-transfer-before-leave flow.

**Architecture:** Use WebSocket event broadcast for instant rename sync, deterministic message sorting in persistence retrieval, and a new owner-only transfer endpoint plus minimal transfer UI in team settings.

**Tech Stack:** Go, Gorilla WebSocket, BoltDB, React, Axios

---

### Task 1: Backend tests for ordering and ownership transfer
- Add failing tests for timestamp tie ordering and transfer ownership validation/success.

### Task 2: Backend rename sync + ordering fixes
- Add `user_profile_updated` WS broadcast helper and invoke from profile update.
- Update connected WS client display names.
- Prefer fresh DB display name in websocket connect flow.
- Deterministic message sort `(CreatedAt, ID)`.

### Task 3: Backend ownership transfer endpoint
- Add store/helper to transfer `team.owner_id`.
- Add `POST /api/teams/{id}/transfer-ownership`.
- Wire route in `main.go`.

### Task 4: Frontend realtime rename handling + ownership transfer UI
- Handle `user_profile_updated` in chat WS handler.
- Add transfer ownership UI to `TeamSettingsModal`.
- Surface owner leave error and support transfer-then-leave flow.

### Task 5: Cleanup + verification
- Remove obvious unused root functions/imports if safe.
- Run `go test ./...` and `npm run build`.

