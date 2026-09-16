# 08_tasks_multi_connection_refinement.md
## Danh Sách Task Triển Khai: Master Multi-Connection Refinement & LIMIT 1 Elimination

### Phase 1: Database Migration & CMS Backend (`cdc-cms-service`)
- [x] 1.1. Tạo migration `migrations/schema/recon_dlq/104_add_master_binding_id_to_transmute_jobs.sql`: Thêm cột `master_binding_id BIGINT REFERENCES cdc_system.master_binding(id)` + Index `idx_transmute_jobs_binding_status` + backfill query.
- [x] 1.2. Áp dụng migration vào DB PostgreSQL hệ thống (qua psql / migrate).
- [x] 1.3. Cập nhật `internal/infra/persistence/master/master_read_repo_gorm.go`: Sửa `ListEnriched` join LATERAL vào `transmute_jobs` theo `tj.master_binding_id = mb.id` và guard `mb.schema_status = 'approved'` (sửa dứt điểm Lỗi 1).
- [x] 1.4. Cập nhật `internal/infra/persistence/master/master_repo_gorm.go`:
  - Thêm helper `findBindingForAction(ctx context.Context, name string, bindingID int64) (*dbMasterBinding, error)`.
  - Cập nhật `ApproveSchemaTx`, `RejectSchema`, `RevertSchemaTx`, `ResolveMasterBindingByName` sử dụng helper này (sửa dứt điểm Lỗi 2).
- [x] 1.5. Cập nhật API Handlers (`internal/api/master/master_registry_handler_approve.go`, `reject.go`, `toggle.go`, `update_spec.go`): Cho phép nhận `binding_id` từ query parameter hoặc body.
- [x] 1.6. Cập nhật `internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`:
  - Sửa hàm `Save` nhận `masterBindingID int64`. Xóa bỏ hoàn toàn query `ORDER BY id DESC LIMIT 1`.
  - Sửa `GetHeaderByID` SELECT thêm `ts.master_binding_id`.
- [x] 1.7. Cập nhật `internal/infra/persistence/scheduler/transmute_schedule_read_repo_gorm.go`: Sửa `ListSchedules` SELECT thêm `ts.master_binding_id` và `COALESCE(mc.connection_code, 'default') AS master_connection_code`.
- [x] 1.8. Cập nhật NATS Publisher trong CMS:
  - `internal/app/commands/governance/approve_master.go`: NATS `cdc.cmd.master-create` payload đính kèm `"master_binding_id": masterBindingID`.
  - `internal/app/commands/scheduler/run_now.go`: NATS `cdc.cmd.transmute` payload đính kèm `"master_binding_id": header.MasterBindingID`.
- [x] 1.9. Cập nhật `internal/infra/persistence/master/master_transmute_job_repo_gorm.go`: Lưu và query `master_binding_id` trong `cdc_system.transmute_jobs`.
- [x] 1.10. Hoàn thành tích hợp backend CMS xử lý toàn diện MasterBindingID.

---

### Phase 2: Centralized Data Service Engine (`centralized-data-service`)
- [x] 2.1. Cập nhật `internal/handler/master/master_ddl_handler.go`:
  - Struct `masterCreateRequest` thêm `MasterBindingID int64 `json:"master_binding_id,omitempty"``.
  - Truyền `req.MasterBindingID` vào `h.gen.Apply`.
- [x] 2.2. Cập nhật `internal/service/master/master_ddl_generator.go`:
  - Sửa `loadBinding(ctx context.Context, masterName string, bindingID ...int64) (*masterDDLBindingRow, error)`.
  - Nếu `bindingID > 0` hoặc `masterName` là số nguyên: query trực tiếp `WHERE mb.id = ?`. Triệt tiêu hoàn toàn `LIMIT 1` mò mẫm!
  - `Apply` và `Generate` hỗ trợ tra cứu chính xác theo binding ID.
- [x] 2.3. Cập nhật `internal/handler/master/transmute_handler.go`:
  - Struct `TransmuteRequest` thêm `MasterBindingID int64 `json:"master_binding_id,omitempty"``.
  - Truyền `req.MasterBindingID` vào `h.svc.Run` và debouncer key.
- [x] 2.4. Cập nhật `internal/service/master/transmuter.go`:
  - Sửa `loadMaster(ctx context.Context, name string, bindingID ...int64) (*masterBindingRuntime, error)`.
  - Nếu `bindingID > 0`: query trực tiếp `WHERE mb.id = ?`. Triệt tiêu hoàn toàn `LIMIT 1` mò mẫm!
  - `finishTransmuteJob`: Lưu `master_binding_id` vào `cdc_system.transmute_jobs`.
- [x] 2.5. Cập nhật `internal/repository/master/master_binding_repo.go` & Realtime Fanout:
  - Thêm struct `MasterTargetIdentity` (`ID`, `MasterFQN`, `MasterConnectionKey`).
  - Thêm hàm `ListMasterTargetsByShadowIdentity(ctx, shadowTable, shadowSchema, ...) ([]MasterTargetIdentity, error)`.
  - Sửa `HandleTransmuteShadow` trong `transmute_handler.go`: Loop qua từng target, bắn `cdc.cmd.transmute` mang đúng `master_binding_id` riêng của từng binding.
- [x] 2.6. Cập nhật `internal/service/master/transmute_scheduler.go`:
  - Sửa query claim due: SELECT thêm `ts.master_binding_id`.
  - Bổ sung `"master_binding_id": d.masterBindingID` vào payload NATS `cdc.cmd.transmute`.
- [x] 2.7. Cập nhật `internal/repository/transmute_job_repo.go`: `TransmuteJob` model và `UpdateStatus` lưu `master_binding_id`.

---

### Phase 3: CMS Web Frontend (`cdc-cms-web`)
- [x] 3.1. Cập nhật `src/pages/MasterRegistry.tsx`:
  - Sửa `opMut` (approve, reject, toggle): Gọi API đính kèm `binding_id` trong URL và body.
  - Sửa `editMut`: Đính kèm `binding_id`.
  - Sửa `submitSwap`: Đính kèm `binding_id`.
  - Sửa `syncMut`: Truyền `master_binding_id: args.row.id` vào `POST /api/v1/schedules`. Khi tìm schedule immediate, lọc chính xác `s.master_binding_id === args.row.id`.
  - Sửa `TransmuteJobStatus`: Nhận `bindingId={r.id}` và query status/cancel theo binding ID.
  - Sửa `activeTransmuteJobs`: Hỗ trợ key dạng số nguyên `r.id` tránh va chạm giữa các master bindings cùng tên.
- [x] 3.2. Cập nhật `src/pages/TransmuteSchedules.tsx`:
  - Cập nhật `ScheduleRow`: Thêm `master_binding_id` và `master_connection_code`.
  - Cập nhật cột `Master`: Hiển thị `master_table`, Tag `#{master_binding_id}`, `master_schema`, Tag `conn: {master_connection_code}`.
  - Thêm ô tìm kiếm tức thì theo table, schema, connection code, binding ID, shadow table.
  - Form tạo schedule hỗ trợ chọn trực tiếp từ danh sách Master Binding đích đã đăng ký hoặc gán `master_binding_id`.

---

### Phase 4: End-to-End Verification & Validation
- [x] 4.1. Khởi động lại các service (CMS Backend, Worker CDS) và kiểm tra tích hợp toàn trình.
- [x] 4.2. Verify Approve Master 2: Bắn payload mang `master_binding_id` xác định chính xác binding đích trên container PostgreSQL port 5437, không còn ambiguous.
- [x] 4.3. Verify Sync Master 2: Transmute payload mang `master_binding_id`, triệt tiêu lỗi hiển thị chéo và lỗi sync nhầm database.
- [x] 4.4. Verify Schedule Sync của Master 1 và Master 2 hoạt động độc lập, không bị ghi đè lẫn nhau.
- [x] 4.5. Hoàn tất báo cáo kết quả và audit log.
