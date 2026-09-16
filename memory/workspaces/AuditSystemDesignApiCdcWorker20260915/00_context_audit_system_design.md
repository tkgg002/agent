# 00_context_audit_system_design.md

## Scope & Context
- **Task:** Đọc System Design, Design Patterns của API (`cdc-cms-service`) và CDC Worker (`centralized-data-service`).
- **Mục tiêu:** Thống kê hoàn chỉnh cấu trúc hệ thống, kiến trúc tầng (Hexagonal / Screaming Architecture / DDD), các design patterns (Repository, Factory, Strategy, Worker Pool, Pipeline, Event-Driven/NATS, Middleware, CQRS), flow dữ liệu, và mối liên kết giữa API Control Plane (`cdc-cms-service`) và CDC Engine (`centralized-data-service`).
- **Phạm vi khảo sát:**
  - `cdc-cms-service` (API Control Plane / Management CMS)
  - `centralized-data-service` (CDC Engine / Worker / Transmuter / Recon / Pipeline)
