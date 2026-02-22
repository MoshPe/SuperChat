# Backend Layered Refactor (Phase 1) Design

**Date:** 2026-02-22
**Status:** Approved

## Goal

Restructure the Go backend from large root-level files (especially `handlers.go`) into a layered architecture that separates HTTP transport, business logic, and BoltDB persistence while preserving all current API routes and response behavior.

## Current Problems

- `handlers.go` is large (`~680` lines) and mixes transport, validation, authorization, and persistence calls.
- `main.go` wires routes and startup concerns together.
- Persistence functions are global and directly coupled to handlers through package globals (`db`, `hub`).
- Refactoring any endpoint is high-risk because concerns are not isolated.

## Approved Target Architecture (Phase 1 Foundation)

- `internal/httpapi`
  - HTTP handlers
  - Router registration
  - HTTP-specific request/response concerns
  - Middleware integration and endpoint wiring
- `internal/service`
  - Business logic orchestration (auth, teams, chat, uploads)
  - Validation/authorization decisions that are not transport-specific
  - Depends on interfaces, not BoltDB directly
- `internal/store`
  - BoltDB-backed repositories / persistence adapters
  - Bucket access and CRUD operations
- `main.go`
  - Flags/config
  - Process startup
  - Dependency wiring only

Phase 1 keeps WebSocket logic operational with minimal movement; it can remain close to current code while HTTP handlers are extracted. WebSocket hub extraction into a dedicated `internal/realtime` package is deferred to a later phase.

## Dependency Direction

- `httpapi -> service`
- `service -> store` (via interfaces)
- `main -> httpapi/service/store` (composition root)

No package outside `main` should depend on root package globals.

## Phase 1 Scope

- Preserve all existing HTTP route paths and methods.
- Preserve JSON response payloads and status codes.
- Preserve authentication behavior and request contracts.
- Reduce `handlers.go` by moving handlers into `internal/httpapi`.
- Introduce service/store interfaces and move logic incrementally.
- Add router smoke tests to protect wiring during refactor.

## Non-Goals (Phase 1)

- No API contract changes.
- No feature changes.
- No database schema changes.
- No websocket protocol changes.
- No frontend changes.

## Migration Strategy (Incremental / Strangler)

1. Introduce dependency container (`App` / services / stores) and router builder.
2. Move handlers into `internal/httpapi` with behavior preserved.
3. Wrap existing persistence access in `internal/store` interfaces/adapters.
4. Extract handler business logic into `internal/service` methods in small slices.
5. Retire root-level handler functions after parity verification.

Each step should compile and pass tests independently.

## Error Handling and Testing Strategy

- Keep `writeSuccessResponse` / `writeErrorResponse` semantics identical via a shared helper or adapter.
- Add focused tests for moved logic rather than broad rewrites.
- Add router smoke tests for representative routes:
  - `/api/auth/login`
  - `/api/teams`
  - `/api/uploads/{id}`
- Run `go test ./...` after each refactor slice.

## Risks and Mitigations

- **Risk:** Behavioral drift while moving handlers.
  - **Mitigation:** Mechanical moves first, no logic edits in the same step.
- **Risk:** Package cycles during extraction.
  - **Mitigation:** Define small interfaces at the service boundary; add a shared model package only if needed.
- **Risk:** Global state (`db`, hub) leaks into new packages.
  - **Mitigation:** Inject dependencies from `main` and avoid package-level singletons in new code.

