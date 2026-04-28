# Phase 38 — Requirements: search_path + schema-drift recovery

## Bối cảnh
Sau khi 4 service được start (auth/worker/cms/cms-fe), 11 endpoint kiểm tra của
operator-flow đều trả 500. Phiên trước báo PASS sai vì chỉ probe `/health=ok`.
Lần này phải verify đầy đủ cả **operator-flow** lẫn **auto-flow** (Debezium →
Kafka → Worker) trước khi tuyên bố done.

## Triệu chứng (CMS log + curl)
- 11/11 endpoints lúc đầu → 500.
- Lỗi nhóm 1: `relation "cdc_table_registry" does not exist`,
  `relation "cdc_reconciliation_report" does not exist`,
  `relation "cdc_activity_log" does not exist`,
  `relation "failed_sync_logs" does not exist`.
- Lỗi nhóm 2: `relation "cdc_internal.schema_proposal" does not exist`.
- Lỗi nhóm 3: `column tr.sync_status does not exist`,
  `column tr.recon_drift does not exist`.
- Lỗi nhóm 4: `invalid field found for struct WorkerScheduleResponse`.

## Root cause
1. **search_path chưa cập nhật**. Migration 037/038 di tản tables sang schema
   `cdc_system`, nhưng DB role vẫn dùng search_path mặc định (`"$user", public`).
   GORM `TableName()` của các model như `cdc_table_registry`, `failed_sync_logs`,
   `cdc_activity_log`, `cdc_reconciliation_report` không qualify schema
   (chỉ trả tên table) → 42P01.
2. **`cdc_internal` schema không tồn tại trong DB**. Code có ~20 chỗ hardcode
   `cdc_internal.schema_proposal`, `cdc_internal.transmute_schedule`, các DDL
   shadow table, sonyflake function → tất cả query fail. DB chỉ có 2 schema:
   `cdc_system` và `public`.
3. **Schema drift trong code vs DB**:
   - Bảng `cdc_table_registry` không có cột `sync_status` và `recon_drift`,
     nhưng `source_objects_handler.go` tham chiếu chúng.
   - Bảng `transmute_schedule` không có cột `master_table` (chỉ có
     `master_binding_id`); query GORM `Order("master_table, mode")` fail.
4. **GORM scan không hỗ trợ nested struct**. `WorkerScheduleResponse` có field
   `Scope WorkerScheduleScope` (sub-struct); `Raw().Scan(&[]WorkerScheduleResponse)`
   không map được flat columns → `invalid field`.
5. **shadow_binding orphan data**. 9/10 record có `shadow_schema='cdc_internal'`
   nhưng schema đó không tồn tại → debt riêng (không trong scope endpoint).

## Definition of Done
- 11/11 operator endpoints trả 200 với token JWT hợp lệ.
- Auto-flow infrastructure xanh: Debezium connector RUNNING, Kafka topics có
  prefix `cdc.goopay.*`, Worker discovery loop chạy mà không panic.
- Build pass cả `cdc-cms-service` và `centralized-data-service`.
- Migration `039_set_search_path.sql` đã apply (verified).
- Phase 38 docs đầy đủ bộ prefix theo CLAUDE.md.
- Lesson được tổng quát hóa Global Pattern (post-mortem search_path miss + schema
  drift gating).

## Out of scope (debt note)
- Reconcile orphan `shadow_schema='cdc_internal'` rows trong
  `cdc_system.shadow_binding` (9 record). Cần một phase riêng để hoặc
  (a) tạo schema `cdc_internal` + materialize shadow tables, hoặc
  (b) migrate `shadow_schema='cdc_internal'` → schema thực (vd. `cdc_system`),
  cộng update `physical_table_fqn`.
- Sửa `shadow_automator.go` (DDL tạo shadow table) và `mapping_preview_handler.go`
  (đọc `cdc_internal.<shadow>`) — cùng debt với (a) ở trên.
- Sync registry `source_object_registry` với Debezium `collection.include.list`
  để auto-flow ingest thật sự. Hiện registry chỉ có 1 row debezium (`payments`)
  nhưng Debezium publish topic theo collections `payment-bills`, `refund-requests`,
  `export-jobs`, …
- Bảng `cdc_table_registry` thiếu `sync_status` + `recon_drift` columns: nên
  hoặc bổ sung migration thêm cột (giữ semantic), hoặc xóa hẳn các tham chiếu
  còn sót. Hiện đã chọn xóa tham chiếu để unblock.
