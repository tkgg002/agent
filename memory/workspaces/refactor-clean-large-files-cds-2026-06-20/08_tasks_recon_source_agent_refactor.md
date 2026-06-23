# 08_tasks_recon_source_agent_refactor

Danh sách các bước thực thi chi tiết cho việc tái cấu trúc `recon_source_agent.go`.

- [ ] Task 1: Khởi tạo 5 file helper mới trong package `recon` (`recon_models.go`, `recon_hash.go`, `recon_query.go`, `recon_stream.go`, `recon_legacy.go`) và chuyển logic tương ứng vào đó.
- [ ] Task 2: Cập nhật rút gọn file `recon_source_agent.go` gốc, dọn dẹp các imports không còn sử dụng.
- [ ] Task 3: Biên dịch toàn bộ project bằng `go build ./...` để xác minh cú pháp và các symbol.
- [ ] Task 4: Chạy unit tests cho package `recon` bằng `go test -v ./internal/service/recon/...`.
- [ ] Task 5: Chạy unit tests cho toàn bộ project bằng `go test ./...`.
- [ ] Task 6: Thực hiện quét bảo mật tĩnh bằng `security-agent` và lưu kết quả tại `report_security_recon_refactor.md`.
- [ ] Task 7: Thống kê số dòng code thay đổi thực tế và tạo báo cáo `report_recon_source_agent_refactor.md`.
- [ ] Task 8: Cập nhật nhật ký tiến độ `05_progress.md` bằng cách append lịch sử thực thi.
