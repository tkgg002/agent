## Task: Refactor Admin Helpers and Migrate Tests
- **Phase**: GĐ2
- **Service Group**: Utilities
- **Service(s)**: centralized-data-service/internal/admin
- **Mô tả**: Tái cấu trúc file `internal/admin/helpers.go` theo yêu cầu của User và di chuyển các file test tương ứng sang package `admin` trong `internal/admin/` để truy cập các hàm private trực tiếp.
- **Trạng thái**: [ ] TODO

### [Context]
- Current state: Đã được User approve kế hoạch triển khai.
- Dependencies: `internal/admin/helpers.go`, `test/internal/admin/server_test.go`, `test/internal/admin/main_test.go`.
- ADR liên quan: N/A.
- Logs/Error: N/A.

### [Definition of Done]
- [ ] 1. Lưu đè mã nguồn mới do User cung cấp vào `internal/admin/helpers.go`.
- [ ] 2. Di chuyển `server_test.go` và `main_test.go` từ `test/internal/admin/` sang `internal/admin/`.
- [ ] 3. Xóa các file test cũ ở `test/internal/admin/`.
- [ ] 4. Đổi package của hai file test mới sang `package admin`.
- [ ] 5. Loại bỏ import `centralized-data-service/internal/admin` trong các file test mới.
- [ ] 6. Thay thế các cuộc gọi `admin.ContainsCSVForTest` thành `containsCSV`, `admin.ShadowSchemaForForTest` thành `shadowSchemaFor`, `admin.TopicNameForForTest` thành `topicNameFor`, `admin.ExtendDatabaseListForTest` thành `extendDatabaseList`, và `admin.ExtendConfigInMemoryForTest` thành `extendConfigInMemory`.
- [ ] 7. Đảm bảo toàn bộ dự án biên dịch thành công (`go build ./...`).
- [ ] 8. Đảm bảo unit test chạy qua (`go test ./internal/admin/...`).
- [ ] 9. Đảm bảo toàn bộ suite test chạy qua (`go test ./...`).
- [ ] **[QA Gate]**: Run test coverage và verify test pass.
- [ ] **[Security Gate]**: Chạy rà soát hoặc verify không phát sinh lỗi bảo mật mới.
- [ ] Blast radius verified.
- [ ] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.
