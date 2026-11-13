---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T11:10:00+07:00
---

# Báo Cáo Audit Phản Biện Toàn Diện Quá Trình & Code Đã Thực Hiện — Nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

---

## 0. Bảng Tổng Hợp Phản Biện "Quá Trình" Vừa Thực Hiện, Các Điểm Lệch Còn Sót (`P1`, `P2`, `F-NEW-1..F-NEW-3`) & Các Hạng Mục Đã Khắc Phục (`F1`–`F6`)

### A. Phản Biện Quá Trình Vừa Thực Hiện & Các Phát Hiện Mới Cần Tinh Chỉnh Dứt Điểm (`P1`, `P2`, `F-NEW-1`–`F-NEW-3`)

| Mã ID | Mức độ | Phân loại Phản biện | Tệp và Dòng (`file:line` hiện tại) | Phản biện Gốc rễ Không Khoan Nhượng (Zero-Spin Adversarial Critique) | Hướng xử lý / Phối hợp (`1-to-1 Action`) | Trạng thái |
| :--- | :---: | :--- | :--- | :--- | :--- | :---: |
| **`P1`** | **HIGH** | **Sai sót Quy trình Thu thập Bằng chứng (`L011` — Mislabeled Evidence Kind)** | `06_evidence/EV-20261103T130601-f0ff.json`<br>`06_evidence/EV-20261103T130617-356e.json` | Ở lượt thực thi `TASK-9` vừa rồi, khi chạy 2 lệnh kiểm tra tĩnh `npx tsc -p tsconfig.build.json --noEmit` và `npx eslint ...`, em đã truyền cờ `--kind runtime` vào `evidence_run.py`. Theo đúng bài học `L011`, kiểm tra biên dịch/lint tĩnh **không phải** là bằng chứng chạy runtime của service (`moleculer-runner`) hay DB thật. | Đã tự giác chạy lại 2 lệnh này với nhãn chuẩn `--kind unit` để thu biên lai trung thực **`EV-20261103T133200-bb18`** (`tsc`, `exit=0`) và **`EV-20261103T133203-462f`** (`eslint`, `exit=0`), đồng thời ghi rõ sự thật trong Mục 2 & Mục 4. | **SELF-CORRECTED (`EV-20261103T133200-bb18`, `EV-20261103T133203-462f`)** |
| **`P2`** | **CRITICAL** | **Chặn đứng Suy diễn từ Unit Test Mockup & Báo cáo Trung thực nhu cầu Trace Runtime Thật (`Rule #9.8b`, `Rule #9.8c`, `L008`, `L027`)** | `test/unit/trans-his-pg/*.test.ts`<br>`test/integration/trans-his-pg.integration.test.ts:18-26`<br>`test/tools/trans-his-parity.ts:1-443` | **Kiểm tra chống báo cáo láo & chống mockup:**<br>1. Toàn bộ `111/111` bài test PASS trong biên lai `EV-20261103T130714-efe5` đều là **Unit Test chạy in-memory bằng Mock (`vi.fn()`, `vi.spyOn`)**: `trans-his-pg.connection.test.ts:4` mock `pg.Pool` và spy `DbConnectorFactory.getInstance`; `trans-his-pg.model.test.ts:38-43` truyền fake pool `{ query: vi.fn() }`; `trans-his-pg.handlers.test.ts` mock `TransHisPgModel`.<br>2. Toàn bộ `13/13` bài Integration Test trong `test/integration/trans-his-pg.integration.test.ts` (`EV-20261103T130726-e8a5`) đang bị **`skipped`** vì trên máy local chưa có biến môi trường `TRANS_HIS_PG_IT_URL` và chưa có file `.run.local.env` chứa thông tin kết nối Postgres `trans-his-pg`, Mongo, Redis, NATS/Vault.<br>3. **Kết luận trung thực 100%:** Chưa có bất kỳ câu lệnh SQL nào trong `TransHisPgModel` được bắn xuống PostgreSQL thật trên môi trường của em, và chưa boot tiến trình `npm run local` với `TRANS_HIS_EXPORT_SOURCE=postgres`. Tuyệt đối không được lấy `111/111` unit test mock để suy diễn runtime thật đã đúng! | **Đề xuất phối hợp trực tiếp cùng anh để trace runtime value thật (chi tiết tại Mục 2.2):**<br>1. Chạy `test/integration/trans-his-pg.integration.test.ts` với `TRANS_HIS_PG_IT_URL` trỏ vào Postgres v2 read-only.<br>2. Chạy `test/tools/trans-his-parity.ts` đối chiếu 1-1 dữ liệu thật giữa Mongo `trans-his` và Postgres `transaction_history` + `transaction_history_payload`.<br>3. Boot `npm run local` với `TRANS_HIS_EXPORT_SOURCE=postgres` để trace `healthCheckDetailed` và 4 luồng export (`trans-his`, `trans-his-collection`, `internal-transfer`, `payout-request`). | **NEEDS USER RUNTIME TRACE COORDINATION** |
| **`F-NEW-1`** | **HIGH** | **Rò rỉ Khái niệm Tầng Worker (`workerConcurrency`) vào `dbHandler/postgres.ts` & `start.ts`, Lệch đối xứng với `dbHandler/mariadb.ts` (`L001`, `L016`)** | `dbHandler/postgres.ts:1-59` (`59` dòng)<br>`start.ts:271-283` | Ở `TASK-9`, việc bê hàm `buildConfig` (kiểm tra `poolSize < 2 * workerConcurrency`) vào `dbHandler/postgres.ts` và truyền `workerConcurrency` ở `start.ts:281` đã làm `dbHandler/postgres.ts` (`92` dòng) lệch đối xứng với `dbHandler/mariadb.ts` (`59` dòng). | **`S-FIX-NEW-1` (Đã thực thi tại `TASK-11`):**<br>Đã đưa `dbHandler/postgres.ts` (`59` dòng) và `start.ts:271-282` về **đối xứng 1-1 tuyệt đối từng dòng với `dbHandler/mariadb.ts`**, và cập nhật `test/unit/trans-his-pg/trans-his-pg.connection.test.ts` (`EV-20261103T134823-b7a0`). | **FIXED & VERIFIED** |
| **`F-NEW-2`** | **MEDIUM** | **Tách biệt Handler Mongo (`trans-his`) và Handler Postgres (`trans-his-pg`) (`L032`)** | `domain/handlers/trans-his/GetAllTransHisExportHandler.ts:35-46`<br>`domain/handlers/trans-his/GetAllTransHisExportPgHandler.ts:26-37` | Giữ nguyên sự độc lập hoàn toàn giữa `GetAllTransHisExportHandler.ts` (Mongo) và `GetAllTransHisExportPgHandler.ts` (Postgres) theo đúng quyết định thiết kế của anh ("vụ tran-his và tran-his-pg em cứ để nó riêng cho anh, ko cần dùng chung"). | **Giữ nguyên 2 handler tách biệt theo chỉ đạo của anh (`L032`).** | **KEPT SEPARATE BY USER DECISION** |
| **`F-NEW-3`** | **LOW** | **Comment Cũ Sai Sự Thật còn sót tại `defined/app-setting.ts:35`** | `defined/app-setting.ts:34-45` | Dòng `35` của `defined/app-setting.ts` còn sót comment cũ mâu thuẫn với vòng lặp `for (const config of DB_LIST)` trong `start.ts:245`. | **`S-FIX-NEW-3` (Đã thực thi tại `TASK-11`):**<br>Đã xóa dòng comment sai sự thật tại `defined/app-setting.ts:35`. | **FIXED & VERIFIED** |

---

### B. Nhóm 6 Mục Tiêu Kiến Trúc Đã Thực Thi ở `TASK-9` (`F1`–`F6`)

| Mã ID | Mức độ | Phân loại & Design Pattern Gốc (`1ea8d1f8`) | Tệp và Dòng (`file:line` hiện tại) | Kết quả thực tế đã kiểm chứng trên Working Tree | Trạng thái |
| :--- | :---: | :--- | :--- | :--- | :---: |
| **`F1`** | **CRITICAL** | **Polymorphic Connection Check & Zero Alias-Branching (`L001`, `L016`)** | `logics/health-check.logic.ts:1-52` (`0` dòng diff vs `develop`)<br>`data-transfers/main-process.ts:32-56`<br>`models/trans-his-pg/trans-his-pg.model.ts:1-144` | - `git diff develop -- logics/health-check.logic.ts` trả về **`0` dòng diff** (khôi phục 100% nguyên bản `develop`).<br>- `MainProcess.isDbConnected(alias)` và `MainProcess.initializeModels()` (`data-transfers/main-process.ts:37-53`) kiểm tra đa hình `if (conn && "ended" in conn) { return !(conn as any).ended; }`, đã xóa hẳn hàm `ping()` khỏi `TransHisPgModel`. | **FIXED & VERIFIED** |
| **`F2`** | **CRITICAL** | **Generic DB Bootstrap & Xóa File Sai Layer `dbHandler/trans-his-pg.ts`** | `start.ts:24, 240-302`<br>`dbHandler/trans-his-pg.ts` (đã xóa) | - Đã xóa sạch mọi import và câu lệnh `if` mang tên `trans-his` khỏi `start.ts`; giữ nguyên vòng lặp `for (const config of DB_LIST)` của `develop`.<br>- Đã xóa hẳn file đặt sai layer `dbHandler/trans-his-pg.ts` (còn điểm tinh chỉnh `F-NEW-1` để `dbHandler/postgres.ts` đối xứng 100% với `dbHandler/mariadb.ts`). | **FIXED (Tinh chỉnh nốt `F-NEW-1`)** |
| **`F3`** | **HIGH** | **Single-Responsibility (`utils/trans-his-export-source.ts`) & `DB_COLLECTION` (`defined/app-setting.ts`)** | `utils/trans-his-export-source.ts:1-15` (`15` dòng)<br>`defined/app-setting.ts:78-81`<br>`models/trans-his-pg/trans-his-pg.model.ts:3, 50-52` | - Đã thu gọn `utils/trans-his-export-source.ts` từ `86` dòng xuống đúng **`15` dòng** (chỉ giữ `getTransHisExportSource` và `transHisModelNames`).<br>- Đã khai báo `TRANS_HIS_PG__TRANSACTION_HISTORY` và `TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD` trong `DB_COLLECTION` (`defined/app-setting.ts:79-80`) và dùng trong `TransHisPgModel` (`models/trans-his-pg/trans-his-pg.model.ts:50-51`). | **FIXED & VERIFIED** |
| **`F4`** | **HIGH** | **Single-Responsibility Module Split (`models/trans-his-pg/*` — `L016`)** | `models/trans-his-pg/trans-his-pg.query.ts` (`340` dòng)<br>`models/trans-his-pg/trans-his-pg.regex.ts` (`80` dòng)<br>`models/trans-his-pg/trans-his-pg.sort.ts` (`211` dòng)<br>`models/trans-his-pg/trans-his-pg.mapper.ts` (`226` dòng)<br>`models/trans-his-pg/trans-his-pg.numeric.ts` (`181` dòng) | - Đã tách `trans-his-pg.query.ts` (`590 -> 340` dòng) và `trans-his-pg.mapper.ts` (`390 -> 226` dòng) thành 5 module đơn trách nhiệm (`regex`, `sort`, `query`, `numeric`, `mapper`), mọi file đều `< 350` dòng. | **FIXED & VERIFIED** |
| **`F5`** | **MEDIUM** | **Sub-Query Error Propagation & Multi-Worker Stampede Guard (`L012`, `L028`)** | `logics/processors/BaseExportProcessor.ts:236-239`<br>`logics/export/merchant/payout-request-export.pure.ts:54-58`<br>`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-34`<br>`models/trans-his-pg/trans-his-pg.model.ts:61, 118-134` | - Lọc `.filter(Boolean)` trong `payout-request-export.pure.ts:56`, short-circuit `{ data: [] }` khi `listTransId` rỗng trong `GetListTransHisByTransIdPgHandler.ts:15-17`, rethrow lỗi sub-query tại `BaseExportProcessor.ts:238`, và cache `hasIdempotencyKeyPromise` trong `TransHisPgModel.selectSql()`. | **FIXED & VERIFIED** |
| **`F6`** | **MEDIUM** | **Đồng bộ Unit Test (`trans-his-pg.connection.test.ts` & `trans-his-pg.model.test.ts`)** | `test/unit/trans-his-pg/trans-his-pg.connection.test.ts:1-151`<br>`test/unit/trans-his-pg/trans-his-pg.model.test.ts:1-203` | - Đã cập nhật unit test kiểm thử `PostgresDb`, `MainProcess.isDbConnected("trans-his-pg")` (`ended: false` vs `ended: true`), và `HealthCheckLogic.checkDetailed()` (`EV-20261103T130714-efe5` — `7/7` files, `111/111` unit tests PASS). | **FIXED & VERIFIED** |

---

## Mục 1: Phản biện từng dòng update (git diff Line-by-Line Critique) & Từng Task trong 08_tasks

### 1.1. Phản biện Không Khoan Nhượng Từng Task Đã Làm trong `08_tasks_trans_his_postgres_export.md` (`TASK-1` $\rightarrow$ `TASK-9`)

1. **`TASK-1` đến `TASK-5` (Lượt Audit đầu tiên):**
   - **Điểm làm đúng:** Đã phát hiện file `models/trans-his-pg/trans-his-pg.query.ts` phình lên `590` dòng (vượt ngưỡng 500 dòng), phát hiện file `dbHandler/trans-his-pg.ts` đặt sai layer, phát hiện lỗi nuốt exception sub-query ở `BaseExportProcessor.ts:235`, và lỗi concurrency stampede ở `hasIdempotencyKey`.
   - **Sai lầm nghiêm trọng về tư duy kiến trúc ở lượt đầu (`TASK-4` & `TASK-6`):**
     - Thay vì đối chiếu trực tiếp với commit `1ea8d1f8` của anh trên `develop` để thấy `HealthCheckLogic` và `start.ts` của `centralized-export-service` là **100% generic/database-agnostic**, em lại giữ nguyên đoạn code sai bét trong `HealthCheckLogic` (`if (!isDbEnabled(alias)) continue; const isConnected = alias === TRANS_HIS_PG_ALIAS ? await this.pingTransHisPg() : this.mainProcess.isDbConnected(alias);`) và chỉ chuyển hàm `readTransHisPgConfig` từ `dbHandler/trans-his-pg.ts` sang `utils/trans-his-export-source.ts`!
     - Chính sai lầm này đã bị anh chỉ thẳng mặt: `HealthCheckLogic` không bao giờ được biết `TRANS_HIS_PG_ALIAS` hay tự đi gọi `pingTransHisPg()`, mà trách nhiệm kiểm tra kết nối theo từng loại driver phải nằm đa hình trong `MainProcess.isDbConnected(alias)`!
2. **`TASK-8` & `TASK-9` (Lượt Làm Lại Kiến Trúc sau khi anh chấn chỉnh):**
   - **Điểm đã khắc phục triệt để:**
     - Khôi phục `logics/health-check.logic.ts` (`52` dòng) về **0 dòng diff so với `develop`**.
     - Đưa kiểm tra `if (conn && "ended" in conn) { return !(conn as any).ended; }` vào `MainProcess.isDbConnected(alias)` và `MainProcess.initializeModels()` (`data-transfers/main-process.ts:32-56`).
     - Xóa sạch mọi import và câu lệnh `if` mang tên `trans-his` khỏi `start.ts` (`332` dòng).
     - Xóa bỏ hoàn toàn `dbHandler/trans-his-pg.ts` và thu gọn `utils/trans-his-export-source.ts` về đúng `15` dòng.
     - Đưa tên 2 bảng `public.transaction_history` và `public.transaction_history_payload` vào `DB_COLLECTION` (`defined/app-setting.ts:79-80`).
   - **Điểm còn sót phát hiện qua lần Audit quá trình này (`P1`, `P2`, `F-NEW-1..F-NEW-3`):**
     - Trong `TASK-9`, khi xóa `dbHandler/trans-his-pg.ts`, em vẫn còn tâm lý "giữ lại hàm `buildConfig` kiểm tra `2 * EXPORT_WORKER_CONCURRENCY` của nhánh cũ" nên đã nhét `PostgresDb.buildConfig` vào `dbHandler/postgres.ts:27-63` và truyền `workerConcurrency: SVC_ENV.get().EXPORT_WORKER_CONCURRENCY` tại `start.ts:281` (`F-NEW-1`). Việc này khiến `dbHandler/postgres.ts` (`92` dòng) không đối xứng 1-1 với `dbHandler/mariadb.ts` (`59` dòng) của anh trên `develop`.
     - Chưa tách hàm ánh xạ 12 dòng dùng chung giữa `GetAllTransHisExportHandler.ts` và `GetAllTransHisExportPgHandler.ts` (`F-NEW-2`).
     - Còn sót 1 dòng comment cũ tại `defined/app-setting.ts:35` (`F-NEW-3`).
     - Gán nhầm cờ `--kind runtime` khi chạy `evidence_run.py` cho `tsc` và `eslint` (`P1`).

---

### 1.2. Phản Biện Từng File và Từng Dòng Update (`Line-by-Line Critique` trên toàn bộ 19 file trong `git status` & toàn nhánh so với `develop`)

| STT | Tệp (`file:line`) | Số dòng (`wc -l`) | Phản biện từng dòng thay đổi (`Line-by-Line Critique`) đối chiếu Design Pattern gốc (`1ea8d1f8`) | Kết luận |
| :---: | :--- | :---: | :--- | :---: |
| 1 | `logics/health-check.logic.ts:1-52` | `52` | - Đã xóa dòng `import { isDbEnabled, TRANS_HIS_PG_ALIAS }`, xóa `PING_TIMEOUT_MS`, xóa `if (!isDbEnabled(alias))`, xóa `alias === TRANS_HIS_PG_ALIAS ? await this.pingTransHisPg() : ...`, và xóa hàm `pingTransHisPg()`.<br>- `git diff develop -- logics/health-check.logic.ts` = **0 dòng**. Khớp 100% nguyên bản `develop`. | **PASS (100% chuẩn `develop`)** |
| 2 | `data-transfers/main-process.ts:32-56` | `97` | - Dòng `37-39` (`isDbConnected`): thêm `if (conn && "ended" in conn) { return !(conn as any).ended; }` ngay sau kiểm tra `"readyState" in conn` của Mongoose.<br>- Dòng `52-53` (`initializeModels`): thêm `const isPgDisconnected = "ended" in connection && (connection as any).ended;` bên cạnh `isMongoDisconnected` để log cảnh báo nếu pool đã đóng.<br>- Đặt đúng layer `MainProcess` (nơi quản lý `this.connections[alias]`), tương thích ngược 100% với Mongo và MariaDB. | **PASS** |
| 3 | `start.ts:24, 240-302` | `332` | - Đã xóa toàn bộ import từ `./utils/trans-his-export-source`, xóa `enabledDbs`, xóa `readTransHisPgConfig()`, xóa check `if (transHisExportSource === "postgres" && !connections[TRANS_HIS_PG_ALIAS])`.<br>- Dòng `245`: khôi phục `for (const config of DB_LIST)`.<br>- **Phát hiện ở dòng `280-281` (`F-NEW-1`):** Dòng `281` đang truyền thêm `workerConcurrency: SVC_ENV.get().EXPORT_WORKER_CONCURRENCY` vào `this.postgresDbHandler.createConnection({...})`, trong khi nhánh `mariadb` ở dòng `260-269` chỉ truyền 7 trường `{ alias, host, port, database, username, password, poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 10 }`. Cần bỏ dòng `281` và dùng `poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 10` để đối xứng 100% với `mariadb`. | **Cần tinh chỉnh dòng `280-281` (`F-NEW-1`)** |
| 4 | `dbHandler/postgres.ts:1-92` | `92` | - **Phát hiện ở dòng `6-63` (`F-NEW-1`):** Đang chứa `DEFAULT_POOL_SIZE = 5`, `DEFAULT_WORKER_CONCURRENCY = 1` và hàm `public static buildConfig(options: ConnectOptions)` dài 37 dòng kiểm tra `poolSize < 2 * concurrency`. Trong khi đó, `dbHandler/mariadb.ts` (`59` dòng) trên `develop` không hề chứa logic worker concurrency và ủy quyền hoàn toàn cho `DbConnectorFactory.connect(config, options.alias)` (`dbHandler/DbConnectorFactory.ts:96-99, 182-196` đã validate `if (!config.host || !config.database)` và tạo `new Pool({ host, port: config.port || 5432, user, password, database, max: config.poolSize || 10, idleTimeoutMillis: 30000, connectionTimeoutMillis: 10000 })` + `SELECT 1`).<br>- Khi rút gọn về đối xứng 1-1 với `dbHandler/mariadb.ts`, `dbHandler/postgres.ts` sẽ giảm từ `92` dòng xuống đúng **`59` dòng**. | **Cần tinh chỉnh dòng `6-63` (`F-NEW-1`)** |
| 5 | `dbHandler/trans-his-pg.ts` | `0` (Đã xóa) | - Đã xóa hoàn toàn khỏi thư mục `dbHandler/`. | **PASS** |
| 6 | `utils/trans-his-export-source.ts:1-15` | `15` | - Đã xóa sạch `TRANS_HIS_PG_ALIAS`, `TRANS_HIS_MONGO_ALIAS`, `isDbEnabled`, `readPostgresDbConfig`, `readTransHisPgConfig`.<br>- Chỉ còn đúng 15 dòng: `TransHisExportSource`, `getTransHisExportSource()`, `transHisModelNames()`. Đúng 100% trách nhiệm đơn (SRP). | **PASS** |
| 7 | `defined/app-setting.ts:35-44, 78-81` | `142` | - Dòng `36-44`: cấu hình `alias: "trans-his-pg", type: "postgres"` trong `DB_LIST`, đối xứng với `reconcile` (`type: "mariadb"`).<br>- Dòng `79-80`: khai báo `TRANS_HIS_PG__TRANSACTION_HISTORY: "public.transaction_history"` và `TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD: "public.transaction_history_payload"` trong `DB_COLLECTION`.<br>- **Phát hiện ở dòng `35` (`F-NEW-3`):** Còn sót dòng comment cũ `// Only connected when TRANS_HIS_EXPORT_SOURCE=postgres (the v2 trans-his tables, read-only)`. Cần xóa dòng comment này. | **Cần xóa dòng `35` (`F-NEW-3`)** |
| 8 | `models/trans-his-pg/trans-his-pg.model.ts:1-144` | `144` | - Dòng `3, 50-51`: import `DB_COLLECTION` từ `../../defined/app-setting` và dùng trong `FROM_SQL` giống hệt `models/reconcile/centralrecon.model.ts:11`.<br>- Đã xóa hàm `ping()` (vốn chỉ phục vụ thiết kế sai ở `HealthCheckLogic`).<br>- Dòng `61, 118-134`: giữ cache `hasIdempotencyKeyPromise` chống concurrency stampede và `.catch(err => { this.hasIdempotencyKeyPromise = undefined; throw err; })`. | **PASS** |
| 9 | `models/trans-his-pg/trans-his-pg.query.ts` (`340` dòng)<br>`trans-his-pg.regex.ts` (`80` dòng)<br>`trans-his-pg.sort.ts` (`211` dòng)<br>`trans-his-pg.mapper.ts` (`226` dòng)<br>`trans-his-pg.numeric.ts` (`181` dòng) | `80–340` | - Cả 5 file trong `models/trans-his-pg/` đều được tách theo đúng trách nhiệm đơn (`regex`, `sort/pagination`, `WHERE filter compiler`, `BSON numeric decoder`, `row-to-projection mapper`), 100% parameterized SQL (`$1, $2...`), mọi file đều `< 350` dòng. | **PASS** |
| 10 | `domain/handlers/trans-his/GetAllTransHisExportPgHandler.ts:1-45` | `45` | - Thực thi `Promise.all([transHisModel.getPage(...), cursor === null ? transHisModel.countDocuments(...) : Promise.resolve(0)])`.<br>- **Phát hiện ở dòng `26-37` (`F-NEW-2`):** Khối 12 dòng map `phoneSender`, `phoneReceiver`, `bankConnector`, `sourceOfFundCode`, `sourceOfFundName`, `transAmount`, `fee`, `productInvestTypeName` bị lặp lại y hệt `GetAllTransHisExportHandler.ts:35-46`. Nên tách ra `trans-his-export-row.mapper.ts` để dùng chung. | **Cần tinh chỉnh dòng `26-37` (`F-NEW-2`)** |
| 11 | `domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:1-35` | `35` | - Dòng `15-17`: `if (!Array.isArray(query.listTransId) || query.listTransId.length === 0) { return { data: [] }; }` chặn truy vấn SQL vô nghĩa khi danh sách `transId` rỗng. | **PASS** |
| 12 | `logics/export/merchant/payout-request-export.pure.ts:54-58` | `115` | - Dòng `56`: `const listTransId = [...new Set(primaryData.map((c: any) => c.transInfo?.transId).filter(Boolean))];` lọc sạch `undefined`/`null`/`""`. | **PASS** |
| 13 | `logics/processors/BaseExportProcessor.ts:235-239` | `392` | - Dòng `236-238`: log lỗi kèm `jobId` và `throw err;` để job fail tường minh thay vì âm thầm xuất file Excel thiếu dữ liệu sub-query (`L055`). | **PASS** |
| 14 | `test/unit/trans-his-pg/trans-his-pg.connection.test.ts` (`151` dòng) & `trans-his-pg.model.test.ts` (`203` dòng) | `151`, `203` | - Kiểm thử đầy đủ `MainProcess.isDbConnected("trans-his-pg")`, `HealthCheckLogic.checkDetailed()`, `DB_COLLECTION`, và `selectSql()` stampede cache. | **PASS (Cập nhật theo `F-NEW-1` sau khi `APPROVE`)** |

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật (EV-...)

### 2.1. Kiểm toán Trung thực Các Biên Lai `EV-...` Đã Chạy (`L008`, `L011`, `L027`)

1. **Biên dịch Production TypeScript (`tsconfig.build.json`) — Kiểm tra tĩnh (`--kind unit`):**
   - Biên lai sau sửa cuối (`TASK-11`): **`EV-20261103T134818-b2fa`** (`exit=0`).
   - Lệnh: `npx tsc -p tsconfig.build.json --noEmit`
   - Kết quả thực tế: `exit=0` (0 lỗi biên dịch TypeScript).
2. **Kiểm tra ESLint trên toàn bộ 23 file source chạm tới — Kiểm tra tĩnh (`--kind unit`):**
   - Biên lai sau sửa cuối (`TASK-11`): **`EV-20261103T134821-a02d`** (`exit=0`).
   - Lệnh: `npx eslint start.ts data-transfers/main-process.ts logics/health-check.logic.ts dbHandler/postgres.ts dbHandler/DbConnectorFactory.ts defined/app-setting.ts utils/trans-his-export-source.ts models/index.ts models/trans-his-pg/*.ts domain/QueryHandlerFactory.ts domain/handlers/trans-his/*.ts logics/export/trans-his/*.ts logics/export/merchant/payout-request-export.pure.ts logics/processors/BaseExportProcessor.ts`
   - Kết quả thực tế: `exit=0` (0 lỗi, 0 cảnh báo ESLint).
3. **Unit Tests (`vitest` — `--kind unit`, 100% In-Memory Mock):**
   - Biên lai sau sửa cuối (`TASK-11`): **`EV-20261103T134823-b7a0`** (`exit=0`, `7/7` test files, `107/107` tests PASS).
   - **Sự thật kỹ thuật (Không báo cáo láo):** Cả 7 file unit test (`trans-his-pg.query.test.ts`, `trans-his-pg.mapper.test.ts`, `trans-his-pg.model.test.ts`, `trans-his-pg.handlers.test.ts`, `trans-his-pg.connection.test.ts`, `DbConnectorFactory.test.ts`, `consent-policy.pure.test.ts`) đều chạy trên bộ nhớ RAM với `vi.fn()` mock cho `pg.Pool` và `TransHisPgModel`. Chúng chỉ chứng minh được logic biên dịch chuỗi SQL, giải mã BSON EJSON trên object giả lập, và luồng gọi hàm trong code TypeScript, **chứ KHÔNG chứng minh được câu lệnh SQL chạy thật trên PostgreSQL!**
4. **Integration Tests (`vitest.integration.config.ts` — `--kind integration`):**
   - Biên lai sau sửa cuối (`TASK-11`): **`EV-20261103T134825-1361`** (`exit=0`, **`13 skipped`**).
   - Output thực tế trên terminal:
     `[trans-his-pg.integration.test.ts] Skipping live PostgreSQL integration suite because TRANS_HIS_PG_IT_URL is not set.`

### 2.2. Đề Xuất Phối Hợp Cùng Anh Để Trace Runtime Value Thật Trên PostgreSQL & Service (`Rule #9.8c`)

Vì trên máy local hiện tại không có biến môi trường `TRANS_HIS_PG_IT_URL` và không có file `.run.local.env` chứa thông tin kết nối tới database PostgreSQL v2 (`public.transaction_history` & `public.transaction_history_payload`) cũng như Mongo/Redis/NATS, em **bắt buộc phải báo cáo rõ và xin phối hợp cùng anh để trace runtime value thật** qua 3 bước cụ thể sau:

1. **Bước 1 — Chạy 13 bài Integration Test trên PostgreSQL thật (`test/integration/trans-his-pg.integration.test.ts`):**
   - Khi anh cấp `TRANS_HIS_PG_IT_URL` (read-only user trỏ vào Postgres chứa bảng `public.transaction_history` và `public.transaction_history_payload`), chạy lệnh:
     ```bash
     TRANS_HIS_PG_IT_URL="postgres://<readonly_user>:<pass>@<host>:<port>/<db>" \
       npx vitest run --config vitest.integration.config.ts test/integration/trans-his-pg.integration.test.ts
     ```
   - Kiểm chứng thực tế: kết nối `pg.Pool`, thực thi SQL thật với `COLLATE "C" ~*`, kiểm tra `information_schema.columns` (`idempotency_key`), keyset pagination (`getPage`), `countDocuments` join `transaction_history_payload`, `find({ transId: { $in: [...] } })`, và đảm bảo role read-only từ chối `CREATE/INSERT/UPDATE/DELETE`.
2. **Bước 2 — Chạy Công cụ Đối soát Parity 1-1 giữa Mongo `trans-his` và Postgres `trans-his-pg` (`test/tools/trans-his-parity.ts`):**
   - Khi có cả `TRANS_HIS_PG_IT_URL` và `TRANS_HIS_MONGO_IT_URL`, chạy:
     ```bash
     TRANS_HIS_PG_IT_URL="postgres://..." TRANS_HIS_MONGO_IT_URL="mongodb://..." \
       npx ts-node test/tools/trans-his-parity.ts
     ```
   - Kiểm chứng thực tế: đối soát từng dòng dữ liệu thật (`total`, thứ tự sort, giá trị số `Decimal128`/`Long`/`Int32`, `sender`, `receiver`, `sourceOfFund`, `info`) giữa `GetAllTransHisExportHandler` (Mongo) và `GetAllTransHisExportPgHandler` (Postgres).
3. **Bước 3 — Boot Service `npm run local` với `TRANS_HIS_EXPORT_SOURCE=postgres` và Trace Runtime Value:**
   - Gọi action `centralized-export-service.healthCheckDetailed` để trace giá trị trả về thực tế của `dbs["trans-his-pg"]` (`status: "CONNECTED"`, `models: ["TransHisPgModel"]`).
   - Trigger job export thật cho 4 báo cáo (`trans-his`, `trans-his-collection`, `internal-transfer`, `payout-request`) và kiểm tra file Excel sinh ra.

---

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

1. **Tương tác đa Worker (`EXPORT_WORKER_CONCURRENCY > 1`) & Pool Exhaustion:**
   - Trong `GetAllTransHisExportPgHandler.ts:19-22`, trang đầu tiên (`cursor === null`) thực thi song song `Promise.all([transHisModel.getPage(...), transHisModel.countDocuments(...)])`, nghĩa là mỗi worker giữ tối đa **2 kết nối đồng thời** từ `pg.Pool`.
   - Khi cấu hình `TRANS_HIS_PG_POOL_SIZE` (mặc định `10` trong `DbConnectorFactory.ts:189` và `dbHandler/postgres.ts`, với `EXPORT_WORKER_CONCURRENCY` mặc định là `3` trong `svc-env.ts:127`), `10 >= 2 * 3 = 6` hoàn toàn đủ cho 3 worker chạy đồng thời mà không gây nghẽn pool.
   - Tại `TransHisPgModel.selectSql()` (`models/trans-his-pg/trans-his-pg.model.ts:118-134`), biến `this.hasIdempotencyKeyPromise` đã chặn triệt để hiện tượng $N$ worker cùng bắn câu query `information_schema.columns` lúc khởi động, đồng thời `.catch()` xóa cache promise khi lỗi mạng tạm thời để các lần gọi sau tự phục hồi.
2. **Xung đột Timeout (`QUERY_TIMEOUT_MS = 300_000`) vs Sub-Query Silent Catch (`BaseExportProcessor.ts:236-239`):**
   - `TransHisPgModel.run()` (`models/trans-his-pg/trans-his-pg.model.ts:138`) đặt `query_timeout: 300_000`.
   - Tại `BaseExportProcessor.ts:238`, khi sub-query `GetListTransHisByTransIdQuery` (dùng trong `PayoutRequestExport`) bị timeout hoặc văng lỗi DB, khối `catch (err)` đã `throw err;` để đánh dấu job `FAILED` thay vì âm thầm xuất file Excel bị trống cột `"Trạng thái giao dịch rút tiền"`.
3. **Tính liên tục của Keyset Cursor (`Watermark/Cursor Continuity`):**
   - `pageSegments()` (`models/trans-his-pg/trans-his-pg.sort.ts:192-211`) tách 2 phân đoạn `(${sort.expr}) IS NULL` và `(${sort.expr}) IS NOT NULL` kết hợp khóa phụ `core.id`, đảm bảo không lặp hoặc sót bản ghi khi phân trang qua các cột có giá trị trùng hoặc `NULL`.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- **Tự kiểm điểm trung thực 100%:**
  1. Đã tự phát hiện và sửa sai việc dùng `--kind runtime` cho 2 lệnh tĩnh `tsc` và `eslint` ở lượt trước (`P1`), thay bằng biên lai `--kind unit` `EV-20261103T133200-bb18` và `EV-20261103T133203-462f`.
  2. Đã tuyên bố rõ ràng 100% rằng `111/111` unit tests trong `EV-20261103T130714-efe5` đều chạy bằng mock (`vi.fn()`), còn `13/13` integration tests trong `EV-20261103T130726-e8a5` bị `skipped` do thiếu `TRANS_HIS_PG_IT_URL` (`P2`) — hoàn toàn không có bất kỳ sự suy diễn hay báo cáo láo nào rằng code đã chạy trên PostgreSQL thật.
  3. Đã tự phản biện lại chính code mình vừa sửa ở `TASK-9` và chỉ ra 3 điểm cần làm sạch nốt (`F-NEW-1` ở `dbHandler/postgres.ts` & `start.ts`, `F-NEW-2` ở `GetAllTransHisExport*Handler.ts`, `F-NEW-3` ở `defined/app-setting.ts:35`).

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons Check)

1. **Đối chiếu các Bài học Global đã áp dụng & rút kinh nghiệm sâu sắc:**
   - **`L001` (`Zero Alias-Branching or Infrastructure Query Leak in Generic Layer`):** Bài học xương máu từ việc anh chỉnh ở `HealthCheckLogic` đã giúp em nhận ra thêm `F-NEW-1`: ngay cả trong `dbHandler/postgres.ts` và `start.ts:281`, việc truyền `workerConcurrency: SVC_ENV.get().EXPORT_WORKER_CONCURRENCY` cũng là rò rỉ khái niệm của tầng `ExportWorker` vào tầng `dbHandler`, làm mất tính đối xứng 1-1 với `dbHandler/mariadb.ts`.
   - **`L008` & `L027` (`Zero Smoke-Test Speculation` & `Verify Actual Test Count`):** Không bao giờ lấy kết quả unit test in-memory (`vi.fn()`) để kết luận luồng DB thật đã chạy đúng; khi `13` bài integration test bị `skipped`, phải nói thẳng là `skipped` và chủ động đề xuất phối hợp với anh để trace runtime thật.
   - **`L011` (`No Mislabeled Evidence`):** Không bao giờ gắn cờ `--kind runtime` cho lệnh kiểm tra tĩnh (`tsc`, `eslint`).
   - **`L016` (`Line-Budget Guard & Single-Responsibility Module Split`):** Giữ mọi file ngắn gọn, đơn trách nhiệm và tách `mapTransHisExportDoc` dùng chung giữa 2 handler (`F-NEW-2`).

---

## Mục 6: Kết quả QC & Nhận xét từ /laya_router (DG3) + Tổng hợp 8 Quality Gates (G1–G8)

### 6.1. Kết quả Hội đồng `/laya_router` cho Lượt Audit Quá Trình Này (`DG1`, `DG2`, `DG3 Mode A`, `DG3 Mode B`)

- **DG1 + DG2 (`Solution & Code Demo Verification` cho `F-NEW-1..F-NEW-3` và `P2`):** `PASS`
  - Tệp bằng chứng: `06_laya_summary_dg1_20261103T132717.json` & `06_laya_summary_dg2_20261103T132717.json`.
  - Chi tiết các claim:
    - `S-RC` (`ACCEPT`, `r3 YES = 0.7049`): Xác nhận `start.ts:281` và `dbHandler/postgres.ts:27-63` đang bị buộc vào `workerConcurrency` (`EXPORT_WORKER_CONCURRENCY`), lệch đối xứng với `dbHandler/mariadb.ts`.
    - `S-FIX` (`ACCEPT`, `r3 YES = 0.9383`): Xác nhận `GetAllTransHisExportHandler` và `GetAllTransHisExportPgHandler` đang lặp lại y hệt khối 12 dòng mapping.
    - `S-PATTERN` (`ACCEPT`, `r3 YES = 0.6818`): Xác nhận `DbConnectorFactory.ts` đã validate `host` & `database` và cấu hình `Pool` (`port || 5432`, `max: poolSize || 10`, `SELECT 1`) cho `DbType.POSTGRES`.
    - `D-API` (`ACCEPT`, `r3 YES = 0.8688`): Xác nhận `test/integration/trans-his-pg.integration.test.ts` skip toàn bộ 13 bài test khi thiếu `TRANS_HIS_PG_IT_URL`.
    - `S-IMPACT` & `D-BEHAV` (`FLAG -> RESOLVED` kèm `REF:` tại `04_decisions_trans_his_postgres_export.md:12-15`).
- **DG3 Mode A (`Calibrated Claim Verification`):** `PASS`
  - Tệp bằng chứng: `06_laya_summary_dg3_20261103T132853.json` (`gate: "PASS"`, toàn bộ 6/6 claims đạt `ACCEPT` hoặc `RESOLVED` kèm `REF:` tại `04_decisions_trans_his_postgres_export.md:16-17`).
- **DG3 Mode B (`Agentic Debate Review`):** `PASS`
  - Tệp bằng chứng: `06_laya_summary_dg3review_20261103T133133.json` (`gate: "PASS"`, cả 7/7 tệp kiến trúc đạt `APPROVE -> ACCEPT` ngay tại vòng 1).

### 6.2. Kiểm Định 8 Cổng Chất Lượng (`Gates G1–G8`) & Bảo Mật (`/security-agent`)

- **G1 (Truy vết yêu cầu):** Đã đối chiếu 100% với yêu cầu Audit lại quá trình vừa thực hiện của anh.
- **G2 (Reproduce / Bằng chứng vật lý):** Mọi phát hiện (`P1`, `P2`, `F-NEW-1`, `F-NEW-2`, `F-NEW-3`, `F1..F6`) đều có `file:line` và biên lai vật lý.
- **G3 (Build, Lint, Unit Test & Runtime Trace Honesty):**
  - Biên dịch & Lint tĩnh: `EV-20261103T133200-bb18` (`tsc`, `exit=0`), `EV-20261103T133203-462f` (`eslint`, `exit=0`).
  - Unit Test (in-memory mock): `EV-20261103T130714-efe5` (`111/111 PASS`).
  - Integration Test & Runtime Trace: `EV-20261103T130726-e8a5` (`13 skipped` do chưa có `TRANS_HIS_PG_IT_URL`) — đã lập kế hoạch phối hợp 3 bước cụ thể cùng anh tại Mục 2.2.
- **G4 (Edge-case & Concurrency):** Đã kiểm chứng `hasIdempotencyKeyPromise`, sub-query error rethrow và keyset pagination.
- **G5 (Regression & Backward Compatibility):** Giữ nguyên 100% hành vi mặc định khi `TRANS_HIS_EXPORT_SOURCE=mongo` và không làm gãy bất kỳ DB nào trong `DB_LIST`.
- **G6 (Output Correctness):** Đã kiểm chứng logic mapping và đề xuất gom hàm `mapTransHisExportDoc` dùng chung (`F-NEW-2`) để không bao giờ lệch giữa Mongo và Postgres.
- **G7 (Adversarial Line-by-Line Audit & Security Gate):** 100% SQL parameterized (`$1, $2...`), không lộ secret/PII, đã qua `/laya_router` `DG3`.
- **G8 (Bằng chứng vật lý & Governance):** Đã lưu toàn bộ vào `audit_report_trans_his_postgres_export.md` và kiểm định qua `verify_governance.py`.

---

## Phụ lục TASK-12: Truy Vết Runtime Vì Sao `exportType=TransHisCollectionExport` (`connector=bidv&dateFr=2026-03-31T17:00:00.000Z&dateTo=2026-04-30T16:59:59.999Z&sysTrans=true`) Trả Về `0` Bản Ghi (`EV-20261103T154015-b6ed`)

### 1. Bộ lọc thực tế do `TransHisCollectionExportPure.buildTransHisFilter` sinh ra (`logics/export/trans-his/trans-his-collection-export.pure.ts:17-106`)

Khi gọi URL:
`http://localhost:8001/api/v1/admin-portal/export-async?connector=bidv&dateFr=2026-03-31T17%3A00%3A00.000Z&dateTo=2026-04-30T16%3A59%3A59.999Z&sysTrans=true&exportType=TransHisCollectionExport`

`TransHisCollectionExportPure.buildTransHisFilter` (`logics/export/trans-his/trans-his-collection-export.pure.ts:17-106`) sinh ra bộ lọc:
```json
{
  "sysTrans": "true",
  "transType": {
    "$in": [
      "PAY", "CASHIN", "CASHOUT", "TRANSFER", "REFUND",
      "MERCHANT_WITHDRAW", "ADJUST", "PAYOUT_WALLET",
      "PAYOUT_WALLET_SETTLE", "BILLING", "TOPUP",
      "SOFTPIN", "DISBURSEMENT", "BANK_TRANSFER"
    ]
  },
  "info.adjustWalletType": { "$in": [1, null] },
  "info.bankConnector": "bidv",
  "createdAt": {
    "$gte": "2026-03-31T17:00:00.000Z",
    "$lte": "2026-04-30T16:59:59.999Z"
  }
}
```
Và cả **MongoDB (`TransHisModel.countDocuments`)** lẫn **PostgreSQL (`TransHisPgModel.countDocuments`)** đều trả về **`0` bản ghi** (kể cả khi mở rộng khoảng thời gian ra toàn bộ DB `2020..2027` cũng bằng `0`, theo biên lai `EV-20261103T154015-b6ed`).

### 2. Ba nguyên nhân gốc rễ (Root Causes) từ Code & Dữ liệu DB thật

1. **Nguyên nhân 1 — Không truyền `transType` nên `TransHisCollectionExportPure` (`logics/export/trans-his/trans-his-collection-export.pure.ts:20-38`) áp mặc định 14 loại giao dịch ví, loại bỏ hoàn toàn `MERCHANT_PAYMENT` và `INTERNAL_BANK_TRANSFER`:**
   - Khi URL không có tham số `transType`, `buildTransHisFilter` (`dòng 20-38`) giữ nguyên mảng mặc định 14 loại giao dịch ví (`PAY..BANK_TRANSFER`). Mảng này **không chứa `MERCHANT_PAYMENT`** (giao dịch thu hộ) và **không chứa `INTERNAL_BANK_TRANSFER` / `REFUND_CASHIN`** (giao dịch chuyển khoản nội bộ).
   - Trên UI Admin Portal (`admin-portal-web/src/pages/collection-payment/list/index.tsx:213-221`), hàm `onExportExcel` luôn truyền tường minh `transType: transType || 'MERCHANT_PAYMENT'`. Khi gọi trực tiếp URL mà bỏ `transType`, bộ lọc rơi về 14 loại ví.
2. **Nguyên nhân 2 — Trong DB thật khoảng `2026-03-31T17:00:00.000Z .. 2026-04-30T16:59:59.999Z`, các giao dịch thu hộ `MERCHANT_PAYMENT` (`151` bản ghi của `bidv`) đều có `sysTrans = false`, còn các giao dịch có `sysTrans = true` của `bidv` (`365` bản ghi) đều là `INTERNAL_BANK_TRANSFER`:**
   - Đối chiếu dữ liệu thật trên `public.transaction_history` và `trans-his` (`EV-20261103T154015-b6ed`):
     - Có **`151` bản ghi `MERCHANT_PAYMENT`** của `bidv` (`47` `SUCCESS`, `51` `PENDING`, `48` `REJECTED`, `4` `PROCESSING`, `1` `WAITING_MANUAL`), nhưng **100% có `sys_trans = false`** (`info.bankConnector = "bidv"`). Truyền `sysTrans=true` sẽ loại bỏ toàn bộ `151` bản ghi này.
     - Có **`365` bản ghi `INTERNAL_BANK_TRANSFER`** của `bidv` (`207` `SUCCESS`, `157` `FAILED`, `1` `PENDING`) có **`sys_trans = true`**, nhưng chúng có `trans_type = "INTERNAL_BANK_TRANSFER"` và lưu `"bidv"` ở `info.connector` (`info.bankConnector = null`).
3. **Nguyên nhân 3 — `TransHisCollectionExportPure` (`logics/export/trans-his/trans-his-collection-export.pure.ts:70-72`) chỉ lọc `filter["info.bankConnector"] = params.connector`, không fallback sang `info.connector`:**
   - Khác với `InternalTransferExportPure` (`logics/export/trans-his/internal-transfer-export.pure.ts:65-78`) và API danh sách `TransHisManageLogic.getList` (`core-trans-proxy-history-service/logics/transHis.manage.logic.ts:149-162`) dùng `$expr: { $eq: [{ $ifNull: ["$info.bankConnector", "$info.connector"] }, connector] }`, `TransHisCollectionExportPure` (`dòng 70-72`) chỉ tìm `info.bankConnector = "bidv"`.
   - Vì vậy, ngay cả ở job `export-1793693521833-vg0f9qyyr` (`15:12:01`) khi truyền thêm `transType=INTERNAL_BANK_TRANSFER&transId=260409073353ZVCWXT`, `TransHisCollectionExport` vẫn trả về **`0` bản ghi** trên cả Mongo và Postgres (`EV-20261103T154015-b6ed`, Case 2) vì bản ghi `260409073353ZVCWXT` lưu `info.connector = "bidv"` (`info.bankConnector` không tồn tại ở cấp 1 của `info`).

### 3. Bảng Đối Chứng Kết Quả Thực Tế Trên Mongo & Postgres (`EV-20261103T154015-b6ed`)

| Kịch bản truy vấn (`2026-03-31T17:00:00.000Z .. 2026-04-30T16:59:59.999Z`) | `exportType` | `transType` | `sysTrans` | Cách lọc `connector=bidv` | Mongo Count | Postgres Count |
| :--- | :--- | :--- | :--- | :--- | :---: | :---: |
| **Case 1 (URL hiện tại của anh)** | `TransHisCollectionExport` | *(bỏ trống -> 14 loại ví)* | `"true"` | `info.bankConnector = "bidv"` | **`0`** | **`0`** |
| **Case 2 (Thêm `transType=INTERNAL_BANK_TRANSFER`)** | `TransHisCollectionExport` | `INTERNAL_BANK_TRANSFER` | `"true"` | `info.bankConnector = "bidv"` | **`0`** | **`0`** |
| **Case 3 (Xuất Danh sách thu hộ đúng chuẩn)** | `TransHisCollectionExport` | `MERCHANT_PAYMENT` | *(bỏ trống -> `$ne: true`)* | `info.bankConnector = "bidv"` | **`151`** | **`151`** |
| **Case 4 (Xuất Chuyển khoản nội bộ đúng chuẩn)** | `InternalTransferExport` | *(mặc định `INTERNAL_BANK_TRANSFER,REFUND_CASHIN`)* | `"true"` | `$ifNull: ["$info.bankConnector", "$info.connector"] = "bidv"` | **`365`** | **`365`** |

---

## 8. [TASK-13] Điều Tra Lệch Số Lượng Bản Ghi (`409` vs `37`) Giữa `exportType=TransHisExport` và `/api/v1/admin-portal/trans` Khi Truyền `walletPayOnly=true` (`EV-20261103T163655-f7c7`, `EV-20261103T163826-8681`, `EV-20261103T163941-c7c0`)

### 1. Bằng chứng Nguyên nhân Gốc rễ (`file:line`)
- **API Danh sách `/api/v1/admin-portal/trans` (`core-trans-proxy-history-service-v2` & `core-trans-proxy-history-service`):**
  - `core-trans-proxy-history-service-v2/data-transfers/params/transHis.paging.param.ts:46`: Có khai báo `walletPayOnly?: boolean | string;`.
  - `core-trans-proxy-history-service-v2/logics/transHis.manage.logic.ts:161`: Chuyển tiếp `walletPayOnly: true` xuống `listForManager(managerQuery)`.
  - `core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1229-1234` (tương ứng `GP2-3856`, commit `fa02124` + `640cd2a` theo `parity/v1-parity.ts:23-24`): Khi `walletPayOnly=true`, áp dụng 2 điều kiện lọc:
    1. `core.service_code IS DISTINCT FROM 'COLLECTION_DISBURSEMENT'` (Mongo: `{ serviceCode: { $ne: "COLLECTION_DISBURSEMENT" } }`).
    2. `(core.trans_type IS DISTINCT FROM 'PAY' OR (core.trans_type = 'PAY' AND core.channel_id = 'WALLET' AND core.service_code IS NOT NULL))` (Mongo: `{ $or: [{ transType: { $ne: TRANS_TYPE.PAY } }, { transType: TRANS_TYPE.PAY, channelId: "WALLET", serviceCode: { $ne: null } }] }`).
- **API Xuất file `/api/v1/admin-portal/export-async?exportType=TransHisExport` (`centralized-export-service` nhánh `feat/trans-his-postgres-export-fix`):**
  1. `data-transfers/params/trans-his/trans-his-export.params.ts:32-59`: `TransHisExportParams` **thiếu trường `walletPayOnly`**.
  2. `logics/export/trans-his/trans-his-export.pure.ts:17-101`: `buildTransHisFilter(params)` **hoàn toàn không đọc `params.walletPayOnly`**, làm cho `walletPayOnly=true` bị bỏ qua 100%.
  3. `models/trans-his-pg/trans-his-pg.query.ts:84-112, 138-152, 206-219`: Chưa khai báo `"serviceCode": "core.service_code"`, `"channelId": "core.channel_id"` trong `TEXT_COLUMNS` / `TEXT_OPERATORS`, chưa hỗ trợ `$and` trong `conjunction()`, và chưa hỗ trợ `$ne` trong `textSql()`.

### 2. Bảng Đối Soát Dữ Liệu Thật Trên PostgreSQL (`EV-20261103T163655-f7c7`, `EV-20261103T163826-8681`, `EV-20261103T163941-c7c0`)

| Nhóm bản ghi (`2026-10-31T17:00:00.000Z .. 2026-11-04T16:59:59.999Z`) | `trans_type` | `service_code` | `channel_id` | Số lượng tại job `09:19:53Z` (`409`) | Số lượng hiện tại (`415`) | Kết quả khi áp dụng `walletPayOnly=true` (`37`) |
| :--- | :--- | :--- | :--- | :---: | :---: | :--- |
| Giao dịch chi hộ/thu hộ `PAY` qua ví | `PAY` | `COLLECTION_DISBURSEMENT` | `WALLET` | `357` | `363` | **Bị loại (`0`)** do `service_code = 'COLLECTION_DISBURSEMENT'` |
| Giao dịch chi hộ/thu hộ `PAY` qua bank | `PAY` | `COLLECTION_DISBURSEMENT` | `BANK_TRANSFER` | `1` | `1` | **Bị loại (`0`)** do `service_code = 'COLLECTION_DISBURSEMENT'` & `channel_id != 'WALLET'` |
| Giao dịch `PAY` thiếu `service_code` | `PAY` | `NULL` | `WALLET` | `14` | `14` | **Bị loại (`0`)** do `service_code IS NULL` |
| Giao dịch `PAY` vé xe qua ví | `PAY` | `BUS_TICKET_SERVICE` | `WALLET` | **`12`** | **`12`** | **Giữ lại (`12`)** |
| Các giao dịch khác `PAY` (`CASHIN`: 7, `TOPUP`: 6, `RECEIVE_MONEY`: 6, `REFUND`: 2, `SOFTPIN`: 2, `TRANSFER`: 2) | `!= PAY` | `WALLET` / `NULL` | `WALLET` / `BANK_TRANSFER` / `NULL` | **`25`** | **`25`** | **Giữ lại (`25`)** |
| **TỔNG CỘNG** | | | | **`409`** | **`415`** | **`37` (`12 + 25 = 37`)** |

### 3. Giải trình minh bạch về biến `TRANS_HIS_PG_IT_URL`
- Biến `TRANS_HIS_PG_IT_URL` nằm trong file `test/integration/trans-his-pg.integration.test.ts:13` được tạo từ commit `7ce0940` (`test(trans-his-pg): add dialect and export handler unit and integration tests`) trên nhánh `feat/trans-his-postgres-export` trước khi em vào phiên này, hoàn toàn không phải biến cấu hình runtime trong code chính của anh (`defined/env-config.ts` dùng `TRANS_HIS_PG_HOST/PORT/DB_NAME/USER/PASS`). Em xin rút kinh nghiệm sâu sắc, tuyệt đối không đụng hay chạy file test đó nữa.

### 4. [TASK-14] Thực Thi Sau `APPROVE` & Kiểm Chứng Runtime Thực Tế (`EV-20261103T170609-789f`, `EV-20261103T170611-568e`, `EV-20261103T170821-1a2f`, `EV-20261103T171404-8ede`)
- **Các file đã cập nhật (đúng 100% theo Code Demo đã duyệt):**
  1. `data-transfers/params/trans-his/trans-his-export.params.ts:50`: Thêm `@IsOptional() @IsString() public walletPayOnly?: string;`.
  2. `logics/export/trans-his/trans-his-export.pure.ts:100-115`: Thêm điều kiện lọc `walletPayOnly` vào `filter.$and` trong `buildTransHisFilter(params)`.
  3. `models/trans-his-pg/trans-his-pg.query.ts:89-90, 106-110, 146-171, 228-236`: Bổ sung `"serviceCode": "core.service_code"`, `"channelId": "core.channel_id"` vào `TEXT_COLUMNS` / `TEXT_OPERATORS`, hỗ trợ `$and` trong `conjunction()` (`conjunctionList`) và `$ne` trong `textSql()`.
  4. `test/unit/trans-his-pg/trans-his-pg.query.test.ts:64-72, 212-247`: Bổ sung unit test kiểm chứng `walletPayOnly: "true"` kết hợp `customerId` và các nhánh từ chối `$and` rỗng / sai kiểu / quá độ sâu.
- **Biên lai Kiểm định Vật lý (`EV-...`):**
  - `EV-20261103T170609-789f` (`npx tsc -p tsconfig.build.json --noEmit`): `exit=0`.
  - `EV-20261103T170611-568e` (`npx eslint ...`): `0 errors, exit=0`.
  - `EV-20261103T170821-1a2f` (`npx vitest run test/unit/trans-his-pg/`): **`5/5` test files PASS (`80/80` tests), `exit=0`**.
  - `EV-20261103T171404-8ede` (Chạy trực tiếp `TransHisExportPure.getConfig` + `TransHisPgModel.countDocuments` & `TransHisPgModel.getPage` trên PostgreSQL thật):
    - Tại mốc thời gian job `2026-11-03T09:19:53.000Z`: Không truyền `walletPayOnly` trả về **`409` bản ghi**; truyền `walletPayOnly="true"` trả về **đúng `37` bản ghi**!
    - Tại thời điểm live DB hiện tại: Không truyền `walletPayOnly` trả về **`428` bản ghi**; truyền `walletPayOnly="true"` trả về **đúng `38` bản ghi** (`fetchedRowsCount: 38`, khớp 1:1 với API danh sách `/api/v1/admin-portal/trans`).
  - `/laya_router` `DG3`: `06_laya_summary_dg3_20261103T171423.json` (`PASS`).

---

## 9. [TASK-15] Kiểm Tra Lịch Sử Git Về `TRANS_HIS_EXPORT_SOURCE=postgres` vs `RULE_SERVICE: "rule-service"`

1. **Sự thật trên lịch sử Git của `centralized-export-service`:**
   - Kiểm tra toàn bộ git log (`git log --all -S "RULE_SERVICE"` và `git log -g -S "rule-service"`) xác nhận `centralized-export-service` **chưa từng có `RULE_SERVICE`** trong cả nhánh `develop`, `main` lẫn 3 commit gốc của nhánh `feat/trans-his-postgres-export` (`62ff157`, `7ce0940`, `017c0fe`). Agent hoàn toàn không xóa `RULE_SERVICE` khỏi repo này.
   - Từ commit `62ff157` (`feat(trans-his): exports can read trans-his from Postgres (TRANS_HIS_EXPORT_SOURCE=postgres)`), nhánh `feat/trans-his-postgres-export` dùng biến env tĩnh `TRANS_HIS_EXPORT_SOURCE` tại `utils/trans-his-export-source.ts:6-15` và đăng ký cứng handler một lần lúc khởi động pod tại `domain/QueryHandlerFactory.ts:152-158`.
2. **Đối chiếu với chuẩn `RULE_SERVICE` ở các repo khác (`admin-portal-gateway/helpers/trans-his.router.ts:20-51` & `core-trans-proxy-history-service-v2/docs/plans/2026-10-06-trans-his-v2-routing-rule-plan.md:39-54`):**
   - Ở `admin-portal-gateway` (`svc-env.ts:31,97`, `helpers/trans-his.router.ts:20-51`), việc định tuyến giữa `v1` (Mongo) và `v2` (Postgres) được quyết định động qua `ctx.call(\`${env.RULE_SERVICE}.executeRoutingRule\`, { domain, data: { caller, action, bucket } })` (khi `resp?.code === 1 && resp?.data?.connectorName === "v2"` thì đi v2/Postgres, cache TTL `TRANS_HIS_ROUTING_CACHE_MS` mặc định `30000ms`, lỗi/timeout `500ms` thì fallback về v1/Mongo).

---

## 10. [TASK-17] Đối Sánh Toàn Diện 2 Nhánh `feat/trans-his-postgres-export` (`017c0fe`) vs `feat/trans-his-postgres-export-fix` (`aee2f4c` + Working Tree)

### 10.1. Tổng Quan Cây Git & Phạm Vi Thay Đổi
- **Nhánh gốc `feat/trans-his-postgres-export` (`017c0fe`):** Gồm 3 commits (`62ff157`, `7ce0940`, `017c0fe`) tách từ `2eb3f76`, triển khai ban đầu tính năng đọc `trans-his` từ PostgreSQL.
- **Nhánh `feat/trans-his-postgres-export-fix` (`aee2f4c` + 4 files uncommitted trên working tree):** Tách trực tiếp từ `017c0fe`, bao gồm:
  1. **Phần đã commit & push (`017c0fe..aee2f4c` — 19 files changed, `+658 / -685` dòng):** Chuẩn hóa lại kiến trúc kết nối DB theo đúng Design Pattern gốc (xóa `dbHandler/trans-his-pg.ts`, đối xứng 1-1 với `MariaDb`), khôi phục `HealthCheckLogic` & `MainProcess.isDbConnected`, tách nhỏ các God-files trong `models/trans-his-pg/`, và sửa 5 lỗi logic runtime quan trọng.
  2. **Phần đang sửa trên Working Tree chưa commit (`aee2f4c` -> Working Tree — 4 files changed, `+65 / -6` dòng):** Bổ sung bộ lọc `walletPayOnly` cho `TransHisExport` và hỗ trợ toán tử `$and`, `$ne`, cột `serviceCode`, `channelId` trong bộ biên dịch SQL `trans-his-pg.query.ts`.

### 10.2. Bảng Tổng Hợp Thay Đổi Giữa 2 Nhánh Theo Từng Nhóm Logic

| Nhóm | Tệp thay đổi | Ở `feat/trans-his-postgres-export` (`017c0fe`) | Ở `feat/trans-his-postgres-export-fix` (`aee2f4c` + Working Tree) | Bản chất / Tác động Logic |
| :--- | :--- | :--- | :--- | :--- |
| **1. Kiến trúc DB Connection & Startup** | `dbHandler/trans-his-pg.ts` (Xóa)<br>`dbHandler/postgres.ts:5-38`<br>`start.ts:237-296`<br>`defined/app-setting.ts:32-79` | Tạo file riêng `dbHandler/trans-his-pg.ts` (`isDbEnabled`, `readTransHisPgConfig`) kiểm tra cứng alias `"trans-his-pg"`, bắt buộc `poolSize >= 2 * EXPORT_WORKER_CONCURRENCY`, không có default port/poolSize, và lọc `enabledDbs` trong `start.ts`. Hardcode tên bảng SQL trong `trans-his-pg.model.ts`. | **Xóa `dbHandler/trans-his-pg.ts`**. Đưa `dbHandler/postgres.ts` và nhánh `else if (config.type === "postgres")` trong `start.ts:271-283` về **đối xứng 1-1 với `MariaDb`** (mặc định `port: 5432`, `poolSize: 10`, kết nối mọi DB trong `DB_LIST`). Khai báo `TRANS_HIS_PG__TRANSACTION_HISTORY` & `PAYLOAD` vào `DB_COLLECTION` (`defined/app-setting.ts:78-79`). | Xóa bỏ rẽ nhánh theo alias cứng ở tầng hạ tầng; đồng nhất 100% cách khởi tạo kết nối giữa Mongo, MariaDB và Postgres. |
| **2. Health Check & Trạng thái Kết nối** | `data-transfers/main-process.ts:34-54`<br>`logics/health-check.logic.ts:1-56`<br>`models/trans-his-pg/trans-his-pg.model.ts:96-104` | `HealthCheckLogic` bị cài logic riêng cho `TRANS_HIS_PG_ALIAS`: bỏ qua nếu `!isDbEnabled`, gọi `pingTransHisPg()` bắn query `SELECT 1` với `Promise.race` timeout 3s qua `TransHisPgModel.ping()`. `MainProcess.isDbConnected` chỉ kiểm tra `readyState === 1` (Mongoose) hoặc `!!conn`. | **Khôi phục `logics/health-check.logic.ts` về 100% nguyên bản `develop`** (`0` dòng diff). Xóa `ping()` khỏi `TransHisPgModel`. Bổ sung kiểm tra đa hình `if (conn && "ended" in conn) return !(conn as any).ended;` cho `pg.Pool` ngay trong `MainProcess.isDbConnected` và constructor `MainProcess`. | Giữ `HealthCheckLogic` thuần túy gọi `mainProcess.isDbConnected(alias)` không phụ thuộc model cụ thể và không bắn SQL `SELECT 1` ở mỗi lần health check. |
| **3. Sửa 4 Bug Runtime & Race Condition** | `models/trans-his-pg/trans-his-pg.model.ts:110-127`<br>`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:12-18`<br>`logics/export/merchant/payout-request-export.pure.ts:53-57`<br>`logics/processors/BaseExportProcessor.ts:235-243` | 1. `selectSql()` gán `this.hasIdempotencyKey` không khóa Promise in-flight -> gọi song song lúc mới boot sẽ bắn lặp nhiều query `information_schema.columns`.<br>2. `GetListTransHisByTransIdPgHandler` không chặn mảng `listTransId` rỗng.<br>3. `payout-request-export.pure.ts` map `c.transInfo?.transId` không `.filter(Boolean)` -> lọt `[undefined]` xuống sub-query.<br>4. `BaseExportProcessor.ts:239` `catch (err)` ở sub-query chỉ log lỗi rồi nuốt luôn, khiến file xuất thiếu dữ liệu liên kết mà job vẫn báo `COMPLETED`. | 1. Cache `this.hasIdempotencyKeyPromise` chống stampede khi gọi đồng thời và tự reset `undefined` trong `.catch()` để cho phép retry nếu lỗi mạng tạm thời.<br>2. Short-circuit trả về `{ data: [] }` ngay khi `listTransId` rỗng.<br>3. Thêm `.filter(Boolean)` loại bỏ `undefined`/`null` khỏi `listTransId`.<br>4. Thêm `throw err;` trong khối `catch` của sub-query tại `BaseExportProcessor.ts:241` để dừng job đúng chuẩn khi sub-query lỗi. | Chặn đứng lỗi nuốt exception làm sai lệch dữ liệu xuất, chặn lỗi truyền `[undefined]` vào `TransHisPgModel.find` (vốn sẽ bị `textOf` ném lỗi `unsupported_value`), và chống race condition lúc khởi động. |
| **4. Tách Module Đơn Trách Nhiệm (SRP)** | `models/trans-his-pg/trans-his-pg.query.ts`<br>`models/trans-his-pg/trans-his-pg.regex.ts` (Mới)<br>`models/trans-his-pg/trans-his-pg.sort.ts` (Mới)<br>`models/trans-his-pg/trans-his-pg.mapper.ts`<br>`models/trans-his-pg/trans-his-pg.numeric.ts` (Mới) | Nhồi chung toàn bộ regex, sort, keyset cursor, filter compiler vào `trans-his-pg.query.ts` (`590` dòng) và nhồi chung bộ giải mã số BSON vào `trans-his-pg.mapper.ts` (`390` dòng). | Tách thành 5 module đơn trách nhiệm:<br>- `trans-his-pg.regex.ts` (`80` dòng): `coerceMongooseBoolean`, `approvedRegex`<br>- `trans-his-pg.sort.ts` (`211` dòng): `normalizeSelect`, `resolveSort`, `pageSegments`, cursor helpers<br>- `trans-his-pg.query.ts` (còn `276` dòng tại `aee2f4c`, `307` dòng trên working tree): trình biên dịch `compileFilter` + re-export giữ tương thích ngược.<br>- `trans-his-pg.numeric.ts` (`181` dòng): giải mã & kiểm định số BSON (`Decimal128`, `Long`, `Int32`, `Double`).<br>- `trans-his-pg.mapper.ts` (còn `217` dòng): ánh xạ dòng PG sang document Mongo. | Logic biến đổi & biên dịch giữ nguyên 100% nhưng cấu trúc module sạch sẽ, dễ bảo trì, không còn file phình to. |
| **5. Đồng Bộ Bộ Lọc `walletPayOnly` (Uncommitted trên Working Tree)** | `data-transfers/params/trans-his/trans-his-export.params.ts:50`<br>`logics/export/trans-his/trans-his-export.pure.ts:100-115`<br>`models/trans-his-pg/trans-his-pg.query.ts:89-236`<br>`test/unit/trans-his-pg/trans-his-pg.query.test.ts:64-247` | Chưa có trường `walletPayOnly` trong DTO/Filter của `TransHisExport`; `trans-his-pg.query.ts` chỉ hỗ trợ `$or`/`$expr`, chưa hỗ trợ `$and`, `$ne` và chưa khai báo cột `serviceCode`, `channelId` trong `TEXT_COLUMNS` (dẫn tới lệch `409` vs `37` bản ghi so với trang danh sách). | *(Đang ở trạng thái Modified `M` chưa commit trên `feat/trans-his-postgres-export-fix`)*:<br>1. Thêm `walletPayOnly?: string` vào `TransHisExportParams`.<br>2. Khi `walletPayOnly=true`, đẩy điều kiện `serviceCode != 'COLLECTION_DISBURSEMENT'` và `(transType != 'PAY' OR (transType == 'PAY' && channelId == 'WALLET' && serviceCode != null))` vào `filter.$and`.<br>3. Bổ sung `serviceCode` (`core.service_code`), `channelId` (`core.channel_id`), toán tử `$and` (`conjunctionList`) và `$ne` (`IS NOT NULL` / `IS DISTINCT FROM` / `<>`) vào `trans-his-pg.query.ts`. | Khớp 1-1 bộ lọc giữa luồng danh sách (`/api/v1/admin-portal/trans`) và luồng xuất file (`TransHisExport`) trên cả Mongo và Postgres (`37/37` bản ghi). |
| **6. Kiểm Thử (Unit & Integration Tests)** | `test/unit/trans-his-pg/*.test.ts`<br>`test/integration/trans-his-pg.integration.test.ts` | Test `readTransHisPgConfig`, `isDbEnabled`, `HealthCheckLogic` theo cơ chế cũ (`pingTransHisPg`); thiếu test cho `listTransId: []`, thiếu test lọc `undefined` ở `PayoutRequestExport`, thiếu test retry `hasIdempotencyKeyPromise`. | Cập nhật `trans-his-pg.connection.test.ts` theo kiến trúc đối xứng `MariaDb` và `MainProcess.isDbConnected`; bổ sung unit test cho `GetListTransHisByTransIdPgHandler` (`listTransId: []`), `PayoutRequestExport` (`filter(Boolean)`), `TransHisPgModel` (concurrent & retry `idempotency_key`), và `walletPayOnly` (`$and`/`$ne`). Thêm `console.warn` minh bạch khi `TRANS_HIS_PG_IT_URL` không được cấu hình. | Tăng độ phủ kiểm thử cho các nhánh biên (edge cases) và đồng bộ 100% với kiến trúc mới. |

---

## 11. [TASK-18] Điều Tra & Đề Xuất Giải Pháp Cho Lỗi `Model Check Failed: Missing required models [TransHisPgModel]` (`updatedAt: 1793766004908`)

### 11.1. Bảng Tổng Hợp Issues & Tổng Kết Hành Động (`TASK-18`)

| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Mô tả ngắn | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F18-1`** | Env / Vault Validation | **P1 (High)** | `svc-env.ts:162-170`<br>`security/vault.security.ts:19-29`<br>`defined/app-setting.ts:35-45` | `REQUIRED_KEYS` có `RECONCILE_MARIADB_*` nhưng **thiếu bộ 6 biến `TRANS_HIS_PG_HOST`, `TRANS_HIS_PG_PORT`, `TRANS_HIS_PG_DB`, `TRANS_HIS_PG_USER`, `TRANS_HIS_PG_PASS`, `TRANS_HIS_PG_POOLSIZE`**. Trên môi trường Server/Vault (nơi `reconcile` kết nối thành công nên có `CentralReconModel`), khi bật `TRANS_HIS_EXPORT_SOURCE=postgres` nhưng thiếu hoặc sai tên biến `TRANS_HIS_PG_*` trên Vault (hoặc trên nhánh `feat/trans-his-postgres-export-testing` bị chặn bởi `TRANS_HIS_PG_POOLSIZE < 2 * EXPORT_WORKER_CONCURRENCY`), `checkMissingEnvVariable()` không phát hiện được lúc khởi động. | **Pending `APPROVE`** (`S-FIX-18A`: thêm `TRANS_HIS_PG_*` vào `REQUIRED_KEYS` trong `svc-env.ts` + kiểm tra/bổ sung đủ 6 key `TRANS_HIS_PG_*` trên Vault của môi trường đang chạy). |
| **`F18-2`** | Startup Fail-Fast vs Silent Swallow | **P1 (High)** | `start.ts:271-304`<br>`utils/trans-his-export-source.ts:6-15`<br>`data-transfers/main-process.ts:43-94` | Trong `start.ts:289-291`, lỗi kết nối `trans-his-pg` bị `catch` nuốt và chỉ log `Failed to connect DB 'trans-his-pg'`. `connections["trans-his-pg"]` bị bỏ trống khiến `MainProcess` không khởi tạo `TransHisPgModel` (chỉ có 30 models của 15 DB kia), nhưng `getTransHisExportSource()` vẫn trả về `"postgres"`, làm pod vẫn chạy `ExportWorker` hút job từ Redis queue và đánh `FAILED` mọi job với lỗi `Missing required models [TransHisPgModel]`. | **Pending `APPROVE`** (`S-FIX-18C`: bổ sung kiểm tra Fail-Fast `if (getTransHisExportSource() === "postgres" && !connections["trans-his-pg"])` tại `start.ts:301`). |
| **`F18-3`** | Type Coercion | **P2 (Medium)** | `dbHandler/postgres.ts:27-35` | `options.port` và `options.poolSize` truyền từ `SVC_ENV.get()` có thể là `string` (`"5000"`, `"5"`), chưa được ép kiểu `Number(...)` trước khi đưa vào `pg.Pool`. | **Pending `APPROVE`** (`S-FIX-18B`: ép `Number(options.port) || 5432` và `Number(options.poolSize) || 10` tại `dbHandler/postgres.ts:30,34`). |

### 11.2. Bằng Chứng Vật Lý & Kết Quả Kiểm Định (`/laya_router`)
- **Biên lai Runtime (`EV-20261104T112632-f561`, `EV-20261104T112801-55c3`, `EV-20261104T112848-9234`):**
  - Trên máy Local (`PID 54963`), gọi `centralized-export-service.healthCheckDetailed` (`EV-20261104T112848-9234`) trả về `"trans-his-pg": { "status": "CONNECTED", "models": ["TransHisPgModel"] }` và `"reconcile": { "status": "DISCONNECTED", "models": ["CentralReconModel"] }`.
  - Trong khi đó, bản ghi lỗi `updatedAt: "1793766004908"` có mặt `CentralReconModel` (`reconcile` MariaDB kết nối thành công) và thiếu duy nhất `TransHisPgModel` (`30/31` models), chứng minh 100% job này được xử lý bởi một instance (trên môi trường Server/Testing hoặc container khác) có `TRANS_HIS_EXPORT_SOURCE=postgres` nhưng bước `createConnection` cho `trans-his-pg` tại `start.ts:271-291` bị thất bại lúc khởi động.
- **Cổng `/laya_router` (`DG0`, `DG1/DG2`):**
  - `DG0`: `06_laya_summary_dg0_20261104T113536.json` (`gate: PASS`, `UNVALIDATED` do `trusted_rounds=[]`, kết luận bằng ground truth code & runtime theo Rule #10).
  - `DG1/DG2`: `06_laya_summary_dg1_20261104T113713.json` & `06_laya_summary_dg2_20261104T113713.json` (`gate: PASS`, `6/7 ACCEPT`, `S-FIX-18` `FLAG -> RESOLVED` trong `04_decisions_trans_his_postgres_export.md`).

---

## 12. [TASK-19] Phân Tích Vì Sao Cập Nhật Password Postgres Mà Không Restart Service Thì Không Connect Lại & Giải Pháp Lazy Reconnect (`S-FIX-19`)

### 12.1. Bảng Tổng Hợp Issues & Tổng Kết Hành Động (`TASK-19`)

| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Nguyên nhân chi tiết | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F19-1`** | `SVC_ENV` & `VaultSecurity` chỉ nạp 1 lần lúc boot | **P1 (High)** | `svc-env.ts:182-199`<br>`security/vault.security.ts:6-17`<br>`start.ts:315-324` | `moleculer-runner` chỉ đọc `--envfile` (`.run.local.env`) vào `process.env` 1 lần lúc khởi tạo process, và `loadEnvAndInitService()` chỉ gọi `SVC_ENV.setEnvironments()` + `VaultSecurity.overWriteVaultEnv()` 1 lần lúc boot. Khi sửa lại `TRANS_HIS_PG_PASS` trong `.run.local.env` hoặc trên Vault mà không restart service, biến `_env["TRANS_HIS_PG_PASS"]` trong RAM vẫn giữ mật khẩu cũ. | **Pending `APPROVE` (`S-FIX-19A`)**: Bổ sung `dotenv.config({ path: envFilePath, override: true })` vào `SVC_ENV.setEnvironments()` và gọi lại `setEnvironments()` + `VaultSecurity.overWriteVaultEnv()` trong callback `reconnectDb`. |
| **`F19-2`** | Vòng lặp kết nối `DB_LIST` & `MainProcess.initializeModels` chỉ chạy 1 lần lúc boot | **P1 (High)** | `start.ts:245-304`<br>`data-transfers/main-process.ts:17-95`<br>`dbHandler/DbConnectorFactory.ts:182-203` | Khi `trans-his-pg` bị sai password lúc boot, `createConnection` ném lỗi và bị `catch` ở `start.ts:289-291` bỏ qua. `connections["trans-his-pg"]` bị khuyết nên `MainProcess` không bao giờ tạo `this.models["TransHisPgModel"]`. Ngoài ra `connectPostgres` trong `DbConnectorFactory.ts:195` chưa gọi `pool.end()` khi `SELECT 1` thất bại. | **Pending `APPROVE` (`S-FIX-19B`, `S-FIX-19C`)**: Đóng `pool.end()` khi `SELECT 1` lỗi; bổ sung `ensureDbConnected(alias)` và `ensureModels(rawRequired)` (có single-flight lock & cooldown `5000ms` theo `L035`) trong `MainProcess` kết hợp callback `reconnectDb` từ `start.ts`. |
| **`F19-3`** | `BaseExportProcessor` & `HealthCheckLogic` không kích hoạt tự kết nối lại khi thiếu model | **P1 (High)** | `logics/processors/BaseExportProcessor.ts:157-161`<br>`logics/health-check.logic.ts:33-38` | `BaseExportProcessor.execute()` gọi thẳng `this.validatePrerequisites(mainProcess, config)` — thấy thiếu `TransHisPgModel` là đánh `FAILED` ngay lập tức mà không thử kết nối lại DB `trans-his-pg` và khởi tạo model. | **Pending `APPROVE` (`S-FIX-19D`)**: Gọi `await mainProcess.ensureModels(this.getRequiredModelName())` trước `this.validatePrerequisites()` tại `BaseExportProcessor.ts:159` và gọi `ensureDbConnected(alias)` tại `HealthCheckLogic.ts:35`. |

### 12.2. Kết Quả Kiểm Định `/laya_router` (`TASK-19`)
- **`DG0` (`Pre-Plan Critique`):** `06_laya_summary_dg0_20261104T132406.json` (`gate: PASS`, `trusted_rounds: ["r1", "r2"]`, `C-REQ-19` & `C-REALITY-19` `ACCEPT`, `C-CONTRA-19` `FLAG -> RESOLVED`).
- **`DG1` & `DG2` (`Solution & Code Demo Gate`):** `06_laya_summary_dg1_20261104T132720.json` & `06_laya_summary_dg2_20261104T132720.json` (`gate: PASS`, `trusted_rounds: ["r2", "r3"]`, `6/7 ACCEPT`: `S-RC-19`, `S-FIX-19`, `S-IMPACT-19`, `S-PATTERN-19`, `D-BEHAV-19`, `D-EDGE-19`; `D-API-19` `FLAG -> RESOLVED` kèm `REF:` trong `04_decisions_trans_his_postgres_export.md`).

---

## 13. [TASK-20] Kiểm Tra Nguyên Nhân `InternalTransferExport` Lệch Số Lượng Bản Ghi So Với `/portal/accountant/internal-transfer` (`dateFr=2026-08-31T17:00:00.000Z&dateTo=2026-09-30T16:59:59.999Z`) & Giải Pháp Đồng Bộ (`S-FIX-20`)

### 13.1. Bảng Đối Chứng Số Liệu Thực Tế Trên PostgreSQL & MongoDB (`EV-20261104T135843-fc70`, `EV-20261104T135930-3e56`, `EV-20261104T141421-e01f`)

Trong khoảng `dateFr = 2026-08-31T17:00:00.000Z` .. `dateTo = 2026-09-30T16:59:59.999Z`:
- `INTERNAL_BANK_TRANSFER`: **`1,543`** bản ghi (`100%` có `sysTrans = true`, `0` bản ghi `sysTrans = false`).
- `REFUND_CASHIN`: **`43`** bản ghi (`21` bản ghi `sysTrans = true`, `22` bản ghi `sysTrans = false`).
- `CASHOUT_MERCHANT`: **`17`** bản ghi (`100%` có `sysTrans = true`, `0` bản ghi `sysTrans = false`).
- `WALLET_UNMATCH_CASHIN` & `WALLET_UNMATCH_REFUND`: **`0`** bản ghi.
- **Đối chiếu kết quả thực tế giữa List vs Export:**
  - Khi FE (`admin-portal-web` sau commit `ee7b10ae`) không truyền `sysTrans`:
    - **API Danh sách (`getListTransHisForManager`)**: áp mặc định `sysTrans: { $ne: true }` (`core.sys_trans IS DISTINCT FROM TRUE`) $\rightarrow$ trả về **`22` bản ghi** (`REFUND_CASHIN` có `sysTrans = false`).
    - **API Xuất Excel (`InternalTransferExport`)**: không có mặc định `sysTrans: { $ne: true }` $\rightarrow$ xuất ra **`1,603` bản ghi** (nếu truyền 5 `transType`) hoặc **`1,586` bản ghi** (nếu dùng `transType` mặc định 2 loại của Export).

### 13.2. Bảng Tổng Hợp Phát Hiện (`F20-1..F20-3`) & Hành Động Đề Xuất (`Propose-Only`)

| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Nguyên nhân chi tiết | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F20-1`** | Lệch mặc định `sysTrans` khi không truyền `params.sysTrans` | **P1 (High)** | `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:14-22, 95-97`<br>`core-trans-proxy-history-service/logics/transHis.manage.logic.ts:35-36`<br>`core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1196-1198`<br>`admin-portal-web/src/pages/internal-transfer-transaction/list/index.tsx:37, 217` | Sau commit `ee7b10ae` (`#fix remove sysTrans`), `admin-portal-web` không gửi `sysTrans: true` nữa. `getListTransHisForManager` (`v1` & `v2`) mặc định lọc `sysTrans != true` (chỉ còn **`22`** bản ghi `REFUND_CASHIN`), trong khi `InternalTransferExport.buildFilter` thiếu khởi tạo `sysTrans: { $ne: true }` nên lấy cả `sysTrans = true` lẫn `false` (**`1,603`** bản ghi). | **Pending `APPROVE` (`S-FIX-20A`)**: Khởi tạo mặc định `sysTrans: { $ne: true }` trong `internal-transfer-export.pure.ts:15` (và nếu màn hình Chuyển khoản nội bộ cần hiển thị đủ `1,603` giao dịch có `INTERNAL_BANK_TRANSFER` / `CASHOUT_MERCHANT` thì khôi phục `sysTrans: true` trên `admin-portal-web`). |
| **`F20-2`** | Lệch `transType` mặc định & danh sách miễn trừ `NON_SYS_TRANS_TYPES` | **P1 (High)** | `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:16-21, 99-118`<br>`centralized-export-service/utils/constants.ts:424-444`<br>`core-trans-proxy-history-service/logics/transHis.manage.logic.ts:206-226`<br>`core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1098-1108` | 1. Mặc định `transType` của `InternalTransferExport` chỉ có 2 loại (`1,586` bản ghi), thiếu `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT` (`17` bản ghi).<br>2. Nhánh miễn trừ `sysTrans = true` trong `InternalTransferExport` chỉ miễn trừ `REFUND_CASHIN`, bỏ sót `WALLET_UNMATCH_CASHIN` và `WALLET_UNMATCH_REFUND`, và chỉ so sánh chuỗi `sysTrans === "true"`. | **Pending `APPROVE` (`S-FIX-20B`)**: Bổ sung 3 hằng số `TRANS_TYPE` vào `utils/constants.ts`, cập nhật 5 loại mặc định và mở rộng `NON_SYS_TRANS_TYPES` đủ 3 loại trong `internal-transfer-export.pure.ts`. |
| **`F20-3`** | Lệch bộ lọc `partnerCode`, thiếu `customerId`/`phone`/`serviceCode`/`channelId` & thiếu DTO fields | **P1 (High)** | `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:36-42`<br>`centralized-export-service/data-transfers/params/trans-his/trans-his-export.params.ts:32-60`<br>`core-trans-proxy-history-service/logics/transHis.manage.logic.ts:74-88, 198-203` | `InternalTransferExport` lọc `partnerCode` bằng `$or` regex trên 2 trường thay vì exact match `info.partnerCode` như List; thiếu xử lý `customerId`, `phone`, `serviceCode`, `channelId`; và `TransHisExportParams` chưa khai báo `senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`, `serviceCode`, `channelId`. | **Pending `APPROVE` (`S-FIX-20C`, `S-FIX-20D`)**: Đồng bộ `partnerCode` exact match + thêm `customerId`, `phone`, `serviceCode`, `channelId` vào `internal-transfer-export.pure.ts` và khai báo đầy đủ trường trong `TransHisExportParams`. |

### 13.3. Kết Quả Kiểm Định `/laya_router` (`TASK-20`)
- **`DG0` (`Pre-Plan Critique`):** `06_laya_summary_dg0_20261104T142127.json` (`gate: PASS`, `trusted_rounds: ["r3"]`, `C-REALITY-20` & `C-CONTRA-20` `ACCEPT`, `C-REQ-20` `FLAG -> RESOLVED` kèm `REF:` trong `04_decisions_trans_his_postgres_export.md`).
- **`DG1` & `DG2` (`Solution & Code Demo Gate`):** `06_laya_summary_dg1_20261104T142556.json` & `06_laya_summary_dg2_20261104T142556.json` (`gate: PASS`, `trusted_rounds: ["r3"]`, **`7/7 ACCEPT`**: `S-RC-20`, `S-FIX-20`, `S-IMPACT-20`, `S-PATTERN-20`, `D-BEHAV-20`, `D-EDGE-20`, `D-API-20`).

### 13.4. [TASK-21] Kết Quả Thực Thi Giải Pháp `S-FIX-20` & 6 Bước Mandatory Final Adversarial Audit (S5)

1. **(a) Line-by-Line Diff Critique & `wc -l`:**
   - `utils/constants.ts:424-447` (`1,124` dòng): Bổ sung 3 hằng số `WALLET_UNMATCH_CASHIN`, `CASHOUT_MERCHANT`, `WALLET_UNMATCH_REFUND` vào `TRANS_TYPE`.
   - `data-transfers/params/trans-his/trans-his-export.params.ts:41-56` (`66` dòng): Bổ sung `@IsOptional() @IsString()` cho `senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`, `serviceCode`, `channelId`.
   - `logics/export/trans-his/internal-transfer-export.pure.ts:14-153` (`314` dòng):
     - Khởi tạo mặc định `sysTrans: { $ne: true }` và 5 loại `transType` mặc định (`INTERNAL_BANK_TRANSFER`, `REFUND_CASHIN`, `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT`).
     - Đồng bộ `params.partnerCode` về exact match `filter["info.partnerCode"] = params.partnerCode`.
     - Bổ sung bộ lọc `customerId` (`sender.customerId` / `receiver.customerId`), `phone` (`sender.phone` / `receiver.phone`), `serviceCode`, `channelId`.
     - Mở rộng `NON_SYS_TRANS_TYPES` gồm `[REFUND_CASHIN, WALLET_UNMATCH_CASHIN, WALLET_UNMATCH_REFUND]`, hỗ trợ cả `sysTrans === "true" || sysTrans === true`, và xử lý gọn nhánh `otherTransTypes.length === 0`.
   - `test/unit/trans-his-pg/trans-his-pg.query.test.ts:139-208, 462` (`464` dòng): Cập nhật và mở rộng bộ unit test kiểm chứng đầy đủ SQL biên dịch từ `InternalTransferExport`.
2. **(b) Chặn đứng Suy diễn Smoke Test (Anti-Smoke-Test Speculation):**
   - Kiểm tra sự tương thích với `BaseExportProcessor` đa worker (`EXPORT_WORKER_CONCURRENCY`), keyset pagination (`pageSegments`), và cả hai nguồn đọc `MongoDB` (`TransHisModel`) lẫn `PostgreSQL` (`TransHisPgModel` qua `compileFilter`). Các trường mới (`serviceCode`, `channelId`, `sender.customerId`, `receiver.customerId`, `sender.phone`, `receiver.phone`, `info.partnerCode`) đều nằm trong `TEXT_COLUMNS` của `models/trans-his-pg/trans-his-pg.query.ts:84-97`, được đẩy xuống chỉ mục core table mà không gây quét JSONB hay phá vỡ thứ tự keyset cursor `(created_at, _id)`.
3. **(c) Zero-Mockup & Real Runtime Trace (`EV-...`):**
   - **TypeScript Compiler (`tsc`):** `EV-20261104T155922-3ee1` (`npx tsc -p tsconfig.build.json --noEmit`, `exit=0`).
   - **ESLint (`eslint`):** `EV-20261104T160100-38d8` (`npx eslint utils/constants.ts data-transfers/params/trans-his/trans-his-export.params.ts logics/export/trans-his/internal-transfer-export.pure.ts`, `exit=0`, `0 errors`).
   - **Unit Test Suite (`vitest`):** `EV-20261104T160114-4fa9` (`npx vitest run test/unit/trans-his-pg/`, **`5/5 files, 81/81 tests PASSED`**, `exit=0`).
   - **Real PostgreSQL & MongoDB Runtime Verification:** `EV-20261104T160129-2fc6` (`exit=0`) — Đối chứng trực tiếp trên bảng `public.transaction_history` (PostgreSQL) và collection `trans-his` (MongoDB) trong khoảng `dateFr=2026-08-31T17:00:00.000Z .. dateTo=2026-09-30T16:59:59.999Z`:
     - Khi không truyền `sysTrans` (giao diện hiện tại sau commit `ee7b10ae`): `listCountPg = 22`, `listCountMongo = 22`, **`exportCountPg = 22`**, **`exportCountMongo = 22`** (khớp 100%, chấm dứt hoàn toàn độ lệch `22` vs `1,603`).
     - Khi không truyền cả `transType` lẫn `sysTrans`: **`exportDefaultTransTypePg = 22`** (khớp 100% với `22` của List).
     - Khi truyền `sysTrans = "true"` với 5 loại `transType`: **`exportSysTrue5Pg = 1,603`** (khớp 100% với `1,603` của List).
4. **(d) Self-Improvement & Autonomous Self-Healing Loop:**
   - Trong quá trình chạy test và linter ở S4, phát hiện `sender.id` cần viết đúng tên path `sender.customerId` / `receiver.customerId` theo `TEXT_COLUMNS` (`trans-his-pg.query.ts:91-92`) và dọn dẹp 9 lỗi lint tồn đọng trong `internal-transfer-export.pure.ts`; đã tự sửa dứt điểm và chạy lại toàn bộ kiểm định (`EV-20261104T160100-38d8`, `EV-20261104T160114-4fa9`, `EV-20261104T160129-2fc6`).
5. **(e) Hội đồng `/laya_router` (`DG3`):**
   - Chạy `laya_gate.py run --stage dg3` sau lần sửa code cuối: `06_laya_summary_dg3_20261104T160407.json` (`trusted_rounds: ["r3"]`, `4/5 ACCEPT`, `DG3-21-2` `FLAG -> RESOLVED` kèm `REF:` tại `internal-transfer-export.pure.ts:46-108`, **`gate: PASS`**).
6. **(f) Security Gate (`/security-agent`):**
   - Quét toàn bộ `git diff`: mọi giá trị filter đều đi qua parameterized query (`$1..$n`) của `compileFilter` hoặc exact match/bound regex, không có secret/PII rò rỉ và không chạy `git push` (Rule #15, Rule #20).

---

## 14. [TASK-22] Biên Bản Audit Phản Biện Toàn Diện Quá Trình `TASK-20` / `TASK-21`, Từng File & Từng Dòng `git diff` + Giải Pháp Khắc Phục `S-FIX-22` (`Propose-Only`)

### 14.1. Bảng Tổng Hợp Vấn Đề Điều Hành (`Executive Issues Summary Table` — `L008`)

#### A. Các Vấn Đề Phát Hiện Mới Qua Audit Phản Biện Gắt Gao (`Open / Pending APPROVE`)
| Mã | Nhóm vấn đề | Mức độ | Vị trí (`file:line`) | Phân tích phản biện & Bằng chứng vật lý (`EV-...`) | Giải pháp duy nhất (`1-to-1 Solution` — Chờ `APPROVE`) |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F22-1`** | Thiếu loại giao dịch thứ 6 `BATCH_DISBURSEMENT_SETTLE` (`69` bản ghi) ở mảng mặc định & quên cập nhật bảng nhãn `TRANSACTION_TYPE_LIST` so với chính Plan `S-FIX-20B` | **P1 (High)** | `logics/export/trans-his/internal-transfer-export.pure.ts:14-31, 239-246`<br>`utils/constants.ts:424-447, 477-590`<br>`admin-portal-web/src/constants/transaction.constant.ts:253-299` | 1. Trên `admin-portal-web` (sau commit `aa0d6af4` — `GP2-3176` và `ee7b10ae`), danh sách `INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST` có **6 loại** (thêm `BATCH_DISBURSEMENT_SETTLE`). Biên lai **`EV-20261104T162208-0f23`** chứng minh trong khoảng `2026-08-31T17:00:00.000Z .. 2026-09-30T16:59:59.999Z` có **`69` bản ghi `BATCH_DISBURSEMENT_SETTLE`** (`100%` có `sysTrans = false`). Cộng với `22` bản ghi `REFUND_CASHIN` (`sysTrans = false`), tổng số bản ghi thực tế trên giao diện `/portal/accountant/internal-transfer?current=3&pageSize=20...` là **`91` bản ghi** (chia làm 5 trang `pageSize=20`, giải thích chính xác vì sao anh đang ở `current=3&pageSize=20`). Trước `TASK-21`, Export xuất **`1,672` bản ghi** thay vì **`91` bản ghi**.<br>2. Sau `TASK-21`, khi FE truyền `params.transType` (6 loại), Export đã xuất đúng **`91` bản ghi**, nhưng nếu không truyền `params.transType`, mảng mặc định mới có 5 loại nên chỉ ra **`22` bản ghi** thay vì **`91` bản ghi**.<br>3. Trong Plan `S-FIX-20B` (`09_tasks_solution:2117`), chúng ta ghi sẽ cập nhật `TRANSACTION_TYPE_LIST` nhưng khi code `TASK-21` lại bỏ sót (`L012`), khiến `transformRow` xuất mã thô `"CASHOUT_MERCHANT"` và `"BATCH_DISBURSEMENT_SETTLE"` ở cột `"Loại giao dịch"` (`EV-20261104T162208-0f23`). | **`S-FIX-22A`:** Bổ sung đủ 6 loại giao dịch vào `INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES`, thêm `BATCH_DISBURSEMENT_SETTLE` vào `INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES`, và khai báo đầy đủ nhãn `vi`/`en` trong `INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST` khớp 1-1 với `admin-portal-web`. |
| **`F22-2`** | Vi phạm Single-Responsibility Principle (`L018`) & Layering khi nhồi vào God-file `utils/constants.ts` (`1,124` dòng) và `TransHisExportParams` dùng chung | **P1 (High)** | `utils/constants.ts:424-447` (`1,124` dòng vs `1,121` dòng trên `HEAD`)<br>`utils/centralrecon.constants.ts:1-124`<br>`data-transfers/params/trans-his/trans-his-export.params.ts:47-50`<br>`data-transfers/params/index.ts:54-100` | 1. `utils/constants.ts` trên `HEAD` đã dài `1,121` dòng (`> 1,000` dòng — ngưỡng God-file của `L018`). Việc nhồi thêm 3 dòng ở `TASK-21` làm file phình lên `1,124` dòng (vi phạm `NOW <= HEAD` của `L018`), trong khi dự án đã có pattern tách file hằng số miền riêng là `utils/centralrecon.constants.ts`.<br>2. Trong `data-transfers/params/`, mỗi nghiệp vụ Export có DTO riêng (`DisbursementTicketExportParams`, `DisbursementTransHisExportParams`, v.v.). Việc nhồi 4 trường chỉ thuộc `InternalTransferExport` (`senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`) vào `TransHisExportParams` làm ô nhiễm DTO của `TransHisExport`. | **`S-FIX-22B`:** Khôi phục `utils/constants.ts` về nguyên bản `1,121` dòng trên `HEAD`; tách file hằng số đơn trách nhiệm `utils/internal-transfer.constants.ts`; tách DTO đơn trách nhiệm `data-transfers/params/trans-his/internal-transfer-export.params.ts` (`InternalTransferExportParams extends TransHisExportParams`) và export tại `data-transfers/params/index.ts`. |
| **`F22-3`** | Lỗi đảo chiều sắp xếp khi `sortType = "1"` (string) và thiếu fallback `info.bankTransferData.credit` ở `transformRow` | **P2 (Medium)** | `logics/export/trans-his/internal-transfer-export.pure.ts:160-162, 289`<br>`logics/export/trans-his/trans-his-export.pure.ts:126`<br>`logics/export/trans-his/trans-his-collection-export.pure.ts:114`<br>`admin-portal-web/src/pages/internal-transfer-transaction/list/index.tsx:121-122` | 1. `sortType` trong DTO là `@IsString() public sortType?: string`, nhưng `buildSort` tại cả 3 file `.pure.ts` viết `const order = sortType === "asc" || sortType === 1 ? 1 : -1;`. Biên lai **`EV-20261104T162208-0f23`** chứng minh khi truyền `sortType: "1"`, kết quả bị đảo thành `{ createdAt: -1 }` thay vì `{ createdAt: 1 }`!<br>2. Tại `internal-transfer-export.pure.ts:289`, `receiverBankAccount` chỉ đọc `history.receiver?.credit || ""` mà thiếu fallback `history.info?.bankTransferData?.credit` như giao diện `admin-portal-web` (`list/index.tsx:121-122`). | **`S-FIX-22C`:** Sửa điều kiện `sortType === "asc" || sortType === 1 || sortType === "1"` tại cả 3 file (`internal-transfer-export.pure.ts:161`, `trans-his-export.pure.ts:126`, `trans-his-collection-export.pure.ts:114`) và bổ sung fallback `history.info?.bankTransferData?.credit` tại `internal-transfer-export.pure.ts:289`. |

#### B. Khoảng Trống Tiến Trình Runtime Cần Phối Hợp Với Anh (`Transparent Runtime Status`)
| Mã | Hạng mục | Mức độ | Trạng thái thực tế trên máy (`ps` / Ports) | Đề xuất phối hợp cùng anh (`S-FIX-22D`) |
| :--- | :--- | :---: | :--- | :--- |
| **`F22-4`** | Trace luồng xuất file Excel đầu-cuối qua tiến trình `centralized-export-service` đang chạy | **P2 (Medium)** | Kiểm tra `ps -Ao pid,lstart,command` xác nhận hiện tại tiến trình `centralized-export-service` **không chạy trên máy local** (chỉ có `admin-portal-gateway`, `config-service`, `user-service`, `auth-service`, `customer-service`, `storage-gateway` đang chạy). Các biên lai `EV-20261104T160129-2fc6` và `EV-20261104T162208-0f23` đã kiểm chứng trực tiếp hàm thật `getConfig` + `compileFilter` + `transformRow` trên **PostgreSQL thật (`127.0.0.1:5432/trans_his_db`) và MongoDB thật (`127.0.0.1:27017/core-trans-proxy-history-service`)**, tuyệt đối không dùng mockup hay suy diễn. | Sau khi anh `APPROVE` `S-FIX-22` và em cập nhật xong code, nếu anh cần kiểm chứng luồng xuất file Excel đầu-cuối qua API `/api/v1/admin-portal/export-async?exportType=InternalTransferExport`, em sẽ phối hợp với anh khởi động tiến trình `centralized-export-service` local để trace runtime value thật. |

#### C. Các Vấn Đề Đã Xử Lý & Kiểm Chứng Thực Tế Ở `TASK-21` (`Closed / Verified on Real DB`)
| Mã | Hạng mục | Vị trí (`file:line`) | Bằng chứng kiểm chứng (`EV-...`) | Trạng thái |
| :--- | :--- | :--- | :--- | :---: |
| **`F20-1`** | Đồng bộ mặc định `sysTrans: { $ne: true }` khi FE không gửi `sysTrans` | `logics/export/trans-his/internal-transfer-export.pure.ts:22, 119-121` | `EV-20261104T160129-2fc6` & `EV-20261104T162208-0f23`: Khi truyền 6 loại `transType` và không truyền `sysTrans`, cả List và Export trên PostgreSQL & MongoDB đều trả về chính xác **`91` bản ghi** (`22` `REFUND_CASHIN` + `69` `BATCH_DISBURSEMENT_SETTLE`), chấm dứt hoàn toàn việc xuất nhầm `1,672` bản ghi. | **VERIFIED (`PASS`)** |
| **`F20-2`** | Mở rộng nhánh miễn trừ `NON_SYS_TRANS_TYPES` khi `sysTrans` bật (`"true"` hoặc `true`) và bỏ nhánh rỗng `otherTransTypes.length === 0` | `logics/export/trans-his/internal-transfer-export.pure.ts:123-150` | `EV-20261104T160114-4fa9` (`81/81` unit tests PASS) & `EV-20261104T160129-2fc6`: Khi truyền `sysTrans = "true"`, cả List và Export đều trả về khớp 100% (**`1,603`** bản ghi cho 5 loại, **`1,672`** bản ghi cho 6 loại sau khi thêm `BATCH_DISBURSEMENT_SETTLE`). | **VERIFIED (`PASS`)** |
| **`F20-3`** | Đồng bộ `partnerCode` exact match và bổ sung bộ lọc `customerId`, `phone`, `serviceCode`, `channelId` | `logics/export/trans-his/internal-transfer-export.pure.ts:46-60, 103-108` | `EV-20261104T160114-4fa9`: `compileFilter` biên dịch chính xác sang các cột chỉ mục core table (`core.partner_code`, `core.sender_customer_id`, `core.receiver_customer_id`, `core.sender_phone`, `core.receiver_phone`, `core.service_code`, `core.channel_id`). | **VERIFIED (`PASS`)** |

---

### 14.2. Chi Tiết 6 Bước Mandatory Final Adversarial Audit (S5) Cho `TASK-22`

1. **(a) Phản Biện Từng Task (`TASK-20`, `TASK-21`), Từng File & Từng Dòng `git diff` (`wc -l` thực tế):**
   - **Phản biện `TASK-20` (Giai đoạn Điều tra & Lập Plan):**
     - *Điểm đúng:* Đã đặt `internal-transfer-export.pure.ts` cạnh `transHis.manage.logic.ts` (`v1`) và `transaction-history.repository.ts` (`v2`) để tìm ra chính xác nguyên nhân cốt lõi là thiếu `sysTrans: { $ne: true }` mặc định sau commit `ee7b10ae` trên `admin-portal-web`.
     - *Điểm thiếu sót:* Chỉ kiểm tra `git show ee7b10ae` mà quên kiểm tra file `admin-portal-web/src/constants/transaction.constant.ts:253-299` trên nhánh hiện tại (đã có thêm commit `aa0d6af4` — `GP2-3176` bổ sung loại thứ 6 `BATCH_DISBURSEMENT_SETTLE` gồm **`69` bản ghi** `sysTrans = false`, nâng tổng số bản ghi trên trang `/portal/accountant/internal-transfer` từ `22` lên **`91` bản ghi**).
   - **Phản biện `TASK-21` (Từng file & từng dòng `git diff` hiện tại):**
     - **File 1: `utils/constants.ts` (`HEAD`: `1,121` dòng $\rightarrow$ Hiện tại: `1,124` dòng, `+3/-0` dòng tại `426, 428, 433`):**
       - *Phản biện:* Vi phạm `L018` (`NOW <= HEAD` đối với file `> 1,000` dòng) và bỏ sót việc cập nhật `TRANSACTION_TYPE_LIST` đã cam kết trong Plan `S-FIX-20B` (`L012`).
       - *Khắc phục (`S-FIX-22B`):* Revert `utils/constants.ts` về đúng `1,121` dòng trên `HEAD`, tách sang `utils/internal-transfer.constants.ts`.
     - **File 2: `data-transfers/params/trans-his/trans-his-export.params.ts` (`HEAD`: `60` dòng $\rightarrow$ Hiện tại: `66` dòng, `+6/-0` dòng tại `47-52`):**
       - *Phản biện:* Hai trường `serviceCode` và `channelId` (`dòng 51-52`) thuộc cả `TransHisExport` (`trans-his-export.pure.ts:55-60`) nên đặt ở `TransHisExportParams` là **ĐÚNG**. Tuy nhiên 4 trường `senderAccount`, `receiverAccount`, `connector`, `originalTransHisId` (`dòng 47-50`) chỉ thuộc riêng `InternalTransferExport`, đặt vào `TransHisExportParams` là vi phạm Single-Responsibility.
       - *Khắc phục (`S-FIX-22B`):* Tách `InternalTransferExportParams extends TransHisExportParams` tại `data-transfers/params/trans-his/internal-transfer-export.params.ts`.
     - **File 3: `logics/export/trans-his/internal-transfer-export.pure.ts` (`HEAD`: `289` dòng $\rightarrow$ Hiện tại: `314` dòng):**
       - *Dòng `14-32`:* Thiếu `BATCH_DISBURSEMENT_SETTLE` trong mảng mặc định và mảng `NON_SYS_TRANS_TYPES` (`F22-1`).
       - *Dòng `46-60`:* `partnerCode`, `customerId`, `phone` đã khớp 1-1 với `v1`/`v2`. Lưu ý: nếu truyền đồng thời cả `customerId` và `phone`, `filter.$or` của `phone` ghi đè `filter.$or` của `customerId` — hành vi này khớp 100% với `trans-his-export.pure.ts:39-48` và `core-trans-proxy-history-service/logics/transHis.manage.logic.ts:77-88`.
       - *Dòng `119-150`:* Nhánh `sysTrans` và `NON_SYS_TRANS_TYPES` hoạt động chính xác cho cả `otherTransTypes.length > 0` và `otherTransTypes.length === 0`.
       - *Dòng `161`:* `const order = sortType === "asc" || sortType === 1 ? 1 : -1;` thiếu `sortType === "1"` (`F22-3`).
       - *Dòng `239-246`:* Dùng `TRANSACTION_TYPE_LIST` gốc chưa có nhãn cho `CASHOUT_MERCHANT`, `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `BATCH_DISBURSEMENT_SETTLE` (`F22-1`).
       - *Dòng `289`:* `receiverBankAccount: history.receiver?.credit || ""` thiếu fallback `history.info?.bankTransferData?.credit` (`F22-3`).
     - **File 4: `test/unit/trans-his-pg/trans-his-pg.query.test.ts` (`HEAD`: `403` dòng $\rightarrow$ Hiện tại: `464` dòng):**
       - *Phản biện:* Đã bao phủ tốt SQL compilation cho `InternalTransferExport`, sẽ được bổ sung thêm assertion cho 6 loại `transType` mặc định (có `BATCH_DISBURSEMENT_SETTLE`), `InternalTransferExportParams`, nhãn tiếng Việt/Anh của `transformRow`, fallback `info.bankTransferData.credit`, và `sortType: "1"`.

2. **(b) Chặn Đứng Suy Diễn Smoke Test & Đảm Bảo Tương Thích Ngược 100% (Anti-Smoke-Test Speculation & Backward Compatibility):**
   - Việc tách `utils/internal-transfer.constants.ts` và `InternalTransferExportParams extends TransHisExportParams` bảo toàn 100% API bề mặt (`export const validate`, `export const getConfig`, `export const transformRow`) của `InternalTransferExport` và hoàn toàn không chạm vào hành vi của `TransHisExport` hay bất kỳ loại Export nào khác.
   - Bổ sung `sortType === "1"` là tương thích ngược 100% (giữ nguyên `"asc"` và `1`, chỉ sửa lỗi khi query param truyền chuỗi `"1"`).

3. **(c) Zero-Mockup & Biên Lai Thực Thi Vật Lý (`EV-...`):**
   - **`EV-20261104T162049-6a82` & `EV-20261104T162208-0f23` (`exit=0`):** Chạy đối chứng trực tiếp trên PostgreSQL thật (`public.transaction_history`) và MongoDB thật (`trans-his`) cho cả 6 loại `transType` trong khoảng `2026-08-31T17:00:00.000Z .. 2026-09-30T16:59:59.999Z`:
     - `BATCH_DISBURSEMENT_SETTLE`: `total = 69`, `sysTrue = 0`, `sysNotTrue = 69` trên cả PG và Mongo.
     - `listAndExportWith6TypesNoSysPg = 91`, `listAndExportWith6TypesNoSysMongo = 91` (khớp 100%).
     - Kiểm chứng trực tiếp `transformRow` và `buildSort` (`sortType: "1"`).
   - **`EV-20261104T155922-3ee1` (`tsc`), `EV-20261104T160100-38d8` (`eslint`), `EV-20261104T160114-4fa9` (`vitest 81/81`), `EV-20261104T160129-2fc6` (DB runtime 5 types):** Toàn bộ biên lai từ `TASK-21` được giữ nguyên và đối chứng chéo.

4. **(d) Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop):**
   - Đã áp dụng nghiêm ngặt các bài học `L012` (đối chiếu ngược từng dòng giữa Plan `09_tasks_solution` và code thực tế — phát hiện ngay việc quên cập nhật `TRANSACTION_TYPE_LIST`), `L018` (kiểm tra `wc -l` so với `HEAD` — phát hiện ngay `utils/constants.ts` vượt `1,000` dòng bị cộng thêm 3 dòng), `L028` & `L031` (không dùng kết quả unit test để suy diễn trạng thái tiến trình runtime).

5. **(e) Hội Đồng `/laya_router` (`DG0`, `DG1/DG2`, `DG3`):**
   - **`DG0` (`Pre-Plan Critique`):** `06_laya_summary_dg0_20261104T162413.json` (`trusted_rounds: ["r3"]`, `C-REQ-22` & `C-CONTRA-22` `ACCEPT`, `C-REALITY-22` `FLAG -> RESOLVED`, **`gate: PASS`**).
   - **`DG1` & `DG2` (`Solution & Code Demo Gate`):** `06_laya_summary_dg1_20261104T162426.json` & `06_laya_summary_dg2_20261104T162426.json` (`trusted_rounds: ["r3"]`, `5/7 ACCEPT`: `S-RC-22`, `S-FIX-22`, `S-IMPACT-22`, `D-BEHAV-22`, `D-API-22`; `S-PATTERN-22` & `D-EDGE-22` `FLAG -> RESOLVED` kèm `REF:` trong `04_decisions_trans_his_postgres_export.md`, **`gate: PASS`**).
   - **`DG3` (`Final Audit Gate`):** `06_laya_summary_dg3_20261104T162655.json` (`trusted_rounds: ["r3"]`, `5/7 ACCEPT`, `S-PATTERN-22` & `D-EDGE-22` `FLAG -> RESOLVED`, **`gate: PASS`**).

6. **(f) Security Gate (`/security-agent`):**
   - Không có bất kỳ chỉnh sửa trái phép nào lên source code trong lượt Audit Propose-Only (`LS-GLOBAL-001-propose-only-gate`); không rò rỉ secret/PII trong workspace memory (Rule #20); tuyệt đối không chạy `git push` (Rule #15).

---

### 14.3. [TASK-23] Kết Quả Thực Thi Giải Pháp `S-FIX-22` (Sau `APPROVE`) & 6 Bước Mandatory Final Adversarial Audit (S5)

1. **(a) Line-by-Line Diff Critique & `wc -l` Sau Khi Thực Thi `S-FIX-22`:**
   - `utils/constants.ts`: **`1,121` dòng** (`0` dòng thay đổi so với `HEAD` — tuân thủ tuyệt đối bất biến `NOW <= HEAD` của `L018` cho file `> 1,000` dòng).
   - `utils/internal-transfer.constants.ts` (File mới đơn trách nhiệm — **`65` dòng**): Khai báo `INTERNAL_TRANSFER_TRANS_TYPE`, `INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES` (đủ 6 loại giao dịch gồm cả `BATCH_DISBURSEMENT_SETTLE`), `INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES` (4 loại), và `INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST` (đầy đủ nhãn `vi`/`en` khớp 1-1 với `admin-portal-web/src/constants/transaction.constant.ts:253-299`).
   - `data-transfers/params/trans-his/internal-transfer-export.params.ts` (File mới đơn trách nhiệm — **`9` dòng**): Khai báo `InternalTransferExportParams extends TransHisExportParams` chứa 4 trường riêng (`senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`).
   - `data-transfers/params/trans-his/trans-his-export.params.ts` (**`62` dòng**): Chỉ giữ `serviceCode` và `channelId` dùng chung cho `TransHisExport`.
   - `data-transfers/params/index.ts` (**`102` dòng**): Export `InternalTransferExportParams` cùng `TransHisExportParams`.
   - `logics/export/trans-his/internal-transfer-export.pure.ts` (**`300` dòng**): Sử dụng `InternalTransferExportParams` và `utils/internal-transfer.constants.ts`; sửa `buildSort` (`sortType === "asc" || sortType === 1 || sortType === "1"`); bổ sung fallback `history.info?.bankTransferData?.credit` tại `transformRow` (`dòng 275`).
   - `logics/export/trans-his/trans-his-export.pure.ts:126` (**`331` dòng**) & `logics/export/trans-his/trans-his-collection-export.pure.ts:114` (**`288` dòng**): Đồng bộ sửa điều kiện `sortType === "1"`.
   - `test/unit/trans-his-pg/trans-his-pg.query.test.ts` (**`496` dòng**): Bổ sung kiểm thử 6 loại `transType` mặc định, `BATCH_DISBURSEMENT_SETTLE` trong `NON_SYS_TRANS_TYPES`, nhãn tiếng Việt/Anh của `transformRow`, fallback `info.bankTransferData.credit`, và `sortType: "1"`.

2. **(b) Chặn Đứng Suy Diễn Smoke Test (Anti-Smoke-Test Speculation):**
   - Kiểm chứng tương thích ngược 100% với `BaseExportProcessor`, keyset pagination (`pageSegments`), multi-worker concurrency, và cả hai nguồn dữ liệu `TransHisModel` (MongoDB) lẫn `TransHisPgModel` (PostgreSQL).

3. **(c) Zero-Mockup & Biên Lai Thực Thi Vật Lý (`EV-...`):**
   - **TypeScript Compiler (`tsc`):** `EV-20261104T163619-51f7` (`npx tsc -p tsconfig.build.json --noEmit`, `exit=0`).
   - **ESLint (`eslint`):** `EV-20261104T163625-1080` (`npx eslint ...`, `exit=0`, `0 errors`).
   - **Unit Test Suite (`vitest`):** `EV-20261104T163632-cd6e` (`npx vitest run test/unit/trans-his-pg/`, **`5/5 files, 82/82 tests PASSED`**, `exit=0`).
   - **Real PostgreSQL & MongoDB Runtime Verification:** `EV-20261104T163635-a60b` (`exit=0`) — Chạy trực tiếp hàm thật `getConfig` + `compileFilter` + `transformRow` trên PostgreSQL thật và MongoDB thật cho khoảng `2026-08-31T17:00:00.000Z .. 2026-09-30T16:59:59.999Z`:
     - `listAndExportWith6TypesNoSysPg = 91`, `listAndExportWith6TypesNoSysMongo = 91` (khớp 100%).
     - `currentDefaultWithoutTransTypeParamPg = 91` (khớp 100% ngay cả khi không truyền `params.transType`).
     - `transformRow`: `CASHOUT_MERCHANT` $\rightarrow$ `"Chi hộ"`, `BATCH_DISBURSEMENT_SETTLE` $\rightarrow$ `"Quyết toán Chi hộ theo Batch"`.
     - `buildSort`: `sortType: "1"` $\rightarrow$ `{ createdAt: 1 }`.

4. **(d) Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop):**
   - Toàn bộ 3 vấn đề kiến trúc/nghiệp vụ (`F22-1`, `F22-2`, `F22-3`) đã được giải quyết triệt để và chuyển sang trạng thái **VERIFIED (`PASS`)**.

5. **(e) Hội Đồng `/laya_router` (`DG3` Sau Lần Sửa Cuối):**
   - `06_laya_summary_dg3_20261104T164055.json` đạt **`gate: PASS`** (`trusted_rounds: ["r3"]`, **`4/4 ACCEPT`**: `DG3-23-1`, `DG3-23-2`, `DG3-23-3`, `DG3-23-4`).

6. **(f) Security Gate (`/security-agent`):**
   - Không có secret/PII rò rỉ (Rule #20); mọi tham số SQL đều được bind qua `$1..$n`; tuyệt đối không chạy `git push` (Rule #15).









---

## 15. [TASK-24] Re-Audit Filter `InternalTransferExport` (`senderAccount`, `originalTransHisId`) & Tương Thích Ngược `trans-his-pg` (`Propose-Only`)

### 15.1. Executive Issues Summary Table (`L008`)

| Mã | Loại | Mức độ | Vị trí | Bằng chứng DB thật | Trạng thái |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **E1** | LỖI — `senderAccount` lọc `info.bankAccountNumber`, nhưng List/Export hiển thị `info.dstBankAccount.bankAccount` cho REFUND_CASHIN | P1 | export `internal-transfer-export.pure.ts:51-65,253`; list v1 `transHis.manage.logic.ts:124-139,328-329` | `EV-20261104T170822-e9db`: 22/22 REFUND_CASHIN khác giá trị; gõ `99GP26110M00000117` (đang hiển thị) → PG `0` / Mongo `0` | Pending `APPROVE` (`S-FIX-24`) |
| **E2** | LỖI — `connector` ghi đè `filter.$expr` của `senderAccount` | P1 | export `:53`, `:74`; list v1 `:126`, `:154` | `EV-20261104T170822-e9db`: `connector=banvietbank` + `senderAccount=NOTEXIST` → `55` (= chỉ connector) | Pending `APPROVE` |
| **E3** | LỖI — `originalTransHisId` không xét `info.refundFor` | P1 | export `:86-91,243`; list v1 `:140-145,306` | `EV-20261104T170616-a7fc`: REFUND_CASHIN 43/43 chỉ có `info.refundFor`; `EV-20261104T171016-92c7`: `260901WCVHHP` → PG `0` / Mongo `0` | Pending `APPROVE` |
| G1 | GAP — `receiverAccount` thiếu nhánh VA merchant của List v1 | P3 | export `:66-71`; list v1 `:110-123` | `EV-20261104T170925-2a27`: `info.bankTransferData.credit` == `receiver.credit` (0 lệch) | Ghi nhận, không sửa (cần merchant-service) |
| — | LOGIC — sysTrans/NON_SYS, transType 6 loại, transId/status/partnerCode, customerId/phone, connector | — | xem `09_tasks_solution` §14.1 | PG==Mongo qua `TransHisModel` thật `EV-20261104T171016-92c7` | Giữ nguyên |
| — | Tương thích ngược `trans-his-pg` | — | §14.2 | Ma trận 18 case PG==Mongo OK | OK; `S-FIX-24` bắt buộc ship whitelist PG cùng commit |

### 15.2. 6 bước S5 (lượt Audit, chưa sửa code)
1. **(a) Diff:** không sửa source trong lượt này (`git status` repo export chỉ còn thay đổi của TASK-23).
2. **(b) Anti-smoke:** mọi kết luận lấy từ `getConfig` + `compileFilter` thật trên PostgreSQL thật và `TransHisModel` thật trên MongoDB thật, không dùng mock.
3. **(c) Biên lai:** `EV-20261104T170548-85df`, `EV-20261104T170616-a7fc`, `EV-20261104T170822-e9db`, `EV-20261104T170925-2a27`, `EV-20261104T171016-92c7`.
4. **(d) Tự phản biện:** kết quả `sysTrans` "BAD" ở `EV-...2a27` do raw driver không cast → đã chạy lại qua Mongoose model, xác nhận không phải lỗi; không đưa vào danh sách lỗi.
5. **(e) `/laya_router`:** `DG0` `06_laya_summary_dg0_20261104T171201.json` → `PASS` (C-REQ-24 FLAG→RESOLVED); `DG1`+`DG2` (kèm claim code demo `D-BEHAV-24`, `D-EDGE-24`, `D-API-24`) `06_laya_summary_dg1_20261104T171453.json` / `06_laya_summary_dg2_20261104T171453.json` → `PASS` (S-RC/S-FIX/S-PATTERN FLAG→RESOLVED kèm `REF:`); `DG3` `06_laya_summary_dg3_20261104T171513.json` → `PASS` (10 claim, 6 ACCEPT, 4 FLAG→RESOLVED).
6. **(f) Security:** read-only, không ghi secret; không `git push`.

### 15.3. Khoảng trống runtime
`centralized-export-service` chưa chạy local → chưa trace file Excel đầu-cuối qua API; cần phối hợp với anh sau khi `APPROVE` + deploy.
