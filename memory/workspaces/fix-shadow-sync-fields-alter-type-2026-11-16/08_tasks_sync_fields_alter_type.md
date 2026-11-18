---
tier: T2
suffix: sync_fields_alter_type
workspace: fix-shadow-sync-fields-alter-type-2026-11-16
---
# 08 Tasks — Điều Tra & Thiết Kế Giải Pháp Sửa Lỗi "Sync Fields" Không `ALTER COLUMN TYPE` (`TEXT` → `JSONB`)

## Task List

### T1. [S2 Investigate] Truy vết End-to-End luồng "Sync Fields to Shadow" từ FE -> CMS API -> NATS Command -> Worker Schema Manager
- **Context (`file:line`):**
  - Frontend: `cdc-cms-web/src/pages/MappingFieldsPage.tsx:165-185` (`handleSyncFields`), `:228-247` (`fetchShadowColumns`), `:321-333` (`handleUpdateDataType`), `:470-491` (`Data Type Current` drift check).
  - Backend CMS (`cdc-cms-service`): route `/api/v1/source-objects/:id/create-default-columns`, handler/command phát tín hiệu sang worker, và endpoint `/api/mapping-rules/:id` khi đổi `data_type`.
  - Worker (`centralized-data-service`): consumer nhận lệnh `create-default-columns` / `sync_fields`, `schema_manager.go` (hoặc tương đương) thực thi `ADD COLUMN` / `ALTER COLUMN TYPE` cho bảng Shadow.
- **Logs / Test:** Kiểm tra code thực tế + container/process logs hiện có (Read-Only) nếu service đang chạy trên máy để xem lỗi runtime thực tế khi User vừa bấm `Sync Fields`.
- **Definition of Done:** Xác định chính xác 100% (`VERIFIED` kèm `file:line` và log thực tế nếu có) tại sao `Target: JSONB · Current: TEXT` không được `ALTER COLUMN TYPE` (hoặc tại sao FE không cập nhật / worker bỏ qua / PostgreSQL báo lỗi `42804` thiếu `USING` / whitelist chặn `TEXT -> JSONB` / điều kiện `status`/`is_active`).

### T2. [S2.5 Pre-Plan Critique (DG0) + S3 Solution + DG1/DG2] Phản biện Giả định & Thiết kế Giải pháp Core Thuần Khiết (Không Vá Data Cũ — `L097`)
- **Context (`file:line`):** Toàn bộ các điểm chạm Core trong `cdc-cms-web` (`MappingFieldsPage.tsx`), `cdc-cms-service` (`source_object_actions_handler.go`, `sync_shadow_columns.go`), `centralized-data-service` (`dynamic_mapper.go`, `schema_adapter_coerce.go`, `schema_adapter.go`, `schema_ddl_handler.go`).
- **Logs / Test:** Chạy `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` (`PASS`).
- **Definition of Done:**
  - Tuân thủ tuyệt đối `L097`: loại bỏ 100% logic khôi phục `_raw_data` / regex `map[...]` cho data cũ; chỉ chuẩn hóa luồng Core từ hiện tại trở về sau.
  - Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** ánh xạ 1-1 với từng phát hiện `F1..F4` kèm **Code Demo** trong `09_tasks_solution_sync_fields_alter_type.md` và hiển thị trực tiếp **Code Demo** trên chat; dừng chờ User `APPROVE`.

### T3. [S4 Execute + S5 Adversarial Audit (DG3) + S6 Close — COMPLETED]
- **Context (`file:line`):**
  - `centralized-data-service`: `dynamic_mapper.go:233-241`, `schema_adapter_coerce.go:31-43`, `schema_adapter.go:822-862`, `schema_ddl_handler.go:275-358,453-466,711-714`, `shadow_alter_column_jsonb_test.go:19-149`.
  - `cdc-cms-service`: `sync_shadow_columns.go:1-70`, `sync_shadow_columns_test.go:1-64`, `server.go:301`, `source_object_actions_handler.go:242-282`.
  - `cdc-cms-web`: `MappingFieldsPage.tsx:166-190,484-495`.
- **Logs / Test:** 10 biên lai `EV-...` (`EV-20261117T103553-620b` .. `EV-20261117T103656-434c`) + `DG3` (`06_laya_summary_dg3_20261117T103932.json`, `gate=PASS`, `trusted_rounds=["r3"]`).
- **Definition of Done:** Đạt đủ Gates `G1–G8`, hoàn tất `audit_report_sync_fields_alter_type.md` và `verify_governance.py` `PASSED`.
