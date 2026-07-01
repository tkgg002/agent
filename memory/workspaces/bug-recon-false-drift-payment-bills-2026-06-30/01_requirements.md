# Requirements: Fix False Drift on Recon payment_bills / Yêu cầu: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

## 1. Yêu cầu của User (User Request)
* Khắc phục hiện tượng đối soát báo lệch ảo 1.410 bản ghi trên bảng `payment_bills` giữa MongoDB Source và Postgres Shadow.
* Xác định rõ nguyên nhân tại sao MongoDB không lệch thực tế nhưng hệ thống vẫn báo đỏ.
* Cần kiểm tra chéo và sửa triệt để từ lõi (core systems), không chấp nhận các giải pháp vá lỗi tạm thời ở tầng giao diện (UI workarounds).

## 2. Tiêu chuẩn Nghiệm thu (Definition of Done - DoD)
* **Chính xác hệ quy chiếu**: destination agent (`ReconDestAgent`) và các phương thức query/hash phải hỗ trợ cấu hình động cột thời gian (domain timestamp) được resolve từ source mapping config (`TimestampField`), thay vì fix cứng `_source_ts` ở Tier 1.
* **Đồng bộ các Tiers**:
  * Tier 1 (Source vs Shadow): Lọc window và tính toán fingerprint dựa theo domain timestamp (`lastUpdatedAt`).
  * Tier 2 (Shadow vs Master): Giữ nguyên lọc theo metadata CDC timestamp `_source_ts` (để so sánh stream time).
* **Unit Tests**:
  * Viết thêm unit tests chứng minh `ReconDestAgent` hoạt động đúng đắn khi sử dụng domain timestamp (`time.Time`) lẫn default `_source_ts`.
  * Bộ test suite của gói `recon` phải PASS 100%.
* **Đảm bảo không hồi quy (No Regression)**: Dự án build thành công và chạy ổn định.
