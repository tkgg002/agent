# Plan: Reconcile Component Overhaul

## Các Phase thực hiện

### Phase 1: Phân tích hiện trạng & Rà soát mã nguồn (Research & Audit)
- **Mục tiêu**: Nắm rõ cách hệ thống hiện tại đang ghi và truy vấn `cdc_reconciliation_report`.
- **Hành động**:
  1. Phân tích cách `recon_tier_a.go` và `recon_tier_b.go` khởi tạo và lưu trữ report (`stampA`, `stampB`).
  2. Rà soát các câu lệnh SQL migration và cách cdc-cms-service truy vấn bảng này để hiển thị trên UI.
  3. Tổng hợp danh sách các điểm hạn chế (Gaps) của cấu trúc và logic hiện tại.

### Phase 2: Thiết kế giải pháp kiến trúc (Redesign & Architecture)
- **Mục tiêu**: Xây dựng cấu trúc DB tối ưu và logic ghi report thông minh.
- **Hành động**:
  1. Thiết kế lại bảng `cdc_reconciliation_report` (Unified Schema) gọn gàng, có chỉ mục (indices) rõ ràng.
  2. Thiết kế logic ghi report thông minh (Smart Write / Deduplication): Chỉ ghi đè/chèn mới khi có thay đổi trạng thái hoặc drift.
  3. Thiết kế cơ chế Pruning tự động để quét sạch rác `ok` cũ định kỳ.
  4. Viết tài liệu kỹ thuật chi tiết (`03_implementation_reconcile_overhaul.md`) và mã nguồn demo cụ thể.

### Phase 3: Review & Phê duyệt (User Approval Gate)
- **Mục tiêu**: Đảm bảo kế hoạch và thiết kế đáp ứng hoàn hảo các yêu cầu của User trước khi sửa code.
- **Hành động**:
  1. Trình bày thiết kế giải pháp chi tiết cho User.
  2. Dừng chờ User duyệt hoàn toàn.

### Phase 4: Triển khai & Kiểm thử (Execution & Verification)
- **Mục tiêu**: Thực thi thiết kế và xác thực độ tin cậy.
- **Hành động**:
  1. Giao Muscle sửa đổi mã nguồn và chạy các câu lệnh migration/test.
  2. Xác minh hoạt động của toàn bộ hệ thống smoke test và integration test.
  3. Cập nhật nhật ký bàn giao (`report_*.md`).
