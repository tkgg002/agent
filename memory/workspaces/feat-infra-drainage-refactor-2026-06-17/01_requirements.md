# Requirements: Infra Drainage and Audit

## Yêu cầu chi tiết:
1. **Di chuyển GORM & NATS**:
   - Tất cả các command và query trong `internal/app` và handler trong `internal/api` không được chứa các import liên quan đến `"gorm.io/gorm"` (ngoại trừ các chỗ bắt buộc như định nghĩa kiểu transaction nếu có port hỗ trợ, nhưng tốt nhất là ẩn sau port interface).
   - Di chuyển các repository call hoặc raw query trực tiếp sử dụng `h.db` về `internal/infra/persistence`.
   - Di chuyển logic gọi NATS client (`natsConn` hoặc client nats) về `internal/infra/messaging`.

2. **Rà soát & Audit**:
   - Rà soát các file trong `internal/app`, `internal/api`, `internal/infra` của `cdc-cms-service`.
   - Audit so sánh chi tiết các câu SQL và NATS logic so với `/Users/trainguyen/Documents/work/data-hub-bf/cdc-cms-service/`.
   - Đảm bảo các logic nghiệp vụ không bị thay đổi hoặc mất mát trong quá trình refactor.

3. **Nguyên lý thực hiện**:
   - "Simplicity First, minimal impact".
   - Không thay đổi config, không cheat db.
   - Viết test hoặc verify hoạt động của service trước khi bàn giao.
