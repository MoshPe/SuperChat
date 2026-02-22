# Backend Layered Refactor Remaining Team Endpoints Design

**Date:** 2026-02-22

## Scope and Route Ownership

This slice migrates all remaining team endpoints to the layered backend (`internal/httpapi -> internal/service -> internal/store`) so all `/api/teams*` routes are owned by `apiServer`.

Routes in scope:

- `GET /api/teams/{id}/members`
- `POST /api/teams/{id}/members`
- `DELETE /api/teams/{id}/members/{userId}`
- `POST /api/teams/{id}/join`
- `POST /api/teams/{id}/leave`
- `POST /api/teams/{id}/transfer-ownership`

Behavior contract to preserve:

- Same paths, auth middleware, status codes, JSON envelopes, and messages
- Same membership/owner authorization semantics
- Owner cannot leave until ownership is transferred
- Transfer ownership behavior added in the bugfix pass remains unchanged

After this slice, all `/api/teams*` routes in `main.go` should use `apiServer` handlers.

## Layered Contracts

### internal/store additions

Extend `BoltStore` with membership and ownership persistence operations:

- `AddTeamMember(*model.TeamMember) error`
- `RemoveTeamMember(teamID, userID string) error`
- `GetTeamMembers(teamID string) ([]*model.TeamMember, error)`
- `IsTeamMember(teamID, userID string) (bool, error)`
- `TransferTeamOwnership(teamID, currentOwnerID, newOwnerID string) error`
- `GetUserByID(userID string) (*model.User, error)`

`GetUserByUsername` and team-core methods already exist and will be reused.

### internal/service additions (extend TeamService)

Extend `TeamService` to own all team business rules:

- `GetTeamMembers(teamID, requesterID string) ([]TeamMemberView, error)`
- `AddTeamMember(teamID, requesterID, targetUsername string) (targetUserID string, error)`
- `RemoveTeamMember(teamID, requesterID, targetUserID string) error`
- `JoinTeam(teamID, requesterID string) (*model.Team, error)`
- `LeaveTeam(teamID, requesterID string) error`
- `TransferOwnership(teamID, requesterID, newOwnerID string) (*model.Team, error)`

The service will handle owner/member checks, conflict cases, and transfer/leave restrictions. It returns typed/sentinel errors for `httpapi` mapping.

### internal/httpapi additions

Add handlers and router registration for all remaining team endpoints. Handlers remain transport-only:

- parse path/body
- read auth context
- call `TeamService`
- map service errors to legacy HTTP status/messages with `WriteSuccess`/`WriteError`

## Error Mapping and Data Shape Compatibility

### Error model

Use the current service pattern:

- `ErrForbidden`, `ErrNotFound`
- `ValidationError{Message}` for message-specific rule failures

Use `ValidationError` for cases like:

- already member
- not a member
- cannot remove team owner
- owner cannot leave
- target user required / username required / cannot transfer to self

### Response payload compatibility

Preserve current payload shapes:

- `GET /teams/{id}/members` returns enriched member objects (`id`, `team_id`, `user_id`, `username`, `name`, `role`, `joined_at`)
- `POST /teams/{id}/members` returns `{ "user_id": "..." }`
- `POST /teams/{id}/join` returns team object
- `POST /teams/{id}/leave` returns `data: null`
- `POST /teams/{id}/transfer-ownership` returns updated team object

## Migration Strategy

1. Add store methods + store tests for membership/transfer/user lookup.
2. Extend `TeamService` + tests for all remaining team behaviors.
3. Add `httpapi` handlers + tests + router registration.
4. Cut over remaining team routes in `main.go` to `apiServer`.
5. Verify `go test ./...` and `npm run build`.

Legacy root team handlers and DB helpers can remain temporarily after cutover for cleanup in a later pass.

## Testing and Merge Checkpoint

TDD coverage:

- `internal/store`: membership CRUD + transfer ownership persistence behavior
- `internal/service`: owner/member rules, join/leave conflicts, transfer constraints, member list enrichment
- `internal/httpapi`: representative endpoint status/message mappings
- `internal/httpapi` router smoke tests for all remaining team routes

Merge checkpoint:

- All `/api/teams*` routes in `main.go` use `apiServer`
- `go test ./...` passes
- `npm run build` passes
