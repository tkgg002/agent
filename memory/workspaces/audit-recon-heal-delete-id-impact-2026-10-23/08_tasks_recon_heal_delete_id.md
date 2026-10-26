# 08 Tasks: Audit & Khắc Phục Ảnh Hưởng Delete / `_source_id` / `_gpay_id` Tới Recon & Heal (A & B, 2h / 7d / 30d)

## Danh mục Task

- [x] **TASK-01 (S2 Investigate Runtime & Git Diff):**
  - **Context:** Bản ghi recon `id=7494` trên DB và các commit `30b1b00`, `08c05a8`, `43e5da7` tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` (`internal/service/master/transmuter.go`, `internal/handler/shadow/batch_buffer_utils.go`, `internal/service/recon/recon_engine_*.go`, `internal/handler/recon/recon_execute_heal_handler.go`, `internal/handler/recon/recon_check_heal_handler.go`).
  - **Logs/Test:** Truy vấn read-only bản ghi recon `7494` và đối chiếu dữ liệu thật trên Shadow vs Master để tìm nguyên nhân chính xác vì sao `4,536` vs `4,539` lại ra `missing=1534`, `orphan=1537`.
  - **Definition of Done:** Xác định chính xác 100% root cause với bằng chứng `file:line` và dữ liệu DB runtime thật (`VERIFIED`).

- [x] **TASK-02 (S2 Audit Toàn Diện Luồng Recon & Heal A/B và 3 Loại Recon 2h, 7d, 30d):**
  - **Context:** Toàn bộ pipeline `Source -> Shadow -> Master`, `ReconEngine` (Segment A & Segment B, 3 chiến lược/cửa sổ `2h`, `7d`, `30d` / `recon_tier`), và `Heal` (`recon_check_heal_handler.go`, `recon_execute_heal_handler.go`).
  - **Logs/Test:** Kiểm tra từng điểm chạm của `_gpay_id`, `_source_id`, `conflictTarget`, `_deleted` trên cả Segment A, Segment B và 3 loại recon (`2h`, `7d`, `30d`).
  - **Definition of Done:** Bóc tách đầy đủ mọi điểm đứt gãy/bất nhất vào `13_analysis_recon_heal_delete_id.md` và `09_tasks_solution_recon_heal_delete_id.md`.

- [x] **TASK-03 (S3 Solution & Decision Gate DG1/DG2/DG3):**
  - **Context:** `09_tasks_solution_recon_heal_delete_id.md` và `13_analysis_recon_heal_delete_id.md`.
  - **Logs/Test:** Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` và `--stage dg3` đạt `PASS`.
  - **Definition of Done:** Trình phương án tối ưu duy nhất kèm code demo và kết quả kiểm định Laya Gate cho User duyệt (`APPROVE`).

- [x] **TASK-04 (S4 Execute & S5 Audit sau khi có `APPROVE`):**
  - **Context:**
    1. Chỉ sửa tối giản, trúng đích đúng **3 file bị regression trong đợt commit `38ba1f4..08c05a8`** tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` (`internal/service/master/transmuter.go` + `internal/service/master/transmuter_utils.go`, `internal/handler/recon/recon_execute_heal_handler.go`, `internal/repository/master/master_binding_repo.go`). **Tuyệt đối không đụng chạm** các file Engine Recon/Shadow đang chạy ổn định (`recon_stream_bucket_engine.go`, `recon_tier_a.go`, `recon_tier_b.go`, `recon_heal_fetch.go`, `recon_heal.go`, `schema_adapter.go`).
    2. Loại bỏ toàn bộ lệnh `DeleteStream` lên các core stream (`CDC_TRANSMUTE`, `CDC_EVENTS`, `SCHEMA_DRIFT`, `SCHEMA_CONFIG`) trên NATS thật (`config-local.yml` `:14222`) trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/nats_monitor_handler_test.go:350-352` và `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/pkgs/natsconn/nats_client_test.go:55, 104-105`, đồng thời khôi phục consumer `transmute-shadow-consumer-v2` trên `dev`.
  - **Logs/Test:** Viết unit/integration test tái hiện 3 lỗi regression (Red -> Green), kiểm tra `http://localhost:18222/jsz?consumers=true` (`consumer_count >= 1` trên `CDC_TRANSMUTE`), chạy `evidence_run.py` lấy biên lai `EV-...`, chạy `laya_gate.py` `DG3` sau lần sửa cuối, xuất `audit_report_recon_heal_delete_id.md` và `verify_governance.py` đạt `PASSED`.
  - **Definition of Done:** Toàn bộ Gate `G1-G8` PASS.

- [x] **TASK-05 (S4 Execute & S5 Audit sau khi có `APPROVE` — Xóa bỏ 100% Fallback/Hardcode Cheat & Chuẩn hóa Config/Metadata Xuyên Suốt 3 Repo):**
  - **Context:**
    1. **`centralized-data-service`:**
       - Xóa `resolveBusinessPKForMaster` (`internal/service/master/transmuter_utils.go:751-768`); chỉ đọc PK bảng Master từ `Spec (JSON) *` (`resolveConflictTarget(binding, colTypes)`) trong `transmuter.go:1114-1146, 1686-1728` kèm kiểm tra `mapping_rule_master`.
       - Trong `internal/handler/recon/recon_execute_heal_handler.go:529, 545, 615, 690-723, 854, 875`: tách `resolveMasterPKFromSpec` (chỉ đọc `ref.TransformSpec.pk`, bỏ hoàn toàn fallback sang `entry.PrimaryKeyField` và `"_id"`) và dùng `"_source_id"` khi truy vấn bảng Shadow ở dòng `875`.
       - Trong `internal/repository/master/master_binding_repo.go:110`: xóa `IN ('', 'default', 'default_shadow')`, khớp chính xác theo `sb.binding_code` hoặc `scr.connection_code = @shadow_connection_key`.
       - Tại cả 5 nơi bắn `cdc.cmd.transmute-shadow` (`batch_buffer_fanout.go:74-82, 140-169`, `sinkworker/worker.go:369-425`, `recon_execute_heal_handler.go:339-392`, `recon_tier_a.go:580-590`, `batch_transform_handler.go:588-618`): truy xuất đúng `(binding_code, shadow_schema, shadow_connection_key)` từ `cdc_system.shadow_binding` JOIN `cdc_system.connection_registry`.
       - Bổ sung `ShadowConnectionKey`, `ShadowBindingCode` vào `MasterBindingRef` (`recon_engine_segment_b.go:19-33, 77-99`) và chuẩn hóa `ConnectionManager.GetMasterDB`/`GetShadowDB` (`connection_manager.go:53-129`).
    2. **`cdc-cms-service`:**
       - Cưỡng chế `transform_spec` (`Spec (JSON) *`) luôn có `"pk"` khác rỗng trong `create_master.go:64-79, 114-117`, `update_master_spec.go:38-56`, `master_repo_gorm.go:409-418`, và `provisioning_orchestrator.go:447-471`.
       - Chuẩn hóa `ApproveSchemaTx`, `CloneMappingRules`, `ResolveMasterConnection` (`master_repo_gorm.go:315-334, 476-489, 502-543`), `EnsureDefaultMasterConnection` (`bootstrap/master_connection.go:11-28`), và trả thêm `shadow_connection_code`, `primary_key_field` trong `ListShadowBindings` (`source_object_read_repo_gorm.go:410-495`).
    3. **`cdc-cms-web`:**
       - Trong `src/pages/MasterRegistry.tsx:291, 295, 591, 771, 1303-1315, 1364-1377, 1401-1407, 1600-1604`: tự động điền `"pk"` từ `selectedShadow.primary_key_field`, chặn submit nếu `Spec (JSON) *` thiếu `"pk"`, và dùng `connection_code` thật từ `targetMasterConnections`.
       - Trong `src/pages/ActivityManager.tsx:629, 739, 893`: đọc `sb.shadow_connection_code` từ API thay vì hardcode `'default_shadow'`.
  - **Logs/Test:** Chạy full unit/integration test trên cả 3 repo qua `evidence_run.py` (`EV-...`), chạy `laya_gate.py` `DG3` sau lần sửa cuối, cập nhật `audit_report_recon_heal_delete_id.md` và chạy `verify_governance.py` đạt `PASSED`.
  - **Definition of Done:** Toàn bộ Gate `G1-G8` PASS, sạch bóng 100% fallback/hardcode cheat trên cả 3 repo.

- [x] **TASK-06 (S5 -> S3 -> S4 -> S5 Autonomous Self-Healing Remediation — Sửa lỗi `mb.shadow_connection_id` trên PostgreSQL thật, Xóa bỏ Mock Schema & Các Fallback Còn Sót):**
  - **Context:**
    1. `centralized-data-service/internal/repository/master/master_binding_repo.go:83, 102`, `internal/handler/shadow/batch_buffer_fanout.go:180`, `internal/sinkworker/worker.go:436`: sửa `COALESCE(mb.shadow_connection_id, sb.shadow_connection_id)` thành `sb.shadow_connection_id` (vì bảng `cdc_system.master_binding` trên PostgreSQL thật không có cột `shadow_connection_id`) và sửa `COALESCE(cr.connection_code, 'default')` ở dòng `83` thành `COALESCE(cr.connection_code, '')`.
    2. `centralized-data-service/internal/handler/shadow/batch_buffer_fanout_test.go`: xóa cột `shadow_connection_id` giả lập trên `master_binding` và bổ sung bài test tích hợp chạy trực tiếp trên PostgreSQL thật (`gpay-postgres-cdc` `:5433`) cho `MasterBindingRepo.ListMasterTargetsByShadowIdentity`, `BatchBuffer.resolvePostIngestSchedule`, `SinkWorker.resolvePostIngestSchedule`, và `ReconCore.ResolveShadowBindingIdentity`.
    3. `cdc-cms-service/internal/infra/persistence/governance/provisioning_orchestrator.go:383, 435-446`: bỏ fallback `'_id'`, trả lỗi nếu `src.PrimaryKeyField == ""`, và khôi phục 11 dòng comment giải thích `ON CONFLICT (binding_code)`.
    4. `cdc-cms-web/src/pages/MasterRegistry.tsx:1336, 1442` & `src/pages/ActivityManager.tsx:629, 701, 740, 894`: bỏ fallback `|| '_id'` và `|| 'default_shadow'` / `|| 'default_master'`.
  - **Logs/Test:** Chạy integration test trực tiếp trên PostgreSQL thật (`gpay-postgres-cdc` `:5433`) và unit test/build trên cả 3 repo qua `evidence_run.py` (`EV-...`), chạy `laya_gate.py run --stage dg3` và `laya_gate.py review` sau lần sửa cuối, cập nhật `audit_report_recon_heal_delete_id.md` và chạy `verify_governance.py`.
  - **Definition of Done:** Không còn bất kỳ lỗi SQL nào trên PostgreSQL thật, không dùng mock schema sai lệch, pass 100% `DG1`, `DG3` (Mode A + Mode B) và `verify_governance.py`.

