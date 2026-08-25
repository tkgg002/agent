# 05_progress.md — Nhật ký tiến độ (Audit Log - Append ONLY)

- **[2026-08-20 13:27:00] [Brain:Gemini-3.7-Flash] Khởi tạo phân tích sự cố Snapshot bank_requests dừng tại mốc 5,125,000 / 12,614,888.**
  - Phân tích thông báo `Heartbeat timeout: progress was stuck in running state for too long (worker stopped)`.
  - Xác định vị trí sinh lỗi tại `snapshot_progress_read_repo_gorm.go` khi `updated_at < NOW() - INTERVAL '5 minutes'`.
- **[2026-08-20 13:33:00] [Brain:Gemini-3.7-Flash] Phân tích trace log connection refused.**
  - Truy vết `failed to connect to user=cdc-shadow-user database=cdc_shadow: 10.200.185.20:5432 (10.200.185.20): dial error: dial tcp 10.200.185.20:5432: connect: connection refused`.
  - Khớp nối sự cố: PostgreSQL bị crash/restart do Disk I/O bão hòa 95% dẫn tới ngắt kết nối worker, gây ra Heartbeat timeout trên CMS.
- **[2026-08-20 13:40:00] [Brain:Gemini-3.7-Flash] Đề xuất các giải pháp hạ tải Disk I/O.**
  - Phân tích cơ chế `snapshot_max_rps` và `snapshot_batch_size` có sẵn trong CDS.
  - Phân tích cơ chế WAL & Checkpoint tuning cho PostgreSQL.
- **[2026-08-20 14:05:00] [Brain:Gemini-3.7-Flash] Kiểm chứng mã nguồn `snapshot_max_rps`.**
  - Xác nhận 100% logic đã tồn tại trong `snapshot_runner_handler.go:L880-L886` với công thức `time.Sleep(expectedDuration - elapsed)`.
  - Tiếp thu phản biện của User: loại bỏ phương án Manual Pause/Resume, chuyển hoàn toàn sang Tự động hoá qua Rate Limiting.
- **[2026-08-20 14:15:00] [Brain:Gemini-3.7-Flash] Phân tích lỗ hổng trên CMS Modal.**
  - Phát hiện Modal "Chỉnh sửa Source Object" trên `TableRegistry.tsx` và API Backend `cdc-cms-service` chưa hỗ trợ trường `snapshot_max_rps`.
  - Lập Implementation Plan chi tiết cho Frontend và Backend.
- **[2026-08-20 14:24:00] [Brain:Gemini-3.7-Flash] Khởi tạo Workspace `fix-snapshot-rps-and-disk-throttle` theo chuẩn Rule #4 & Rule #5.**
  - Khởi tạo đầy đủ bộ 13 file tài liệu chuẩn.
  - Ghi nhận bài học vi phạm quy trình Governance vào `lessons.md`.
- **[2026-08-20 14:40:00] [Brain:Gemini-3.7-Flash] Phân tích lỗi Trace ID khi Resume snapshot không cập nhật trên UI và không kế thừa Parent Trace.**
  - Xác định vị trí 1: `snapshot_runner_state.go:L44-L47` trong `claimProgress` không UPDATE cột `trace_id` khi resume.
  - Xác định vị trí 2: `snapshot_progress_handler.go:L82-L113` khi bắn NATS không truyền lại Parent Trace ID gốc của snapshot, khiến Otel tạo Trace ID mới độc lập thay vì gắn vào cùng 1 trace tree trên SigNoz.
  - Thiết kế giải pháp sửa đổi cho cả CDS và CMS.
- **[2026-08-20 15:00:00] [Brain:Gemini-3.7-Flash] Phân tích cấu trúc Spans và Độ trễ 10s-20s của bank_requests trên SigNoz.**
  - Giải thích hiện tượng Child Spans của bảng khác: Do `BatchBuffer` là bộ đệm chia sẻ (Shared Buffer), khi Snapshot gọi `FlushBatchBuffer(ctx)`, toàn bộ bản ghi CDC streaming (payments, wallets, payment_bills) đang nằm trong buffer được flush chung dưới Trace Context của Snapshot.
  - Giải thích độ trễ 10s-20s: Do nghẽn I/O đĩa PostgreSQL 95% + Forced Checkpoints + B-Tree Index buffer cache miss trên bảng 5M+ records + phân mảnh signature grouping.
- **[2026-08-21 13:37:00] [Brain:Gemini-3.7-Flash] Thực hiện toàn diện quy trình Audit & Phản tỉnh (Self-Improvement Loop) và xuất báo cáo `audit_report_snapshot_rps_trace_fix.md`.**
  - Rà soát phản biện từng dòng code: Phát hiện và vá gap trong `Validate()` của `update_source_object_v2.go`.
  - Fact-check 100% các kết luận về mã nguồn và cơ chế hạ tầng, xác nhận không có suy diễn hay báo cáo láo.
  - Đánh giá đạt chuẩn 8 Quality Gates DoD (G1–G8).
- **[2026-08-24 09:58:00] [Muscle:Gemini-3.7-Flash] Triển khai toàn bộ các thay đổi mã nguồn sau khi nhận lệnh APPROVE từ User.**
  - Đã cập nhật 5 file Backend: `source_objects_read_models.go`, `source_object_read_repo_gorm.go`, `update_source_object_v2.go`, `source_object_actions_handler.go`, `snapshot_progress_handler.go`.
  - Đã cập nhật 2 file Frontend: `types/index.ts`, `TableRegistry.tsx` (Thêm InputNumber `Snapshot Max RPS`, hỗ trợ xóa trắng = clear về NULL).
  - Đã cập nhật 1 file Worker: `snapshot_runner_state.go` (Cập nhật `trace_id` mới và xóa `error_msg` cũ khi resume).
  - Hoàn tất 100% các đầu việc trong `08_tasks.md`.
- **[2026-08-24 13:12:00] [Brain:Gemini-3.7-Flash] Thực hiện toàn diện quy trình Post-Execution Audit & Phản tỉnh, xuất bản `audit_report_final_post_execution.md`.**
  - Rà soát 100% dòng code thực tế trên 8 file đã chỉnh sửa, đối chiếu với CQRS architecture và database migration.
  - Fact-check toàn diện, xác nhận không có suy diễn, không báo cáo khống.
  - Hoàn tất bàn giao vận hành cho Operator.





