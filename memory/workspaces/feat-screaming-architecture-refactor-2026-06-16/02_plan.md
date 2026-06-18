# Active Plan: Screaming Architecture - Functional Groups

## Kế hoạch thực hiện (High-Level Steps)
1. **Bước 1**: Phân tích và lập danh sách phân chia chi tiết các file commands và queries vào 7 nhóm chức năng: `source`, `shadow`, `master`, `governance`, `recon`, `scheduler`, `system`.
2. **Bước 2**: Tái cấu trúc các thư mục con tương ứng dưới `internal/app/commands/` và `internal/app/queries/` (Đã hoàn thành và verify test pass).
3. **Bước 3**: Tiến hành phân loại 53 files trong `internal/api/` thành 7 nhóm chức năng tương ứng:
   - `source`: Quản lý connections, connectors và source objects.
   - `shadow`: Quản lý shadow bindings, mapping rules và preview.
   - `master`: Quản lý master registry, master bindings, master mapping rules và swaps.
   - `governance`: Quản lý schema proposals, approvals, sensitive fields và audit logs.
   - `recon`: Quản lý reconciliation, backfill, heal và failed logs.
   - `scheduler`: Quản lý wizard sessions, schedules và jobs.
   - `system`: Quản lý health, alerts, introspection và action trace.
4. **Bước 4**: Tạo các thư mục con dưới `internal/api/` và di chuyển các file code api vào thư mục con tương ứng.
5. **Bước 5**: Cập nhật lại package name của các file API vừa di chuyển (ví dụ: `package source`, `package shadow`, v.v.).
6. **Bước 6**: Export các hàm helper trong `internal/api/utils.go` thành chữ in hoa (`IntQuery`, `NormalizeShadowIdent`, `IsValidTimestampField`) để các package api con có thể gọi qua `api.IntQuery`.
7. **Bước 7**: Cập nhật lại các import path và alias tương ứng trong `internal/router/router.go` and `internal/server/server.go`.
8. **Bước 8**: Biên dịch dự án (`go build ./...`) và chạy unit test suite (`go test ./...`) để xác minh tính đúng đắn của việc tái cấu trúc `internal/api`.

