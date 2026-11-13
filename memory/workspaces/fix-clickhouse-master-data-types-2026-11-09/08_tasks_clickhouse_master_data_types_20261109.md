# 08. Tasks — Kiểm tra & Chuẩn hóa Master Data Type cho ClickHouse

## Task List

### T1. Điều tra toàn diện Master Data Type cho ClickHouse (S2)
- **Mô tả:** Truy vết danh sách `Master Data Type` trên UI (`cdc-cms-web`), API tạo/sync/update rule (`cdc-cms-service`), và bộ sinh DDL / ép kiểu (`centralized-data-service`) khi đích là ClickHouse.
- **Context:** `cdc-cms-web/src/`, `cdc-cms-service/internal/`, `centralized-data-service/internal/service/master/`.
- **Logs/Test:** Kiểm tra tĩnh `file:line` trên cả 3 repo.
- **Definition of Done:** Xác định chính xác mọi vị trí đang dùng bộ kiểu PostgreSQL cho ClickHouse và cơ chế ánh xạ kiểu ở Backend/Worker.

### T2. Phản biện Pre-Plan (S2.5 — DG0) & Thiết kế Giải pháp + Code Demo (S3 — DG1/DG2)
- **Mô tả:** Chạy `laya_gate.py --stage dg0`, lập phương án tối ưu duy nhất trong `09_tasks_solution_clickhouse_master_data_types_20261109.md` kèm Code Demo 1-1, chạy `laya_gate.py --stage dg1` và trình User chờ `APPROVE`.
- **Context:** `09_tasks_solution_clickhouse_master_data_types_20261109.md`.
- **Logs/Test:** `laya_gate.py run --stage dg0` & `--stage dg1` đạt `PASS`.
- **Definition of Done:** Báo cáo rõ ràng câu trả lời cho User kèm Code Demo và dừng chờ `APPROVE`.

### T3. Thực thi Giải pháp Chuẩn hóa ClickHouse Master Data Type (S4)
- **Mô tả:** Triển khai bộ kiểu ClickHouse chuẩn trên `cdc-cms-web`, `cdc-cms-service`, và `centralized-data-service` sau khi có `APPROVE`.
- **Context:** `cdc-cms-web/src/constants/clickhouseDataTypes.ts`, `cdc-cms-web/src/constants/postgresDataTypes.ts`, `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`, `cdc-cms-service/pkgs/utils/dbutil/clickhouse_types.go`, `cdc-cms-service/pkgs/utils/dbutil/validate.go`, `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go`, `centralized-data-service/internal/service/shadow/type_resolver.go`, `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go`, `centralized-data-service/internal/service/master/transmuter_clickhouse.go`, `centralized-data-service/internal/service/master/transmuter_utils.go`.
- **Logs/Test:** `evidence_run.py` cho unit test, integration test trên `gpay-clickhouse` thật, và runtime query trên `goopay_clickhouse.transaction_history`.
- **Definition of Done:** Tách module đúng Single-Responsibility (`clickhouseDataTypes.ts`, `clickhouse_types.go`), giữ `MasterMappingFieldsPage.tsx <= 1388` dòng (`NOW <= HEAD`), xử lý trọn vẹn `bool -> UInt8` ở `ValidateValue` và `bulkUpsertClickHouseMaster`, tương thích ngược 100%.

### T4. Kiểm định Phản biện Từng Dòng & Vòng Lặp Phản Tỉnh `S5 -> S3 -> S4 -> S5` (DG3)
- **Mô tả:** Thực hiện 6 bước QC phản biện cuối workflow (`git diff`, `wc -l`, Anti-Smoke-Test Speculation, Zero-Mockup runtime trace `EV-...`, Self-Improvement Loop vào `lessons.md`, gọi `/laya_router` `DG3` cả `run` và `review`, xuất `audit_report_clickhouse_master_data_types_20261109.md`).
- **Context:** `audit_report_clickhouse_master_data_types_20261109.md`, `lessons.md`.
- **Logs/Test:** `laya_gate.py run --stage dg3`, `laya_gate.py review --stage dg3`, `verify_governance.py --workspace fix-clickhouse-master-data-types-2026-11-09 --suffix clickhouse_master_data_types_20261109`.
- **Definition of Done:** Toàn bộ phát hiện `G1..G3` được khắc phục dứt điểm, `DG3` đạt `PASS`, `verify_governance.py` đạt `PASSED 🟢`.
