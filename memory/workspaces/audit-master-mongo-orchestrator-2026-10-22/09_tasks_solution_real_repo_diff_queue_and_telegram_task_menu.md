---
tier: T2
suffix: real_repo_diff_queue_and_telegram_task_menu
stage: S3
---
# 📐 S3 Technical Solution & Code Demo: Real Repo Diff (`centralized-export-service`), Quản Lý Hàng Đợi Task (Theo Repo) & Telegram `/task` Click-to-Run

## 1. Phương Án Kỹ Thuật Duy Nhất & Tối Ưu Nhất (`Single Optimal Solution`)

### 1.1. `TASK-RDQ-01` — Thực thi chuyển đổi `trans_his` từ MongoDB sang PostgreSQL thật trên `/Users/trainguyen/Documents/work/centralized-export-service` & Đồng bộ `git diff` thật vào `S4 Implement`
1. **Triển khai code thật trong `/Users/trainguyen/Documents/work/centralized-export-service` (tạo `git diff` thực tế trên repo):**
   - **`dbHandler/postgres.ts` (`lines 1-61`):** Mở comment và chuẩn hóa class `PostgresDb` nhận `ConnectOptions` (`alias`, `host`, `port`, `database`, `username`, `password`, `poolSize`), gọi `DbConnectorFactory.getInstance(this.logger).connect(config, options.alias)` với `DbType.POSTGRES` và trả về `conn.rawPool as Pool` (đồng bộ chuẩn kiến trúc với `dbHandler/mariadb.ts`).
   - **`svc-env.ts` (`lines 40-42, 121-123`):** Bổ sung các biến môi trường PostgreSQL cho `trans-his`: `TRANS_HIS_POSTGRES_HOST`, `TRANS_HIS_POSTGRES_PORT`, `TRANS_HIS_POSTGRES_DB`, `TRANS_HIS_POSTGRES_USER`, `TRANS_HIS_POSTGRES_PASS`, `TRANS_HIS_POSTGRES_POOLSIZE`.
   - **`defined/app-setting.ts` (`lines 10-14`):** Chuyển cấu hình `alias: "trans-his"` trong `DB_LIST` sang `type: "postgres"` (kèm `hostEnvKey: "TRANS_HIS_POSTGRES_HOST"`, `portEnvKey: "TRANS_HIS_POSTGRES_PORT"`, `dbEnvKey: "TRANS_HIS_POSTGRES_DB"`, `userEnvKey: "TRANS_HIS_POSTGRES_USER"`, `passEnvKey: "TRANS_HIS_POSTGRES_PASS"`, `poolSizeEnvKey: "TRANS_HIS_POSTGRES_POOLSIZE"`, vẫn giữ `uriEnvKey: "TRANS_HIS_MONGO_URI"` làm fallback tương thích ngược).
   - **`start.ts` (`lines 6-7, 29-30, 233-268`):** Import `PostgresDb from "./dbHandler/postgres"`, khởi tạo `this.postgresDbHandler = new PostgresDb(this.logger)` và xử lý nhánh `else if (config.type === "postgres")` trong `initService()`.
   - **`models/trans-his/trans-his.model.ts` (`lines 1-94`):** Nâng cấp `TransHisModel` sử dụng `knex({ client: "pg" })` và `pg.Pool` để truy vấn bảng PostgreSQL (`trans_his` / `DB_COLLECTION.TRANS_HIS__TRANS_HIS`) với SQL parameterized (`find(filter).sort(sort).select(select).limit(limit).skip(cursor).lean().exec()`, `countDocuments(filter)`, `findOne(filter, select)`, `getAll(...)`), đồng thời vẫn tương thích ngược với Mongoose mock trong unit test hiện hữu.
2. **Sửa `S4 Implement` & `inspectTargetRepoSymbols` trong `ai-orchestrator` (`internal/webui/server.go`, `internal/webui/task_inspector.go`, `internal/webui/telegram_bridge.go`):**
   - Trong `extractKeywords` (`internal/webui/task_inspector.go:338-357`): Thêm tiếng Việt stop-words (`"con"`, `"hiện"`, `"đang"`, `"dùng"`, `"giờ"`, `"tao"`, `"muốn"`, `"chuyển"`, `"qua"`) và chuẩn hóa từ khóa `"posgret"` / `"postgres"` -> `"postgres"`, `"trans_his"` -> `"trans-his"`.
   - Trong `inspectTargetRepoSymbols` (`internal/webui/task_inspector.go:3124-3182`): Ưu tiên chấm điểm cao cho các file trong `models/`, `dbHandler/`, `domain/handlers/`, `defined/`, `start.ts`, `svc-env.ts` thay vì `data-transfers/params/`.
   - Trong `S4 Implement` & `S5 QA` (`internal/webui/server.go:1496-1510, 1680-1703`): Khi `isolatedRepoWT/.git` không tồn tại nhưng `repoPath` tồn tại trên đĩa và `!s.isTestTaskOrTempRoot(qt.ExternalID)`, giữ nguyên `execDir = repoPath` để `S4` và `S5` đọc trực tiếp `git status --short`, `git diff --stat`, và `collectRepoUnifiedDiffWithUntracked(ctx, repoPath)` từ `/Users/trainguyen/Documents/work/centralized-export-service` rồi lưu vào `.orchestrator/tasks/TASK-13/diff.patch`!
   - Trong `synthesizeIsolatedRemediationDiffIfApproved` (`internal/webui/telegram_bridge.go:1037-1109`): Nếu repo đích đã có `git diff` thật trên đĩa (`collectRepoUnifiedDiffWithUntracked(ctx, repoDir) != ""`), ghi trực tiếp `git diff` thật của repo vào `.orchestrator/tasks/<TaskKey>/diff.patch` và `.orchestrator/tasks/<TaskKey>/subtasks/<SubID>/diff.patch` thay vì sinh comment giả.

### 1.2. `TASK-RDQ-02` — Quản lý tường minh `Hàng Đợi Task (Theo Repo)` trên WebUI (`:8090`)
- **Backend (`internal/webui/server.go` / `task_inspector.go`):**
  - Bổ sung vào `WaveTaskNode`:
    - `QueuePosition int` (`json:"queue_position"` — thứ tự `#1, #2...` trong hàng đợi chờ chạy).
    - `IsRunning bool` (`json:"is_running"` — đang thực thi trực tiếp).
    - `SubTasksDone int` (`json:"sub_tasks_done"`) & `SubTasksTotal int` (`json:"sub_tasks_total"`).
    - `RepoDiffFiles int` (`json:"repo_diff_files"`) & `RepoDiffSummary string` (`json:"repo_diff_summary"` — ví dụ: `5 files changed` lấy từ `git status --short` thật của `TargetRepo` hoặc `diff.patch`).
- **Frontend (`internal/webui/static/index.html`, `internal/webui/static/app.js`, `internal/webui/static/app.css`):**
  - Thêm thanh **Queue Summary & Filter Bar** ngay dưới tiêu đề `Hàng Đợi Task (Theo Repo)`:
    - Hiển thị số lượng trực quan: `⚡ Đang chạy: X` · `⏸ Chờ duyệt: Y` · `⏳ Chờ chạy: Z` · `✓ Đã xong: W`.
    - Bộ lọc 1-click: `Tất cả` | `⚡ Đang chạy & Chờ` | `⏸ Chờ Duyệt` | `✓ Đã xong`.
  - **Gom nhóm theo từng Repository (`📦 centralized-export-service`, `📦 centralized-data-service`...):**
    - Header mỗi nhóm Repo hiện rõ số task đang chạy / chờ duyệt / tổng task của Repo đó.
    - Mỗi thẻ Task hiện rõ trạng thái tiếng Việt nổi bật (`⚡ ĐANG CHẠY`, `⏸ CHỜ APPROVE S3`, `⏳ HÀNG ĐỢI #1`, `✓ ĐÃ XONG`), tiến độ `🧩 X/Y Sub-Tasks`, chỉ báo `📝 Git Diff: N files` trên repo thật, và các nút thao tác trực tiếp (`▶ Chạy Ngay`, `✓ Duyệt & Chạy S4`, `↻ Chạy Lại S4`, `⏹ Dừng`).

### 1.3. `TASK-RDQ-03` — Lệnh Telegram `/task` liệt kê danh sách Task hiện tại kèm Inline Keyboard Click-to-Run (`RUN_TASK <TaskKey>`)
- **Trong `internal/webui/telegram_bridge.go`:**
  1. Mở rộng `processIncomingTelegramCommand`:
     - Hỗ trợ lệnh **`/task`** (cùng các bí danh `/tasks`, `/list`, `danh sách task`, `/task@*`):
       - Gọi `s.handleTelegramTaskListCommand(ctx, chatID, fromUser)` để lấy toàn bộ danh sách task hiện tại từ `QueueStore.ListAll(ctx)`, gom nhóm theo Repo, hiển thị trạng thái từng task (`⚡ ĐANG CHẠY`, `⏸ CHỜ DUYỆT S3`, `⏳ HÀNG ĐỢI`, `✅ ĐÃ XONG`), tiến độ `Sub-Tasks (done/total)`, số file `git diff` thật trên repo, và gửi kèm **Inline Keyboard (`inline_keyboard`)** với mỗi nút là:
         - `▶️ Chạy TASK-13 (centralized-export · NEEDS_YOU)` -> `callback_data: "RUN_TASK TASK-13"`
         - Кèm nút `🔄 Làm mới danh sách (/task)` -> `callback_data: "/task"`
     - Hỗ trợ lệnh/callback **`RUN_TASK <TaskKey>`** (hoặc `/run <TaskKey>`, `chạy <TaskKey>`):
       - Chọn `<TaskKey>` làm task đang active trên WebUI (`s.setActiveTaskKey(targetKey)`).
       - Gửi thông báo phản hồi tức thì lên Telegram: `"🚀 Đang kích hoạt chạy <TaskKey> (<Title>) trên repo <RepoName>..."`.
       - Nếu task đang ở `RAW` / `PRIORITIZED` / `READY` và chưa có `09_tasks_solution.md`: tự động chạy `"plan"` (`S1 -> S2 -> S2.5 DG0 -> S3 -> DG1/DG2`) rồi bắn thông báo S3 kèm nút `✅ APPROVE` về Telegram.
       - Nếu task đang ở `NEEDS_YOU`, `READY` (đã có S3 Plan), hoặc `DONE`: tự động đánh dấu `.user-approved` + `.audit-remediation-ready` và chạy `"approve_and_run"` (`S4 Implement -> S5 Adversarial Audit -> DG3`) rồi bắn báo cáo hoàn tất kèm `git diff` thật về Telegram!

---

## 2. Concrete Code Demo (`DG2` Verification)

### 2.1. Code Demo 1: `/Users/trainguyen/Documents/work/centralized-export-service/dbHandler/postgres.ts` & `models/trans-his/trans-his.model.ts`
```typescript
// dbHandler/postgres.ts
import { LoggerInstance } from "moleculer";
import { Pool } from "pg";
import DbConnectorFactory, { DbType, DbConfig } from "./DbConnectorFactory";

export interface PostgresConnectOptions {
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
	private alias: string;

	public constructor(logger: LoggerInstance, alias = "trans-his") {
		this.logger = logger;
		this.alias = alias;
	}

	public async createConnection(
		options: PostgresConnectOptions,
		callback: (err: any, pool: Pool | null) => void
	) {
		const targetAlias = options.alias || this.alias;
		const config: DbConfig = {
			type: DbType.POSTGRES,
			host: options.host!,
			port: Number(options.port) || 5432,
			database: options.database!,
			username: options.username,
			password: options.password,
			poolSize: Number(options.poolSize) || 10
		};

		try {
			const factory = DbConnectorFactory.getInstance(this.logger);
			this.logger.info(`PostgresDb wrapper requesting connection: ${targetAlias}`);
			const conn = await factory.connect(config, targetAlias);
			if (conn.rawPool) {
				callback(null, conn.rawPool as Pool);
			} else {
				throw new Error(`Connection '${targetAlias}' established but Postgres Pool instance missing`);
			}
		} catch (err) {
			this.logger.error(`PostgresDb (${targetAlias}) initialization failed`, err);
			callback(err, null);
		}
	}

	public getConnection(alias?: string) {
		return DbConnectorFactory.getInstance(this.logger).getConnection(alias || this.alias);
	}

	public closeAll() {
		return DbConnectorFactory.getInstance(this.logger).closeAll();
	}
}

export = PostgresDb;
```

```typescript
// models/trans-his/trans-his.model.ts (PostgreSQL Pool + Knex PG Builder with Mongoose Chain Compatibility)
import { MongoFuncHelper } from "@goopay/goopay-library/helpers";
import knex, { Knex } from "knex";
import _ from "lodash";
import { LoggerInstance } from "moleculer";
import { Connection, Schema } from "mongoose";
import { Pool } from "pg";
import { TransHisEntity } from "../../data-transfers/entities";
import { DB_COLLECTION } from "../../defined/app-setting";

class TransHisModel {
	private readonly logger: LoggerInstance;
	private readonly dbConnection: any;
	private readonly pgPool?: Pool;
	private readonly knexPg: Knex;
	private readonly isPostgres: boolean;
	private readonly schema?: Schema<TransHisEntity>;
	private readonly model: any;
	private readonly collectionName: string = DB_COLLECTION.TRANS_HIS__TRANS_HIS;
	private readonly tableName: string = "trans_his";
	// Supports both PostgreSQL parameterized queries via pg.Pool + knex({ client: "pg" })
	// and legacy Mongoose model interface for existing unit test mocks.
}
```

### 2.2. Code Demo 2: Telegram `/task` & `RUN_TASK <TaskKey>` (`internal/webui/telegram_bridge.go`)
```go
func isTelegramTaskListCommand(lower string) bool {
	fields := strings.Fields(lower)
	if len(fields) == 0 {
		return false
	}
	cmd := fields[0]
	if idx := strings.Index(cmd, "@"); idx > 0 {
		cmd = cmd[:idx]
	}
	return cmd == "/task" || cmd == "/tasks" || cmd == "/list" || lower == "task" || lower == "tasks" || lower == "danh sách task"
}

func isTelegramRunTaskCommand(lower string) bool {
	return strings.HasPrefix(lower, "run_task ") ||
		strings.HasPrefix(lower, "/run ") ||
		strings.HasPrefix(lower, "/start_task ")
}
```
