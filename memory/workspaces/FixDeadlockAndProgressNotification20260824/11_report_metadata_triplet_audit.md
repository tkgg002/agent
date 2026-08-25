# 11_report_metadata_triplet_audit.md — Báo Cáo Audit Toàn Diện Bộ Ba Metadata (Connection/DB, Schema, Table)

**Ngày kiểm toán:** 2026-08-24  
**Phạm vi:** 
- **Tier 1: `Source → Shadow`** (Ingest / Oplog / Bridge / Transform / Snapshot)
- **Tier 2: `Shadow → Master`** (Transmute / Schedule / Sync)

---

## 1. TỔNG QUAN KẾT QUẢ AUDIT

| Thành phần | Module | Kiểm tra Metadata (Conn, Schema, Table) | Trạng thái trước Audit | Kết quả sau Fix |
| :--- | :--- | :--- | :--- | :--- |
| **Tier 1: Bridge** | `bridge_handler.go` | `SourceDB`, `SourceTable`, `SchemaName`, `TableName`, `shadowConnectionKey` | Đầy đủ và chuẩn | **PASS** |
| **Tier 1: Transform** | `batch_transform_handler.go` | `ResolveTargetSchema()`, `pureTable`, `targetTable` | Chuẩn hóa FQN và short name | **PASS** |
| **Tier 1: Snapshot** | `snapshot_runner_handler.go` | `SourceConnectionID`, `SourceSchema`, `srcColl`, `ShadowSchema`, `ShadowTable` | Trích xuất chuẩn từ registry và connector config | **PASS** |
| **Tier 2: Scheduler** | `transmute_scheduler.go` | `COALESCE(master_schema, 'public') || '.' || master_table` (FQN) | Chưa sinh `trace_id` khi fire cron tick | **ĐÃ FIX & PASS** |
| **Tier 2: Transmuter** | `transmuter.go` | `loadMaster(FQN/pure)`, `quoteTransmuteQualified(schema, table)` | `EnsureMaster` gọi `masterRow.MasterTable` trần | **ĐÃ FIX & PASS** |
| **Tier 2: Utils** | `transmuter_utils.go` | `quoteTransmuteQualified(schema, table)` | `schema=""` sinh ra `""."table"` (lỗi SQL) | **ĐÃ FIX & PASS** |
| **Tier 2: Job Tracking** | `transmute_job_repo.go` | `GetLatestByMasterBindingID`, `GetLatestByTable` | Chỉ tìm short name, miss khi job lưu FQN | **ĐÃ FIX & PASS** |
| **Tier 2: Read CMS** | `transmute_schedule_read_repo_gorm.go` | `JOIN sync_runtime_state srs` | `runtime_scope = 'transmute'` (sai DDL) | **ĐÃ FIX & PASS** |
| **Tier 2: Master List** | `master_read_repo_gorm.go` | `JOIN transmute_jobs tj` | `tj.master_table = mb.master_table` (miss FQN) | **ĐÃ FIX & PASS** |

---

## 2. CHI TIẾT TỪNG LỖ HỔNG ĐƯỢC PHÁT HIỆN VÀ ĐÃ XỬ LÝ

### 2.1. `transmuter.go` — `EnsureMaster` gọi tên bảng ngắn làm mất schema isolation
- **Vấn đề:** Khi `Transmuter.Run()` thực hiện DDL EnsureMaster, câu lệnh gọi `t.ddlEnsurer.EnsureMaster(ctx, masterRow.MasterTable)`.
- **Hậu quả:** Nếu có 2 bảng cùng tên `bank_requests` ở 2 schema khác nhau, `EnsureMaster` chỉ nhận `"bank_requests"` và query `LIMIT 1` có thể nạp nhầm binding của schema khác.
- **Fix:** Đã cập nhật truyền `masterFQN = masterRow.MasterSchema + "." + masterRow.MasterTable` và lưu cache `ensuredMasters` theo FQN.

### 2.2. `transmuter_utils.go` — `quoteTransmuteQualified` không fallback schema rỗng
- **Vấn đề:** `quoteTransmuteQualified(schemaName, tableName)` gọi trực tiếp `quoteTransmuteIdent(schemaName) + "." + quoteTransmuteIdent(tableName)`.
- **Hậu quả:** Nếu `schemaName` bị rỗng `""`, câu SQL tạo ra chuỗi `""."table_name"` ➔ Gây lỗi cú pháp SQLSTATE trong PostgreSQL.
- **Fix:** Bổ sung fallback `if schema == "" { schema = "public" }`.

### 2.3. `transmute_scheduler.go` — Cron tick chưa sinh và lưu `last_trace_id`
- **Vấn đề:** Khi scheduler kích hoạt cron tick định kỳ, câu UPDATE chỉ set `last_status='running'` mà không gán `last_trace_id`, và NATS payload không có `trace_id`.
- **Hậu quả:** Khi cron chạy xong, cột Trace ID trên `/schedules` bị trống `—`.
- **Fix:** Tự động sinh `traceID = uuid.New()` gắn vào câu lệnh UPDATE `last_trace_id` và truyền qua NATS payload `cdc.cmd.transmute`.

### 2.4. `master_read_repo_gorm.go` — LATERAL JOIN `transmute_jobs` chỉ so sánh tên bảng ngắn
- **Vấn đề:** Câu JOIN `WHERE tj.master_table = mb.master_table` chỉ match khi job được lưu dưới dạng short table name, bỏ qua các job được lưu dưới dạng FQN (`master_bidv_connector_service.bank_requests`).
- **Hậu quả:** Trên trang Master Registry, dòng master không hiện đúng `last_transmute_rows` và `last_transmute_status` sau khi reload trang.
- **Fix:** Cập nhật điều kiện JOIN:
  ```sql
  WHERE tj.master_table = (COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table)
     OR tj.master_table = mb.master_table
  ```

---

## 3. KIỂM ĐỊNH BUILD VÀ TOÀN VẸN HỆ THỐNG
- `centralized-data-service`: `go build ./internal/... ./cmd/...` ➔ **PASS (Exit 0)**
- `cdc-cms-service`: `go build ./internal/... ./cmd/...` ➔ **PASS (Exit 0)**
- `cdc-cms-web`: `npm run build` ➔ **PASS (Exit 0)**
