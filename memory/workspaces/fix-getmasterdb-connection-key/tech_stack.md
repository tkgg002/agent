# tech_stack.md — Công nghệ & Quy chuẩn kỹ thuật

## Tech Stack
- **Language:** Go 1.22+
- **Database:** PostgreSQL 15+ (GORM, PGX Pool)
- **Message Bus:** NATS JetStream / Core NATS
- **Framework:** Fiber v2 (CMS Web API)
- **Telemetry:** OpenTelemetry, Zap Logger

## Nguyên tắc cốt lõi (Core Principles)
- **FQN Identifier:** Mọi bảng Master phải được định danh dạng `<master_schema>.<master_table>`.
- **Postgres Concat Safety:** Luôn dùng `COALESCE(NULLIF(schema, ''), 'public')` khi nối chuỗi với `||`.
- **Anti-DB Cheat:** Tuyệt đối không can thiệp thủ công sửa bảng state `cdc_system.sync_runtime_state`.
