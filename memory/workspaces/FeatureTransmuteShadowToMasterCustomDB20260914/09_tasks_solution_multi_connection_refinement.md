# HỒ SƠ GIẢI PHÁP KỸ THUẬT CHI TIẾT (TECHNICAL SOLUTIONS)
## Khắc Phục Nguy Cơ DDL/Transmute Nhầm DB & Xóa Bỏ Query LIMIT 1 Mò Mẫm

### Module 1: Database Migration
**File**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/104_add_master_binding_id_to_transmute_jobs.sql`
- Thêm cột `master_binding_id BIGINT REFERENCES cdc_system.master_binding(id) ON DELETE CASCADE`.
- Tạo index `idx_transmute_jobs_binding_status ON cdc_system.transmute_jobs(master_binding_id, status, created_at DESC)`.
- Backfill dữ liệu `master_binding_id` từ `master_binding` cho các job cũ.

---

### Module 2: CMS Backend Refinement

#### 1. File: `internal/infra/persistence/master/master_read_repo_gorm.go`
- **Mục tiêu**: Giải quyết Lỗi 1 (`Synced 509/509` hiển thị chéo trên Master mới `pending_review`).
- **Chi tiết sửa đổi**:
  ```sql
  LEFT JOIN LATERAL (
      SELECT status, rows_affected, total_rows, trace_id, job_id, error_message
        FROM cdc_system.transmute_jobs tj
       WHERE (tj.master_binding_id = mb.id 
              OR (tj.master_binding_id IS NULL AND (
                  tj.master_table = (COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table)
                  OR (mb.master_schema IS NOT NULL AND mb.master_schema <> '' AND tj.master_table = (mb.master_schema || '.' || mb.master_table))
                  OR (mb.physical_table_fqn IS NOT NULL AND tj.master_table = mb.physical_table_fqn)
                  OR tj.master_table = mb.master_table
              )))
         AND mb.schema_status = 'approved'
       ORDER BY tj.created_at DESC
       LIMIT 1
  ) tj ON true
  ```

#### 2. File: `internal/infra/persistence/master/master_repo_gorm.go`
- **Mục tiêu**: Giải quyết Lỗi 2 (`ambiguous_master_name` khi Approve/Reject/Revert).
- **Hàm mới**: `findBindingForAction(ctx context.Context, name string, bindingID int64) (*dbMasterBinding, error)`
  - Nếu `bindingID > 0`: query `WHERE id = ?`.
  - Nếu `name` là chuỗi số: query `WHERE id = ?`.
  - Nếu `name` là `schema.table`: query `WHERE master_schema = ? AND master_table = ? LIMIT 2`. Nếu `> 1` -> trả về `ambiguous_master_name`.
- Cập nhật `ApproveSchemaTx`, `RejectSchema`, `RevertSchemaTx`, `ResolveMasterBindingByName` sử dụng helper này.

#### 3. File: `internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`
- **Mục tiêu**: Xóa bỏ hoàn toàn query `ORDER BY id DESC LIMIT 1`.
- Sửa hàm `Save`: Nhận `masterBindingID int64`. Nếu `masterBindingID > 0`, insert/update trực tiếp. Nếu không, tra cứu chính xác (nếu > 1 dòng thì báo lỗi ambiguous).
- Sửa `GetHeaderByID`: SELECT thêm `ts.master_binding_id`.

#### 4. File: `internal/app/commands/governance/approve_master.go` & `scheduler/run_now.go`
- `approve_master.go`: NATS `cdc.cmd.master-create` payload:
  ```go
  payload, _ := json.Marshal(map[string]any{
      "master_binding_id": masterBindingID,
      "master_table":      physicalTableFQN,
      "triggered_by":      cmd.UpdatedBy,
      "correlation_id":    correlationID,
  })
  ```
- `run_now.go`: NATS `cdc.cmd.transmute` payload:
  ```go
  runCmd := TransmuteRunCommand{
      MasterBindingID: header.MasterBindingID,
      JobID:           jobID,
      MasterTable:     masterTableFQN,
      TriggeredBy:     cmd.Actor,
      CorrelationID:   "run-now-" + strconv.FormatInt(cmd.ID, 10) + "-" + time.Now().UTC().Format(time.RFC3339Nano),
      ScheduleID:      cmd.ID,
      TraceID:         traceID,
  }
  ```

---

### Module 3: CDS Worker Engine Refinement

#### 1. File: `internal/handler/master/master_ddl_handler.go` & `master_ddl_generator.go`
- Thêm field `MasterBindingID int64 `json:"master_binding_id,omitempty"`` vào `masterCreateRequest`.
- Sửa `MasterDDLGenerator.loadBinding(ctx, masterName, bindingID int64)`:
  - Nếu `bindingID > 0`: `SELECT ... FROM cdc_system.master_binding mb WHERE mb.id = ?`.
  - Triệt tiêu `LIMIT 1` bốc nhầm database!

#### 2. File: `internal/handler/master/transmute_handler.go` & `transmuter.go`
- Thêm field `MasterBindingID int64 `json:"master_binding_id,omitempty"`` vào `TransmuteRequest`.
- Sửa `TransmuterModule.loadMaster(ctx, name, bindingID int64)`:
  - Nếu `bindingID > 0`: `SELECT ... FROM cdc_system.master_binding mb WHERE mb.id = ?`.
  - Triệt tiêu `LIMIT 1` bốc nhầm database!

#### 3. File: `internal/repository/master/master_binding_repo.go` & Realtime Fanout
- Thêm struct `MasterTargetIdentity` chứa `ID int64`, `MasterFQN string`, `MasterConnectionKey string`.
- Hàm `ListMasterTargetsByShadowIdentity`: trả về danh sách các target master.
- `HandleTransmuteShadow`: Duyệt danh sách targets, bắn NATS `cdc.cmd.transmute` mang đúng `master_binding_id` cho từng target.

#### 4. File: `internal/service/master/transmute_scheduler.go`
- Bổ sung `master_binding_id` vào claim due query và payload NATS `cdc.cmd.transmute`.

---

### Module 4: CMS Web Frontend Refinement

#### 1. File: `src/pages/MasterRegistry.tsx`
- Sửa `syncMut`: Truyền `master_binding_id: args.row.id` khi tạo schedule và khi run-now.
- Sửa `handleApprove`: Truyền `binding_id: O.row.id` vào request.
- Sửa `TransmuteStatusRenderer`: Nhận `bindingId={r.id}`.

#### 2. File: `src/pages/TransmuteSchedules.tsx`
- Hiển thị cột `Connection Code`.
- Lọc schedule theo `master_binding_id`.
