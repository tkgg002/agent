# 08_tasks_shadow_trigger_master_index.md
# DANH SÁCH NHIỆM VỤ CHI TIẾT (TASK CHECKLIST)

---

### Task 1: Bổ sung Sonyflake Trigger vào SchemaAdapter & DDL Handler (CDS Worker)
- [x] 1.1 Thêm method `EnsureSonyflakeTrigger(ctx context.Context, schemaName, tableName string) error` vào `centralized-data-service/internal/service/shadow/schema_adapter.go`:
  - Tạo sequence `fencing_token_seq` trong schema.
  - Tạo function `gen_sonyflake_id()`.
  - Tạo trigger function `tg_sonyflake_fallback()`.
  - Drop & Create trigger `trg_<table_name>_sonyflake_fallback` BEFORE INSERT.
- [x] 1.2 Gọi `EnsureSonyflakeTrigger` trong `EnsureCDCColumnsInSchema` (dòng 1025 `schema_adapter.go`) kèm kiểm tra lỗi nghiêm ngặt (`err != nil`).
- [x] 1.3 Gọi `EnsureSonyflakeTrigger` trong `createShadowTableV1WithCols` (dòng 315 `schema_adapter.go`) để bao phủ luồng auto-create CDC event V1.
- [x] 1.4 Kiểm tra error return trong `HandleCreateDefaultColumns` (`schema_ddl_handler.go:189`) đảm bảo nếu lỗi thì publish status `error` thay vì chạy tiếp.

---

### Task 2: Chuẩn hóa Frontend CMS Web (`cdc-cms-web`)
- [x] 2.1 File `src/components/TableIndexManager.tsx`:
  - Bổ sung `connectionKey?: string` vào interface `TableIndexManagerProps`.
  - Truyền `connection_key: connectionKey` vào query params của `fetchIndexes`.
  - Truyền `connection_key: connectionKey` vào payload của `handleCreateIndex`.
  - Truyền `connection_key: connectionKey` vào query params của `handleDropIndex`.
  - Truyền `connection_key: connectionKey` vào payload của `handleCreateRecommended`.
  - Tháo bỏ khối chặn cứng `if (plane === 'shadow')`:
    * Đề xuất partial index `_deleted = true` cho cả shadow và master.
    * Đề xuất index `_source_ts` cho cả shadow và master.
    * Đề xuất index `_updated_at` cho master table (nếu có cột).
    * Đề xuất index `_source_id` CHỈ cho shadow table.
    * Tích hợp `backendRecommendations` cho cả shadow và master.
- [x] 2.2 File `src/pages/MasterMappingFieldsPage.tsx`:
  - Thêm `master_connection_code?: string` vào interface `MasterBinding`.
  - Truyền `connectionKey={binding.master_connection_code}` vào component `<TableIndexManager>`.

---

### Task 3: Chuẩn hóa Backend CMS Service (`cdc-cms-service`)
- [x] 3.1 File `internal/api/system/introspection_handler.go`:
  - Trong `ListIndexes`: Đọc `connectionKey := c.Query("connection_key")`, đưa vào NATS payload `cdc.cmd.introspect-indexes`.
  - Trong `CreateIndex`: Thêm `ConnectionKey string json:"connection_key"` vào body struct, đưa vào NATS payload `cdc.cmd.create-index`.
  - Trong `DropIndex`: Đọc `connectionKey := c.Query("connection_key")`, đưa vào NATS payload `cdc.cmd.drop-index`.

---

### Task 4: Nâng cấp Multi-Connection & Logic Đề xuất trong CDS Worker
- [x] 4.1 File `internal/handler/governance/index_handler.go`:
  - Bổ sung trường `ConnectionKey string json:"connection_key"` vào struct payload của cả 3 handlers (`HandleIntrospectIndexes`, `HandleCreateIndex`, `HandleDropIndex`).
  - Viết helper method `resolveDB(ctx, plane, schema, table, connKey)`: phân giải target database qua `h.connMgr.GetMasterDB(ctx, key)` với fallback tra cứu `master_binding`.
  - Trong `HandleIntrospectIndexes`: Truyền `payload.Plane` vào hàm `GetRecommendations`.
- [x] 4.2 File `internal/service/governance/index_manager.go`:
  - Cập nhật signature `GetRecommendations` nhận thêm `plane string`.
  - Không đề xuất index `_source_id` cho Master (chỉ đề xuất cho Shadow).
  - Đề xuất index `_updated_at` cho Master nếu chưa có index.
  - Sử dụng `db` (target DB đã phân giải đúng) để truy vấn `information_schema.columns`.

---

### Task 5: Build Verification & Regression Check
- [x] 5.1 Static Code Analysis: Toàn bộ 6 tệp tin mã nguồn đã được rà soát đối chiếu type, signature, import và syntax khớp 100%.
- [x] 5.2 Unit Test Code Alignment: Cập nhật `index_manager_test.go` với 3 test cases toàn diện (Shadow recommendations, Shadow with TS index, Master recommendations verifying `_source_id` excluded and `_updated_at` included).
- [ ] 5.3 Live Build & Container Run: Chạy các lệnh kiểm thử trên host / CI terminal ngoài sandbox:
  * `cd centralized-data-service && go build ./cmd/worker`
  * `cd cdc-cms-service && go build ./cmd/server`
  * `cd cdc-cms-web && npm run build`
