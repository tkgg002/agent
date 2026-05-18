# 00_context — UnitTestCdcAuthService

> **Owner**: Muscle (CC CLI, claude-opus-4-7)
> **Created**: 2026-05-14
> **Parent project**: cdc-system / cdc-auth-service

## Scope
Bổ sung bộ unit test cho service `cdc-auth-service` (Go 1.26.1, Fiber v2 + GORM + golang-jwt). Hiện trạng theo `project_context.md`: 9 .go file, **0 test** → đây là gap hạng HIGH của bucket B1.

## In-scope packages
- `config/` — `validateConfig`, `NewConfig` (pure-fn + env binding).
- `internal/service/` — `AuthService.Login`, `Register`, `RefreshToken`, `generateTokens`.
- `internal/api/` — `AuthHandler.Login`, `Register`, `RefreshToken`, `Health` (fiber.Test).
- `internal/model/` — `User.TableName`.

## Out-of-scope (deferred theo lesson L-2305)
- `internal/repository/user_repo.go` — Repository adapter qua GORM. Theo lesson L-2305, repo KHÔNG phải unit test target qua sqlmock. Defer sang integration test với real Postgres (testcontainers hoặc deploy-time E2E). Project convention "no sqlmock/no testcontainers" (lesson L-2258).
- `internal/server/server.go` — Wiring code, defer.
- `pkgs/database/postgres.go` — DB connection helper, defer.
- `cmd/server/main.go` — Entrypoint, defer.

## Constraints (project convention)
- KHÔNG thêm `sqlmock`, `testcontainers`, `mockery` (lesson L-2258, L-2305).
- Cho phép: `stretchr/testify` (đã có trong go.sum v1.11.1), `httptest`, fiber's `app.Test()`, handcrafted stubs.
- Mỗi file test 50–150 dòng, table-driven where applicable.

## Definition of Done
1. Mỗi file source trong scope có ≥1 test file tương ứng.
2. `go test ./... -count=1` PASS toàn bộ.
3. `go build ./...` PASS (không vỡ binary).
4. File `report_unittest_cdc_auth_service.md` ghi lại từng file thay đổi + output `go test` thực tế.
5. Workspace doc đầy đủ theo CLAUDE.md §7 prefix registry.
