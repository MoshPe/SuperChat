# Backend Layered Refactor Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the Go backend into `internal/httpapi`, `internal/service`, and `internal/store` with `main.go` as wiring-only, while preserving all existing routes and behavior.

**Architecture:** Introduce a composition root in `main.go`, move HTTP routing/handlers into `internal/httpapi`, wrap BoltDB access in `internal/store`, and extract business logic into `internal/service` incrementally. Use behavior-preserving mechanical moves first, then logic extraction behind interfaces.

**Tech Stack:** Go, Gorilla Mux, Gorilla WebSocket (phase-1 compatibility), BoltDB (`bbolt`), `net/http`, `httptest`

---

### Task 1: Create Package Skeleton and Shared Contracts

**Files:**
- Create: `internal/httpapi/router.go`
- Create: `internal/httpapi/response.go`
- Create: `internal/service/service.go`
- Create: `internal/store/store.go`
- Modify: `main.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Create a new smoke test file placeholder:

```go
// internal/httpapi/router_test.go
package httpapi

import "testing"

func TestPackageCompiles(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL due to missing new package files/imports while wiring is introduced.

**Step 3: Write minimal implementation**

- Define `httpapi` package with:
  - `type HandlerDeps struct { ... }`
  - `func NewRouter(deps HandlerDeps) http.Handler`
- Define `service` package with interfaces and a minimal `Services` container.
- Define `store` package with interfaces only (no Bolt implementation yet).
- Update `main.go` only enough to keep current code compiling (can continue using existing router temporarily).

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add main.go internal/httpapi/router.go internal/httpapi/response.go internal/service/service.go internal/store/store.go
git commit -m "refactor: add backend layered package skeleton"
```

### Task 2: Add Router Smoke Tests for Existing API Surface

**Files:**
- Create: `internal/httpapi/router_smoke_test.go`
- Modify: `internal/httpapi/router.go`
- Test: `internal/httpapi/router_smoke_test.go`

**Step 1: Write the failing test**

Add smoke tests that assert route registration exists (status not `404` from router dispatch; auth failures like `401` are acceptable):

```go
func TestRouterRegistersAuthLoginRoute(t *testing.T) { /* GET/POST dispatch check */ }
func TestRouterRegistersTeamsRoute(t *testing.T) { /* /api/teams */ }
func TestRouterRegistersUploadDownloadRoute(t *testing.T) { /* /api/uploads/{id} */ }
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run Router -v`
Expected: FAIL until routes are registered in the new router builder.

**Step 3: Write minimal implementation**

- Build the mux router in `internal/httpapi.NewRouter(...)` mirroring route registration from `main.go`.
- Keep handler implementations delegated to existing root functions initially via adapters (or temporary function injection).

**Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -run Router -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/httpapi/router.go internal/httpapi/router_smoke_test.go
git commit -m "test: add router smoke tests for api registration"
```

### Task 3: Extract HTTP Response Helpers Into `internal/httpapi`

**Files:**
- Modify: `internal/httpapi/response.go`
- Modify: `auth.go`
- Modify: `handlers.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Add a unit test in `internal/httpapi/response_test.go` verifying JSON success/error payload shape:

```go
func TestWriteSuccessResponseShape(t *testing.T) {}
func TestWriteErrorResponseShape(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run Write -v`
Expected: FAIL until response helpers are implemented.

**Step 3: Write minimal implementation**

- Implement `WriteJSON`, `WriteSuccess`, `WriteError` in `internal/httpapi`.
- Keep payload shape identical to current `APIResponse`.
- Provide a small compatibility layer so migrated handlers can use the new helpers without changing response contracts.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -run Write -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/httpapi/response.go internal/httpapi/response_test.go auth.go handlers.go
git commit -m "refactor: extract shared http response helpers"
```

### Task 4: Move Auth HTTP Handlers to `internal/httpapi` (Mechanical Move)

**Files:**
- Create: `internal/httpapi/handlers_auth.go`
- Modify: `internal/httpapi/router.go`
- Modify: `handlers.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Add auth endpoint smoke tests in `internal/httpapi/handlers_auth_test.go` for invalid request bodies and missing fields (expect same status codes as today).

**Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run Auth -v`
Expected: FAIL until handlers are moved/wired.

**Step 3: Write minimal implementation**

- Copy `handleRegister`, `handleLogin`, `handleLogout` into `internal/httpapi` methods on an injected handler type.
- Route `NewRouter` to the new handlers.
- Keep logic unchanged except dependency access through injected interfaces/services.
- Leave root `handlers.go` auth functions temporarily or remove once compile parity is confirmed.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -run Auth -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/httpapi/handlers_auth.go internal/httpapi/router.go internal/httpapi/handlers_auth_test.go handlers.go
git commit -m "refactor: move auth handlers into httpapi package"
```

### Task 5: Introduce Bolt Store Adapter for Users/Teams/Messages/Uploads

**Files:**
- Create: `internal/store/bolt.go`
- Create: `internal/store/bolt_users.go`
- Create: `internal/store/bolt_teams.go`
- Create: `internal/store/bolt_messages.go`
- Create: `internal/store/bolt_uploads.go`
- Modify: `database.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Add adapter-focused tests (can reuse temp Bolt DB pattern from `database_test.go`) for representative operations:

```go
func TestBoltStore_GetTeamMessagesOrdersByCreatedAt(t *testing.T) {}
func TestBoltStore_GetUserByUsername(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store -v`
Expected: FAIL until Bolt adapter is implemented.

**Step 3: Write minimal implementation**

- Create `BoltStore` struct with injected `*bbolt.DB`.
- Move/duplicate current DB logic into methods incrementally.
- Keep root `database.go` wrappers temporarily delegating to the adapter to avoid a big-bang rewrite.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/store -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/*.go database.go
git commit -m "refactor: add bolt store adapter and repository methods"
```

### Task 6: Introduce Service Layer for Auth and Teams (Thin Orchestration)

**Files:**
- Create: `internal/service/auth_service.go`
- Create: `internal/service/team_service.go`
- Modify: `internal/service/service.go`
- Modify: `internal/httpapi/handlers_auth.go`
- Modify: `internal/httpapi/handlers_team.go` (if created in next task, stage together as needed)
- Test: `go test ./internal/service -v`

**Step 1: Write the failing test**

Add service tests for representative behavior:

```go
func TestAuthService_RegisterRejectsDuplicateUsername(t *testing.T) {}
func TestTeamService_ListUsersRequiresOwner(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -v`
Expected: FAIL until service methods and interfaces are implemented.

**Step 3: Write minimal implementation**

- Define service interfaces and concrete services using store interfaces.
- Move business rules from handlers (validation/authorization orchestration) without changing outcomes.
- Keep HTTP formatting in handlers only.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/service/*.go internal/httpapi/handlers_auth.go
git commit -m "refactor: add service layer for auth and team workflows"
```

### Task 7: Move User/Team/Chat/Upload Handlers to `internal/httpapi`

**Files:**
- Create: `internal/httpapi/handlers_user.go`
- Create: `internal/httpapi/handlers_team.go`
- Create: `internal/httpapi/handlers_chat.go`
- Create: `internal/httpapi/handlers_upload.go`
- Modify: `internal/httpapi/router.go`
- Modify: `handlers.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Add endpoint-level smoke tests for representative routes in each group:

```go
func TestTeamsHandlers_ReturnsAuthFailureWithoutToken(t *testing.T) {}
func TestChatHandlers_RejectMissingBody(t *testing.T) {}
func TestUploadHandlers_RejectUnauthorizedDownload(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -v`
Expected: FAIL until all handlers are moved/wired.

**Step 3: Write minimal implementation**

- Move remaining HTTP handlers into package files grouped by domain.
- Route all endpoints in `internal/httpapi.NewRouter`.
- Inject services/store dependencies instead of using root globals directly.
- Preserve request parsing, status codes, and JSON payloads exactly.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/httpapi/*.go handlers.go
git commit -m "refactor: move remaining handlers into httpapi package"
```

### Task 8: Convert `main.go` to Composition Root Only

**Files:**
- Modify: `main.go`
- Modify: `database.go`
- Modify: `websocket.go`
- Test: `go test ./...`

**Step 1: Write the failing test**

Add a small router-construction test (or integration smoke test) in root package to ensure `main` wiring builds the app handler without panicking.

**Step 2: Run test to verify it fails**

Run: `go test ./... -run MainWiring -v`
Expected: FAIL until wiring is extracted and injectable.

**Step 3: Write minimal implementation**

- `main.go` should:
  - parse flags/config
  - open Bolt DB
  - start janitors
  - instantiate `store`, `service`, `httpapi`
  - start HTTP server
- Move route registration fully out of `main.go`.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run MainWiring -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add main.go database.go websocket.go
git commit -m "refactor: make main go a composition root"
```

### Task 9: Retire Legacy Root Handler Functions and Cleanup

**Files:**
- Delete: `handlers.go` (or reduce to compatibility shims temporarily)
- Modify: `auth.go`
- Modify: `database.go`
- Modify: `README.md` (optional follow-up if behavior docs drifted)
- Test: `go test ./...`

**Step 1: Write the failing test**

No new behavior test; rely on existing suite and smoke tests. Add compile guard by removing one legacy function reference and confirming build fails.

**Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL if any route/wiring still depends on legacy handlers.

**Step 3: Write minimal implementation**

- Remove dead root-level handler code once all routes point to `internal/httpapi`.
- Keep only shared code that is truly cross-cutting (or move it into proper packages).
- Resolve imports and cleanup comments/docs.

**Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove legacy monolithic handlers file"
```

### Task 10: Final Verification and Review Prep

**Files:**
- Modify: none required (verification only)
- Test: backend verification commands

**Step 1: Run full verification**

Run:

```bash
go test ./...
go test ./internal/httpapi -v
go test ./internal/service -v
go test ./internal/store -v
```

Expected: All PASS

**Step 2: Manual smoke checklist**

- Start server and verify:
  - `POST /api/auth/login` returns same error/success payloads
  - `GET /api/teams` still requires auth
  - `GET /api/uploads/{id}` auth behavior unchanged
  - Websocket chat endpoint still connects and broadcasts

**Step 3: Commit (if any verification-only fixes were needed)**

```bash
git add -A
git commit -m "test: finalize backend refactor verification"
```

