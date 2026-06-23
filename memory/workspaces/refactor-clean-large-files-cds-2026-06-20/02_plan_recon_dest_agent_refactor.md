# Kế hoạch triển khai recon_dest_agent.go Refactor

## Quy trình 4 bước

### Bước 1: Khảo sát & Thiết kế chi tiết
- Phân tích các hàm, struct trong `recon_dest_agent.go`.
- Xác định cấu trúc phân tách file helper thích hợp nhất, ánh xạ 1-1 với cấu trúc phân tách của `recon_source_agent.go`.
- Thiết lập giải pháp mã nguồn chi tiết (Code Demo) tại `09_tasks_solution_recon_dest_agent_refactor.md`.
- Trình bày kế hoạch lên User thông qua `implementation_plan.md` ở root brain.

### Bước 2: Thực thi phân tách (Execution)
Sau khi được User phê duyệt:
- Tạo các file helper: `recon_dest_models.go`, `recon_dest_hash.go`, `recon_dest_query.go`, `recon_dest_stream.go`, `recon_dest_legacy.go`, `recon_dest_safety.go`.
- Ghi đè file core `recon_dest_agent.go` đã rút gọn.
- Kiểm tra các lỗi import, dọn dẹp các thư viện chưa sử dụng để tránh build fail.

### Bước 3: Xác minh (Verification)
- Biên dịch dự án: `go build ./...`
- Chạy unit tests: `go test -v ./internal/service/recon/...`
- Chạy toàn bộ test suite của dự án: `go test ./...`
- Chạy rà soát bảo mật qua `report_security_recon_refactor.md` (cập nhật nội dung audit cho dest agent).

### Bước 4: Báo cáo (Reporting)
- Tạo báo cáo thay đổi số dòng code `report_recon_dest_agent_refactor.md`.
- Cập nhật progress log `05_progress.md` và walkthrough của phiên.
