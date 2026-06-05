# Requirements: TriggeredBy Refactor

## English
- Centralize supported `triggered_by` values instead of scattering string literals across worker code.
- Make debug output explicit: logs should identify TriggeredBy, operation/action, batch counts, and action outcome where relevant.
- Keep existing behavior for scheduler, NATS command, and Kafka consumer paths.
- Extend Kafka consumer with a post-consume action hook that runs immediately after a consumed batch is persisted/logged.
- Add focused tests for the new TriggeredBy and Kafka post-consume behavior.
- Produce a repo-local `report_*.md` describing actual changed files and verification results.

## Tiếng Việt
- Gom các giá trị `triggered_by` được hỗ trợ vào một nơi quản lý, tránh rải string literal.
- Debug phải rõ: log nêu TriggeredBy, operation/action, số lượng batch, và kết quả action khi liên quan.
- Giữ nguyên hành vi hiện tại của scheduler, NATS command, và Kafka consumer.
- Mở rộng Kafka consumer bằng hook action chạy ngay sau khi batch Kafka được consume/persist/log.
- Thêm test tập trung cho TriggeredBy và post-consume action của Kafka.
- Tạo `report_*.md` trong repo ghi rõ file thay đổi và kết quả verify thực tế.

