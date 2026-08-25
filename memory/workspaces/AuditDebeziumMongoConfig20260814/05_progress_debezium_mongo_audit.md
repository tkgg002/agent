# Progress Audit Log: Debezium MongoDB Source Connector Config Review

- [2026-08-14T14:47:00+07:00] [Agent:Gemini-3.6-Flash] Khởi tạo workspace AuditDebeziumMongoConfig20260814.
- [2026-08-14T14:47:00+07:00] [Agent:Gemini-3.6-Flash] Đọc GEMINI.md và lessons.md.
- [2026-08-14T14:47:00+07:00] [Agent:Gemini-3.6-Flash] Phân tích 7 bẫy Tripwires và tư vấn giải pháp chuẩn hóa Connector Config cho Production.
- [2026-08-14T14:53:00+07:00] [Agent:Gemini-3.6-Flash] Mid-session fix: Nhận phản hồi từ User. Ghi lesson mới vào lessons.md về việc tránh phán đoán giáo điều rập khuôn.
- [2026-08-14T14:55:00+07:00] [Agent:Gemini-3.6-Flash] Trực tiếp đọc mã nguồn dự án: cdc-cms-service, centralized-export-service, centralized-data-service.
- [2026-08-14T15:14:00+07:00] [Agent:Gemini-3.6-Flash] Phân tích & xác minh kiến trúc Kafka Connect Error Handling KIP-298 (Source Connector vs Sink Connector DLQ) và PII/Disk Full risk của errors.log.include.messages=true trong SourceConnectors.tsx. Ghi bài học kinh nghiệm mới vào lessons.md.
