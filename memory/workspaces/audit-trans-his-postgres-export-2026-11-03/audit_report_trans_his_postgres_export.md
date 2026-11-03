---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T11:10:00+07:00
---

# Báo Cáo Audit Phản Biện Toàn Diện — Nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

---

## 0. Bảng Tổng Hợp Issues (Executive Summary Table — `L006`)

### A. Nhóm Issues Phát Hiện Qua Audit & Đã Khắc Phục Sau `APPROVE` (`F1`–`F6`)

| Mã ID | Mức độ | Phân loại | Tệp và Dòng (`file:line` sau khi sửa) | Mô tả chi tiết vấn đề & Kết quả khắc phục (`S4 -> S5`) | Trạng thái |
| :--- | :---: | :--- | :--- | :--- | :---: |
| **`F1`** | **HIGH** | **SRP & God-File (`L016`)** | `models/trans-his-pg/trans-his-pg.query.ts:1-340` (`340` dòng)<br>`models/trans-his-pg/trans-his-pg.regex.ts:1-80` (`80` dòng)<br>`models/trans-his-pg/trans-his-pg.sort.ts:1-211` (`211` dòng)<br>`models/trans-his-pg/trans-his-pg.mapper.ts:1-226` (`226` dòng)<br>`models/trans-his-pg/trans-his-pg.numeric.ts:1-181` (`181` dòng) | - Đã tách `trans-his-pg.query.ts` từ `590` dòng xuống **`340` dòng** bằng cách tách riêng `trans-his-pg.regex.ts` (`80` dòng) và `trans-his-pg.sort.ts` (`211` dòng).<br>- Đã tách `trans-his-pg.mapper.ts` từ `390` dòng xuống **`226` dòng** bằng cách tách riêng `trans-his-pg.numeric.ts` (`181` dòng).<br>- Giữ nguyên 100% re-export public API tại `trans-his-pg.query.ts` và `trans-his-pg.mapper.ts`. | **FIXED & VERIFIED** |
| **`F2`** | **HIGH** | **Layering & Design Pattern (`L014`, `L016`)** | `utils/trans-his-export-source.ts:1-86`<br>`dbHandler/trans-his-pg.ts:1-11`<br>`start.ts:25-31, 282-295`<br>`logics/health-check.logic.ts:5` | - Đã hợp nhất toàn bộ cấu hình nguồn `trans-his` và `readPostgresDbConfig(entry, env)` về `utils/trans-his-export-source.ts:1-86`, giữ `dbHandler/trans-his-pg.ts:1-11` làm barrel re-export mỏng để tương thích ngược.<br>- Tại `start.ts:283`, truyền biến `config` của vòng lặp vào `readPostgresDbConfig(config)` tổng quát cho mọi DB `type === "postgres"`. | **FIXED & VERIFIED** |
| **`F3`** | **HIGH** | **Tương thích ngược & Khớp nối DB (`Backward Compatibility`)** | `utils/trans-his-export-source.ts:30-39`<br>`test/unit/trans-his-pg/trans-his-pg.connection.test.ts:27-37` | - Cập nhật `isDbEnabled(alias, env)`: khi `alias === "trans-his"` và `getTransHisExportSource() === "postgres"`, chỉ bật kết nối Mongo `trans-his` nếu `env.TRANS_HIS_MONGO_URI` thực sự được cấu hình có giá trị; tránh mở kết nối thừa hoặc báo lỗi `DISCONNECTED` giả khi hạ tầng đã cắt sang Postgres. | **FIXED & VERIFIED** |
| **`F4`** | **MEDIUM** | **Silent Error Swallowing & Sub-Query (`L012`, `L028`)** | `logics/processors/BaseExportProcessor.ts:236-239`<br>`logics/export/merchant/payout-request-export.pure.ts:54-58`<br>`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-34` | - Đã lọc `.filter(Boolean)` trong `payout-request-export.pure.ts:56` và thêm short-circuit return `{ data: [] }` khi `listTransId` rỗng tại `GetListTransHisByTransIdPgHandler.ts:15-17`.<br>- Đã thêm `throw err;` tại `BaseExportProcessor.ts:238` để không nuốt lỗi sub-query. | **FIXED & VERIFIED** |
| **`F5`** | **MEDIUM** | **Concurrency Stampede (`Multi-Worker`)** | `models/trans-his-pg/trans-his-pg.model.ts:61, 122-138`<br>`test/unit/trans-his-pg/trans-his-pg.model.test.ts:183-209` | - Đã lưu `hasIdempotencyKeyPromise` trong `TransHisPgModel.selectSql()` để gộp mọi truy vấn đồng thời vào đúng 1 câu lệnh `information_schema.columns` duy nhất, và reset `this.hasIdempotencyKeyPromise = undefined` trong `.catch(...)` để cho phép retry khi lỗi mạng tạm thời. | **FIXED & VERIFIED** |
| **`F6`** | **LOW** | **ESLint, Silent Test Skip (`L007`) & Git Topology (`L001`)** | `start.ts:2-4, 223`<br>`test/integration/trans-his-pg.integration.test.ts:22-24` | - Đã sửa dứt điểm 2 lỗi ESLint tại `start.ts` (`EV-20261103T113358-97a0` exit=0 trên toàn bộ 23 file source).<br>- Đã bổ sung `console.warn` minh bạch trong `trans-his-pg.integration.test.ts:22-24` khi thiếu `TRANS_HIS_PG_IT_URL` (`EV-20261103T113239-b095`). | **FIXED & VERIFIED** |

---

### B. Nhóm Hạng Mục Đã Xử Lý Qua Các Commit / Đạt Chuẩn (Resolved / Verified)

| Mã ID | Phân loại | Tệp và Dòng (`file:line` tại HEAD `017c0fe`) | Nội dung đã kiểm chứng đạt chuẩn | Trạng thái |
| :--- | :--- | :--- | :--- | :---: |
| **`R1`** | **Adapter Pattern (`dbHandler/`)** | `dbHandler/postgres.ts:1-61`<br>`dbHandler/DbConnectorFactory.ts:59-64, 144-169` | Commit `62ff157` ban đầu gọi thẳng `DbConnectorFactory` từ `start.ts`, nhưng commit `017c0fe` đã khôi phục `PostgresDb` wrapper (`dbHandler/postgres.ts`) đồng bộ 1-1 với `MariaDb` (`dbHandler/mariadb.ts`) và gắn listener `pool.on("error")` để ngăn unhandled error làm sập tiến trình Node.js. | **VERIFIED** |
| **`R2`** | **Explicit Connection Config & Pool Sizing** | `dbHandler/trans-his-pg.ts:22-57` | Commit `7ce0940` đã bỏ các giá trị mặc định nguy hiểm (như port `5432` trỏ nhầm vào primary ghi), bắt buộc khai báo đủ 5 biến `TRANS_HIS_PG_HOST/PORT/DB/USER/PASS`, và cưỡng chế `poolSize >= 2 * EXPORT_WORKER_CONCURRENCY` (do mỗi job giữ 2 client song song cho `getPage` và `countDocuments`). | **VERIFIED** |
| **`R3`** | **Joined Count Consistency** | `models/trans-his-pg/trans-his-pg.model.ts:49-51, 104-110` | Commit `62ff157` ban đầu bỏ qua `JOIN transaction_history_payload` trong `countDocuments` khi filter không chạm tới `payload`, gây lệch `total` so với `getPage`. Commit `7ce0940` đã sửa `countDocuments` dùng chung `FROM_SQL` (`INNER JOIN public.transaction_history_payload`) khớp tuyệt đối với `getPage`. | **VERIFIED** |
| **`R4`** | **Generic `find(filter, select)` API Parity** | `models/trans-his-pg/trans-his-pg.model.ts:112-119`<br>`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-31` | Commit `62ff157` ban đầu tạo hàm riêng `findByTransIds(listTransId)`. Commit `017c0fe` đã chuẩn hóa về `find(filter, select)` nhận `{ transId: { $in: listTransId } }` và `selectFields`, đồng nhất giao diện với `TransHisModel` bên Mongo. | **VERIFIED** |
| **`R5`** | **Parameterized SQL & Injection Safety** | `models/trans-his-pg/trans-his-pg.query.ts:60-65, 161-419` | 100% giá trị đầu vào của người dùng được đưa vào mảng `values` (`$1, $2...`), tuyệt đối không nối chuỗi trực tiếp vào SQL; từ chối ngay các toán tử lạ hoặc ký tự `\u0000` bằng `TransHisPgUnsupportedError`. | **VERIFIED** |
| **`R6`** | **Regex Translation (`approvedRegex`) & Collation `"C"`** | `models/trans-his-pg/trans-his-pg.query.ts:90-157, 297-318` | Commit `017c0fe` đã port `approvedRegex` từ `core-trans-proxy-history-service-v2`, dịch cú pháp JS regex (`^`, `$`, `.`, `\d`, `\w`, `\s`, `\b`, `\B`) sang PostgreSQL ARE dưới `COLLATE "C" ~*`, và chặn các regex không tương thích (lookaround, backreference, non-ASCII). | **VERIFIED** |
| **`R7`** | **Unknown Filter Keys (`transIdRef`, spread `params.info` string)** | `models/trans-his-pg/trans-his-pg.query.ts:390-405` | Commit `017c0fe` đã xử lý đúng hành vi của `trans-his-export.pure.ts:82-98` (khi truyền `transIdRef` hoặc spread `params.info` kiểu string thành các key `"0"`, `"1"`...): biên dịch thành `FALSE` (trả về rỗng giống Mongo) thay vì văng lỗi làm hỏng job. | **VERIFIED** |
| **`R8`** | **BSON Numeric & Metadata Fidelity** | `models/trans-his-pg/trans-his-pg.mapper.ts:48-390` | Giải mã chuẩn xác EJSON (`Int32`, `Double` $\rightarrow$ `number`; `Long` $\rightarrow$ `number` khi an toàn hoặc giữ `mongo.Long`; `Decimal128` $\rightarrow$ `mongo.Decimal128`) dùng chính BSON classes của `mongoose.mongo` và kiểm tra chéo `source_field_types`. | **VERIFIED** |
| **`R9`** | **Tương thích ngược khi `TRANS_HIS_EXPORT_SOURCE` mặc định (`mongo`)** | `utils/trans-his-export-source.ts:6-15`<br>`domain/QueryHandlerFactory.ts:52-55, 81-83` | Khi không cấu hình `TRANS_HIS_EXPORT_SOURCE` (hoặc đặt `"mongo"`), hệ thống giữ nguyên 100% hành vi của `develop`: không kết nối `trans-his-pg`, dùng `TransHisModel` và `GetAllTransHisExportHandler` / `GetListTransHisByTransIdHandler` cũ. | **VERIFIED** |

---

## Mục 1: Phản biện từng dòng update (git diff Line-by-Line Critique) & Từng Task trong 08_tasks

### 1.1. Phản biện Topology Git & Tiến trình 3 Commit trên `feat/trans-his-postgres-export`
- **Điểm xuất phát của nhánh (`L001`):**
  - Nhánh `feat/trans-his-postgres-export` (`HEAD = 017c0fe`) được rẽ nhánh từ commit `2eb3f76` (`Merge branch 'release/v1.23.0' into 'main'`), trong khi nhánh `develop` đang ở `9a6cc43` (chứa thêm commit `a181ef1` nâng cấp `Dockerfile` lên `node:24.22.1-alpine3.22` vá lỗ hổng bảo mật).
  - Do đó, `git diff develop...HEAD` chỉ hiển thị đúng **28 tệp** thuộc tính năng Postgres export (`2eb3f76..017c0fe`), còn `git diff develop HEAD` có thêm chênh lệch tại `Dockerfile`.
- **Phân tích phản biện quá trình tiến hóa qua 3 commit (`2eb3f76..017c0fe`):**
  1. **Commit 1 — `62ff157` (`feat(trans-his): exports can read trans-his from Postgres`):**
     - Thiết lập khung kết nối và truy vấn Postgres đầu tiên cho 3 báo cáo `TransHisExport`, `TransHisCollectionExport`, `InternalTransferExport` và sub-query `PayoutRequestExport`.
     - **Sai sót thiết kế ở bước 1:** (i) Bỏ qua lớp `dbHandler/postgres.ts` mà gọi thẳng `DbConnectorFactory` trong `start.ts`; (ii) Đặt hàm riêng `findByTransIds` trên `TransHisPgModel` lệch chuẩn `TransHisModel.find`; (iii) Tối ưu hóa sai trong `countDocuments` khi bỏ `JOIN transaction_history_payload` nếu filter không có điều kiện payload; (iv) Quên dịch JS regex sang Postgres regex và văng lỗi khi gặp các key lạ (`transIdRef`, hoặc khi `params.info` là chuỗi bị `_.keys(params.info)` tách thành key `"0", "1"...`); (v) Cho phép default port `5432` và không kiểm tra `poolSize >= 2 * EXPORT_WORKER_CONCURRENCY`.
  2. **Commit 2 — `7ce0940` (`fix(trans-his-pg): explicit connection settings, pool sizing, joined count, health ping, opt-in integration and parity tools`):**
     - Khắc phục rủi ro vận hành: bắt buộc khai báo tường minh `TRANS_HIS_PG_HOST/PORT/DB/USER/PASS`, kiểm tra `poolSize >= 2 * EXPORT_WORKER_CONCURRENCY`, luôn `INNER JOIN payload` trong `countDocuments`, thêm `ping()` 3s trong `HealthCheckLogic`, và bổ sung `trans-his-pg.integration.test.ts` + công cụ đối soát `trans-his-parity.ts`.
     - **Hệ lụy phát sinh ở bước 2:** Tạo file `dbHandler/trans-his-pg.ts` nằm trong thư mục `dbHandler/` nhưng chứa logic nghiệp vụ/cấu hình riêng của `trans-his-pg`.
  3. **Commit 3 — `017c0fe` (`fix(trans-his-pg): unknown filter keys match nothing, sort by more stored fields, approvedRegex, idempotencyKey, PostgresDb wrapper, generic find`):**
     - Hoàn thiện độ chính xác ngữ nghĩa: khôi phục `dbHandler/postgres.ts` (`PostgresDb`), đổi `findByTransIds` thành `find(filter, select)`, thêm `approvedRegex`, hỗ trợ tự động phát hiện cột `idempotency_key` qua `information_schema.columns`, mở rộng bảng `SORT_EXPRESSIONS` cho `InternalTransferExport`, và xử lý unknown top-level filter keys thành `FALSE`.
     - **Hệ lụy còn sót lại sau bước 3 (`F1`–`F6`):**
       - Việc nhồi thêm `approvedRegex` (+68 dòng) và mở rộng `SORT_EXPRESSIONS` (+25 dòng) vào thẳng `models/trans-his-pg/trans-his-pg.query.ts` đẩy file này từ `491` dòng vượt ngưỡng lên **`590` dòng** (`F1`).
       - `dbHandler/trans-his-pg.ts` sau khi trả việc tạo pool về cho `dbHandler/postgres.ts` trở thành một file cấu hình đặt sai layer (`F2`), và `start.ts:278` vẫn gọi cứng `readTransHisPgConfig()` trong vòng lặp `config.type === "postgres"` (`F2`).
       - `selectSql()` thêm truy vấn `information_schema.columns` nhưng thiếu cơ chế khóa/cache in-flight Promise khi chạy đa luồng (`F5`).

### 1.2. Đo đạc thực tế `wc -l` trên toàn bộ tệp tính năng sau khi sửa (`S5 Final Line-by-Line Audit`)

| STT | Đường dẫn tệp (`centralized-export-service`) | Số dòng trước sửa | Số dòng sau sửa (`wc -l`) | Phân tầng Layer | Đánh giá SRP & Ngân sách dòng (`L016`) |
| :---: | :--- | :---: | :---: | :--- | :--- |
| 1 | `dbHandler/DbConnectorFactory.ts` | `239` | `239` | `dbHandler/` (Infra) | Đạt (`< 500` dòng). |
| 2 | `dbHandler/postgres.ts` | `61` | `61` | `dbHandler/` (Infra) | Đạt (`< 500` dòng). |
| 3 | `dbHandler/trans-his-pg.ts` | `57` | **`11`** | `dbHandler/` (Barrel) | Đã chuyển thành barrel re-export mỏng (`11` dòng) giữ tương thích ngược. |
| 4 | `defined/app-setting.ts` | `140` | `140` | `defined/` | Đạt. |
| 5 | `domain/QueryHandlerFactory.ts` | `160` | `160` | `domain/` | Đạt. |
| 6 | `domain/handlers/trans-his/GetAllTransHisExportPgHandler.ts` | `45` | `45` | `domain/handlers/` | Đạt. |
| 7 | `domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts` | `32` | **`35`** | `domain/handlers/` | Đạt (`F4`: thêm short-circuit khi `listTransId` rỗng). |
| 8–10 | `logics/export/trans-his/*.export.ts` (3 tệp) | `28–29` | `28–29` | `logics/export/` | Đạt. |
| 11 | `logics/export/merchant/payout-request-export.pure.ts` | `115` | `115` | `logics/export/` | Đạt (`F4`: lọc `.filter(Boolean)` trong `buildSubQueryParams`). |
| 12 | `logics/processors/BaseExportProcessor.ts` | `391` | `392` | `logics/processors/` | Đạt (`F4`: rethrow lỗi sub-query tại dòng `238`). |
| 13 | `logics/health-check.logic.ts` | `71` | `71` | `logics/` | Đạt (`F2`/`F3`: import chuẩn từ `utils/trans-his-export-source`). |
| 14 | `models/index.ts` | `175` | `175` | `models/` | Đạt. |
| 15 | `models/trans-his-pg/trans-his-pg.mapper.ts` | `390` | **`226`** | `models/trans-his-pg/` | Đạt (`F1`: đã tách numeric sang `trans-his-pg.numeric.ts`). |
| 16 | `models/trans-his-pg/trans-his-pg.numeric.ts` | *(mới)* | **`181`** | `models/trans-his-pg/` | Đạt (`F1`: đơn trách nhiệm giải mã & kiểm định BSON numeric). |
| 17 | `models/trans-his-pg/trans-his-pg.model.ts` | `136` | **`148`** | `models/trans-his-pg/` | Đạt (`F5`: thêm cache `hasIdempotencyKeyPromise`). |
| 18 | `models/trans-his-pg/trans-his-pg.query.ts` | `590` | **`340`** | `models/trans-his-pg/` | Đạt (`F1`: giảm từ `590` xuống `340` dòng < 500 dòng). |
| 19 | `models/trans-his-pg/trans-his-pg.regex.ts` | *(mới)* | **`80`** | `models/trans-his-pg/` | Đạt (`F1`: đơn trách nhiệm `approvedRegex` & `coerceMongooseBoolean`). |
| 20 | `models/trans-his-pg/trans-his-pg.sort.ts` | *(mới)* | **`211`** | `models/trans-his-pg/` | Đạt (`F1`: đơn trách nhiệm `normalizeSelect`, `resolveSort`, `pageSegments`). |
| 21 | `start.ts` | `341` | **`346`** | Root Entry | Đạt (`F2` & `F6`: dùng `readPostgresDbConfig(config)` và `0` lỗi ESLint). |
| 22 | `svc-env.ts` | `200` | `200` | Root Config | Đạt. |
| 23 | `utils/trans-his-export-source.ts` | `15` | **`86`** | `utils/` | Đạt (`F2` & `F3`: nơi duy nhất quản lý cấu hình nguồn `trans-his`). |
| 24–33 | `test/unit/**`, `test/integration/**`, `test/tools/**` (10 tệp) | `1960` | **`2022`** | `test/` | Đạt (`< 450` dòng/file, `114/114` unit tests PASS). |

---

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

1. **Tương tác đa Worker (`EXPORT_WORKER_CONCURRENCY > 1`) & Pool Exhaustion:**
   - Trong `GetAllTransHisExportPgHandler.ts:19-28`, trang đầu tiên (`cursor === null`) thực thi `Promise.all([transHisModel.getPage(...), transHisModel.countDocuments(...)])`, giữ **2 kết nối đồng thời** trên mỗi worker.
   - `readTransHisPgConfig()` (`dbHandler/trans-his-pg.ts:41-47`) đã kiểm tra chặt `poolSize >= 2 * concurrency` (nếu `poolSize < 2 * concurrency` sẽ chặn ngay lúc khởi động).
   - **Điểm hở khi chạy đa worker (`F5`):** Ngay khi service vừa boot xong và nhận đồng thời $N$ job (hoặc cả `getPage` + `find` chạy song song trước khi câu query đầu tiên kịp trả về), `this.hasIdempotencyKey` trong `TransHisPgModel` (`models/trans-his-pg/trans-his-pg.model.ts:60, 121-126`) vẫn là `undefined`. Do không lưu `hasIdempotencyKeyPromise`, cả $N$ luồng đều bắn câu query `SELECT 1 FROM information_schema.columns ...` xuống Postgres cùng lúc, chiếm dụng kết nối trong pool ngay thời điểm khởi phát.
2. **Xung đột Timeout (`QUERY_TIMEOUT_MS = 300_000`) vs Sub-Query Silent Catch (`F4`):**
   - `TransHisPgModel.run()` (`models/trans-his-pg/trans-his-pg.model.ts:15, 128-133`) đặt `query_timeout: 300_000` (5 phút) ở phía client `pg`.
   - Trong các luồng chính (`TransHisExport`, `TransHisCollectionExport`, `InternalTransferExport`), nếu `getPage()` hoặc `countDocuments()` vượt quá 300s hoặc gặp lỗi kết nối/metadata, `GetAllTransHisExportPgHandler` rethrow lỗi (`GetAllTransHisExportPgHandler.ts:39-41`) và `BaseExportProcessor` đánh dấu job `FAILED`.
   - **Ngược lại, ở luồng `PayoutRequestExport` (`F4`):** `GetListTransHisByTransIdPgHandler` được gọi với tư cách **sub-query** bên trong khối `try { ... } catch (err) { this.context.broker.logger.error(...); }` của `BaseExportProcessor.ts:220-238`. Nếu truy vấn Postgres `find({ transId: { $in: listTransId } })` bị timeout hoặc văng lỗi, lỗi bị nuốt hoàn toàn! File Excel xuất ra vẫn báo `COMPLETED` nhưng toàn bộ các dòng trong trang đó bị mất trắng cột `"Trạng thái giao dịch rút tiền"` (`cashOutTransStatus: null`).
3. **Tính liên tục của Keyset Cursor (`Watermark/Cursor Continuity`):**
   - `pageSegments()` (`models/trans-his-pg/trans-his-pg.query.ts:571-590`) tách rõ 2 phân đoạn `(${sort.expr}) IS NULL` và `(${sort.expr}) IS NOT NULL`, kết hợp khóa phụ `core.id` cùng chiều sắp xếp (`ASC` hoặc `DESC`).
   - Nhờ đó, ngay cả khi cột sắp xếp (như `sender.fullName`, `info.fee`, `createdAt`) có nhiều dòng trùng giá trị hoặc mang giá trị `NULL`, con trỏ `PageCursor = { value: last.sortKey, id: last._id }` (`trans-his-pg.model.ts:94`) vẫn đảm bảo duyệt qua toàn bộ tập kết quả mà không bị lặp dòng hay sót dòng ở ranh giới chuyển từ `NOT NULL` sang `NULL`.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật (EV-...)

Toàn bộ lệnh kiểm tra biên dịch, lint và test đều được thực thi trực tiếp qua `evidence_run.py` trên nhánh `feat/trans-his-postgres-export` (`HEAD = 017c0fe`):

1. **Biên dịch Production TypeScript (`tsconfig.build.json`):**
   - Biên lai trước sửa: **`EV-20261103T105759-fb0c`** (`exit=0`)
   - Biên lai sau sửa cuối: **`EV-20261103T113403-b34e`** (`exit=0`)
   - Lệnh: `npx tsc -p tsconfig.build.json --noEmit`
   - Kết quả: **PASS tuyệt đối (0 lỗi biên dịch)** trên toàn bộ source code production.
2. **Biên dịch toàn bộ repo gồm cả `test/` (`tsconfig.json`):**
   - Biên lai: **`EV-20261103T105700-86a1`** (`exit=2`)
   - Kết quả: **0 lỗi** trên toàn bộ các file mới/sửa của `trans-his-pg`; 104 lỗi đều nằm ở 21 file test/config cũ tồn tại từ trước trên `develop`.
3. **Kiểm tra ESLint (`npm run lint` & kiểm tra cô lập các file chạm tới):**
   - Biên lai trước sửa: **`EV-20261103T105725-23d3`** (`npx eslint start.ts`, `exit=1` — 2 lỗi), **`EV-20261103T105729-22f1`** (`exit=0` trên 17/18 file source).
   - Biên lai sau sửa cuối: **`EV-20261103T113358-97a0`** (`exit=0` trên toàn bộ **23/23 file source** gồm cả `start.ts`, `models/trans-his-pg/*.ts`, `utils/trans-his-export-source.ts`, `dbHandler/trans-his-pg.ts`, `logics/export/merchant/payout-request-export.pure.ts`, `logics/processors/BaseExportProcessor.ts`).
   - Kết quả: **0 lỗi, 0 cảnh báo ESLint** trên 100% file source chạm tới.
4. **Kiểm tra Unit Tests (`vitest`):**
   - Biên lai trước sửa: **`EV-20261103T105710-0538`** (`exit=0`, `110/110` tests passed).
   - Biên lai sau sửa cuối: **`EV-20261103T113336-1859`** (`exit=0`, **`7/7` test files, `114/114` tests passed** — đã bao gồm 4 unit test mới kiểm chứng `F2`, `F3`, `F4`, `F5`).
5. **Kiểm tra Integration Test với PostgreSQL thật (`vitest.integration.config.ts`):**
   - Biên lai sau sửa cuối: **`EV-20261103T113239-b095`** (`exit=0`, `13 skipped`, in rõ `[trans-his-pg.integration.test.ts] Skipping live PostgreSQL integration suite because TRANS_HIS_PG_IT_URL is not set.`)
   - **Báo cáo trung thực (`Rule #9.8c` & `L007`):** Trên máy local hiện tại chưa có biến môi trường `TRANS_HIS_PG_IT_URL` trỏ tới database PostgreSQL chứa bảng `public.transaction_history` và `public.transaction_history_payload`, nên 13 bài test trong `test/integration/trans-his-pg.integration.test.ts` được skip kèm cảnh báo rõ ràng. Khi có endpoint PostgreSQL test (`TRANS_HIS_PG_IT_URL`), có thể chạy ngay lệnh trên và `test/tools/trans-his-parity.ts` để đối soát dữ liệu thực tế.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty Check)

- Toàn bộ nhận định trong báo cáo đều gắn kèm bằng chứng vật lý `file:line`, `wc -l` thực tế và biên lai `EV-...` (`EV-20261103T113403-b34e`, `EV-20261103T113358-97a0`, `EV-20261103T113336-1859`, `EV-20261103T113239-b095`). Không suy diễn từ unit test in-memory sang runtime PostgreSQL thật, và đã báo cáo trung thực việc môi trường local chưa có `TRANS_HIS_PG_IT_URL`.

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons Check)

Sau khi nhận `APPROVE` từ User, toàn bộ 6 hạng mục `S-FIX-F1` $\rightarrow$ `S-FIX-F6` đã được thực thi và kiểm định khép kín:
1. Ghi nhận git restore-point tại `017c0fe33a7cc3351f36eb469e430dbac8886b2f`.
2. Tách `models/trans-his-pg/trans-his-pg.regex.ts` (`80` dòng), `models/trans-his-pg/trans-his-pg.sort.ts` (`211` dòng), `models/trans-his-pg/trans-his-pg.numeric.ts` (`181` dòng), giảm `trans-his-pg.query.ts` từ `590` xuống `340` dòng và `trans-his-pg.mapper.ts` từ `390` xuống `226` dòng (`F1`).
3. Hợp nhất cấu hình Postgres về `utils/trans-his-export-source.ts` (`86` dòng), giữ `dbHandler/trans-his-pg.ts` (`11` dòng) làm barrel re-export tương thích ngược, và truyền `config` vào `readPostgresDbConfig(config)` tại `start.ts:283` (`F2`).
4. Cập nhật `isDbEnabled("trans-his")` để tắt kết nối thừa tới Mongo `trans-his` khi chạy `TRANS_HIS_EXPORT_SOURCE=postgres` và không cấu hình `TRANS_HIS_MONGO_URI` (`F3`).
5. Lọc `.filter(Boolean)` trong `payout-request-export.pure.ts:56`, short-circuit `listTransId` rỗng trong `GetListTransHisByTransIdPgHandler.ts:15-17`, và rethrow lỗi sub-query tại `BaseExportProcessor.ts:238` (`F4`).
6. Cache `hasIdempotencyKeyPromise` trong `TransHisPgModel.selectSql()` (`F5`).
7. Sửa dứt điểm 2 lỗi ESLint tại `start.ts` và bổ sung `console.warn` khi thiếu `TRANS_HIS_PG_IT_URL` trong `trans-his-pg.integration.test.ts` (`F6`).

---

## Mục 6: Kết quả QC & Nhận xét từ /laya_router (DG3) + Tổng hợp 8 Quality Gates (G1–G8)

- **DG0 (`Pre-Plan Critique`):** `PASS` — Tệp `06_laya_summary_dg0_20261103T110447.json` (`C-REQ`, `C-REALITY`, `C-CONTRA` đều `ACCEPT`, vượt qua đầy đủ đối chứng âm/dương `CTRL-TRUE-1`, `CTRL-FALSE-1`, `CTRL-FALSE-2`).
- **DG1 + DG2 (`Solution & Code Demo Verification`):** `PASS` — Tệp `06_laya_summary_dg1_20261103T110837.json` & `06_laya_summary_dg2_20261103T110837.json` (`S-RC`, `S-FIX`, `S-IMPACT`, `S-PATTERN`, `D-BEHAV`, `D-EDGE`, `D-API` đều `ACCEPT`).
- **DG3 (`Post-Edit Final Audit Verification`):** `PASS` — Tệp `06_laya_summary_dg3_20261103T113449.json` (`A-F1-SRP-SPLIT` `ACCEPT`, `A-F2-F3-CONFIG` & `A-F5-STAMPEDE` đã xác minh và phân giải tường minh vào `04_decisions_trans_his_postgres_export.md`).

---

## 6. (f) Kiểm Định 8 Cổng Chất Lượng (`Gates G1–G8`) & Bảo Mật (`/security-agent`)

- **G1 (Truy vết yêu cầu):** Đã đối chiếu 100% với yêu cầu Audit của User tại `01_requirements_trans_his_postgres_export.md`.
- **G2 (Reproduce / Bằng chứng lỗi):** Đã chứng minh bằng `file:line` và biên lai `EV-20261103T105725-23d3`, `EV-20261103T105803-0a6d`.
- **G3 (Build, Lint & Unit Test):** Đã lưu 8 biên lai `EV-...` thực tế trong `06_evidence/`.
- **G4 (Edge-case & Concurrency):** Đã phản biện sâu `hasIdempotencyKey` concurrency stampede (`F5`), sub-query error swallowing (`F4`), và keyset pagination (`pageSegments`).
- **G5 (Regression & Backward Compatibility):** Đã kiểm chứng toàn bộ luồng mặc định `TRANS_HIS_EXPORT_SOURCE=mongo` và chỉ ra điểm cần hoàn thiện khi cắt hẳn Mongo (`F3`).
- **G6 (Output Correctness):** Đã đối soát logic ánh xạ `toHistoryProjection`, EJSON numeric decoding và `countDocuments` join `transaction_history_payload`.
- **G7 (Adversarial Line-by-Line Audit & Security Gate):**
  - Quét bảo mật (`/security-agent`) trên toàn bộ diff `2eb3f76..017c0fe`:
    - Không có secret/mật khẩu hardcode trong code hay test.
    - Thông báo lỗi thiếu biến môi trường trong `readTransHisPgConfig()` (`dbHandler/trans-his-pg.ts:27-29`) chỉ in tên biến (`TRANS_HIS_PG_*`), tuyệt đối không in giá trị password.
    - Toàn bộ câu lệnh SQL trong `trans-his-pg.query.ts` và `trans-his-pg.model.ts` đều dùng parameterized queries (`$1, $2...`) và chỉ chạy `SELECT`.
- **G8 (Bằng chứng vật lý & Governance):** Đã lưu đầy đủ hồ sơ tại workspace `audit-trans-his-postgres-export-2026-11-03`.
