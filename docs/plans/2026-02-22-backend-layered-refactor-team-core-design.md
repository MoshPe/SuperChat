# Backend Layered Refactor Team Core Slice Design

**Date:** 2026-02-22

## Scope and Route Ownership

This slice moves only the team-core routes to the layered backend while preserving behavior:

- `POST /api/teams`
- `GET /api/teams`
- `GET /api/teams/{id}`
- `PUT /api/teams/{id}`
- `DELETE /api/teams/{id}`

Deferred routes remain on legacy root handlers for now:

- `/teams/{id}/members*`
- `/teams/{id}/join`
- `/teams/{id}/leave`
- `/teams/{id}/transfer-ownership`
- chat/uploads/ws routes

Behavior contract for this slice:

- Same route paths and auth middleware behavior
- Same request/response JSON shapes
- Same status codes/messages
- No frontend changes required for this refactor slice

## Layered Contracts

### internal/store (Bolt)

Add a `TeamStore` interface (team-core only):

- `CreateTeam(*model.Team) error`
- `GetTeamByID(teamID string) (*model.Team, error)`
- `GetUserTeams(userID string) ([]*model.Team, error)`
- `UpdateTeam(*model.Team) error`
- `DeleteTeam(teamID string) error`

`internal/store.BoltStore` implements these by owning the BoltDB logic currently in `database.go` for team-core persistence.

### internal/service (use-cases)

Add `TeamService` that depends on `TeamStore` plus narrow authorization dependencies for now (membership/owner checks as injected funcs or interfaces):

- `CreateTeam(userID string, req model.CreateTeamRequest) (*model.Team, error)`
- `ListTeams(userID string) ([]*model.Team, error)`
- `GetTeam(teamID, userID string) (*model.Team, error)`
- `UpdateTeam(teamID, userID string, updates ...) (*model.Team, error)`
- `DeleteTeam(teamID, userID string) error`

The service owns validation and authorization decisions. `httpapi` remains transport-only.

### internal/httpapi (transport)

Add team-core handlers to `internal/httpapi.Server`:

- `HandleCreateTeam`
- `HandleGetTeams`
- `HandleGetTeam`
- `HandleUpdateTeam`
- `HandleDeleteTeam`

Handlers parse requests, read auth context, call the service, and write the same response envelopes/status codes as the current legacy handlers.

## Error Mapping and Migration Strategy

### Error model

Use sentinel service errors for stable HTTP mapping:

- `ErrForbidden`
- `ErrNotFound`
- `ErrValidation` (or validation errors mapped explicitly)

`internal/httpapi` maps these to the same legacy HTTP status codes/messages. Unknown errors map to `500`.

### Migration mechanics

1. Add/move team-core Bolt methods into `internal/store`.
2. Add `internal/service.TeamService` using `TeamStore` + injected membership/owner checks.
3. Implement team-core handlers in `internal/httpapi` using the service.
4. Cut over only the five team-core routes in `main.go`.
5. Keep legacy handlers/database functions for deferred routes and parity fallback during migration.

## Testing Strategy

TDD for this slice:

- `internal/store` tests for team-core persistence methods.
- `internal/service` tests for validation/authorization/happy paths using fakes.
- `internal/httpapi` tests for representative status/JSON mapping.
- `internal/httpapi` router smoke tests for team-core routes.
- Full verification after cutover: `go test ./...` and `npm run build`.

## Boundaries and Merge Checkpoint

Stop this slice after:

- Team-core routes are served by `internal/httpapi` in `main.go`
- `internal/service` and `internal/store` own the team-core vertical path
- `go test ./...` passes
- `npm run build` passes

Do not migrate membership/join/leave/transfer/chat/upload in this slice.
