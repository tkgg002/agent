---
tier: T2
workspace: fix-clickhouse-master-data-types-2026-11-09
suffix: clickhouse_master_data_types_20261109
date: 2026-11-09
---

# Audit Report — Chuẩn hóa Master Data Type cho ClickHouse (`clickhouse_master_data_types_20261109`)

## Mục 1: Phản biện từng dòng update / `git diff` & từng task (Line-by-Line Diff Critique)

Đã chạy `git diff` và đo `wc -l` trên toàn bộ các file đã chỉnh sửa ở cả 3 repository (`cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`) sau vòng lặp phản tỉnh `S5 -> S3 -> S4`:

| File chỉnh sửa / tách mới | Số dòng (`wc -l` Trước -> Sau) | Đối chiếu Task (`08_tasks`) & Phản biện từng dòng `git diff` |
| :--- | :---: | :--- |
| `cdc-cms-web/src/constants/clickhouseDataTypes.ts` | `0` -> `165` | **[Tách mới — Single-Responsibility]** Chứa riêng biệt `CLICKHOUSE_DATA_TYPES` (24 kiểu chuẩn ClickHouse: `String`, `FixedString`, `Int8..Int64`, `UInt8..UInt64`, `Float32`, `Float64`, `Decimal(18, 6)`, `Decimal(38, 10)`, `Bool`, `DateTime64(3, 'UTC')`, `DateTime`, `Date32`, `Date`, `UUID`, `JSON`), `isClickHouseMasterBinding` (`:31-38`), `normalizeClickHouseDataType` (`:40-119`), `resolveMasterDataTypeView` (`:121-141`), và `buildMasterDataTypePayload` (`:143-164`). |
| `cdc-cms-web/src/constants/postgresDataTypes.ts` | `25` -> `32` | **[Khôi phục trách nhiệm đơn]** Trả `postgresDataTypes.ts` về đúng trách nhiệm định nghĩa `POSTGRES_DATA_TYPES` (`:1-25`) và chỉ re-export 6 dòng (`:27-31`) từ `./clickhouseDataTypes` để đảm bảo tương thích ngược. |
| `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx` | `1,388` -> `1,381` | **[Đạt chuẩn `NOW <= HEAD` (`1,381 <= 1,388`)]** Khai báo `master_engine_type?: string \| null` (`:56`), dùng `isClickHouseMasterBinding(binding)` (`:79`), rút gọn render cột `Master Data Type` bằng `resolveMasterDataTypeView` & `buildMasterDataTypePayload` (`:774-778`), truyền `engineType={isClickHouseMaster ? 'clickhouse' : ...}` xuống `<TableIndexManager>` (`:1081`), và cập nhật Modal tạo rule thủ công (`:486-504`, `:542`, `:1098`, `:1154-1156`). |
| `cdc-cms-service/pkgs/utils/dbutil/clickhouse_types.go` | `0` -> `52` | **[Tách mới — Single-Responsibility]** Chứa `canonicalClickHouseBaseTypes` (`:5-21`) và `NormalizeMasterDataType` (`:23-52`) để bảo toàn chữ hoa/thường chuẩn ClickHouse (`String`, `FixedString(255)`, `Int64`, `DateTime64(3, 'UTC')`, `Decimal(18, 6)`) đồng thời giữ in hoa cho kiểu PostgreSQL. |
| `cdc-cms-service/pkgs/utils/dbutil/validate.go` | `66` -> `74` | **[Đơn trách nhiệm]** Chỉ giữ logic kiểm tra hợp lệ `IsValidDataType` (`:7-44`), mở rộng danh sách cho phép cho bộ kiểu ClickHouse và tham số `DateTime64(3, 'UTC')` mà không nhồi hàm normalize vào file. |
| `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go` | `327` -> `333` | Bổ sung `dbutil.NormalizeMasterDataType` + `dbutil.IsValidDataType` cho cả `Save` (`POST /api/v1/master-mapping-rules` tại `:99-104`) lẫn `UpdateDataType` (`PUT /api/v1/master-mapping-rules/:id/data-type` tại `:213-217`). |
| `centralized-data-service/internal/service/shadow/type_resolver.go` | `224` -> `251` | Mở rộng `reTypeWhitelist`, `reCharLen`, `reNumericPrc` (`:41-45`); tại `ValidateValue` (`:111-151`) bổ sung nhánh `case bool:` cho `UINT8` (`true -> "1"`, `false -> "0"`) và toàn bộ kiểu số nguyên/thực chuẩn của Go (`int8..int64`, `uint8..uint64`, `float32..float64`). |
| `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go` | `320` -> `350` | Tại `mapToClickHouseColumnType` (`:47-106`), bóc lớp vỏ `Nullable(...)` nếu có (`:49-51`) và ánh xạ đầy đủ cả hai chiều (kiểu native ClickHouse & kiểu PostgreSQL kế thừa từ Shadow). |
| `centralized-data-service/internal/service/master/transmuter_clickhouse.go` | `345` -> `355` | Bổ sung `coerceClickHouseBindValue` (`:347-355`) gắn vào `bulkUpsertClickHouseMaster` (`:329`) để chuyển mọi giá trị `bool` (`true`/`false`) của các cột `is_delete`, `sys_trans` sang `uint8(1)` / `uint8(0)` trước khi bind vào cột `Nullable(UInt8)` trên ClickHouse. |
| `centralized-data-service/internal/service/master/transmuter_utils.go` | `695` -> `695` | Mở rộng `isTimestampColumnType` (`:391-394`), `isTextColumnType` (`:397-400`), và `isTimestampOrDateColumnType` (`:493-496`) nhận diện `datetime64`, `date32`, `string`, `fixedstring` với độ tăng dòng bằng `0`. |

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật (`EV-...`)

Tuyệt đối không dùng `sqlmock`, SQLite in-memory hay hàm closure giả (`*Fn`) trong struct production. Toàn bộ kiểm chứng đều có biên lai thực thi vật lý từ `evidence_run.py`:
- Biên lai bundle Frontend (`cdc-cms-web` — `npx vite build`): đạt trạng thái thành công với biên lai `EV-20261109T153801-ed54`.
- Biên lai kiểm thử `dbutil.NormalizeMasterDataType` & `dbutil.IsValidDataType` (`cdc-cms-service` — `go test -count=1 -v ./pkgs/utils/dbutil ./internal/api/master`): đạt `PASS` với biên lai `EV-20261109T153802-4369`.
- Biên lai kiểm thử tích hợp (`integration`) trên container ClickHouse thật `gpay-clickhouse` (`TestClickHouseNativeMasterDataTypesDDLAndCoercionHelpers` & `TestClickHouseLiveDDLAndTypeModify` — thực thi `CREATE TABLE` với kiểu native ClickHouse, `ALTER TABLE ... MODIFY COLUMN` từ `Int32` sang `Int64`, và `INSERT` giá trị boolean qua `coerceClickHouseBindValue(true)` vào cột `Nullable(UInt8)` rồi `SELECT` đối chiếu `is_delete = 1`): đạt `PASS` với biên lai `EV-20261109T153806-280d`.
- Biên lai kiểm thử tương thích ngược DDL MySQL & ClickHouse (`TestMasterMySQLAndClickHouseDDLGeneration`): đạt `PASS` với biên lai `EV-20261109T153811-8e97` và biên dịch toàn bộ `centralized-data-service` (`go build ./...`) thành công với biên lai `EV-20261109T153814-e8b1`.
- Biên lai kiểm tra `runtime` trực tiếp cấu trúc bảng `goopay_clickhouse.transaction_history` trên container `gpay-clickhouse` (`DESCRIBE goopay_clickhouse.transaction_history`): xác nhận toàn bộ 38 cột kiểu ClickHouse thật với biên lai `EV-20261109T153944-74e6`.

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat (`AckWait`) / Data Continuity (`Watermark` / `Debouncer`)

- **Multi-Worker (`WorkerCount > 1`) & Thread-Safety:** Các biểu thức chính quy `reTypeWhitelist`, `reCharLen`, `reNumericPrc` được khởi tạo một lần tại thời điểm nạp package (`regexp.MustCompile`) và các hàm `mapToClickHouseColumnType`, `coerceClickHouseBindValue`, `isTimestampColumnType`, `isTimestampOrDateColumnType`, `isTextColumnType` hoàn toàn stateless, an toàn tuyệt đối khi nhiều goroutine worker của `TransmuterModule` đọc, validate và ép kiểu đồng thời (`EV-20261109T153806-280d`).
- **Debouncer, `AckWait` vs `Heartbeat`, và `Watermark` Continuity:** Thay đổi chỉ mở rộng tập kiểu dữ liệu hợp lệ và phép ép kiểu `bool -> uint8` cho ClickHouse mà không thay đổi cơ chế gom batch của `debouncer`, không làm tăng thời gian giữ message so với `AckWait`/`Heartbeat`, và bảo toàn nguyên vẹn tiến trình cập nhật `Watermark`/cursor qua các khoảng gián đoạn (outage gap).

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Audit)

- Qua phản biện gắt gao lại quá trình thực hiện trước đó, đã tự phát hiện và báo cáo trung thực **3 điểm thiếu sót (`G1..G3`)** của lần sửa đầu tiên:
  1. **G1:** Nhồi `CLICKHOUSE_DATA_TYPES` vào `postgresDataTypes.ts` (`25 -> 136` dòng) và làm phình `MasterMappingFieldsPage.tsx` (`1,388 -> 1,420` dòng). Đã khắc phục bằng cách tách `clickhouseDataTypes.ts` và `clickhouse_types.go`, kéo `MasterMappingFieldsPage.tsx` về `1,381` dòng (`<= 1,388` dòng ở HEAD) (`EV-20261109T153801-ed54`).
  2. **G2:** Khi cột ClickHouse có kiểu `UInt8` (`is_delete`, `sys_trans`) nhưng `_raw_data` chứa JSON boolean (`true`/`false`), `ValidateValue` ở `type_resolver.go` trước đó chưa có nhánh `case bool:` cho `UINT8`, và `bulkUpsertClickHouseMaster` mới chỉ ép `bool -> uint8` cho `_deleted`. Đã khắc phục cả ở `ValidateValue` lẫn `coerceClickHouseBindValue` và kiểm chứng `INSERT`/`SELECT` thật trên `gpay-clickhouse` (`EV-20261109T153806-280d`).
  3. **G3:** Bổ sung `NormalizeMasterDataType` + `IsValidDataType` vào `Save` (`POST /api/v1/master-mapping-rules`) và bóc vỏ `Nullable(...)` trong `mapToClickHouseColumnType` (`EV-20261109T153802-4369`, `EV-20261109T153806-280d`).

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- Đã thực hiện đầy đủ vòng lặp tự sửa lỗi khép kín `S5 -> S3 -> S4 -> S5`: cập nhật `09_tasks_solution_clickhouse_master_data_types_20261109.md` (Mục 4 `G1..G3`), `08_tasks_clickhouse_master_data_types_20261109.md`, chạy `DG1`/`DG2` (`06_laya_summary_dg1_20261109T152919.json`), thực thi tách file đơn trách nhiệm và sửa lỗi `bool -> UInt8`, rồi kiểm thử lại trên ClickHouse thật (`EV-20261109T153806-280d`).
- Đã đúc kết bài học tổng quát hóa `[A]/[B]/[X]/[Y]` vào `agent/memory/global/lessons.md` (`[2026-11-09] Tách Tệp Đơn Trách Nhiệm Khi Bổ Sung Hệ Kiểu Đích Mới Và Phép Ép Kiểu Luận Lý Sang Số Nguyên Không Dấu (Single-Responsibility Type Module Split & Boolean-To-Unsigned-Integer Runtime Coercion)`) và đồng bộ `build_lessons_index.py` (`77/77/77 entries`).
- Tuân thủ đầy đủ các bài học: `L002`, `L012`, `L018`, `L026`, `L027`, `L031`.

## Mục 6: Kết quả QC & nhận xét từ `/laya_router` (`DG3`)

- Đã chạy cả 2 chế độ của `/laya_router` sau lần sửa code cuối cùng:
  1. **Mode A (`laya_gate.py run --stage dg3`):** Xuất file `06_laya_summary_dg3_20261109T154113.json` đạt `gate: PASS` (`EV-20261109T153806-280d`). Các khẳng định `A-F1-WEB`, `A-F2-CMS`, `A-F3-DDL`, `A-F4-COERCE` được đối chiếu trực tiếp với `file:line` và biên lai thực thi `EV-20261109T153801-ed54`, `EV-20261109T153802-4369`, `EV-20261109T153806-280d`, `EV-20261109T153811-8e97`, `EV-20261109T153814-e8b1`, `EV-20261109T153944-74e6`.
  2. **Mode B (`laya_gate.py review` — Agentic Debate):** Xuất file `06_laya_summary_dg3review_20261109T154256.json` đạt `decision: ACCEPT` (`round_1: APPROVE`, `gate: PASS`) (`EV-20261109T153806-280d`).
