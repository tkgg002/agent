# 01_requirements — UnitTestCdcAuthService

## Yêu cầu từ User
> "unittest cho cdc-auth-service"
>
> Note:
> - Đọc lesson trước tất cả.
> - Làm theo core /agent.
> - Chỉ làm đúng những gì đc yêu cầu.
> - Report dựa trên kết quả tính toán thực tế, ko đc report láo.
> - Kết thúc luôn kiểm tra các service work mới báo done.
> - Luôn có 1 file `report_*.md` ghi lại những gì thay đổi.

## Test coverage requirements (per file)

### config/config.go
- `validateConfig`: cover 5 nhánh bắt buộc (`server.port`, `db.host`, `db.database`, `db.username`, `jwt.secret`) + nhánh production-with-placeholder + nhánh happy.
- `NewConfig`: ít nhất 1 case load file + 1 case env override + 1 case fail-validate.

### internal/service/auth_service.go
- `Login`: user not found → error; password mismatch → error; happy → trả TokenResponse có claims đúng.
- `Register`: username trùng → error; email trùng → error; role invalid → error; role rỗng → default "operator"; happy → user persisted với password đã bcrypt.
- `RefreshToken`: token invalid → error; signing method sai → error; type != "refresh" → error; user not found → error; happy → trả TokenResponse mới.
- `generateTokens`: claims có đủ user_id/username/email/role/type/iat/exp.

### internal/api/auth_handler.go
- `Login`: body invalid JSON → 400; missing field → 400; service error → 401; happy → 200 + JSON body.
- `Register`: body invalid JSON → 400; missing field → 400; service error → 409; happy → 201.
- `RefreshToken`: body invalid → 400; missing refresh_token → 400; service error → 401; happy → 200.
- `Health`: 200 với `status=ok` + `service=cdc-auth`.

### internal/model/user.go
- `TableName` trả `cdc_auth_service.auth_users`.

## Non-functional
- Sử dụng `t.Run` + table-driven.
- Test phải deterministic (không phụ thuộc thời gian thực; nếu cần — dùng injectable clock hoặc verify tolerance).
- KHÔNG mở socket / DB real trong test.
