# 02_plan.md — Roadmap Triển khai Chuẩn hóa Schema Isolation

## Các Phase Thực hiện:
- **Phase 1: Rà soát & Chuẩn hóa Handlers (Recon / Heal / SysOps):**
  - Cập nhật payload struct: bổ sung `shadow_schema`, `shadow_table`, `source_database`, `source_table`.
  - Chuẩn hóa targetTable thành `shadow_schema.target_table` trước khi dispatch hoặc lookup.
- **Phase 2: Loại bỏ Fallback Guessing trong Metadata & Base Handlers:**
  - Chuẩn hóa `resolveTargetTableConfig` trong `recon_base_handler.go` và `helpers.go`.
  - Cập nhật `recon_smoke.go` và `recon_tier_b.go` dùng `GetByTargetTableAndSchema`.
- **Phase 3: Chuẩn hóa DB Queries trong Governance & Repo:**
  - Cập nhật `schema_validator.go`, `backfill_source_ts.go`, `recon_job_repo.go`.
- **Phase 4: Kiểm thử, Demo Code & Báo cáo:**
  - Chạy full unit tests, build worker binary.
