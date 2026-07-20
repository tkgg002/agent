# Danh sách Task: Sửa lỗi Quyền NATS & Timeout khi Chữa lành

## Phase 1: Phân tích & Lên kế hoạch
- [x] Đọc GEMINI.md và lessons.md.
- [x] Tạo các tài liệu yêu cầu, tiến độ, danh sách task.
- [ ] Soạn thảo kế hoạch triển khai chi tiết 12_implementation_plan_nats_permission_fix.md.

## Phase 2: Triển khai
- [ ] Cập nhật file `deployments/nats/nats-server.conf`:
  - Thêm `_INBOX.>` vào block `publish` cho `cdc_worker`.
  - Thêm `_INBOX.>` vào block `publish` cho `cms_service`.
  - Thêm `_INBOX.>` vào block `publish` cho `debezium`.
- [ ] Khởi động lại dịch vụ nats để áp dụng cấu hình mới.

## Phase 3: Hoàn thành & Nghiệm thu
- [ ] Chạy linter verify_governance.py.
- [ ] Cập nhật nhật ký tiến độ và hoàn thành walkthrough.md.
