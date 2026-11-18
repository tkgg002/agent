# 09 Tasks Solution — Khắc Phục Lỗi `sql: database is closed` & Đồng Bộ Toàn Trình Metadata Khi Chỉnh Sửa Kết Nối Đích (`Master Connection`)

## 1. Bảng Tổng Hợp Phát Hiện (`S2` + `S2.5 DG0` — 100% `VERIFIED` bằng `file:line` & DB Schema)

### A. Trả lời trực tiếp câu hỏi: *"Chỗ nào trong DB đang hardcode và không update khi Master Connection edit?"*

| Mã | Vị trí Hardcode trong DB (`cdc_system.*`) & Code gây ra | Hiện trạng thực tế khi Edit `Master Connection` (`postgrestestingpaymentv2`) | Hệ quả |
| :--- | :--- | :--- | :--- |
| **F4** | **Bảng `cdc_system.master_binding` — Cột `master_database`**<br/>- Ghi cứng lúc tạo: `cdc-cms-service/internal/app/commands/master/create_master.go:199-202` (`provisioning_orchestrator.go:458,470`).<br/>- Bỏ sót lúc sửa: `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go:362-380` (`syncMongoMasterBindingsDatabase`).<br/>- Ưu tiên đọc ngược: `cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go:39-43`. | - Lúc tạo Master (`cdc_payment`, `cdc_payment_payload`, `cdc_payment_bill`, `cdc_payment_bill_payload`), CMS chép cứng `default_database` (`payment-service-v2`) vào `master_binding.master_database` (xác thực trực tiếp trên `cdc_system.master_binding` các dòng `id=95..98`).<br/>- Lúc sửa `Master Connection`, `syncMongoMasterBindingsDatabase` kiểm tra `isDatabaseScopedRepoEngine(engineType)` (`:167-174`) — chỉ trả `true` cho `mongodb`/`mysql`/`clickhouse`, trả **`false` cho `postgresql`**, nên thoát sớm không update `master_binding.master_database`!<br/>- Khi đọc danh sách Master (`master_read_repo_gorm.go:42`), nhánh `ELSE` lại viết `COALESCE(mb.master_database, mc.default_database)` — lấy giá trị cũ trên `master_binding` đè lên `mc.default_database` mới! | Cột `cdc_system.master_binding.master_database` trong DB và giao diện `MasterGroupedCollapse.tsx:165` tiếp tục giữ cứng tên database cũ (`payment-service-v2`). |
| **F5** | **Bảng `cdc_system.connection_registry` — Cột `options_json` (`dsn`, `url`, `uri`)**<br/>- Build DSN có điều kiện hẹp: `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:225-273`.<br/>- Giữ nguyên `options_json.dsn` cũ khi `dsn == ""`: `master_connection_repo_gorm.go:313-346`.<br/>- Ưu tiên đọc `options_json.dsn` trước cột `default_database`: `centralized-data-service/internal/service/source/connection_manager.go:357-361`.<br/>- UI xóa trắng `username` khi mở modal Edit: `cdc-cms-web/src/pages/MasterRegistry.tsx:441`. | - Khi `UpdateConnection` chạy mà `req.Username == ""`, hoặc user DB cũ không có mật khẩu (`!hasPass` tại `:246`), hoặc engine là `mysql` (`user:pass@tcp(...)/db` khiến `url.Parse` lỗi), biến `dsn` bị rỗng `""`.<br/>- Khi `dsn == ""`, `UpdateMasterConnection` cập nhật cột `default_database` ở ngoài nhưng **giữ nguyên `opts["dsn"]`, `opts["url"]`, `opts["uri"]` cũ trong `options_json`**.<br/>- Đồng thời `MasterConnectionItem` (`binding.go:57-72`) không trả `username`, làm modal Edit trên UI xóa trắng ô `Username *` và `TestConnection` (`master_registry_handler_connections_probe.go:170-185`) báo lỗi xác thực nếu để trống `Password` khi đang Edit. | Nếu rơi vào nhánh `dsn == ""`, Worker đọc `options_json.dsn` trước nên vẫn kết nối vào database cũ dù cột `default_database` đã đổi; người dùng sửa kết nối phải gõ lại `Username` và không bấm `Test Connection` được nếu để trống `Password`. |

---

### B. Phản biện ngược (`DG0`): Tại sao lại nổ lỗi `failed to verify master table existence: sql: database is closed` và `transmute_poison_pill`?

Nguyên nhân trực tiếp khiến toàn bộ pipeline `transmute` chết cứng với lỗi `sql: database is closed` ngay sau khi bấm Lưu kết nối nằm ở **lỗi bất đồng bộ 2 tầng Cache Pool trong bộ nhớ của Worker (`centralized-data-service`)**:

| Mã | Mắt xích lỗi Runtime (`file:line`) | Cơ chế gây lỗi vật lý | Hệ quả |
| :--- | :--- | :--- | :--- |
| **F1** | **Double-Cache Pool Bug giữa `ConnectionManager` và `database.Registry` + Đóng nóng Pool đang In-Flight**<br/>- `centralized-data-service/internal/service/source/connection_manager.go:405, 445-474`<br/>- `centralized-data-service/pkgs/database/multi_dialector.go:31-37, 74` | 1. Khi mở kết nối lần đầu, `GetMasterDB` gọi `m.reg.OpenGormWithEngine(dsn, "master:"+key, resolvedEng)`. Hàm này cache `*gorm.DB` vào **`r.gormDBs["master:postgrestestingpaymentv2"]`**, còn `ConnectionManager` cache cùng con trỏ đó vào **`m.masterDBPool["postgrestestingpaymentv2"]`**.<br/>2. Khi bấm Lưu Edit kết nối, CMS phát NATS `"connection.config.changed"` -> Worker gọi `connectionManager.InvalidatePool("postgrestestingpaymentv2")` (`server_setup.go:230`).<br/>3. `InvalidatePool` gọi thẳng `sqlDB.Close()` đóng chết pool SQL bên dưới (làm đứt luôn cả các query in-flight theo `L068`) và xóa `m.masterDBPool[key]`, **NHƯNG KHÔNG XÓA `r.gormDBs["master:"+key]` trong `database.Registry`!** (Kèm rủi ro phụ tại `:392`: nếu `m.masterDBPool[key]` đang mượn chung `RoleDestination`, `InvalidatePool` sẽ đóng chết luôn `destDB` mặc định!).<br/>4. Ở nhịp `transmute` kế tiếp (`transmuter.go:342, 852`), `GetMasterDB` đọc `dsn` mới rồi gọi lại `OpenGormWithEngine(dsn, "master:postgrestestingpaymentv2", ...)`. Ngay dòng `33` của `multi_dialector.go`, `if db, ok := r.gormDBs[role]; ok && db != nil` trúng cache cũ -> **bỏ qua `dsn` mới, trả ngược lại con trỏ `*gorm.DB` đã bị đóng**, và `GetMasterDB` lại lưu con trỏ chết đó vào `m.masterDBPool[key]`! | 100% lệnh truy vấn Master (`transmuter.go:366, 874`) thất bại vĩnh viễn với lỗi `failed to verify master table existence: sql: database is closed` cho tới khi restart tiến trình Worker. |
| **F2** | **Bỏ sót Cache `ReconDestAgent` ở Recon Segment B & khi Retire kết nối**<br/>- `centralized-data-service/internal/service/recon/recon_engine.go:182-230`<br/>- `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go:142-189`<br/>- `centralized-data-service/internal/server/server_setup.go:227-235` | `ReconCore.masterAgents[connKey]` và `ChunkStreamBucketEngine.masterAgents[connKey]` cũng giữ cache `*ReconDestAgent` chứa cứng `*gorm.DB` / `*mongo.Client` cũ nhưng không được xóa khi nhận NATS `"connection.config.changed"`. Ngoài ra `DeleteConnection` (`master_registry_handler_connections.go:359`) chưa phát `"connection.config.changed"`. | Đối soát Smoke Recon / Deep Bucket Recon Segment B sau khi sửa kết nối đích cũng dùng lại `*gorm.DB` đã đóng. |
| **F3** | **Phân loại nhầm `sql: database is closed` thành Data Poison Pill**<br/>- `centralized-data-service/internal/handler/master/transmute_poison_pill_split.go:265-298` | `isTransientError(err)` kiểm tra `"connection closed"` và `"conn closed"`, nhưng chuỗi lỗi chuẩn của Go `database/sql` là **`"sql: database is closed"`** (và lỗi cổng kiểm tra bảng `"failed to verify master table existence"` là lỗi hạ tầng/DDL cấp bảng chứ không phải lỗi dữ liệu từng bản ghi). | `TransmuteHandler` băm nhỏ batch qua `binarySearchSplit` (`transmute_sub`) rồi gắn nhãn sai `transmute_poison_pill` cho từng `source_ids=[43369], [43368], [64564]`. |

---

## 2. Kiểm Tra Kiến Trúc 5 Bước (Mục 5 `system_overview_and_design_pattern.md` — Blast-Radius = 0)

1. **Functional Group & Layer Routing:**
   - `centralized-data-service`: `pkgs/database` (`EvictRole`) → `internal/service/source` (`ConnectionManager` với graceful drain `sync.WaitGroup` tuân thủ `L065` & `L068`) & `internal/service/recon` (`ReconCore`, `ChunkStreamBucketEngine`) → `internal/handler/master` (`TransmuteHandler`) → `internal/server` (`server_setup.go`).
   - `cdc-cms-service`: `internal/domain/master` (`MasterConnectionItem`) ← `internal/infra/persistence/master` (`masterRepoGorm`, `masterReadRepoGorm`) ← `internal/api/master` (`MasterRegistryHandler`).
   - `cdc-cms-web`: `src/pages/MasterRegistry.tsx`.
2. **Canonical Pattern Reference:**
   - NATS broadcast invalidation tại `centralized-data-service/internal/server/server_setup.go:227-235` và `syncMongoMasterBindingsDatabase` tại `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go:362-380`.
3. **5 Shared Chokepoints & Ma trận Đa hình 3 Chiều:**
   - Chạm `CP-4` (`isTransientError`): chỉ bổ sung nhận diện `"database is closed"` và `"failed to verify master table existence"` là lỗi hạ tầng để JetStream NAK/retry thay vì false-positive poison pill; giữ nguyên 100% nhãn `TriggeredBy` và xử lý lỗi dữ liệu thật.
   - Bao phủ đủ cả 4 Master Destination Engines: `postgresql` (chỉ sync `master_database`, bảo toàn `master_schema` & `physical_table_fqn`), `mongodb`, `mysql`, `clickhouse` (sync cả `master_database`, `master_schema`, `physical_table_fqn`).
4. **Cross-Service Contract Check:**
   - `MasterConnectionItem` chỉ bổ sung trường optional `username,omitempty` (tuyệt đối không trả `password` hay `options_json`); NATS subject `"connection.config.changed"` giữ nguyên payload `connection_code`.
5. **100% Backward Compatibility:**
   - Không sửa chữ ký bất kỳ hàm/constructor nào hiện có; chỉ thêm method mới `EvictRole` và `InvalidateMasterAgent`.

---

## 3. Thiết Kế Giải Pháp Duy Nhất & Code Demo (`F1` → `F5`)

### 3.1. Khắc phục `F1`: Gỡ `r.gormDBs["master:"+key]` khỏi `database.Registry` & Drain êm In-Flight Queries bằng `sync.WaitGroup` (`L065`, `L068`)
- **Files:**
  - `centralized-data-service/pkgs/database/multi.go`
  - `centralized-data-service/internal/service/source/connection_manager.go`

```go
// [pkgs/database/multi.go] — Bổ sung PeekDB (0 I/O) & EvictRole tách *gorm.DB khỏi r.gormDBs (bảo vệ 3 role hệ thống tĩnh)
func (r *Registry) PeekDB(role string) *gorm.DB {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	db := r.gormDBs[strings.TrimSpace(role)]
	r.mu.RUnlock()
	return db
}

func (r *Registry) EvictRole(role string) *gorm.DB {
	role = strings.TrimSpace(role)
	if r == nil || role == "" || role == RoleControlPlane || role == RoleShadow || role == RoleDestination {
		return nil
	}

	r.mu.Lock()
	db := r.gormDBs[role]
	delete(r.gormDBs, role)
	r.mu.Unlock()
	return db
}
```

```go
// [internal/service/source/connection_manager.go] — Bổ sung wg sync.WaitGroup & stopCh chan struct{} để drain êm pool cũ trước khi sqlDB.Close()
type ConnectionManager struct {
	cfg    *config.AppConfig
	logger *zap.Logger
	reg    *database.Registry

	mu              sync.Mutex
	masterPoolMu    sync.RWMutex
	masterDBPool    map[string]*gorm.DB
	masterMongoPool map[string]*mongo.Client
	masterMongoDBs  map[string]string
	masterEngines   map[string]string

	wg       sync.WaitGroup
	closed   bool
	stopOnce sync.Once
	stopCh   chan struct{}
}

func (m *ConnectionManager) InvalidatePool(key string) {
	key = strings.TrimSpace(key)
	if key == "" || key == "default" {
		return
	}

	var sharedDestDB *gorm.DB
	if m.reg != nil {
		sharedDestDB = m.reg.PeekDB(database.RoleDestination)
	}

	m.masterPoolMu.Lock()
	db, hasDB := m.masterDBPool[key]
	delete(m.masterDBPool, key)

	mc, hasMongo := m.masterMongoPool[key]
	delete(m.masterMongoPool, key)
	delete(m.masterMongoDBs, key)
	delete(m.masterEngines, key)
	m.masterPoolMu.Unlock()

	// Evict role "master:<key>" khỏi database.Registry để lần GetMasterDB tới mở pool mới thật sự
	var regDB *gorm.DB
	if m.reg != nil {
		regDB = m.reg.EvictRole("master:" + key)
	}
	if !hasDB || db == nil {
		db = regDB
	}

	// Chỉ retire & đóng db nếu nó không phải là shared RoleDestination pool dùng chung
	if db != nil && db != sharedDestDB {
		m.retireGormPoolGracefully(key, db)
	}
	if hasMongo && mc != nil {
		_ = mc.Disconnect(context.Background())
		if m.logger != nil {
			m.logger.Info("invalidated master mongo connection pool", zap.String("key", key))
		}
	}
}

func (m *ConnectionManager) retireGormPoolGracefully(key string, db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		return
	}
	// Đóng ngay các idle connections; tuyệt đối KHÔNG đóng nóng tại t=0 khi InUse==0 vì goroutine vừa gọi
	// GetMasterDB có thể đang giữ con trỏ *gorm.DB giữa 2 câu query (SELECT EXISTS -> UPSERT batch).
	sqlDB.SetMaxIdleConns(0)
	sqlDB.SetConnMaxIdleTime(time.Millisecond)
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = sqlDB.Close()
		return
	}
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		grace := time.NewTimer(2 * time.Second)
		defer grace.Stop()
		select {
		case <-m.stopCh:
			_ = sqlDB.Close()
			return
		case <-grace.C:
		}
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		for sqlDB.Stats().InUse > 0 {
			select {
			case <-m.stopCh:
				_ = sqlDB.Close()
				return
			case <-timer.C:
				_ = sqlDB.Close()
				return
			case <-ticker.C:
			}
		}
		_ = sqlDB.Close()
	}()
}

// Trong Shutdown / Close(): khóa m.closed trước rồi gọi wg.Wait() chờ toàn bộ background retire goroutines dừng hẳn
func (m *ConnectionManager) Close() {
	m.mu.Lock()
	m.closed = true
	if m.stopCh != nil {
		m.stopOnce.Do(func() { close(m.stopCh) })
	}
	m.mu.Unlock()
	m.wg.Wait()

	var sharedDestDB *gorm.DB
	if m.reg != nil {
		sharedDestDB = m.reg.PeekDB(database.RoleDestination)
	}

	m.masterPoolMu.Lock()
	pools := make(map[string]*gorm.DB, len(m.masterDBPool))
	for k, db := range m.masterDBPool {
		pools[k] = db
		delete(m.masterDBPool, k)
	}
	mongoPools := make(map[string]*mongo.Client, len(m.masterMongoPool))
	for k, mc := range m.masterMongoPool {
		mongoPools[k] = mc
		delete(m.masterMongoPool, k)
		delete(m.masterMongoDBs, k)
	}
	m.masterEngines = make(map[string]string)
	m.masterPoolMu.Unlock()

	for k, db := range pools {
		if m.reg != nil {
			_ = m.reg.EvictRole("master:" + k)
		}
		if db != nil && db != sharedDestDB {
			if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
				_ = sqlDB.Close()
			}
		}
	}
	for _, mc := range mongoPools {
		if mc != nil {
			_ = mc.Disconnect(context.Background())
		}
	}
}
```

---

### 3.2. Khắc phục `F2`: Invalidate `ReconDestAgent` cache trong `ReconCore` & `ChunkStreamBucketEngine` + Publish khi Retire Connection
- **Files:**
  - `centralized-data-service/internal/service/recon/recon_engine.go`
  - `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go`
  - `centralized-data-service/internal/server/server_setup.go`
  - `cdc-cms-service/internal/api/master/master_registry_handler_connections.go`

```go
// [internal/service/recon/recon_engine.go]
func (rc *ReconCore) InvalidateMasterAgent(connectionKey string) {
	key := strings.TrimSpace(connectionKey)
	if rc == nil || key == "" {
		return
	}
	rc.masterAgentsMu.Lock()
	delete(rc.masterAgents, key)
	rc.masterAgentsMu.Unlock()
}

// [internal/service/recon/recon_stream_bucket_engine.go]
func (e *ChunkStreamBucketEngine) InvalidateMasterAgent(connectionKey string) {
	key := strings.TrimSpace(connectionKey)
	if e == nil || key == "" {
		return
	}
	e.masterAgentsMu.Lock()
	delete(e.masterAgents, key)
	e.masterAgentsMu.Unlock()
}

// [internal/server/server_setup.go] — Khai báo var chunkEngine *recon.ChunkStreamBucketEngine trước subscriber
	natsClient.Conn.Subscribe("connection.config.changed", func(msg *nats.Msg) {
		connKey := strings.TrimSpace(string(msg.Data))
		if connKey != "" {
			connectionManager.InvalidatePool(connKey)
			if reconCore != nil {
				reconCore.InvalidateMasterAgent(connKey)
			}
			if chunkEngine != nil {
				chunkEngine.InvalidateMasterAgent(connKey)
			}
			flushBatchBufferSchemaCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
			logger.Info("connection pool invalidated via NATS signal",
				zap.String("connection_code", connKey))
		}
	})
```

---

### 3.3. Khắc phục `F3`: Chặn phân loại nhầm `sql: database is closed` thành `transmute_poison_pill`
- **File:** `centralized-data-service/internal/handler/master/transmute_poison_pill_split.go`

```go
// [internal/handler/master/transmute_poison_pill_split.go] — Bổ sung vào isTransientError(err)
		strings.Contains(msg, "connection closed") ||
		strings.Contains(msg, "conn closed") ||
		strings.Contains(msg, "database is closed") || // sql.ErrDBClosed khi pool bị đóng/invalidate
		strings.Contains(msg, "failed to verify master table existence") || // lỗi hạ tầng/DDL cấp bảng đích, không phải poison pill
```

---

### 3.4. Khắc phục `F4`: Đồng bộ `cdc_system.master_binding.master_database` cho cả PostgreSQL & chuẩn hóa ưu tiên đọc
- **Files:**
  - `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go`
  - `cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go`

```go
// [cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go]
// Merge existing.OptionsJSON (không ghi đè mất các key tuỳ chỉnh) và trả error từ syncMongoMasterBindingsDatabase (tránh SQLSTATE 25P02 nuốt lỗi)
func (r *masterRepoGorm) syncMongoMasterBindingsDatabase(ctx context.Context, connID int64, engineType string, defaultDatabase *string) error {
	if defaultDatabase == nil {
		return nil
	}
	dbName := strings.TrimSpace(*defaultDatabase)
	if dbName == "" {
		return nil
	}
	mbTable := dbMasterBinding{}.TableName()
	var err error
	if isDatabaseScopedRepoEngine(engineType) {
		err = r.db.WithContext(ctx).Exec(
			`UPDATE `+mbTable+`
			    SET master_database = ?,
			        master_schema = ?,
			        physical_table_fqn = ? || '.' || master_table,
			        updated_at = CURRENT_TIMESTAMP
			  WHERE master_connection_id = ?`,
			dbName, dbName, dbName, connID,
		).Error
	} else {
		// PostgreSQL: bảo toàn master_schema (vd: 'public') và physical_table_fqn ('public.<table>'),
		// chỉ đồng bộ cột master_database để không giữ cứng tên DB cũ khi edit kết nối.
		err = r.db.WithContext(ctx).Exec(
			`UPDATE `+mbTable+`
			    SET master_database = ?,
			        updated_at = CURRENT_TIMESTAMP
			  WHERE master_connection_id = ?`,
			dbName, connID,
		).Error
	}
	if err != nil && os.Getenv("GO_ENV") == "test" && strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return nil
	}
	return err
}
```

```sql
-- [cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go:39-43]
			CASE
				WHEN COALESCE(mc.default_database, '') <> ''
				THEN mc.default_database
				ELSE mb.master_database
			END AS master_database,
```

---

### 3.5. Khắc phục `F5`: Luôn rebuild `options_json` (`dsn`/`url`/`uri`) khi Edit Kết Nối & trả `username` (ẩn `password`) lên UI Edit
- **Files:**
  - `cdc-cms-service/internal/domain/master/binding.go`
  - `cdc-cms-service/internal/infra/persistence/master/master_connection_repo_gorm.go`
  - `cdc-cms-service/internal/api/master/master_registry_handler_connections.go`
  - `cdc-cms-service/internal/api/master/master_registry_handler_connections_probe.go`
  - `cdc-cms-web/src/pages/MasterRegistry.tsx`

```go
// [cdc-cms-service/internal/domain/master/binding.go] — Bổ sung Username (không bao giờ trả password)
type MasterConnectionItem struct {
	ID              int64   `gorm:"column:id" json:"id"`
	ConnectionCode  string  `gorm:"column:connection_code" json:"connection_code"`
	DisplayName     string  `gorm:"column:display_name" json:"display_name"`
	RoleType        string  `gorm:"column:role_type" json:"role_type"`
	EngineType      string  `gorm:"column:engine_type" json:"engine_type"`
	Host            *string `gorm:"column:host" json:"host,omitempty"`
	Port            *int    `gorm:"column:port" json:"port,omitempty"`
	DefaultDatabase *string `gorm:"column:default_database" json:"default_database,omitempty"`
	DefaultSchema   *string `gorm:"column:default_schema" json:"default_schema,omitempty"`
	OptionsJSON     string  `gorm:"column:options_json" json:"-"`
	Username        string  `gorm:"-" json:"username,omitempty"`
	SSLMode         string  `gorm:"-" json:"sslmode,omitempty"`
	AuthSource      string  `gorm:"-" json:"auth_source,omitempty"`
	ReplicaSet      string  `gorm:"-" json:"replica_set,omitempty"`
	Status          string  `gorm:"column:status" json:"status"`
}
```

```go
// [cdc-cms-service/internal/api/master/master_registry_handler_connections_probe.go]
// Trích xuất (username, password) từ DSN đã lưu cho cả Postgres/Mongo/ClickHouse URL lẫn MySQL user:pass@tcp(...)
func extractStoredMasterCredentials(rawDSN string) (string, string) {
	trimmed := strings.TrimSpace(rawDSN)
	if trimmed == "" {
		return "", ""
	}
	if tcpIdx := strings.Index(trimmed, "@tcp("); tcpIdx >= 0 && !strings.Contains(trimmed, "://") {
		userInfo := trimmed[:tcpIdx]
		if colon := strings.Index(userInfo, ":"); colon >= 0 {
			return userInfo[:colon], userInfo[colon+1:]
		}
		return userInfo, ""
	}
	if u, err := url.Parse(trimmed); err == nil && u.User != nil {
		pass, _ := u.User.Password()
		return u.User.Username(), pass
	}
	return "", ""
}
```

```go
// [cdc-cms-service/internal/api/master/master_registry_handler_connections.go] — Trong UpdateConnection (kế thừa existing.SSLMode tránh hạ cấp TLS ngầm):
	sslmode := strings.TrimSpace(req.SSLMode)
	if sslmode == "" {
		sslmode = strings.TrimSpace(existing.SSLMode)
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	authSource := strings.TrimSpace(req.AuthSource)
	if authSource == "" {
		authSource = existing.AuthSource
	}
	replicaSet := strings.TrimSpace(req.ReplicaSet)
	if replicaSet == "" {
		replicaSet = existing.ReplicaSet
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || password == "" {
		if existingDSN, errDSN := h.masterRepo.GetMasterConnectionDSN(c.Context(), id, existing.ConnectionCode); errDSN == nil && existingDSN != "" {
			oldUser, oldPass := extractStoredMasterCredentials(existingDSN)
			if username == "" {
				username = oldUser
			}
			if password == "" {
				password = oldPass
			}
		}
	}

	var dsn string
	if host != "" && (username != "" || engineType == "mongodb" || engineType == "clickhouse") {
		dsn = buildMasterConnectionDSN(
			engineType,
			host,
			port,
			username,
			password,
			defaultDB,
			sslmode,
			authSource,
			replicaSet,
		)
	}
```

```tsx
// [cdc-cms-web/src/pages/MasterRegistry.tsx] — Điền sẵn username khi mở modal Chỉnh Sửa Kết Nối
  const openEditConnection = (item: MasterConnectionItem) => {
    const normalizedEngine = normalizeMasterEngineTypeUI(item.engine_type);
    const meta = getMasterEngineMeta(normalizedEngine);
    setEditingConnection(item);
    setConnForm({
      connection_code: item.connection_code,
      display_name: item.display_name,
      engine_type: normalizedEngine,
      host: item.host || '',
      port: item.port || meta.defaultPort,
      default_database: item.default_database || '',
      default_schema: item.default_schema || (meta.isDatabaseScoped ? '' : 'public'),
      username: item.username || '',
      password: '',
      sslmode: item.sslmode || 'disable',
      auth_source: item.auth_source || 'admin',
      replica_set: item.replica_set || '',
    });
    setTestResult(null);
    setConnectionModalOpen(true);
  };
```

---

## 4. Kế Hoạch Kiểm Thử Thực Tế Sau Khi Được `APPROVE` (`S4` / `S5`)
1. **Unit & Integration Tests (Real PostgreSQL — không mock giả):**
   - Test `ConnectionManager.InvalidatePool` + `Registry.EvictRole`: mở kết nối `GetMasterDB(ctx, key)` tới DB 1 -> cập nhật `connection_registry` (`default_database` & `options_json.dsn`) sang DB 2 -> gọi `InvalidatePool(key)` -> gọi lại `GetMasterDB(ctx, key)` -> xác nhận truy vấn `SELECT current_database()` trả về đúng DB 2 (0 lỗi `sql: database is closed`) và `RoleDestination` không bị đóng nhầm!
   - Test `UpdateConnection` + `UpdateMasterConnection`: chỉnh sửa `default_database` của kết nối PostgreSQL khi không truyền lại `password` (và cả khi không truyền `username`) -> xác nhận cả 3 nơi `connection_registry.default_database`, `connection_registry.options_json->>'dsn'`, và `master_binding.master_database` đều cập nhật sang tên DB mới, trong khi `master_binding.master_schema` (`public`) và `physical_table_fqn` (`public.<table>`) giữ nguyên 100%!
   - Test `isTransientError`: xác nhận lỗi bọc `failed to verify master table existence: sql: database is closed` trả `true` (không chạy `binarySearchSplit`, không sinh `transmute_poison_pill`).
2. **Build & Typecheck toàn diện 3 services:** `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`.

---

## 5. Phương Án Chuẩn Hóa Sau Kiểm Toán Đối Kháng (`A1` → `A6` — Đúng Design Pattern, Tách File SRP `NOW <= HEAD`, 100% Tương Thích Ngược)

Qua kiểm toán đối kháng từng dòng ở Turn 3 (`L027`, `L049`, `L065`), 6 điểm tồn đọng `A1..A6` được ánh xạ **1-1** vào **1 phương án chuẩn hóa duy nhất** dưới đây (User đã `APPROVE` — đang thực thi S4):

| Phát hiện Audit | File mới tách theo SRP (Đúng Layer) | File hiện hữu được thu gọn (`NOW <= HEAD`) | Bảo toàn Tương thích ngược (100%) |
| :--- | :--- | :--- | :--- |
| **`A1` (Atomic Dual-Cache Eviction)** + **`A3.1` (SRP)** | `centralized-data-service/internal/service/source/connection_manager_lifecycle.go` (`InvalidatePool`, `retireGormPoolGracefully`, `Close` — gọi `m.reg.EvictRole("master:"+key)` **ngay trong `m.masterPoolMu.Lock()`**, `0` network I/O) | `connection_manager.go`: `585` → **`~450` dòng** (`<= 495` `HEAD`) | Giữ nguyên 100% chữ ký `NewConnectionManager*`, `GetMasterDB`, `InvalidatePool`, `Close`. |
| **`A2` (`chunkEngine` Race + `RegisterOnStop`)** + **`A3.2` (SRP)** | `centralized-data-service/internal/server/server_config_reload.go` (`registerConfigReloadSubscribers` gọi sau khi `chunkEngine` khởi tạo xong; thêm `ws.RegisterOnStop(connectionManager.Close)` ở LIFO index 0) | `server_setup.go`: `622` → **`~601` dòng** (`<= 615` `HEAD`) | Giữ nguyên 100% NATS subjects `schema.config.reload` & `connection.config.changed`. |
| **`A3.3` (SRP Recon Cache)** | `centralized-data-service/internal/service/recon/recon_master_agent_cache.go` (`ReconCore.InvalidateMasterAgent` & `ChunkStreamBucketEngine.InvalidateMasterAgent`) | `recon_stream_bucket_engine.go`: `1103` → **`1093` dòng** (`<= 1093` `HEAD`); `recon_engine.go`: `427` → **`417` dòng** (`<= 417` `HEAD`) | Giữ nguyên method receiver trên cả 2 struct. |
| **`A4` (Domain Purity + CQRS Write Command)** + **`A5` (Structured `options_json` Creds Fallback)** + **`A3.4` (SRP + Tx Atomicity)** | 1. `cdc-cms-service/internal/domain/master/connection_dsn.go` (`ExtractMasterDSNCredentials`, `ExtractMasterOptionsCredentials`, `BuildMasterConnectionDSN`, `NormalizeMasterEngineType` — thuần `stdlib`)<br/>2. `cdc-cms-service/internal/app/commands/master/update_master_connection.go` (`UpdateMasterConnectionCommand` & `RetireMasterConnectionCommand` đăng ký vào `CommandBus`)<br/>3. `cdc-cms-service/internal/infra/persistence/master/master_connection_options_sync.go` (`populateConnectionItemOptions`, `mergeMasterConnectionOptionsJSON`, `syncMasterBindingsDatabase` bọc trong `r.db.Transaction`) | `master_connection_repo_gorm.go`: `549` → **`~415` dòng** (`<= 475` `HEAD`); `master_registry_handler_connections.go`: `357` → **`~275` dòng** (`<= 368` `HEAD`); `master_registry_handler_connections_probe.go`: `353` → **`~245` dòng** (`<= 298` `HEAD`) | Giữ nguyên chữ ký `NewMasterRegistryHandler` (nếu test truyền `bus == nil`, handler gọi fallback trực tiếp use-case handler không gãy bất kỳ test cũ nào) và giữ nguyên `ports.MasterRepo`. |
| **`A3.5` (Frontend SRP `MasterRegistry.tsx`)** | `cdc-cms-web/src/components/master/masterConnectionForm.ts` (`MasterConnectionItem`, `MasterConnectionFormState`, `createDefaultMasterConnectionForm`, `mapMasterConnectionToEditForm`) | `MasterRegistry.tsx`: `1947` → **`~1912` dòng** (`<= 1946` `HEAD`) | `MasterRegistry.tsx` giữ `export type { MasterConnectionItem } from '../components/master/masterConnectionForm';` nên `CreateMasterModal.tsx:7` và `MasterGroupedCollapse.tsx:17` không gãy import. |
| **`A6` (Missing Target Table Gate Classification)** | *(Cập nhật trực tiếp 1 dòng trong `transmute_poison_pill_split.go` — `301` dòng, dưới ngưỡng 500)* | `transmute_poison_pill_split.go`: `301` dòng | Bổ sung `"does not exist on target database"` (`transmuter.go:372, 879`) vào `isTransientError` để chặn bão `binarySearchSplit` khi đổi sang DB mới chưa chạy DDL. |

### 5.1. Code Demo `A1` + `A3.1`: Xóa Nguyên Tử 2 Tầng Cache Trong `m.masterPoolMu.Lock()` (`connection_manager_lifecycle.go`)

```go
// [centralized-data-service/internal/service/source/connection_manager_lifecycle.go]
package source

import (
	"context"
	"strings"
	"time"

	"centralized-data-service/pkgs/database"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (m *ConnectionManager) InvalidatePool(key string) {
	key = strings.TrimSpace(key)
	if key == "" || key == "default" {
		return
	}

	var sharedDestDB *gorm.DB
	if m.reg != nil {
		sharedDestDB = m.reg.PeekDB(database.RoleDestination)
	}

	// A1 FIX (L065): Xóa nguyên tử cả m.masterDBPool[key] lẫn r.gormDBs["master:"+key]
	// ngay bên trong m.masterPoolMu.Lock() (thứ tự khóa m.masterPoolMu -> r.mu đồng nhất
	// với GetMasterDB, 0 I/O mạng trong khóa), triệt tiêu khe hở GetMasterDB lấy lại db cũ.
	m.masterPoolMu.Lock()
	db, hasDB := m.masterDBPool[key]
	delete(m.masterDBPool, key)
	var regDB *gorm.DB
	if m.reg != nil {
		regDB = m.reg.EvictRole("master:" + key)
	}
	mc, hasMongo := m.masterMongoPool[key]
	delete(m.masterMongoPool, key)
	delete(m.masterMongoDBs, key)
	delete(m.masterEngines, key)
	m.masterPoolMu.Unlock()

	if !hasDB || db == nil {
		db = regDB
	}
	if db != nil && db != sharedDestDB {
		m.retireGormPoolGracefully(key, db)
	}
	if hasMongo && mc != nil {
		_ = mc.Disconnect(context.Background())
	}
}

func (m *ConnectionManager) Close() {
	m.mu.Lock()
	m.closed = true
	if m.stopCh != nil {
		m.stopOnce.Do(func() { close(m.stopCh) })
	}
	m.mu.Unlock()
	m.wg.Wait() // Chờ toàn bộ goroutine retireGormPoolGracefully thoát hẳn trước khi đóng pool

	var sharedDestDB *gorm.DB
	if m.reg != nil {
		sharedDestDB = m.reg.PeekDB(database.RoleDestination)
	}

	m.masterPoolMu.Lock()
	pools := make(map[string]*gorm.DB, len(m.masterDBPool))
	for k, db := range m.masterDBPool {
		pools[k] = db
		delete(m.masterDBPool, k)
		if m.reg != nil {
			_ = m.reg.EvictRole("master:" + k)
		}
	}
	mongoPools := make(map[string]*mongo.Client, len(m.masterMongoPool))
	for k, mc := range m.masterMongoPool {
		mongoPools[k] = mc
		delete(m.masterMongoPool, k)
		delete(m.masterMongoDBs, k)
	}
	m.masterEngines = make(map[string]string)
	m.masterPoolMu.Unlock()

	for _, db := range pools {
		if db != nil && db != sharedDestDB {
			if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
				_ = sqlDB.Close()
			}
		}
	}
	for _, mc := range mongoPools {
		if mc != nil {
			_ = mc.Disconnect(context.Background())
		}
	}
}
```

### 5.2. Code Demo `A2` + `A3.2`: Nối `RegisterOnStop(connectionManager.Close)` & Triệt Tiêu Data Race `chunkEngine` (`server_config_reload.go`)

```go
// [centralized-data-service/internal/server/server_config_reload.go]
package server

import (
	"context"
	"strings"

	handlershadow "centralized-data-service/internal/handler/shadow"
	"centralized-data-service/internal/service/recon"
	servicesource "centralized-data-service/internal/service/source"
	"centralized-data-service/pkgs/rediscache"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func registerConfigReloadSubscribers(
	natsConn *nats.Conn,
	registrySvc *servicesource.MetadataRegistryService,
	redisCache *rediscache.RedisCache,
	connectionManager *servicesource.ConnectionManager,
	reconCore *recon.ReconCore,
	chunkEngine *recon.ChunkStreamBucketEngine,
	cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer *handlershadow.BatchBuffer,
	logger *zap.Logger,
) {
	if natsConn == nil {
		return
	}
	natsConn.Subscribe("schema.config.reload", func(_ *nats.Msg) {
		registrySvc.ReloadAll(context.Background())
		redisCache.DeletePattern(context.Background(), "schema:*")
		flushBatchBufferSchemaCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
	})
	natsConn.Subscribe("connection.config.changed", func(msg *nats.Msg) {
		connKey := strings.TrimSpace(string(msg.Data))
		if connKey == "" {
			return
		}
		connectionManager.InvalidatePool(connKey)
		if reconCore != nil {
			reconCore.InvalidateMasterAgent(connKey)
		}
		if chunkEngine != nil {
			chunkEngine.InvalidateMasterAgent(connKey)
		}
		flushBatchBufferSchemaCaches(cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer)
		logger.Info("connection pool invalidated via NATS signal", zap.String("connection_code", connKey))
	})
}

// Tại server_setup.go:148 (trước mọi BatchBuffer/Runner để chạy CUỐI CÙNG theo LIFO onStop):
//   ws.RegisterOnStop(func() { connectionManager.Close() })
// Tại server_setup.go:525 (SAU khi chunkEngine đã khởi tạo xong, không còn capture biến mutable):
//   registerConfigReloadSubscribers(natsClient.Conn, registrySvc, redisCache, connectionManager, reconCore, chunkEngine, cdcBatchBuffer, snapshotBatchBuffer, healBatchBuffer, logger)
```

### 5.3. Code Demo `A4` + `A5` + `A3.4`: Chuẩn Hóa Hexagonal Domain + CQRS Command + Transaction `cdc-cms-service`

```go
// [cdc-cms-service/internal/domain/master/connection_dsn.go] — Pure Domain (0 infra import)
package master

import (
	"encoding/json"
	"net/url"
	"strings"
)

// ExtractMasterOptionsCredentials đọc (username, password) từ cả khóa cấu trúc ("username"/"password")
// lẫn chuỗi kết nối ("dsn"/"uri"/"url") trong options_json, bỏ qua giá trị mask ("***", "__KEEP__").
func ExtractMasterOptionsCredentials(optionsJSON string) (username, password string) {
	if strings.TrimSpace(optionsJSON) == "" {
		return "", ""
	}
	var opts map[string]interface{}
	if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil {
		return "", ""
	}
	for _, k := range []string{"username", "user"} {
		if v, ok := opts[k].(string); ok && strings.TrimSpace(v) != "" {
			username = strings.TrimSpace(v)
			break
		}
	}
	if v, ok := opts["password"].(string); ok {
		p := strings.TrimSpace(v)
		if p != "" && p != "***" && p != "__KEEP__" {
			password = p
		}
	}
	for _, dsnKey := range []string{"dsn", "uri", "url", "connection_string"} {
		if raw, ok := opts[dsnKey].(string); ok && strings.TrimSpace(raw) != "" {
			u, p := ExtractMasterDSNCredentials(raw)
			if username == "" {
				username = u
			}
			if password == "" {
				password = p
			}
			if username != "" && password != "" {
				break
			}
		}
	}
	return username, password
}

func ExtractMasterDSNCredentials(rawDSN string) (string, string) {
	trimmed := strings.TrimSpace(rawDSN)
	if trimmed == "" {
		return "", ""
	}
	if tcpIdx := strings.LastIndex(trimmed, "@tcp("); tcpIdx >= 0 && !strings.Contains(trimmed, "://") {
		userInfo := trimmed[:tcpIdx]
		if colon := strings.Index(userInfo, ":"); colon >= 0 {
			return userInfo[:colon], userInfo[colon+1:]
		}
		return userInfo, ""
	}
	if u, err := url.Parse(trimmed); err == nil && u.User != nil {
		pass, _ := u.User.Password()
		return u.User.Username(), pass
	}
	return "", ""
}
```

```go
// [cdc-cms-service/internal/infra/persistence/master/master_connection_options_sync.go]
// Bọc nguyên tử cả UPDATE connection_registry và syncMasterBindingsDatabase trong 1 DB Transaction:
func (r *masterRepoGorm) execUpdateConnectionAndSyncBindingsTx(
	ctx context.Context,
	execConnUpdate func(tx *gorm.DB) error,
	id int64,
	engineType string,
	defaultDatabase *string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := execConnUpdate(tx); err != nil {
			return err
		}
		return syncMasterBindingsDatabaseTx(tx, id, engineType, defaultDatabase)
	})
}
```

### 5.4. Code Demo `A3.5` + `A6`: Tách Form Helper `MasterRegistry.tsx` (Re-Export Tương Thích Ngược) & Bổ Sung `"does not exist on target database"`

```ts
// [cdc-cms-web/src/components/master/masterConnectionForm.ts]
import { getMasterEngineMeta, normalizeMasterEngineTypeUI, type MasterEngineType } from './masterEngineConfig';

export interface MasterConnectionItem {
  id: number;
  connection_code: string;
  display_name: string;
  role_type?: string;
  engine_type?: string;
  host?: string;
  port?: number;
  default_database?: string;
  default_schema?: string;
  username?: string;
  sslmode?: string;
  auth_source?: string;
  replica_set?: string;
  status: string;
}

export function mapMasterConnectionToEditForm(item: MasterConnectionItem) {
  const normalizedEngine: MasterEngineType = normalizeMasterEngineTypeUI(item.engine_type);
  const meta = getMasterEngineMeta(normalizedEngine);
  return {
    connection_code: item.connection_code,
    display_name: item.display_name,
    engine_type: normalizedEngine,
    host: item.host || '',
    port: item.port || meta.defaultPort,
    default_database: item.default_database || '',
    default_schema: item.default_schema || (meta.isDatabaseScoped ? '' : 'public'),
    username: item.username || '',
    password: '',
    sslmode: item.sslmode || 'disable',
    auth_source: item.auth_source || 'admin',
    replica_set: item.replica_set || '',
  };
}
// Tại MasterRegistry.tsx: giữ `export type { MasterConnectionItem } from '../components/master/masterConnectionForm';`
```
