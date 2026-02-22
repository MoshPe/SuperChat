# Backend Layered Refactor Remaining Team Endpoints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move the remaining team endpoints (`members`, `join`, `leave`, `transfer-ownership`) into the layered backend so all `/api/teams*` routes are served by `internal/httpapi` using `internal/service` and `internal/store`.

**Architecture:** Extend the existing team-core vertical slice. `internal/store` gains membership/transfer/user lookup methods, `internal/service.TeamService` is extended to own all team business rules, and `internal/httpapi` adds the remaining team handlers with legacy-compatible HTTP mappings. `main.go` is then updated so every team route uses `apiServer`.

**Tech Stack:** Go, Gorilla Mux, BoltDB (`bbolt`), existing `internal/model`, `internal/httpapi` response helpers, `internal/service.TeamService`.

---

### Task 1: Add Store Tests for Membership + Transfer (RED)

**Files:**
- Modify: `internal/store/bolt_teams_test.go`

**Step 1: Write failing tests**
- `AddTeamMember`, `GetTeamMembers`, `RemoveTeamMember`, `IsTeamMember`
- `GetUserByID`
- `TransferTeamOwnership` updates `teams.owner_id` and membership roles
- transfer rejects non-member target

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/store -run "Team|Member|Transfer" -v`
- Expected: FAIL for missing methods.

### Task 2: Implement Store Membership + Transfer Methods (GREEN)

**Files:**
- Modify: `internal/store/store.go`
- Modify/Create: `internal/store/bolt_users.go` (add `GetUserByID`)
- Modify: `internal/store/bolt_teams.go`

**Step 1: Add interface methods**
- Extend store interfaces with membership/user lookup operations needed by `TeamService`.

**Step 2: Implement minimal Bolt methods**
- Port logic from legacy `database.go` for membership and transfer ownership.
- Return `ErrTeamNotFound` / regular errors as appropriate.

**Step 3: Run tests**
- Run: `go test ./internal/store -run "Team|Member|Transfer" -v`
- Expected: PASS.

### Task 3: Extend TeamService Tests for Remaining Team Endpoints (RED)

**Files:**
- Modify: `internal/service/team_service_test.go`

**Step 1: Write failing tests**
- member list requires membership and enriches with user names
- add member owner-only / username required / user not found / already member / success
- remove member owner-only / cannot remove owner / not member / success
- join team not found / already member / success
- leave team owner blocked / not member / success
- transfer ownership owner-only / target required / self transfer blocked / target not member / success

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/service -run TeamService -v`
- Expected: FAIL for missing methods or dependencies.

### Task 4: Implement TeamService Membership/Join/Leave/Transfer (GREEN)

**Files:**
- Modify: `internal/service/team_service.go`

**Step 1: Extend deps and service methods**
- Add membership/user lookup/transfer methods to dependencies
- Implement all remaining team behaviors with legacy-compatible rule messages via `ValidationError`

**Step 2: Run tests**
- Run: `go test ./internal/service -run TeamService -v`
- Expected: PASS.

### Task 5: Add HTTPAPI Tests + Router Smoke for Remaining Team Endpoints (RED)

**Files:**
- Modify: `internal/httpapi/handlers_teams_test.go`
- Modify: `internal/httpapi/router_smoke_test.go`

**Step 1: Write failing tests**
- representative mappings for each endpoint (400/403/404/409/200 equivalents via legacy messages)
- router route registration for `/members`, `/members/{userId}`, `/join`, `/leave`, `/transfer-ownership`

**Step 2: Run tests to verify RED**
- Run: `go test ./internal/httpapi -run Team -v`
- Expected: FAIL for missing handlers/routes.

### Task 6: Implement HTTPAPI Remaining Team Handlers + Route Wiring (GREEN)

**Files:**
- Modify: `internal/httpapi/router.go`
- Modify: `internal/httpapi/handlers_teams.go`

**Step 1: Implement handlers**
- Add `HandleGetTeamMembers`, `HandleAddTeamMember`, `HandleRemoveTeamMember`, `HandleJoinTeam`, `HandleLeaveTeam`, `HandleTransferOwnership`
- Map service errors to current legacy statuses/messages

**Step 2: Register routes in router**
- Add route registrations under `/api/teams/{id}` paths with auth middleware

**Step 3: Run tests**
- Run: `go test ./internal/httpapi -run Team -v`
- Expected: PASS.

### Task 7: Cut Over Remaining Team Routes in main.go

**Files:**
- Modify: `main.go`

**Step 1: Wire full TeamService deps from `BoltStore`**
- Remove legacy team function dependencies from `newHTTPAPIDeps` team wiring path
- Inject store-backed membership/transfer methods

**Step 2: Route cutover**
- Change remaining team routes to `apiServer` handlers:
  - `/teams/{id}/members` GET/POST
  - `/teams/{id}/members/{userId}` DELETE
  - `/teams/{id}/join`
  - `/teams/{id}/leave`
  - `/teams/{id}/transfer-ownership`

**Step 3: Run backend tests**
- Run: `go test ./...`
- Expected: PASS.

### Task 8: Verification + Cleanup

**Files:**
- Modify only if needed for dead imports/comments

**Step 1: Format**
- Run: `gofmt -w` on changed Go files

**Step 2: Final verification**
- Run: `go test ./...`
- Run: `npm run build` (frontend regression check)

**Step 3: Commit**
```bash
git add internal/store internal/service internal/httpapi main.go docs/plans/2026-02-22-backend-layered-refactor-team-remaining-design.md docs/plans/2026-02-22-backend-layered-refactor-team-remaining-plan.md
git commit -m "refactor: move remaining team routes into layered backend"
```
