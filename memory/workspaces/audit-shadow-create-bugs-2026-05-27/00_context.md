# 00_context — Audit Shadow Table Create Bugs (2026-05-27)

## Scope
Audit-only (no code change yet). Hai bug user phát hiện khi thao tác trên FE `http://localhost:5173/shadow`:

1. **B1 — Auto-leak field từ source cũ**: Tạo `sd_export_jobs_1` (shadow mới, chưa cấu hình gì) → form/server đã móc field từ `dbsource → export_jobs` (entity đã đăng ký từ trước) gắn vào shadow mới.
2. **B2 — `_source_ts` không được tạo ở shadow table**: Cột `_source_ts` là **system metadata bắt buộc** (OCC anchor). Shadow mới CREATE thiếu cột này.

## Why "khủng khiếp"
- B1: vi phạm isolation giữa các shadow entity. Người tạo shadow mới có thể thấy / kế thừa field của shadow khác chỉ vì tên gần giống → spaghetti binding, không thể audit nguồn dữ liệu.
- B2: `_source_ts` là gốc của OCC + ordering (xem L-three-layer-trust, L-1037, project_context.md §Domain Knowledge). Thiếu nó → mọi insert/update đi qua `_source_ts older` guard sẽ behave undefined: hoặc crash khi reference cột không tồn tại, hoặc transmute bypass OCC silently → data race / duplicate / overwrite mới-bằng-cũ.

## Bối cảnh service
- FE: `cdc-cms-web/src/pages/...` (route `/shadow`)
- API: `cdc-cms-service` (control plane) — endpoint POST/GET shadow tables, validate, ghi `cdc_system.shadow_binding`
- Worker: `centralized-data-service` — handler shadow_bind / shadow_automator → CREATE TABLE physical (PG shadow DB 5436 path B, hoặc 5433 cdc_dw path A hybrid)

## System columns chuẩn của Shadow (per project_context.md)
Mọi shadow table BẮT BUỘC có:
- `_gpay_source_id` (V2 anchor UNIQUE, ON CONFLICT key cho master)
- `_raw_data` (JSONB raw từ Debezium)
- `_source_ts` (BIGINT/TIMESTAMP — OCC older-wins anchor)
- `_synced_at` (server timestamp)
- `_version` (monotonic event version)
- `_hash` (content hash cho recon)
- `_gpay_deleted` (BOOLEAN tombstone)

Thêm các col business mirror từ source (per shadow_bind handler) — không phải clone từ entity khác.

## Definition of Done (cho audit)
- Xác định file:line nơi B1 leak xảy ra (FE form prefill HOẶC BE handler).
- Xác định file:line nơi B2 build column list thiếu `_source_ts`.
- Đề xuất fix elegant (Demand Elegance §6 GEMINI) — không cheat DB, không workaround.
- Verify build sạch ở các service liên quan.
- Report `report_audit_shadow_create_bugs_2026-05-27.md` với file changes (nếu có), line count.
