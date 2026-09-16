# KẾ HOẠCH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PLAN)
## Tinh Chỉnh Kiến Trúc Master Multi-Connection: Triệt Tiêu Nguy Cơ DDL/Transmute Nhầm DB & Xóa Bỏ Query LIMIT 1 Mò Mẫm

### 1. Tổng quan các điểm chạm (Touchpoints Registry)

| STT | Thành phần | File chạm | Vấn đề hiện tại | Giải pháp khắc phục |
|---|---|---|---|---|
| 1 | DB Migration | `migrations/schema/recon_dlq/104_add_master_binding_id_to_transmute_jobs.sql` | `transmute_jobs` thiếu FK `master_binding_id` | Thêm cột `master_binding_id BIGINT REFERENCES cdc_system.master_binding(id)` + Index |
| 2 | CMS Repo | `internal/infra/persistence/master/master_read_repo_gorm.go` | `ListEnriched` join LATERAL vào `transmute_jobs` chỉ bằng tên text | Join theo `tj.master_binding_id = mb.id` và guard `mb.schema_status = 'approved'` |
| 3 | CMS Repo | `internal/infra/persistence/master/master_repo_gorm.go` | `ApproveSchemaTx`, `RejectSchema`, `ResolveMasterBindingByName` query `LIMIT 2` gây `ambiguous_master_name` | Tạo helper `findBindingForAction` hỗ trợ resolve theo `binding_id` hoặc số nguyên |
| 4 | CMS Handler | `internal/api/master/master_registry_handler_approve.go` | Handler không đọc `binding_id` | Nhận `c.Query("binding_id")` hoặc body `binding_id` |
| 5 | CMS NATS | `internal/app/commands/governance/approve_master.go` | Payload `cdc.cmd.master-create` chỉ có `master_table` | Bổ sung `"master_binding_id": masterBindingID` vào payload |
| 6 | CDS Handler | `internal/handler/master/master_ddl_handler.go` | `masterCreateRequest` không có `master_binding_id` | Thêm field `MasterBindingID int64`, truyền vào generator |
| 7 | CDS Service | `internal/service/master/master_ddl_generator.go` | `loadBinding` query `WHERE mb.master_table = ? LIMIT 1` | Nếu `masterBindingID > 0` hoặc name là số -> query `WHERE mb.id = ?` |
| 8 | CMS Scheduler | `internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go` | `Save` chạy query `ORDER BY id DESC LIMIT 1` | Nhận `masterBindingID int64`, INSERT/UPDATE trực tiếp không cần query LIMIT 1 |
| 9 | CMS Scheduler | `internal/infra/persistence/scheduler/transmute_schedule_read_repo_gorm.go` | `ListSchedules` không SELECT `master_binding_id` và `connection_code` | SELECT `ts.master_binding_id` và `connection_code` |
| 10 | CMS Run-Now | `internal/app/commands/scheduler/run_now.go` | NATS `cdc.cmd.transmute` thiếu `master_binding_id` | Lấy `header.MasterBindingID` gửi vào payload NATS |
| 11 | CDS Scheduler | `internal/service/master/transmute_scheduler.go` | Query claim due bỏ quên `master_binding_id` khi publish NATS | Gửi `"master_binding_id": d.masterBindingID` vào `cdc.cmd.transmute` |
| 12 | CDS Fanout | `internal/repository/master/master_binding_repo.go` | `ListMasterTablesByShadowIdentity` chỉ trả về `[]string` tên bảng | Trả về `[]MasterTargetIdentity` chứa `ID`, `MasterFQN`, `MasterConnectionKey` |
| 13 | CDS Handler | `internal/handler/master/transmute_handler.go` | `HandleTransmuteShadow` loop string, gọi `loadMaster LIMIT 1` | Loop qua từng target, gửi `master_binding_id` vào `cdc.cmd.transmute` |
| 14 | CDS Transmuter | `internal/service/master/transmuter.go` | `loadMaster` query `WHERE mb.master_table = ? LIMIT 1` | Nếu `masterBindingID > 0` -> query `WHERE mb.id = ?`, connect đúng DB |
| 15 | CMS Web UI | `src/pages/MasterRegistry.tsx` | UI gọi Approve/Reject/Sync thiếu `binding_id` | Gửi `binding_id` cho mutation Approve, Reject, Toggle, và Schedule |
| 16 | CMS Web UI | `src/pages/TransmuteSchedules.tsx` | Không hiển thị target connection | Hiển thị tag connection và filter chính xác theo `master_binding_id` |

### 2. Chi Tiết Kế Hoạch Thực Hiện
1. Tạo migration `104_add_master_binding_id_to_transmute_jobs.sql`.
2. Sửa CMS Backend:
   - `master_read_repo_gorm.go`: Fix `ListEnriched` join LATERAL.
   - `master_repo_gorm.go`: Thêm `findBindingForAction`, sửa `ApproveSchemaTx`, `RejectSchema`, `RevertSchemaTx`, `ResolveMasterBindingByName`.
   - `master_registry_handler_approve.go` và các handlers: Tiếp nhận `binding_id`.
   - `approve_master.go`: Thêm `master_binding_id` vào payload NATS `cdc.cmd.master-create`.
   - `transmute_schedule_repository_gorm.go`: Xóa bỏ `ORDER BY id DESC LIMIT 1` trong `Save`, thêm `master_binding_id` vào `GetHeaderByID`.
   - `run_now.go`: Gửi `master_binding_id` vào NATS `cdc.cmd.transmute`.
   - `transmute_schedule_read_repo_gorm.go`: SELECT thêm `master_binding_id` và `connection_code`.
3. Sửa CDS Worker Engine:
   - `master_ddl_handler.go`: Bổ sung `MasterBindingID` vào `masterCreateRequest`.
   - `master_ddl_generator.go`: `loadBinding` tra cứu trực tiếp theo `mb.id = ?` khi có `bindingID > 0`.
   - `transmute_handler.go`: Bổ sung `MasterBindingID` vào `TransmuteRequest`.
   - `transmuter.go`: `loadMaster` tra cứu trực tiếp theo `mb.id = ?` khi có `bindingID > 0`.
   - `master_binding_repo.go`: Thêm `ListMasterTargetsByShadowIdentity` trả về danh sách `MasterTargetIdentity` (chứa `id`, `master_fqn`, `master_connection_key`).
   - `transmute_handler.go:HandleTransmuteShadow`: Fanout NATS `cdc.cmd.transmute` mang đúng `master_binding_id` cho từng target.
   - `transmute_scheduler.go`: Bổ sung `master_binding_id` vào payload NATS `cdc.cmd.transmute`.
4. Sửa CMS Web Frontend:
   - `MasterRegistry.tsx`: Bổ sung `binding_id` khi gọi Approve, Reject, Toggle, Edit Spec, và Schedule Sync.
   - Sửa component `TransmuteStatusRenderer` truyền `bindingId`.
   - `TransmuteSchedules.tsx`: Hiển thị thông tin target connection và filter chính xác theo `master_binding_id`.
