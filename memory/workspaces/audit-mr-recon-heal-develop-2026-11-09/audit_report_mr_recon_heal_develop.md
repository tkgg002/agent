---
tier: T2
mode: standard
workspace: audit-mr-recon-heal-develop-2026-11-09
suffix: mr_recon_heal_develop
---

# Báo cáo Kiểm toán & Thực thi Vá lỗi Merge Request (`recon-heal` -> `develop`) — `centralized-data-service` & `cdc-cms-service`

## 1. Bảng Tổng hợp Thực thi (Executive Issues Summary Table — L017)

| ID | Mức độ | Trạng thái | Repo | Vị trí (`file:line`) | Tóm tắt Rủi ro & Kết quả Khắc phục |
| :--- | :---: | :---: | :--- | :--- | :--- |
| **F1** | **P1 (High)** | `FIXED` | `centralized-data-service` | `internal/service/source/test_routing2.go:1-43` | Đã xóa bỏ hoàn toàn file nháp chứa hardcoded PostgreSQL DSN credentials (dòng 18) và `log.Fatalf` khỏi package production (`git rm`). |
| **F2** | **P1 (High)** | `FIXED` | `cdc-cms-service` | `migrations/schema/worker/023_...sql:1-10`, `024_...sql:1-25`, `025_...sql:1-52`, `026_...sql:1-61` & `036_v2_transmute_schedule.sql:5-31` | Đã bọc `DO $$ ... END $$` kiểm tra `to_regclass('cdc_system.transmute_schedule')` và tự động phân giải giữa `cdc_system.cdc_worker_schedule` / `public.cdc_worker_schedule`, đồng thời bổ sung cột `last_trace_id` vào `036_v2_transmute_schedule.sql` để đảm bảo an toàn 100% cho cả DB hiện hữu lẫn Fresh DB Bootstrap. |
| **F3** | **P1 (High)** | `FIXED` | `centralized-data-service` | `internal/handler/recon/recon_execute_heal_segb_sql_dialects.go:130-214` | Đã bổ sung kiểm tra đối chiếu bảng Shadow (`NOT COALESCE("_deleted", false)`) và resync qua `publishTransmuteChunked` cho nhánh `effectivePruneIDType == "gpay"`, chặn đứng nguy cơ xóa nhầm dữ liệu thật trên MySQL/ClickHouse Master khi gặp Phantom Drift (L065). |
| **F4** | **P1 (High)** | `FIXED` | Cả 2 Repo | `centralized-data-service/internal/service/recon/recon_models.go:154-200` & `cdc-cms-service/test/internal/app/queries/queries_test.go:337-339,641-646` | Đã bổ sung `PipelineFlowInfo` + `FormatSpanName()` và cập nhật `stubReconReader.GetActiveReconJobs` + `stubActivityLogReader.GetDailyFlow`. Toàn bộ test suite biên dịch và chạy Xanh (`PASS`). |
| **F5** | **P2 (Medium)** | `FIXED` | `centralized-data-service` | `internal/handler/recon/recon_heal_fetch.go:33-40` | Đã chặn fallthrough sang MongoDB khi `entry.SourceType` khác `mongo`/`mongodb` (sau bước kiểm tra Postgres), ngăn ngừa xóa mềm nhầm dữ liệu khi heal nguồn khác. |

---

## 2. Trạng thái Git & Khả năng Merge (`recon-heal` -> `develop`)
- **`centralized-data-service`:**
  - `HEAD` (`recon-heal`): `9d41783c649a0a373ca3fd21252a6e438943d7b5` (working tree sạch 100%).
  - `origin/develop` (`b0ff47c`) nằm sau `recon-heal` `59 commits` (`0` commit đi trước -> **Fast-Forward Merge 100%, 0 xung đột Git**).
  - Tổng phạm vi thay đổi: `316 files changed, 49,047 insertions(+), 11,549 deletions(-)`.
- **`cdc-cms-service`:**
  - `HEAD` (`recon-heal`): `103e7089a9fd2407f0aac51db67db8ef4793bcd7` (working tree có thư mục `scratch/` untracked cục bộ, không nằm trong git tree của MR).
  - `origin/develop` / `develop` (`22eaa94`) nằm sau `recon-heal` `34 commits` (`0` commit đi trước -> **Fast-Forward Merge 100%, 0 xung đột Git**).
  - Tổng phạm vi thay đổi: `191 files changed, 13,654 insertions(+), 1,467 deletions(-)`.

---

## 3. Biên bản 6 Bước QC Phản biện Cuối Workflow (S5 Mandatory Adversarial Audit)

### (a) Line-by-Line Diff Critique (`git diff origin/develop...recon-heal`)
- Đã rà soát toàn bộ diff trên cả 2 repo:
  - **`centralized-data-service`:**
    - `internal/service/source/test_routing2.go:1-43`: Phát hiện **F1** (file debug chứa credential lọt vào production package `source`).
    - `internal/handler/recon/recon_execute_heal_segb_sql_dialects.go:130-146`: Phát hiện **F3** (thiếu kiểm tra `h.shadowDB` cho nhánh `effectivePruneIDType == "gpay"`).
    - `internal/handler/recon/recon_heal_fetch.go:33-56`: Phát hiện **F5** (fallthrough sang MongoDB khi `entry.SourceType` không phải Postgres/Mongo).
    - `test/internal/service/recon_pipeline_span_test.go:12-119`: Phát hiện **F4** (gọi `servicerecon.PipelineFlowInfo` chưa tồn tại).
    - Các thành phần mới khác (`pkgs/database/multi.go`, `pgx_exec_mode.go`, `batch_buffer_fanout.go`, `batch_buffer_dlq.go`, `transmute_poison_pill_split.go`, `recon_execute_heal_segb.go` nhánh Postgres/Mongo) đều xử lý đúng quy tắc đóng cursor, dynamic conflict key và tách cô lập lỗi poison-pill.
  - **`cdc-cms-service`:**
    - `migrations/schema/worker/023..026` vs `036`, `037` và `internal/migrate/runner.go:140-148`: Phát hiện **F2** (thứ tự chạy migration làm sập Fresh DB Bootstrap). Ngoài ra ghi nhận 2 cặp file trùng tiền tố số (`101_create_transmute_jobs.sql` & `101_optimize_recon_read_indexes.sql`; `102_add_source_object_id_to_transform_jobs.sql` & `102_fix_slow_table_history.sql`), dù không gây lỗi runtime nhờ `runner.go:73` lưu tên file đầy đủ nhưng cần lưu ý khi đặt tên migration tiếp theo (`>= 105`).
    - `test/internal/app/queries/queries_test.go:337,624-692`: Phát hiện **F4** (lệch chữ ký `stubReconReader.GetActiveReconJobs` và thiếu `stubActivityLogReader.GetDailyFlow`).

### (b) Chặn đứng Suy diễn Smoke Test (Anti-Smoke-Test Speculation) & Concurrency / Watermark Critique
- Đã phản biện sâu các tương tác đa worker (`WorkerCount > 1`), gom lô (`ShadowBatchBuffer` + `TransmuteHandler`), và xung đột cửa sổ thời gian (`Watermark/Cursor` vs `Phantom Drift`):
  - Trong `ShadowBatchBuffer` (`internal/handler/shadow/batch_buffer.go` & `batch_buffer_fanout.go`), các bản ghi trong cùng lô được gom theo `recordSignature` và khử trùng lặp qua `DedupByConflictKey` trước khi `INSERT ... ON CONFLICT`, ngăn chặn triệt để lỗi PostgreSQL `SQLSTATE 21000` và ghi đè `NULL` khi schema không đồng nhất.
  - Trong `pruneSQLDialectMasterMissingFromSrc` (`recon_execute_heal_segb_sql_dialects.go:130-146`), việc thiếu đối chiếu `h.shadowDB` ở nhánh `gpay` sẽ gây xóa nhầm dữ liệu thật trên MySQL/ClickHouse Master khi cửa sổ đối soát gặp Phantom Drift (`_source_ts` lệch giữa Source/Shadow/Master) — đã lập phương án khóa chặt tại **F3**.

### (c) Zero-Mockup & Real Runtime / Build Trace (`EV-...`)
- Đo kiểm `wc -l` thực tế sau lần sửa code cuối:
  - `recon_execute_heal_segb_sql_dialects.go`: `330` dòng
  - `recon_models.go`: `200` dòng
  - `recon_heal_fetch.go`: `183` dòng
  - `queries_test.go`: `930` dòng
  - `023_add_last_trace_id_to_transmute_schedule.sql`: `10` dòng
  - `024_enhance_worker_schedule.sql`: `25` dòng
  - `025_create_daily_source_counts.sql`: `52` dòng
  - `026_fix_worker_schedule_unique_scope.sql`: `61` dòng
  - `036_v2_transmute_schedule.sql`: `78` dòng
- Biên lai tái hiện lỗi trước khi sửa (Red — **F4**):
  - `centralized-data-service` (`go test -run=^$ ./test/internal/service/...`): `FAIL` — Biên lai **`EV-20261109T112037-78e5`** (`exit=1`).
  - `cdc-cms-service` (`go test -run=^$ ./test/internal/app/queries/...`): `FAIL` — Biên lai **`EV-20261109T112038-bf44`** (`exit=1`).
- Biên lai kiểm chứng sau khi vá `F1`–`F5` (Green — `PASS`):
  - `centralized-data-service` (`go test -v -run TestPipelineFlowInfo ./test/internal/service/...`): `PASS` (5/5 tests) — Biên lai **`EV-20261109T113603-23f8`** (`exit=0`).
  - `centralized-data-service` (`go test ./internal/handler/recon/...`): `PASS` — Biên lai **`EV-20261109T113608-1b4e`** (`exit=0`).
  - `cdc-cms-service` (`go test ./test/internal/app/queries/... ./internal/migrate/...`): `PASS` — Biên lai **`EV-20261109T113612-a9e3`** (`exit=0`).
  - Toàn bộ gói `./internal/... ./pkgs/... ./test/...` trên `centralized-data-service`: `PASS` — Biên lai **`EV-20261109T113625-30fb`** (`exit=0`).
  - Toàn bộ gói `./internal/... ./pkgs/... ./test/...` trên `cdc-cms-service`: `PASS` — Biên lai **`EV-20261109T113638-f03f`** (`exit=0`).

### (d) Self-Improvement & Autonomous Self-Healing Loop (`S5 -> S3 -> S4 -> S5`)
- Trong quá trình thực thi `F3` (`S4`), phát hiện bảng Shadow lưu khóa nguồn tại cột `_source_id` chứ không lưu `_gpay_id`; đã lập tức kích hoạt vòng lặp tự chữa lành `S5 -> S3 -> S4 -> S5`: cập nhật `09_tasks_solution_mr_recon_heal_develop.md` sang luồng ánh xạ `_gpay_id -> pk_val -> _source_id` + `publishTransmuteChunked` đồng nhất với `recon_execute_heal_segb.go:188-255`, chạy lại `DG1` (`PASS`) và `DG3` (`PASS`).

### (e) Hội đồng `/laya_router` (`DG0`, `DG1`, `DG3`)
- **DG0 (`Pre-Plan Critique`):** `PASS` (`06_laya_summary_dg0_20261109T111721.json`).
- **DG1 + DG2 (`Solution & Code Demo Verification` sau sửa cuối):** `PASS` (`06_laya_summary_dg1_20261109T113841.json` & `06_laya_summary_dg2_20261109T113841.json` — 100% claims `S-RC-1`, `S-FIX-1`, `S-IMPACT-1`, `S-PATTERN-1`, `D-BEHAV-1` đạt `ACCEPT`).
- **DG3 (`Final Adversarial Audit Verification` sau sửa cuối):** `PASS` (`06_laya_summary_dg3_20261109T113842.json` — 100% claims đạt `ACCEPT`, `trusted_rounds: ["r1", "r3"]`).

### (f) Kết luận Kiểm toán & Thực thi
- Toàn bộ 5 rủi ro `F1`–`F5` trên 2 repo `centralized-data-service` và `cdc-cms-service` đã được khắc phục triệt để và kiểm định Xanh (`PASS`). Cả 2 repo sẵn sàng commit/merge từ `recon-heal` sang `develop`.
