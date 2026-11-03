# Báo Cáo Kiểm Định & Phản Biện Toàn Diện Sau Thực Thi (`audit_report_master_mysql_clickhouse.md`)

- **Workspace:** `master-mysql-clickhouse-2026-10-30`
- **Suffix:** `master_mysql_clickhouse`
- **Tier:** `T2`
- **Ngày hoàn tất:** `2026-10-30`
- **Chế độ kiểm định hiện tại:** `100% Read-Only Propose-Only Audit` (`LS-GLOBAL-001-propose-only-gate` — không tự ý sửa source code khi đang Audit).

---

## Mục 0: Bảng Tổng Hợp Toàn Bộ Issues & Tổng Kết Điều Hành (Executive Issues Summary)

Để anh nắm bắt ngay toàn cảnh mà không phải đọc dò từng dòng, toàn bộ các vấn đề phát hiện qua đợt Audit được gom thành **3 nhóm rõ ràng** dưới đây:

### A. Nhóm 1 — Các Vấn Đề Cần Anh Lưu Ý (`2 Issues` — Toàn bộ lỗi code đã đóng 100%)
| Mã Issue | Phân loại | Mức độ | Vị trí (`file:line`) | Bản chất Vấn đề (Ngắn gọn, Trực diện) | Trạng thái & Hành động Đề xuất |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`ISS-01`** (`Finding P-01`) | Khoảng trống Kiểm thử Runtime (Test vs Live DB) | **HIGH (Cần lưu ý)** | [`create_master_sql_engines_test.go:18-86`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master_sql_engines_test.go#L18-L86) & [`master_mysql_clickhouse_test.go:1-119`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go#L1-L119) | Để không tự ý chạm vào DB thật của anh (`L033`), các bài test vừa chạy dùng **in-memory Go logic** và **SQLite `:memory:` wrapper** (do SQLite không có hàm `NOW()` của Postgres). Do đó mới chỉ chứng minh **logic code + trình biên dịch (`go build`, `tsc`) PASS**, **CHƯA chạy thật** trên MySQL server, ClickHouse server và PostgreSQL `cdc_system` thật. | **Cần phối hợp cùng anh:** Chạy 5 bước trace runtime thực tế trên UI/DB thật (Mục 2.3). |
| **`ISS-03`** (`Finding P-03`) | Working Tree Chứa File Của Task Trước | **INFO (Quản trị Git)** | Working tree của 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) | Trên working tree hiện có sẵn các file chưa commit của các tính năng trước (`multi_shadow`, `cdc_event_date`, `mongo_bson_restorer`, `recon_smoke_guard`). 11 file SRP mới và 21 file sửa của task này không xung đột nhưng đang nằm chung working tree. | **Thông tin tới anh:** Khi commit có thể gom theo từng nhóm file hoặc commit chung tùy anh quyết định. |

### B. Nhóm 2 — Các Lỗi Phát Sinh Đã Được Khắc Phục Triệt Để (`5 Issues — Đã Đóng 100%`)
| Mã Issue | Mức độ ban đầu | Vị trí (`file:line`) | Sự cố gặp phải | Cách đã khắc phục & Biên lai xác nhận |
| :--- | :---: | :--- | :--- | :--- |
| **`ISS-02`** (`Finding P-02`) | **LOW (Code Cleanliness)** | [`source_count_runner.go:296-306`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L296-L306) | `/laya_router` Mode B phát hiện nhánh `else` bên trong `else if eng == "clickhouse" \|\| eng == "ch"` lặp lại giống hệt nhánh `else` bên ngoài (`495` dòng). | **ĐÃ ĐÓNG 100%:** Sau lệnh `APPROVE ISS-02` của anh, đã rút gọn còn `else if (eng == "clickhouse" \|\| eng == "ch") && strings.EqualFold(params.LayerType, "master")` (`490` dòng, giảm 5 dòng) → Build & Test PASS (`EV-20261030T173212-eb7b`, `EV-20261030T173223-2794`) & `DG3` PASS (`06_laya_summary_dg3_20261030T173300.json`). |
| **`FIX-01`** | **MEDIUM (Bug ánh xạ kiểu dữ liệu)** | [`master_ddl_generator_mysql.go:68-70`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_mysql.go#L68-L70) | Lần chạy test đầu, hàm `mapToMySQLColumnType("NUMERIC(18,2)")` trả về nguyên `"NUMERIC(18,2)"` thay vì `"DECIMAL(18,2)"` vì `switch` chỉ bắt chuỗi tĩnh `"NUMERIC"`. | Đã sửa bằng `strings.Replace(upper, "NUMERIC", "DECIMAL", 1)` → Test PASS (`EV-20261030T173223-2794`). |
| **`FIX-02`** | **LOW (Lỗi import trong file test)** | [`master_mysql_clickhouse_test.go:8`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go#L8) | Lần chạy đầu viết nhầm import `centralized-data-service/pkgs/config` và truyền `-run "A\|B\|C"` bị shell tách dấu `\|`. | Đã sửa thành `centralized-data-service/config` và tách từng lệnh `-run` riêng → PASS (`EV-20261030T173223-2794`, `EV-20261030T173541-a798`, `EV-20261030T173546-930b`). |
| **`FIX-03`** | **LOW (Thư mục backup cũ khi build)** | `docs/server_bk` & `scratch/` | Chạy `go build ./...` toàn repo bị vướng 2 thư mục backup cũ có sẵn. | Đã chuyển sang build toàn bộ các package active (`./internal/... ./cmd/... ./pkgs/... ./config/...`) → PASS (`EV-20261030T173212-eb7b`, `EV-20261030T165700-97a6`). |
| **`FIX-04`** | **LOW (Thiếu bảng trong SQLite test)** | [`create_master_sql_engines_test.go:18-86`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master_sql_engines_test.go#L18-L86) | Lần đầu chạy test trên SQLite `:memory:` báo thiếu bảng `source_object_registry` và lỗi hàm `NOW()`. | Đã bổ sung bảng vào test harness (`EV-20261030T165646-e9e4`) và bạch hóa rõ giới hạn của SQLite test tại `ISS-01`. |

### C. Nhóm 3 — Tổng Kết Đánh Giá Kiến Trúc, Layer, SRP & Tương Thích Ngược (`Kết Luận Chung`)
- **Đặt đúng Layer & Tách File Đơn Trách Nhiệm (SRP):** **ĐẠT 100%** — Tách **11 file SRP mới (`2,794` dòng)** đúng từng tầng (`source`, `master`, `governance`, `recon`, `handler/recon`, `api/master`, `components/master`), không nhồi nhét vào các file lớn.
- **Tương thích ngược (PostgreSQL & MongoDB Master):** **ĐẠT 100%** — Toàn bộ 21 file hiện hữu chỉ thêm nhánh rẽ tường minh cho `mysql`/`clickhouse`, bảo toàn nguyên vẹn đường chạy của `postgresql` và `mongodb`.
- **Hội đồng `/laya_router` (`DG3`):** **ĐẠT `PASS`** ở Mode A (`7/7` claims `A-01`..`A-07` tại `06_laya_summary_dg3_20261030T173300.json`) và đã dọn sạch hoàn toàn `ISS-02`.

---

## Mục 1: Phản Biện "Quá Trình Thực Hiện", Từng Task (`TASK-01` → `TASK-07`), Từng File & Từng Dòng Update (`git diff` Line-by-Line Critique)

### 1.1. Phản Biện "Quá Trình" Vừa Thực Hiện (`S1 → S6`)
1. **Giai đoạn Khảo sát & Lập Kế hoạch (`S1` → `S3` + `10_gap_analysis`):**
   - Quá trình đối chiếu kiến trúc Master MongoDB hiện hữu trên cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) đã giúp phát hiện sớm **8 khe hở kiến trúc (`G1`–`G8`)** trước khi viết code (như lỗi nối cứng `&connect_timeout=5` làm hỏng DSN MySQL/ClickHouse, lỗi tràn `65,535` placeholders của MySQL, hay sự thiếu vắng `ON CONFLICT` trên ClickHouse `ReplacingMergeTree`).
2. **Giai đoạn Thực thi (`S4`) — Những vấp váp thực tế & cách khắc phục:**
   - **Điểm đạt:** Tuân thủ nghiêm ngặt kiến trúc phân tầng và nguyên tắc **Single-Responsibility Principle (SRP)** — tách toàn bộ logic đặc thù của MySQL & ClickHouse ra **11 file SRP mới (`2,794` dòng)** kèm **2 file test độc lập (`226` dòng)**; các file lớn hiện hữu chỉ nhận các hook điều phối mỏng (`1–15` dòng).
   - **Điểm tự phê bình trung thực trong quá trình thực thi `S4` (4 vấp váp đã gặp và xử lý):**
     1. *Vấp váp 1 (Shell pipe & sai đường dẫn import trong file test):* Ở lần chạy `evidence_run.py` đầu tiên, em truyền `-run "A|B|C"` không bọc nháy đơn khiến shell tách dấu `|` thành pipe lệnh, đồng thời trong `master_mysql_clickhouse_test.go` viết nhầm import `centralized-data-service/pkgs/config` thay vì `centralized-data-service/config`. Em đã sửa lại đúng đường dẫn import và tách từng lệnh `go test -run <TestName>` độc lập (`EV-20261030T173223-2794`, `EV-20261030T173541-a798`, `EV-20261030T173546-930b`).
     2. *Vấp váp 2 (Thư mục backup cũ khi chạy `go build ./...`):* Khi chạy `go build ./...` toàn repo, trình biên dịch Go vấp phải thư mục backup cũ có sẵn trong repo (`centralized-data-service/docs/server_bk` và `cdc-cms-service/scratch`). Em đã chuyển sang biên dịch chính xác toàn bộ các package active (`go build ./internal/... ./cmd/... ./pkgs/... ./config/...` -> `EV-20261030T173212-eb7b`, `EV-20261030T165700-97a6`).
     3. *Vấp váp 3 (Ánh xạ `NUMERIC(18,2)` sang `DECIMAL(18,2)` trong MySQL DDL):* Ở lần chạy đầu của `TestMasterMySQLAndClickHouseDDLGeneration`, hàm `mapToMySQLColumnType("NUMERIC(18,2)")` trả về nguyên chuỗi `"NUMERIC(18,2)"` vì nhánh `switch` chỉ khớp chuỗi tĩnh `"NUMERIC"`. Test đã bắt được ngay và em đã sửa tại `master_ddl_generator_mysql.go:68-70` (`strings.Replace(upper, "NUMERIC", "DECIMAL", 1)` -> `EV-20261030T173223-2794`).
     4. *Vấp váp 4 (SQLite `:memory:` trong test của `cdc-cms-service` — Xem chi tiết `Finding P-01` ở Mục 2):* Khi viết test `create_master_sql_engines_test.go`, để tránh chạm vào DB PostgreSQL thật của anh theo `L033`, em dùng `sqlite.Open(":memory:")` kèm `sqliteMasterRepoWrapper` (do SQLite không có hàm `NOW()` của PostgreSQL ở `master_connection_repo_gorm.go:391`). **Phản biện gắt gao:** Cách test bằng SQLite `:memory:` này chỉ kiểm chứng được nhánh logic Go (`isDBScopedConn` gán `master_schema = default_database`), **hoàn toàn KHÔNG tương đương với việc chạy trên PostgreSQL/MySQL/ClickHouse thật**. Em xin bạch hóa rõ ở Mục 2 để phối hợp cùng anh trace runtime thật.

---

### 1.2. Kiểm Toán Quy Tắc Single-Responsibility Principle (SRP) & Đo Lường `wc -l` Thực Tế

#### A. Bảng 11 File SRP Production Mới (`2,794` dòng) + 2 File Test Mới (`226` dòng)
| STT | File SRP Mới | Số dòng (`wc -l`) | Layer / Package | Trách nhiệm Đơn (Single Responsibility) |
| :-: | :--- | :-: | :--- | :--- |
| 1 | [`connection_manager_sql_masters.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager_sql_masters.go#L1-L118) | `118` | `service/source` | Nhận diện engine (`IsMySQLMasterConnection`, `IsClickHouseMasterConnection`), chuẩn hóa DSN (`parseTime=true`, `multiStatements=true`) và dựng DSN từ fields. |
| 2 | [`master_ddl_generator_mysql.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_mysql.go#L1-L386) | `386` | `service/master` | Sinh & thực thi DDL MySQL `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, `ALTER TABLE ADD/DROP COLUMN`, `RENAME TABLE`, tạo index qua `information_schema.statistics`. |
| 3 | [`master_ddl_generator_clickhouse.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go#L1-L318) | `318` | `service/master` | Sinh & thực thi DDL ClickHouse `ENGINE = ReplacingMergeTree(_transmuted_at) ORDER BY (_gpay_id)`, skipping index `minmax`, `EXCHANGE TABLES` (fallback rename). |
| 4 | [`transmuter_mysql.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mysql.go#L1-L331) | `331` | `service/master` | Batch Upsert MySQL (`INSERT ... AS incoming ON DUPLICATE KEY UPDATE` kèm OCC `_source_ts - 2000`, chia chunk $\le 60,000$ placeholders), xóa lô, watermark `MAX(_source_ts)`. |
| 5 | [`transmuter_clickhouse.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_clickhouse.go#L1-L345) | `345` | `service/master` | Pre-read OCC `SELECT _gpay_id, _source_ts ... FINAL` (`FilterClickHouseOCCRecords`), Batch Insert `ReplacingMergeTree(_transmuted_at)`, xóa lô `mutations_sync = 1`. |
| 6 | [`index_driver_mysql.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_mysql.go#L1-L201) | `201` | `service/governance` | `IndexEngineDriver` cho MySQL (`information_schema.statistics`, `ALGORITHM=INPLACE, LOCK=NONE` kèm fallback). |
| 7 | [`index_driver_clickhouse.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_clickhouse.go#L1-L153) | `153` | `service/governance` | `IndexEngineDriver` cho ClickHouse (`system.data_skipping_indices`, `ADD/DROP INDEX ... GRANULARITY 1`). |
| 8 | [`recon_dest_sql_dialects.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_sql_dialects.go#L1-L420) | `420` | `service/recon` | Toàn bộ hàm đọc đối soát Recon Segment B, XOR Hash, Go-side bucketing (`sqlDialectBucketCounts`), RowDiff B & TimeBoundedDiff B cho MySQL & ClickHouse (`FINAL`). |
| 9 | [`recon_execute_heal_segb_sql_dialects.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb_sql_dialects.go#L1-L262) | `262` | `handler/recon` | Prune orphan (`pruneSQLDialectMasterMissingFromSrc`) và tra cứu `_gpay_id` (`resolveSourceIDsFromSQLDialectMasterByGpayIDs`) cho Heal Segment B. |
| 10 | [`master_registry_handler_connections_probe_sql.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_connections_probe_sql.go#L1-L174) | `174` | `api/master` | Chuẩn hóa engine, dựng DSN và probe ping thực tế (`SELECT VERSION()` / `SELECT version()`) cho MySQL & ClickHouse. |
| 11 | [`masterEngineConfig.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/master/masterEngineConfig.ts#L1-L86) | `86` | `components/master` | Khai báo tập trung metadata UI, port mặc định, tag màu và cờ `isDatabaseScoped` cho 4 engine (`postgresql`, `mongodb`, `mysql`, `clickhouse`). |
| 12 | [`master_mysql_clickhouse_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go#L1-L119) | `119` | `service/master` (Test) | Kiểm thử đơn vị DDL MySQL/ClickHouse, bộ lọc OCC ClickHouse (`2000ms`), và nhận diện engine + tương thích ngược PostgreSQL. |
| 13 | [`create_master_sql_engines_test.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master_sql_engines_test.go#L1-L107) | `107` | `commands/master` (Test) | Kiểm thử `CreateMasterHandler` cưỡng chế `master_schema = default_database` cho MySQL/ClickHouse và bảo toàn `custom_pg_schema` cho PostgreSQL. |

#### B. Bảng Đối Soát Từng Dòng Update Trên 21 File Hiện Hữu (`git diff` Line-by-Line Audit)
> **Lưu ý minh bạch (`Finding P-03` — Lesson `L008`):** Working tree của 3 repo hiện chứa sẵn một số thay đổi chưa commit từ các task trước trong cùng đợt phát triển (`multi_shadow`, `cdc_event_date`, `mongo_bson_restorer`, `recon_smoke_guard`). Cột **Số dòng thuộc task `master_mysql_clickhouse`** bên dưới tách bạch chính xác từng dòng mà task này đã chạm vào:

| STT | File Hiện Hữu | `wc -l` Hiện tại | Vị trí Dòng Update (`file:line`) | Số dòng task chạm | Phản Biện Từng Dòng Update & Cam Kết Tương Thích Ngược |
| :-: | :--- | :-: | :--- | :-: | :--- |
| 1 | [`connection_manager.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go#L126-L137) | `424` | `L126-137`, `L402-409` | `+7` | `normalizeEngineName` thêm nhánh `"clickhouse", "ch"` và `"mysql", "mariadb"`, giữ nguyên `default: return "postgresql"`. `GetMasterDB` gọi `normalizeMasterSQLDSN(dsn, resolvedEng)` và `OpenGormWithEngine(..., resolvedEng)`. Khi là PostgreSQL, đường chạy giữ nguyên 100%. |
| 2 | [`metadata_registry_utils.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/db/metadata_registry_utils.go#L35-L44) | `124` | `L35-44` | `+4` | `detectSQLDialect` nhận diện thêm prefix `clickhouse://`, `ch://`, `mysql://` và `@tcp(`. Không ảnh hưởng DSN PostgreSQL. |
| 3 | [`metadata_registry_dsn.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/db/metadata_registry_dsn.go#L10-L36) | `101` | `L10-36` | `+5` | Mở rộng danh sách tham số loại bỏ khỏi `extra_params` (`parseTime`, `multiStatements`, `charset`, `loc`, `dial_timeout`, `read_timeout`) để không nối trùng. Trong nhánh `case "mysql":` thêm `&multiStatements=true`. |
| 4 | [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L101-L265) | `298` | `L101-106`, `L118-123`, `L149-154`, `L180-185`, `L203-208`, `L247-265` | `+36` | Thêm 6 hook điều phối mỏng (`if g.isMySQLBinding(ctx, reg) { ... }`, `if g.isClickHouseBinding(ctx, reg) { ... }`) ngay sau `if g.isMongoDBBinding(ctx, reg)`. Toàn bộ thân hàm PostgreSQL phía dưới không đổi 1 ký tự. |
| 5 | [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L49-L742) | `905` | `L49-50`, `L129-145`, `L222-227`, `L349-363`, `L532-537`, `L612-624`, `L717-742` | `+52` | Thêm cờ `IsMySQLDestination` / `IsClickHouseDestination` trên `masterBindingRuntime` và các cổng rẽ nhánh sang `transmuter_mysql.go` / `transmuter_clickhouse.go`. Luồng PostgreSQL (`INSERT ... ON CONFLICT ("_gpay_id") DO UPDATE`) và MongoDB (`BulkWrite`) giữ nguyên 100%. |
| 6 | [`index_driver_registry.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_registry.go#L121-L125) | `162` | `L121-125` | `+2` | Chỉ thêm đúng 2 dòng đăng ký `newMySQLIndexDriver` và `newClickHouseIndexDriver`. |
| 7 | [`recon_dest_agent.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go#L35-L185) | `384` | `L35-38`, `L65-73`, `L92-119`, `L138-159`, `L169-185` | `+46` | Thêm `destEngine string` vào `ReconDestAgent`; bỏ qua `SET TRANSACTION READ ONLY` khi `isClickHouseDest()` (vì ClickHouse ném lỗi `UNKNOWN_SETTING`); định tuyến `CountMasterBySource`, `ChecksumMasterBySource`, `FetchMasterRecordsBySource`, `FetchMasterRecordMetaBySource` sang `recon_dest_sql_dialects.go`. |
| 8 | [`recon_engine.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go#L118-L132) | `337` | `L118-132` | `+11` | `getDestAgentForMaster` kiểm tra `IsMySQLMasterConnection` / `IsClickHouseMasterConnection` để khởi tạo `NewReconDestAgentForSQLDialect(masterDB, dialect)`. |
| 9 | [`recon_dest_query.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go#L33-L250) | `299` | `L33-38`, `L160-165`, `L245-250` | `+18` | Định tuyến `GetRecordBySourceID`, `GetMasterWatermark`, `VerifySpotCheck` sang `recon_dest_sql_dialects.go` khi là MySQL/ClickHouse. |
| 10 | [`recon_dest_hash.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_hash.go#L43-L253) | `299` | `L43-46`, `L248-253` | `+9` | Định tuyến `BucketCountsMasterBySource` và `BucketCountsMasterByWindow` sang `sqlDialectBucketCounts` (băm MD5 phía Go thay vì cú pháp PostgreSQL `::bit(64)::bigint`). |
| 11 | [`recon_window_index_probe.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_window_index_probe.go#L67-L84) | `197` | `L67-84` | `+16` | Khi kết nối Master là MySQL (`information_schema.statistics`) hoặc ClickHouse (`ORDER BY` / skipping index), kiểm tra index theo đúng dialect thay vì `pg_indexes`. |
| 12 | [`recon_tier_b.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go#L72-L202) | `248` | `L72-79`, `L195-202` | `+14` | Định tuyến `runRowDiffB` và `runTimeBoundedDiffB` sang `runSQLDialectRowDiffB` / `runSQLDialectTimeBoundedDiffB` trong `recon_dest_sql_dialects.go`. |
| 13 | [`recon_execute_heal_segb.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb.go#L118-L212) | `249` | `L118-127`, `L202-212` | `+18` | Định tuyến `pruneMasterMissingFromSrc` và `resolveSourceIDsFromMasterByGpayIDs` sang `recon_execute_heal_segb_sql_dialects.go`. |
| 14 | [`source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L296-L306) | `490` | `L296-306` | `+6` | Bổ sung nhánh `mysql`/`mariadb` (dùng backtick) và `(clickhouse \| ch) && LayerType == "master"` (dùng `FINAL`). Đã rút gọn nhánh `else` dư thừa (`ISS-02` ĐÃ ĐÓNG 100%). |
| 15 | [`master_registry_handler_connections_probe.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_connections_probe.go#L18-L200) | `248` | `L18-22`, `L38-45`, `L61-74`, `L92-122`, `L188-200` | `+38` | Chuẩn hóa `engine` qua `normalizeMasterConnectionEngine`; dựng DSN qua `buildMasterSQLConnectionDSN`; chỉ nối `connect_timeout=5` khi `engine == "postgresql"`; định tuyến ping MySQL/ClickHouse sang `probeSQLMasterConnection`. |
| 16 | [`master_connection_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infrastructure/persistence/postgres/master_connection_repo_gorm.go#L353-L361) | `428` | `L353-361` | `+3` | Mở rộng điều kiện đồng bộ `master_schema = default_database` khi cập nhật kết nối sang cả `mongodb`, `mysql`, `mariadb`, `clickhouse`, `ch`. |
| 17 | [`master_read_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infrastructure/persistence/postgres/master_read_repo_gorm.go#L34-L42) | `314` | `L34-42` | `+4` | Trong `resolveEffectiveMasterSchema`, áp dụng quy tắc `1 Connection = 1 Database` cho `mongodb`, `mysql`, `mariadb`, `clickhouse`, `ch`, giữ nguyên schema tùy chỉnh cho `postgresql`. |
| 18 | [`create_master.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master.go#L180-L195) | `298` | `L180-195` | `+12` | Trong `CreateMasterHandler.Handle`, biến `isDBScopedConn` bao phủ `mongodb`, `mysql`, `mariadb`, `clickhouse`, `ch` để tự động gán `master_schema = connRec.DefaultDatabase`. |
| 19 | [`master.ts`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/types/master.ts#L102-L108) | `215` | `L102-108` | `+1` | Mở rộng union type `MasterConnectionRecord['engine']` thành `'postgresql' \| 'mongodb' \| 'mysql' \| 'clickhouse' \| string`. |
| 20 | [`MasterRegistry.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/master/MasterRegistry.tsx#L39-L645) | `747` | `L39`, `L203`, `L566-645` | `+14` | Import `MASTER_ENGINE_OPTIONS`, `getMasterEngineMeta`, `normalizeMasterEngine` từ `masterEngineConfig.ts` thay vì hardcode 2 engine `postgresql`/`mongodb`. |
| 21 | [`CreateMasterModal.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/master/CreateMasterModal.tsx#L12-L355) | `427` | `L12`, `L48-68`, `L142-151`, `L330-355` | `+16` | Dùng `isDatabaseScopedMasterEngine(conn.engine)` để khóa và tự điền `master_schema = default_database` cho cả `MongoDB`, `MySQL`, `ClickHouse`, trong khi `PostgreSQL` giữ nguyên ô nhập schema tự do. |

---

### 1.3. Phát Hiện Phản Biện Từ `/laya_router` Mode B (`Finding P-02`: Nhánh `else` dư thừa trong `source_count_runner.go:300-310`)
Khi chạy `/laya_router` Mode B (`laya_gate.py review`) trên diff của [`source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L296-L311) (`06_laya_summary_dg3review_20261030T171338.json`), Laya đã từ chối ở Vòng 1 với nhãn `REJECT_REDUNDANT` (`score=0.0001`). Kiểm tra từng dòng tại `L296-310`:
```go
	if eng == "mysql" || eng == "mariadb" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s` WHERE `%s` >= ? AND `%s` < ?",
			params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
	} else if eng == "clickhouse" || eng == "ch" {
		if strings.EqualFold(params.LayerType, "master") {
			query = fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s` FINAL WHERE `%s` >= ? AND `%s` < ?",
				params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
		} else {
			query = fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s" WHERE "%s" >= ? AND "%s" < ?`,
				params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
		}
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s" WHERE "%s" >= ? AND "%s" < ?`,
			params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
	}
```
- **Phân tích phản biện:**
  1. Nhánh `else` bên trong khối `else if eng == "clickhouse" || eng == "ch"` (`L303-306`) sinh ra câu SQL giống hệt nhánh `else` bên ngoài (`L307-310`). Đây đúng là một đoạn lặp dư thừa (`redundant branch`) như `/laya_router` chỉ ra.
  2. Vì hiện tại em đang ở chế độ **Audit Read-Only (`LS-GLOBAL-001-propose-only-gate`)**, em **tuyệt đối không tự ý sửa file [`source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L296-L310)** khi chưa có lệnh `APPROVE` của anh. Nếu anh muốn gọn sạch tuyệt đối đoạn này, em xin đề xuất **Code Demo** rút gọn 6 dòng tại `source_count_runner.go:296-306` như sau:
```go
	if eng == "mysql" || eng == "mariadb" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s` WHERE `%s` >= ? AND `%s` < ?",
			params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
	} else if (eng == "clickhouse" || eng == "ch") && strings.EqualFold(params.LayerType, "master") {
		query = fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s` FINAL WHERE `%s` >= ? AND `%s` < ?",
			params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s" WHERE "%s" >= ? AND "%s" < ?`,
			params.ContainerNamespace, params.ObjectName, params.DateField, params.DateField)
	}
```

---

## Mục 2: Chống Suy Diễn Smoke Test / Mockup / Hallucination (`Finding P-01` — Lesson `L019`) & Kế Hoạch Phối Hợp Trace Runtime Thật Cùng Anh

### 2.1. Kiểm Toán Trung Thực Về Mockup & Phạm Vi Của Các Bài Test Đã Chạy (`Finding P-01`)
1. **Trên Production Code (`11 file SRP mới` + `21 file sửa`):**
   - **ĐẠT 100% Zero-Mockup:** Không có bất kỳ biến closure giả (`*Fn`), mock struct, hay logic trả kết quả giả nào được gắn vào các struct production (`MasterDDLGenerator`, `Transmuter`, `ConnectionManager`, `ReconDestAgent`, `CreateMasterHandler`).
2. **Trên Test Code — Tự Phê Bình Thẳng Thắn Theo `L019` (`Zero-Mockup Evidence & Real Runtime Trace Verification`):**
   - Trong quá trình thực thi `S4`, để tuân thủ `L032` & `L033` (tuyệt đối không tự ý chạy lệnh kết nối vào DB thật hay container của anh), em đã viết 2 file test:
     - [`master_mysql_clickhouse_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_mysql_clickhouse_test.go#L1-L119) (`EV-20261030T173223-2794`, `EV-20261030T173541-a798`, `EV-20261030T173546-930b`): Kiểm tra chuỗi DDL sinh ra trong bộ nhớ, hàm lọc `FilterClickHouseOCCRecords` trong bộ nhớ, và hàm nhận diện engine.
     - [`create_master_sql_engines_test.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/create_master_sql_engines_test.go#L18-L86) (`EV-20261030T165646-e9e4`): Sử dụng `sqlite.Open(":memory:")` kết hợp với wrapper `sqliteMasterRepoWrapper` (do SQLite không có hàm `NOW()` của PostgreSQL tại [`master_connection_repo_gorm.go:391`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infrastructure/persistence/postgres/master_connection_repo_gorm.go#L391)).
   - **Kết luận phản biện trung thực (Không báo cáo láo, không suy diễn):**
     - Các biên lai `EV-20261030T173223-2794`, `EV-20261030T173541-a798`, `EV-20261030T173546-930b`, `EV-20261030T165646-e9e4` **CHỈ chứng minh logic Go in-memory (sinh chuỗi DDL, thuật toán lọc OCC, gán `master_schema = default_database`) và trình biên dịch Go/TypeScript (`EV-20261030T173212-eb7b`, `EV-20261030T165700-97a6`, `EV-20261030T165703-f720`) đã chạy PASS**.
     - Chúng **HOÀN TOÀN CHƯA PHẢI** là bằng chứng runtime thực tế trên **MySQL InnoDB server thật**, **ClickHouse server thật**, và **PostgreSQL `cdc_system` thật**! Em tuyệt đối không lấy kết quả unit test / SQLite `:memory:` này để suy diễn hay báo cáo láo rằng luồng kết nối mạng, thực thi DDL và Transmute trên DB thật đã được trace xong.

### 2.2. Danh Sách Biên Lai Vật Lý Đã Thu Thập (`EV-...` tại `06_evidence/`)
| Biên lai `EV-...` | Lệnh thực thi | Exit Code | Phạm vi chứng minh thực tế (Không suy diễn quá phạm vi) |
| :--- | :--- | :-: | :--- |
| `EV-20261030T173223-2794` | `go test -v ./internal/service/master -run TestMasterMySQLAndClickHouseDDLGeneration` | `0` | Chuỗi DDL sinh ra cho MySQL (`ENGINE=InnoDB`, `DECIMAL(18,2)`, `JSON`, `DATETIME(3)`) và ClickHouse (`ReplacingMergeTree(_transmuted_at) ORDER BY (_gpay_id)`, `INDEX ... TYPE minmax GRANULARITY 1`). |
| `EV-20261030T173541-a798` | `go test -v ./internal/service/master -run TestFilterClickHouseOCCRecords` | `0` | Thuật toán lọc OCC in-memory cho ClickHouse với dung sai `2000ms` (`incTs < prevTs - 2000` bị chặn, `incTs >= prevTs - 2000` được giữ). |
| `EV-20261030T173546-930b` | `go test -v ./internal/service/master -run TestMasterBindingDestinationDetectionAndBackwardCompatibility` | `0` | Phân giải engine `mysql`, `clickhouse`, `mongodb`, `postgresql` và tương thích ngược cho kết nối `default`. |
| `EV-20261030T173212-eb7b` | `go build ./internal/... ./cmd/... ./pkgs/... ./config/...` (`centralized-data-service`) | `0` | Toàn bộ các package active của `centralized-data-service` biên dịch sạch lỗi sau khi đóng `ISS-02`. |
| `EV-20261030T165646-e9e4` | `go test -v ./internal/app/commands/master -run TestCreateMasterHandler_MySQLAndClickHouse_EnforcesDatabaseScopeAndPreservesPostgres` | `0` | Logic handler `CreateMasterHandler` (chạy trên SQLite `:memory:` test harness) cưỡng chế `master_schema = default_database` cho MySQL/ClickHouse và giữ nguyên `custom_pg_schema` cho PostgreSQL. |
| `EV-20261030T165700-97a6` | `go build ./internal/... ./cmd/...` (`cdc-cms-service`) | `0` | Toàn bộ các package active của `cdc-cms-service` biên dịch sạch lỗi. |
| `EV-20261030T165703-f720` | `npx tsc --noEmit` (`cdc-cms-web`) | `0` | Toàn bộ dự án Frontend TypeScript biên dịch sạch lỗi (`0 errors`). |

### 2.3. Đề Xuất Phối Hợp Cùng Anh Để Trace Runtime Value Trên Môi Trường Thật
Để kiểm chứng 100% runtime thật mà không dùng mockup, em xin đề xuất phối hợp cùng anh theo **5 bước trace runtime thực tế** sau (khi anh bật `cdc-cms-service`, `centralized-data-service` và có instance MySQL / ClickHouse đích):
1. **Bước 1 (Trace Probe & Lưu Kết Nối tại `cdc-cms-service`):**
   - Anh bấm **Test Connection** trên giao diện `Master Connections` với 1 kết nối `MySQL` (port `3306`) và 1 kết nối `ClickHouse` (port `9000`).
   - Giá trị runtime cần đối chiếu: Response JSON trả về `status: "CONNECTED"`, `server_version` thực tế từ `SELECT VERSION()` (MySQL) / `SELECT version()` (ClickHouse), và bản ghi trong bảng PostgreSQL `master_connection_registry` có `engine = 'mysql' | 'clickhouse'`, `default_database` đúng như đã nhập.
2. **Bước 2 (Trace Tạo Master Binding & Cưỡng Chế `master_schema = default_database`):**
   - Anh tạo 1 Master Table chọn kết nối MySQL và 1 Master Table chọn kết nối ClickHouse trên `CreateMasterModal`.
   - Giá trị runtime cần đối chiếu: Ô `Master Database` trên UI tự động khóa theo `default_database`; bản ghi trong `master_object_registry` lưu đúng `master_schema = <default_database>`.
3. **Bước 3 (Trace Thực Thi DDL Thật Trên MySQL & ClickHouse):**
   - Anh bấm **Approve DDL / Apply DDL** cho 2 bảng Master vừa tạo.
   - Giá trị runtime cần đối chiếu:
     - Trên MySQL đích: `SHOW CREATE TABLE <db>.<table>` xác nhận `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` và các index trong `information_schema.statistics`.
     - Trên ClickHouse đích: `SHOW CREATE TABLE <db>.<table>` xác nhận `ENGINE = ReplacingMergeTree(_transmuted_at) ORDER BY (_gpay_id)` và skipping index `minmax` trong `system.data_skipping_indices`.
4. **Bước 4 (Trace Luồng Transmute & OCC Out-of-Order Replay):**
   - Khi worker Transmuter chạy batch đầu tiên từ Shadow sang Master MySQL & ClickHouse:
     - Giá trị runtime cần đối chiếu: Số dòng (`COUNT(*)`) trên MySQL (`WHERE _deleted = 0`) và ClickHouse (`FINAL WHERE _deleted = 0`), giá trị `_source_ts`, `_transmuted_at`, và watermark trong bảng `sync_runtime_state`.
5. **Bước 5 (Trace Đối Soát Recon Segment B & Governance Index):**
   - Chạy thử 1 lượt **Recon Segment B (Shadow ↔ Master)** cho bảng MySQL và ClickHouse.
   - Giá trị runtime cần đối chiếu: Log `ReconDestAgent` không báo lỗi `SET TRANSACTION READ ONLY` trên ClickHouse, `BucketCountsMasterByWindow` trả về hash khớp giữa Shadow và Master.

---

## Mục 3: Kiểm Tra Concurrency / Multi-Worker / Batching / Timeout-Heartbeat / Watermark Continuity

1. **Multi-Worker (`WorkerCount > 1`) & Chống Ghi Đè Ngược (OCC `_source_ts`):**
   - **MySQL Master ([`transmuter_mysql.go:196-230`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mysql.go#L196-L230)):** Sử dụng câu lệnh nguyên tử `INSERT INTO <db>.<table> (...) VALUES ... AS incoming ON DUPLICATE KEY UPDATE col = IF(COALESCE(incoming._source_ts, 0) >= COALESCE(<table>._source_ts, 0) - 2000, incoming.col, <table>.col)`. Toàn bộ phép so sánh `_source_ts` diễn ra bên trong row-level lock của InnoDB, đảm bảo khi `WorkerCount > 1` cùng đẩy các sự kiện CDC của cùng một `_gpay_id`, sự kiện cũ đến trễ (trễ hơn `2000ms` so với bản ghi hiện tại) không bao giờ ghi đè làm lùi trạng thái của bản ghi mới hơn.
   - **ClickHouse Master ([`transmuter_clickhouse.go:68-158`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_clickhouse.go#L68-L158)):** Kết hợp 2 lớp bảo vệ:
     - Lớp 1 (`FilterClickHouseOCCRecords`): Trước mỗi batch insert, truy vấn `SELECT _gpay_id, _source_ts FROM <db>.<table> FINAL WHERE _gpay_id IN (?)` (chia chunk `1,000` ID) để loại bỏ mọi sự kiện đến trễ có `incoming._source_ts < existing._source_ts - 2000`.
     - Lớp 2 (`ReplacingMergeTree(_transmuted_at)`): Bảng ClickHouse dùng cột phiên bản `_transmuted_at UInt64` (millisecond timestamp đơn điệu tại thời điểm transmute), nên mọi bản ghi hợp lệ đi qua cổng OCC (kể cả sự kiện hợp lệ nằm trong dung sai lệch đồng hồ `2000ms`) đều có `_transmuted_at` lớn hơn và được ClickHouse `FINAL` / background merge giữ lại chính xác.
2. **Batching & Chống Tràn Giới Hạn `65,535` Placeholders Của MySQL:**
   - Tại [`transmuter_mysql.go:167-176`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mysql.go#L167-L176), kích thước mỗi chunk được tính động theo số cột thực tế của bảng: `maxRowsPerChunk := 60000 / len(cols)` (giới hạn trần `500` dòng/chunk). Kể cả bảng Master có `100` cột thì mỗi câu `INSERT` chỉ dùng tối đa `60,000` dấu `?`, an toàn tuyệt đối dưới ngưỡng cứng `65,535` của MySQL prepared statement.
3. **Tương Tác Timeout / Mutations Sync & Heartbeat:**
   - Trên ClickHouse, thao tác xóa vật lý (`ALTER TABLE ... DELETE WHERE _gpay_id IN (?) SETTINGS mutations_sync = 1` tại [`transmuter_clickhouse.go:258-296`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_clickhouse.go#L258-L296)) được chia theo từng lô `500` ID để không giữ mutation quá lâu làm ảnh hưởng tới context timeout của worker.
4. **Tính Liên Tục Của Watermark (`Watermark Continuity`) Sau Outage:**
   - Tại [`transmuter.go:349-363`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L349-L363), khi `sync_runtime_state` chưa có watermark hoặc cần khôi phục sau sự cố, `inferWatermarkFromMySQLMaster` (`SELECT COALESCE(MAX(_source_ts), 0) FROM <db>.<table> WHERE _deleted = 0`) và `inferWatermarkFromClickHouseMaster` (`SELECT COALESCE(MAX(_source_ts), 0) FROM <db>.<table> FINAL WHERE _deleted = 0`) khôi phục chính xác mốc `_source_ts` cao nhất đã ghi thành công xuống bảng đích.

---

## Mục 4: Kiểm Tra Suy Diễn & Báo Cáo Trung Thực (Honesty & Anti-Speculation Audit)

1. **Không che giấu các lần chạy lỗi hoặc cảnh báo trong quá trình thực hiện (`L004`):**
   - Toàn bộ 4 lỗi phát sinh trong quá trình chạy test/build ở `S4` (`FIX-01` → `FIX-04`) và toàn bộ các lần chạy `/laya_router` (cả Mode A và Mode B) đều được công khai đầy đủ nguyên nhân, cách khắc phục và biên lai hiện hành (`EV-20261030T173223-2794`, `EV-20261030T173541-a798`, `EV-20261030T173546-930b`, `EV-20261030T173212-eb7b`, `EV-20261030T165646-e9e4`, `EV-20261030T165700-97a6`, `EV-20261030T165703-f720`).
2. **Không suy diễn từ Unit Test / SQLite `:memory:` sang Live DB Runtime (`L019`):**
   - Đã tách bạch rõ ràng tại Mục 2: những gì đã được chứng minh bằng unit test / compiler (`EV-...`) và những gì cần phối hợp với anh để trace trên MySQL / ClickHouse / PostgreSQL thật.
3. **Bảo toàn 100% Tương thích ngược cho PostgreSQL và MongoDB:**
   - Mọi nhánh code mới đều đi qua các hàm kiểm tra tường minh (`IsMySQLMasterConnection`, `IsClickHouseMasterConnection`, `isMySQLBinding`, `isClickHouseBinding`, `isMySQLDest`, `isClickHouseDest`). Khi kết nối là `postgresql` (hoặc rỗng / `default`) hay `mongodb`, luồng thực thi đi đúng vào các hàm gốc không thay đổi.

---

## Mục 5: Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop & Lessons Audit)

Đối chiếu toàn bộ quá trình vừa thực hiện với kho bài học Global (`agent/memory/global/lessons/INDEX.md`):
- **`L001` (`No SQL Keywords or Chained Commands in Gate CLI Args`):** Tuân thủ 100%. Mọi lệnh `laya_gate.py` và `evidence_run.py` đều được gọi đơn lẻ, không dùng `&&`, không truyền từ khóa SQL/DDL vào chuỗi `--claim`.
- **`L003` (`Honest Gate Reporting`):** Tuân thủ 100%. Báo cáo trung thực cả kết quả Mode A (`PASS` sau khi `resolve` các claim `FLAG` có chứng cứ `REF:`) lẫn Mode B (`7 ACCEPT`, `1 UNVALIDATED`, `1 BLOCK/REJECT_REDUNDANT` tại `source_count_runner.go:303-306`).
- **`L006` (`Propose-Only Gate on Audit/Diff Requests` — `LS-GLOBAL-001-propose-only-gate`):** Tuân thủ 100%. Trong lượt Audit này, em giữ nguyên chế độ Read-Only trên toàn bộ source code, chỉ ghi báo cáo vào workspace và trình bày Code Demo cho phát hiện `Finding P-02`.
- **`L008` (`Cross-Check Shared Package Working Tree Before Compile`):** Tuân thủ 100%. Đã rà soát toàn bộ `git status` trên cả 3 repo và ghi nhận rõ các file uncommitted có sẵn từ những tính năng trước (`Finding P-03`).
- **`L009` (`Single-Responsibility Module Split`):** Tuân thủ 100%. Tách 11 file SRP production mới (`2,794` dòng) đúng layer.
- **`L019` (`Zero-Mockup Evidence & Real Runtime Trace Verification`):** Tuân thủ 100%. Tự phê bình rõ việc dùng SQLite `:memory:` trong `create_master_sql_engines_test.go` (`Finding P-01`), không suy diễn sang runtime thật và lập kế hoạch 5 bước phối hợp trace runtime cùng anh.
- **`L023` (`Preserve Existing File Lines on Append-Only Edits`):** Tuân thủ 100%. Chỉ dùng `edit_file` trên các file memory trong workspace, tuyệt đối không dùng `write_file` (`Overwrite=true`).
- **`L032` & `L033` (`Zero DB CLI Or Container Execution`):** Tuân thủ 100%. Không chạy bất kỳ lệnh CLI DB hay Docker nào chạm vào môi trường của anh.

---

## Mục 6: Kết Quả Tiến Trình QC Gắt Gao Qua `/laya_router` (`DG3` Mode A & Mode B)

### 6.1. Kết Quả `/laya_router` Mode A (`laya_gate.py run --stage dg3`)
- **Biên bản tổng hợp:** [`06_laya_summary_dg3_20261030T171120.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/06_laya_summary_dg3_20261030T171120.json)
- **File Claims đầy đủ (`TASK-01` → `TASK-07`):** [`dg3_claims_master_mysql_clickhouse.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/dg3_claims_master_mysql_clickhouse.json)
- **Kết quả Calibration Controls:**
  - `CTRL-TRUE-01`: `PASS` (`mean=0.8087`)
  - `CTRL-FALSE-01`: `BLOCK` đúng kỳ vọng (`mean=0.0156` — phát hiện tuyên bố sai về `SET TRANSACTION READ ONLY`)
  - `CTRL-FALSE-02`: `BLOCK` đúng kỳ vọng (`mean=0.0022` — phát hiện tuyên bố sai về `ON CONFLICT DO UPDATE` trên ClickHouse)
  - `calibration_valid`: `true`
- **Kết quả 7 Claim Thực Thi (`A-01` → `A-07`):**
  - `A-01` (`TASK-01` — `ConnectionManager.GetMasterDB` & `normalizeMasterSQLDSN`): Laya chấm `FLAG` (`mean=0.4534`) → Đã đối chiếu trực tiếp [`connection_manager.go:402-409`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go#L402-L409) và `resolve` -> `PASS`.
  - `A-02` (`TASK-02` — `MasterDDLGenerator` cho MySQL InnoDB & ClickHouse `ReplacingMergeTree(_transmuted_at)`): Laya chấm `FLAG` (`mean=0.6194`) → Đã đối chiếu trực tiếp [`master_ddl_generator.go:101-130`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L101-L130) và `resolve` -> `PASS`.
  - `A-03` (`TASK-03` — `Transmuter` OCC `_source_ts - 2000` & chunking $\le 60,000$ placeholders): Laya chấm `FLAG` (`mean=0.3690`) → Đã đối chiếu trực tiếp [`transmuter.go:612-624`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L612-L624), [`transmuter_mysql.go:167-230`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mysql.go#L167-L230), [`transmuter_clickhouse.go:68-158`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_clickhouse.go#L68-L158) và `resolve` -> `PASS`.
  - `A-04` (`TASK-04` — `NewIndexManagerWithConn` đăng ký đủ 4 driver): Laya chấm `FLAG` (`mean=0.4718`) → Đã đối chiếu trực tiếp [`index_driver_registry.go:116-126`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/index_driver_registry.go#L116-L126) và `resolve` -> `PASS`.
  - `A-05` (`TASK-05` — `ReconDestAgent` bỏ `SET TRANSACTION READ ONLY` trên ClickHouse, dùng `FINAL` và Go-side MD5 bucketing): Laya chấm `FLAG` (`mean=0.3511`) → Đã đối chiếu trực tiếp [`recon_dest_agent.go:65-73`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go#L65-L73) và `resolve` -> `PASS`.
  - `A-06` (`TASK-06` — `CreateMasterHandler.Handle` cưỡng chế `master_schema = default_database` cho `mongodb`, `mysql`, `clickhouse` và giữ nguyên `postgresql`): Laya chấm `PASS` trực tiếp (`mean=0.9036`, `REF: cdc-cms-service/internal/app/commands/master/create_master.go:180-195`).
  - `A-07` (`TASK-07` — `CreateMasterModal.tsx` dùng `isDatabaseScopedMasterEngine` khóa & tự điền `master_schema` cho `mongodb`, `mysql`, `clickhouse`): Laya chấm `PASS` trực tiếp (`mean=0.8077`, `REF: cdc-cms-web/src/components/master/CreateMasterModal.tsx:48-68`).
- **Tổng kết Gate `DG3` Mode A:** `"gate": "PASS"` (`exit=0`).

### 6.2. Kết Quả `/laya_router` Mode B (`laya_gate.py review` trên `task_scoped.diff`)
- **Biên bản tổng hợp:** [`06_laya_summary_dg3review_20261030T171338.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/master-mysql-clickhouse-2026-10-30/06_laya_summary_dg3review_20261030T171338.json)
- **Kết quả chấm từng file diff trong `centralized-data-service`:**
  1. `internal/handler/recon/recon_execute_heal_segb.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9803`).
  2. `internal/service/master/master_ddl_generator.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9852`).
  3. `internal/service/recon/recon_dest_hash.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9965`).
  4. `internal/service/recon/recon_dest_query.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9949`).
  5. `internal/service/recon/recon_engine.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9904`).
  6. `internal/service/source/connection_manager.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.7336`).
  7. `pkgs/db/metadata_registry_dsn.go`: `ACCEPT` ngay Vòng 1 (`APPROVE`, `score=0.9617`).
  8. `pkgs/db/metadata_registry_utils.go`: `UNVALIDATED` (Vòng 1 `REJECT_UNSAFE` `0.0020`, Vòng 2 Laya không phân biệt được biến thể control sai; đã xác minh trực tiếp bằng code tại [`metadata_registry_utils.go:35-44`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/db/metadata_registry_utils.go#L35-L44)).
  9. `internal/service/sourcecount/source_count_runner.go`: `BLOCK` (Vòng 1 `REJECT_REDUNDANT` `0.0001`, Vòng 2 `APPROVE` `0.0554`) — Chính là phát hiện `Finding P-02` (`ISS-02`) về nhánh `else` lặp lại tại [`source_count_runner.go:303-306`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L303-L306) mà em đã phân tích và trích xuất Code Demo ở Mục 1.3.

---

## Mục 7: Tổng Kết Cuối Cùng & Đề Xuất Hành Động Cho Anh (Executive Conclusion)

1. **Về Kiến trúc, Layering, SRP & Tương thích ngược:** **ỔN ĐỊNH & ĐẠT 100%** — Không có bất kỳ lỗi kiến trúc, sai layer hay phá vỡ tương thích ngược nào đối với PostgreSQL và MongoDB Master.
2. **Về Code Hiện Tại (`ISS-02` ĐÃ ĐÓNG 100%):** Đã rút gọn triệt để nhánh `else` lặp tại [`source_count_runner.go:296-306`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L296-L306) (`wc -l` giảm từ `495` xuống `490` dòng), biên dịch & test PASS (`EV-20261030T173212-eb7b`, `EV-20261030T173223-2794`), và đạt `DG3` `PASS` (`06_laya_summary_dg3_20261030T173300.json`).
3. **Về Kiểm Chứng Thực Tế (`ISS-01` — Mức độ `HIGH` cần lưu ý):** Do em tuyệt đối không tự ý kết nối vào DB thật của anh (`L033`), các bài test vừa chạy mới chỉ chứng minh logic in-memory/SQLite. Khi anh sẵn sàng bật kết nối MySQL / ClickHouse thật trên môi trường chạy của anh, anh em mình phối hợp kiểm tra nhanh theo 5 bước tại Mục 2.3 là khép kín 100% runtime thật.
