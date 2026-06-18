# Plan: Refactor and Drainage of DB & NATS from API and App Layers

## Các phase thực hiện:

### Phase 1: Khảo sát & Audit (Research & Triage)
- Quét toàn bộ thư mục `internal/api` và `internal/app` để tìm sự xuất hiện của `gorm.io/gorm` và NATS client (`nats`, `natsConn`).
- So sánh các câu SQL và NATS logic hiện tại ở `internal/infra` với `/Users/trainguyen/Documents/work/data-hub-bf/cdc-cms-service`.
- Ghi nhận chi tiết các file cần sửa đổi vào `08_tasks.md` và giải pháp vào `09_tasks_solution.md`.

### Phase 2: Thiết kế kỹ thuật & Giải pháp (Technical Design & Solution)
- Định nghĩa các interface (ports) cần thiết trong tầng `internal/domain` hoặc `internal/app/ports`.
- Thiết kế các repository concrete implementation trong tầng `internal/infra/persistence` hoặc `internal/infra/messaging`.
- Viết code demo chi tiết cho giải pháp trong `09_tasks_solution.md`.

### Phase 3: Thực thi (Execution)
- Trình phương án và giải pháp lên User. Sau khi có tín hiệu duyệt ("làm đi", "approve", "ok"), tiến hành sửa đổi các file code.
- Di chuyển `h.db` và client nats về `internal/infra`.
- Cập nhật wiring/composition root trong `main.go` hoặc `server.go`.

### Phase 4: Kiểm thử & Xác minh (Verification)
- Biên dịch dự án: `go build ./...`
- Chạy staticcheck / go vet: `go vet ./...`
- Chạy unit tests: `go test ./...`
- Tạo báo cáo `report_*.md` lưu lại các thay đổi và số dòng code chênh lệch.
