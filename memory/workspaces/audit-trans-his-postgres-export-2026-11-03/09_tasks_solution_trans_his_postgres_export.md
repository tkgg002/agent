---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T11:06:30+07:00
---

# 09. Giải pháp Kỹ thuật Duy nhất (Single-Best Solution) — Nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

## 1. Bảng Ánh xạ 1-1 Phát hiện (`F1..F6`) $\leftrightarrow$ Giải pháp (`S-FIX-F1..F6`)

| Mã phát hiện | Mức độ | Tệp và dòng (`file:line`) tại HEAD `017c0fe` | Tóm tắt vấn đề (Root Cause) | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :---: | :--- | :--- | :--- |
| **`F1`** | **HIGH** | `models/trans-his-pg/trans-his-pg.query.ts:1-590` (`590` dòng)<br>`models/trans-his-pg/trans-his-pg.mapper.ts:1-390` (`390` dòng) | **Vi phạm Trách nhiệm đơn (SRP) & Ngân sách dòng (`L016`):**<br>1. `trans-his-pg.query.ts` (`590` dòng > ngưỡng 500 dòng) nhồi nhét 4 trách nhiệm riêng biệt vào 1 file: bộ dịch regex `approvedRegex` (dòng `67-158`), bộ dịch filter SQL `compileFilter` (dòng `159-419`), bộ chuẩn hóa `normalizeSelect` (dòng `420-446`), và bộ phân giải `resolveSort` + keyset cursor `pageSegments` (dòng `447-590`).<br>2. `trans-his-pg.mapper.ts` (`390` dòng) gộp chung 2 module của `v2` (`numeric-value.ts` dòng `48-220` và `transaction-history.response.ts` dòng `221-390`) vào 1 file. | **`S-FIX-F1`:** Tách theo đúng trách nhiệm đơn (Single-Responsibility):<br>- Tách `models/trans-his-pg/trans-his-pg.regex.ts` (`approvedRegex`, `coerceMongooseBoolean`).<br>- Tách `models/trans-his-pg/trans-his-pg.sort.ts` (`SORT_EXPRESSIONS`, `resolveSort`, `encodeCursor`, `decodeCursor`, `pageSegments`, `normalizeSelect`).<br>- Giữ `models/trans-his-pg/trans-his-pg.query.ts` (`< 310` dòng) chuyên trách `compileFilter` và re-export các hàm từ `.regex` / `.sort` để giữ tương thích ngược 100% với mọi caller và unit test.<br>- Tách `models/trans-his-pg/trans-his-pg.numeric.ts` (`decodeCanonicalValue`, `canonicalValueType`, `validateNumericValue`, `toNativeNumericValue`) khỏi `trans-his-pg.mapper.ts` (`< 200` dòng) và re-export tại `trans-his-pg.mapper.ts`. |
| **`F2`** | **HIGH** | `dbHandler/trans-his-pg.ts:1-57`<br>`utils/trans-his-export-source.ts:1-15`<br>`start.ts:248-251, 277-290`<br>`logics/health-check.logic.ts:39-41, 60-70` | **Đặt sai Layer theo Design Pattern gốc & Hardcode `trans-his-pg` trong vòng lặp DB chung (`L014`, `L016`):**<br>1. Tầng `dbHandler/` chỉ dành cho các adapter kết nối DB hạ tầng (`DbConnectorFactory.ts`, `mongo.ts`, `mariadb.ts`, `postgres.ts`). Sau commit `017c0fe`, `dbHandler/trans-his-pg.ts` không còn tạo pool nữa mà chỉ chứa hằng số domain và đọc env `SVC_ENV`/`DB_LIST`, bị chia cắt nửa vời với `utils/trans-his-export-source.ts`.<br>2. Trong `start.ts:277-290`, nhánh `else if (config.type === "postgres")` nằm trong vòng lặp `for (const config of enabledDbs)` nhưng lại phớt lờ biến `config` của vòng lặp và gọi cứng `readTransHisPgConfig()` (khác hẳn `mongo` và `mariadb` đọc động từ `config.*EnvKey`). | **`S-FIX-F2`:**<br>- Gom trọn trách nhiệm đọc cấu hình nguồn export và `readPostgresDbConfig(config, env)` (kèm hàm wrapper `readTransHisPgConfig(env)` cho `trans-his-pg`) về `utils/trans-his-export-source.ts`; giữ `dbHandler/trans-his-pg.ts` làm barrel re-export mỏng để tương thích ngược 100% với các test hiện hữu.<br>- Tại `start.ts:277-290`, truyền biến `config` của vòng lặp vào `readPostgresDbConfig(config)` để xử lý tổng quát cho mọi DB `type === "postgres"` trong `DB_LIST`. |
| **`F3`** | **HIGH** | `dbHandler/trans-his-pg.ts:11-14`<br>`svc-env.ts:129-130`<br>`start.ts:242, 329-332` | **Khớp nối cứng biến môi trường & kết nối Mongo `trans-his` khi chạy `TRANS_HIS_EXPORT_SOURCE=postgres`:**<br>Khi `TRANS_HIS_EXPORT_SOURCE=postgres`, 100% truy vấn `trans-his` đã chuyển sang `TransHisPgModel`, nhưng `svc-env.ts:129-130` vẫn bắt buộc phải có `TRANS_HIS_MONGO_URI` & `TRANS_HIS_MONGO_POOLSIZE` (nếu thiếu sẽ văng `MIS_ENV_KEY`), và `isDbEnabled("trans-his")` vẫn luôn trả về `true` khiến `start.ts` mở kết nối thừa tới cụm Mongo `trans-his` và `HealthCheckLogic` báo `DISCONNECTED` khi cụm Mongo cũ ngắt. | **`S-FIX-F3`:**<br>- Trong `isDbEnabled(alias)`: khi `alias === "trans-his"` và `getTransHisExportSource() === "postgres"`, chỉ bật kết nối Mongo `trans-his` nếu `SVC_ENV.get().TRANS_HIS_MONGO_URI` thực sự được cấu hình có giá trị (giữ tương thích với bài test `trans-his-pg.connection.test.ts` vốn set `TRANS_HIS_MONGO_URI`, đồng thời cho phép gỡ bỏ kết nối Mongo `trans-his` trên môi trường chỉ chạy Postgres). |
| **`F4`** | **MEDIUM** | `logics/processors/BaseExportProcessor.ts:219-239`<br>`logics/export/merchant/payout-request-export.pure.ts:54-58`<br>`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-31` | **Nuốt lỗi Sub-Query ở `BaseExportProcessor` & Truy vấn thừa `[undefined]` ở `PayoutRequestExport` (`L012`, `L028`):**<br>1. `payout-request-export.pure.ts:56` lấy `primaryData.map(c => c.transInfo?.transId)` không lọc `undefined`. Khi cả trang không có `transId`, `GetListTransHisByTransIdPgHandler` vẫn gọi `TransHisPgModel.find({ transId: { $in: [undefined] } })`, bắn truy vấn `information_schema.columns` và `SELECT ... WHERE FALSE` vô ích xuống Postgres.<br>2. `BaseExportProcessor.ts:236-238` bắt `catch (err)` của sub-query và chỉ `logger.error` mà không throw, làm `PayoutRequestExport` hoàn tất `COMPLETED` giả tạo với cột `"Trạng thái giao dịch rút tiền"` bị trống trơn khi Postgres timeout/lỗi. | **`S-FIX-F4`:**<br>- Trong `GetListTransHisByTransIdPgHandler.ts:14-27` (và `payout-request-export.pure.ts:55-58`): lọc danh sách `transId` hợp lệ (`id !== undefined && id !== null && String(id).trim() !== ""`); nếu mảng rỗng thì trả về ngay `{ data: [] }` không gọi DB.<br>- Trong `BaseExportProcessor.ts:236-238`: rethrow lỗi (`throw err;`) sau khi ghi log để job dừng lại đúng chuẩn thay vì xuất file thiếu dữ liệu ngầm. |
| **`F5`** | **MEDIUM** | `models/trans-his-pg/trans-his-pg.model.ts:60, 121-126` | **Concurrency Stampede khi kiểm tra `hasIdempotencyKey` trong `TransHisPgModel.selectSql()`:**<br>`selectSql()` kiểm tra `if (this.hasIdempotencyKey === undefined)` rồi `await this.run(HAS_IDEMPOTENCY_KEY_SQL, [], 3000)` mà không cache Promise đang bay (`in-flight promise`). Khi nhiều worker (`EXPORT_WORKER_CONCURRENCY > 1`) hoặc nhiều truy vấn chạy đồng thời lúc khởi động, tất cả đều bắn trùng lặp câu query `information_schema.columns` vào pool. | **`S-FIX-F5`:**<br>- Lưu `private hasIdempotencyKeyPromise?: Promise<boolean>;` trên `TransHisPgModel` để hợp nhất mọi lời gọi `selectSql()` đồng thời vào đúng 1 truy vấn `HAS_IDEMPOTENCY_KEY_SQL` duy nhất (và reset `this.hasIdempotencyKeyPromise = undefined` nếu truy vấn bị lỗi để lần gọi kế tiếp có thể retry). |
| **`F6`** | **LOW** | `start.ts:4, 218`<br>`test/integration/trans-his-pg.integration.test.ts:18-22`<br>`git log develop..HEAD` | **Lỗi ESLint trên file chạm tới `start.ts`, Silent Skip ở Integration Test (`L007`) & Lệch nhánh gốc (`L001`):**<br>1. `npx eslint start.ts` báo 2 lỗi (`4:1` `import/order` và `218:38` `arrow-parens`).<br>2. `trans-his-pg.integration.test.ts:22` gọi `describe.skipIf(!pgUrl)` im lặng mà không in cảnh báo rõ biến môi trường `TRANS_HIS_PG_IT_URL` đang thiếu.<br>3. Nhánh `feat/trans-his-postgres-export` tách từ `2eb3f76` (`main`) thay vì `develop` (`9a6cc43`), thiếu commit nâng cấp bảo mật `Dockerfile` (`a181ef1`). | **`S-FIX-F6`:**<br>- Sửa dứt điểm 2 lỗi ESLint tại `start.ts:4` và `start.ts:218`.<br>- Bổ sung `console.warn` minh bạch trong `test/integration/trans-his-pg.integration.test.ts` khi thiếu `TRANS_HIS_PG_IT_URL` (`L007`).<br>- Rebase/merge `develop` vào `feat/trans-his-postgres-export` khi chuẩn bị合nhánh để đồng bộ `Dockerfile`. |

---

## 2. Bảng Truy vết Điểm Gọi Runtime (Explicit Runtime Call-Site Wiring — `L012`)

| Luồng thực thi (Runtime Flow) | Điểm xuất phát (Entry Call-Site) | Chuỗi uỷ quyền (Callchain) | Điểm thực thi đích (Target Wiring) |
| :--- | :--- | :--- | :--- |
| **1. Khởi tạo Pool PostgreSQL & Kiểm tra Env lúc boot** | `start.ts:325-334` (`loadEnvAndInitService`) $\rightarrow$ `start.ts:228-323` (`initService`) | `getTransHisExportSource()` (`utils/trans-his-export-source.ts:6-12`) $\rightarrow$ `isDbEnabled(db.alias)` (`dbHandler/trans-his-pg.ts:12-13`) $\rightarrow$ `readTransHisPgConfig()` (`dbHandler/trans-his-pg.ts:22-57`) $\rightarrow$ `PostgresDb.createConnection()` (`dbHandler/postgres.ts:18-56`) $\rightarrow$ `DbConnectorFactory.createConnector()` (`dbHandler/DbConnectorFactory.ts:144-169`) | Tạo `pg.Pool` (`max: poolSize`, `connectionTimeoutMillis: 10000`, `idleTimeoutMillis: 30000`) và truyền vào `new MainProcess(..., connections, [paginate])` (`start.ts:313`). |
| **2. Đăng ký Model & CQRS Query Handlers** | `data-transfers/main-process.ts:37-111` (`initializeModels`) & `start.ts:316` (`queryHandlerFactory.registerHandlers`) | `ModelRegistry["trans-his-pg"]` (`models/index.ts:116`) $\rightarrow$ `new ModelClass(connection, this.logger)` (`data-transfers/main-process.ts:66`) $\rightarrow$ `QueryHandlerFactory.registerHandlers()` (`domain/QueryHandlerFactory.ts:52-55, 81-83`) | Khi `TRANS_HIS_EXPORT_SOURCE=postgres`: gắn `GetAllTransHisExportQuery` $\rightarrow$ `GetAllTransHisExportPgHandler` và `GetListTransHisByTransIdQuery` $\rightarrow$ `GetListTransHisByTransIdPgHandler`. |
| **3. Thực thi 3 luồng Export lịch sử giao dịch (`TransHisExport`, `TransHisCollectionExport`, `InternalTransferExport`)** | `logics/export/trans-his/trans-his.export.ts:20`<br>`logics/export/trans-his/trans-his-collection.export.ts:20`<br>`logics/export/trans-his/internal-transfer.export.ts:21` | `transHisModelNames()` (`utils/trans-his-export-source.ts:15`) $\rightarrow$ `BaseExportProcessor.execute()` (`logics/processors/BaseExportProcessor.ts:166-215`) $\rightarrow$ `GetAllTransHisExportPgHandler.execute()` (`domain/handlers/trans-his/GetAllTransHisExportPgHandler.ts:13-43`) | Chạy song song `transHisModel.getPage(filter, sort, select, cursor, limit)` và (ở trang đầu `cursor === null`) `transHisModel.countDocuments(filter)` (`models/trans-his-pg/trans-his-pg.model.ts:71-110`). |
| **4. Thực thi Sub-Query lấy trạng thái giao dịch rút tiền trong `PayoutRequestExport`** | `logics/export/merchant/payout-request.export.ts:16` (`getRequiredModels`) & `logics/export/merchant/payout-request-export.pure.ts:54-70` (`getConfig`) | `BaseExportProcessor.execute()` (`logics/processors/BaseExportProcessor.ts:219-239`) $\rightarrow$ `QueryBus.execute(new GetListTransHisByTransIdQuery(subQueryParams))` $\rightarrow$ `GetListTransHisByTransIdPgHandler.execute()` (`domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-31`) | Gọi `TransHisPgModel.find({ transId: { $in: listTransId } }, { _id: 1, transId: 1, status: 1 })` (`models/trans-his-pg/trans-his-pg.model.ts:113-119`) và merge `cashOutTransStatus` vào từng dòng `payoutRequests`. |
| **5. Kiểm tra sức khỏe kết nối (`healthCheckDetailed`)** | `start.ts:101-104` (`handleHealthCheckDetailed`) | `HealthCheckLogic.checkDetailed()` (`logics/health-check.logic.ts:18-58`) $\rightarrow$ `HealthCheckLogic.pingTransHisPg()` (`logics/health-check.logic.ts:60-70`) | Gọi `this.mainProcess.models?.TransHisPgModel.ping()` (`SELECT 1` với timeout `3000ms`) đua qua `Promise.race` `PING_TIMEOUT_MS = 3000`. |

---

## 3. Code Demo Nguyên văn từ HEAD `017c0fe` (`L004`)

### 3.1. Code Demo cho `F1` (Tách SRP cho `models/trans-his-pg/trans-his-pg.query.ts` & `models/trans-his-pg/trans-his-pg.mapper.ts`)

- **Trích dẫn nguyên văn `models/trans-his-pg/trans-his-pg.query.ts:83-98` và `447-530` tại HEAD:**
```ts
// File: models/trans-his-pg/trans-his-pg.query.ts (Lines 83-98)
/** Mongoose's lenient Boolean cast (SchemaBoolean): true/"true"/1/"1"/"yes" -> true; false/"false"/0/"0"/"no" -> false. */
export function coerceMongooseBoolean(value: unknown): boolean | undefined {
	if (value === true || value === "true" || value === 1 || value === "1" || value === "yes") { return true; }
	if (value === false || value === "false" || value === 0 || value === "0" || value === "no") { return false; }
	return undefined;
}

/** A JavaScript regex (ASCII only, no lookaround or flags) as the PostgreSQL ARE that matches the same text; throws for what it cannot translate. */
export function approvedRegex(value: string): string {
	if (/[^\x09-\x0d\x20-\x7e]/.test(value)) { throw new Error("Invalid transaction history regex"); }
	try {
		new RegExp(value, "i");
	} catch {
		throw new Error("Invalid transaction history regex");
	}
```
```ts
// File: models/trans-his-pg/trans-his-pg.query.ts (Lines 503-530)
export interface SortSpec {
	readonly expr: string;
	readonly type: SortType;
	readonly direction: "ASC" | "DESC";
}

const DEFAULT_SORT: SortSpec = { expr: "core.created_at", type: "timestamptz", direction: "DESC" };

/**
 * The one key of a Mongo sort object, `{key: 1|-1}`. Absent / empty is createdAt newest first. A name no stored document has sorts
 * nothing (every document is equal on it), so it also gives the default order. `id` always breaks the ties, in the same direction.
 */
export function resolveSort(sort: unknown): SortSpec {
	if (sort === undefined || sort === null) { return DEFAULT_SORT; }
	if (!isPlainObject(sort)) { return refuse("unsupported_sort", "sort (an object is expected)"); }
	const keys = Object.keys(sort);
	if (keys.length === 0) { return DEFAULT_SORT; }
	if (keys.length > 1) { return refuse("unsupported_sort", "sort (one key is expected)"); }
	const key = keys[0];
	if (sort[key] !== 1 && sort[key] !== -1) { return refuse("unsupported_sort", `sort direction of "${shown(key)}" (1 or -1 is expected)`); }
	if (Object.prototype.hasOwnProperty.call(SORT_EXPRESSIONS, key)) {
		return { ...SORT_EXPRESSIONS[key], direction: sort[key] === 1 ? "ASC" : "DESC" };
	}
	if (STORED_FIELDS.includes(key.split(".")[0])) { return refuse("unsupported_sort", `sort key "${shown(key)}"`); }
	// Mongo: every document lacks the key, so all are equal and come in storage order; the closest stable order is the default one
	return DEFAULT_SORT;
}
```

- **Bản sửa đề xuất (`S-FIX-F1`):**
  1. Tách dòng `83-157` (`coerceMongooseBoolean`, `approvedRegex`) sang `models/trans-his-pg/trans-his-pg.regex.ts`.
  2. Tách dòng `420-590` (`STORED_FIELDS`, `SELECTABLE_ABSENT_FIELDS`, `normalizeSelect`, `SORT_EXPRESSIONS`, `SortSpec`, `resolveSort`, `PageCursor`, `encodeCursor`, `decodeCursor`, `PageSegment`, `pageSegments`) sang `models/trans-his-pg/trans-his-pg.sort.ts`.
  3. Trong `models/trans-his-pg/trans-his-pg.query.ts`, import và re-export lại các hàm trên để giữ nguyên 100% API public:
```ts
// Proposed in models/trans-his-pg/trans-his-pg.query.ts
import { approvedRegex, coerceMongooseBoolean } from "./trans-his-pg.regex";
import {
	decodeCursor,
	encodeCursor,
	normalizeSelect,
	 PageCursor,
	PageSegment,
	pageSegments,
	resolveSort,
	SortSpec,
	STORED_FIELDS
} from "./trans-his-pg.sort";

export {
	approvedRegex,
	coerceMongooseBoolean,
	decodeCursor,
	encodeCursor,
	normalizeSelect,
	PageCursor,
	PageSegment,
	pageSegments,
	resolveSort,
	SortSpec
};
```
  4. Tương tự, tách dòng `48-220` của `models/trans-his-pg/trans-his-pg.mapper.ts` (`decodeCanonicalValue`, `canonicalValueType`, `validateNumericValue`, `toNativeNumericValue`) sang `models/trans-his-pg/trans-his-pg.numeric.ts` và re-export tại `models/trans-his-pg/trans-his-pg.mapper.ts`.

---

### 3.2. Code Demo cho `F2`, `F3` & `F6` (Chuẩn hóa Layer cấu hình Postgres, gỡ khớp nối cứng Mongo `trans-his` & sửa ESLint tại `start.ts`)

- **Trích dẫn nguyên văn `dbHandler/trans-his-pg.ts:11-30` tại HEAD:**
```ts
// File: dbHandler/trans-his-pg.ts (Lines 11-30)
/** Every database of DB_LIST is connected, except the trans-his Postgres when the exports read Mongo. */
export const isDbEnabled = (alias: string): boolean =>
	alias !== TRANS_HIS_PG_ALIAS || getTransHisExportSource() === "postgres";

/**
 * The connection settings of the trans-his Postgres (TRANS_HIS_PG_*). Host, port, database, user and password must all be set: there
 * is no default, so a missing port cannot fall back to 5432 (the writable primary). Each export job holds two pooled clients at once
 * (the page and the count run together), so the pool must have at least 2 x EXPORT_WORKER_CONCURRENCY clients, otherwise a job
 * waits for a free client until the connection timeout. Errors name the keys, never their values. The pool itself is created by
 * PostgresDb (dbHandler/postgres.ts) from this validated config, as MariaDb is for the mariadb entries.
 */
export function readTransHisPgConfig(env: Record<string, any> = SVC_ENV.get()): DbConfig {
	const entry: any = DB_LIST.find(db => db.alias === TRANS_HIS_PG_ALIAS);
	const keys = [entry.hostEnvKey, entry.portEnvKey, entry.dbEnvKey, entry.userEnvKey, entry.passEnvKey] as string[];
	const missing = keys.filter(key => env[key] === undefined || env[key] === null || String(key && env[key]).trim() === "");
	if (missing.length > 0) {
		throw new Error(
			`TRANS_HIS_EXPORT_SOURCE=postgres requires ${missing.join(", ")} to be set (no defaults, so the read-only endpoint is always explicit)`
		);
	}
```

- **Trích dẫn nguyên văn `start.ts:1-10`, `217-221`, và `248-291` tại HEAD:**
```ts
// File: start.ts (Lines 1-10)
// <DOCUMENT filename="start.ts">
import "./tracing";
import { Service, ServiceBroker, Context } from "moleculer";
import * as http from "http";
import paginate from "mongoose-paginate-v2";
import MongoDb from "./dbHandler/mongo";
import MariaDb from "./dbHandler/mariadb";
import PostgresDb from "./dbHandler/postgres";
import { isDbEnabled, readTransHisPgConfig, TRANS_HIS_PG_ALIAS } from "./dbHandler/trans-his-pg";
import { trackingIn, trackingOut } from "./middlewares";
```
```ts
// File: start.ts (Lines 217-221)
        if (this.healthServer) {
            await new Promise<void>((resolve) => {
                this.healthServer.close(() => resolve());
            });
        }
```
```ts
// File: start.ts (Lines 248-291)
            const transHisExportSource = getTransHisExportSource();
            // checked before the loop, whose catch only logs: a missing key or a small pool stops the service with its own message
            if (transHisExportSource === "postgres") { readTransHisPgConfig(); }
            for (const config of enabledDbs) {
                const alias = config.alias;
                try {
                    let connection: any;
                    if (config.type === "mongo") {
                        const uri = SVC_ENV.get()[config.uriEnvKey];
                        if (!uri) { throw new Error(`Missing URI for ${alias}`); }
                        connection = await new Promise((resolve, reject) => {
                            this.dbHandler.createConnection({
                                alias,
                                uri,
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 5
                            }, (err, conn) => err ? reject(err) : resolve(conn));
                        });
                    } else if (config.type === "mariadb") {
                        connection = await new Promise((resolve, reject) => {
                            this.mariaDbHandler.createConnection({
                                alias,
                                host: SVC_ENV.get()[config.hostEnvKey],
                                port: SVC_ENV.get()[config.portEnvKey],
                                database: SVC_ENV.get()[config.dbEnvKey],
                                username: SVC_ENV.get()[config.userEnvKey],
                                password: SVC_ENV.get()[config.passEnvKey],
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 10
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    } else if (config.type === "postgres") {
                        const pg = readTransHisPgConfig();
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: pg.host!,
                                port: pg.port!,
                                database: pg.database!,
                                username: pg.username!,
                                password: pg.password!,
                                poolSize: pg.poolSize!
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
```

- **Bản sửa đề xuất (`S-FIX-F2`, `S-FIX-F3`, `S-FIX-F6`):**
  1. Chuyển `TRANS_HIS_PG_ALIAS`, `isDbEnabled`, `readPostgresDbConfig(entry, env)` và `readTransHisPgConfig(env)` về `utils/trans-his-export-source.ts` (và re-export tại `dbHandler/trans-his-pg.ts` để giữ nguyên tương thích ngược cho `test/unit/trans-his-pg/trans-his-pg.connection.test.ts`).
  2. Trong `isDbEnabled(alias)`: khi `alias === "trans-his"` và `getTransHisExportSource() === "postgres"`, kiểm tra nếu `SVC_ENV.get().TRANS_HIS_MONGO_URI` rỗng/không cấu hình thì trả về `false` (không ép kết nối Mongo `trans-his` thừa), ngược lại nếu có cấu hình thì vẫn giữ `true`.
  3. Trong `start.ts`: sửa thứ tự `import * as http from "http";` lên trước `"moleculer"`, bỏ ngoặc đơn `(resolve)` tại dòng `218`, và truyền `config` vào `readPostgresDbConfig(config)` ở dòng `278`:
```ts
// Proposed in start.ts (Lines 277-290)
                    } else if (config.type === "postgres") {
                        const pg = readPostgresDbConfig(config);
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: pg.host!,
                                port: pg.port!,
                                database: pg.database!,
                                username: pg.username!,
                                password: pg.password!,
                                poolSize: pg.poolSize!
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
```

---

### 3.3. Code Demo cho `F4` (Chặn truy vấn thừa `[undefined]` ở `GetListTransHisByTransIdPgHandler` & không nuốt lỗi Sub-Query ở `BaseExportProcessor`)

- **Trích dẫn nguyên văn `domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:13-31` tại HEAD:**
```ts
// File: domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts (Lines 13-31)
    public async execute(query: GetListTransHisByTransIdQuery, context: any, options: any): Promise<any> {
        const { listTransId } = query;

        const transHisModel = this.mainProcess.models.TransHisPgModel;

        const selectFields = {
            _id: 1,
            transId: 1,
            status: 1
        };

        const filter = { transId: { $in: listTransId } };

        const rawData = await transHisModel.find(filter, selectFields);

        return {
            data: rawData
        };
    }
```

- **Trích dẫn nguyên văn `logics/export/merchant/payout-request-export.pure.ts:54-58` và `logics/processors/BaseExportProcessor.ts:219-239` tại HEAD:**
```ts
// File: logics/export/merchant/payout-request-export.pure.ts (Lines 54-58)
        subQueryClass: GetListTransHisByTransIdQuery,
        buildSubQueryParams: (primaryData: any[]) => {
            const listTransId = [...new Set(primaryData.map(c => c.transInfo?.transId))];
            return listTransId;
        },
```
```ts
// File: logics/processors/BaseExportProcessor.ts (Lines 219-239)
          if (config.subQueryClass && config.buildSubQueryParams && config.mergeData && currentData.length > 0) {
            try {
              // 1. Build params for sub-query
              const subQueryParams = config.buildSubQueryParams(currentData);

              // 2. Initialize Query Instance
              const subQueryInstance = new config.subQueryClass(subQueryParams);

              // 3. Execute Sub-query
              const subResult = await this.queryBus.execute(subQueryInstance, this.context, {
                params: subQueryParams
              }, mainProcess);

              // 4. Merge data
              const subDataList = Array.isArray(subResult) ? subResult : (subResult.data || []);
              currentData = config.mergeData(currentData, subDataList);

            } catch (err) {
              this.context.broker.logger.error(`Error executing sub-query for job ${jobId}`, err);
            }
          }
```

- **Bản sửa đề xuất (`S-FIX-F4`):**
```ts
// Proposed in domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts (Lines 13-31)
    public async execute(query: GetListTransHisByTransIdQuery, context: any, options: any): Promise<any> {
        const listTransId = Array.isArray(query.listTransId)
            ? query.listTransId.filter(id => (typeof id === "string" && id.trim() !== "") || (typeof id === "number" && Number.isFinite(id)))
            : [];
        if (listTransId.length === 0) {
            return { data: [] };
        }

        const transHisModel = this.mainProcess.models.TransHisPgModel;

        const selectFields = {
            _id: 1,
            transId: 1,
            status: 1
        };

        const filter = { transId: { $in: listTransId } };

        const rawData = await transHisModel.find(filter, selectFields);

        return {
            data: rawData
        };
    }
```
```ts
// Proposed in logics/export/merchant/payout-request-export.pure.ts (Lines 54-58)
        subQueryClass: GetListTransHisByTransIdQuery,
        buildSubQueryParams: (primaryData: any[]) => {
            const listTransId = [...new Set(primaryData.map(c => c.transInfo?.transId).filter(Boolean))];
            return listTransId;
        },
```
```ts
// Proposed in logics/processors/BaseExportProcessor.ts (Lines 236-239)
            } catch (err) {
              this.context.broker.logger.error(`Error executing sub-query for job ${jobId}`, err);
              throw err;
            }
```

---

### 3.4. Code Demo cho `F5` (Chống Concurrency Stampede trong `TransHisPgModel.selectSql()`)

- **Trích dẫn nguyên văn `models/trans-his-pg/trans-his-pg.model.ts:57-65` và `121-126` tại HEAD:**
```ts
// File: models/trans-his-pg/trans-his-pg.model.ts (Lines 57-65)
class TransHisPgModel {
	private readonly logger: LoggerInstance;
	private readonly pool: Pool;
	private hasIdempotencyKey: boolean | undefined;

	public constructor(pool: Pool, logger: LoggerInstance) {
		this.pool = pool;
		this.logger = logger;
	}
```
```ts
// File: models/trans-his-pg/trans-his-pg.model.ts (Lines 121-126)
	private async selectSql(): Promise<string> {
		if (this.hasIdempotencyKey === undefined) {
			this.hasIdempotencyKey = (await this.run(HAS_IDEMPOTENCY_KEY_SQL, [], 3000)).rows.length > 0;
		}
		return this.hasIdempotencyKey ? SELECT_SQL.replace(REQUEST_ID_COLUMN, IDEMPOTENCY_KEY_COLUMN) : SELECT_SQL;
	}
```

- **Bản sửa đề xuất (`S-FIX-F5`):**
```ts
// Proposed in models/trans-his-pg/trans-his-pg.model.ts
class TransHisPgModel {
	private readonly logger: LoggerInstance;
	private readonly pool: Pool;
	private hasIdempotencyKey: boolean | undefined;
	private hasIdempotencyKeyPromise: Promise<boolean> | undefined;
```
```ts
	private async selectSql(): Promise<string> {
		if (this.hasIdempotencyKey === undefined) {
			if (!this.hasIdempotencyKeyPromise) {
				this.hasIdempotencyKeyPromise = this.run(HAS_IDEMPOTENCY_KEY_SQL, [], 3000)
					.then(res => {
						this.hasIdempotencyKey = res.rows.length > 0;
						return this.hasIdempotencyKey;
					})
					.catch(err => {
						this.hasIdempotencyKeyPromise = undefined;
						throw err;
					});
			}
			await this.hasIdempotencyKeyPromise;
		}
		return this.hasIdempotencyKey ? SELECT_SQL.replace(REQUEST_ID_COLUMN, IDEMPOTENCY_KEY_COLUMN) : SELECT_SQL;
	}
```
