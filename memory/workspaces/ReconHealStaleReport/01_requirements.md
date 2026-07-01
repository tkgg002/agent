# Requirements: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## 1. Yêu cầu nghiệp vụ (Business Requirements)
- Hệ thống không được phép gửi tín hiệu hoặc lệnh trigger heal dư thừa cho các ID đã được đồng bộ thông qua CDC bình thường.
- Đảm bảo dữ liệu đối soát dùng để heal là dữ liệu tươi mới (fresh), phản ánh đúng hiện trạng thực tế của database tại thời điểm click/trigger heal.

## 2. Yêu cầu kỹ thuật (Technical Requirements)
- Định nghĩa ngưỡng thời gian tối đa để tái sử dụng một báo cáo đối soát cũ: `healReportMaxAge = 5 * time.Minute`.
- Nếu báo cáo mới nhất trong DB (`cdc_system.cdc_reconciliation_report`) có `healed_at IS NULL` nhưng khoảng cách từ lúc chạy đối soát (`checked_at`) đến hiện tại lớn hơn `healReportMaxAge`:
  - Hệ thống phải bỏ qua báo cáo này (coi là stale).
  - Tự động trigger một phiên chạy đối soát mới (`RunTier2` cho Segment A, `RunSegmentBFor` cho Segment B) để lấy danh sách lệch mới nhất.
- Viết unit test tự động để chứng minh luồng stale report fallback hoạt động chính xác (không sử dụng báo cáo quá hạn, tự động gọi hàm đối soát).
- Toàn bộ thay đổi phải tương thích ngược và không phá vỡ cấu trúc của hệ thống Recon/Heal hiện tại.
