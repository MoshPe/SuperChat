# Backend Layered Refactor Team Core Slice Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move the team-core API routes (`/api/teams`, `/api/teams/{id}` CRUD) to the layered backend (`internal/httpapi` -> `internal/service` -> `internal/store`) while preserving the existing API contract.

**Architecture:** Implement a bounded vertical slice. `internal/store` owns Bolt team-core persistence, `internal/service` owns team-core validation/authorization using injected membership/owner checks, and `internal/httpapi` owns transport parsing/response mapping. `main.go` cuts over only the five team-core routes; deferred team membership/chat/upload routes remain on legacy handlers.

**Tech Stack:** Go, Gorilla Mux, BoltDB (`bbolt`), existing `internal/model`, existing `internal/httpapi` response helpers.

---

### Task 1: Add TeamStore Interface + BoltStore Team-Core Methods

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/bolt_teams.go`
- Create/Modify: `internal/store/bolt_teams_test.go` (new)

**Step 1: Write the failing tests**
- Add tests for `CreateTeam`, `GetTeamByID`, `GetUserTeams`, `UpdateTeam`, `DeleteTeam` on `BoltStore` using temp Bolt DB.
- Include delete cascading behavior for team record/members/messages/uploads to preserve legacy behavior.

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/store -run Team -v`
- Expected: FAIL for missing `BoltStore` team methods.

**Step 3: Write minimal implementation**
- Add `TeamStore` interface to `internal/store/store.go`.
- Implement team-core methods in `internal/store/bolt_teams.go` by porting legacy Bolt logic from `database.go`.

**Step 4: Run tests to verify GREEN**
- Run: `go test ./internal/store -run Team -v`
- Expected: PASS.

### Task 2: Add TeamService with Validation/Authz (TDD)

**Files:**
- Create: `internal/service/team_service.go`
- Create: `internal/service/team_service_test.go`
- Modify: `internal/service/service.go` (if dependency container/helper needed)

**Step 1: Write failing service tests**
- Test create team validation (name required)
- Test get team forbidden when not member
- Test update team forbidden when not owner
- Test delete team forbidden when not owner
- Test happy paths for list/get/update/delete

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/service -run TeamService -v`
- Expected: FAIL for missing types/functions.

**Step 3: Write minimal implementation**
- Add `TeamService` with injected dependencies:
  - `store.TeamStore`
  - `IsTeamMember func(teamID, userID string) bool`
  - `IsTeamOwner func(teamID, userID string) bool`
- Add sentinel errors (`ErrForbidden`, `ErrNotFound`, validation error helper).
- Preserve legacy semantics for updates (owner-only, trimmed fields, required name).

**Step 4: Run tests to verify GREEN**
- Run: `go test ./internal/service -run TeamService -v`
- Expected: PASS.

### Task 3: Add internal/httpapi Team-Core Handlers + Tests (TDD)

**Files:**
- Modify: `internal/httpapi/router.go`
- Modify: `internal/httpapi/handlers_auth.go` (only if `Server` type reuse is easiest) or Create `internal/httpapi/handlers_teams.go`
- Create: `internal/httpapi/handlers_teams_test.go`
- Modify: `internal/httpapi/router_smoke_test.go`

**Step 1: Write failing handler/router tests**
- Handler tests for representative cases:
  - create team invalid body / validation failure
  - get team forbidden
  - get team not found
  - update team success
  - delete team success
- Router smoke tests for team-core route registration using injected handlers

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/httpapi -run Team -v`
- Expected: FAIL for missing handlers/route wiring.

**Step 3: Write minimal implementation**
- Extend `HandlerDeps` to carry `TeamService` and legacy fallbacks (if desired during migration).
- Add `Server` methods for team-core routes.
- Parse auth context from request (using existing middleware-populated context keys).
- Map service errors to legacy HTTP response envelopes/messages with `WriteSuccess`/`WriteError`.
- Register team-core routes in `NewRouter`.

**Step 4: Run tests to verify GREEN**
- Run: `go test ./internal/httpapi -run Team -v`
- Expected: PASS.

### Task 4: Wire TeamService + Team-Core Route Cutover in main.go

**Files:**
- Modify: `main.go`

**Step 1: Write/adjust failing wiring tests (optional smoke if needed)**
- Prefer extending existing router smoke tests instead of new `main.go` tests.

**Step 2: Implement wiring**
- Build `internal/store.BoltStore` (already exists) and `internal/service.TeamService` in `main.go`
- Inject service into `httpapi.NewServer` deps
- Cut over only these routes in `main.go` to `apiServer` methods:
  - `/teams` GET/POST
  - `/teams/{id}` GET/PUT/DELETE
- Leave membership/join/leave/transfer routes on legacy handlers

**Step 3: Run backend tests**
- Run: `go test ./...`
- Expected: PASS.

### Task 5: Verification + Conservative Cleanup

**Files:**
- Modify only if needed for dead imports/comments

**Step 1: Run frontend build (regression check)**
- Run: `npm run build` in `frontend/`
- Expected: PASS.

**Step 2: Cleanup obvious dead code only**
- Remove only now-unused imports or misleading comments introduced by the slice.
- Do not delete legacy handlers/database functions still needed by deferred routes.

**Step 3: Final verification**
- Run: `go test ./...`
- Run: `npm run build`

**Step 4: Commit**
```bash
git add internal/store internal/service internal/httpapi main.go docs/plans/2026-02-22-backend-layered-refactor-team-core-design.md docs/plans/2026-02-22-backend-layered-refactor-team-core-plan.md
git commit -m "refactor: move team core routes into layered backend"
```
