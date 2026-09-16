# Progress Log: Fix Kafka Consumer Oplog UPDATE Event Processing

- [2026-08-28T08:49:00+07:00] [Agent:Gemini-3.6-Flash] [INIT] Phát hiện nguyên nhân gốc rễ luồng Kafka Consumer bỏ sót sự kiện UPDATE (`op == "u"`).
- [2026-08-28T08:49:15+07:00] [Agent:Gemini-3.6-Flash] [ANALYSIS] Phân tích 3 rào cản: (1) `kafka_consumer.go` drop `afterData == nil && opStr != "d"`, (2) Debezium Mongo Connector thiếu `change_streams_with_full_update`, (3) `event_handler.go` trả error khi `after == nil`.
- [2026-08-28T08:49:30+07:00] [Agent:Gemini-3.6-Flash] [PLAN] Khởi tạo workspace `FixKafkaConsumerOplogUpdate20260828` và lập kế hoạch sửa lỗi 3 tầng.
- [2026-08-28T09:25:00+07:00] [Agent:Gemini-3.6-Flash] [RECON_AUDIT] Phân tích sự cố Recon báo trùng ID giữa "Thiếu ở Shadow" và "Thừa ở Shadow": Xảy ra do mốc thời gian so sánh bị lệch giữa `lastUpdatedAt` (Source) và `_updated_at` (Shadow) do batch transform làm trôi window.
- [2026-08-28T10:20:00+07:00] [Agent:Gemini-3.6-Flash] [EXECUTE] Triển khai fix `ColumnExists` và `GetRealColumnName` (case-insensitive column match) trong `recon_dest_query.go`, `recon_tier_a.go`, `recon_stream_bucket_engine.go`.
- [2026-08-28T10:20:15+07:00] [Agent:Gemini-3.6-Flash] [CONFIG] Cập nhật `cdc_system.sources` trong `cdc_dw` với `"capture.mode": "change_streams_with_full_update"` cho 3 MongoDB connectors.
- [2026-08-28T10:28:30+07:00] [Agent:Gemini-3.6-Flash] [VERIFY] Chạy `go test ./internal/service/recon/...` và `go test ./internal/handler/shadow/...` - PASS 100%.
- [2026-08-28T10:55:00+07:00] [Agent:Gemini-3.6-Flash] [AUDIT_V1] Tạo `audit_report_fix_oplog_update.md` tuyên bố "fix thành công" dựa trên smoke recon `drift:0`.
- [2026-08-28T13:24:00+07:00] [Agent:Claude-Opus-4.6] [FAILURE_DETECTED] User test lại → vẫn thấy 77 ID "Thiếu ở Shadow" + 76 ID "Thừa ở Shadow". Audit V1 đã SAI — smoke `drift:0` là false positive do tolerance mechanism.
- [2026-08-28T13:25:00+07:00] [Agent:Claude-Opus-4.6] [MID_SESSION_FIX] Áp dụng Rule #5. Dừng lại, thừa nhận sai, ghi lesson mới, bắt đầu điều tra lại root cause từ đầu.
- [2026-08-28T13:40:00+07:00] [Agent:Claude-Opus-4.6] [DEEP_INVESTIGATION] Phân tích code path: smoke recon (`recon_smoke.go`) vs manual job (`recon_stream_bucket_engine.go`). Phát hiện smoke dùng window rộng 2h có tolerance, manual job dùng sub-window 15 phút không có tolerance.
- [2026-08-28T13:45:00+07:00] [Agent:Claude-Opus-4.6] [DB_EVIDENCE] Query `cdc_system.recon_jobs` xác nhận: payment_bills_1 có source_count=964=dest_count nhưng vẫn báo 76 phantom drift. `mismatched: null`. 76 ID xuất hiện ở CẢ 2 danh sách.
- [2026-08-28T13:48:00+07:00] [Agent:Claude-Opus-4.6] [ROOT_CAUSE_CONFIRMED] Nguyên nhân thực sự: MongoDB `lastUpdatedAt` đã được UPDATE nhưng shadow giữ giá trị cũ (CDC missed UPDATEs). Cùng ID rơi vào sub-window khác nhau → phantom drift. capture.mode fix đúng cho tương lai nhưng chưa re-sync data cũ.
- [2026-08-28T13:55:00+07:00] [Agent:Claude-Opus-4.6] [AUDIT_V2] Tạo `audit_report_recon_deep_investigation.md` với bằng chứng DB thực tế. Ghi lesson mới vào `lessons.md`.
- [2026-08-28T13:55:30+07:00] [Agent:Claude-Opus-4.6] [PENDING] Cần: (1) Chạy Bridge/Full-Sync để re-sync 76 bản ghi stale + bổ sung ID 49933. (2) Cải tiến Recon algorithm deduplicate phantom drift.

