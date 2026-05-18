# report_unittest_cdc_auth_service.md

> **Task**: Bổ sung unit test cho `cdc-auth-service`.
> **Date**: 2026-05-14 (ICT)
> **Agent**: Muscle (CC CLI, claude-opus-4-7)
> **Workspace**: `agent/memory/workspaces/UnitTestCdcAuthService/`

---

## 1. Mục tiêu
Service `cdc-auth-service` (Go 1.26.1, Fiber v2 + GORM + golang-jwt) trước đó có **0 test** — gap HIGH theo `project_context.md` (B1 hygiene bucket). Yêu cầu của user: viết unit test, tôn trọng project convention, không report láo.

## 2. Lessons đã consult (ràng buộc thiết kế)
- **L-2258** (no sqlmock / no testcontainers): codebase chốt convention validation/shape tests only — KHÔNG được thêm `sqlmock`, `testcontainers`, `mockery` silently. Cover qua pure-fn + httptest + handcrafted stubs.
- **L-2305** (Repository ≠ unit test target via mock): repo adapter qua GORM defer sang integration test phase, service test qua interface stub.

## 3. Thay đổi source code (refactor minimal)

| File | Loại | Mục đích |
|---|---|---|
| `internal/service/auth_service.go` | MODIFY | Thêm interface `UserRepository`; đổi field `userRepo` từ `*repository.UserRepo` → `UserRepository`; bỏ import `cdc-auth-service/internal/repository` (không còn cần vì đã dùng interface); xóa 2 dòng `fmt.Println(user)` + `fmt.Println(user.Password)` trong `Login` (ADR-002 — đây là debug log lộ hashed password ra stdout). |
| `internal/api/auth_handler.go` | MODIFY | Thêm interface `AuthSvc` + import `context` / `cdc-auth-service/internal/model`; đổi field `authSvc` từ `*service.AuthService` → `AuthSvc`. |
| `go.mod` | MODIFY | `go mod tidy` thêm `github.com/stretchr/testify v1.11.1` (đã có sẵn trong go.sum, chỉ là viper transitive — giờ là direct dep), + 2 indirect `davecgh/go-spew`, `pmezard/go-difflib`. **Không thêm sqlmock/testcontainers.** |

Wiring trong `internal/server/server.go` không thay đổi (concrete `*repository.UserRepo` và `*service.AuthService` tự động satisfy interface). Verify bằng `go build ./...` → PASS.

## 4. File test mới (4 file)

| File | Số test (parent / sub) | Cover |
|---|---|---|
| `config/config_test.go` | 5 / 14 | validateConfig table-driven 9 case + NewConfig 4 case (load YAML, env override, fail validate, missing file) |
| `internal/model/user_test.go` | 1 / 1 | `(User).TableName()` |
| `internal/service/auth_service_test.go` | 4 / 18 | Login (3 case) + Register (6 case) + RefreshToken (5 case) + generateTokens claim shape (1 case) — dùng `fakeUserRepo` handcrafted stub |
| `internal/api/auth_handler_test.go` | 4 / 16 | Login (4) + Register (4) + RefreshToken (4) + Health (1) — dùng `fakeAuthSvc` handcrafted stub + `fiber.App.Test()` |

**Tổng**: 14 + 1 + 18 + 16 = **49 sub-tests**.

## 5. Kết quả test thực tế (verified by running)

### 5.1 `go test ./... -count=1` (final run)
```
?   	cdc-auth-service/cmd/server	[no test files]
ok  	cdc-auth-service/config	0.224s
?   	cdc-auth-service/docs	[no test files]
ok  	cdc-auth-service/internal/api	0.379s
ok  	cdc-auth-service/internal/model	0.536s
?   	cdc-auth-service/internal/repository	[no test files]
?   	cdc-auth-service/internal/server	[no test files]
ok  	cdc-auth-service/internal/service	0.965s
?   	cdc-auth-service/pkgs/database	[no test files]
```
→ **0 FAIL**, 4 package ok.

### 5.2 `go test ./... -count=1 -cover`
```
	cdc-auth-service/cmd/server		coverage: 0.0% of statements
ok  	cdc-auth-service/config	0.427s	coverage: 81.8% of statements
	cdc-auth-service/docs		coverage: 0.0% of statements
ok  	cdc-auth-service/internal/api	1.252s	coverage: 100.0% of statements
ok  	cdc-auth-service/internal/model	0.831s	coverage: 100.0% of statements
	cdc-auth-service/internal/repository		coverage: 0.0% of statements
	cdc-auth-service/internal/server		coverage: 0.0% of statements
ok  	cdc-auth-service/internal/service	1.943s	coverage: 90.7% of statements
	cdc-auth-service/pkgs/database		coverage: 0.0% of statements
```

### 5.3 `go build ./...`
→ exit 0, không output → BUILD OK.

### 5.4 `go vet ./...`
→ exit 0, không output → VET OK.

## 6. Package được intentionally SKIP (kèm lý do)

| Package | Lý do |
|---|---|
| `internal/repository` | Lesson L-2305: GORM adapter không phải unit test target via mock. Project convention L-2258 không cho phép sqlmock. Defer sang integration test với real Postgres. |
| `internal/server` | Wiring code (DI), không có pure-fn surface. Cover bằng E2E khi dựng service. |
| `pkgs/database` | Helper mở `*gorm.DB` connection thật; không thể test mà không có DB. |
| `cmd/server` | Main entrypoint, không có logic. |
| `docs` | File swagger generated. |

## 7. Tóm tắt files đã thay đổi (full path)
- **Source modified**:
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/internal/service/auth_service.go`
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/internal/api/auth_handler.go`
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/go.mod` + `go.sum` (qua `go mod tidy`)
- **Test files added**:
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/config/config_test.go`
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/internal/model/user_test.go`
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/internal/service/auth_service_test.go`
  - `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/internal/api/auth_handler_test.go`
- **Workspace docs (agent/memory)**:
  - `agent/memory/workspaces/UnitTestCdcAuthService/00_context.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/01_requirements.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/02_plan.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/04_decisions.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/05_progress.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/06_test_cases.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/07_status_report.md`
  - `agent/memory/workspaces/UnitTestCdcAuthService/report_unittest_cdc_auth_service.md` (file này)

## 8. Verification checklist (CLAUDE.md §3 — Plan & Verify)
- [x] `go test ./... -count=1` PASS — verified bằng output thực tế ở §5.1.
- [x] `go build ./...` PASS — §5.3.
- [x] `go vet ./...` PASS — §5.4.
- [x] Convention check: không thêm sqlmock/testcontainers — go.sum diff verified, chỉ thêm `testify` (đã có), `davecgh/go-spew`, `pmezard/go-difflib`.
- [x] Workspace docs đầy đủ prefix 00/01/02/04/05/06/07 + report — đã tạo file vật lý.
- [x] 05_progress.md APPEND ONLY (không overwrite).

## 9. Khuyến nghị tiếp theo (out-of-scope, không thi công ở task này)
1. Integration test cho `internal/repository/user_repo.go` qua deploy-time E2E (real Postgres 5432) hoặc khi nào Brain approve testcontainers (cần RFC).
2. Khi nào có CI: add step `make test` vào pipeline.
3. Có thể bổ sung benchmark cho bcrypt cost trong `auth_service` nếu thấy `Register` chậm trong test (hiện 0.25s cho 6 sub-cases là chấp nhận được với MinCost trong test).
