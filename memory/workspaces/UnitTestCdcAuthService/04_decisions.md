# 04_decisions — UnitTestCdcAuthService

## ADR-001: Service & Handler depend on interface, not concrete struct
**Date**: 2026-05-14
**Decision**: Refactor `AuthService.userRepo` từ `*repository.UserRepo` → `UserRepository` (interface định nghĩa trong package `service`). Tương tự cho `AuthHandler.authSvc` → `AuthSvc` interface.
**Rationale**: Theo lesson L-2305, service layer unit test qua interface stub. Project convention L-2258 "no sqlmock". Concrete dependency = không thể inject fake.
**Impact**: Chỉ đổi field type. `*repository.UserRepo` tự động satisfy interface (đã có đủ methods). Wiring trong `server.go` không cần đổi.
**Alternatives rejected**:
- Sqlite in-memory + GORM AutoMigrate: schema name `cdc_auth_service.auth_users` không tương thích sqlite, phải override TableName (đụng source code nhiều hơn).
- Sqlmock: vi phạm convention L-2258.

## ADR-002: Bỏ 2 dòng fmt.Println debug ở AuthService.Login
**Date**: 2026-05-14
**Decision**: Xóa `fmt.Println(user)` + `fmt.Println(user.Password)` ở `auth_service.go:61,65`.
**Rationale**:
- In bcrypt-hashed password ra stdout = log leak (mặc dù đã hashed, vẫn là PII / sensitive payload).
- Trong unit test sẽ làm noisy output (mỗi case login thành công sẽ in struct user).
- Scope minimal cleanup, không thay đổi behavior public.
**Impact**: Không ảnh hưởng caller, không thay đổi response payload.

## ADR-003: SKIP unit test cho repository/user_repo.go
**Date**: 2026-05-14
**Decision**: Không viết unit test cho `UserRepo`. Document deferred sang integration test phase.
**Rationale**: Lesson L-2305 — Repository adapter qua GORM không phải unit test target qua mock library. Project convention "no sqlmock" (L-2258).
**Impact**: Coverage cho package `repository` = 0% intentionally. Trade-off chấp nhận.

## ADR-004: SKIP unit test cho server/postgres/main
**Date**: 2026-05-14
**Decision**: Không viết unit test cho `internal/server/server.go`, `pkgs/database/postgres.go`, `cmd/server/main.go`.
**Rationale**: Wiring + DB connection helper + entry point — không có pure-fn surface. Test sẽ phải mở DB thật hoặc mock toàn bộ → vi phạm convention.

## ADR-005: Tách test ra folder riêng `test/` mirror source structure (external test packages)
**Date**: 2026-05-15
**Decision**: Toàn bộ test file di chuyển từ same-folder (`config/`, `internal/api/`, ...) sang cây mirror `test/<pkg-path>/` và đổi package thành external `*_test` (vd `config_test`, `api_test`).
**Rationale**:
- User requirement rõ ràng: "vị trí đặt các file test đang đc chèn thẳng vào cấu trúc chính. cần có 1 folder riêng chứa các file test".
- External test package buộc test chỉ phụ thuộc API public → catch sớm các thay đổi breaking exported surface; tránh "test cheat" vào internals.
- Source folder gọn (chỉ production code), dev/CI dễ filter `test/...` cho coverage scoping.
**Impact**:
- Mất khả năng test trực tiếp unexported (vd `validateConfig`, `defaultJWTPlaceholder`).
- Mitigation: test through entrypoint `NewConfig()` (gián tiếp gọi `validateConfig`); sao chép literal placeholder thành const local + comment cảnh báo contract drift (test sẽ fail nếu source đổi → forcing function).
- Coverage report bị "0% per test package" nếu chạy `-cover` mặc định (vì test package `_test` không chứa logic). Mitigation: dùng `-coverpkg=./...` hoặc `-coverpkg=cdc-auth-service/<src-pkg>` (đã wire vào Makefile target `test-cover`, `test-cover-per-pkg`).
**Alternatives rejected**:
- Same-folder `*_test.go` (Go default): loang source, mất ranh giới API public/private khi test.
- `export_test.go` re-export pattern: chỉ giúp same-folder test binary, không visible cho external folder → KHÔNG áp dụng được cho cây `test/`.
**Verification**: 4 ok package, coverage parity 100% với baseline 2026-05-14 (config 81.8% | api 100.0% | model 100.0% | service 90.7%).
