# Report — Test Folder Reorganization (cdc-auth-service)

> **Date**: 2026-05-15 ICT
> **Owner**: Muscle (CC CLI, claude-opus-4-7)
> **Status**: ✅ DONE — build/vet/test green, coverage parity với baseline.

## 1. Bối cảnh & yêu cầu user

Sau khi REVERT các file test out-of-scope (xem `report_revert_out_of_scope_20260515.md`), user clarify yêu cầu tiếp theo:

> "ko hiểu yêu cầu, ý tao nói là vị trí đặt các file test đang đc chèn thẳng vào cấu trúc chính. cần có 1 folder riêng chứa các file test"

→ Scope **CHỈ** là refactor vị trí file (folder structure). KHÔNG mở rộng scope, KHÔNG đụng production code.

## 2. Thay đổi — file by file

### 2.1 Test files moved (4 file)

| Cũ (same-folder) | Mới (external mirror) | Package |
|---|---|---|
| `config/config_test.go` | `test/config/config_test.go` | `config_test` |
| `internal/model/user_test.go` | `test/internal/model/user_test.go` | `model_test` |
| `internal/service/auth_service_test.go` | `test/internal/service/auth_service_test.go` | `service_test` |
| `internal/api/auth_handler_test.go` | `test/internal/api/auth_handler_test.go` | `api_test` |

Source folders sau khi move: zero `*_test.go` — không còn loang.

### 2.2 Refactor đi kèm — `test/config/config_test.go`

External package KHÔNG truy cập được unexported (`validateConfig`, `defaultJWTPlaceholder`). Giải pháp:

1. **Bỏ direct call `validateConfig`**, test mọi branch qua entrypoint `config.NewConfig()`.
2. **YAML temp file pattern**:
   - `yamlCfg` struct mô tả input (port, mode, dbHost, dbDB, dbUser, jwtSec, omitDB, omitJWT).
   - `renderYAML(c)` → string YAML hợp lệ.
   - `writeTempYAML(t, body)` → ghi vào `t.TempDir()`.
   - `loadCfg(t, c)` → wrapper set `cfgPath` env + gọi `NewConfig()`.
3. **`clearAuthEnv(t)`** sạch AUTH_*/JWT_SECRET/cfgPath để tránh leak giữa test (`viper.AutomaticEnv`).
4. **`const jwtPlaceholder = "change-me-in-production"`** — bản sao literal với comment cảnh báo contract drift (source đổi value → test fail = signal đúng, không phải sticky bug).

Coverage `validateConfig` vẫn 100% (gián tiếp qua `NewConfig` → `validateConfig`).

### 2.3 Refactor — `service`, `api`, `model`

Chỉ đổi `package <name>` → `package <name>_test`. Logic giữ nguyên vì:
- `service.NewAuthService`, `service.AuthService`, `service.UserRepository` interface đều public.
- `api.NewAuthHandler`, `api.AuthHandler`, `api.AuthSvc` interface đều public.
- `model.User` exported.

Stub handcrafted (`fakeUserRepo`, `fakeAuthSvc`) implement interface qua satisfy normalize.

### 2.4 Thử nghiệm thất bại — `config/export_test.go`

Đã thử pattern re-export unexported qua `export_test.go`:
```go
// trong config/export_test.go
package config
var DefaultJWTPlaceholder = defaultJWTPlaceholder
var ValidateConfig = validateConfig
```
→ External folder test `package config_test` import `cdc-auth-service/config` báo `undefined: config.DefaultJWTPlaceholder`.

**Root cause**: `*_test.go` file chỉ compile vào TEST BINARY của FOLDER chứa nó. Test ở external folder import package qua go module path → chỉ nhìn thấy production symbols, không thấy test-only re-exports. → Rollback `export_test.go`.

### 2.5 Makefile update — `/Users/trainguyen/Documents/work/data-hub/cdc-auth-service/Makefile`

Thêm 2 target để track coverage đúng (test ở external folder → cần `-coverpkg`):

```makefile
# Run với coverage tracking về source packages (test ở ./test/... external).
# -coverpkg=./... đảm bảo coverage report tính cho config/, internal/*, pkgs/*
# thay vì chỉ tính code trong ./test/ (vốn là package _test, coverage = 0%).
test-cover:
	go test ./... -count=1 -coverpkg=./... -cover

# Coverage chi tiết per-source-package (parity check với baseline workspace).
test-cover-per-pkg:
	@for pkg in config internal/api internal/model internal/service; do \
		go test "./test/$$pkg/..." -count=1 -coverpkg="cdc-auth-service/$$pkg" 2>&1 | tail -1; \
	done
```

## 3. Verification — actual command output

### 3.1 Build & Vet
```
$ go build ./...
(exit 0) → BUILD_OK
$ go vet ./...
(exit 0) → VET_OK
```

### 3.2 Test suite
```
$ go test ./... -count=1
ok  	cdc-auth-service/test/config	1.947s
ok  	cdc-auth-service/test/internal/api	1.456s
ok  	cdc-auth-service/test/internal/model	1.026s
ok  	cdc-auth-service/test/internal/service	0.920s
(9 package khác: no test files — đúng kỳ vọng)
```

### 3.3 Coverage parity với baseline 2026-05-14
```
$ make test-cover-per-pkg
ok  cdc-auth-service/test/config           coverage: 81.8% of statements in cdc-auth-service/config
ok  cdc-auth-service/test/internal/api     coverage: 100.0% of statements in cdc-auth-service/internal/api
ok  cdc-auth-service/test/internal/model   coverage: 100.0% of statements in cdc-auth-service/internal/model
ok  cdc-auth-service/test/internal/service coverage: 90.7% of statements in cdc-auth-service/internal/service
```

| Package | Baseline (2026-05-14) | Sau reorg (2026-05-15) | Δ |
|---|---|---|---|
| config | 81.8% | 81.8% | 0.0 |
| internal/api | 100.0% | 100.0% | 0.0 |
| internal/model | 100.0% | 100.0% | 0.0 |
| internal/service | 90.7% | 90.7% | 0.0 |

→ **KHỚP HOÀN TOÀN**. Reorg không hi sinh coverage.

## 4. Files changed — full list

```
cdc-auth-service/
├── Makefile                                          (M) +2 target coverage
├── config/
│   └── config_test.go                                (D) → moved
├── internal/
│   ├── api/auth_handler_test.go                      (D) → moved
│   ├── model/user_test.go                            (D) → moved
│   └── service/auth_service_test.go                  (D) → moved
└── test/                                             (A) NEW DIR
    ├── config/config_test.go                         (A) refactored qua NewConfig entrypoint
    └── internal/
        ├── api/auth_handler_test.go                  (A) package api_test
        ├── model/user_test.go                        (A) package model_test
        └── service/auth_service_test.go              (A) package service_test
```

Source production (`*.go` không-test): **0 thay đổi**.

## 5. Convention compliance

- ✅ Tuân thủ user requirement: tách test ra folder riêng `test/` mirror structure.
- ✅ Không thêm dep mới (testify đã có sẵn).
- ✅ Không cheat DB, không sửa config production.
- ✅ Không vi phạm L-2258 (no sqlmock) / L-2305 (no repo unit test qua mock).
- ✅ Scope đúng yêu cầu user: chỉ đổi vị trí + adapt package, không expand scope.

## 6. ADRs liên quan

- ADR-005 (new) — Tách test ra folder riêng `test/` mirror source structure (external test packages).

## 7. Tham chiếu

- Audit log: `05_progress.md` entry 2026-05-15 12:55 ICT.
- Quyết định: `04_decisions.md` ADR-005.
- Baseline status: `07_status_report.md` (cập nhật trong cùng commit).
