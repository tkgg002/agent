# 02_plan — UnitTestCdcAuthService

## High-level plan
Phase 1: Refactor MINIMAL để testability — thêm interface `UserRepository` trong package `service` và `AuthSvc` trong package `api`. Lý do: hiện tại `AuthService` nhận `*repository.UserRepo` concrete, không thể inject fake. Tương tự `AuthHandler` nhận `*service.AuthService`. Theo lesson L-2305: "Service layer S unit test qua interface stub". Refactor không thay đổi behavior, chỉ đổi field type từ concrete → interface (concrete impl tự động satisfy interface).

Phase 2: Viết test cho từng package:
- `config/config_test.go`
- `internal/model/user_test.go`
- `internal/service/auth_service_test.go` + helper fake repo
- `internal/api/auth_handler_test.go` + helper fake service

Phase 3: Run `go test ./... -count=1 -v` và `go build ./...`. Verify PASS. Ghi log output.

Phase 4: Tạo `report_unittest_cdc_auth_service.md` với diff thực tế.

## Refactor diff (minimal)

### service/auth_service.go
```go
// THÊM:
type UserRepository interface {
    GetByUsername(ctx context.Context, username string) (*model.User, error)
    GetByID(ctx context.Context, id uint) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    ExistsByUsername(ctx context.Context, username string) (bool, error)
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// ĐỔI:
type AuthService struct {
    userRepo UserRepository   // was: *repository.UserRepo
    cfg      *config.AppConfig
}

func NewAuthService(userRepo UserRepository, cfg *config.AppConfig) *AuthService { ... }
```
`*repository.UserRepo` đã có đủ methods để tự động satisfy interface. Không cần đổi `server.go`.

**Bonus cleanup**: bỏ 2 dòng `fmt.Println(user)` + `fmt.Println(user.Password)` ở Login — đây là debug log lộ password ra stdout, vi phạm security. Phù hợp với scope unit test vì 2 dòng này khi chạy test sẽ in ra noisy output và bản thân chúng là bug. Ghi nhận vào `04_decisions.md`.

### api/auth_handler.go
```go
// THÊM:
type AuthSvc interface {
    Login(ctx context.Context, req service.LoginRequest) (*service.TokenResponse, error)
    Register(ctx context.Context, req service.RegisterRequest) (*model.User, error)
    RefreshToken(ctx context.Context, refreshToken string) (*service.TokenResponse, error)
}

// ĐỔI:
type AuthHandler struct {
    authSvc AuthSvc   // was: *service.AuthService
}
```

## Risks & Mitigations
- **R1**: Refactor làm vỡ `server.go` wiring → Mitigate: concrete struct vẫn satisfy interface, không phải đổi caller.
- **R2**: Test phụ thuộc time → Mitigate: dùng `time.Now()` rồi check tolerance ±2s với `assert.WithinDuration` hoặc kiểm tra claim exists, không kiểm exact value.
- **R3**: Bỏ `fmt.Println` được hiểu là out-of-scope → Mitigate: ghi rõ trong `04_decisions.md` + report. Đây là security cleanup minimal, không phải feature change.

## Sequencing
1. Workspace docs (00–04, 06, 08).
2. Refactor `service/auth_service.go` + `api/auth_handler.go`.
3. `go build ./...` verify không vỡ.
4. Viết test files.
5. `go test ./... -v -count=1` lần 1 (cho thấy có thể đỏ).
6. Fix nếu fail.
7. `go test ./... -v -count=1` lần cuối (must green).
8. `go build ./...` lần cuối.
9. Write `report_unittest_cdc_auth_service.md`.
10. APPEND `05_progress.md`.
