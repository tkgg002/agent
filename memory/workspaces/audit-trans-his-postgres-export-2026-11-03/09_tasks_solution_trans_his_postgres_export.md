---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T11:06:30+07:00
---

# 09. Giải pháp Kỹ thuật Duy nhất (Single-Best Solution) — Nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

## 1. Bảng Ánh xạ 1-1 Phát hiện (`F1..F6`) $\leftrightarrow$ Giải pháp (`S-FIX-F1..F6`) theo Đúng Design Pattern Gốc (`1ea8d1f8` trên `develop`)

| Mã phát hiện | Mức độ | Tệp và dòng (`file:line`) tại HEAD hiện hành | Phản biện Gốc rễ (Adversarial Root-Cause Critique) | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :---: | :--- | :--- | :--- |
| **`F1`** | **CRITICAL** | `logics/health-check.logic.ts:1-71`<br>`data-transfers/main-process.ts:32-38` | **Phá vỡ Tính Đa hình của `MainProcess.isDbConnected(alias)` & Đặt sai Layer trong `HealthCheckLogic` (`L001`, `L014`, `L016`):**<br>1. Trên `develop` (`commit 1ea8d1f8`), `HealthCheckLogic` hoàn toàn độc lập với loại CSDL và tên domain: chỉ duyệt `DB_LIST` và gọi `this.mainProcess.isDbConnected(alias)`.<br>2. Nhánh `feat/trans-his-postgres-export` đã làm hỏng `HealthCheckLogic` khi nhét `import { isDbEnabled, TRANS_HIS_PG_ALIAS }`, lọc `if (!isDbEnabled(alias))`, rẽ nhánh cứng `alias === TRANS_HIS_PG_ALIAS ? await this.pingTransHisPg() : this.mainProcess.isDbConnected(alias)` và gọi vòng qua Model nghiệp vụ `this.mainProcess.models?.TransHisPgModel.ping()`. | **`S-FIX-F1`:**<br>1. **Khôi phục `logics/health-check.logic.ts` về nguyên bản 100% của nhánh `develop` (`0` dòng diff so với `develop`!)**: xóa toàn bộ `isDbEnabled`, `TRANS_HIS_PG_ALIAS`, `PING_TIMEOUT_MS` và `pingTransHisPg()`.<br>2. **Tại `data-transfers/main-process.ts:32-38` (`MainProcess.isDbConnected`):** bổ sung kiểm tra đa hình cho `pg.Pool`: `if (conn && "ended" in conn) { return !(conn as any).ended; }` bên cạnh `"readyState" in conn` (Mongoose) và `!!conn` (MariaDB). |
| **`F2`** | **CRITICAL** | `start.ts:24-31, 242-316`<br>`dbHandler/postgres.ts:1-61`<br>`dbHandler/trans-his-pg.ts:1-11` | **Rò rỉ Logic Nghiệp vụ `trans-his` vào Bootstrapper `start.ts` & Lệch chuẩn `PostgresDb` so với `MariaDb` (`L001`, `L014`, `L016`):**<br>1. Trên `develop` (`commit 1ea8d1f8`), `start.ts` duyệt `DB_LIST` và chỉ rẽ nhánh theo `config.type` (`"mongo"` \| `"mariadb"`), không hề biết tới tên domain (`reconcile`, `trans-his`).<br>2. Hiện tại `start.ts` đang import `getTransHisExportSource`, `isDbEnabled`, `readPostgresDbConfig`, `readTransHisPgConfig`, `TRANS_HIS_PG_ALIAS` và đặt các câu lệnh `if (transHisExportSource === "postgres")` ngay trong `initService()`.<br>3. Tầng `dbHandler/` chỉ chứa các driver wrapper (`DbConnectorFactory.ts`, `mongo.ts`, `mariadb.ts`, `postgres.ts`), nhưng lại bị nhét thêm file `dbHandler/trans-his-pg.ts` mang tên domain. | **`S-FIX-F2`:**<br>1. **Tại `start.ts`:** Xóa sạch mọi import và câu lệnh `if` liên quan đến `trans-his-export-source` / `TRANS_HIS_PG_ALIAS` / `isDbEnabled`. Giữ nguyên vòng lặp `for (const config of DB_LIST)` của `develop` và chỉ thêm nhánh `else if (config.type === "postgres")` gọi `this.postgresDbHandler.createConnection(...)` đối xứng 1:1 với `this.mariaDbHandler.createConnection(...)`.<br>2. **Tại `dbHandler/postgres.ts` (`PostgresDb`):** Đóng gói hàm `validatePostgresOptions(options)` ngay bên trong `dbHandler/postgres.ts` (kiểm tra `host`, `port`, `database`, `username`, `password`, `poolSize` và `poolSize >= 2 * concurrency`) hoàn toàn độc lập với tên domain.<br>3. **Xóa bỏ file đặt sai layer `dbHandler/trans-his-pg.ts`.** |
| **`F3`** | **HIGH** | `utils/trans-his-export-source.ts:1-86`<br>`defined/app-setting.ts:48-79`<br>`models/trans-his-pg/trans-his-pg.model.ts:49-51` | **Ôm đồm Trách nhiệm trong `utils/trans-his-export-source.ts` & Hardcode Tên Bảng SQL trong `TransHisPgModel` (`L016`, `L044`):**<br>1. `utils/trans-his-export-source.ts` (`86` dòng) đang gánh cả việc đọc `DB_LIST` và tạo `DbConfig` của tầng `dbHandler`.<br>2. Trong `models/reconcile/centralrecon.model.ts:11` (pattern chuẩn trên `develop`), tên bảng SQL lấy từ `DB_COLLECTION` (`defined/app-setting.ts`), nhưng `TransHisPgModel` lại hardcode `"public.transaction_history"` và `"public.transaction_history_payload"`. | **`S-FIX-F3`:**<br>1. Thu gọn `utils/trans-his-export-source.ts` về đúng `22` dòng đơn trách nhiệm: chỉ giữ `TransHisExportSource`, `getTransHisExportSource()` và `transHisModelNames()`.<br>2. Khai báo `TRANS_HIS_PG__TRANSACTION_HISTORY: "public.transaction_history"` và `TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD: "public.transaction_history_payload"` trong `DB_COLLECTION` (`defined/app-setting.ts`) và tham chiếu từ `TransHisPgModel`. |
| **`F4`** | **HIGH** | `models/trans-his-pg/trans-his-pg.query.ts` (`340` dòng)<br>`models/trans-his-pg/trans-his-pg.sort.ts` (`211` dòng)<br>`models/trans-his-pg/trans-his-pg.regex.ts` (`80` dòng)<br>`models/trans-his-pg/trans-his-pg.mapper.ts` (`226` dòng)<br>`models/trans-his-pg/trans-his-pg.numeric.ts` (`181` dòng) | **Kiểm chứng Tách Module Đơn Trách Nhiệm (SRP Split & Line-Budget Guard `< 500` dòng — `L016`):**<br>Trên commit `017c0fe`, `trans-his-pg.query.ts` dài `590` dòng và `trans-his-pg.mapper.ts` dài `390` dòng. Đã tách thành 5 file đơn trách nhiệm (`regex`, `sort`, `query`, `numeric`, `mapper`), giảm `query.ts` xuống `340` dòng và `mapper.ts` xuống `226` dòng, giữ re-export tương thích ngược 100%. | **`S-FIX-F4`:**<br>Giữ nguyên cấu trúc 5 file đơn trách nhiệm đã tách trong `models/trans-his-pg/` (mọi file đều `< 350` dòng, `NOW <= HEAD`). |
| **`F5`** | **MEDIUM** | `domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts:10-33`<br>`logics/export/merchant/payout-request-export.pure.ts:55-58`<br>`logics/processors/BaseExportProcessor.ts:235-239`<br>`models/trans-his-pg/trans-his-pg.model.ts:122-138` | **Chặn Sub-Query Rỗng, Chặn Silent Data Loss (`L055`) & Chống Concurrency Stampede (`G4`):**<br>1. Lọc `.filter(Boolean)` trong `payout-request-export.pure.ts:56` và short-circuit `{ data: [] }` khi `listTransId` rỗng trong `GetListTransHisByTransIdPgHandler`.<br>2. Rethrow `throw err;` trong `BaseExportProcessor.ts:238` khi sub-query lỗi.<br>3. Cache `this.hasIdempotencyKeyPromise` trong `TransHisPgModel.selectSql()` và reset về `undefined` khi lỗi mạng tạm thời. | **`S-FIX-F5`:**<br>Giữ nguyên các chốt bảo vệ runtime đã kiểm chứng tại `GetListTransHisByTransIdPgHandler.ts`, `payout-request-export.pure.ts`, `BaseExportProcessor.ts` và `TransHisPgModel.selectSql()`. |
| **`F6`** | **MEDIUM** | `test/unit/trans-his-pg/trans-his-pg.connection.test.ts:1-174`<br>`test/integration/trans-his-pg.integration.test.ts:22-25` | **Đồng bộ Unit Test theo Đúng Design Pattern Gốc (`PostgresDb`, `MainProcess.isDbConnected`, `HealthCheckLogic`):**<br>`trans-his-pg.connection.test.ts` đang import từ `dbHandler/trans-his-pg` (file đặt sai layer) và đang kỳ vọng `HealthCheckLogic` gọi `models.TransHisPgModel.ping()` thay vì gọi `mainProcess.isDbConnected("trans-his-pg")`. | **`S-FIX-F6`:**<br>Cập nhật `test/unit/trans-his-pg/trans-his-pg.connection.test.ts` để test trực tiếp:<br>1. `PostgresDb.createConnection` (`dbHandler/postgres.ts`) validate `host`, `port`, `database`, `username`, `password`, `poolSize >= 2 * EXPORT_WORKER_CONCURRENCY` và gọi `DbConnectorFactory`.<br>2. `MainProcess.isDbConnected("trans-his-pg")` kiểm tra `pg.Pool` (`ended: false` $\rightarrow$ `true`, `ended: true` $\rightarrow$ `false`).<br>3. `HealthCheckLogic.checkDetailed()` gọi `mainProcess.isDbConnected(alias)` đồng nhất cho mọi DB trong `DB_LIST`. |

---

## 2. Bảng Truy vết Điểm Gọi Runtime Chuẩn Kiến Trúc Gốc (Explicit Runtime Call-Site Wiring — `L012`)

| Luồng thực thi (Runtime Flow) | Điểm xuất phát (Entry Call-Site) | Chuỗi uỷ quyền (Callchain) | Điểm thực thi đích (Target Wiring) |
| :--- | :--- | :--- | :--- |
| **1. Khởi tạo kết nối Đa CSDL lúc boot (`mongo`, `mariadb`, `postgres`)** | `start.ts:323-336` (`loadEnvAndInitService`) $\rightarrow$ `start.ts:233-321` (`initService`) | `for (const config of DB_LIST)` $\rightarrow$ `else if (config.type === "postgres")` $\rightarrow$ `this.postgresDbHandler.createConnection(...)` (`dbHandler/postgres.ts:47-88`) $\rightarrow$ `DbConnectorFactory.getInstance(this.logger).connect(config, options.alias)` (`dbHandler/DbConnectorFactory.ts:56-122, 182-203`) | Tạo `pg.Pool` (`SELECT 1` lúc connect), lưu vào `connections[alias]` và truyền vào `new MainProcess(this.broker, this.logger, connections, [paginate])`. |
| **2. Kiểm tra sức khỏe kết nối (`healthCheckDetailed`)** | `start.ts:106-109` (`handleHealthCheckDetailed`) | `HealthCheckLogic.checkDetailed()` (`logics/health-check.logic.ts:18-52` — nguyên bản `develop`) $\rightarrow$ `this.mainProcess.isDbConnected(alias)` (`data-transfers/main-process.ts:32-41`) | Kiểm tra đa hình: Mongoose `readyState === 1`, Postgres `pg.Pool` `!conn.ended`, MariaDB `!!conn`. |
| **3. Đăng ký Model & CQRS Query Handlers** | `data-transfers/main-process.ts:43-92` (`initializeModels`) & `start.ts:314` (`queryHandlerFactory.registerHandlers`) | `ModelRegistry["trans-his-pg"]` (`models/index.ts:172-174`) $\rightarrow$ `new TransHisPgModel(connection, this.logger)` $\rightarrow$ `QueryHandlerFactory.registerHandlers()` (`domain/QueryHandlerFactory.ts:151-158`) | Khi `getTransHisExportSource() === "postgres"`: đăng ký `GetAllTransHisExportPgHandler` và `GetListTransHisByTransIdPgHandler`. |
| **4. Thực thi Export & Sub-Query** | `logics/export/trans-his/*.export.ts` & `logics/export/merchant/payout-request.export.ts` | `BaseExportProcessor` (`logics/processors/BaseExportProcessor.ts:57-89, 219-247`) $\rightarrow$ `QueryBus.execute(...)` | Truy vấn qua `TransHisPgModel.getPage()`, `countDocuments()`, `find()` trên các bảng khai báo trong `DB_COLLECTION`. |

---

## 3. Code Demo Nguyên văn từ HEAD Hiện hành (`L004`)

### 3.1. Code Demo cho `F1`: Khôi phục `logics/health-check.logic.ts` về 100% nguyên bản `develop` & Đa hình hóa `MainProcess.isDbConnected(alias)`

- **Trích dẫn nguyên văn `logics/health-check.logic.ts:1-71` tại HEAD hiện hành:**
```ts
// File: logics/health-check.logic.ts (Lines 1-71)
import { RedisCaching } from "../caching/redis";
import MainProcess from "../data-transfers/main-process";
import { DB_LIST } from "../defined/app-setting";
import { ModelRegistry } from "../models";
import { isDbEnabled, TRANS_HIS_PG_ALIAS } from "../utils/trans-his-export-source";

const PING_TIMEOUT_MS = 3000;

export class HealthCheckLogic {
    private redisInstance: RedisCaching;
    private mainProcess: MainProcess;

    public constructor(redisInstance: RedisCaching, mainProcess: MainProcess) {
        this.redisInstance = redisInstance;
        this.mainProcess = mainProcess;
    }

    public async checkDetailed(): Promise<any> {
        let redisPing = "NOT_INIT";
        try {
            if (this.redisInstance) {
                redisPing = await this.redisInstance.ping();
            }
        } catch (e) {
            redisPing = "ERROR";
        }

        const dbStatus: any = {};

        for (const db of DB_LIST) {
            const alias = db.alias;
            if (!isDbEnabled(alias)) { continue; }
            const models = ModelRegistry[alias]
                ? ModelRegistry[alias].map((m: any) => m.name || m.prototype?.constructor?.name)
                : [];

            try {
                // the trans-his Postgres is asked, not just looked up: a pool object exists even when the endpoint is gone
                const isConnected = alias === TRANS_HIS_PG_ALIAS
                    ? await this.pingTransHisPg()
                    : this.mainProcess.isDbConnected(alias);
                dbStatus[alias] = {
                    status: isConnected ? "CONNECTED" : "DISCONNECTED",
                    models
                };
            } catch (err) {
                dbStatus[alias] = {
                    status: "ERROR",
                    models
                };
            }
        }

        return {
            redis: redisPing,
            dbs: dbStatus
        };
    }

    private async pingTransHisPg(): Promise<boolean> {
        const model = this.mainProcess.models?.TransHisPgModel;
        if (!model) { return false; }
        let timer: NodeJS.Timeout;
        const timeout = new Promise<boolean>(resolve => { timer = setTimeout(() => resolve(false), PING_TIMEOUT_MS); });
        try {
            return await Promise.race([model.ping().then(() => true, () => false), timeout]);
        } finally {
            clearTimeout(timer);
        }
    }
}
```

- **Bản sửa đề xuất tại `logics/health-check.logic.ts` (Khôi phục 100% nguyên bản `develop` — `0` dòng diff với `develop`):**
```ts
// Proposed in logics/health-check.logic.ts (Lines 1-52)
import { RedisCaching } from "../caching/redis";
import MainProcess from "../data-transfers/main-process";
import { DB_LIST } from "../defined/app-setting";
import { ModelRegistry } from "../models";

export class HealthCheckLogic {
    private redisInstance: RedisCaching;
    private mainProcess: MainProcess;

    public constructor(redisInstance: RedisCaching, mainProcess: MainProcess) {
        this.redisInstance = redisInstance;
        this.mainProcess = mainProcess;
    }

    public async checkDetailed(): Promise<any> {
        let redisPing = "NOT_INIT";
        try {
            if (this.redisInstance) {
                redisPing = await this.redisInstance.ping();
            }
        } catch (e) {
            redisPing = "ERROR";
        }

        const dbStatus: any = {};

        for (const db of DB_LIST) {
            const alias = db.alias;
            const models = ModelRegistry[alias]
                ? ModelRegistry[alias].map((m: any) => m.name || m.prototype?.constructor?.name)
                : [];

            try {
                const isConnected = this.mainProcess.isDbConnected(alias);
                dbStatus[alias] = {
                    status: isConnected ? "CONNECTED" : "DISCONNECTED",
                    models
                };
            } catch (err) {
                dbStatus[alias] = {
                    status: "ERROR",
                    models
                };
            }
        }

        return {
            redis: redisPing,
            dbs: dbStatus
        };
    }
}
```

- **Trích dẫn nguyên văn `data-transfers/main-process.ts:32-38` tại HEAD hiện hành:**
```ts
// File: data-transfers/main-process.ts (Lines 32-38)
  public isDbConnected(alias: string = "default"): boolean {
    const conn = this.connections[alias];
    if (conn && "readyState" in conn) {
      return (conn as any).readyState === 1;
    }
    return !!conn;
  }
```

- **Bản sửa đề xuất tại `data-transfers/main-process.ts:32-41`:**
```ts
// Proposed in data-transfers/main-process.ts (Lines 32-41)
  public isDbConnected(alias: string = "default"): boolean {
    const conn = this.connections[alias];
    if (conn && "readyState" in conn) {
      return (conn as any).readyState === 1;
    }
    if (conn && "ended" in conn) {
      return !(conn as any).ended;
    }
    return !!conn;
  }
```

---

### 3.2. Code Demo cho `F2`: Làm sạch `start.ts` (100% Agnostic theo `config.type`) & Đóng gói Validation tại `dbHandler/postgres.ts`

- **Trích dẫn nguyên văn `start.ts:24-32` và `242-316` tại HEAD hiện hành:**
```ts
// File: start.ts (Lines 24-32)
import { DB_LIST } from "./defined/app-setting";
import {
    getTransHisExportSource,
    isDbEnabled,
    readPostgresDbConfig,
    readTransHisPgConfig,
    TRANS_HIS_PG_ALIAS
} from "./utils/trans-his-export-source";
```
```ts
// File: start.ts (Lines 242-316)
        this.dbHandler = new MongoDb(this.logger);
        this.mariaDbHandler = new MariaDb(this.logger);
        this.postgresDbHandler = new PostgresDb(this.logger);

        try {
            const enabledDbs = DB_LIST.filter(db => isDbEnabled(db.alias));
            this.logger.info(`Starting connection to ${enabledDbs.length} databases...`);

            const connections: any = {};
            let successCount = 0;

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
                    if (connection) {
                        connections[alias] = connection;
                        successCount++;
                        this.logger.info(`DB '${alias}' (${config.type}) connected.`);
                    }
                } catch (error) {
                    this.logger.error(`Failed to connect DB '${alias}':`, error.message);
                }

            }

            this.logger.info(`Connected to ${successCount}/${enabledDbs.length} databases.`);
            this.logger.info("Connections object keys before MainProcess instantiation:", Object.keys(connections));

            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }
            if (transHisExportSource === "postgres" && !connections[TRANS_HIS_PG_ALIAS]) {
                throw new Error("CRITICAL: TRANS_HIS_EXPORT_SOURCE=postgres but the trans-his-pg database failed to connect!");
            }
```

- **Bản sửa đề xuất tại `start.ts` (Xóa sạch mọi tham chiếu `trans-his`, đối xứng 1:1 với `mariadb`):**
```ts
// Proposed in start.ts (Lines 24-25 & 235-305)
import { DB_LIST } from "./defined/app-setting";
```
```ts
        this.dbHandler = new MongoDb(this.logger);
        this.mariaDbHandler = new MariaDb(this.logger);
        this.postgresDbHandler = new PostgresDb(this.logger);

        try {
            this.logger.info(`Starting connection to ${DB_LIST.length} databases...`);

            const connections: any = {};
            let successCount = 0;

            for (const config of DB_LIST) {
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
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: SVC_ENV.get()[config.hostEnvKey],
                                port: SVC_ENV.get()[config.portEnvKey],
                                database: SVC_ENV.get()[config.dbEnvKey],
                                username: SVC_ENV.get()[config.userEnvKey],
                                password: SVC_ENV.get()[config.passEnvKey],
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey]
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
                    if (connection) {
                        connections[alias] = connection;
                        successCount++;
                        this.logger.info(`DB '${alias}' (${config.type}) connected.`);
                    }
                } catch (error) {
                    this.logger.error(`Failed to connect DB '${alias}':`, error.message);
                }

            }

            this.logger.info(`Connected to ${successCount}/${DB_LIST.length} databases.`);
            this.logger.info("Connections object keys before MainProcess instantiation:", Object.keys(connections));

            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }
```

- **Trích dẫn nguyên văn `dbHandler/postgres.ts:1-61` tại HEAD hiện hành:**
```ts
// File: dbHandler/postgres.ts (Lines 1-61)
// <DOCUMENT filename="dbHandler/postgres.ts">
import { LoggerInstance } from "moleculer";
import { Pool } from "pg";
import DbConnectorFactory, { DbType, DbConfig } from "./DbConnectorFactory";

interface ConnectOptions {
    alias: string;
    host: string;
    port: number;
    database: string;
    username: string;
    password: string;
    poolSize: number;
}

// No default for host, port, database or pool size (unlike MariaDb): the caller passes the validated settings (readTransHisPgConfig),
// so a missing port can never fall back to 5432 (the writable primary).
class PostgresDb {
    private logger: LoggerInstance;

    public constructor(logger: LoggerInstance) {
        this.logger = logger;
    }

    public async createConnection(
        options: ConnectOptions,
        callback: (err: any, pool: Pool | null) => void
    ) {
        const config: DbConfig = {
            type: DbType.POSTGRES,
            host: options.host,
            port: options.port,
            database: options.database,
            username: options.username,
            password: options.password,
            poolSize: options.poolSize
        };

        try {
            const factory = DbConnectorFactory.getInstance(this.logger);

            this.logger.info(`PostgresDb wrapper requesting connection: ${options.alias}`);
            const conn = await factory.connect(config, options.alias);

            if (conn.rawPool) {
                callback(null, conn.rawPool as Pool);
            } else {
                throw new Error(`Connection '${options.alias}' established but Postgres Pool instance missing`);
            }
        } catch (err) {
            this.logger.error(`PostgresDb (${options.alias}) initialization failed`, err);
            callback(err, null);
        }
    }

    public closeAll() {
        return DbConnectorFactory.getInstance(this.logger).closeAll();
    }
}

export = PostgresDb;
```

- **Bản sửa đề xuất tại `dbHandler/postgres.ts` (Đóng gói toàn bộ việc chuẩn hóa & kiểm tra `ConnectOptions` của Postgres ngay trong `PostgresDb`):**
```ts
// Proposed in dbHandler/postgres.ts
// <DOCUMENT filename="dbHandler/postgres.ts">
import { LoggerInstance } from "moleculer";
import { Pool } from "pg";
import { SVC_ENV } from "../svc-env";
import DbConnectorFactory, { DbType, DbConfig } from "./DbConnectorFactory";

export interface PostgresConnectOptions {
    alias: string;
    host?: string;
    port?: number | string;
    database?: string;
    username?: string;
    password?: string;
    poolSize?: number | string;
}

const DEFAULT_POOL_SIZE = 5;
const DEFAULT_WORKER_CONCURRENCY = 1;

class PostgresDb {
    private logger: LoggerInstance;

    public constructor(logger: LoggerInstance) {
        this.logger = logger;
    }

    public static buildConfig(options: PostgresConnectOptions, workerConcurrency?: number): DbConfig {
        const missing: string[] = [];
        if (!options.host || String(options.host).trim() === "") { missing.push("host"); }
        if (options.port === undefined || options.port === null || String(options.port).trim() === "") { missing.push("port"); }
        if (!options.database || String(options.database).trim() === "") { missing.push("database"); }
        if (!options.username || String(options.username).trim() === "") { missing.push("username"); }
        if (options.password === undefined || options.password === null || String(options.password).trim() === "") { missing.push("password"); }
        if (missing.length > 0) {
            throw new Error(`Postgres (${options.alias}) requires explicit ${missing.join(", ")}`);
        }

        const port = Number(options.port);
        if (!Number.isInteger(port) || port < 1 || port > 65535) {
            throw new Error(`Postgres (${options.alias}) port must be a valid port number (1-65535)`);
        }

        const noPoolSize = options.poolSize === undefined || options.poolSize === null || String(options.poolSize).trim() === "";
        const poolSize = noPoolSize ? DEFAULT_POOL_SIZE : Number(options.poolSize);
        if (!Number.isInteger(poolSize) || poolSize < 1) {
            throw new Error(`Postgres (${options.alias}) poolSize must be a positive integer`);
        }

        const envConcurrency = Number(workerConcurrency ?? SVC_ENV.get().EXPORT_WORKER_CONCURRENCY);
        const concurrency = envConcurrency > 0 ? envConcurrency : DEFAULT_WORKER_CONCURRENCY;
        if (poolSize < 2 * concurrency) {
            throw new Error(
                `Postgres (${options.alias}) poolSize (${poolSize}) must be at least 2 x EXPORT_WORKER_CONCURRENCY (${concurrency})`
            );
        }

        return {
            type: DbType.POSTGRES,
            host: String(options.host),
            port,
            database: String(options.database),
            username: String(options.username),
            password: String(options.password),
            poolSize
        };
    }

    public async createConnection(
        options: PostgresConnectOptions,
        callback: (err: any, pool: Pool | null) => void
    ) {
        try {
            const config = PostgresDb.buildConfig(options);
            const factory = DbConnectorFactory.getInstance(this.logger);

            this.logger.info(`PostgresDb wrapper requesting connection: ${options.alias}`);
            const conn = await factory.connect(config, options.alias);

            if (conn.rawPool) {
                callback(null, conn.rawPool as Pool);
            } else {
                throw new Error(`Connection '${options.alias}' established but Postgres Pool instance missing`);
            }
        } catch (err) {
            this.logger.error(`PostgresDb (${options.alias}) initialization failed`, err);
            callback(err, null);
        }
    }

    public getConnection(alias = "default") {
        return DbConnectorFactory.getInstance(this.logger).getConnection(alias);
    }

    public closeAll() {
        return DbConnectorFactory.getInstance(this.logger).closeAll();
    }
}

export = PostgresDb;
```

---

### 3.3. Code Demo cho `F3`: Thu gọn `utils/trans-his-export-source.ts` về Đúng Trách Nhiệm Đơn & Khai báo Bảng trong `DB_COLLECTION` (`defined/app-setting.ts`)

- **Trích dẫn nguyên văn `utils/trans-his-export-source.ts:1-25` tại HEAD hiện hành:**
```ts
// File: utils/trans-his-export-source.ts (Lines 1-25)
import { DB_LIST } from "../defined/app-setting";
import { DbConfig, DbType } from "../dbHandler/DbConnectorFactory";
import { SVC_ENV } from "../svc-env";

export type TransHisExportSource = "mongo" | "postgres";

export const TRANS_HIS_PG_ALIAS = "trans-his-pg";
export const TRANS_HIS_MONGO_ALIAS = "trans-his";

const DEFAULT_POOL_SIZE = 5;
const DEFAULT_WORKER_CONCURRENCY = 1; // as ExportWorker

/** TRANS_HIS_EXPORT_SOURCE: where the trans-his exports read from. `mongo` (default) is the trans-his collection, `postgres` the v2 tables. */
export const getTransHisExportSource = (): TransHisExportSource => {
	const source = String(SVC_ENV.get().TRANS_HIS_EXPORT_SOURCE || "mongo").trim().toLowerCase();
	if (source !== "mongo" && source !== "postgres") {
		throw new Error(`Invalid TRANS_HIS_EXPORT_SOURCE "${source}": expected "mongo" or "postgres"`);
	}
	return source;
};

/** The model the trans-his exports need in the current source. */
export const transHisModelNames = (): string[] => (getTransHisExportSource() === "postgres" ? ["TransHisPgModel"] : ["TransHisModel"]);
```

- **Bản sửa đề xuất tại `utils/trans-his-export-source.ts` (`16` dòng thuần túy):**
```ts
// Proposed in utils/trans-his-export-source.ts (Lines 1-16)
import { SVC_ENV } from "../svc-env";

export type TransHisExportSource = "mongo" | "postgres";

/** TRANS_HIS_EXPORT_SOURCE: where the trans-his exports read from. `mongo` (default) is the trans-his collection, `postgres` the v2 tables. */
export const getTransHisExportSource = (): TransHisExportSource => {
	const source = String(SVC_ENV.get().TRANS_HIS_EXPORT_SOURCE || "mongo").trim().toLowerCase();
	if (source !== "mongo" && source !== "postgres") {
		throw new Error(`Invalid TRANS_HIS_EXPORT_SOURCE "${source}": expected "mongo" or "postgres"`);
	}
	return source;
};

/** The model the trans-his exports need in the current source. */
export const transHisModelNames = (): string[] => (getTransHisExportSource() === "postgres" ? ["TransHisPgModel"] : ["TransHisModel"]);
```

- **Trích dẫn nguyên văn `defined/app-setting.ts:75-79` và `models/trans-his-pg/trans-his-pg.model.ts:46-52` tại HEAD hiện hành:**
```ts
// File: defined/app-setting.ts (Lines 75-79)
	RECONCILE__CENTRALIZED_RECONCILIATION: "centralized_reconciliation",
	TICKET__TICKET_CATEGORIES: "ticket-categories",
	TICKET__TICKET_HISTORY: "ticket-history",
	TRANS_HIS__TRANS_HIS: "trans-his"
};
```
```ts
// File: models/trans-his-pg/trans-his-pg.model.ts (Lines 46-52)
const HAS_IDEMPOTENCY_KEY_SQL = `SELECT 1 FROM information_schema.columns
	WHERE table_schema = 'public' AND table_name = 'transaction_history' AND column_name = 'idempotency_key'`;

const FROM_SQL = `FROM public.transaction_history AS core
	JOIN public.transaction_history_payload AS payload
		ON payload.transaction_history_id = core.id`;
```

- **Bản sửa đề xuất tại `defined/app-setting.ts` và `models/trans-his-pg/trans-his-pg.model.ts`:**
```ts
// Proposed in defined/app-setting.ts (Lines 75-81)
	RECONCILE__CENTRALIZED_RECONCILIATION: "centralized_reconciliation",
	TICKET__TICKET_CATEGORIES: "ticket-categories",
	TICKET__TICKET_HISTORY: "ticket-history",
	TRANS_HIS__TRANS_HIS: "trans-his",
	TRANS_HIS_PG__TRANSACTION_HISTORY: "public.transaction_history",
	TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD: "public.transaction_history_payload"
};
```
```ts
// Proposed in models/trans-his-pg/trans-his-pg.model.ts
import { DB_COLLECTION } from "../../defined/app-setting";
...
const FROM_SQL = `FROM ${DB_COLLECTION.TRANS_HIS_PG__TRANSACTION_HISTORY} AS core
	JOIN ${DB_COLLECTION.TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD} AS payload
		ON payload.transaction_history_id = core.id`;
```

---

## 4. Phản biện Bổ sung đối với Quá Trình & Code vừa Thực Thi (`F-NEW-1..F-NEW-3`) + Code Demo Nguyên Văn

### 4.1. Bảng Ánh xạ 1-1 Phát hiện Mới (`F-NEW-1..F-NEW-3`) $\leftrightarrow$ Giải pháp (`S-FIX-NEW-1..S-FIX-NEW-3`)

| Mã phát hiện | Mức độ | Tệp và dòng (`file:line` hiện tại) | Phản biện Gốc rễ (Adversarial Root-Cause Critique) | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :---: | :--- | :--- | :--- |
| **`F-NEW-1`** | **HIGH** | `dbHandler/postgres.ts:6-63` (`92` dòng)<br>`start.ts:271-283` | **Rò rỉ Khái Niệm Tầng Worker (`workerConcurrency` / `EXPORT_WORKER_CONCURRENCY`) vào Tầng Adapter Hạ Tầng `dbHandler/postgres.ts` & `start.ts` (`L001`, `L016`):**<br>1. Trong commit gốc `1ea8d1f8` của anh trên `develop`, `dbHandler/mariadb.ts` (`59` dòng) chỉ là một thin wrapper chuyển `ConnectOptions` (`alias`, `host`, `port`, `database`, `username`, `password`, `poolSize`) xuống `DbConnectorFactory.getInstance(this.logger).connect(config, options.alias)`, và `DbConnectorFactory.ts:96-99, 182-196` đã validate `if (!config.host || !config.database) throw new Error("Postgres requires host & database")` cùng `port: config.port || 5432, max: config.poolSize || 10`.<br>2. Ở lượt sửa vừa rồi (`TASK-9`), việc bê hàm `buildConfig` từ `dbHandler/trans-his-pg.ts` sang `dbHandler/postgres.ts:27-63` và truyền `workerConcurrency: SVC_ENV.get().EXPORT_WORKER_CONCURRENCY` tại `start.ts:281` đã làm tầng `dbHandler/postgres.ts` bị phình lên `92` dòng và bị buộc chặt vào khái niệm `workerConcurrency` của `workers/ExportWorker.ts` — mất đối xứng 1-1 với `dbHandler/mariadb.ts` (`59` dòng). | **`S-FIX-NEW-1`:**<br>1. Đưa `dbHandler/postgres.ts` (`59` dòng) và `start.ts:271-283` về **đối xứng tuyệt đối 1-1 từng dòng với `dbHandler/mariadb.ts`**: xóa `workerConcurrency` khỏi `start.ts:281` và xóa `PostgresDb.buildConfig` khỏi `dbHandler/postgres.ts`, để `DbConnectorFactory` (`dbHandler/DbConnectorFactory.ts:96-99, 182-196`) đảm nhiệm toàn bộ việc khởi tạo `pg.Pool` giống hệt `MariaDb`.<br>2. Cập nhật `test/unit/trans-his-pg/trans-his-pg.connection.test.ts` khớp với `PostgresDb` đối xứng `MariaDb`. |
| **`F-NEW-2`** | **MEDIUM** | `domain/handlers/trans-his/GetAllTransHisExportHandler.ts:35-46`<br>`domain/handlers/trans-his/GetAllTransHisExportPgHandler.ts:26-37` | **Trùng lặp 12 Dòng Biến đổi Row (`mapTransHisExportDoc`) giữa Handler Mongo và Handler Postgres (`L016`):**<br>Cả 2 handler `GetAllTransHisExportHandler.ts:35-46` và `GetAllTransHisExportPgHandler.ts:26-37` đều viết lặp lại nguyên khối 12 dòng gán `phoneSender`, `phoneReceiver`, `bankConnector`, `sourceOfFundCode`, `sourceOfFundName`, `transAmount`, `fee`, `productInvestTypeName`. Nếu sau này thêm/sửa 1 trường chung của báo cáo `trans-his`, rất dễ sửa 1 bên quên 1 bên. | **`S-FIX-NEW-2`:**<br>Tách hàm thuần đơn trách nhiệm `mapTransHisExportDoc(doc: any)` ra file `domain/handlers/trans-his/trans-his-export-row.mapper.ts` (`18` dòng) và tái sử dụng ở cả `GetAllTransHisExportHandler.ts` lẫn `GetAllTransHisExportPgHandler.ts`. |
| **`F-NEW-3`** | **LOW** | `defined/app-setting.ts:35` | **Comment Cũ Sai Sự Thật còn sót lại tại `defined/app-setting.ts:35`:**<br>Dòng `35` vẫn còn comment `// Only connected when TRANS_HIS_EXPORT_SOURCE=postgres (the v2 trans-his tables, read-only)`, mâu thuẫn với vòng lặp `for (const config of DB_LIST)` đã khôi phục tại `start.ts:245`. | **`S-FIX-NEW-3`:**<br>Xóa dòng comment sai sự thật tại `defined/app-setting.ts:35`. |

---

### 4.2. Code Demo Nguyên văn cho `F-NEW-1..F-NEW-3` (`L004`)

#### A. Code Demo `F-NEW-1`: Đối xứng 1-1 tuyệt đối giữa `dbHandler/postgres.ts` + `start.ts:271-283` với `dbHandler/mariadb.ts`

- **Hiện trạng tại `dbHandler/postgres.ts:1-92` và `start.ts:271-283`:**
```ts
// Hiện trạng tại start.ts:271-283
                    } else if (config.type === "postgres") {
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: SVC_ENV.get()[config.hostEnvKey],
                                port: SVC_ENV.get()[config.portEnvKey],
                                database: SVC_ENV.get()[config.dbEnvKey],
                                username: SVC_ENV.get()[config.userEnvKey],
                                password: SVC_ENV.get()[config.passEnvKey],
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey],
                                workerConcurrency: SVC_ENV.get().EXPORT_WORKER_CONCURRENCY
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
```
- **Bản sửa đề xuất tại `start.ts:271-282` và `dbHandler/postgres.ts:1-59` (Đối xứng 100% với `dbHandler/mariadb.ts`):**
```ts
// Đề xuất tại start.ts:271-282 (giống hệt nhánh mariadb ở dòng 259-270)
                    } else if (config.type === "postgres") {
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: SVC_ENV.get()[config.hostEnvKey],
                                port: SVC_ENV.get()[config.portEnvKey],
                                database: SVC_ENV.get()[config.dbEnvKey],
                                username: SVC_ENV.get()[config.userEnvKey],
                                password: SVC_ENV.get()[config.passEnvKey],
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 10
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
```
```ts
// Đề xuất tại dbHandler/postgres.ts (59 dòng — đối xứng 1-1 với dbHandler/mariadb.ts)
// <DOCUMENT filename="dbHandler/postgres.ts">
import { LoggerInstance } from "moleculer";
import { Pool } from "pg";
import DbConnectorFactory, { DbType, DbConfig } from "./DbConnectorFactory";

interface ConnectOptions {
    alias: string;
    host?: string;
    port?: number;
    database?: string;
    username?: string;
    password?: string;
    poolSize?: number;
}

class PostgresDb {
    private logger: LoggerInstance;

    public constructor(logger: LoggerInstance) {
        this.logger = logger;
    }

    public async createConnection(
        options: ConnectOptions,
        callback: (err: any, pool: Pool | null) => void
    ) {
        const config: DbConfig = {
            type: DbType.POSTGRES,
            host: options.host || "localhost",
            port: options.port || 5432,
            database: options.database,
            username: options.username,
            password: options.password,
            poolSize: options.poolSize || 10
        };

        try {
            const factory = DbConnectorFactory.getInstance(this.logger);

            this.logger.info(`PostgresDb wrapper requesting connection: ${options.alias}`);
            const conn = await factory.connect(config, options.alias);

            if (conn.rawPool) {
                callback(null, conn.rawPool as Pool);
            } else {
                throw new Error(`Connection '${options.alias}' established but Postgres Pool instance missing`);
            }
        } catch (err) {
            this.logger.error(`PostgresDb (${options.alias}) initialization failed`, err);
            callback(err, null);
        }
    }

    public getConnection(alias = "default") {
        return DbConnectorFactory.getInstance(this.logger).getConnection(alias);
    }

    public closeAll() {
        return DbConnectorFactory.getInstance(this.logger).closeAll();
    }
}

export = PostgresDb;
```

#### B. Code Demo `F-NEW-2`: Tách `domain/handlers/trans-his/trans-his-export-row.mapper.ts` dùng chung cho cả 2 Handler

```ts
// File mới đề xuất: domain/handlers/trans-his/trans-his-export-row.mapper.ts (18 dòng)
import { BANK_NAME_BY_CODE, INVEST_PRODUCT_TYPE } from "../../../defined/trans-his-mapping";

export const mapTransHisExportDoc = (doc: any) => ({
	...doc,
	phoneSender: doc.sender?.phone ?? null,
	phoneReceiver: doc.receiver?.phone ?? null,
	bankConnector: BANK_NAME_BY_CODE[doc.sourceOfFund?.bankConnector] || doc.sourceOfFund?.bankConnector || null,
	sourceOfFundCode: doc.sourceOfFund?.code ?? null,
	sourceOfFundName: doc.sourceOfFund?.name ?? null,
	transAmount: doc.info?.transAmount ?? 0,
	fee: doc.info?.fee ?? 0,
	productInvestTypeName: INVEST_PRODUCT_TYPE[doc.info?.productInvestType] || null
});
```

#### C. Code Demo `F-NEW-3`: Xóa comment sai sự thật tại `defined/app-setting.ts:35`

```ts
// Hiện trạng tại defined/app-setting.ts:34-44
	},
	// Only connected when TRANS_HIS_EXPORT_SOURCE=postgres (the v2 trans-his tables, read-only)
	{
		alias: "trans-his-pg",
		type: "postgres",
// Đề xuất tại defined/app-setting.ts:34-43
	},
	{
		alias: "trans-his-pg",
		type: "postgres",
```

---

## 9. [TASK-13] Điều tra & Giải pháp Lệch Số Lượng Bản Ghi (`409` vs `37`) Khi Truyền `walletPayOnly=true` Giữa `exportType=TransHisExport` và `/api/v1/admin-portal/trans`

### 9.1. Bằng chứng Nguyên nhân Gốc rễ (`VERIFIED` 100% qua `file:line` & Runtime DB `EV-20261103T163655-f7c7`, `EV-20261103T163826-8681`, `EV-20261103T163941-c7c0`)

1. **Ở API Danh sách `/api/v1/admin-portal/trans` (`core-trans-proxy-history-service` & `core-trans-proxy-history-service-v2`, ticket `GP2-3856`):**
   - `core-trans-proxy-history-service-v2/data-transfers/params/transHis.paging.param.ts:46`: Khai báo `walletPayOnly?: boolean | string;`.
   - `core-trans-proxy-history-service-v2/logics/transHis.manage.logic.ts:161`: Truyền `...(params.walletPayOnly === true || params.walletPayOnly === "true" ? { walletPayOnly: true } : {})` xuống repository.
   - `core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1229-1234` (và `core-trans-proxy-history-service/logics/transHis.manage.logic.ts:228-244` + commit `640cd2a` theo `parity/v1-parity.ts:23-24`): Khi `walletPayOnly === true || walletPayOnly === "true"`, áp dụng 2 điều kiện lọc:
     - Loại bỏ toàn bộ giao dịch có `serviceCode = "COLLECTION_DISBURSEMENT"`: `core.service_code IS DISTINCT FROM 'COLLECTION_DISBURSEMENT'` (Mongo: `{ serviceCode: { $ne: "COLLECTION_DISBURSEMENT" } }`).
     - Với giao dịch `transType = "PAY"`, chỉ giữ lại giao dịch có `channelId = "WALLET"` và `serviceCode != null`: `(core.trans_type IS DISTINCT FROM 'PAY' OR (core.trans_type = 'PAY' AND core.channel_id = 'WALLET' AND core.service_code IS NOT NULL))` (Mongo: `{ $or: [{ transType: { $ne: TRANS_TYPE.PAY } }, { transType: TRANS_TYPE.PAY, channelId: "WALLET", serviceCode: { $ne: null } }] }`).

2. **Ở API Xuất file `/api/v1/admin-portal/export-async?exportType=TransHisExport` (`centralized-export-service` nhánh `feat/trans-his-postgres-export-fix`):**
   - **Phát hiện `F-WPO-1` (`data-transfers/params/trans-his/trans-his-export.params.ts:32-59`):** Class `TransHisExportParams` **hoàn toàn chưa khai báo trường `walletPayOnly`**.
   - **Phát hiện `F-WPO-2` (`logics/export/trans-his/trans-his-export.pure.ts:17-101`):** Hàm `buildTransHisFilter(params)` **hoàn toàn không đọc và không xử lý `params.walletPayOnly`**, dẫn tới tham số `walletPayOnly=true` trên URL bị bỏ qua 100%.
   - **Phát hiện `F-WPO-3` (`models/trans-his-pg/trans-his-pg.query.ts:84-112, 138-152, 206-219`):** Bộ biên dịch Mongo-filter sang SQL của `trans-his-pg` hiện tại chưa khai báo cột `"serviceCode": "core.service_code"` và `"channelId": "core.channel_id"` trong `TEXT_COLUMNS` / `TEXT_OPERATORS`, chưa hỗ trợ toán tử `$and` trong `conjunction()`, và chưa hỗ trợ toán tử `$ne` (bao gồm `$ne: "COLLECTION_DISBURSEMENT"`, `$ne: "PAY"`, `$ne: null`) trong `textSql()`.

3. **Đối soát Toán học Trực tiếp trên PostgreSQL Thật (`EV-20261103T163655-f7c7`, `EV-20261103T163826-8681`, `EV-20261103T163941-c7c0`):**
   - Trong khoảng `dateFr=2026-10-31T17:00:00.000Z` → `dateTo=2026-11-04T16:59:59.999Z` với 15 `transType` đã truyền:
     - Tại thời điểm job `export-1793697593284-enf0v715q` chạy lúc `09:19:53Z` (`EV-20261103T163941-c7c0`), tổng số bản ghi khi **bỏ qua `walletPayOnly=true`** là **`409` bản ghi** (và sau khi có thêm 6 giao dịch `PAY` `COLLECTION_DISBURSEMENT` mới phát sinh là **`415` bản ghi**, `EV-20261103T163826-8681`).
     - Phân rã chi tiết `415` bản ghi (`409` bản ghi lúc `09:19:53Z`):
       - `PAY` có `service_code = 'COLLECTION_DISBURSEMENT'`, `channel_id = 'WALLET'`: **`363` bản ghi** (`357` bản ghi lúc `09:19:53Z`) → **Bị loại bởi `service_code IS DISTINCT FROM 'COLLECTION_DISBURSEMENT'`**.
       - `PAY` có `service_code = 'COLLECTION_DISBURSEMENT'`, `channel_id = 'BANK_TRANSFER'`: **`1` bản ghi** → **Bị loại bởi cả 2 điều kiện**.
       - `PAY` có `service_code = NULL`, `channel_id = 'WALLET'`: **`14` bản ghi** (`13` `SUCCESS` + `1` `PENDING`) → **Bị loại bởi `service_code IS NOT NULL`**.
       - `PAY` có `service_code = 'BUS_TICKET_SERVICE'`, `channel_id = 'WALLET'`: **`12` bản ghi** → **ĐƯỢC GIỮ LẠI**.
       - Các `transType != 'PAY'` (`service_code` là `'WALLET'` hoặc `NULL`, không có dòng nào là `'COLLECTION_DISBURSEMENT'`): `CASHIN` (**`7`**), `TOPUP` (**`6`**), `RECEIVE_MONEY` (**`6`**), `REFUND` (**`2`**), `SOFTPIN` (**`2`**), `TRANSFER` (**`2`**) = **`25` bản ghi** → **ĐƯỢC GIỮ LẠI**.
     - **Tổng số bản ghi hợp lệ khi áp dụng `walletPayOnly=true`:** `12 + 25 =` **ĐÚNG CHÍNH XÁC `37` BẢN GHI** (`409 - 357 - 1 - 14 = 37` tại thời điểm `09:19:53Z`, và `415 - 363 - 1 - 14 = 37` tại thời điểm hiện tại)!

### 9.2. Giải pháp Duy nhất Tối ưu (`S3` — Ánh xạ 1-1 `F-WPO-1..3`) & Code Demo (Chờ `APPROVE`)

1. **`F-WPO-1` → `data-transfers/params/trans-his/trans-his-export.params.ts:50`:** Bổ sung `@IsOptional() @IsString() public walletPayOnly?: string;` vào `TransHisExportParams`.
2. **`F-WPO-2` → `logics/export/trans-his/trans-his-export.pure.ts:99`:** Trong `buildTransHisFilter(params)`, khi `params.walletPayOnly === true || params.walletPayOnly === "true"`, bổ sung điều kiện lọc vào `filter.$and` (dùng `filter.$and` để không ghi đè `filter.$or` khi người dùng lọc kèm `customerId` hoặc `phone`) khớp 100% với logic `GP2-3856` trên cả Mongo (`trans-his`) và Postgres (`trans-his-pg`).
3. **`F-WPO-3` → `models/trans-his-pg/trans-his-pg.query.ts:84-112, 138-165, 206-225`:**
   - Thêm `"serviceCode": "core.service_code"` và `"channelId": "core.channel_id"` vào `TEXT_COLUMNS`.
   - Thêm `"$ne"` cho `"transType"` và khai báo `"serviceCode": ["eq", "$ne"]`, `"channelId": ["eq"]` trong `TEXT_OPERATORS`.
   - Hỗ trợ toán tử `$and` trong `conjunction()` (qua hàm `conjunctionList()`) và toán tử `$ne` trong `textSql()` (`operand === null` biên dịch thành `${column} IS NOT NULL`; `operand` chuỗi biên dịch thành `${column} IS DISTINCT FROM $n` cho cột nullable).

#### Code Demo 1: `data-transfers/params/trans-his/trans-his-export.params.ts:41-54`
```ts
// Hiện trạng tại data-transfers/params/trans-his/trans-his-export.params.ts:41-54
    @IsOptional() @IsString() public transId?: string;
    @IsOptional() @IsString() public transType?: string; // Ví dụ: "PAY,CASHOUT"
    @IsOptional() @IsString() public status?: string; // Ví dụ: "SUCCESS,FAILED"
    @IsOptional() @IsString() public partnerCode?: string;
    @IsOptional() @IsString() public customerId?: string;
    @IsOptional() @IsString() public phone?: string;
    @IsOptional() @IsString() public sysTrans?: string;
    @IsOptional() @IsString() public transIdRef?: string;
    @IsOptional() @IsString() public info?: string; // JSON string for complex filtering

    @IsOptional() @IsString() public sortBy?: string;
    @IsOptional() @IsString() public sortType?: string;

// Đề xuất cập nhật tại data-transfers/params/trans-his/trans-his-export.params.ts:41-55
    @IsOptional() @IsString() public transId?: string;
    @IsOptional() @IsString() public transType?: string; // Ví dụ: "PAY,CASHOUT"
    @IsOptional() @IsString() public status?: string; // Ví dụ: "SUCCESS,FAILED"
    @IsOptional() @IsString() public partnerCode?: string;
    @IsOptional() @IsString() public customerId?: string;
    @IsOptional() @IsString() public phone?: string;
    @IsOptional() @IsString() public sysTrans?: string;
    @IsOptional() @IsString() public transIdRef?: string;
    @IsOptional() @IsString() public info?: string; // JSON string for complex filtering
    @IsOptional() @IsString() public walletPayOnly?: string;

    @IsOptional() @IsString() public sortBy?: string;
    @IsOptional() @IsString() public sortType?: string;
```

#### Code Demo 2: `logics/export/trans-his/trans-his-export.pure.ts:93-101`
```ts
// Hiện trạng tại logics/export/trans-his/trans-his-export.pure.ts:93-101
    if ("info" in params && params.info && _.keys(params.info).length > 0) {
        const keys = _.keys(params.info);
        keys.forEach(key => {
            filter[key] = params.info[key];
        });
    }

    return filter;
};

// Đề xuất cập nhật tại logics/export/trans-his/trans-his-export.pure.ts:93-116
    if ("info" in params && params.info && _.keys(params.info).length > 0) {
        const keys = _.keys(params.info);
        keys.forEach(key => {
            filter[key] = params.info[key];
        });
    }

    if (params.walletPayOnly === true || params.walletPayOnly === "true") {
        filter.$and = [
            ...(filter.$and || []),
            { serviceCode: { $ne: "COLLECTION_DISBURSEMENT" } },
            {
                $or: [
                    { transType: { $ne: TRANS_TYPE.PAY } },
                    {
                        transType: TRANS_TYPE.PAY,
                        channelId: "WALLET",
                        serviceCode: { $ne: null }
                    }
                ]
            }
        ];
    }

    return filter;
};
```

#### Code Demo 3: `models/trans-his-pg/trans-his-pg.query.ts:84-165, 206-220`
```ts
// Hiện trạng tại models/trans-his-pg/trans-his-pg.query.ts:84-112
const TEXT_COLUMNS: Readonly<Record<string, string>> = {
	"transId": "core.trans_id",
	"transType": "core.trans_type",
	"status": "core.status",
	"originalTransHisId": "core.original_trans_his_id",
	"sender.customerId": "core.sender_customer_id",
	"receiver.customerId": "core.receiver_customer_id",
	"sender.phone": "core.sender_phone",
	"receiver.phone": "core.receiver_phone",
	"receiver.credit": "core.receiver_credit",
	"info.partnerCode": "core.partner_code"
};

/** Text paths whose column is NOT NULL: no document has them missing or null. */
const NEVER_NULL_TEXT: readonly string[] = ["transId"];

/** Operators each text path takes: "eq", "$in", "$regex" (case-insensitive only). */
const TEXT_OPERATORS: Readonly<Record<string, readonly string[]>> = {
	"transId": ["eq", "$in"],
	"transType": ["eq", "$in"],
	"status": ["eq", "$in"],
	"originalTransHisId": ["$regex"],
	"sender.customerId": ["eq"],
	"receiver.customerId": ["eq"],
	"sender.phone": ["eq"],
	"receiver.phone": ["eq"],
	"receiver.credit": ["$regex"],
	"info.partnerCode": ["eq", "$regex"]
};

// Đề xuất cập nhật tại models/trans-his-pg/trans-his-pg.query.ts:84-116
const TEXT_COLUMNS: Readonly<Record<string, string>> = {
	"transId": "core.trans_id",
	"transType": "core.trans_type",
	"status": "core.status",
	"originalTransHisId": "core.original_trans_his_id",
	"serviceCode": "core.service_code",
	"channelId": "core.channel_id",
	"sender.customerId": "core.sender_customer_id",
	"receiver.customerId": "core.receiver_customer_id",
	"sender.phone": "core.sender_phone",
	"receiver.phone": "core.receiver_phone",
	"receiver.credit": "core.receiver_credit",
	"info.partnerCode": "core.partner_code"
};

/** Text paths whose column is NOT NULL: no document has them missing or null. */
const NEVER_NULL_TEXT: readonly string[] = ["transId"];

/** Operators each text path takes: "eq", "$in", "$ne", "$regex" (case-insensitive only). */
const TEXT_OPERATORS: Readonly<Record<string, readonly string[]>> = {
	"transId": ["eq", "$in"],
	"transType": ["eq", "$in", "$ne"],
	"status": ["eq", "$in"],
	"originalTransHisId": ["$regex"],
	"serviceCode": ["eq", "$ne"],
	"channelId": ["eq"],
	"sender.customerId": ["eq"],
	"receiver.customerId": ["eq"],
	"sender.phone": ["eq"],
	"receiver.phone": ["eq"],
	"receiver.credit": ["$regex"],
	"info.partnerCode": ["eq", "$regex"]
};
```

```ts
// Hiện trạng tại models/trans-his-pg/trans-his-pg.query.ts:138-152 & 206-219
function conjunction(filter: Record<string, any>, bind: Bind, depth: number): string[] {
	const parts: string[] = [];
	for (const key of Object.keys(filter)) {
		if (key === "$or") {
			parts.push(disjunction(filter.$or, bind, depth));
		} else if (key === "$expr") {
			parts.push(exprSql(filter.$expr, bind));
		} else if (key.startsWith("$")) {
			return refuse("unsupported_operator", `top-level operator "${shown(key)}"`);
		} else {
			parts.push(...fieldSql(key, filter[key], bind));
		}
	}
	return parts;
}
// ...
function textSql(key: string, column: string, spec: unknown, bind: Bind): string {
	const allowed = TEXT_OPERATORS[key];
	const operators = operatorsOf(key, spec, allowed);
	if (operators.length !== 1) { return refuse("unsupported_operator", `"${shown(key)}" (one operator)`); }
	const [name, operand] = operators[0];
	if (name === "eq") { return `${column} = ${bind(textOf(key, operand))}`; }
	if (name === "$in") {
		const items = listOf(key, operand).filter(item => !(NEVER_NULL_TEXT.includes(key) && (item === null || item === undefined)));
		const texts = items.map(item => textOf(key, item));
		return texts.length === 0 ? "FALSE" : `${column} = ANY(${bind(texts)}::text[])`;
	}
	return `${column} COLLATE "C" ~* ${bind(approvedRegex(operand as string))}`;
}

// Đề xuất cập nhật tại models/trans-his-pg/trans-his-pg.query.ts:138-179 & 220-241
function conjunction(filter: Record<string, any>, bind: Bind, depth: number): string[] {
	const parts: string[] = [];
	for (const key of Object.keys(filter)) {
		if (key === "$or") {
			parts.push(disjunction(filter.$or, bind, depth));
		} else if (key === "$and") {
			parts.push(...conjunctionList(filter.$and, bind, depth));
		} else if (key === "$expr") {
			parts.push(exprSql(filter.$expr, bind));
		} else if (key.startsWith("$")) {
			return refuse("unsupported_operator", `top-level operator "${shown(key)}"`);
		} else {
			parts.push(...fieldSql(key, filter[key], bind));
		}
	}
	return parts;
}

function conjunctionList(branches: unknown, bind: Bind, depth: number): string[] {
	if (depth >= MAX_FILTER_DEPTH) { return refuse("unsupported_filter", "$and (nested too deep)"); }
	if (!Array.isArray(branches) || branches.length === 0) { return refuse("unsupported_value", "$and (a non-empty array is expected)"); }
	const compiled: string[] = [];
	for (const branch of branches) {
		if (!isPlainObject(branch)) { return refuse("unsupported_value", "$and (an array of objects is expected)"); }
		const parts = conjunction(branch, bind, depth + 1);
		if (parts.length === 0) { return refuse("unsupported_value", "$and (empty branch)"); }
		compiled.push(...parts);
	}
	return compiled;
}
// ...
function textSql(key: string, column: string, spec: unknown, bind: Bind): string {
	const allowed = TEXT_OPERATORS[key];
	const operators = operatorsOf(key, spec, allowed);
	if (operators.length !== 1) { return refuse("unsupported_operator", `"${shown(key)}" (one operator)`); }
	const [name, operand] = operators[0];
	if (name === "eq") { return `${column} = ${bind(textOf(key, operand))}`; }
	if (name === "$ne") {
		if (operand === null) {
			return NEVER_NULL_TEXT.includes(key) ? "TRUE" : `${column} IS NOT NULL`;
		}
		const text = textOf(key, operand);
		return NEVER_NULL_TEXT.includes(key)
			? `${column} <> ${bind(text)}`
			: `${column} IS DISTINCT FROM ${bind(text)}`;
	}
	if (name === "$in") {
		const items = listOf(key, operand).filter(item => !(NEVER_NULL_TEXT.includes(key) && (item === null || item === undefined)));
		const texts = items.map(item => textOf(key, item));
		return texts.length === 0 ? "FALSE" : `${column} = ANY(${bind(texts)}::text[])`;
	}
	return `${column} COLLATE "C" ~* ${bind(approvedRegex(operand as string))}`;
}
```

---

## 10. [TASK-16] Thiết kế Cơ chế Đọc Rule Điều Tuyến `trans-his` Trực tiếp từ Database của `rule-service` (`Propose-Only`, Chờ `APPROVE`)

### 10.1. Kết quả Khảo sát `rule-service` & Thực trạng Database (`VERIFIED` 100% qua `file:line`)

1. **Cấu trúc Lưu trữ trong `rule-service` (`rule-service/defined/app-setting.ts:4-84`, `rule-service/models/rule.model.ts:17-46`):**
   - `rule-service` lưu trữ rule trong MongoDB (database `rule-service`, collection `DB_COLLECTION.RULE = "rules"`).
   - Schema của `RuleModel` (`rule-service/models/rule.model.ts:17-37`):
     - `name: String`, `domain: String` (ObjectId dạng chuỗi của `domains`), `domainCode: String` (ví dụ `"THR"`), `activeFr: String` (ISO-8601), `activeTo: String` (ISO-8601), `priority: Number` (mặc định `1`), `status: String` (`"ACTIVATED"`, `"INACTIVATED"`, `"INITIAL"`), `conditions: Object` (`{ all: [...] }` hoặc `{ any: [...] }`), `event: Object` (`{ isLoadBalancing: false, params: { connectorName: "v2" }, type: "..." }`), `ruleType: String` (`"routing"`), `ruleAction: String` (`"JSON_RULE"` / `"JSON"` / `"QUERY"`), `isActive: Boolean`, `isDelete: Boolean`.
2. **Cách `rule-service` Truy vấn & Đánh giá Rule Điều tuyến (`rule-service/application/rule/actions/executeRoutingRule.handler.ts:33-46`, `ruleMainLogic.ts:16-34`, `executeRoutingRule.processors.ts:57-76`):**
   - Bộ lọc MongoDB lấy danh sách rule đang kích hoạt (`RuleMainLogic.getDefaultRuleFilter`):
     ```ts
     const nowIso = moment(new Date()).utc().toISOString();
     const ruleFilter = {
       ruleType: "routing",
       activeFr: { $lte: nowIso },
       activeTo: { $gte: nowIso },
       ruleAction: { $ne: "QUERY" },
       isActive: true,
       isDelete: false,
       status: "ACTIVATED",
       $or: [{ domain: ruleDomain }, { domainCode: ruleDomain }]
     };
     ```
   - Các trường select: `{ _id: 1, priority: 1, conditions: 1, event: 1, activeFr: 1, activeTo: 1 }`.
   - Đánh giá điều kiện (`RoutingRuleLogic.execute`): sắp xếp giảm dần theo `priority`, duyệt cây điều kiện `conditions` (`all` / `any` và 10 toán tử `equal`, `notEqual`, `lessThan`, `lessThanInclusive`, `greaterThan`, `greaterThanInclusive`, `in`, `notIn`, `contains`, `doesNotContain` tại `rule-service/defined/app-setting.ts:42-55`) trên dữ kiện đầu vào `{ caller, action }`. Rule đầu tiên khớp trả về `event?.params?.connectorName`. Khi `connectorName === "v2"`, hệ thống điều hướng sang `v2` (`postgres`); ngược lại về `v1` (`mongo`).
3. **Thực trạng Dữ liệu trên MongoDB `rule-service` hiện tại:**
   - Domain `THR` (`_id: "6ad070df13780e2852ba69f5"`, `code: "THR"`, `name: "Trans-History-Routing"`).
   - Có 2 bản ghi `ruleType: "routing"` thuộc `domainCode: "THR"`:
     - `_id: "6ad0760413780e2852ba6a4d"` (`status: "INACTIVATED"`, `isActive: false`, `priority: 1`, `conditions.all`: `action in ["getListTransHisForManager", "getDetailTransHisForManager", "getMerchantTransHisDetails", "findOneTransHis"]`, `event.params.connectorName: "v2"`).
     - `_id: "6ae2c7a021fb0505e59e1277"` (`status: "ACTIVATED"`, `isActive: true`, `priority: 10`, `activeFr: "2026-10-29T00:00:00+07:00"`, `activeTo: "2099-12-31T23:59:59+07:00"`, `conditions: { all: [] }`, `event.params.connectorName: "v2"`).

### 10.2. Bảng Ánh xạ 1-1 Phát hiện (`F-RULE-1..F-RULE-4`) $\leftrightarrow$ Giải pháp Duy nhất (`S-RULE-1..S-RULE-4`)

| Mã phát hiện | Tệp và dòng (`file:line`) | Phản biện Hiện trạng | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :--- | :--- | :--- |
| **`F-RULE-1`** | `defined/app-setting.ts:5-80`<br>`svc-env.ts:5-94`<br>`data-transfers/entities/index.ts`<br>`models/index.ts` | `centralized-export-service` kết nối trực tiếp tới DB của các microservice thông qua `DB_LIST` và `ModelRegistry`, nhưng hiện chưa có kết nối tới DB `rule` (`rule-service`) cũng như chưa có `RuleEntity` và `RuleModel`. | **`S-RULE-1`:**<br>- Thêm `{ alias: "rule", type: "mongo", uriEnvKey: "RULE_MONGO_URI", poolSizeEnvKey: "RULE_MONGO_POOLSIZE" }` vào `DB_LIST` và `RULE__RULE: "rules"` vào `DB_COLLECTION` (`defined/app-setting.ts`).<br>- Khai báo `RULE_MONGO_URI`, `RULE_MONGO_POOLSIZE`, `TRANS_HIS_ROUTING_RULE_DOMAIN`, `TRANS_HIS_ROUTING_CACHE_MS` trong `svc-env.ts`.<br>- Tạo `data-transfers/entities/rule/rule.entity.ts` (`RuleEntity`) và `models/rule/rule.model.ts` (`RuleModel`, khớp schema `rule-service/models/rule.model.ts:17-46`), đăng ký `"rule": [RuleModel]` vào `ModelRegistry` (`models/index.ts`). |
| **`F-RULE-2`** | `utils/trans-his-routing-rule.ts` *(file mới)* | Cần một module đơn trách nhiệm (SRP) thực thi đúng bộ lọc `getDefaultRuleFilter` và bộ đánh giá điều kiện `conditions` / `priority` của `rule-service` trực tiếp từ `RuleModel` mà không phụ thuộc RPC hay thêm thư viện ngoài. | **`S-RULE-2`:**<br>Tạo `utils/trans-his-routing-rule.ts` chứa `buildRoutingRuleFilter(domain, nowIso)` và `evaluateRoutingRules(ruleList, facts)` khớp 100% với `rule-service/application/rule/actions/ruleMainLogic.ts:16-34` và `executeRoutingRule.processors.ts:57-76` (`priority` giảm dần, hỗ trợ `all`/`any` và đủ 10 toán tử `OPERATOR_TYPES`, trả về `connectorName === "v2"` $\rightarrow$ `"postgres"`, ngược lại `"mongo"`). |
| **`F-RULE-3`** | `utils/trans-his-export-source.ts:1-15` | `utils/trans-his-export-source.ts` hiện chỉ đọc biến tĩnh `TRANS_HIS_EXPORT_SOURCE` mà chưa đọc từ `RuleModel` (kèm cache TTL `TRANS_HIS_ROUTING_CACHE_MS` mặc định `30000` ms và giữ lại quyết định gần nhất khi DB lỗi theo đúng thiết kế rollout `2026-10-13-trans-his-v2-rollout-phases.md:14-15`). | **`S-RULE-3`:**<br>Cập nhật `utils/trans-his-export-source.ts`:<br>- Bổ sung `refreshTransHisExportSource(mainProcess, action?)`: khi `mainProcess?.models?.RuleModel` sẵn sàng, truy vấn rule thuộc domain `TRANS_HIS_ROUTING_RULE_DOMAIN || "THR"` với `facts = { caller: "centralized-export-service", action: action || "getListTransHisForManager" }`, cache kết quả theo `TRANS_HIS_ROUTING_CACHE_MS` (mặc định `30000` ms).<br>- Khi `RuleModel` chưa được khởi tạo (ví dụ trong unit test cũ chỉ set `TRANS_HIS_EXPORT_SOURCE`), tự động fallback về `SVC_ENV.get().TRANS_HIS_EXPORT_SOURCE` (mặc định `"mongo"`). |
| **`F-RULE-4`** | `domain/QueryHandlerFactory.ts:77-159`<br>`start.ts:302-308`<br>`logics/processors/BaseExportProcessor.ts:151-160` | `QueryHandlerFactory.registerHandlers` hiện chỉ đăng ký 1 cặp handler (`Pg` hoặc `Mongo`) lúc khởi động. Nếu rule trên DB `rule-service` được bật/tắt lúc runtime, `getHandler` sẽ không đổi theo nếu chỉ đăng ký 1 nhánh tĩnh. Đồng thời cần giữ nguyên 2 cặp handler riêng biệt (`GetAllTransHisExportHandler` / `GetAllTransHisExportPgHandler` và `GetListTransHisByTransIdHandler` / `GetListTransHisByTransIdPgHandler`) theo đúng yêu cầu của anh. | **`S-RULE-4`:**<br>- Trong `domain/QueryHandlerFactory.ts`: khởi tạo sẵn cả 2 map riêng biệt `transHisMongoHandlers` và `transHisPgHandlers` trong `registerHandlers(mainProcess)`, và tại `getHandler(queryName)` trả về handler tương ứng theo `getTransHisExportSource()`.<br>- Trong `start.ts:305`: gọi `await refreshTransHisExportSource(this.mainProcess)` ngay sau khi khởi tạo `MainProcess`.<br>- Trong `logics/processors/BaseExportProcessor.ts:158`: gọi `await refreshTransHisExportSource(mainProcess)` ở đầu `execute(job, mainProcess)` (trước `validatePrerequisites` và ngoài vòng lặp `do ... while (cursor)` để đảm bảo tính liên tục của cursor trong suốt vòng đời 1 job). |

### 10.3. Code Demo Nguyên văn (`Propose-Only` — Chờ `APPROVE`)

#### 1. `defined/app-setting.ts:20-82`
```ts
// Hiện trạng tại defined/app-setting.ts:20-25 & 75-80
	{ alias: "disbursement", type: "mongo", uriEnvKey: "DISBURSEMENT_MONGO_URI", poolSizeEnvKey: "DISBURSEMENT_MONGO_POOLSIZE" },
	{ alias: "config", type: "mongo", uriEnvKey: "CONFIG_MONGO_URI", poolSizeEnvKey: "CONFIG_MONGO_POOLSIZE" },
	{ alias: "merchant", type: "mongo", uriEnvKey: "MERCHANT_MONGO_URI", poolSizeEnvKey: "MERCHANT_MONGO_POOLSIZE" },
	{ alias: "ticket", type: "mongo", uriEnvKey: "TICKET_MONGO_URI", poolSizeEnvKey: "TICKET_MONGO_POOLSIZE" },
	{ alias: "core-trans-proxy", type: "mongo", uriEnvKey: "CORE_TRANS_PROXY_MONGO_URI", poolSizeEnvKey: "CORE_TRANS_PROXY_MONGO_POOLSIZE" },
// ...
	RECONCILE__CENTRALIZED_RECONCILIATION: "centralized_reconciliation",
	TICKET__TICKET_CATEGORIES: "ticket-categories",
	TICKET__TICKET_HISTORY: "ticket-history",
	TRANS_HIS__TRANS_HIS: "trans-his",
	TRANS_HIS_PG__TRANSACTION_HISTORY: "public.transaction_history",
	TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD: "public.transaction_history_payload"
};

// Đề xuất cập nhật tại defined/app-setting.ts
	{ alias: "disbursement", type: "mongo", uriEnvKey: "DISBURSEMENT_MONGO_URI", poolSizeEnvKey: "DISBURSEMENT_MONGO_POOLSIZE" },
	{ alias: "config", type: "mongo", uriEnvKey: "CONFIG_MONGO_URI", poolSizeEnvKey: "CONFIG_MONGO_POOLSIZE" },
	{ alias: "rule", type: "mongo", uriEnvKey: "RULE_MONGO_URI", poolSizeEnvKey: "RULE_MONGO_POOLSIZE" },
	{ alias: "merchant", type: "mongo", uriEnvKey: "MERCHANT_MONGO_URI", poolSizeEnvKey: "MERCHANT_MONGO_POOLSIZE" },
	{ alias: "ticket", type: "mongo", uriEnvKey: "TICKET_MONGO_URI", poolSizeEnvKey: "TICKET_MONGO_POOLSIZE" },
	{ alias: "core-trans-proxy", type: "mongo", uriEnvKey: "CORE_TRANS_PROXY_MONGO_URI", poolSizeEnvKey: "CORE_TRANS_PROXY_MONGO_POOLSIZE" },
// ...
	RECONCILE__CENTRALIZED_RECONCILIATION: "centralized_reconciliation",
	RULE__RULE: "rules",
	TICKET__TICKET_CATEGORIES: "ticket-categories",
	TICKET__TICKET_HISTORY: "ticket-history",
	TRANS_HIS__TRANS_HIS: "trans-his",
	TRANS_HIS_PG__TRANSACTION_HISTORY: "public.transaction_history",
	TRANS_HIS_PG__TRANSACTION_HISTORY_PAYLOAD: "public.transaction_history_payload"
};
```

#### 2. `svc-env.ts:60-88`
```ts
// Hiện trạng tại svc-env.ts:60-88
	CONFIG_MONGO_URI?: string;
	CONFIG_MONGO_POOLSIZE?: number;
// ...
	TRANS_HIS_EXPORT_SOURCE?: string;
	TRANS_HIS_PG_HOST?: string;

// Đề xuất cập nhật tại svc-env.ts
	CONFIG_MONGO_URI?: string;
	CONFIG_MONGO_POOLSIZE?: number;

	RULE_MONGO_URI?: string;
	RULE_MONGO_POOLSIZE?: number;
// ...
	TRANS_HIS_EXPORT_SOURCE?: string;
	TRANS_HIS_ROUTING_RULE_DOMAIN?: string;
	TRANS_HIS_ROUTING_CACHE_MS?: string;
	TRANS_HIS_PG_HOST?: string;
```

#### 3. File mới `data-transfers/entities/rule/rule.entity.ts` & cập nhật `data-transfers/entities/index.ts`
```ts
// File mới đề xuất: data-transfers/entities/rule/rule.entity.ts
import { Document } from "mongoose";

class RuleEntity extends Document {
    // @ts-ignore
    public _id: any;
    public name?: string;
    public domain?: string;
    public domainCode?: string;
    public activeFr?: string;
    public activeTo?: string;
    public priority?: number;
    public status?: string;
    public description?: string;
    public conditions?: Record<string, any>;
    public event?: Record<string, any>;
    public ruleType?: string;
    public ruleAction?: string;
    public isDefault?: boolean;
    public isActive?: boolean;
    public isDelete?: boolean;
    public extData?: any;
    public createdBy?: string;
    public lastUpdatedBy?: string;
    public createdAt?: Date;
    public lastUpdatedAt?: Date;
}

export = RuleEntity;
```

#### 4. File mới `models/rule/rule.model.ts` & cập nhật `models/index.ts`
```ts
// File mới đề xuất: models/rule/rule.model.ts
import { LoggerInstance } from "moleculer";
import { Connection, Schema } from "mongoose";
import { RuleEntity } from "../../data-transfers/entities";
import { DB_COLLECTION } from "../../defined/app-setting";

class RuleModel {
    private readonly logger: LoggerInstance;
    private readonly dbConnection: Connection;
    private readonly schema: Schema<RuleEntity>;
    private readonly model: any;
    private readonly collectionName: string = DB_COLLECTION.RULE__RULE;

    public constructor(
        dbConnection: Connection,
        logger: LoggerInstance,
        plugins = []
    ) {
        this.logger = logger;
        this.dbConnection = dbConnection;
        this.schema = new Schema<RuleEntity>(
            {
                name: { type: String, required: true },
                domain: { type: String, required: true },
                domainCode: { type: String, required: true },
                activeFr: { type: String, required: true },
                activeTo: { type: String, required: true },
                priority: { type: Number, default: 1 },
                status: { type: String, default: "" },
                description: { type: String, default: "" },
                conditions: { type: Object, default: {} },
                event: { type: Object, default: {} },
                ruleType: { type: String, default: "" },
                ruleAction: { type: String, default: "" },
                isDefault: { type: Boolean, default: false },
                isActive: { type: Boolean, default: true },
                isDelete: { type: Boolean, default: false },
                extData: { type: Object, default: null },
                createdBy: { type: String, default: "" },
                lastUpdatedBy: { type: String, default: "" }
            },
            {
                timestamps: { createdAt: "createdAt", updatedAt: "lastUpdatedAt" }
            }
        );
        plugins.forEach((plugin: any) => this.schema.plugin(plugin));
        this.schema.set("minimize", false);
        this.schema.set("toObject", { getters: true });
        this.schema.set("toJSON", { getters: true });

        if (this.dbConnection.models[this.collectionName]) {
            this.model = this.dbConnection.models[this.collectionName];
        } else {
            this.model = this.dbConnection.model<RuleEntity>(
                this.collectionName,
                this.schema
            );
        }
    }

    public find(filter: object) {
        return this.model.find(filter);
    }
}

export = RuleModel;
```

#### 5. File mới `utils/trans-his-routing-rule.ts` (Đánh giá `routing` rule theo chuẩn `rule-service`)
```ts
// File mới đề xuất: utils/trans-his-routing-rule.ts
import moment from "moment";

export const DEFAULT_TRANS_HIS_RULE_DOMAIN = "THR";
export const TRANS_HIS_EXPORT_CALLER = "centralized-export-service";
export const TRANS_HIS_EXPORT_DEFAULT_ACTION = "getListTransHisForManager";

const RULE_TYPE_ROUTING = "routing";
const RULE_ACTION_QUERY = "QUERY";
const RULE_STATUS_ACTIVATED = "ACTIVATED";

export function isIsoDateString(str: unknown): str is string {
	if (typeof str !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(str)) {
		return false;
	}
	const date = new Date(str);
	return !isNaN(date.valueOf()) && date.toISOString() === str;
}

/** Matches RuleMainLogic.getDefaultRuleFilter in rule-service/application/rule/actions/ruleMainLogic.ts:16-34 */
export function buildRoutingRuleFilter(ruleDomain: string, now: Date = new Date()): Record<string, any> {
	const nowIso = moment(now).utc().toISOString();
	return {
		ruleType: RULE_TYPE_ROUTING,
		activeFr: { $lte: nowIso },
		activeTo: { $gte: nowIso },
		ruleAction: { $ne: RULE_ACTION_QUERY },
		isActive: true,
		isDelete: false,
		status: RULE_STATUS_ACTIVATED,
		$or: [{ domain: ruleDomain }, { domainCode: ruleDomain }]
	};
}

function normalizeFactValue(val: unknown): any {
	return isIsoDateString(val) ? new Date(val).valueOf() : val;
}

function evaluateLeafCondition(cond: Record<string, any>, facts: Record<string, any>): boolean {
	const rawFact = facts[cond.fact];
	if (rawFact === undefined) {
		return false;
	}
	const factVal = normalizeFactValue(rawFact);
	const targetVal = cond.value;

	switch (cond.operator) {
		case "equal":
			return factVal === targetVal;
		case "notEqual":
			return factVal !== targetVal;
		case "in":
			return Array.isArray(targetVal) && targetVal.includes(factVal);
		case "notIn":
			return Array.isArray(targetVal) && !targetVal.includes(factVal);
		case "lessThan":
			return typeof factVal === "number" && factVal < Number(targetVal);
		case "lessThanInclusive":
			return typeof factVal === "number" && factVal <= Number(targetVal);
		case "greaterThan":
			return typeof factVal === "number" && factVal > Number(targetVal);
		case "greaterThanInclusive":
			return typeof factVal === "number" && factVal >= Number(targetVal);
		case "contains":
			return Array.isArray(factVal)
				? factVal.includes(targetVal)
				: typeof factVal === "string" && factVal.includes(String(targetVal));
		case "doesNotContain":
			return Array.isArray(factVal)
				? !factVal.includes(targetVal)
				: typeof factVal === "string" && !factVal.includes(String(targetVal));
		default:
			return false;
	}
}

export function evaluateConditionTree(node: any, facts: Record<string, any>): boolean {
	if (!node || typeof node !== "object") {
		return true;
	}
	if (Array.isArray(node.all)) {
		return node.all.every((child: any) => evaluateConditionTree(child, facts));
	}
	if (Array.isArray(node.any)) {
		return node.any.length === 0 ? true : node.any.some((child: any) => evaluateConditionTree(child, facts));
	}
	if ("fact" in node) {
		return evaluateLeafCondition(node, facts);
	}
	return true;
}

/** Matches RoutingRuleLogic.execute in rule-service/application/rule/processors/executeRoutingRule.processors.ts:57-76 */
export function evaluateRoutingRules(
	ruleList: any[],
	facts: Record<string, any> = { caller: TRANS_HIS_EXPORT_CALLER, action: TRANS_HIS_EXPORT_DEFAULT_ACTION }
): { connectorName?: string; ruleId?: string } {
	if (!Array.isArray(ruleList) || ruleList.length === 0) {
		return {};
	}
	const sorted = [...ruleList].sort((a, b) => (Number(b?.priority) || 1) - (Number(a?.priority) || 1));
	for (const rule of sorted) {
		if (evaluateConditionTree(rule?.conditions, facts)) {
			return {
				ruleId: rule?.event?.type,
				...(rule?.event?.params || {})
			};
		}
	}
	return {};
}
```

#### 6. Cập nhật `utils/trans-his-export-source.ts`
```ts
// Đề xuất cập nhật tại utils/trans-his-export-source.ts
import { SVC_ENV } from "../svc-env";
import {
	buildRoutingRuleFilter,
	DEFAULT_TRANS_HIS_RULE_DOMAIN,
	evaluateRoutingRules,
	TRANS_HIS_EXPORT_CALLER,
	TRANS_HIS_EXPORT_DEFAULT_ACTION
} from "./trans-his-routing-rule";

export type TransHisExportSource = "mongo" | "postgres";

const DEFAULT_CACHE_MS = 30000;
const RULE_SELECT_FIELDS = { _id: 1, priority: 1, conditions: 1, event: 1, activeFr: 1, activeTo: 1 };

let cachedDbSource: TransHisExportSource | null = null;
let cacheExpiresAt = 0;

function readCacheMs(value: string | undefined): number {
	const n = Number.parseInt(String(value ?? "").trim(), 10);
	return Number.isFinite(n) && n >= 0 ? n : DEFAULT_CACHE_MS;
}

function getEnvFallbackSource(): TransHisExportSource {
	const source = String(SVC_ENV.get().TRANS_HIS_EXPORT_SOURCE || "mongo").trim().toLowerCase();
	if (source !== "mongo" && source !== "postgres") {
		throw new Error(`Invalid TRANS_HIS_EXPORT_SOURCE "${source}": expected "mongo" or "postgres"`);
	}
	return source;
}

/** Test seam: reset cached DB routing decision. */
export const resetTransHisExportSourceState = (): void => {
	cachedDbSource = null;
	cacheExpiresAt = 0;
};

/** Resolve trans-his export source directly from rule-service's `rules` collection when RuleModel is available. */
export const refreshTransHisExportSource = async (
	mainProcess: any,
	action: string = TRANS_HIS_EXPORT_DEFAULT_ACTION
): Promise<TransHisExportSource> => {
	const ruleModel = mainProcess?.models?.RuleModel;
	if (!ruleModel || typeof ruleModel.find !== "function") {
		return getTransHisExportSource();
	}

	const env = SVC_ENV.get();
	const cacheMs = readCacheMs(env.TRANS_HIS_ROUTING_CACHE_MS);
	if (cacheMs > 0 && cachedDbSource !== null && cacheExpiresAt > Date.now()) {
		return cachedDbSource;
	}

	const domain = String(env.TRANS_HIS_ROUTING_RULE_DOMAIN || DEFAULT_TRANS_HIS_RULE_DOMAIN).trim();
	try {
		const filter = buildRoutingRuleFilter(domain);
		const query = ruleModel.find(filter);
		const ruleList = typeof query?.select === "function"
			? await query.select(RULE_SELECT_FIELDS).lean().exec()
			: await query;
		const matched = evaluateRoutingRules(ruleList, { caller: TRANS_HIS_EXPORT_CALLER, action });
		const resolved: TransHisExportSource = matched.connectorName === "v2" ? "postgres" : "mongo";
		cachedDbSource = resolved;
		cacheExpiresAt = cacheMs > 0 ? Date.now() + cacheMs : 0;
		return resolved;
	} catch (err: any) {
		mainProcess?.logger?.warn?.(
			`[trans-his-export-source] Failed to evaluate routing rule from DB (domain=${domain}), using ${cachedDbSource ?? "fallback"}: ${err?.message || err}`
		);
		if (cachedDbSource !== null) {
			return cachedDbSource;
		}
		return getEnvFallbackSource();
	}
};

/** Returns DB-resolved source when RuleModel has been evaluated, or falls back to TRANS_HIS_EXPORT_SOURCE ("mongo" default). */
export const getTransHisExportSource = (): TransHisExportSource => {
	if (cachedDbSource !== null) {
		return cachedDbSource;
	}
	return getEnvFallbackSource();
};

/** The model the trans-his exports need in the current source. */
export const transHisModelNames = (): string[] => (getTransHisExportSource() === "postgres" ? ["TransHisPgModel"] : ["TransHisModel"]);
```

#### 7. Cập nhật `domain/QueryHandlerFactory.ts:77-160`, `start.ts:302-308`, và `logics/processors/BaseExportProcessor.ts:151-160`
```ts
// Đề xuất cập nhật tại domain/QueryHandlerFactory.ts:77-84 & 151-160
export default class QueryHandlerFactory {
  private readonly handlers: Map<string, IQueryHandler> = new Map();
  private readonly transHisMongoHandlers: Map<string, IQueryHandler> = new Map();
  private readonly transHisPgHandlers: Map<string, IQueryHandler> = new Map();

  public getHandler(queryName: string): IQueryHandler | undefined {
    if (this.transHisMongoHandlers.has(queryName)) {
      return getTransHisExportSource() === "postgres"
        ? this.transHisPgHandlers.get(queryName)
        : this.transHisMongoHandlers.get(queryName);
    }
    return this.handlers.get(queryName);
  }
// ...
    // trans-his (keep Mongo and Postgres handlers separate, resolved dynamically via getTransHisExportSource)
    this.transHisMongoHandlers.set("GetAllTransHisExportQuery", new GetAllTransHisExportHandler(mainProcess));
    this.transHisMongoHandlers.set("GetListTransHisByTransIdQuery", new GetListTransHisByTransIdHandler(mainProcess));
    this.transHisPgHandlers.set("GetAllTransHisExportQuery", new GetAllTransHisExportPgHandler(mainProcess));
    this.transHisPgHandlers.set("GetListTransHisByTransIdQuery", new GetListTransHisByTransIdPgHandler(mainProcess));
  }
}
```
```ts
// Đề xuất cập nhật tại start.ts:302-308
            // Initialize MainProcess
            this.mainProcess = new MainProcess(this.broker, this.logger, connections, [paginate]);

            // Resolve initial trans-his export source from rule-service DB (if connected)
            await refreshTransHisExportSource(this.mainProcess);

            // Register query handlers
            this.queryHandlerFactory.registerHandlers(this.mainProcess);
```
```ts
// Đề xuất cập nhật tại logics/processors/BaseExportProcessor.ts:157-161
    try {
      await refreshTransHisExportSource(mainProcess);
      config = await this.getConfig();
      this.validatePrerequisites(mainProcess, config);
    } catch (validationError: any) {
```

---

## 11. [TASK-18] Giải Pháp Xử Lý Lỗi `Model Check Failed: Missing required models [TransHisPgModel]` (`updatedAt: 1793766004908`)

### 11.1. Phân Tích Nguyên Nhân Gốc Rễ (Root Cause Analysis — `VERIFIED`)

1. **Đối chứng thực tế giữa môi trường Local (`PID 54963`) và Instance sinh lỗi (`updatedAt: 1793766004908`):**
   - Trên máy Local (`PID 54963`), kiểm tra trực tiếp qua `centralized-export-service.healthCheckDetailed` (`EV-20261104T112848-9234`) và `check_pg_init.ts` (`EV-20261104T112632-f561`):
     - `trans-his-pg` (`10.200.186.110:5000`) đang **`CONNECTED`** -> `TransHisPgModel` **CÓ** trong `mainProcess.models`.
     - `reconcile` (`localhost:3306` MariaDB) đang **`DISCONNECTED`** -> `CentralReconModel` **KHÔNG** có trong `mainProcess.models`.
   - Ngược lại, trong payload lỗi anh nhận được (`updatedAt: "1793766004908"`):
     - Danh sách `Available models` có đủ **30 models của toàn bộ 15 DB còn lại** trong `DB_LIST`, bao gồm cả **`CentralReconModel`** (`reconcile` MariaDB kết nối thành công), và **chỉ thiếu duy nhất `TransHisPgModel`** (`trans-his-pg`, DB thứ 16 trong `DB_LIST`).
     - Đồng thời `getRequiredModelName()` (`utils/trans-his-export-source.ts:15`) lại đòi `["TransHisPgModel"]`, chứng tỏ trên instance thực thi job đó biến **`TRANS_HIS_EXPORT_SOURCE=postgres` đã được bật**, nhưng kết nối tới DB `trans-his-pg` ở bước khởi động `initService()` (`start.ts:271-291`) **đã bị lỗi và rơi vào khối `catch`**.

2. **Ba mắt xích gây ra lỗi trên Instance đó (`F18-1..F18-3`):**
   - **`F18-1` — Thiếu khai báo `TRANS_HIS_PG_*` trong `REQUIRED_KEYS` (`svc-env.ts:162-170`) & trên Vault/Env của Server:**
     - Ở `svc-env.ts:162-167`, bộ biến của MariaDB (`RECONCILE_MARIADB_HOST`, `PORT`, `DB`, `USER`, `PASS`, `POOLSIZE`) đều nằm trong `REQUIRED_KEYS`, nên nếu thiếu trên HashiCorp Vault (`security/vault.security.ts:19-29`), service sẽ báo ngay `MIS_ENV_KEY` lúc khởi động.
     - Ngược lại, 6 biến `TRANS_HIS_PG_HOST`, `TRANS_HIS_PG_PORT`, `TRANS_HIS_PG_DB`, `TRANS_HIS_PG_USER`, `TRANS_HIS_PG_PASS`, `TRANS_HIS_PG_POOLSIZE` (`defined/app-setting.ts:36-44`) **chưa được đưa vào `REQUIRED_KEYS`** trong `svc-env.ts:101-170`. Khi trên môi trường Server/Vault mới bật `TRANS_HIS_EXPORT_SOURCE=postgres` mà chưa thêm đủ 6 key `TRANS_HIS_PG_*` vào Vault của `centralized-export-service` (hoặc đặt tên lệch sang `POSTGRESQL_*` như bên `core-trans-proxy-history-service-v2/svc-env.ts:9-13`), `VaultSecurity.checkMissingEnvVariable()` vẫn cho qua, dẫn tới `DbConnectorFactory.connect` (`dbHandler/DbConnectorFactory.ts:97`) ném lỗi `Error: Postgres requires host & database` (hoặc trên nhánh `feat/trans-his-postgres-export-testing` `8f2bc6f`, `readTransHisPgConfig()` tại `dbHandler/trans-his-pg.ts:25-47` ném lỗi thiếu key hoặc `TRANS_HIS_PG_POOLSIZE < 2 * EXPORT_WORKER_CONCURRENCY`).
   - **`F18-2` — `start.ts:289-304` nuốt lỗi kết nối `trans-his-pg` lúc khởi động khi `TRANS_HIS_EXPORT_SOURCE=postgres` (Split-Brain giữa `MainProcess` và `getTransHisExportSource`):**
     - Khi `this.postgresDbHandler.createConnection` ném lỗi tại `start.ts:271-283`, khối `catch (error)` tại `start.ts:289-291` chỉ log `Failed to connect DB 'trans-his-pg'` rồi bỏ qua, không gán `connections["trans-his-pg"]`.
     - `start.ts:298-300` chỉ kiểm tra `if (!connections.default)` mà không kiểm tra `if (getTransHisExportSource() === "postgres" && !connections["trans-his-pg"])`.
     - Hậu quả: `MainProcess` (`data-transfers/main-process.ts:44-91`) không khởi tạo `TransHisPgModel`, nhưng `QueryHandlerFactory` (`domain/QueryHandlerFactory.ts:152-158`) và `transHisModelNames()` (`utils/trans-his-export-source.ts:15`) vẫn ép dùng `TransHisPgModel` theo biến tĩnh `TRANS_HIS_EXPORT_SOURCE=postgres`, làm cho pod vẫn chạy `ExportWorker` hút job từ Redis queue rồi đánh `FAILED` toàn bộ job `trans-his` với lỗi `Model Check Failed: Missing required models [TransHisPgModel]`.
   - **`F18-3` — `dbHandler/postgres.ts:27-35` chưa ép kiểu `Number(...)` cho `port` và `poolSize`:**
     - Giá trị đọc từ `SVC_ENV.get()` (`process.env` hoặc Vault) thường ở dạng `string` (`"5000"`, `"5"`). Trong `dbHandler/postgres.ts:30,34`, `port: options.port || 5432` và `poolSize: options.poolSize || 10` truyền thẳng chuỗi vào `pg.Pool({ port, max })` thay vì ép kiểu `Number(...)`.

---

### 11.2. Phương Án Duy Nhất & Code Demo (`S-FIX-18` — Ánh xạ 1-1 `F18-1..F18-3`)

#### 1. `svc-env.ts:162-177` (`S-FIX-18A` — Khai báo `TRANS_HIS_PG_*` vào `REQUIRED_KEYS` đối xứng 1-1 với `RECONCILE_MARIADB_*`)
```ts
// Hiện tại tại svc-env.ts:162-170
	"RECONCILE_MARIADB_HOST",
	"RECONCILE_MARIADB_PORT",
	"RECONCILE_MARIADB_DB",
	"RECONCILE_MARIADB_USER",
	"RECONCILE_MARIADB_PASS",
	"RECONCILE_MARIADB_POOLSIZE",

	"EXPORT_STORAGE_PATH"
];

// Đề xuất cập nhật tại svc-env.ts:162-177
	"RECONCILE_MARIADB_HOST",
	"RECONCILE_MARIADB_PORT",
	"RECONCILE_MARIADB_DB",
	"RECONCILE_MARIADB_USER",
	"RECONCILE_MARIADB_PASS",
	"RECONCILE_MARIADB_POOLSIZE",

	"TRANS_HIS_PG_HOST",
	"TRANS_HIS_PG_PORT",
	"TRANS_HIS_PG_DB",
	"TRANS_HIS_PG_USER",
	"TRANS_HIS_PG_PASS",
	"TRANS_HIS_PG_POOLSIZE",

	"EXPORT_STORAGE_PATH"
];
```

#### 2. `dbHandler/postgres.ts:27-36` (`S-FIX-18B` — Chuẩn hóa kiểu `Number` cho `port` và `poolSize`)
```ts
// Hiện tại tại dbHandler/postgres.ts:27-36
        const config: DbConfig = {
            type: DbType.POSTGRES,
            host: options.host!,
            port: options.port || 5432,
            database: options.database!,
            username: options.username,
            password: options.password,
            poolSize: options.poolSize || 10
        };

// Đề xuất cập nhật tại dbHandler/postgres.ts:27-36
        const config: DbConfig = {
            type: DbType.POSTGRES,
            host: options.host!,
            port: Number(options.port) || 5432,
            database: options.database!,
            username: options.username,
            password: options.password,
            poolSize: Number(options.poolSize) || 10
        };
```

#### 3. `start.ts:24-25` & `start.ts:298-304` (`S-FIX-18C` — Fail-Fast khi `TRANS_HIS_EXPORT_SOURCE=postgres` nhưng DB `trans-his-pg` không kết nối được)
```ts
// Hiện tại tại start.ts:24-25 & 298-304
import { DB_LIST } from "./defined/app-setting";
...
            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }

            // Initialize MainProcess
            this.mainProcess = new MainProcess(this.broker, this.logger, connections, [paginate]);

// Đề xuất cập nhật tại start.ts:24-26 & 298-308
import { DB_LIST } from "./defined/app-setting";
import { getTransHisExportSource } from "./utils/trans-his-export-source";
...
            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }
            if (getTransHisExportSource() === "postgres" && !connections["trans-his-pg"]) {
                throw new Error("CRITICAL: TRANS_HIS_EXPORT_SOURCE=postgres but 'trans-his-pg' database failed to connect!");
            }

            // Initialize MainProcess
            this.mainProcess = new MainProcess(this.broker, this.logger, connections, [paginate]);
```

---

## 12. [TASK-19] Giải Thích Vì Sao Cập Nhật Password Postgres Mà Không Restart Service Thì Không Connect Lại & Giải Pháp Tự Động Reload Config + Lazy Reconnect (`S-FIX-19`)

### 12.1. Phân Tích 3 Nguyên Nhân Gốc Rễ Trong Code Hiện Tại (`VERIFIED`)

1. **`F19-1` — `SVC_ENV` (`svc-env.ts:182-199`) và `VaultSecurity` (`security/vault.security.ts:6-17`) chỉ nạp biến môi trường / Vault 1 lần duy nhất lúc khởi động process:**
   - Khi chạy lệnh `moleculer-runner ... --envfile .run.local.env start.ts`, `moleculer-runner` chỉ đọc file `.run.local.env` vào `process.env` **1 lần lúc khởi tạo process**.
   - Tiếp đó, `loadEnvAndInitService()` (`start.ts:315-324`) gọi `SVC_ENV.setEnvironments()` và `await VaultSecurity.overWriteVaultEnv()` **1 lần duy nhất** để chép sang biến `_env` trong RAM (`svc-env.ts:172`).
   - Khi anh sửa lại `TRANS_HIS_PG_PASS` (trong `.run.local.env` hoặc trên Vault) mà **không restart service**, biến `_env["TRANS_HIS_PG_PASS"]` trong RAM của process đang chạy vẫn giữ nguyên mật khẩu sai ban đầu (vì `--hot` của Moleculer chỉ watch các file code `.ts`/`.js`, không đọc lại `.run.local.env` và không gọi lại Vault).
2. **`F19-2` — Vòng lặp kết nối `DB_LIST` (`start.ts:245-304`) và `MainProcess.initializeModels` (`data-transfers/main-process.ts:27-30, 43-94`) chỉ chạy 1 lần lúc boot:**
   - Lúc mới boot mà bị sai password Postgres, `this.postgresDbHandler.createConnection` ném lỗi `password authentication failed`, bị khối `catch (error)` tại `start.ts:289-291` bắt và bỏ qua.
   - Do đó `connections["trans-his-pg"]` bị khuyết, và `new MainProcess(...)` (`start.ts:303`) bỏ qua luôn `trans-his-pg`, dẫn tới **`this.models["TransHisPgModel"]` chưa bao giờ được khởi tạo**.
   - Ngoài ra, tại `dbHandler/DbConnectorFactory.ts:182-196` (`connectPostgres`), khi `await pool.query("SELECT 1")` ném lỗi sai password, instance `pool` vừa tạo không được gọi `await pool.end()` trước khi rethrow.
3. **`F19-3` — `BaseExportProcessor.execute` (`logics/processors/BaseExportProcessor.ts:157-160`) và `HealthCheckLogic.checkDetailed` (`logics/health-check.logic.ts:27-45`) không có cơ chế tự kết nối lại (Lazy Reconnect & Model Init) khi thiếu model:**
   - Khi job chạy vào `BaseExportProcessor.execute()`, nó gọi thẳng `this.validatePrerequisites(mainProcess, config)` (`BaseExportProcessor.ts:57-79`).
   - Hàm `validatePrerequisites` thấy `!mainProcess.models["TransHisPgModel"]` là lập tức đánh `FAILED` (`Model Check Failed: Missing required models [TransHisPgModel]`) mà không hề thử reload lại `.envfile`/Vault và kết nối lại DB `trans-his-pg` để khởi tạo `TransHisPgModel`.

---

### 12.2. Bảng Đấu Nối Điểm Gọi Thực Thi & Vòng Đời Kích Hoạt (`L014` — Explicit Runtime Call-Site Wiring)

| Hàm / Phương thức mới | Vị trí định nghĩa (`file:line`) | Điểm gọi thực thi (`Caller file:line`) | Thời điểm kích hoạt (`Lifecycle / Event Trigger`) |
| :--- | :--- | :--- | :--- |
| `SVC_ENV.reloadEnvFile()` | `svc-env.ts:174-215` | Được gọi bên trong `SVC_ENV.setEnvironments()` (`svc-env.ts:183`) | Khi khởi động (`start.ts:317`) và mỗi khi `reconnectDb(alias)` chạy lại để nạp lại `--envfile` (`.run.local.env`) từ đĩa vào `process.env` (`dotenv.config({ path, override: true })`). |
| `CentralizedExportService.connectDatabase(config)` | `start.ts:227-262` | (1) `initService()` (`start.ts:270`)<br>(2) Callback `reconnectDb(alias)` truyền vào `new MainProcess(...)` (`start.ts:294-305`) | (1) Lúc khởi động process.<br>(2) Khi `mainProcess.ensureModels()` hoặc `mainProcess.ensureDbConnected()` yêu cầu kết nối lại một `alias` bị lỗi lúc boot. |
| `MainProcess.ensureModels(rawRequired)` | `data-transfers/main-process.ts:44-95` | `BaseExportProcessor.execute()` (`logics/processors/BaseExportProcessor.ts:159`) | Ngay trước `this.validatePrerequisites(mainProcess, config)` của mọi job export: nếu model đã có sẵn thì trả về ngay (`0ms`); nếu thiếu model (như `TransHisPgModel`), tự động gọi `reconnectDb(alias)` (có single-flight lock & cooldown `5000ms` theo `L035`) để nạp lại password mới, kết nối DB và khởi tạo `this.models[modelName]`. |
| `MainProcess.ensureDbConnected(alias)` | `data-transfers/main-process.ts:56-95` | `HealthCheckLogic.checkDetailed()` (`logics/health-check.logic.ts:34`) | Khi gọi `healthCheckDetailed`: nếu một `alias` trong `DB_LIST` đang `DISCONNECTED`, thử tự phục hồi kết nối (với cooldown `5000ms` chống bão theo `L035`). |

---

### 12.3. Phương Án Duy Nhất & Code Demo Nguyên Văn Từ HEAD (`S-FIX-19`)

#### 1. `svc-env.ts:1-4` & `180-200` — Tự động đọc lại `--envfile` (`.run.local.env`) bằng `dotenv` khi gọi `setEnvironments`
```ts
// Hiện tại tại svc-env.ts:1-4 & 182-199
// eslint-disable-next-line no-underscore-dangle
import { ServiceBroker } from "moleculer";
import _ from "lodash";
...
	setEnvironments(broker: ServiceBroker, serviceName: string) {
		let envServices = JSON.parse(JSON.stringify(process.env));

		const brokerEnv = broker.envServices ? broker.envServices[serviceName] : null;
		if (brokerEnv && !_.isEmpty(brokerEnv)) {
			envServices = { ...envServices, ...brokerEnv };
		}

		const serviceEnv = broker.services.find(x => x.name === serviceName)?.settings?.envServices;
		if (serviceEnv && !_.isEmpty(serviceEnv)) {
			envServices = { ...envServices, ...serviceEnv };
		}

		Object.keys(envServices).forEach(key => {
			SVC_ENV.set(key, envServices[key] || "");
		});
		return Promise.resolve();
	}

// Đề xuất cập nhật tại svc-env.ts:1-6 & 182-212
// eslint-disable-next-line no-underscore-dangle
import * as fs from "fs";
import * as dotenv from "dotenv";
import { ServiceBroker } from "moleculer";
import _ from "lodash";
...
	setEnvironments(broker: ServiceBroker, serviceName: string) {
		const envFileIdx = process.argv.indexOf("--envfile");
		const envFilePath = envFileIdx >= 0 ? process.argv[envFileIdx + 1] : ".run.local.env";
		if (envFilePath && fs.existsSync(envFilePath)) {
			dotenv.config({ path: envFilePath, override: true });
		}

		let envServices = JSON.parse(JSON.stringify(process.env));

		const brokerEnv = broker.envServices ? broker.envServices[serviceName] : null;
		if (brokerEnv && !_.isEmpty(brokerEnv)) {
			envServices = { ...envServices, ...brokerEnv };
		}

		const serviceEnv = broker.services.find(x => x.name === serviceName)?.settings?.envServices;
		if (serviceEnv && !_.isEmpty(serviceEnv)) {
			envServices = { ...envServices, ...serviceEnv };
		}

		Object.keys(envServices).forEach(key => {
			SVC_ENV.set(key, envServices[key] || "");
		});
		return Promise.resolve();
	}
```

#### 2. `dbHandler/DbConnectorFactory.ts:182-203` & `dbHandler/postgres.ts:27-36` — Đóng `pool` ngay khi `SELECT 1` lỗi (sai pass) và chuẩn hóa kiểu `Number`
```ts
// Hiện tại tại dbHandler/DbConnectorFactory.ts:182-203
  private async connectPostgres(config: DbConfig): Promise<DbConnection> {
    const pool = new Pool({
      host: config.host!,
      port: config.port || 5432,
      database: config.database!,
      user: config.username,
      password: config.password,
      max: config.poolSize || 10, // Use poolSize
      connectionTimeoutMillis: 10000 // fail a connect instead of waiting for the OS
    });
    // an idle connection dropped by the proxy is an 'error' event of the pool: without a listener it would stop the process
    pool.on("error", err => this.logger.error(`Postgres pool (${config.host}:${config.port}/${config.database}) idle client error:`, err.message));

    await pool.query("SELECT 1");

    return {
      type: DbType.POSTGRES,
      close: () => pool.end(),
      query: async (sql: string, params?: any[]) => pool.query(sql, params),
      rawPool: pool
    };
  }

// Đề xuất cập nhật tại dbHandler/DbConnectorFactory.ts:182-208
  private async connectPostgres(config: DbConfig): Promise<DbConnection> {
    const pool = new Pool({
      host: config.host!,
      port: Number(config.port) || 5432,
      database: config.database!,
      user: config.username,
      password: config.password,
      max: Number(config.poolSize) || 10, // Use poolSize
      connectionTimeoutMillis: 10000 // fail a connect instead of waiting for the OS
    });
    // an idle connection dropped by the proxy is an 'error' event of the pool: without a listener it would stop the process
    pool.on("error", err => this.logger.error(`Postgres pool (${config.host}:${config.port}/${config.database}) idle client error:`, err.message));

    try {
      await pool.query("SELECT 1");
    } catch (err) {
      await pool.end().catch(() => undefined);
      throw err;
    }

    return {
      type: DbType.POSTGRES,
      close: () => pool.end(),
      query: async (sql: string, params?: any[]) => pool.query(sql, params),
      rawPool: pool
    };
  }
```

#### 3. `data-transfers/main-process.ts:11-97` — Bổ sung `ensureModels` và `ensureDbConnected` (Single-Flight + Cooldown `5000ms` theo `L035`)
```ts
// Hiện tại tại data-transfers/main-process.ts:11-97
export default class MainProcess {
  public models: any = {};
  public broker: ServiceBroker;
  public logger: LoggerInstance;
  private connections: MultiDbConnections;

  public constructor(
    broker: ServiceBroker,
    logger?: LoggerInstance,
    connections?: MultiDbConnections,
    plugins?: any[]
  ) {
    this.broker = broker;
    this.logger = logger || broker.logger;
    this.connections = connections || { default: undefined };

    if (plugins) {
      this.initializeModels(plugins);
    }
  }
...

// Đề xuất cập nhật tại data-transfers/main-process.ts:11-145
const RECONNECT_COOLDOWN_MS = 5000;

export default class MainProcess {
  public models: any = {};
  public broker: ServiceBroker;
  public logger: LoggerInstance;
  private connections: MultiDbConnections;
  private plugins: any[];
  private reconnectDb?: (alias: string) => Promise<any>;
  private reconnectInFlight: Map<string, Promise<boolean>> = new Map();
  private lastReconnectAt: Map<string, number> = new Map();

  public constructor(
    broker: ServiceBroker,
    logger?: LoggerInstance,
    connections?: MultiDbConnections,
    plugins?: any[],
    reconnectDb?: (alias: string) => Promise<any>
  ) {
    this.broker = broker;
    this.logger = logger || broker.logger;
    this.connections = connections || { default: undefined };
    this.plugins = plugins || [];
    this.reconnectDb = reconnectDb;

    if (plugins) {
      this.initializeModels(plugins);
    }
  }

  public isDbConnected(alias: string = "default"): boolean {
    const conn = this.connections[alias];
    if (conn && "readyState" in conn) {
      return (conn as any).readyState === 1;
    }
    if (conn && "ended" in conn) {
      return !(conn as any).ended;
    }
    return !!conn;
  }

  public async ensureDbConnected(alias: string): Promise<boolean> {
    if (this.isDbConnected(alias)) {
      return true;
    }
    if (!this.reconnectDb) {
      return false;
    }
    const existing = this.reconnectInFlight.get(alias);
    if (existing) {
      return existing;
    }
    const lastAttempt = this.lastReconnectAt.get(alias) || 0;
    if (Date.now() - lastAttempt < RECONNECT_COOLDOWN_MS) {
      return false;
    }

    const promise = (async () => {
      this.lastReconnectAt.set(alias, Date.now());
      try {
        this.logger.info(`[MainProcess] Attempting on-demand reconnect for DB '${alias}'...`);
        const conn = await this.reconnectDb!(alias);
        if (conn) {
          this.connections[alias] = conn;
          this.initializeAliasModels(alias, this.plugins);
          return this.isDbConnected(alias);
        }
        return false;
      } catch (err: any) {
        this.logger.error(`[MainProcess] On-demand reconnect failed for DB '${alias}':`, err?.message || err);
        return false;
      } finally {
        this.reconnectInFlight.delete(alias);
      }
    })();

    this.reconnectInFlight.set(alias, promise);
    return promise;
  }

  public async ensureModels(rawRequired: string | string[]): Promise<void> {
    const requiredModels = Array.isArray(rawRequired) ? rawRequired : [rawRequired];
    for (const modelName of requiredModels) {
      if (this.models[modelName]) {
        continue;
      }
      const alias = this.findAliasByModelName(modelName);
      if (alias) {
        await this.ensureDbConnected(alias);
      }
    }
  }

  private findAliasByModelName(modelName: string): string | undefined {
    return Object.keys(ModelRegistry).find(alias =>
      (ModelRegistry[alias] || []).some(
        ModelClass => (ModelClass.name || ModelClass.prototype?.constructor?.name) === modelName
      )
    );
  }

  private initializeModels(plugins: any[]) {
    Object.keys(this.connections).forEach(alias => {
      this.initializeAliasModels(alias, plugins);
    });

    this.logger.info(`MainProcess initialization completed. Total models: ${Object.keys(this.models).length}`);
  }

  private initializeAliasModels(alias: string, plugins: any[]) {
    const connection = this.connections[alias];

    let isReady = false;
    if (connection) {
      if ("readyState" in connection) {
        isReady = (connection as any).readyState === 1;
      } else if ("ended" in connection) {
        isReady = !(connection as any).ended;
      } else {
        isReady = true;
      }
    }

    if (!isReady) {
      this.logger.warn(`[MainProcess] Connection '${alias}' is not ready. Skipping models.`);
      return;
    }

    const ModelClasses = ModelRegistry[alias];

    if (!ModelClasses || ModelClasses.length === 0) {
      this.logger.debug(`[MainProcess] No models registered for DB alias '${alias}'.`);
      return;
    }

    this.logger.info(`[MainProcess] Initializing ${ModelClasses.length} models for DB '${alias}'...`);

    ModelClasses.forEach(ModelClass => {
      try {
        const modelName = ModelClass.name || ModelClass.prototype.constructor.name;

        if (connection.models && connection.models[modelName]) {
          this.logger.warn(`[MainProcess] Model '${modelName}' already exists in DB '${alias}'. Skipping re-initialization.`);
          return;
        }

        if (this.models[modelName]) {
          this.logger.warn(`[MainProcess] Duplicate model name '${modelName}'. Overwriting! Check your naming.`);
        }

        this.models[modelName] = new ModelClass(connection, this.logger, plugins);

      } catch (err) {
        this.logger.error(`[MainProcess] Failed to init model for DB '${alias}':`, err);
      }
    });
  }
}
```

#### 4. `start.ts:245-308` — Tách `connectDatabase(config)` và truyền callback `reconnectDb` (kèm reload `SVC_ENV` & `VaultSecurity`) vào `MainProcess`
```ts
// Hiện tại tại start.ts:245-308
            for (const config of DB_LIST) {
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
                        connection = await new Promise((resolve, reject) => {
                            this.postgresDbHandler.createConnection({
                                alias,
                                host: SVC_ENV.get()[config.hostEnvKey],
                                port: SVC_ENV.get()[config.portEnvKey],
                                database: SVC_ENV.get()[config.dbEnvKey],
                                username: SVC_ENV.get()[config.userEnvKey],
                                password: SVC_ENV.get()[config.passEnvKey],
                                poolSize: SVC_ENV.get()[config.poolSizeEnvKey] || 10
                            }, (err, pool) => err ? reject(err) : resolve(pool));
                        });
                    }
                    if (connection) {
                        connections[alias] = connection;
                        successCount++;
                        this.logger.info(`DB '${alias}' (${config.type}) connected.`);
                    }
                } catch (error) {
                    this.logger.error(`Failed to connect DB '${alias}':`, error.message);
                }

            }

            this.logger.info(`Connected to ${successCount}/${DB_LIST.length} databases.`);
            this.logger.info("Connections object keys before MainProcess instantiation:", Object.keys(connections));

            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }

            // Initialize MainProcess
            this.mainProcess = new MainProcess(this.broker, this.logger, connections, [paginate]);

// Đề xuất cập nhật tại start.ts:245-315
            for (const config of DB_LIST) {
                const alias = config.alias;
                try {
                    const connection = await this.connectDatabase(config);
                    if (connection) {
                        connections[alias] = connection;
                        successCount++;
                        this.logger.info(`DB '${alias}' (${config.type}) connected.`);
                    }
                } catch (error) {
                    this.logger.error(`Failed to connect DB '${alias}':`, error.message);
                }
            }

            this.logger.info(`Connected to ${successCount}/${DB_LIST.length} databases.`);
            this.logger.info("Connections object keys before MainProcess instantiation:", Object.keys(connections));

            if (!connections.default) {
                throw new Error("CRITICAL: Default database failed to connect!");
            }

            // Initialize MainProcess with on-demand config reload & DB reconnect callback
            this.mainProcess = new MainProcess(
                this.broker,
                this.logger,
                connections,
                [paginate],
                async (alias: string) => {
                    const dbConfig = DB_LIST.find(item => item.alias === alias);
                    if (!dbConfig) {
                        return null;
                    }
                    await SVC_ENV.setEnvironments(this.broker, ServiceName);
                    await VaultSecurity.overWriteVaultEnv();
                    return await this.connectDatabase(dbConfig);
                }
            );
```
*(Hàm `private async connectDatabase(config: any): Promise<any>` đóng gói đúng 3 nhánh `mongo` / `mariadb` / `postgres` ở `start.ts:249-283`).*

#### 5. `logics/processors/BaseExportProcessor.ts:157-161` & `logics/health-check.logic.ts:33-38` — Kích hoạt Lazy Reconnect trước khi kiểm tra Model / HealthCheck
```ts
// Hiện tại tại logics/processors/BaseExportProcessor.ts:157-161
    try {
      config = await this.getConfig();
      this.validatePrerequisites(mainProcess, config);
    } catch (validationError: any) {

// Đề xuất cập nhật tại logics/processors/BaseExportProcessor.ts:157-164
    try {
      config = await this.getConfig();
      if (typeof mainProcess?.ensureModels === "function") {
        await mainProcess.ensureModels(this.getRequiredModelName());
      }
      this.validatePrerequisites(mainProcess, config);
    } catch (validationError: any) {
```
```ts
// Hiện tại tại logics/health-check.logic.ts:33-38
            try {
                const isConnected = this.mainProcess.isDbConnected(alias);
                dbStatus[alias] = {
                    status: isConnected ? "CONNECTED" : "DISCONNECTED",
                    models
                };

// Đề xuất cập nhật tại logics/health-check.logic.ts:33-42
            try {
                let isConnected = this.mainProcess.isDbConnected(alias);
                if (!isConnected && typeof (this.mainProcess as any).ensureDbConnected === "function") {
                    isConnected = await (this.mainProcess as any).ensureDbConnected(alias);
                }
                dbStatus[alias] = {
                    status: isConnected ? "CONNECTED" : "DISCONNECTED",
                    models
                };
```

---

## 12. [TASK-20] Phân Tích Nguyên Nhân Lệch Số Lượng Bản Ghi Giữa `InternalTransferExport` Và `/portal/accountant/internal-transfer` (`dateFr=2026-08-31T17:00:00.000Z` & `dateTo=2026-09-30T16:59:59.999Z`) & Giải Pháp Đồng Bộ (`S-FIX-20`, `Propose-Only`)

### 12.1. Số Liệu Thực Tế Trên PostgreSQL (`public.transaction_history`) & MongoDB (`trans-his`) (`VERIFIED` 100% qua `EV-20261104T135843-fc70`, `EV-20261104T135930-3e56`, `EV-20261104T141421-e01f`)

Trong khoảng thời gian `dateFr = 2026-08-31T17:00:00.000Z` đến `dateTo = 2026-09-30T16:59:59.999Z` (tháng 09/2026 GMT+7), số lượng bản ghi thực tế của 5 loại giao dịch chuyển khoản nội bộ (`INTERNAL_TRANSFER_TRANS_TYPES` trên `admin-portal-web`) trên cả PostgreSQL và MongoDB khớp nhau 100% như sau:

| `transType` | `sysTrans = true` | `sysTrans = false` (hoặc `!= true`) | Tổng cộng (`All`) |
| :--- | ---: | ---: | ---: |
| `INTERNAL_BANK_TRANSFER` | `1,543` | `0` | **`1,543`** |
| `REFUND_CASHIN` | `21` | `22` | **`43`** |
| `CASHOUT_MERCHANT` | `17` | `0` | **`17`** |
| `WALLET_UNMATCH_CASHIN` | `0` | `0` | **`0`** |
| `WALLET_UNMATCH_REFUND` | `0` | `0` | **`0`** |
| **Tổng 5 loại (`INTERNAL_TRANSFER_TRANS_TYPES`)** | **`1,581`** | **`22`** | **`1,603`** |
| **Tổng 2 loại mặc định của Export (`INTERNAL_BANK_TRANSFER`, `REFUND_CASHIN`)** | **`1,564`** | **`22`** | **`1,586`** |

### 12.2. Bảng Đối Chiếu 3 Nguyên Nhân Gốc Rễ (`F20-1..F20-3`) $\leftrightarrow$ Giải Pháp Duy Nhất (`S-FIX-20A..S-FIX-20D`)

| Mã phát hiện | Tệp và dòng (`file:line`) | Nguyên nhân gốc rễ (`Root Cause`) | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :--- | :--- | :--- |
| **`F20-1`** | - `admin-portal-web/src/pages/internal-transfer-transaction/list/index.tsx:32-42, 210-222` (commit `ee7b10ae`)<br>- `core-trans-proxy-history-service/logics/transHis.manage.logic.ts:35-36`<br>- `core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1196-1198`<br>- `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:14-22, 95-97` | **Lệch điều kiện mặc định `sysTrans` khi không truyền `params.sysTrans` (`22` bản ghi trên List vs `1,603` bản ghi trên Export):**<br>1. Trên `admin-portal-web`, commit `ee7b10ae` (`#fix remove sysTrans`, ngày `12/08/2026`) đã xóa `sysTrans: true` khỏi cả `getTransactionList` và `onExportExcel`.<br>2. Tại Service Danh sách (`getListTransHisForManager`), cả `v1` (`transHis.manage.logic.ts:36`) và `v2` (`transaction-history.repository.ts:1197`) đều khởi tạo mặc định `sysTrans: { $ne: true }` (`core.sys_trans IS DISTINCT FROM TRUE`). Do đó khi FE không gửi `sysTrans`, danh sách loại bỏ toàn bộ `1,581` bản ghi có `sysTrans = true` và chỉ trả về **`22` bản ghi** (`REFUND_CASHIN` có `sysTrans = false`).<br>3. Ngược lại, tại `InternalTransferExport` (`internal-transfer-export.pure.ts:14-22, 95-97`), `buildFilter` **không khởi tạo mặc định `sysTrans: { $ne: true }`** (chỉ gán `filter.sysTrans` khi `"sysTrans" in params`). Khi FE không gửi `sysTrans`, `InternalTransferExport` lấy cả `sysTrans = true` lẫn `sysTrans = false` $\rightarrow$ xuất ra **`1,603` bản ghi** (lệch **`1,581` bản ghi** so với danh sách!). | **`S-FIX-20A`:**<br>- Tại `internal-transfer-export.pure.ts:15`: khởi tạo mặc định `sysTrans: { $ne: true }` giống hệt `trans-his-export.pure.ts:19` và `getListTransHisForManager`.<br>- *(Lưu ý đồng bộ FE `admin-portal-web`)*: Vì 100% giao dịch `INTERNAL_BANK_TRANSFER` (`1,543` bản ghi) và `CASHOUT_MERCHANT` (`17` bản ghi) có `sysTrans = true`, nếu nghiệp vụ màn hình `/portal/accountant/internal-transfer` cần hiển thị và xuất đủ cả `1,603` giao dịch nội bộ (thay vì chỉ `22` bản ghi `REFUND_CASHIN`), cần khôi phục `sysTrans: true` (và truyền `customerId` vào `onExportExcel`) tại `admin-portal-web/src/pages/internal-transfer-transaction/list/index.tsx:37, 217`. |
| **`F20-2`** | - `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:16-21, 95-118`<br>- `centralized-export-service/utils/constants.ts:424-444, 477-590`<br>- `core-trans-proxy-history-service/logics/transHis.manage.logic.ts:206-226`<br>- `core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1098-1108` | **Lệch danh sách `transType` mặc định (`1,586` vs `1,603`) và lệch nhánh miễn trừ `NON_SYS_TRANS_TYPES` khi `sysTrans` bật:**<br>1. Khi không truyền `params.transType`, `internal-transfer-export.pure.ts:16-21` chỉ mặc định 2 loại `[INTERNAL_BANK_TRANSFER, REFUND_CASHIN]` (**`1,586` bản ghi**), thiếu 3 loại `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT` (`17` bản ghi). Ngoài ra `utils/constants.ts:424-444` cũng chưa khai báo 3 hằng số này.<br>2. Khi `sysTrans` được truyền (`"true"` hoặc `true`), Service Danh sách (`v1:207-226`, `v2:1098-1108`) miễn trừ `sysTrans = true` cho cả **3 loại** `[REFUND_CASHIN, WALLET_UNMATCH_CASHIN, WALLET_UNMATCH_REFUND]`, trong khi `internal-transfer-export.pure.ts:100-118` chỉ miễn trừ duy nhất `REFUND_CASHIN` (`hasRefundCashin`), làm rớt các bản ghi `WALLET_UNMATCH_CASHIN` / `WALLET_UNMATCH_REFUND` (`sysTrans = false`).<br>3. Dòng `108` so sánh cứng `sysTrans === "true"`, nên nếu `params.sysTrans` là boolean `true` thì nhánh miễn trừ không chạy, làm rớt mất **`22` bản ghi** `REFUND_CASHIN` (`sysTrans = false`); đồng thời khi `otherTransTypes.length === 0` (ví dụ chỉ lọc `transType = "REFUND_CASHIN"` — có **`43` bản ghi** chia làm 3 trang, trang 3 `current=3&pageSize=20` có 3 bản ghi), dòng `110-115` vẫn sinh nhánh thừa `{ transType: { $in: [] }, sysTrans: "true" }`. | **`S-FIX-20B`:**<br>- Bổ sung `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT` vào `TRANS_TYPE` và `TRANSACTION_TYPE_LIST` tại `utils/constants.ts:424-444, 477-590`.<br>- Cập nhật `transType` mặc định trong `internal-transfer-export.pure.ts:16-25` đủ 5 loại.<br>- Mở rộng `sysTransIgnoredTransTypes` trong `internal-transfer-export.pure.ts` cho cả 3 loại `[REFUND_CASHIN, WALLET_UNMATCH_CASHIN, WALLET_UNMATCH_REFUND]`, chấp nhận cả `sysTrans === "true" \|\| sysTrans === true`, và khi `otherTransTypes.length === 0` thì chỉ giữ `{ ...restFilter, transType: { $in: sysTransIgnoredTransTypes } }` khớp 1-1 với `v2:1103-1107`. |
| **`F20-3`** | - `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:36-42`<br>- `centralized-export-service/data-transfers/params/trans-his/trans-his-export.params.ts:32-60`<br>- `core-trans-proxy-history-service/logics/transHis.manage.logic.ts:74-88, 198-203`<br>- `core-trans-proxy-history-service-v2/dbHandler/postgres/transaction-history.repository.ts:1128-1148, 1214-1221` | **Lệch bộ lọc `partnerCode`, thiếu bộ lọc `customerId`, `phone`, `serviceCode`, `channelId`, và thiếu khai báo DTO trong `TransHisExportParams`:**<br>1. `partnerCode`: Service Danh sách (`v1:74-76`, `v2:1145-1148`) lọc exact match `filter["info.partnerCode"] = params.partnerCode` (`core.partner_code = $n`), còn `internal-transfer-export.pure.ts:36-42` lọc `$or` regex trên cả `info.partnerCode` lẫn `info.bankTransferResponse.partnerCode` (và gán đè `filter.$or`).<br>2. `internal-transfer-export.pure.ts` chưa xử lý `params.customerId`, `params.phone`, `params.serviceCode`, `params.channelId` (trong khi `getTransactionList` trên FE gửi `customerId` từ URL query vào `getListTransHisForManager`).<br>3. `TransHisExportParams` (`trans-his-export.params.ts:32-60`) chưa khai báo `@IsOptional() @IsString()` cho `senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`, `serviceCode`, `channelId`. | **`S-FIX-20C` & `S-FIX-20D`:**<br>- Tại `internal-transfer-export.pure.ts`: đổi `params.partnerCode` về exact match `filter["info.partnerCode"] = params.partnerCode;` và bổ sung xử lý `customerId`, `phone`, `serviceCode`, `channelId` khớp 1-1 với `getListTransHisForManager` và `trans-his-export.pure.ts:55-63`.<br>- Tại `trans-his-export.params.ts:41-54`: bổ sung `@IsOptional() @IsString()` cho `senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`, `serviceCode`, `channelId`. |

### 12.3. Code Demo Nguyên Văn Từ HEAD (`Propose-Only` — Chờ `APPROVE`)

#### 1. `centralized-export-service/utils/constants.ts:424-444` — Bổ sung `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT`
```ts
// Hiện tại tại utils/constants.ts:424-444
export const TRANS_TYPE = {
	PAY: "PAY",
	CASHIN: "CASHIN",
	CASHOUT: "CASHOUT",
	MERCHANT_WITHDRAW: "MERCHANT_WITHDRAW",
	TRANSFER: "TRANSFER",
	REFUND: "REFUND",
	REFUND_CASHIN: "REFUND_CASHIN",
	TRANSFER_SENDER: "TRANSFER_SENDER",
	TRANSFER_RECEIVER: "TRANSFER_RECEIVER",
	MERCHANT_REVENUE: "MERCHANT_REVENUE",
	ADJUST: "ADJUST",
	PAYOUT_WALLET: "PAYOUT_WALLET",
	PAYOUT_WALLET_SETTLE: "PAYOUT_WALLET_SETTLE",
	BILLING: "BILLING",
	TOPUP: "TOPUP",
	SOFTPIN: "SOFTPIN",
	DISBURSEMENT: "DISBURSEMENT",
	BANK_TRANSFER: "BANK_TRANSFER",
	INTERNAL_BANK_TRANSFER: "INTERNAL_BANK_TRANSFER"
};

// Đề xuất cập nhật tại utils/constants.ts:424-447
export const TRANS_TYPE = {
	PAY: "PAY",
	CASHIN: "CASHIN",
	WALLET_UNMATCH_CASHIN: "WALLET_UNMATCH_CASHIN",
	CASHOUT: "CASHOUT",
	CASHOUT_MERCHANT: "CASHOUT_MERCHANT",
	MERCHANT_WITHDRAW: "MERCHANT_WITHDRAW",
	TRANSFER: "TRANSFER",
	REFUND: "REFUND",
	REFUND_CASHIN: "REFUND_CASHIN",
	WALLET_UNMATCH_REFUND: "WALLET_UNMATCH_REFUND",
	TRANSFER_SENDER: "TRANSFER_SENDER",
	TRANSFER_RECEIVER: "TRANSFER_RECEIVER",
	MERCHANT_REVENUE: "MERCHANT_REVENUE",
	ADJUST: "ADJUST",
	PAYOUT_WALLET: "PAYOUT_WALLET",
	PAYOUT_WALLET_SETTLE: "PAYOUT_WALLET_SETTLE",
	BILLING: "BILLING",
	TOPUP: "TOPUP",
	SOFTPIN: "SOFTPIN",
	DISBURSEMENT: "DISBURSEMENT",
	BANK_TRANSFER: "BANK_TRANSFER",
	INTERNAL_BANK_TRANSFER: "INTERNAL_BANK_TRANSFER"
};
```

#### 2. `centralized-export-service/data-transfers/params/trans-his/trans-his-export.params.ts:41-55` — Khai báo đầy đủ các trường filter của `InternalTransferExport` & `TransHisExport`
```ts
// Hiện tại tại data-transfers/params/trans-his/trans-his-export.params.ts:41-55
    @IsOptional() @IsString() public transId?: string;
    @IsOptional() @IsString() public transType?: string; // Ví dụ: "PAY,CASHOUT"
    @IsOptional() @IsString() public status?: string; // Ví dụ: "SUCCESS,FAILED"
    @IsOptional() @IsString() public partnerCode?: string;
    @IsOptional() @IsString() public customerId?: string;
    @IsOptional() @IsString() public phone?: string;
    @IsOptional() @IsString() public sysTrans?: string;
    @IsOptional() @IsString() public transIdRef?: string;
    @IsOptional() @IsString() public info?: string; // JSON string for complex filtering
    @IsOptional() @IsString() public walletPayOnly?: string;

    @IsOptional() @IsString() public sortBy?: string;
    @IsOptional() @IsString() public sortType?: string;

// Đề xuất cập nhật tại data-transfers/params/trans-his/trans-his-export.params.ts:41-61
    @IsOptional() @IsString() public transId?: string;
    @IsOptional() @IsString() public transType?: string; // Ví dụ: "PAY,CASHOUT"
    @IsOptional() @IsString() public status?: string; // Ví dụ: "SUCCESS,FAILED"
    @IsOptional() @IsString() public partnerCode?: string;
    @IsOptional() @IsString() public customerId?: string;
    @IsOptional() @IsString() public phone?: string;
    @IsOptional() @IsString() public senderAccount?: string;
    @IsOptional() @IsString() public receiverAccount?: string;
    @IsOptional() @IsString() public connector?: string;
    @IsOptional() @IsString() public originalTransHisId?: string;
    @IsOptional() @IsString() public serviceCode?: string;
    @IsOptional() @IsString() public channelId?: string;
    @IsOptional() @IsString() public sysTrans?: string;
    @IsOptional() @IsString() public transIdRef?: string;
    @IsOptional() @IsString() public info?: string; // JSON string for complex filtering
    @IsOptional() @IsString() public walletPayOnly?: string;

    @IsOptional() @IsString() public sortBy?: string;
    @IsOptional() @IsString() public sortType?: string;
```

#### 3. `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:14-121` — Đồng bộ 1-1 `buildFilter` với `getListTransHisForManager`
```ts
// Hiện tại tại logics/export/trans-his/internal-transfer-export.pure.ts:14-121
const buildFilter = (params: any) => {
    let filter: any = {
        transType: {
            $in: [
                TRANS_TYPE.INTERNAL_BANK_TRANSFER,
                TRANS_TYPE.REFUND_CASHIN
            ]
        }
    };

    if (params.transType) {
        const arrayTransType = params.transType.split(",");
        filter.transType = { $in: arrayTransType };
    }
    if (params.transId) {
        filter.transId = params.transId;
    }
    if (params.status) {
        const arrayStatus = params.status.split(",");
        filter.status = { $in: arrayStatus };
    }

    if (params.partnerCode) {
        const keyword = params.partnerCode.trim();
        filter.$or = [
            { "info.partnerCode": { $regex: keyword, $options: "i" } },
            { "info.bankTransferResponse.partnerCode": { $regex: keyword, $options: "i" } }
        ];
    }

    if (params.senderAccount) {
        const keyword = params.senderAccount.trim();
        filter.$expr = {
            $regexMatch: {
                input: {
                    $ifNull: [
                        "$info.bankAccountNumber",
                        "$sender.bankAccount"
                    ]
                },
                regex: keyword,
                options: "i"
            }
        };
    }
    if (params.receiverAccount) {
        filter["receiver.credit"] = {
            $regex: params.receiverAccount.trim(),
            $options: "i"
        };
    }
    if (params.connector) {
        const connector = params.connector;
        filter.$expr = {
            $eq: [
                {
                    $ifNull: [
                        "$info.bankConnector",
                        "$info.connector"
                    ]
                },
                connector
            ]
        };
    }
    if (params.originalTransHisId) {
        filter.originalTransHisId = {
            $regex: params.originalTransHisId.trim(),
            $options: "i"
        };
    }

    if ("dateFr" in params || "dateTo" in params) {
        filter.createdAt = {};
    }
    if ("dateFr" in params) {
        filter.createdAt.$gte = new Date(params.dateFr);
    }
    if ("dateTo" in params) {
        filter.createdAt.$lte = new Date(params.dateTo);
    }
    if ("sysTrans" in params) {
        filter.sysTrans = params.sysTrans;
    }

    const arrTransTypes = params.transType?.split(",") || [];
    const hasRefundCashin = arrTransTypes.includes(TRANS_TYPE.REFUND_CASHIN);
    const otherTransTypes = arrTransTypes.filter((t: string) => t !== TRANS_TYPE.REFUND_CASHIN);
    const { sysTrans, ...restFilter } = filter;

    if (
        "transType" in params &&
        hasRefundCashin &&
        "sysTrans" in filter &&
        sysTrans === "true"
    ) {
        filter.transType = { $in: otherTransTypes };
        const newFilter = {
            $or: [
                { ...restFilter, transType: TRANS_TYPE.REFUND_CASHIN },
                { ...filter }
            ]
        };
        filter = _.cloneDeep(newFilter);
    }

    return filter;
};

// Đề xuất cập nhật tại logics/export/trans-his/internal-transfer-export.pure.ts:14-142
const NON_SYS_TRANS_TYPES = [
    TRANS_TYPE.REFUND_CASHIN,
    TRANS_TYPE.WALLET_UNMATCH_CASHIN,
    TRANS_TYPE.WALLET_UNMATCH_REFUND
];

const buildFilter = (params: any) => {
    let filter: any = {
        sysTrans: { $ne: true },
        transType: {
            $in: [
                TRANS_TYPE.INTERNAL_BANK_TRANSFER,
                TRANS_TYPE.REFUND_CASHIN,
                TRANS_TYPE.WALLET_UNMATCH_CASHIN,
                TRANS_TYPE.WALLET_UNMATCH_REFUND,
                TRANS_TYPE.CASHOUT_MERCHANT
            ]
        }
    };

    if (params.transType) {
        const arrayTransType = params.transType.split(",");
        filter.transType = { $in: arrayTransType };
    }
    if (params.transId) {
        filter.transId = params.transId;
    }
    if (params.status) {
        const arrayStatus = params.status.split(",");
        filter.status = { $in: arrayStatus };
    }

    if (params.partnerCode) {
        filter["info.partnerCode"] = params.partnerCode;
    }
    if (params.customerId) {
        filter.$or = [
            { "sender.customerId": params.customerId },
            { "receiver.customerId": params.customerId }
        ];
    }
    if (params.phone) {
        filter.$or = [
            { "sender.phone": params.phone },
            { "receiver.phone": params.phone }
        ];
    }

    if (params.senderAccount) {
        const keyword = params.senderAccount.trim();
        filter.$expr = {
            $regexMatch: {
                input: {
                    $ifNull: [
                        "$info.bankAccountNumber",
                        "$sender.bankAccount"
                    ]
                },
                regex: keyword,
                options: "i"
            }
        };
    }
    if (params.receiverAccount) {
        filter["receiver.credit"] = {
            $regex: params.receiverAccount.trim(),
            $options: "i"
        };
    }
    if (params.connector) {
        const connector = params.connector;
        filter.$expr = {
            $eq: [
                {
                    $ifNull: [
                        "$info.bankConnector",
                        "$info.connector"
                    ]
                },
                connector
            ]
        };
    }
    if (params.originalTransHisId) {
        filter.originalTransHisId = {
            $regex: params.originalTransHisId.trim(),
            $options: "i"
        };
    }
    if (params.serviceCode) {
        filter.serviceCode = params.serviceCode;
    }
    if (params.channelId) {
        filter.channelId = params.channelId;
    }

    if ("dateFr" in params || "dateTo" in params) {
        filter.createdAt = {};
    }
    if ("dateFr" in params) {
        filter.createdAt.$gte = new Date(params.dateFr);
    }
    if ("dateTo" in params) {
        filter.createdAt.$lte = new Date(params.dateTo);
    }
    if ("sysTrans" in params && params.sysTrans !== undefined && params.sysTrans !== "") {
        filter.sysTrans = params.sysTrans;
    }

    const arrTransTypes: string[] = filter.transType?.$in || [];
    const sysTransIgnoredTransTypes = arrTransTypes.filter((t: string) =>
        NON_SYS_TRANS_TYPES.includes(t)
    );
    const otherTransTypes = arrTransTypes.filter((t: string) =>
        !NON_SYS_TRANS_TYPES.includes(t)
    );
    const { sysTrans, ...restFilter } = filter;

    if (
        sysTransIgnoredTransTypes.length > 0 &&
        (sysTrans === "true" || sysTrans === true)
    ) {
        if (otherTransTypes.length > 0) {
            filter.transType = { $in: otherTransTypes };
            filter = _.cloneDeep({
                $or: [
                    { ...restFilter, transType: { $in: sysTransIgnoredTransTypes } },
                    { ...filter }
                ]
            });
        } else {
            filter = _.cloneDeep({
                ...restFilter,
                transType: { $in: sysTransIgnoredTransTypes }
            });
        }
    }

    return filter;
};
```

---

## 13. [TASK-22] Audit Phản Biện Toàn Diện `TASK-20` / `TASK-21` & Giải Pháp Khắc Phục Kiến Trúc / Nghiệp Vụ (`S-FIX-22`, `Propose-Only`)

### 13.1. Số Liệu Thực Tế Đầy Đủ Cho Cả 6 Loại Giao Dịch Nội Bộ Trên PostgreSQL & MongoDB (`VERIFIED` 100% qua `EV-20261104T162208-0f23`)

Khi mở rộng đối chứng sang nhánh hiện hành của `admin-portal-web` (`src/constants/transaction.constant.ts:253-299`, sau commit `aa0d6af4` — `GP2-3176: display TAC-TAM batch disbursement settlement transactions` và commit `ee7b10ae`), danh sách `INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST` trên FE có **6 loại giao dịch** (không phải 5 loại). Số liệu thực tế trên cả PostgreSQL (`public.transaction_history`) và MongoDB (`trans-his`) trong khoảng `dateFr = 2026-08-31T17:00:00.000Z .. dateTo = 2026-09-30T16:59:59.999Z` như sau:

| `transType` | `sysTrans = true` | `sysTrans = false` (hoặc `!= true`) | Tổng cộng (`All`) |
| :--- | ---: | ---: | ---: |
| `INTERNAL_BANK_TRANSFER` | `1,543` | `0` | **`1,543`** |
| `REFUND_CASHIN` | `21` | `22` | **`43`** |
| `CASHOUT_MERCHANT` | `17` | `0` | **`17`** |
| `WALLET_UNMATCH_CASHIN` | `0` | `0` | **`0`** |
| `WALLET_UNMATCH_REFUND` | `0` | `0` | **`0`** |
| **`BATCH_DISBURSEMENT_SETTLE`** *(bổ sung ở `GP2-3176`)* | **`0`** | **`69`** | **`69`** |
| **Tổng 6 loại trên giao diện `/portal/accountant/internal-transfer`** | **`1,581`** | **`91`** | **`1,672`** |

> **Giải mã hoàn chỉnh URL của anh (`/portal/accountant/internal-transfer?current=3&pageSize=20&dateFr=2026-08-31T17:00:00.000Z&dateTo=2026-09-30T16:59:59.999Z`):**
> - Trên giao diện thực tế, FE gửi `transType` gồm **6 loại** và không gửi `sysTrans`.
> - API Danh sách (`getListTransHisForManager`) lọc `sysTrans != true` $\rightarrow$ trả về chính xác **`22` (`REFUND_CASHIN`) + `69` (`BATCH_DISBURSEMENT_SETTLE`) = `91` bản ghi** (chia làm **5 trang** với `pageSize=20`, đó là lý do vì sao anh đang đứng ở trang 3 `current=3&pageSize=20`!).
> - Trước `TASK-21`, `InternalTransferExport` không có mặc định `sysTrans: { $ne: true }` nên lấy cả `1,581` bản ghi `sysTrans = true` $\rightarrow$ xuất ra **`1,672` bản ghi** thay vì **`91` bản ghi**!
> - Sau `TASK-21`, khi FE truyền `params.transType` (6 loại), `InternalTransferExport` đã xuất đúng **`91` bản ghi** (`listAndExportWith6TypesNoSysPg = 91`, `listAndExportWith6TypesNoSysMongo = 91`). Tuy nhiên, nếu gọi `InternalTransferExport` mà **không truyền `params.transType`**, mảng mặc định trong `internal-transfer-export.pure.ts:24-30` mới có 5 loại (thiếu `BATCH_DISBURSEMENT_SETTLE`) nên chỉ xuất **`22` bản ghi** thay vì **`91` bản ghi**!

### 13.2. Bảng Đối Chiếu 4 Phát Hiện Phản Biện (`F22-1..F22-4`) $\leftrightarrow$ Giải Pháp Tối Ưu Duy Nhất (`S-FIX-22A..S-FIX-22D`)

| Mã phát hiện | Tệp và dòng (`file:line`) | Nội dung Phản biện (`Adversarial Critique`) | Giải pháp tối ưu duy nhất (`1-to-1 Solution`) |
| :--- | :--- | :--- | :--- |
| **`F22-1`** | - `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:14-31, 239-246`<br>- `centralized-export-service/utils/constants.ts:424-447, 477-590`<br>- `admin-portal-web/src/constants/transaction.constant.ts:253-299` | **Thiếu loại giao dịch thứ 6 `BATCH_DISBURSEMENT_SETTLE` (`69` bản ghi) ở mảng mặc định và bỏ sót bảng nhãn dịch `TRANSACTION_TYPE_LIST` so với chính Plan `S-FIX-20B` (`L012`, `L033`):**<br>1. Trong Plan `S-FIX-20B` (`09_tasks_solution_trans_his_postgres_export.md:2117`), chúng ta ghi rõ sẽ cập nhật cả `TRANS_TYPE` lẫn `TRANSACTION_TYPE_LIST`, nhưng khi code `TASK-21` lại quên cập nhật `TRANSACTION_TYPE_LIST`.<br>2. Mảng mặc định trong `internal-transfer-export.pure.ts:24-30` thiếu `BATCH_DISBURSEMENT_SETTLE` (`69` bản ghi `sysTrans = false`), khiến trường hợp không truyền `params.transType` chỉ ra `22` bản ghi thay vì `91` bản ghi (`EV-20261104T162208-0f23`).<br>3. Do thiếu nhãn trong `TRANSACTION_TYPE_LIST`, `transformRow` (`internal-transfer-export.pure.ts:239-246`) xuất mã thô `"CASHOUT_MERCHANT"` và `"BATCH_DISBURSEMENT_SETTLE"` ở cột `"Loại giao dịch"` thay vì nhãn tiếng Việt/Anh (`"Chi hộ"`, `"Quyết toán Chi hộ theo Batch"`). | **`S-FIX-22A`:**<br>- Bổ sung đầy đủ 6 loại giao dịch (`INTERNAL_BANK_TRANSFER`, `REFUND_CASHIN`, `WALLET_UNMATCH_CASHIN`, `WALLET_UNMATCH_REFUND`, `CASHOUT_MERCHANT`, `BATCH_DISBURSEMENT_SETTLE`) vào danh sách mặc định `INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES`.<br>- Bổ sung `BATCH_DISBURSEMENT_SETTLE` vào `INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES` (vì 100% `69` bản ghi `BATCH_DISBURSEMENT_SETTLE` trên DB có `sysTrans = false`, nếu sau này bật `sysTrans = "true"` thì không bị lọc mất `69` bản ghi này).<br>- Khai báo đầy đủ nhãn `vi`/`en` cho cả 6 loại trong `INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST` khớp 1-1 với `admin-portal-web/src/constants/transaction.constant.ts:253-299`. |
| **`F22-2`** | - `centralized-export-service/utils/constants.ts:424-447` (`1,124` dòng vs `1,121` dòng trên `HEAD`)<br>- `centralized-export-service/utils/centralrecon.constants.ts:1-124`<br>- `centralized-export-service/data-transfers/params/trans-his/trans-his-export.params.ts:41-56`<br>- `centralized-export-service/data-transfers/params/index.ts:54-100` | **Vi phạm Single-Responsibility Principle (`L018`, Rule #12) & Layering khi nhồi thêm vào God-file `utils/constants.ts` và làm ô nhiễm DTO `TransHisExportParams`:**<br>1. `utils/constants.ts` đã dài `1,121` dòng trên `HEAD` (vượt ngưỡng `> 1,000` dòng của `L018`). Ở `TASK-21`, việc thêm 3 dòng vào `utils/constants.ts` nâng độ dài lên `1,124` dòng (vi phạm bất biến `NOW <= HEAD` của `L018`), trong khi codebase đã có sẵn pattern tách file `utils/centralrecon.constants.ts`.<br>2. Trong `data-transfers/params/`, mỗi nghiệp vụ Export đều có class DTO riêng. Việc nhồi 4 trường chỉ dành riêng cho `InternalTransferExport` (`senderAccount`, `receiverAccount`, `connector`, `originalTransHisId`) vào `TransHisExportParams` làm lệch ranh giới trách nhiệm giữa `TransHisExport` và `InternalTransferExport`. | **`S-FIX-22B`:**<br>- Khôi phục `utils/constants.ts` về nguyên trạng `1,121` dòng trên `HEAD` (`git diff utils/constants.ts` trống 100%, tuân thủ tuyệt đối `NOW <= HEAD`).<br>- Tạo file đơn trách nhiệm `utils/internal-transfer.constants.ts` chứa các hằng số và bảng nhãn dịch của riêng miền `InternalTransferExport`.<br>- Tách file DTO đơn trách nhiệm `data-transfers/params/trans-his/internal-transfer-export.params.ts` (`export class InternalTransferExportParams extends TransHisExportParams`), export qua `data-transfers/params/index.ts`, và rút 4 trường riêng khỏi `TransHisExportParams` (chỉ giữ `serviceCode`, `channelId` dùng chung cho `TransHisExport`). |
| **`F22-3`** | - `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts:160-162, 289`<br>- `centralized-export-service/logics/export/trans-his/trans-his-export.pure.ts:126`<br>- `centralized-export-service/logics/export/trans-his/trans-his-collection-export.pure.ts:114`<br>- `admin-portal-web/src/pages/internal-transfer-transaction/list/index.tsx:121-122` | **Lỗi đảo chiều sắp xếp khi `sortType = "1"` (kiểu chuỗi) và thiếu fallback `info.bankTransferData.credit` ở `transformRow`:**<br>1. Trong DTO, `@IsString() public sortType?: string` nhận query string từ HTTP. Tuy nhiên `buildSort` (`internal-transfer-export.pure.ts:161`, `trans-his-export.pure.ts:126`, `trans-his-collection-export.pure.ts:114`) lại viết `const order = sortType === "asc" || sortType === 1 ? 1 : -1;`. Khi truyền `sortType: "1"` (string), biểu thức trả về `-1` (DESC) thay vì `1` (ASC) (`EV-20261104T162208-0f23`)!<br>2. Tại `internal-transfer-export.pure.ts:289`, `receiverBankAccount` chỉ lấy `history.receiver?.credit || ""` mà thiếu fallback `history.info?.bankTransferData?.credit` như giao diện `admin-portal-web` (`list/index.tsx:121-122`). | **`S-FIX-22C`:**<br>- Cập nhật điều kiện `sortType` tại cả 3 file (`internal-transfer-export.pure.ts:161`, `trans-his-export.pure.ts:126`, `trans-his-collection-export.pure.ts:114`): `const order = sortType === "asc" || sortType === 1 || sortType === "1" ? 1 : -1;`.<br>- Bổ sung fallback tại `internal-transfer-export.pure.ts:289`: `receiverBankAccount: history.receiver?.credit || history.info?.bankTransferData?.credit || ""`. |
| **`F22-4`** | - Tiến trình `centralized-export-service` trên máy local | **Minh bạch 100% Trạng thái Tiến trình Runtime & Không Suy diễn (`L028`, `L031`):**<br>- Kiểm tra `ps -Ao pid,lstart,command` xác nhận hiện tại tiến trình `centralized-export-service` **không chạy trên máy local** (chỉ có `admin-portal-gateway`, `config-service`, `user-service`, `auth-service`, `customer-service`, `storage-gateway` đang chạy).<br>- Các biên lai `EV-20261104T160129-2fc6` và `EV-20261104T162208-0f23` đã kiểm chứng trực tiếp hàm thật `getConfig` + `compileFilter` + `transformRow` trên **PostgreSQL thật (`127.0.0.1:5432/trans_his_db`) và MongoDB thật (`127.0.0.1:27017/core-trans-proxy-history-service`)**, nhưng chưa gọi qua HTTP/RabbitMQ worker của service đang chạy. | **`S-FIX-22D`:**<br>- Báo cáo trung thực 100% ranh giới kiểm chứng DB thật vs Service Process.<br>- Sau khi anh `APPROVE` `S-FIX-22` và em hoàn tất cập nhật code, nếu anh muốn trace đầu-cuối file Excel xuất ra qua API `/api/v1/admin-portal/export-async?exportType=InternalTransferExport`, em sẽ phối hợp cùng anh bật tiến trình `centralized-export-service` local để trace runtime value thật. |

### 13.3. Code Demo Nguyên Văn Từ HEAD (`Propose-Only` — Chờ `APPROVE`)

#### 1. Khôi phục `centralized-export-service/utils/constants.ts:424-447` về nguyên bản `HEAD` (`1,121` dòng — `NOW <= HEAD`)
```ts
// Hiện tại (sau TASK-21) tại utils/constants.ts:424-447 (1,124 dòng)
export const TRANS_TYPE = {
	PAY: "PAY",
	CASHIN: "CASHIN",
	WALLET_UNMATCH_CASHIN: "WALLET_UNMATCH_CASHIN",
	CASHOUT: "CASHOUT",
	CASHOUT_MERCHANT: "CASHOUT_MERCHANT",
	MERCHANT_WITHDRAW: "MERCHANT_WITHDRAW",
	TRANSFER: "TRANSFER",
	REFUND: "REFUND",
	REFUND_CASHIN: "REFUND_CASHIN",
	WALLET_UNMATCH_REFUND: "WALLET_UNMATCH_REFUND",
	TRANSFER_SENDER: "TRANSFER_SENDER",
	TRANSFER_RECEIVER: "TRANSFER_RECEIVER",
	MERCHANT_REVENUE: "MERCHANT_REVENUE",
	ADJUST: "ADJUST",
	PAYOUT_WALLET: "PAYOUT_WALLET",
	PAYOUT_WALLET_SETTLE: "PAYOUT_WALLET_SETTLE",
	BILLING: "BILLING",
	TOPUP: "TOPUP",
	SOFTPIN: "SOFTPIN",
	DISBURSEMENT: "DISBURSEMENT",
	BANK_TRANSFER: "BANK_TRANSFER",
	INTERNAL_BANK_TRANSFER: "INTERNAL_BANK_TRANSFER"
};

// Đề xuất khôi phục utils/constants.ts:424-444 về nguyên trạng HEAD (1,121 dòng)
export const TRANS_TYPE = {
	PAY: "PAY",
	CASHIN: "CASHIN",
	CASHOUT: "CASHOUT",
	MERCHANT_WITHDRAW: "MERCHANT_WITHDRAW",
	TRANSFER: "TRANSFER",
	REFUND: "REFUND",
	REFUND_CASHIN: "REFUND_CASHIN",
	TRANSFER_SENDER: "TRANSFER_SENDER",
	TRANSFER_RECEIVER: "TRANSFER_RECEIVER",
	MERCHANT_REVENUE: "MERCHANT_REVENUE",
	ADJUST: "ADJUST",
	PAYOUT_WALLET: "PAYOUT_WALLET",
	PAYOUT_WALLET_SETTLE: "PAYOUT_WALLET_SETTLE",
	BILLING: "BILLING",
	TOPUP: "TOPUP",
	SOFTPIN: "SOFTPIN",
	DISBURSEMENT: "DISBURSEMENT",
	BANK_TRANSFER: "BANK_TRANSFER",
	INTERNAL_BANK_TRANSFER: "INTERNAL_BANK_TRANSFER"
};
```

#### 2. Tạo file mới đơn trách nhiệm `centralized-export-service/utils/internal-transfer.constants.ts` (theo đúng pattern `utils/centralrecon.constants.ts`)
```ts
import { TRANS_TYPE, TRANSACTION_TYPE_LIST } from "./constants";

export const INTERNAL_TRANSFER_TRANS_TYPE = {
	INTERNAL_BANK_TRANSFER: TRANS_TYPE.INTERNAL_BANK_TRANSFER,
	REFUND_CASHIN: TRANS_TYPE.REFUND_CASHIN,
	WALLET_UNMATCH_CASHIN: "WALLET_UNMATCH_CASHIN",
	WALLET_UNMATCH_REFUND: "WALLET_UNMATCH_REFUND",
	CASHOUT_MERCHANT: "CASHOUT_MERCHANT",
	BATCH_DISBURSEMENT_SETTLE: "BATCH_DISBURSEMENT_SETTLE"
} as const;

export const INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES: string[] = [
	INTERNAL_TRANSFER_TRANS_TYPE.INTERNAL_BANK_TRANSFER,
	INTERNAL_TRANSFER_TRANS_TYPE.REFUND_CASHIN,
	INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_CASHIN,
	INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_REFUND,
	INTERNAL_TRANSFER_TRANS_TYPE.CASHOUT_MERCHANT,
	INTERNAL_TRANSFER_TRANS_TYPE.BATCH_DISBURSEMENT_SETTLE
];

export const INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES: string[] = [
	INTERNAL_TRANSFER_TRANS_TYPE.REFUND_CASHIN,
	INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_CASHIN,
	INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_REFUND,
	INTERNAL_TRANSFER_TRANS_TYPE.BATCH_DISBURSEMENT_SETTLE
];

export const INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST = [
	...TRANSACTION_TYPE_LIST,
	{
		value: INTERNAL_TRANSFER_TRANS_TYPE.INTERNAL_BANK_TRANSFER,
		label: {
			vi: "Chuyển khoản nội bộ",
			en: "Internal Bank Transfer"
		}
	},
	{
		value: INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_CASHIN,
		label: {
			vi: "Nạp tiền vào ví (Unmatch)",
			en: "Unmatch cash in"
		}
	},
	{
		value: INTERNAL_TRANSFER_TRANS_TYPE.WALLET_UNMATCH_REFUND,
		label: {
			vi: "Hoàn tiền nạp vào ví (Unmatch)",
			en: "Unmatch cash-in refund"
		}
	},
	{
		value: INTERNAL_TRANSFER_TRANS_TYPE.CASHOUT_MERCHANT,
		label: {
			vi: "Chi hộ",
			en: "Cashout Merchant"
		}
	},
	{
		value: INTERNAL_TRANSFER_TRANS_TYPE.BATCH_DISBURSEMENT_SETTLE,
		label: {
			vi: "Quyết toán Chi hộ theo Batch",
			en: "Batch disbursement settlement"
		}
	}
];
```

#### 3. Tách DTO đơn trách nhiệm `centralized-export-service/data-transfers/params/trans-his/internal-transfer-export.params.ts`, làm sạch `trans-his-export.params.ts` và export tại `data-transfers/params/index.ts`
```ts
// File mới: data-transfers/params/trans-his/internal-transfer-export.params.ts
import { IsOptional, IsString } from "class-validator";
import { TransHisExportParams } from "./trans-his-export.params";

export class InternalTransferExportParams extends TransHisExportParams {
	@IsOptional() @IsString() public senderAccount?: string;
	@IsOptional() @IsString() public receiverAccount?: string;
	@IsOptional() @IsString() public connector?: string;
	@IsOptional() @IsString() public originalTransHisId?: string;
}
```
```ts
// Cập nhật tại data-transfers/params/trans-his/trans-his-export.params.ts:41-61 (chỉ giữ trường của TransHisExport)
	@IsOptional() @IsString() public transId?: string;
	@IsOptional() @IsString() public transType?: string; // Ví dụ: "PAY,CASHOUT"
	@IsOptional() @IsString() public status?: string; // Ví dụ: "SUCCESS,FAILED"
	@IsOptional() @IsString() public partnerCode?: string;
	@IsOptional() @IsString() public customerId?: string;
	@IsOptional() @IsString() public phone?: string;
	@IsOptional() @IsString() public serviceCode?: string;
	@IsOptional() @IsString() public channelId?: string;
	@IsOptional() @IsString() public sysTrans?: string;
	@IsOptional() @IsString() public transIdRef?: string;
	@IsOptional() @IsString() public info?: string; // JSON string for complex filtering
	@IsOptional() @IsString() public walletPayOnly?: string;

	@IsOptional() @IsString() public sortBy?: string;
	@IsOptional() @IsString() public sortType?: string;
```
```ts
// Cập nhật tại data-transfers/params/index.ts:54-56 & 98-100
// TRANS-HIS
import { InternalTransferExportParams } from "./trans-his/internal-transfer-export.params";
import { TransHisExportParams } from "./trans-his/trans-his-export.params";
...
	// trans-his
	InternalTransferExportParams,
	TransHisExportParams
};
```

#### 4. Cập nhật `centralized-export-service/logics/export/trans-his/internal-transfer-export.pure.ts` (và sửa `sortType === "1"` tại `trans-his-export.pure.ts:126`, `trans-his-collection-export.pure.ts:114`)
```ts
// Đề xuất cập nhật tại logics/export/trans-his/internal-transfer-export.pure.ts:6-34, 155-192, 239-292
import { InternalTransferExportParams } from "../../../data-transfers/params";
import { validateParams } from "../../../utils/validator";
import { TRANSACTION_STATUS_LIST } from "../../../utils/constants";
import {
    INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES,
    INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES,
    INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST
} from "../../../utils/internal-transfer.constants";

const buildFilter = (params: any) => {
    let filter: any = {
        sysTrans: { $ne: true },
        transType: {
            $in: [...INTERNAL_TRANSFER_DEFAULT_TRANS_TYPES]
        }
    };
...
    const arrTransTypes: string[] = filter.transType?.$in || [];
    const sysTransIgnoredTransTypes = arrTransTypes.filter((t: string) =>
        INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES.includes(t)
    );
    const otherTransTypes = arrTransTypes.filter((t: string) =>
        !INTERNAL_TRANSFER_NON_SYS_TRANS_TYPES.includes(t)
    );
...
const buildSort = (params: any): object => {
    let sort: any = {};
    const sortBy = params.sortBy;
    const sortType = params.sortType;

    if (sortBy && sortType) {
        const order = sortType === "asc" || sortType === 1 || sortType === "1" ? 1 : -1;
        sort[sortBy] = order;
    } else {
        sort = { createdAt: -1 };
    }
    return sort;
};
...
export const validate = (params: any, langCode: string = "vi"): InternalTransferExportParams =>
    validateParams<InternalTransferExportParams>(
        InternalTransferExportParams,
        params || {},
        langCode
    );
...
    const TRANSACTION_TYPE_MAP = new Map(
        INTERNAL_TRANSFER_TRANSACTION_TYPE_LIST.map((type: any) => [type.value, type])
    );
    const transTypeObj = TRANSACTION_TYPE_MAP.get(history.transType);
    const transTypeStr = transTypeObj?.label?.[langCode] || history.transType || "";
...
        receiverBankName: history.receiver?.bankCode || "",
        receiverBankAccount: history.receiver?.credit || history.info?.bankTransferData?.credit || "",
        receiverBankAccountName: history.receiver?.creditName || ""
```
```ts
// Đồng bộ sửa lỗi sortType === "1" tại logics/export/trans-his/trans-his-export.pure.ts:126
// và logics/export/trans-his/trans-his-collection-export.pure.ts:114
    if (sortBy && sortType) {
        const order = sortType === "asc" || sortType === 1 || sortType === "1" ? 1 : -1;
```






---

## 14. [TASK-24] Audit Lại Filter `InternalTransferExport` (`senderAccount`, `originalTransHisId`) & Tương Thích Ngược Nguồn `trans-his-pg` — Giải Pháp `S-FIX-24` (`Propose-Only`)

### 14.1. Bảng Phân Loại Từng Filter: Logic Nghiệp Vụ vs Lỗi (bằng chứng DB thật `EV-20261104T170616-a7fc`, `EV-20261104T170822-e9db`, `EV-20261104T170925-2a27`, `EV-20261104T171016-92c7`)

| Filter | Export hiện tại (`internal-transfer-export.pure.ts`) | List API v1 (`transHis.manage.logic.ts`) | Giá trị FE/Export hiển thị | Kết luận |
| :--- | :--- | :--- | :--- | :--- |
| `sysTrans` mặc định + miễn trừ `NON_SYS` | `:17`, `:108-139` | `:35-36`, `:206-226` | — | **LOGIC** (khớp List; PG==Mongo `91/91`, `sysTrans=true` `1672/1672`, `sysTrans=false` `91/91` qua `TransHisModel` thật) |
| `transType` mặc định 6 loại | `:18-20` | FE luôn gửi 6 loại | — | **LOGIC** |
| `transId`, `status`, `partnerCode` exact | `:27-37` | `:59-76` | — | **LOGIC** (PG==Mongo) |
| `customerId` / `phone` ghi đè `$or` | `:38-49` | `:77-88` (y hệt) | — | **LOGIC parity** với v1; màn hình này không có ô `phone` nên không chạm |
| `receiverAccount` | `receiver.credit` regex `:66-71` | `receiver.credit` regex **OR** `receiver.merchantId ∈ merchant(vaNo)` `:110-123` | `info.bankTransferData.credit \|\| receiver.credit` (DB: `creditDiffers = 0`) | **GAP đã biết** (nhánh VA merchant cần gọi merchant-service, export worker không có) — dữ liệu hiện tại không lệch giá trị hiển thị; không sửa trong `S-FIX-24` |
| `connector` | `$expr $eq $ifNull(bankConnector, connector)` `:72-85` | `:152-165` (y hệt) | `info.connector \|\| info.bankConnector` (DB: `connectorBothDiffer = 0`) | **LOGIC** |
| **`senderAccount`** | `$expr $regexMatch $ifNull(info.bankAccountNumber, sender.bankAccount)` `:51-65` | `:124-139` (y hệt) | REFUND_CASHIN: List gán `sender.bankAccount = info.dstBankAccount.bankAccount` (`:328-329`); Export in `info.dstBankAccount.bankAccount \|\| sender.bankAccount` (`:253`) | **LỖI E1**: REFUND_CASHIN `22/22` bản ghi có `info.bankAccountNumber ≠ info.dstBankAccount.bankAccount` (`8850855585` vs `99GP26110M00000117`). Gõ đúng số đang hiển thị → **`0/0`** (PG/Mongo) |
| **`senderAccount` + `connector`** | cả 2 cùng gán `filter.$expr` (`:53`, `:74`) | cùng lỗi (`:126`, `:154`) | — | **LỖI E2**: `connector=banvietbank` + `senderAccount=NOTEXIST` vẫn ra **`55`** (= chỉ connector) → senderAccount bị nuốt im lặng |
| **`originalTransHisId`** | chỉ `originalTransHisId` regex `:86-91` | `:140-145` (y hệt) | List hiển thị `originalTransHisId ?? info.refundFor` (`:306`); Export in `originalTransHisId \|\| info.refundFor` | **LỖI E3**: REFUND_CASHIN `22/22` (non-sys) + `21/21` (sys) có `originalTransHisId = null`, chỉ có `info.refundFor`. Gõ đúng mã hiển thị `260901WCVHHP` → **`0/0`** |

> [!NOTE]
> E1/E2/E3 tồn tại **y hệt ở List API v1** (filter object giống từng ký tự; chạy filter v1 trên Mongo với `260901WCVHHP` → `0`). Export đang "khớp List" nhưng **cả hai cùng sai so với giá trị hiển thị**.
> Tự phản biện: lần chạy `EV-20261104T170925-2a27` em dùng raw Mongo driver nên `sysTrans="true"/"false"` ra `112/0` (raw driver không cast chuỗi → boolean). Chạy lại qua `TransHisModel` thật (Mongoose 8, đúng như `GetAllTransHisExportHandler.ts:21,32`) → `1672/1672`, `91/91` (`EV-20261104T171016-92c7`). **Không phải lỗi.**

### 14.2. Tương Thích Ngược Nguồn `trans-his-pg`

| Hạng mục | Bằng chứng | Kết luận |
| :--- | :--- | :--- |
| Ma trận PG==Mongo code hiện tại (Internal 13 case, TransHis 3, Collection 2) | `EV-20261104T170925-2a27` + `EV-20261104T171016-92c7` | **OK toàn bộ**, không case nào bị `TransHisPgUnsupportedError` |
| `sortType === "1"` (TASK-23) ở `TransHisExport` / `TransHisCollectionExport` | FE ProTable chỉ đưa form params vào export (`ProTable/index.tsx:391-408`), sort đi qua `opts` → `http.service.ts:65` cho list; PG hỗ trợ ASC (`trans-his-pg.sort.ts:143-145,193-195`); parity `sortType=1` OK `1953/1953`, `2177/2177` | Không đổi hành vi với FE hiện tại; caller gửi `"1"` giờ được ASC đúng nghĩa |
| `TransHisExportParams` bỏ 4 field (TASK-23) | `validateParams` không whitelist (`utils/validator.ts:7-10`); `TransHisExport`/`Collection` không đọc 4 field đó | Không ảnh hưởng |
| **Rủi ro của chính `S-FIX-24`** | `compileFilter` từ chối mọi path không có trong whitelist (`trans-his-pg.query.ts:213-222`) | Nếu chỉ sửa `.pure.ts` thêm `info.refundFor` / `info.dstBankAccount.bankAccount` mà **không** thêm whitelist → job PG **FAILED** (`unsupported_key`). Bắt buộc ship 2 file cùng commit. Thay đổi whitelist là **additive** (không đổi SQL của filter cũ) |

### 14.3. Giải Pháp Duy Nhất `S-FIX-24` (ánh xạ 1-1)

| Lỗi | Sửa |
| :--- | :--- |
| E1 | `senderAccount` = `$or[ $expr(giữ nguyên shape cũ — giữ hành vi List), info.dstBankAccount.bankAccount regex ]` |
| E2 | Đưa điều kiện `senderAccount` ra khỏi `filter.$expr` top-level, gom vào `filter.$and` → không còn bị `connector` ghi đè |
| E3 | `originalTransHisId` = `$or[ originalTransHisId regex, info.refundFor regex ]` trong `filter.$and` (không đụng `$or` của `customerId`) |
| PG | Thêm 2 path `$regex` vào `JSON_STRING_PATHS` (`info.refundFor`, `info.dstBankAccount.bankAccount`) |

Prototype trên DB thật (`EV-20261104T170925-2a27`): `originalTransHisId=260901WCVHHP` → Mongo `1` / PG `1`; `senderAccount=99GP26110M00000117` → `22/22`; `senderAccount=8850855585` (giá trị cũ) → `22/22` (giữ tương thích); `senderAccount=0687041020972` (BATCH) → `33/33` (không đổi); `connector+senderAccount=NOTEXIST` → `0` (hết bị nuốt).

### 14.4. Code Demo (nguyên văn từ HEAD hiện tại)

#### 1. `logics/export/trans-his/internal-transfer-export.pure.ts:15-142`
```ts
// Hiện tại :51-65 và :86-91
    if (params.senderAccount) {
        const keyword = params.senderAccount.trim();
        filter.$expr = {
            $regexMatch: {
                input: {
                    $ifNull: [
                        "$info.bankAccountNumber",
                        "$sender.bankAccount"
                    ]
                },
                regex: keyword,
                options: "i"
            }
        };
    }
...
    if (params.originalTransHisId) {
        filter.originalTransHisId = {
            $regex: params.originalTransHisId.trim(),
            $options: "i"
        };
    }

// Đề xuất
const buildFilter = (params: any) => {
    let filter: any = { /* giữ nguyên :16-21 */ };
    const andConditions: any[] = [];
    ...
    if (params.senderAccount) {
        const keyword = params.senderAccount.trim();
        andConditions.push({
            $or: [
                {
                    $expr: {
                        $regexMatch: {
                            input: { $ifNull: ["$info.bankAccountNumber", "$sender.bankAccount"] },
                            regex: keyword,
                            options: "i"
                        }
                    }
                },
                // REFUND_CASHIN: số TK gửi hiển thị (List v1 :328-329, Export transformRow :253)
                { "info.dstBankAccount.bankAccount": { $regex: keyword, $options: "i" } }
            ]
        });
    }
    ... // receiverAccount, connector giữ nguyên
    if (params.originalTransHisId) {
        const keyword = params.originalTransHisId.trim();
        andConditions.push({
            $or: [
                { originalTransHisId: { $regex: keyword, $options: "i" } },
                // REFUND_CASHIN: mã GD gốc hiển thị (List v1 :306, Export transformRow :243)
                { "info.refundFor": { $regex: keyword, $options: "i" } }
            ]
        });
    }
    ... // serviceCode, channelId, createdAt, sysTrans giữ nguyên
    if (andConditions.length > 0) {
        filter.$and = andConditions;
    }
    // nhánh NON_SYS_TRANS_TYPES giữ nguyên: restFilter mang theo $and
```

#### 2. `models/trans-his-pg/trans-his-pg.query.ts:118-123`
```ts
const JSON_STRING_PATHS: Readonly<Record<string, { column: "info" | "sender"; segments: readonly string[]; operators: readonly string[] }>> = {
	"info.masterMerchantInfo.merchantEmail": { column: "info", segments: ["masterMerchantInfo", "merchantEmail"], operators: ["eq"] },
	"info.bankConnector": { column: "info", segments: ["bankConnector"], operators: ["eq"] },
	"info.bankTransferResponse.partnerCode": { column: "info", segments: ["bankTransferResponse", "partnerCode"], operators: ["$regex"] },
	// InternalTransferExport: REFUND_CASHIN keeps the original trans id / sender account here (shown by the list and the export file)
	"info.refundFor": { column: "info", segments: ["refundFor"], operators: ["$regex"] },
	"info.dstBankAccount.bankAccount": { column: "info", segments: ["dstBankAccount", "bankAccount"], operators: ["$regex"] }
};
```

#### 3. `test/unit/trans-his-pg/trans-his-pg.query.test.ts` — bổ sung case: `connector + senderAccount` cả 2 điều kiện cùng có trong SQL; `originalTransHisId` sinh `core.original_trans_his_id ... OR EXISTS(... refundFor ...)`; `sysTrans=true + senderAccount + originalTransHisId` compile không vượt `MAX_FILTER_DEPTH`.

### 14.5. Ngoài phạm vi repo export (cần anh quyết)
List API v1 (`transHis.manage.logic.ts:124-145`) và v2 (`transaction-history.repository.ts:1175-1193`) có **cùng E1/E2/E3**. Sau `S-FIX-24`, với 3 filter này Export sẽ ra nhiều hơn List (VD `originalTransHisId=260901WCVHHP`: Export `1`, List `0`) cho tới khi List được sửa tương ứng.
