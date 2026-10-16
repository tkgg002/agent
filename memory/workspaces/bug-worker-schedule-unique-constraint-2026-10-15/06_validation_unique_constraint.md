# 06 Validation & Quality Gate Report — bug-worker-schedule-unique-constraint-2026-10-15

## 1. Truy vết Yêu cầu & Kiểm chứng Gate G1–G8 (Rule #14)

| Gate | Tiêu chí kiểm định | Bằng chứng thực tế | Kết quả |
|---|---|---|---|
| **G1** | Truy vết Yêu cầu (`01_requirements_unique_constraint.md`) | Đã gỡ bỏ constraint phẳng `cdc_worker_schedule_operation_target_table_key`, tạo Unique Index 5 thành phần `uq_worker_schedule_source_count_scope` và chuẩn hóa `target_table` theo `<layer_type>:<connection_code>:<container_namespace>.<object_name>` đồng bộ từ DB, Backend, Worker đến Frontend. | **PASS** |
| **G2** | Reproduce trước khi Fix (Red → Green) | Trước fix: tạo 2 lịch trình `source_count` cho `shadow` và `master` cùng tên bảng `payment_bills` bị chặn bởi `SQLSTATE 23505` (`cdc_worker_schedule_operation_target_table_key`). Sau fix: tạo thành công cả 2 bản ghi (`HTTP 201 Created`). | **PASS** |
| **G3** | Test thật (Unit Test + Runtime Build + E2E HTTP) | Unit test `go test -v ./internal/app/commands/scheduler/...` PASS 4/4; Build Go & TypeScript 0 lỗi; E2E HTTP trên `:8083` PASS 100%. | **PASS** |
| **G4** | Edge-case & Negative-path | Test tạo trùng chính xác cả 5 thành phần `(operation, layer_type, connection_code, container_namespace, object_name)` trả về `HTTP 409 Conflict` kèm constraint `uq_worker_schedule_source_count_scope`. | **PASS** |
| **G5** | Chống Regression (Minimal-Impact) | Các tác vụ đơn tầng truyền thống (`bridge`, `transform`, `reconcile`, `field-scan`, `partition-check`) được bảo vệ bởi Partial Unique Index `uq_worker_schedule_legacy_op_target`; `source_count_runner.go` giữ nguyên `params.ObjectName` thuần khi chạy `countSQL`. | **PASS** |
| **G6** | Output Correctness trên dữ liệu thật | 7 bản ghi `source_count` hiện hữu trong `cdc_dw` (bao gồm `ID=13` `shadow:default_shadow:shadow_traitestpbs.payment_bills` và `ID=18` `master:default_master:master_payment_bill_service.payment_bills`) cùng tồn tại và hiển thị chuẩn xác. | **PASS** |
| **G7** | Adversarial Self-Review & Security Check | Không lộ secret/token trong log hoặc memory files (Rule #20); `sanitizeScheduleParamsForLog` và `MaskSensitiveScheduleParams` hoạt động nguyên vẹn. | **PASS** |
| **G8** | Bằng chứng vật lý trong Workspace | Lưu trữ đầy đủ log lệnh chạy và response thực tế trong file này và `05_progress.md`. | **PASS** |

---

## 2. Kết quả Unit Tests & Build

### 2.1. Backend Unit Tests (`cdc-cms-service`)
```text
$ go test -v ./internal/app/commands/scheduler/...
=== RUN   TestCanonicalScheduleTargetTable_DistinguishesShadowAndMasterSameTable
--- PASS: TestCanonicalScheduleTargetTable_DistinguishesShadowAndMasterSameTable (0.00s)
=== RUN   TestCanonicalScheduleTargetTable_PreservesLegacyOperations
--- PASS: TestCanonicalScheduleTargetTable_PreservesLegacyOperations (0.00s)
=== RUN   TestDeleteWorkerScheduleCommandValidation
--- PASS: TestDeleteWorkerScheduleCommandValidation (0.00s)
=== RUN   TestUpdateScheduleCommandValidationWithTargetTable
--- PASS: TestUpdateScheduleCommandValidationWithTargetTable (0.00s)
PASS
ok  	cdc-cms-service/internal/app/commands/scheduler	0.576s
```

### 2.2. Build Verification (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`)
- `cdc-cms-service`: `go build ./cmd/... ./internal/... ./pkgs/...` -> **Exit Code 0**
- `centralized-data-service`: `go build ./cmd/... ./internal/... ./pkgs/...` -> **Exit Code 0**
- `cdc-cms-web`: `npx tsc -b` -> **Exit Code 0**

---

## 3. Bằng chứng Migration & E2E HTTP Runtime trên `:8083`

### 3.1. Log tự động thực thi Migration `026_fix_worker_schedule_unique_scope.sql` khi khởi động `cdc-cms-service`
```json
{"level":"info","ts":1792049416.321824,"msg":"applying migration","version":"026_fix_worker_schedule_unique_scope"}
{"level":"info","ts":1792049416.35007,"msg":"migration applied","version":"026_fix_worker_schedule_unique_scope"}
{"level":"info","ts":1792049416.350092,"msg":"migrations done","total_files":95,"applied_now":1,"already_applied":94,"include_seeds":true}
```

### 3.2. Kết quả chạy E2E HTTP Verification (`GET` / `POST` Shadow / `POST` Master / `POST` Duplicate / `DELETE`)
```text
=== 1. LIST EXISTING SCHEDULES ===
GET /api/worker-schedule -> HTTP 200
  ID=18 op=source_count target_table=master:default_master:master_payment_bill_service.payment_bills
  ID=16 op=source_count target_table=master:default_master:master_payment_bill_service.payment_bills_1
  ID=10 op=source_count target_table=master:master_2:master_centrallized_export_service.export_jobs_2
  ID=8  op=source_count target_table=shadow:default_shadow:shadow_traitestces.export_jobs
  ID=13 op=source_count target_table=shadow:default_shadow:shadow_traitestpbs.payment_bills
  ID=9  op=source_count target_table=source:traitestces:centrallized-export-service.export-jobs
  ID=12 op=source_count target_table=source:traitestpbs:payment-bill-service.payment-bills

=== 2. CREATE SHADOW SCHEDULE FOR payment_bills ===
POST shadow payment_bills -> HTTP 201
target_table = "shadow:default_shadow:shadow_payment_service_e2e.payment_bills"

=== 3. CREATE MASTER SCHEDULE FOR payment_bills (SAME TABLE NAME, DIFFERENT LAYER) ===
POST master payment_bills -> HTTP 201
target_table = "master:default_master:master_payment_service_e2e.payment_bills"

=== 4. CREATE DUPLICATE MASTER SCHEDULE (NEGATIVE TEST: EXACT SAME 5-TUPLE) ===
POST duplicate master payment_bills -> HTTP 409
response = {"error": "ERROR: duplicate key value violates unique constraint \"uq_worker_schedule_source_count_scope\" (SQLSTATE 23505)"}

ALL E2E ASSERTIONS PASSED!

=== 5. CLEANUP CREATED TEST SCHEDULES ===
DELETE /api/worker-schedule/19 -> HTTP 200, response={'id': 19, 'message': 'schedule deleted'}
DELETE /api/worker-schedule/20 -> HTTP 200, response={'id': 20, 'message': 'schedule deleted'}
```
