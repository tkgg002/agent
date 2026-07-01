# Kế Hoạch Triển Khai Chi Tiết của AI (AI Session Implementation Plan)

## 1. Mục tiêu phiên làm việc
- Nghiên cứu và xác minh luồng đối soát Tier 2 (window-based XOR-hash) của centralized-data-service có chỉ đọc (read-only) và không ghi dữ liệu hay không.

## 2. Kế hoạch chi tiết của AI (Đã thực hiện)
- **Bước 1: Khởi động & Định vị**: Đọc `lessons.md` và `GEMINI.md` để nắm bắt các quy tắc cốt lõi của hệ thống.
- **Bước 2: Khởi tạo Workspace**: Tạo thư mục workspace `tier2-xor-hash-check` trong `work/agent/memory/workspaces/` và thiết lập các file context (`00_context.md`), yêu cầu (`01_requirements_tier2_check.md`), task list (`08_tasks_tier2_check.md`), và nhật ký tiến độ (`05_progress_tier2_check.md`).
- **Bước 3: Định vị mã nguồn**: Tìm kiếm mã nguồn thực tế của dịch vụ `centralized-data-service` trong thư mục workspace `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/`.
- **Bước 4: Phân tích mã nguồn**:
  - Đọc hàm `RunTier2` trong `recon_tier_a.go` để hiểu luồng chạy tổng quan (chia window, tính XOR-hash song song, drill-down list ID khi lệch hash, post-processing cross-check).
  - Đọc hàm `HashWindow` trong `recon_hash.go` (Source MongoDB/PostgreSQL) và `recon_dest_hash.go` (Destination Shadow DB PostgreSQL).
  - Đọc hàm `ListIDTsInWindow` trong `recon_stream.go` (Source) và `recon_dest_query.go` (Destination).
- **Bước 5: Xác minh tính chất Chỉ Đọc (Read-only)**:
  - Kiểm tra helper `readOnlyDB(ctx)` của `ReconDestAgent` trong `recon_dest_agent.go`. Xác nhận việc sử dụng `SET TRANSACTION READ ONLY` cấp độ PostgreSQL DB và rollback transaction.
  - Xác nhận không có câu lệnh INSERT, UPDATE, DELETE hay trigger heal dữ liệu nào được thực hiện trong suốt tiến trình Tier 2.
- **Bước 6: Tạo báo cáo phân tích**: Viết báo cáo chi tiết dưới dạng artifact `tier2_xor_hash_analysis.md`.
- **Bước 7: Sửa sai quy tắc ngôn ngữ**: Dịch toàn bộ các tệp tin trong workspace từ tiếng Anh sang tiếng Việt để tuân thủ nghiêm ngặt quy tắc "Luôn trả lời bằng tiếng Việt".
- **Bước 8: Đồng bộ phiên**: Tạo file `12_implementation_plan_tier2_check.md` để ghi nhận chi tiết kế hoạch triển khai của AI.
- **Bước 9: Phân tích Khóa Bảng (withTableLock)**: Xem xét cơ chế ghim connection database để giữ Postgres Advisory Lock, xác định rủi ro cạn kiệt connection pool dưới quy mô lớn (200 bảng * 50tr records) và đề xuất phương án chuyển dịch sang Redis Distributed Lock.
- **Bước 10: Xác thực ánh xạ ID và Timestamp**: Kiểm tra kiểu dữ liệu, định dạng (Epoch Ms vs Timezone), cơ chế tự động dịch Camel sang Snake case của tên cột và cơ chế fallback ObjectID của MongoDB.
- **Bước 11: Phân tích Đầu ra & Luồng Heal**: Điều tra bảng lưu trữ báo cáo (`cdc_system.cdc_reconciliation_report`) và cơ chế heal an toàn qua Debezium Signal (Segment A) và Transmute pipeline (Segment B).

