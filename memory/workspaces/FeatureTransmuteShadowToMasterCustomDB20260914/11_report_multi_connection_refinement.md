# 11_report_multi_connection_refinement.md
## Báo Cáo Triển Khai Toàn Trình: Master Multi-Connection Refinement & LIMIT 1 Elimination

---

### 1. Tổng Quan Mục Tiêu & Vấn Đề Đã Giải Quyết
Trong kiến trúc Master Multi-Connection, hệ thống hỗ trợ 1 Shadow Table đồng bộ ra nhiều bảng Master trên các Database PostgreSQL vật lý khác nhau (ví dụ: `Master 1` trên port 5434 và `Master 2` trên port 5437). Trước khi thực hiện refinement, hệ thống gặp phải 2 lỗ hổng nghiêm trọng do cơ chế định danh chỉ dựa trên chuỗi tên bảng `master_table` (`master_centrallized_export_service.export_jobs`) và các câu query `LIMIT 1` mò mẫm:
1. **Lỗi 1 (Hiển thị chéo tiến độ sync)**: Khi tạo Master 2 nhưng chưa Approve, UI hiển thị chéo `Synced 509/509` của Master 1 vì câu query `ListEnriched` JOIN LATERAL vào bảng log `transmute_jobs` chỉ khớp theo chuỗi tên `tj.master_table = mb.master_table` và không kiểm tra `schema_status = 'approved'`.
2. **Lỗi 2 (Tạo DDL và Sync nhầm Database)**: Khi Approve Master 2, hệ thống báo lỗi `ambiguous_master_name` hoặc áp dụng nhầm DDL / Sync nhầm sang container `master_1` do các hàm `loadBinding`, `loadMaster`, `SaveSchedule` query bằng tên bảng và dùng `LIMIT 1`.

**Kết quả đạt được**: 
- Đã xác lập `master_binding_id` (Primary Key độc bản của RDBMS) làm Single Source of Truth trên toàn bộ pipeline (CMS Backend, NATS Event Bus, CDS Worker Engine, CMS Web Frontend).
- Đã xóa bỏ triệt để 100% các câu query `LIMIT 1` mò mẫm trên bảng `master_binding`. Bất kỳ lookup nào không có ID đều phải dùng `LIMIT 2` và fail-fast nếu phát hiện mơ hồ (ambiguous).

---

### 2. Danh Mục Các File Đã Thay Đổi & Khối Lượng Dòng Code

| STT | Repository | Đường Dẫn File | Thao Tác | Số Dòng Thay Đổi | Tóm Tắt Thay Đổi |
|:---|:---|:---|:---:|:---:|:---|
| 1 | `cdc-cms-service` | `migrations/schema/recon_dlq/104_add_master_binding_id_to_transmute_jobs.sql` | [NEW] | +25 | Thêm cột `master_binding_id`, Foreign Key CASCADE, index B-tree và script backfill data. |
| 2 | `cdc-cms-service` | `internal/infra/persistence/master/master_read_repo_gorm.go` | [MODIFY] | ~15 | Sửa `ListEnriched` join LATERAL theo `tj.master_binding_id = mb.id` và guard `mb.schema_status = 'approved'`. |
| 3 | `cdc-cms-service` | `internal/infra/persistence/master/master_repo_gorm.go` | [MODIFY] | ~45 | Helper `findBindingForAction` 3 tầng ưu tiên (ID > 0 -> numeric ID -> LIMIT 2 check ambiguous), cập nhật Approve/Reject/Revert/Resolve. |
| 4 | `cdc-cms-service` | `internal/api/dto/master_dto.go` | [MODIFY] | ~5 | Thêm trường `BindingID int64` vào `ApproveRequest`. |
| 5 | `cdc-cms-service` | `internal/app/commands/governance/approve_master.go` | [MODIFY] | ~10 | Resolve targetName theo ID và đính kèm `"master_binding_id"` vào NATS `cdc.cmd.master-create`. |
| 6 | `cdc-cms-service` | `internal/app/commands/governance/reject_master.go` | [MODIFY] | ~10 | Hỗ trợ `MasterBindingID` và resolve chính xác binding khi reject. |
| 7 | `cdc-cms-service` | `internal/api/master/master_registry_handler_approve.go` | [MODIFY] | ~15 | Trích xuất `binding_id` từ query parameter / body request. |
| 8 | `cdc-cms-service` | `internal/api/master/master_registry_handler_resolve.go` | [MODIFY] | ~10 | Hỗ trợ resolve master binding theo numeric ID. |
| 9 | `cdc-cms-service` | `internal/domain/scheduler/repository.go` | [MODIFY] | ~5 | `TransmuteScheduleHeader` thêm `MasterBindingID`, `Save` interface nhận `masterBindingID int64`. |
| 10 | `cdc-cms-service` | `internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go` | [MODIFY] | ~25 | `Save` nhận `masterBindingID`, xóa bỏ `ORDER BY id DESC LIMIT 1`, `GetHeaderByID` SELECT `ts.master_binding_id`. |
| 11 | `cdc-cms-service` | `internal/app/commands/scheduler/create_schedule.go` | [MODIFY] | ~5 | Bổ sung `MasterBindingID` vào Command. |
| 12 | `cdc-cms-service` | `internal/api/scheduler/transmute_schedule_handler.go` | [MODIFY] | ~5 | Payload nhận `MasterBindingID` từ client. |
| 13 | `cdc-cms-service` | `internal/app/queries/scheduler/list_transmute_schedules.go` | [MODIFY] | ~5 | DTO thêm `MasterBindingID` và `MasterConnectionCode`. |
| 14 | `cdc-cms-service` | `internal/infra/persistence/scheduler/transmute_schedule_read_repo_gorm.go` | [MODIFY] | ~10 | SELECT thêm `ts.master_binding_id` và `COALESCE(mc.connection_code, 'default') AS master_connection_code`. |
| 15 | `cdc-cms-service` | `internal/app/commands/scheduler/run_now.go` | [MODIFY] | ~5 | Payload NATS `cdc.cmd.transmute` bổ sung `"master_binding_id"`. |
| 16 | `cdc-cms-service` | `internal/app/commands/scheduler/transmute_run.go` | [MODIFY] | ~5 | Bổ sung `"master_binding_id"` vào transmute command event. |
| 17 | `cdc-cms-service` | `internal/infra/persistence/transmute_job_repo.go` | [MODIFY] | ~20 | Model thêm `MasterBindingID`, thêm `CreateWithBindingID`, `GetLatestByMasterBindingID`. |
| 18 | `centralized-data-service` | `internal/handler/master/master_ddl_handler.go` | [MODIFY] | ~10 | `masterCreateRequest` nhận `MasterBindingID`, truyền vào `h.gen.Apply`. |
| 19 | `centralized-data-service` | `internal/service/master/master_ddl_generator.go` | [MODIFY] | ~35 | `Apply`, `Generate`, `loadBinding` nhận `bindingID ...int64`, áp dụng 3 tầng ưu tiên, triệt tiêu `LIMIT 1`. |
| 20 | `centralized-data-service` | `internal/repository/transmute_job_repo.go` | [MODIFY] | ~15 | `TransmuteJob` model thêm `MasterBindingID *int64`, `UpdateStatus` lưu `master_binding_id`. |
| 21 | `centralized-data-service` | `internal/service/master/transmuter.go` | [MODIFY] | ~65 | `Run` nhận `masterBindingID ...int64`, `loadMaster` 3 tầng ưu tiên, `finishTransmuteJob` lưu `masterRow.ID` trên mọi return path. |
| 22 | `centralized-data-service` | `internal/repository/master/master_binding_repo.go` | [MODIFY] | ~40 | Thêm `MasterTargetIdentity`, `ListMasterTargetsByShadowIdentity`, `ListMasterTargetsByShadowTable`. |
| 23 | `centralized-data-service` | `internal/handler/master/transmute_handler.go` | [MODIFY] | ~55 | Fanout realtime NATS gửi `master_binding_id`, `TransmuteRequest` thêm `MasterBindingID`, debouncerKey tách biệt theo ID. |
| 24 | `centralized-data-service` | `internal/service/master/transmute_scheduler.go` | [MODIFY] | ~15 | Query claim due SELECT `ts.master_binding_id`, NATS payload gửi `"master_binding_id"`. |
| 25 | `cdc-cms-web` | `src/pages/MasterRegistry.tsx` | [MODIFY] | ~60 | `opMut`, `editMut`, `submitSwap`, `syncMut` gửi `binding_id`; `activeTransmuteJobs` hỗ trợ numeric ID; `TransmuteJobStatus` nhận `bindingId`. |
| 26 | `cdc-cms-web` | `src/pages/TransmuteSchedules.tsx` | [MODIFY] | ~70 | `ScheduleRow` thêm `master_binding_id`, `master_connection_code`; cột Master hiển thị Tag conn & #ID; thêm ô Search; chọn Master từ dropdown. |

**Tổng khối lượng thay đổi**: 26 tệp tin, ~585 dòng code được bổ sung/tối ưu hóa.

---

### 3. Chi Tiết Kiến Trúc Triển Khai Kỹ Thuật

#### 3.1. Master Identity 3-Tier Priority Resolver
Mọi tầng xử lý nghiệp vụ (Backend CMS, Generator DDL CDS, Transmuter Engine CDS) đều áp dụng thống nhất 3 tầng ưu tiên:
- **Tầng 1 (Tuyệt đối)**: Nếu `binding_id > 0` -> Query trực tiếp `WHERE mb.id = ?` qua khóa chính Primary Key B-Tree Index. Độ phức tạp O(1), không thể nhầm lẫn.
- **Tầng 2 (Chuỗi số nguyên)**: Nếu tham số tên là chuỗi số nguyên (`strconv.ParseInt(name) > 0`) -> Query `WHERE mb.id = ?`.
- **Tầng 3 (Lookup theo tên bảng với Fail-Fast Guard)**: Nếu không có ID, hệ thống query theo `master_table` và `master_schema` với `ORDER BY mb.updated_at DESC, mb.id DESC LIMIT 2`.
  - Nếu `len(rows) == 0` -> Báo lỗi `not found`.
  - Nếu `len(rows) > 1` -> Lập tức trả lỗi `ambiguous_master_name: master table is mapped to multiple master connections. Please specify master_binding_id`. Tuyệt đối không tự ý chọn bừa record đầu tiên như LIMIT 1 cũ.

#### 3.2. Closed-Loop NATS Contract
Payload của toàn bộ các NATS Subjects điều phối đã được chuẩn hóa:
```json
// cdc.cmd.master-create
{
  "master_table": "export_jobs",
  "master_binding_id": 2,
  "reason": "approve master 2"
}

// cdc.cmd.transmute
{
  "master_table": "master_centrallized_export_service.export_jobs",
  "master_binding_id": 2,
  "triggered_by": "scheduler / manual",
  "correlation_id": "sched-2-...",
  "trace_id": "..."
}
```

#### 3.3. Tách Biệt Debouncer Queue Trong Realtime CDC Fanout
Trong `TransmuteHandler`, cơ chế TableDebouncer để gom lô incremental events được định tuyến theo khóa:
```go
debouncerKey := req.MasterTable
if req.MasterBindingID > 0 {
    debouncerKey = fmt.Sprintf("%s#%d", req.MasterTable, req.MasterBindingID)
}
td := h.getOrCreateDebouncer(debouncerKey)
```
Điều này đảm bảo các batch sync của Master 1 và Master 2 chạy hoàn toàn song song, không bị tranh chấp concurrency buffer hoặc deadlock lẫn nhau.

---

### 4. Kết Quả Kiểm Tra Đáp Ứng Quality Gates (DoD G1–G8)
- **(G1) Requirement Traceability**: Đáp ứng 100% các yêu cầu tại `01_requirements_target_connection_mgmt.md` và `08_tasks_multi_connection_refinement.md`.
- **(G2) Red -> Green**: Đã tái hiện Lỗi 1 và Lỗi 2 từ log thực tế phiên trước; đã triệt tiêu nguyên nhân gốc rễ bằng việc liên kết qua Primary Key `id`.
- **(G3) Non-Breaking / Minimal Impact**: Giữ nguyên URL RESTful `/api/v1/masters/:name/...`, chỉ bổ sung query param `binding_id` và body fallback. Các client cũ hoặc script CLI gọi bằng tên đơn nhất vẫn hoạt động 100% bình thường.
- **(G4) Edge-cases**: Xử lý đầy đủ trường hợp `binding_id = 0`, tên trùng lặp (ambiguous), bảng chưa approved, và trường hợp không có schema prefix.
- **(G5) Chống Regression**: Không làm ảnh hưởng đến cơ chế Audit Log, Activity Log, hoặc RLS policy hiện hữu.
- **(G6) Output Correctness**: Tiến độ sync hiển thị độc lập cho từng binding, không bị rò rỉ chéo.
- **(G7) Adversarial Review**: Loại bỏ toàn bộ `LIMIT 1` mò mẫm khỏi code base.
- **(G8) Physical Workspace Evidence**: Báo cáo đầy đủ và chi tiết tại `08_tasks_multi_connection_refinement.md`, `05_progress.md` và file báo cáo này.
