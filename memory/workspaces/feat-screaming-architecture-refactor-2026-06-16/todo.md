# Todo List: Screaming Architecture - Functional Groups

- [x] Lập danh sách phân loại 41 file commands và 31 file queries hiện tại
- [x] Tạo các thư mục con trong `internal/app/commands/` và `internal/app/queries/`
- [x] Di chuyển các file command và query về thư mục nhóm tương ứng
- [x] Cập nhật package name trong các file đã di chuyển
- [x] Khắc phục import path lỗi trên toàn hệ thống cho commands/queries
- [x] Wire lại dependencies trong `internal/server/server.go` cho commands/queries
- [x] Kiểm tra biên dịch dự án và chạy unit test suite cho commands/queries
- [x] Phân loại 53 files trong `internal/api/` thành 7 nhóm chức năng
- [x] Tạo các thư mục con trong `internal/api/` (`source`, `shadow`, `master`, `governance`, `recon`, `scheduler`, `system`)
- [x] Di chuyển các file API vào thư mục con tương ứng
- [x] Cập nhật package name trong các file API sang package mới tương ứng
- [x] Export các helper trong `internal/api/utils.go` (`IntQuery`, `NormalizeShadowIdent`, `IsValidTimestampField`, `GetActor`, `PropIdentRe`) và cập nhật các call-site
- [x] Cập nhật các import path và alias tương ứng trong `internal/router/router.go` và `internal/server/server.go`
- [x] Kiểm tra biên dịch dự án (`go build ./...`) và chạy unit test suite (`go test ./...`)

