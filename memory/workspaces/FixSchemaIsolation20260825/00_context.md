# 00_context.md — Multi-Service Schema Isolation & Zero Bare Table Lookup

## 1. Scope & Goals
- Chuẩn hóa toàn bộ hệ thống CDC / Recon / Governance để **BẢO VỆ TUYỆT ĐỐI SCHEMA ISOLATION** giữa các microservices có cùng tên bảng nguồn/đích (ví dụ: `bidv-connector-service.bank_requests` vs `bvb-connector-service.bank_requests`).
- **Loại bỏ 100% việc lookup theo tên bảng trần (`pureTable`) hoặc fallback đoán mò**.
- Đảm bảo mọi luồng (API Request, NATS Event, Service Lookup, SQL Queries) đều sử dụng cặp định danh đầy đủ `(shadow_schema, target_table)` hoặc qualified key `shadow_schema.target_table`.

## 2. Affected Components
- `centralized-data-service/internal/handler/recon`:
  - `recon_check_handler.go`
  - `recon_check_heal_handler.go`
  - `recon_execute_heal_handler.go`
  - `recon_sysops_handler.go`
  - `recon_base_handler.go`
- `centralized-data-service/internal/service/recon`:
  - `recon_job_worker.go`
  - `recon_smoke.go`
  - `recon_tier_b.go`
- `centralized-data-service/internal/service/metadata`:
  - `helpers.go`
- `centralized-data-service/internal/service/governance`:
  - `schema_validator.go`
  - `backfill_source_ts.go`
- `centralized-data-service/internal/repository`:
  - `recon_job_repo.go`
