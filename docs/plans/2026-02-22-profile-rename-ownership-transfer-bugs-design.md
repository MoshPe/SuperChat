# Profile Rename Realtime Sync + Ownership Transfer Bugfix Design

**Date:** 2026-02-22
**Status:** Approved

## Goals

1. Profile display-name changes should update message author labels for all connected users immediately.
2. Message ordering should remain stable after refresh.
3. Team owner must be able to transfer ownership to an existing member, then leave.
4. Clean obviously unused legacy code in root backend files if safe.

## Root Causes

- Historical messages store author display name snapshots; profile updates do not propagate to message views.
- WebSocket client display name is captured at connection time and can remain stale.
- `getTeamMessages` sorts only by timestamp, which can produce unstable order when timestamps collide.
- Owner leave is blocked but no ownership transfer endpoint/UI exists.

## Backend Design

- Add WebSocket event `user_profile_updated` broadcast on profile rename.
- Update connected websocket clients' display names server-side so future messages use the new display name.
- Broadcast rename event to active rooms where the user is a member (not just rooms they are connected to).
- Make message sorting deterministic by `(CreatedAt, ID)`.
- Add `POST /api/teams/{id}/transfer-ownership` (owner-only, target must be existing member and not self).

## Frontend Design

- Handle `user_profile_updated` in `Chat.jsx` and patch loaded messages by `user_id`.
- Add ownership transfer UI in `TeamSettingsModal` for owners (select existing member, confirm transfer).
- Surface owner-leave error in modal and keep flow explicit (transfer first, then leave).

## Testing

- Backend tests for deterministic message ordering tie-breaks.
- Backend tests for ownership transfer validation/success.
- Backend test for profile update broadcast helper behavior (and client display-name mutation).
- `go test ./...` + `npm run build`

