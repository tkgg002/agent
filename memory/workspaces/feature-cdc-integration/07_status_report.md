# Status Report: Feature CDC Integration (2026-04-06)

## 1. Tiến độ hiện tại (Current Progress)

Hệ thống đã hoàn thành khoảng **75% Phase 1**. Các thành phần cốt lõi đã được xây dựng nhưng còn thiếu "mắt xích" cuối cùng để đóng vòng lặp (Closed-loop) tự động.

### 🟢 Đã hoàn thành (Done)
- **Airbyte OAuth2 Client**: Worker đã có khả năng gọi API Discover Schema của Airbyte (Source-first).
- **Database Migration**: Schema cho `pending_fields`, `mapping_rules`, `schema_change_logs` đã sẵn sàng.
- **CMS Approval Service**: Logic xử lý `Approve/Reject` đã hoàn thiện (xử lý giao dịch ALTER TABLE + Mapping Rule + NATS Publish).
- **Schema Inspector**: Worker đã có logic `HandleIntrospect` để quét schema từ Airbyte và ghi nhận vào `pending_fields`.

### 🟡 Đang dở dang (In Progress)
- **CMS Frontend**: Giao diện quản trị duyệt schema drift chưa được verify đầy đủ.
- **NATS Command Pattern**: Hầu hết các lệnh đã chạy, nhưng thiếu lệnh reload cấu hình.

### 🔴 Lỗ hổng cần xử lý (Gaps)
- **Missing Reload Subscriber**: Worker (`centralized-data-service`) chưa lắng nghe subject `schema.config.reload` từ CMS. Nghĩa là sau khi bạn duyệt trên CMS, Worker vẫn dùng mapping cũ cho đến khi được restart manual.
- **Integration Validation**: Chưa có quy trình chạy thử full-flow từ lúc MongoDB thêm field đến lúc Postgres nhận được data của field đó.
