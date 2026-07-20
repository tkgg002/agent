# Yêu cầu: Tích hợp Cảnh báo Đối soát (Reconciliation Alerts Integration)

## 1. Mục tiêu
Tích hợp cảnh báo đối soát thời gian thực (reconciliation drift/error alerts) vào khung giám sát sức khỏe hệ thống (System Health Monitoring Framework).

## 2. Chi tiết yêu cầu
1. **Instrument Segment A (recon_tier_a.go)**:
   - Thêm logic tự động gọi cảnh báo (`FireAlert`) khi có chênh lệch hoặc lỗi quét cho Segment A (chặng `source_shadow`).
   - Việc gọi cảnh báo này sẽ được tích hợp trực tiếp vào hàm `stampA` của `recon_engine.go` (hoặc `recon_engine_segment_a.go`) để tự động hóa đối với mọi báo cáo được lưu.
2. **Đồng bộ hóa Segment B**:
   - Chuyển logic cảnh báo của Segment B vào hàm `stampB` trong `recon_engine_segment_b.go` để gom nhóm và đảm bảo tính nhất quán (và xóa bỏ các dòng gọi `alertOnReport` thủ công dư thừa ở `recon_tier_b.go`).
3. **Cập nhật Logic ExecuteHealHandler**:
   - Khi tiến hành healing thành công (`status` của report chuyển thành `healed` hoặc `partially_healed`), hệ thống phải thực hiện giải quyết cảnh báo (`ResolveAlert`) tương ứng đối với table/segment đó.
4. **Surfacing alerts trên SystemHealth Dashboard (Frontend)**:
   - Lấy danh sách cảnh báo từ endpoint `/api/alerts/active`.
   - Hiển thị danh sách này một cách nổi bật trong màn hình System Health.
   - Thêm nút/chức năng **Acknowledge** (Xác nhận) và **Silence** (Tạm ẩn) cho từng cảnh báo để vận hành viên quản lý hiệu quả.
