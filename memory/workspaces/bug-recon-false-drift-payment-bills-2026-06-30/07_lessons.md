# Lessons Learned: Fix False Drift on Recon payment_bills / Bài học Kinh nghiệm: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

## 1. Bài học về Nghiệp vụ Đối soát (Reconciliation Domain Lessons)
* **Lệch hệ quy chiếu thời gian**: Đối soát chéo giữa hai hệ thống lưu trữ khác nhau (ví dụ: MongoDB làm Source và PostgreSQL làm Shadow) không được sử dụng metadata CDC stream timestamp (`_source_ts`) cho Tier 1 (Source vs Shadow). CDC stream timestamp dễ bị cập nhật lại (snapshot/backfill) làm thay đổi giá trị, dẫn đến chênh lệch ảo (false drift) mặc dù dữ liệu thực tế không đổi. Bắt buộc phải sử dụng cột thời gian nghiệp vụ thực tế (domain timestamp như `lastUpdatedAt` hoặc `updated_at`).
* **Đồng bộ hóa kiểu dữ liệu thời gian**: Khi đối soát giữa MongoDB (lưu epoch milliseconds) và PostgreSQL (lưu timestamp/timestamptz), phải quy đổi tường minh thời gian của Postgres sang epoch milliseconds đầu giờ:
  `((EXTRACT(EPOCH FROM date_trunc('hour', "timestamp_column")))::bigint * 1000)` để đảm bảo khớp dữ liệu hoàn hảo.

## 2. Bài học về Thiết kế & Bảo mật (Architecture & Security Lessons)
* **SQL Injection Prevention**: Mặc dù tên cột được lấy từ registry cấu hình tin cậy, khi đưa động vào câu SQL raw vẫn bắt buộc phải đi qua các hàm validation (`validateIdent`) và quote (`quoteIdent`, `quoteRelation`) để triệt tiêu mọi nguy cơ tấn công SQL Injection.
* **Tương thích ngược**: Khi thay đổi signatures các hàm Core Agent, luôn cung cấp legacy wrapper hoặc giữ cấu hình default để không làm hỏng (break) các luồng nghiệp vụ khác chưa được migrate.
