# 09. Technical Solution — Hỗ trợ Master MySQL & Master ClickHouse (`master_mysql_clickhouse`)

## 1. Kết quả Phản biện Trước Kế hoạch (`S2.5 Pre-Plan Critique / DG0`)

Qua rà soát thực tế codebase ở cả 3 repository (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) đối chiếu với tính năng **Master MongoDB** hiện hữu, chúng em ghi nhận **6 phát hiện vật lý (`F1`–`F6`)** cần giải quyết đồng bộ theo nguyên tắc **Single-Responsibility (SRP)** và **Không làm phình God-files (`L008`, `L039`)**:

- **F1 (`centralized-data-service` — Quản lý Kết nối & GORM Dialect):**
  - `ConnectionManager.normalizeEngineName` (`internal/service/source/connection_manager.go:126-137`) đã chuẩn hóa `"mysql"/"mariadb" -> "mysql"` và `"clickhouse"/"ch" -> "clickhouse"`.
  - Tuy nhiên, `GetMasterDB` (`connection_manager.go:400`) gọi `m.reg.OpenGorm(dsn, "master:"+key)` vốn đang mở cứng `gorm.Open(postgres.Open(dsn), ...)` tại `pkgs/database/multi.go:211`.
  - Cả 2 thư viện `gorm.io/driver/mysql v1.5.7` và `gorm.io/driver/clickhouse v0.7.0` (cùng `github.com/go-sql-driver/mysql v1.9.3` và `github.com/ClickHouse/clickhouse-go/v2 v2.30.0`) **đã có sẵn trong `go.mod`** của cả `centralized-data-service` (`go.mod:9,52,151-152`) lẫn `cdc-cms-service` (`go.mod:45,76,164-165`), không cần cài thêm dependency ngoại lai.
- **F2 (`centralized-data-service` — Trình sinh & Thực thi DDL Master):**
  - `MasterDDLGenerator.Generate` & `Apply` (`internal/service/master/master_ddl_generator.go:84-390`) hiện chỉ tách nhánh MongoDB sang `master_ddl_generator_mongo.go:17-338`, còn lại mặc định sinh cú pháp PostgreSQL (`SET LOCAL lock_timeout = '5s'`, `"schema"."table"`, `TIMESTAMPTZ`, `JSONB`, `CREATE INDEX IF NOT EXISTS`).
  - Với **MySQL (InnoDB)**: Không có `SET LOCAL lock_timeout`, dùng backtick `` `db`.`table` ``, không có `CREATE INDEX IF NOT EXISTS` chuẩn (phải kiểm tra `information_schema.statistics` trước khi tạo index), và cột `TEXT`/`JSON` không thể làm Unique Key trực tiếp nếu không chỉ định độ dài hoặc dùng `VARCHAR(255)` cho khóa chính `_id`/`id`.
  - Với **ClickHouse**: Dùng bảng `ENGINE = ReplacingMergeTree(_source_ts) ORDER BY (<conflict_target>)`, kiểu dữ liệu `Int64`, `String`, `Float64`, `Decimal(38, 10)`, `DateTime64(3, 'UTC')`, `UInt8` (cho `_deleted`), và `ALTER TABLE <db>.<table> ADD COLUMN IF NOT EXISTS`.
- **F3 (`centralized-data-service` — Luồng Ghi Batch Upsert, OCC `_source_ts` & Xóa):**
  - `TransmuterModule` (`internal/service/master/transmuter.go:277,375,525,692,1449,1588`) hiện tách nhánh MongoDB sang `transmuter_mongo.go:15-433`, còn nhánh SQL đang dùng cú pháp đặc thù PostgreSQL (`$1`, `$2::jsonb`, `ON CONFLICT (...) DO UPDATE SET ... WHERE COALESCE(EXCLUDED._source_ts, 0) >= ... RETURNING (xmax = 0) AS inserted`).
  - Với **MySQL**: Phải dùng placeholder `?` (chia chunk an toàn dưới trần `65,535` placeholders), kiểm soát OCC `_source_ts` (kèm dung sai `2000ms`) qua bí danh dòng `INSERT INTO <db>.<table> (...) VALUES (?, ...) AS incoming ON DUPLICATE KEY UPDATE col = IF(COALESCE(incoming._source_ts, 0) >= COALESCE(<table>._source_ts, 0) - 2000, incoming.col, <table>.col)` (tương thích MySQL 8.0.20+ và fallback `VALUES(col)` cho MariaDB/MySQL cũ), đồng thời giữ nguyên `_gpay_id` gốc khi `conflictTarget != "_gpay_id"`.
  - Với **ClickHouse**: `ReplacingMergeTree(_source_ts)` tự động giữ bản ghi có `_source_ts` lớn nhất theo khóa `ORDER BY (conflictTarget)`; tuy nhiên để chặn tuyệt đối sự kiện đến trễ (Out-of-Order event có `_source_ts` cũ hơn dữ liệu đang có) ngay trước khi chèn và xử lý xóa, tầng `transmuter_clickhouse.go` thực hiện Pre-fetch OCC Guard (tương tự `transmuter_mongo.go:322-342`) + `INSERT INTO <db>.<table>` theo lô và xóa qua `ALTER TABLE <db>.<table> DELETE WHERE ...`.
- **F4 (`centralized-data-service` — Index Governance & Đối soát Recon Segment B + Heal SegB):**
  - `IndexManager` (`internal/service/governance/index_driver_registry.go:121-123`) mới đăng ký `postgresIndexDriver` và `mongoIndexDriver` (trong khi `index_driver_test.go:67` đã dự kiến 4 driver `postgresql`, `mongodb`, `clickhouse`, `mysql`).
  - `ReconDestAgent` (`internal/service/recon/recon_dest_agent.go`, `recon_dest_query.go`, `recon_dest_hash.go`, `recon_window_index_probe.go`) dùng `SET TRANSACTION READ ONLY`, `"schema"."table"`, `pk::text` (chỉ hợp lệ trên PostgreSQL) và tách nhánh `da.IsMongo()` sang `recon_dest_mongo.go`. Cần tách file `recon_dest_sql_dialects.go` xử lý truy vấn Recon cho MySQL (`` `db`.`table` ``, `CAST(pk AS CHAR)`) và ClickHouse (`` `db`.`table` FINAL ``, `toString(pk)`, không mở transaction `SET TRANSACTION READ ONLY`).
- **F5 (`cdc-cms-service` — Quản trị Kết nối Master & Tạo Master Binding):**
  - `master_connection_repo_gorm.go:160-165,240-245` và `master_registry_handler_connections_probe.go:22-29` đang ép mọi `engine_type` khác `"mongodb"` về `"postgresql"`.
  - `create_master.go:176-199`: Giống MongoDB, cả **MySQL** và **ClickHouse** đều tổ chức theo mô hình **Database-scoped (`<database>.<table>`, không có tầng `schema` `"public"` như PostgreSQL)**, nên khi tạo Master Binding trên kết nối MySQL/ClickHouse cần tự động lấy `default_database` của kết nối làm `master_schema` / `master_database` nếu người dùng chưa truyền hoặc truyền `"public"`.
- **F6 (`cdc-cms-web` — Giao diện Quản lý Kết nối Đích & Tạo Master):**
  - `MasterRegistry.tsx:376,1648,1794` và `CreateMasterModal.tsx:172-225` hiện chỉ hiển thị lựa chọn `PostgreSQL (port 5432)` và `MongoDB (port 27017)`.
  - `MasterRegistry.tsx` hiện dài `1942` dòng; cần tách logic nhận diện/cấu hình Engine đích (`postgresql`, `mongodb`, `mysql`, `clickhouse`) ra file helper riêng `cdc-cms-web/src/components/master/masterEngineConfig.ts` để giữ `MasterRegistry.tsx` gọn nhẹ (`NOW <= HEAD`).

---

## 2. Thiết kế Giải pháp Duy nhất (Đối chiếu 1–1 với Master MongoDB & Tách File Đơn Trách nhiệm)

### 2.1. Bảng Đối chiếu Kiến trúc 1–1: Master MongoDB $\rightarrow$ Master MySQL & Master ClickHouse

| Tầng Kiến trúc (Layer) | File Mẫu của Master MongoDB | File Mới / Mở rộng (Đơn Trách nhiệm — SRP) | Trách nhiệm Đơn (Single Responsibility) |
| :--- | :--- | :--- | :--- |
| **1. Infra / Connection Pool (`centralized-data-service`)** | `internal/service/source/connection_manager.go` | `pkgs/database/multi_dialector.go` *(Mới)*<br>`internal/service/source/connection_manager_sql_engines.go` *(Mới)* | Chọn GORM Dialector (`postgres`, `mysql`, `clickhouse`) theo `engineType`/DSN; dựng DSN chuẩn cho MySQL (`tcp(host:3306)/db?parseTime=True&loc=UTC`) & ClickHouse (`clickhouse://user:pass@host:9000/db`); cung cấp `IsMySQLMasterConnection`, `IsClickHouseMasterConnection`, `ResolveMasterTargetDatabase`. |
| **2. Master DDL Generator (`centralized-data-service`)** | `internal/service/master/master_ddl_generator_mongo.go` | `internal/service/master/master_ddl_generator_mysql.go` *(Mới)*<br>`internal/service/master/master_ddl_generator_clickhouse.go` *(Mới)* | Sinh và thực thi `CREATE TABLE`, `ALTER TABLE ADD COLUMN`, `DROP COLUMN`, `DROP TABLE`, và Indexes cho MySQL (`InnoDB`) và ClickHouse (`ReplacingMergeTree(_source_ts)`). |
| **3. Type Converter & Transmuter (`centralized-data-service`)** | `internal/service/master/mongo_type_converter.go`<br>`internal/service/master/transmuter_mongo.go` | `internal/service/master/sql_dialect_type_converter.go` *(Mới)*<br>`internal/service/master/transmuter_mysql.go` *(Mới)*<br>`internal/service/master/transmuter_clickhouse.go` *(Mới)* | Chuẩn hóa kiểu dữ liệu MySQL/ClickHouse; thực thi `bulkUpsertMySQLMaster` (`ON DUPLICATE KEY UPDATE` + OCC `_source_ts`), `bulkUpsertClickHouseMaster` (`ReplacingMergeTree` + Pre-fetch OCC Guard), `inferLastGpayID`, và xóa dòng/orphan. |
| **4. Index Governance (`centralized-data-service`)** | `internal/service/governance/index_driver_mongo.go` | `internal/service/governance/index_driver_mysql.go` *(Mới)*<br>`internal/service/governance/index_driver_clickhouse.go` *(Mới)* | Thực thi `IndexEngineDriver` (`ListIndexes`, `CreateIndex`, `DropIndex`) cho MySQL (`information_schema.statistics`) và ClickHouse (`system.data_skipping_indices` / `ALTER TABLE ADD INDEX`). |
| **5. Recon Segment B & Heal (`centralized-data-service`)** | `internal/service/recon/recon_dest_mongo.go` | `internal/service/recon/recon_dest_sql_dialects.go` *(Mới)* | Thực thi `CountRows`, `CountDeletedRows`, `CountRowsBeforeCutoff`, `CountInWindow`, `HashWindow`, `ListIDTsInWindow`, `FetchRowsByPK` cho MySQL và ClickHouse (`FINAL`). |
| **6. CMS Backend (`cdc-cms-service`)** | `internal/api/master/master_registry_handler_connections_probe.go`<br>`internal/infra/persistence/master/master_connection_repo_gorm.go`<br>`internal/app/commands/master/create_master.go` | `internal/api/master/master_connection_engine_dsn.go` *(Mới)* + cập nhật nhẹ `master_connection_repo_gorm.go` & `create_master.go` | Chuẩn hóa `engine_type` (`postgresql`, `mongodb`, `mysql`, `clickhouse`), dựng DSN & Ping Test Connection cho cả 4 engine, đồng bộ `default_database` làm `master_schema` cho các engine cấp Database (`mongodb`, `mysql`, `clickhouse`). |
| **7. CMS Frontend (`cdc-cms-web`)** | `src/pages/MasterRegistry.tsx`<br>`src/components/master/CreateMasterModal.tsx` | `src/components/master/masterEngineConfig.ts` *(Mới)* + cập nhật `MasterRegistry.tsx` & `CreateMasterModal.tsx` | Cung cấp metadata hiển thị (Tag màu, cổng mặc định `5432`/`27017`/`3306`/`9000`, chế độ khóa Database 1-1 cho `mongodb`/`mysql`/`clickhouse`) mà không làm phình `MasterRegistry.tsx`. |

---

### 2.2. Bảng Ánh xạ Điểm Gọi Thực thi & Thời điểm Kích hoạt (`L004` & `L020`)

| Thành phần Mới | Được gọi tại (`caller file:line`) | Thời điểm Kích hoạt (Lifecycle / Event Trigger) |
| :--- | :--- | :--- |
| `normalizeMasterEngineType` & `buildMasterConnectionDSN` & `probeMasterSQLConnection` (`master_connection_engine_dsn.go`) | `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:80,118,189,234` & `master_registry_handler_connections_probe.go:158,244` | Khi Operator bấm **Test Connection** hoặc **Lưu Kết Nối** (`POST/PUT /api/v1/master-connections`) trên UI. |
| `normalizeMasterConnEngine` & `syncDatabaseScopedMasterBindings` (`master_connection_repo_gorm.go`) | `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go:160,240,296` & `internal/app/commands/master/create_master.go:176` | Khi tạo/cập nhật kết nối Master hoặc khi tạo Master Binding mới (`POST /api/v1/masters`). |
| `Registry.OpenGormWithEngine(dsn, role, engineType)` (`multi_dialector.go`) | `centralized-data-service/internal/service/source/connection_manager.go:400` (`GetMasterDB`) | Khi Worker lần đầu mở kết nối GORM tới Master DB theo `master_connection_key`. |
| `g.isMySQLBinding` / `g.isClickHouseBinding` $\rightarrow$ `applyMySQLTableAndIndexes` / `applyClickHouseTableAndIndexes` | `centralized-data-service/internal/service/master/master_ddl_generator.go:95,352,543,628,727` (`Generate`, `Apply`, `ReconcileColumn`, `DropColumn`, `DropTable`) | Khi Operator bấm **Approve** Master Binding hoặc thêm/xóa Mapping Rule kích hoạt sự kiện NATS `master.ddl.apply`. |
| `binding.IsMySQLDestination` / `binding.IsClickHouseDestination` $\rightarrow$ `bulkUpsertMySQLMaster` / `bulkUpsertClickHouseMaster` | `centralized-data-service/internal/service/master/transmuter.go:277,375,525,692,1449,1588` | Khi Worker chạy Transmute theo lô (Incremental / Full Sync / Realtime Event Batch / Heal) và xóa bản ghi `_deleted`. |
| `newMySQLIndexDriver` & `newClickHouseIndexDriver` | `centralized-data-service/internal/service/governance/index_driver_registry.go:121-124` (`NewIndexManagerWithConn`) | Khi khởi tạo `IndexManager` và khi Operator xem/tạo/xóa Index trên trang Index Governance. |
| `da.IsMySQL()` / `da.IsClickHouse()` (`recon_dest_sql_dialects.go`) | `centralized-data-service/internal/service/recon/recon_dest_query.go:18,75,112`, `recon_dest_hash.go:26`, `recon_window_index_probe.go:198` | Khi chạy Smoke Recon (`scanExact`), Full Recon Segment B (`Shadow <-> Master`), và Heal Segment B. |

---

## 3. Code Demo Chi tiết (Propose-Only — Chờ User `APPROVE`)

### 3.1. Tầng Kết nối & GORM Multi-Dialect (`centralized-data-service`)

#### File mới: `centralized-data-service/pkgs/database/multi_dialector.go`
```go
package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

func ResolveGormDialector(dsn, engineType string) gorm.Dialector {
	eng := strings.ToLower(strings.TrimSpace(engineType))
	trimmedDSN := strings.TrimSpace(dsn)
	switch {
	case eng == "mysql" || eng == "mariadb" || strings.Contains(trimmedDSN, "@tcp(") || strings.HasPrefix(trimmedDSN, "mysql://"):
		cleanDSN := strings.TrimPrefix(trimmedDSN, "mysql://")
		return mysql.Open(cleanDSN)
	case eng == "clickhouse" || eng == "ch" || strings.HasPrefix(trimmedDSN, "clickhouse://") || strings.HasPrefix(trimmedDSN, "tcp://"):
		return clickhouse.Open(trimmedDSN)
	default:
		return postgres.Open(trimmedDSN)
	}
}

func (r *Registry) OpenGormWithEngine(dsn, role, engineType string) (*gorm.DB, error) {
	r.mu.RLock()
	if db, ok := r.gormDBs[role]; ok && db != nil {
		r.mu.RUnlock()
		return db, nil
	}
	r.mu.RUnlock()

	logLevel := logger.Warn
	if r.cfg.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(ResolveGormDialector(dsn, engineType), &gorm.Config{
		Logger: NewCDCLogger(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open (%s): %w", engineType, err)
	}

	tracePlugin := tracing.NewPlugin(tracing.WithoutMetrics(), tracing.WithoutQueryVariables())
	if err := db.Use(tracePlugin); err != nil {
		return nil, fmt.Errorf("gorm otel plugin: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(r.cfg.DBPool.MaxOpenConn)
	sqlDB.SetMaxIdleConns(r.cfg.DBPool.MaxIdleConn)
	if r.cfg.DBPool.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(r.cfg.DBPool.ConnMaxLifetime)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.gormDBs[role]; ok && existing != nil {
		_ = sqlDB.Close()
		return existing, nil
	}
	r.gormDBs[role] = db
	return db, nil
}
```

#### File mới: `centralized-data-service/internal/service/source/connection_manager_sql_engines.go`
```go
package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	modelsource "centralized-data-service/internal/model/source"
)

func (m *ConnectionManager) IsMySQLMasterConnection(ctx context.Context, key string) bool {
	return m.ResolveMasterEngineType(ctx, key) == "mysql"
}

func (m *ConnectionManager) IsClickHouseMasterConnection(ctx context.Context, key string) bool {
	return m.ResolveMasterEngineType(ctx, key) == "clickhouse"
}

func (m *ConnectionManager) ResolveMasterTargetDatabase(ctx context.Context, key, fallbackSchema string) string {
	key = strings.TrimSpace(key)
	sysDB, err := m.GetSystemDB(ctx)
	if err == nil && sysDB != nil && key != "" && key != "default" {
		var conn modelsource.ConnectionRegistry
		if qErr := sysDB.WithContext(ctx).Table("cdc_system.connection_registry").
			Where("(connection_code = ? OR connection_code = ?) AND status = 'active'", key, strings.ToLower(key)).
			First(&conn).Error; qErr == nil && conn.DefaultDatabase != nil {
			if dbName := strings.TrimSpace(*conn.DefaultDatabase); dbName != "" {
				return dbName
			}
		}
	}
	if fb := strings.TrimSpace(fallbackSchema); fb != "" && fb != "public" {
		return fb
	}
	return "gpay_master"
}

func buildMasterDSNByEngine(conn *modelsource.ConnectionRegistry) string {
	if conn == nil {
		return ""
	}
	eng := normalizeEngineName(conn.EngineType)
	if eng != "mysql" && eng != "clickhouse" {
		return buildDSNFromFieldsPatched(conn)
	}
	var user, pass string
	if len(conn.OptionsJSON) > 0 {
		var opts map[string]any
		if json.Unmarshal(conn.OptionsJSON, &opts) == nil {
			user, _ = opts["username"].(string)
			if user == "" {
				user, _ = opts["user"].(string)
			}
			pass, _ = opts["password"].(string)
		}
	}
	host := "localhost"
	if conn.Host != nil && strings.TrimSpace(*conn.Host) != "" {
		host = strings.TrimSpace(*conn.Host)
	}
	dbName := "default"
	if conn.DefaultDatabase != nil && strings.TrimSpace(*conn.DefaultDatabase) != "" {
		dbName = strings.TrimSpace(*conn.DefaultDatabase)
	}
	port := 3306
	if eng == "clickhouse" {
		port = 9000
	}
	if conn.Port != nil && *conn.Port > 0 {
		port = *conn.Port
	}
	if eng == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			user, pass, host, port, dbName)
	}
	return fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?dial_timeout=5s&read_timeout=30s",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, dbName)
}
```

---

### 3.2. Trình sinh & Thực thi DDL cho Master MySQL & ClickHouse (`centralized-data-service`)

#### File mới: `centralized-data-service/internal/service/master/master_ddl_generator_mysql.go`
```go
package master

import (
	"context"
	"fmt"
	"strings"

	mastermodel "centralized-data-service/internal/model/master"
	"gorm.io/gorm"
)

func (g *MasterDDLGenerator) isMySQLBinding(ctx context.Context, reg *masterDDLBindingRow) bool {
	if reg == nil {
		return false
	}
	return strings.EqualFold(reg.EngineType, "mysql") ||
		strings.EqualFold(reg.EngineType, "mariadb") ||
		(g.connMgr != nil && g.connMgr.IsMySQLMasterConnection(ctx, reg.MasterConnectionKey))
}

func mapRuleToMySQLType(colName, pgType string, isIndexedKey bool) string {
	upper := strings.ToUpper(strings.TrimSpace(pgType))
	switch {
	case strings.HasPrefix(upper, "BIGINT") || upper == "INT8":
		return "BIGINT"
	case strings.HasPrefix(upper, "INT") || upper == "INT4" || upper == "INTEGER":
		return "INT"
	case strings.HasPrefix(upper, "SMALLINT") || upper == "INT2":
		return "SMALLINT"
	case strings.HasPrefix(upper, "BOOL"):
		return "TINYINT(1)"
	case strings.HasPrefix(upper, "NUMERIC") || strings.HasPrefix(upper, "DECIMAL"):
		return "DECIMAL(38,10)"
	case strings.HasPrefix(upper, "DOUBLE") || strings.HasPrefix(upper, "FLOAT"):
		return "DOUBLE"
	case strings.HasPrefix(upper, "TIMESTAMP") || upper == "TIMESTAMPTZ":
		return "DATETIME(3)"
	case upper == "DATE":
		return "DATE"
	case upper == "JSON" || upper == "JSONB":
		return "JSON"
	default:
		if isIndexedKey || colName == "_id" || colName == "id" || colName == "_source_id" {
			return "VARCHAR(255)"
		}
		return "LONGTEXT"
	}
}

func quoteMySQLIdent(ident string) string {
	return "`" + strings.ReplaceAll(strings.TrimSpace(ident), "`", "``") + "`"
}

func quoteMySQLQualified(dbName, table string) string {
	if strings.TrimSpace(dbName) == "" || dbName == "public" {
		return quoteMySQLIdent(table)
	}
	return quoteMySQLIdent(dbName) + "." + quoteMySQLIdent(table)
}

func (g *MasterDDLGenerator) generateMySQLDDL(ctx context.Context, reg *masterDDLBindingRow, rules []mastermodel.MappingRuleV2) (*MasterDDLResult, error) {
	dbName := reg.DefaultDatabase
	if g.connMgr != nil {
		dbName = g.connMgr.ResolveMasterTargetDatabase(ctx, reg.MasterConnectionKey, reg.MasterSchema)
	}
	pkCol := parsePKFromSpec(reg.TransformSpec)
	indexedCols := map[string]bool{"_gpay_id": true, "_source_ts": true, "_updated_at": true, pkCol: true}
	if tsCol := g.resolveMasterTimestampField(ctx, reg); tsCol != "" {
		indexedCols[tsCol] = true
	}

	cols := []string{
		"`_gpay_id` BIGINT NOT NULL PRIMARY KEY",
		"`_source_ts` BIGINT",
		"`_deleted` TINYINT(1) NOT NULL DEFAULT 0",
		"`_updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)",
	}
	seen := map[string]bool{"_gpay_id": true, "_source_ts": true, "_deleted": true, "_updated_at": true}
	var alterSQL, finCols []string

	for _, r := range rules {
		if r.Status != "approved" || seen[r.TargetColumn] || !ddlIdentRe.MatchString(r.TargetColumn) {
			continue
		}
		seen[r.TargetColumn] = true
		isFin := financialIndexRe.MatchString(r.TargetColumn)
		if isFin {
			finCols = append(finCols, r.TargetColumn)
			indexedCols[r.TargetColumn] = true
		}
		myType := mapRuleToMySQLType(r.TargetColumn, r.DataType, indexedCols[r.TargetColumn])
		colIdent := quoteMySQLIdent(r.TargetColumn)
		cols = append(cols, fmt.Sprintf("%s %s NULL", colIdent, myType))
		alterSQL = append(alterSQL, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s NULL;",
			quoteMySQLQualified(dbName, reg.MasterTable), colIdent, myType))
	}

	qt := quoteMySQLQualified(dbName, reg.MasterTable)
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
		qt, strings.Join(cols, ",\n  "))

	idxSQL := []string{
		fmt.Sprintf("CREATE INDEX %s ON %s (`_source_ts`);", quoteMySQLIdent("ix_"+reg.MasterTable+"_source_ts"), qt),
		fmt.Sprintf("CREATE INDEX %s ON %s (`_updated_at`);", quoteMySQLIdent("ix_"+reg.MasterTable+"_updated_at"), qt),
	}
	if pkCol != "" && pkCol != "_gpay_id" && seen[pkCol] {
		idxSQL = append(idxSQL, fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s);",
			quoteMySQLIdent("ux_"+reg.MasterTable+"_"+pkCol), qt, quoteMySQLIdent(pkCol)))
	}
	for _, fc := range finCols {
		idxSQL = append(idxSQL, fmt.Sprintf("CREATE INDEX %s ON %s (%s);",
			quoteMySQLIdent("ix_"+reg.MasterTable+"_"+fc), qt, quoteMySQLIdent(fc)))
	}

	return &MasterDDLResult{
		MasterName:    reg.MasterTable,
		MasterSchema:  dbName,
		CreateSQL:     createSQL,
		AlterSQL:      alterSQL,
		IndexSQL:      idxSQL,
		RuleCount:     len(rules),
		FinancialCols: finCols,
	}, nil
}

func (g *MasterDDLGenerator) applyMySQLTableAndIndexes(ctx context.Context, reg *masterDDLBindingRow, res *MasterDDLResult) (*MasterDDLResult, error) {
	db, err := g.connMgr.GetMasterDB(ctx, reg.MasterConnectionKey)
	if err != nil {
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return res, err
	}
	dbName := res.MasterSchema
	if err := db.WithContext(ctx).Exec(res.CreateSQL).Error; err != nil {
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return res, fmt.Errorf("mysql create table: %w", err)
	}
	for _, alt := range res.AlterSQL {
		if err := db.WithContext(ctx).Exec(alt).Error; err != nil && !isMySQLDuplicateColumnOrIndexErr(err) {
			g.markDDLStatus(ctx, reg.ID, "failed", err)
			return res, fmt.Errorf("mysql alter table: %w", err)
		}
	}
	for _, idx := range res.IndexSQL {
		if err := db.WithContext(ctx).Exec(idx).Error; err != nil && !isMySQLDuplicateColumnOrIndexErr(err) {
			g.markDDLStatus(ctx, reg.ID, "failed", err)
			return res, fmt.Errorf("mysql create index: %w", err)
		}
	}
	g.finalizeMasterDDLSuccess(ctx, reg.ID, dbName, reg.MasterTable)
	return res, nil
}

func isMySQLDuplicateColumnOrIndexErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "1060") || strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "1061") || strings.Contains(msg, "duplicate key name")
}
```

#### File mới: `centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go`
```go
package master

import (
	"context"
	"fmt"
	"strings"

	mastermodel "centralized-data-service/internal/model/master"
)

func (g *MasterDDLGenerator) isClickHouseBinding(ctx context.Context, reg *masterDDLBindingRow) bool {
	if reg == nil {
		return false
	}
	return strings.EqualFold(reg.EngineType, "clickhouse") ||
		strings.EqualFold(reg.EngineType, "ch") ||
		(g.connMgr != nil && g.connMgr.IsClickHouseMasterConnection(ctx, reg.MasterConnectionKey))
}

func mapRuleToClickHouseType(pgType string, isOrderKey bool) string {
	upper := strings.ToUpper(strings.TrimSpace(pgType))
	var base string
	switch {
	case strings.HasPrefix(upper, "BIGINT") || upper == "INT8":
		base = "Int64"
	case strings.HasPrefix(upper, "INT") || upper == "INT4" || upper == "INTEGER":
		base = "Int32"
	case strings.HasPrefix(upper, "SMALLINT") || upper == "INT2":
		base = "Int16"
	case strings.HasPrefix(upper, "BOOL"):
		base = "UInt8"
	case strings.HasPrefix(upper, "NUMERIC") || strings.HasPrefix(upper, "DECIMAL"):
		base = "Decimal(38, 10)"
	case strings.HasPrefix(upper, "DOUBLE") || strings.HasPrefix(upper, "FLOAT"):
		base = "Float64"
	case strings.HasPrefix(upper, "TIMESTAMP") || upper == "TIMESTAMPTZ":
		base = "DateTime64(3, 'UTC')"
	case upper == "DATE":
		base = "Date32"
	default:
		base = "String"
	}
	if isOrderKey {
		return base
	}
	return fmt.Sprintf("Nullable(%s)", base)
}

func (g *MasterDDLGenerator) generateClickHouseDDL(ctx context.Context, reg *masterDDLBindingRow, rules []mastermodel.MappingRuleV2) (*MasterDDLResult, error) {
	dbName := reg.DefaultDatabase
	if g.connMgr != nil {
		dbName = g.connMgr.ResolveMasterTargetDatabase(ctx, reg.MasterConnectionKey, reg.MasterSchema)
	}
	pkCol := parsePKFromSpec(reg.TransformSpec)
	orderKey := "_gpay_id"

	seen := map[string]bool{"_gpay_id": true, "_source_ts": true, "_deleted": true, "_updated_at": true}
	for _, r := range rules {
		if r.Status == "approved" && r.TargetColumn == pkCol && ddlIdentRe.MatchString(pkCol) {
			orderKey = pkCol
			break
		}
	}

	cols := []string{
		"`_gpay_id` Int64",
		"`_source_ts` UInt64 DEFAULT 0",
		"`_deleted` UInt8 DEFAULT 0",
		"`_updated_at` DateTime64(3, 'UTC') DEFAULT now64(3)",
	}
	var alterSQL, finCols []string
	qt := quoteMySQLQualified(dbName, reg.MasterTable)

	for _, r := range rules {
		if r.Status != "approved" || seen[r.TargetColumn] || !ddlIdentRe.MatchString(r.TargetColumn) {
			continue
		}
		seen[r.TargetColumn] = true
		if financialIndexRe.MatchString(r.TargetColumn) {
			finCols = append(finCols, r.TargetColumn)
		}
		chType := mapRuleToClickHouseType(r.DataType, r.TargetColumn == orderKey)
		colIdent := quoteMySQLIdent(r.TargetColumn)
		cols = append(cols, fmt.Sprintf("%s %s", colIdent, chType))
		alterSQL = append(alterSQL, fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s;", qt, colIdent, chType))
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n  %s\n) ENGINE = ReplacingMergeTree(_source_ts)\nORDER BY (%s)\nSETTINGS index_granularity = 8192;",
		qt, strings.Join(cols, ",\n  "), quoteMySQLIdent(orderKey),
	)

	return &MasterDDLResult{
		MasterName:    reg.MasterTable,
		MasterSchema:  dbName,
		CreateSQL:     createSQL,
		AlterSQL:      alterSQL,
		IndexSQL:      nil,
		RuleCount:     len(rules),
		FinancialCols: finCols,
	}, nil
}
```

---

### 3.3. Luồng Ghi Batch Upsert & OCC `_source_ts` cho MySQL & ClickHouse (`centralized-data-service`)

#### File mới: `centralized-data-service/internal/service/master/transmuter_mysql.go`
```go
package master

import (
	"centralized-data-service/internal/service/source"
	"context"
	"fmt"
	"strings"
)

const mysqlMaxPlaceholders = 60000
const mysqlOCCClockSkewToleranceMs = 2000

func (b *masterBindingRuntime) IsMySQLDestination(ctx context.Context, connMgr *source.ConnectionManager) bool {
	if b == nil {
		return false
	}
	eng := strings.ToLower(strings.TrimSpace(b.EngineType))
	if eng == "mysql" || eng == "mariadb" {
		return true
	}
	if eng != "" {
		return false
	}
	return connMgr != nil && connMgr.IsMySQLMasterConnection(ctx, b.MasterConnectionKey)
}

func BuildMySQLUpsertSQL(
	dbName, tableName string,
	keys []string,
	records []map[string]any,
	conflictTarget string,
	isFullSync bool,
) (string, []any) {
	qt := quoteMySQLQualified(dbName, tableName)
	cols := make([]string, len(keys))
	for i, k := range keys {
		cols[i] = quoteMySQLIdent(k)
	}

	values := make([]any, 0, len(records)*len(keys))
	placeholders := make([]string, 0, len(records))
	for _, rec := range records {
		rowPh := make([]string, len(keys))
		for j, k := range keys {
			rowPh[j] = "?"
			values = append(values, coerceMySQLBindValue(k, rec[k]))
		}
		placeholders = append(placeholders, "("+strings.Join(rowPh, ", ")+")")
	}

	sets := make([]string, 0, len(keys)+1)
	if isFullSync {
		for _, k := range keys {
			if k == "_gpay_id" || k == conflictTarget {
				continue
			}
			colIdent := quoteMySQLIdent(k)
			sets = append(sets, fmt.Sprintf("%s = VALUES(%s)", colIdent, colIdent))
		}
		sets = append(sets, "`_updated_at` = UTC_TIMESTAMP(3)")
	} else {
		occCond := fmt.Sprintf("COALESCE(VALUES(`_source_ts`), 0) >= COALESCE(%s.`_source_ts`, 0) - %d",
			quoteMySQLIdent(tableName), mysqlOCCClockSkewToleranceMs)
		for _, k := range keys {
			if k == "_gpay_id" || k == conflictTarget {
				continue
			}
			colIdent := quoteMySQLIdent(k)
			sets = append(sets, fmt.Sprintf("%s = IF(%s, VALUES(%s), %s.%s)",
				colIdent, occCond, colIdent, quoteMySQLIdent(tableName), colIdent))
		}
		sets = append(sets, fmt.Sprintf("`_updated_at` = IF(%s, UTC_TIMESTAMP(3), %s.`_updated_at`)",
			occCond, quoteMySQLIdent(tableName)))
	}

	sqlText := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s ON DUPLICATE KEY UPDATE %s",
		qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "), strings.Join(sets, ", "))
	return sqlText, values
}
```

#### File mới: `centralized-data-service/internal/service/master/transmuter_clickhouse.go`
```go
package master

import (
	"centralized-data-service/internal/service/source"
	"context"
	"fmt"
	"strings"
)

const clickhouseOCCClockSkewToleranceMs int64 = 2000

func (b *masterBindingRuntime) IsClickHouseDestination(ctx context.Context, connMgr *source.ConnectionManager) bool {
	if b == nil {
		return false
	}
	eng := strings.ToLower(strings.TrimSpace(b.EngineType))
	if eng == "clickhouse" || eng == "ch" {
		return true
	}
	if eng != "" {
		return false
	}
	return connMgr != nil && connMgr.IsClickHouseMasterConnection(ctx, b.MasterConnectionKey)
}

func FilterClickHouseOCCRecords(
	records []map[string]any,
	existingSourceTs map[string]int64,
	conflictTarget string,
	isFullSync bool,
) ([]map[string]any, int64) {
	if len(records) == 0 {
		return nil, 0
	}
	if conflictTarget == "" {
		conflictTarget = "_gpay_id"
	}
	out := make([]map[string]any, 0, len(records))
	var occSkipped int64
	for _, rec := range records {
		keyVal := rec[conflictTarget]
		if keyVal == nil {
			keyVal = rec["_gpay_id"]
		}
		if keyVal == nil {
			continue
		}
		keyStr := fmt.Sprintf("%v", keyVal)
		incomingTs := extractInt64SourceTs(rec["_source_ts"])
		if !isFullSync && existingSourceTs != nil {
			if prevTs, exists := existingSourceTs[keyStr]; exists && incomingTs > 0 && prevTs > incomingTs+clickhouseOCCClockSkewToleranceMs {
				occSkipped++
				continue
			}
		}
		if existingSourceTs != nil {
			if prevTs, exists := existingSourceTs[keyStr]; !exists || incomingTs > prevTs {
				existingSourceTs[keyStr] = incomingTs
			}
		}
		out = append(out, rec)
	}
	return out, occSkipped
}
```

---

### 3.4. Mở rộng CMS Backend (`cdc-cms-service`) & Frontend (`cdc-cms-web`)

#### File mới: `cdc-cms-service/internal/api/master/master_connection_engine_dsn.go`
```go
package master

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeMasterEngineType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mongodb", "mongo":
		return "mongodb"
	case "mysql", "mariadb":
		return "mysql"
	case "clickhouse", "ch":
		return "clickhouse"
	default:
		return "postgresql"
	}
}

func defaultMasterPortForEngine(engineType string) int {
	switch normalizeMasterEngineType(engineType) {
	case "mongodb":
		return 27017
	case "mysql":
		return 3306
	case "clickhouse":
		return 9000
	default:
		return 5432
	}
}

func isDatabaseScopedMasterEngine(engineType string) bool {
	eng := normalizeMasterEngineType(engineType)
	return eng == "mongodb" || eng == "mysql" || eng == "clickhouse"
}

func buildSQLDialectMasterDSN(engineType, host string, port int, username, password, database string) string {
	eng := normalizeMasterEngineType(engineType)
	if port <= 0 {
		port = defaultMasterPortForEngine(eng)
	}
	if eng == "mysql" {
		if database == "" {
			database = "mysql"
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&timeout=5s",
			username, password, host, port, database)
	}
	if database == "" {
		database = "default"
	}
	return fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?dial_timeout=5s&read_timeout=30s",
		url.QueryEscape(username), url.QueryEscape(password), host, port, database)
}
```

#### File mới: `cdc-cms-web/src/components/master/masterEngineConfig.ts`
```ts
export type MasterEngineType = 'postgresql' | 'mongodb' | 'mysql' | 'clickhouse';

export const MASTER_ENGINE_OPTIONS: { label: string; value: MasterEngineType }[] = [
  { label: 'PostgreSQL (port 5432)', value: 'postgresql' },
  { label: 'MongoDB (port 27017)', value: 'mongodb' },
  { label: 'MySQL / MariaDB (port 3306)', value: 'mysql' },
  { label: 'ClickHouse (port 9000)', value: 'clickhouse' },
];

export function normalizeMasterEngine(raw?: string | null): MasterEngineType {
  const v = (raw || '').trim().toLowerCase();
  if (v === 'mongodb' || v === 'mongo') return 'mongodb';
  if (v === 'mysql' || v === 'mariadb') return 'mysql';
  if (v === 'clickhouse' || v === 'ch') return 'clickhouse';
  return 'postgresql';
}

export function getMasterEngineDefaultPort(engine?: string | null): number {
  switch (normalizeMasterEngine(engine)) {
    case 'mongodb':
      return 27017;
    case 'mysql':
      return 3306;
    case 'clickhouse':
      return 9000;
    default:
      return 5432;
  }
}

export function isDatabaseScopedMasterEngine(engine?: string | null): boolean {
  const norm = normalizeMasterEngine(engine);
  return norm === 'mongodb' || norm === 'mysql' || norm === 'clickhouse';
}

export function getMasterEngineLabel(engine?: string | null): string {
  switch (normalizeMasterEngine(engine)) {
    case 'mongodb':
      return 'MongoDB';
    case 'mysql':
      return 'MySQL';
    case 'clickhouse':
      return 'ClickHouse';
    default:
      return 'PostgreSQL';
  }
}

export function getMasterEngineTagColor(engine?: string | null): string {
  switch (normalizeMasterEngine(engine)) {
    case 'mongodb':
      return 'green';
    case 'mysql':
      return 'orange';
    case 'clickhouse':
      return 'purple';
    default:
      return 'geekblue';
  }
}
```

---

## 4. Bản Đối Soát Line-by-Line Từng File Hiện Hữu Sẽ Chỉnh Sửa & Cam Kết Tương Thích Ngược 100%

Để đảm bảo tuyệt đối nguyên tắc **Single-Responsibility (SRP)**, **Không nhồi nhét vào God-files**, và **Tương thích ngược 100% với PostgreSQL & MongoDB**, dưới đây là bảng kê từng dòng dự kiến cập nhật trên các file hiện hữu (toàn bộ logic thực thi nằm trong 12 file mới độc lập):

| STT | File Hiện Hữu (Có sẵn) | Vị trí Dòng (`file:line`) | Nội dung Cập nhật Line-by-Line (Chỉ thêm nhánh Delegate mỏng) | Đảm bảo Tương thích Ngược (`PostgreSQL` & `MongoDB`) |
| :---: | :--- | :--- | :--- | :--- |
| 1 | `centralized-data-service/internal/service/source/connection_manager.go` | `L85-L100` (`ResolveMasterEngineType`)<br>`L341-L405` (`GetMasterDB`) | - Tại `ResolveMasterEngineType`: nhận diện thêm prefix `mysql://`, `@tcp(`, `clickhouse://`, `tcp://` trong `ConnectionOverrides`.<br>- Tại `GetMasterDB` (`L377-L379`): gọi `buildMasterDSNByEngine(&conn)` (hỗ trợ cả `mysql`/`clickhouse`).<br>- Tại `GetMasterDB` (`L400`): đổi 1 dòng `m.reg.OpenGorm(dsn, "master:"+key)` thành `m.reg.OpenGormWithEngine(dsn, "master:"+key, resolvedEng)`. | Khi `resolvedEng` là `"postgresql"` hoặc `""`, `OpenGormWithEngine` gọi `postgres.Open(dsn)` y hệt `OpenGorm` cũ; khi `"mongodb"` vẫn chặn và trả về lỗi hướng dẫn dùng `GetMasterMongoClient` như cũ (`TestConnectionManager_IsMongoMasterConnection_AndGetMasterDBRejectsMongo` PASS 100%). |
| 2 | `centralized-data-service/internal/service/source/metadata_registry_utils.go` | `L20-L28` (`tryPlainDSN`) | - Thêm `"clickhouse://"`, `"tcp://"` vào danh sách prefix URI hợp lệ và kiểm tra `strings.Contains(s, "@tcp(")` cho MySQL DSN. | Các prefix cũ (`mongodb://`, `postgres://`, `mysql://`) giữ nguyên 100%. |
| 3 | `centralized-data-service/internal/service/master/master_ddl_generator.go` | `L125` (`Generate`)<br>`L352-L355` (`Apply`)<br>`L543-L545` (`ReconcileColumn`)<br>`L628-L630` (`DropColumn`)<br>`L661` (`Swap`)<br>`L727-L729` (`DropTable`) | - Ngay sau nhánh `isMongoBinding` (hoặc sau khi nạp `rules` ở `Generate`), thêm 2 nhánh delegate mỏng:<br>`if g.isMySQLBinding(ctx, reg) { return g.generateMySQLDDL(ctx, reg, rules) }`<br>`if g.isClickHouseBinding(ctx, reg) { return g.generateClickHouseDDL(ctx, reg, rules) }`<br>- Tương tự tại `Apply`, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable` uỷ quyền sang `master_ddl_generator_mysql.go` và `master_ddl_generator_clickhouse.go`. | Nhánh `isMongoBinding` và luồng mặc định PostgreSQL (`SET LOCAL lock_timeout`, `ENABLE ROW LEVEL SECURITY`, `ALTER COLUMN ... TYPE ... USING`) không thay đổi bất kỳ dòng nào. |
| 4 | `centralized-data-service/internal/service/master/transmuter.go` | `L277-L322` (`Run` DDL gate)<br>`L374-L378` (`inferLastGpayID`)<br>`L524-L531` (Orphan cleanup)<br>`L692-L729` (`RunEvents` DDL gate)<br>`L1449-L1452` (`deleteFromMasterBySourceAndGpayIDs`)<br>`L1588-L1591` (`bulkUpsertMaster`) | - Tại 6 điểm rẽ nhánh `IsMongoDestination`, bổ sung `else if masterRow.IsMySQLDestination(ctx, t.connMgr)` và `else if masterRow.IsClickHouseDestination(ctx, t.connMgr)` gọi sang các hàm tương ứng trong `transmuter_mysql.go` và `transmuter_clickhouse.go`. | Không sửa bất kỳ dòng SQL nào của PostgreSQL (`$1`, `ON CONFLICT`, `::text`, `"schema"."table"`) hay MongoDB (`bulkUpsertMongoMaster`). |
| 5 | `centralized-data-service/internal/service/governance/index_driver_registry.go` | `L121-L124` (`NewIndexManagerWithConn`) | - Thêm 2 dòng đăng ký:<br>`im.RegisterDriver(newMySQLIndexDriver(connMgr, logger))`<br>`im.RegisterDriver(newClickHouseIndexDriver(connMgr, logger))` | Giữ nguyên `newPostgresIndexDriver` và `newMongoIndexDriver`. |
| 6 | `centralized-data-service/internal/service/recon/recon_engine.go` & `recon_stream_bucket_engine.go` & `recon_dest_agent.go` & `recon_dest_query.go` & `recon_dest_hash.go` & `recon_window_index_probe.go` & `recon_tier_b.go` & `recon_execute_heal_segb.go` | `recon_engine.go:L216-L229`<br>`recon_stream_bucket_engine.go:L176-L185`<br>`recon_dest_query.go:L17,72,108,201,233,300,369,476,641`<br>`recon_dest_hash.go:L25,173`<br>`recon_window_index_probe.go:L35`<br>`recon_tier_b.go:L671,839`<br>`recon_execute_heal_segb.go:L103,684,877` | - Gắn `engineType` & `defaultDatabase` vào `ReconDestAgent` khi khởi tạo từ `GetMasterAgent`.<br>- Ngay dưới mỗi câu `if da.IsMongo() { return da.mongo... }`, thêm `if da.IsSQLDialect() { return da.sqlDialect... }` uỷ quyền sang `recon_dest_sql_dialects.go`.<br>- Tại `ExecuteHealHandler`, ngay dưới `if h.isMongoMasterRef(ctx, ref)`, thêm `if h.isSQLDialectMasterRef(ctx, ref)` uỷ quyền sang `recon_execute_heal_segb_sql_dialects.go`. | Khi `engineType == "postgresql"` (hoặc `""`), `da.IsSQLDialect()` trả về `false`, chạy 100% luồng PostgreSQL gốc (`readOnlyDB`, `pg_class`, `hashtextextended`). |
| 7 | `centralized-data-service/internal/service/sourcecount/source_count_runner.go` | `L166-L181` (`case "master":`)<br>`L291-L300` (`countSQL`) | - Ở `case "master":`, nếu `r.connMgr != nil && params.ConnectionCode != ""`, đồng bộ `params.EngineType = r.connMgr.ResolveMasterEngineType(queryCtx, params.ConnectionCode)`.<br>- Ở `countSQL`: thêm nhánh `eng == "clickhouse"` dùng `FINAL`. | Giữ nguyên nhánh `shadow`, `source` và `mongodb`. |
| 8 | `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go`, `master_read_repo_gorm.go`, `internal/app/commands/master/create_master.go`, `internal/api/master/master_registry_handler_connections.go`, `master_registry_handler_connections_probe.go` | `master_connection_repo_gorm.go:L160,240,350,440`<br>`master_read_repo_gorm.go:L34,39`<br>`create_master.go:L176-L191`<br>`master_registry_handler_connections.go:L82-L101,L200-L219`<br>`master_registry_handler_connections_probe.go:L22-L29,L184-L192,L244` | - Thay phép gán cứng `mongodb`/`postgresql` bằng `normalizeMasterEngineType(engineType)`.<br>- Mở rộng quy tắc 1-Connection = 1-Database (`isDatabaseScopedMasterEngine`) cho `mongodb`, `mysql`, `clickhouse` tại `create_master.go:176`, `master_read_repo_gorm.go:34`, `syncMongoMasterBindingsDatabase:351`.<br>- Chỉ nối `&connect_timeout=5` khi `engineType == "postgresql"`; uỷ quyền ping MySQL/ClickHouse sang `probeSQLDialectMasterConnection` trong `master_connection_engine_dsn.go`. | Toàn bộ bài test hiện hữu (`TestCreateMasterHandler_LivePostgres_EnforcesMongoDefaultDatabase`, v.v.) và kết nối PostgreSQL/MongoDB giữ nguyên hành vi 100%. |
| 9 | `cdc-cms-web/src/pages/MasterRegistry.tsx` & `src/components/master/CreateMasterModal.tsx` | `MasterRegistry.tsx:L365,376,418-427,439-460,501-565,645-665,1648-1666,1735-1922`<br>`CreateMasterModal.tsx:L79-95,141-144,172-245` | - Import các hàm pure helper từ `src/components/master/masterEngineConfig.ts` (`MASTER_ENGINE_OPTIONS`, `normalizeMasterEngine`, `getMasterEngineDefaultPort`, `isDatabaseScopedMasterEngine`, `getMasterEngineLabel`, `getMasterEngineTagColor`). | Giữ nguyên props cũ của `CreateMasterModal` và giảm số dòng code lặp trong `MasterRegistry.tsx`. |

