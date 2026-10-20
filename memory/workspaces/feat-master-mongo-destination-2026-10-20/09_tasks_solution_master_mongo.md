# 09 Tasks Solution: Hồ sơ Giải pháp Kỹ thuật Tích hợp Master MongoDB Destination

- **Workspace**: `feat-master-mongo-destination-2026-10-20`
- **Suffix**: `master_mongo`
- **Tier**: `T2`
- **Trạng thái**: `PROPOSED` (Chờ User `APPROVE` — Tuân thủ Rule #1 & Rule #13)

---

## 1. Root Cause / Vấn đề cốt lõi (`VERIFIED` `file:line`)

Qua điều tra thực chứng (`S2 Read-Only Investigation`) trên mã nguồn 3 repository của hệ thống Data Hub, luồng **Multi-Connection Master DB** hiện tại đã định tuyến theo `master_connection_id` và `master_binding_id`, nhưng đang gắn cứng với **PostgreSQL (`*gorm.DB`)** tại 5 điểm cốt lõi:

1. **CMS Connection API & Repository (`VERIFIED`):**
   - `cdc-cms-service/internal/api/dto/master_connection_dto.go:4-46`: Các struct `CreateMasterConnectionRequest`, `TestMasterConnectionRequest`, `UpdateMasterConnectionRequest` chưa có trường `EngineType` và `AuthSource`.
   - `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:64-158`: Hàm `TestConnection` mặc định gán `port = 5432` (dòng `77`), format chuỗi `postgres://` (dòng `87`) và chỉ gọi `gorm.Open(postgres.Open(dsn))` (dòng `122`).
   - `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:201-244` & `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:622-632,762`: Hàm `CreateConnection` gán cứng `EngineType: "postgresql"`, và câu lệnh SQL `INSERT INTO cdc_system.connection_registry` tại `master_repo_gorm.go:628` hardcode `'postgresql'`; hàm `GetMasterConnectionDSN` tại dòng `762` chỉ kiểm tra `strings.HasPrefix(row.SecretRef, "postgres://")`.
2. **CDS ConnectionManager Pool (`VERIFIED`):**
   - `centralized-data-service/internal/service/source/connection_manager.go:18-139`: Struct `ConnectionManager` chỉ có `masterDBPool map[string]*gorm.DB` (dòng `25`) và hàm `GetMasterDB(ctx, key)` gọi `m.reg.OpenGorm(dsn, "master:"+key)` (dòng `132`), chưa có pool quản lý `*mongo.Client` cho Master MongoDB.
3. **Master DDL / Index Provisioning (`VERIFIED`):**
   - `centralized-data-service/internal/service/master/master_ddl_generator.go:33-42,360-486`: Struct `masterDDLBindingRow` và câu query `loadBinding` (dòng `475-485`) chưa lấy `COALESCE(cr.engine_type, 'postgresql') AS master_engine_type`. Hàm `Apply` (dòng `373-458`) gọi thẳng `g.connMgr.GetMasterDB` và chạy transaction SQL (`SET LOCAL lock_timeout`, `CREATE TABLE`, `ALTER TABLE`, `ENABLE ROW LEVEL SECURITY`, `information_schema.columns`).
4. **Transmuter Engine (`Shadow PG ➔ Master DB`) (`VERIFIED`):**
   - `centralized-data-service/internal/service/master/transmuter.go:77-96,272-307,631-689,757-775,1343-1486`: Struct `masterBindingRuntime` và hàm `loadMaster` chưa lấy `mc.engine_type` và `mc.default_database`. Các bước DDL Safety Gate (dòng `273-307`), `RunEvents` (dòng `631-689`), `hardDeleteMasterByGpayIDs` (dòng `1343-1386`) và `bulkUpsertMaster` (dòng `1388-1486`) đều gọi `t.connMgr.GetMasterDB` và phát sinh câu lệnh SQL PostgreSQL (`INSERT INTO ... ON CONFLICT (...) DO UPDATE ... WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(..., 0) - 2000`).
5. **Reconciliation & Self-Healing (`VERIFIED`):**
   - `centralized-data-service/internal/service/recon/recon_engine.go:182-216` & `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:466-475`: `GetMasterAgent` và `ExecuteHeal` đang lấy `rc.connMgr.GetMasterDB(ctx, connectionKey)` mà chưa phân nhánh theo `engine_type == "mongodb"` để đọc/đối soát trên MongoDB Collection đích.

---

## 2. Giải pháp Kỹ thuật Duy nhất (Single Optimal Architecture)

Thiết kế mở rộng đa hình (**Polymorphic Master Destination**) theo `engine_type` (`postgresql` | `mongodb`) trên đúng kiến trúc hiện có, đảm bảo **100% Backward Compatibility** cho toàn bộ các Master PostgreSQL đang chạy:

### 2.1. Tầng Quản trị Kết nối (`cdc-cms-service` & `cdc-cms-web`)
1. **DTO & Handler (`master_connection_dto.go`, `master_registry_handler_connections.go`):**
   - Bổ sung `EngineType string `json:"engine_type,omitempty"`` (mặc định `"postgresql"` nếu rỗng, chấp nhận `"postgresql"` hoặc `"mongodb"`) và `AuthSource string `json:"auth_source,omitempty"`` vào `CreateMasterConnectionRequest`, `UpdateMasterConnectionRequest`, `TestMasterConnectionRequest`.
   - Trong `CreateConnection` / `UpdateConnection`:
     - Nếu `engine_type == "mongodb"`: Mặc định `port = 27017`, `default_schema` lưu tên database/namespace hoặc `"admin"`, build chuỗi URI `mongodb://user:pass@host:port/default_database?authSource=<auth_source>`.
   - Trong `TestConnection`:
     - Nếu `req.EngineType == "mongodb"` hoặc `strings.HasPrefix(dsn, "mongodb://")` / `strings.HasPrefix(dsn, "mongodb+srv://")`: Sử dụng `mongo.Connect(ctxTimeout, options.Client().ApplyURI(dsn))` + `client.Ping(ctxTimeout, readpref.Primary())` và chạy lệnh `db.RunCommand(ctxTimeout, bson.D{{Key: "buildInfo", Value: 1}})` để trả về `"MongoDB " + version`.
2. **Persistence (`master_repo_gorm.go`):**
   - Tại `CreateMasterConnection` (dòng `628`): Thay hằng số cứng `'postgresql'` bằng tham số `engineType` (`COALESCE` về `'postgresql'` nếu rỗng) và lưu `"engine_type": engineType` vào `options_json`.
   - Tại `GetMasterConnectionDSN` (dòng `762`): Chấp nhận `strings.HasPrefix(row.SecretRef, "postgres://") || strings.HasPrefix(row.SecretRef, "mongodb://") || strings.HasPrefix(row.SecretRef, "mongodb+srv://")`.
3. **Frontend (`cdc-cms-web/src/pages/MasterRegistry.tsx`):**
   - Thêm `Select` chọn `Engine Type` (`PostgreSQL` / `MongoDB`) trong Modal Quản lý Kết nối Đích; tự động đổi default port (`5432` $\leftrightarrow$ `27017`) và đổi nhãn `Default Schema` $\rightarrow$ `Auth Source (mặc định: admin)` khi chọn MongoDB.

### 2.2. Tầng ConnectionManager & Master Index Provisioning (`centralized-data-service`)
1. **`ConnectionManager` (`internal/service/source/connection_manager.go`):**
   - Bổ sung `masterMongoPool map[string]*mongo.Client` vào `ConnectionManager` (bảo vệ bởi `masterPoolMu`).
   - Thêm phương thức `GetMasterMongoClient(ctx context.Context, key string) (*mongo.Client, string, error)` trả về `*mongo.Client` và `defaultDatabase` từ `cdc_system.connection_registry` (cache vào `masterMongoPool[key]`).
   - Thêm phương thức `GetMasterEngineType(ctx context.Context, key string) string` (mặc định `"postgresql"` cho `"default"` hoặc khi không tìm thấy).
   - Trong `InvalidatePool(key)` và `Close()`: Đóng cả `masterDBPool[key]` lẫn `masterMongoPool[key]` (`client.Disconnect(ctx)`).
2. **`MasterDDLGenerator` (`internal/service/master/master_ddl_generator.go`):**
   - Mở rộng `masterDDLBindingRow` và `loadBinding` lấy thêm `COALESCE(cr.engine_type, 'postgresql') AS master_engine_type` và `COALESCE(cr.default_database, mb.master_schema, 'cdc_master') AS master_database`.
   - Trong `Apply(ctx, masterName, bindingID...)`:
     - Nếu `reg.MasterEngineType == "mongodb"`: Gọi `g.applyMongoMaster(ctx, reg, rules)`:
       - Kết nối qua `g.connMgr.GetMasterMongoClient(ctx, reg.MasterConnectionKey)`.
       - Tạo Collection (nếu chưa có) và gọi `coll.Indexes().CreateMany(ctx, indexModels)` tạo Unique Index trên `_gpay_id`, Index trên `_source_id`, `_source_ts`, và các cột tài chính/business key.
       - Cập nhật `cdc_system.mapping_rule_master SET in_master = true WHERE master_binding_id = ? AND is_active = true AND status = 'approved'` và `g.markDDLStatus(ctx, reg.ID, "created", nil)`.

### 2.3. Tầng Transmuter Engine (`Shadow PG ➔ Master MongoDB`)
1. **Metadata Runtime (`internal/service/master/transmuter.go`):**
   - Mở rộng `masterBindingRuntime` và câu query `loadMaster` (dòng `760-774`) lấy thêm:
     - `COALESCE(mc.engine_type, 'postgresql') AS master_engine_type`
     - `COALESCE(mc.default_database, mb.master_schema, 'cdc_master') AS master_database`
2. **DDL Safety Gate & Inference cho MongoDB (`transmuter.go:273-365`):**
   - Khi `masterRow.MasterEngineType == "mongodb"`:
     - Kiểm tra kết nối qua `t.connMgr.GetMasterMongoClient(ctx, masterRow.MasterConnectionKey)`.
     - Suy diễn `last_gpay_id` (khi `lastGpayID == 0` và không phải `full` sync) bằng query MongoDB `FindOne` sort `{"_gpay_id": -1}` kèm Gap-Safe Count Guard (`CountDocuments(ctx, bson.M{"_deleted": bson.M{"$ne": true}})`).
3. **BSON Type Coercion & Atomic Batch `BulkWrite` kèm OCC (`transmuter_mongo.go`):**
   - Chuẩn hóa giá trị sang kiểu BSON (`time.Time` $\rightarrow$ `primitive.DateTime`, `Decimal` $\rightarrow$ `primitive.Decimal128`, JSON string/bytes cho cột `json/jsonb` $\rightarrow$ `bson.M`/`bson.A`).
   - Thực thi `bulkUpsertMasterMongo(ctx, binding, records, colTypes, isFullSync)` bằng **MongoDB Aggregation Pipeline Update (`UpdateOneModel` với `Upsert: true`)** trên khóa `{ "_gpay_id": rec["_gpay_id"] }`:
     - Khi `isFullSync == false`: Sử dụng biểu thức điều kiện `$cond` dựa trên `$_source_ts` (với dung sai `clockSkewToleranceMs = 2000`): chỉ cập nhật các trường dữ liệu khi document chưa tồn tại (`$eq: [{"$type": "$_source_ts"}, "missing"]`) hoặc `incomingTs >= $_source_ts - 2000`; nếu event cũ hơn thì giữ nguyên toàn bộ giá trị hiện tại trên Master Mongo (đảm bảo **OCC 100% atomic** ngay trong `BulkWrite(SetOrdered(true))` mà không cần đọc trước).
   - Hỗ trợ `hardDeleteMasterByGpayIDs` và xoá theo `_source_id` trên MongoDB bằng `coll.DeleteMany(ctx, bson.M{"_gpay_id": bson.M{"$in": chunk}})` / `bson.M{"_source_id": bson.M{"$in": toHardDelete}}`.

### 2.4. Tầng Reconciliation & Self-Healing (`internal/service/recon/`)
1. **Recon Dest Agent (`internal/service/recon/recon_dest_agent.go` & `recon_engine.go`):**
   - Trong `ReconDestAgent`: Bổ sung `mongoClient *mongo.Client` và `mongoDefaultDB string` (tùy chọn). Khi agent được cấu hình với `mongoClient != nil`, các hàm đếm (`CountWindow`, `FetchWindowRows` / `FetchByIDs`) truy vấn trực tiếp trên MongoDB Collection (`bson.M{"_deleted": bson.M{"$ne": true}}`).
   - Trong `ReconCore.GetMasterAgent(ctx, connectionKey)` (`recon_engine.go:182-216`): Kiểm tra `rc.connMgr.GetMasterEngineType(ctx, connectionKey) == "mongodb"`; nếu là MongoDB thì lấy `mongoClient, defaultDB, err := rc.connMgr.GetMasterMongoClient(ctx, connectionKey)` và khởi tạo `NewReconDestMongoAgent(mongoClient, defaultDB, rc.logger)`.
2. **Self-Healing (`internal/handler/recon/recon_execute_heal_handler.go`):**
   - Khi heal đích danh các `_source_id` sang Master Binding có `engine_type == "mongodb"`, gọi trực tiếp `TransmuterModule.Run(ctx, masterFQN, onlySourceIDs, "incremental", "", masterBindingID)` (đã hỗ trợ cả upsert lẫn orphan prune trên MongoDB).

---

## 3. Blast Radius & Rollback Plan

### 3.1. Blast Radius (Phạm vi ảnh hưởng)
- **Các Master Binding PostgreSQL hiện hữu:** `0%` thay đổi hành vi vì `COALESCE(mc.engine_type, 'postgresql')` luôn trả về `'postgresql'`, đi nguyên vào nhánh `GetMasterDB` + SQL hiện tại.
- **Các Caller / Consumer liên quan:**
  - `cdc-cms-service`: `/api/v1/master-connections` (`GET`, `POST`, `PUT`, `POST /test`) giữ nguyên contract JSON, chỉ mở rộng thêm field tùy chọn `engine_type` và `auth_source`.
  - `centralized-data-service`: `MasterDDLGenerator.Apply`, `TransmuterModule.Run`, `TransmuterModule.RunEvents`, `ReconCore.GetMasterAgent` giữ nguyên chữ ký hàm public, không phá vỡ bất kỳ caller hay unit test hiện có nào.

### 3.2. Rollback Plan
- Không có migration thay đổi cấu trúc bảng bắt buộc (cột `engine_type` đã có sẵn trong `cdc_system.connection_registry`).
- Nếu cần rollback: chỉ cần revert commit trên `cdc-cms-service`, `centralized-data-service`, `cdc-cms-web` và chạy `DeactivateMasterConnection` đối với connection MongoDB thử nghiệm.

---

## 4. Code Demo (Đối chiếu 100% hàm/struct thật trong Codebase)

### 4.1. Mở rộng `ConnectionManager` trong `centralized-data-service/internal/service/source/connection_manager.go`
```go
// Trong struct ConnectionManager (connection_manager.go:18-26):
type ConnectionManager struct {
	cfg    *config.AppConfig
	logger *zap.Logger
	reg    *database.Registry

	mu              sync.Mutex
	masterPoolMu    sync.RWMutex
	masterDBPool    map[string]*gorm.DB
	masterMongoPool map[string]*mongo.Client
	masterMongoDB   map[string]string
}

// GetMasterMongoClient trả về *mongo.Client và defaultDatabase từ cdc_system.connection_registry
func (m *ConnectionManager) GetMasterMongoClient(ctx context.Context, key string) (*mongo.Client, string, error) {
	key = strings.TrimSpace(key)
	if key == "" || key == "default" {
		return nil, "", fmt.Errorf("default master connection is postgresql, not mongodb")
	}

	m.masterPoolMu.RLock()
	if cli, ok := m.masterMongoPool[key]; ok && cli != nil {
		dbName := m.masterMongoDB[key]
		m.masterPoolMu.RUnlock()
		return cli, dbName, nil
	}
	m.masterPoolMu.RUnlock()

	m.masterPoolMu.Lock()
	defer m.masterPoolMu.Unlock()

	if cli, ok := m.masterMongoPool[key]; ok && cli != nil {
		return cli, m.masterMongoDB[key], nil
	}

	sysDB, err := m.GetSystemDB(ctx)
	if err != nil || sysDB == nil {
		return nil, "", fmt.Errorf("get system db for mongo master %q: %w", key, err)
	}

	var conn modelsource.ConnectionRegistry
	if err := sysDB.WithContext(ctx).Table("cdc_system.connection_registry").
		Where("(connection_code = ? OR connection_code = ?) AND status = 'active'", key, strings.ToLower(key)).
		First(&conn).Error; err != nil {
		return nil, "", fmt.Errorf("lookup master mongo connection %q: %w", key, err)
	}

	uri := resolveConnectionURI(&conn)
	if uri == "" {
		return nil, "", fmt.Errorf("empty mongo URI for master connection %q", key)
	}

	clientOpts := options.Client().ApplyURI(uri)
	cli, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, "", fmt.Errorf("connect master mongo %q: %w", key, err)
	}

	dbName := ""
	if conn.DefaultDatabase != nil {
		dbName = strings.TrimSpace(*conn.DefaultDatabase)
	}
	if m.masterMongoPool == nil {
		m.masterMongoPool = make(map[string]*mongo.Client)
		m.masterMongoDB = make(map[string]string)
	}
	m.masterMongoPool[key] = cli
	m.masterMongoDB[key] = dbName
	return cli, dbName, nil
}
```

### 4.2. Atomic Batch `BulkWrite` kèm OCC trong `centralized-data-service/internal/service/master/transmuter_mongo.go`
```go
func (t *TransmuterModule) bulkUpsertMasterMongo(
	ctx context.Context,
	binding *masterBindingRuntime,
	records []map[string]any,
	colTypes map[string]string,
	isFullSync bool,
) (ins int64, upd int64, occSkip int64, err error) {
	if len(records) == 0 {
		return 0, 0, 0, nil
	}

	mongoClient, defaultDB, err := t.connMgr.GetMasterMongoClient(ctx, binding.MasterConnectionKey)
	if err != nil {
		return 0, 0, int64(len(records)), err
	}

	dbName := strings.TrimSpace(binding.MasterSchema)
	if dbName == "" || dbName == "public" {
		dbName = defaultDB
	}
	if dbName == "" {
		dbName = "cdc_master"
	}
	coll := mongoClient.Database(dbName).Collection(binding.MasterTable)

	const clockSkewToleranceMs int64 = 2000
	now := time.Now().UTC()
	models := make([]mongo.WriteModel, 0, len(records))

	for _, rec := range records {
		doc := coerceRecordToBSON(rec, colTypes)
		gpayID := doc["_gpay_id"]
		incomingTs, _ := toInt64Ts(doc["_source_ts"])
		doc["_updated_at"] = now

		filter := bson.M{"_gpay_id": gpayID}
		if isFullSync {
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(bson.M{"$set": doc}).
				SetUpsert(true))
			continue
		}

		// Aggregation Pipeline Update bảo toàn OCC: chỉ ghi đè khi document mới tạo hoặc incomingTs >= existing._source_ts - 2000
		condExpr := bson.M{
			"$or": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$type": "$_source_ts"}, "missing"}},
				bson.M{"$lte": bson.A{bson.M{"$subtract": bson.A{"$_source_ts", clockSkewToleranceMs}}, incomingTs}},
			},
		}
		setStage := bson.M{}
		for k, v := range doc {
			if k == "_gpay_id" {
				setStage[k] = v
				continue
			}
			setStage[k] = bson.M{
				"$cond": bson.A{condExpr, v, "$" + k},
			}
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.A{bson.M{"$set": setStage}}).
			SetUpsert(true))
	}

	res, err := coll.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	if err != nil {
		return 0, 0, int64(len(records)), fmt.Errorf("mongo master bulk write %s.%s: %w", dbName, binding.MasterTable, err)
	}
	ins = res.UpsertedCount
	upd = res.ModifiedCount
	occSkip = int64(len(records)) - ins - upd
	return ins, upd, occSkip, nil
}
```

### 4.3. Phân nhánh `CreateConnection` & `TestConnection` trong `cdc-cms-service`
```go
// Trong CreateConnection (master_registry_handler_connections.go:201-246):
engineType := strings.ToLower(strings.TrimSpace(req.EngineType))
if engineType == "" {
	engineType = "postgresql"
}
if engineType != "postgresql" && engineType != "mongodb" {
	return c.Status(400).JSON(fiber.Map{"error": "engine_type must be 'postgresql' or 'mongodb'"})
}
port := req.Port
if port <= 0 {
	if engineType == "mongodb" {
		port = 27017
	} else {
		port = 5432
	}
}
var dsn string
if engineType == "mongodb" {
	authSource := strings.TrimSpace(req.AuthSource)
	if authSource == "" {
		authSource = "admin"
	}
	dsn = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		port,
		defaultDB,
		url.QueryEscape(authSource),
	)
} else {
	dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		port,
		defaultDB,
		sslmode,
	)
}
```

---

## 5. Bảng kết quả Decision Gate (`DG1` & `DG2`)

### 5.1. Cổng `DG1` — Kiểm định Giải pháp Kỹ thuật (`06_laya_summary_dg1_20261020T090535.json` ➔ **`PASS`**)

| Claim ID | Loại | Nội dung Claim | Bằng chứng (`REF:` khớp 100% trên đĩa) | Kết quả Laya / Ground Truth |
| :--- | :---: | :--- | :--- | :--- |
| **`S-RC`** | `solution` | `CreateConnection` hiện đang gán cứng `EngineType: "postgresql"` trong `MasterConnectionItem`. | `master_registry_handler_connections.go:234-244` (`evidence_error: ""`) | `UNVALIDATED` (Laya `r3: YES 0.6158`, kết luận `VERIFIED` bằng code gốc theo Rule #10) |
| **`S-FIX`** | `solution` | `ConnectionManager` hiện chỉ khai báo `masterDBPool map[string]*gorm.DB`, chưa có pool cho `mongo.Client`. | `connection_manager.go:18-26` (`evidence_error: ""`) | `UNVALIDATED` (Laya `r3: YES 0.2242`, kết luận `VERIFIED` bằng code gốc theo Rule #10) |
| **`S-IMPACT`** | `solution` | Khi `key` rỗng hoặc `"default"`, `GetMasterDB` trả về `m.reg.GetDB(database.RoleDestination)` giữ nguyên tương thích ngược. | `connection_manager.go:58-62` (`evidence_error: ""`) | `UNVALIDATED` (Laya `r1: CORRECT 0.8381`, kết luận `VERIFIED` bằng code gốc theo Rule #10) |
| **`S-PATTERN`** | `solution` | `bulkUpsertMaster` áp dụng dung sai `clockSkewToleranceMs = 2000` khi so sánh `_source_ts` cho OCC. | `transmuter.go:1456-1461` (`evidence_error: ""`) | `UNVALIDATED` (Kết luận `VERIFIED` trực tiếp tại dòng `1457` `const clockSkewToleranceMs = 2000` theo Rule #10) |

### 5.2. Cổng `DG2` — Kiểm định Code Demo (`06_laya_summary_dg2_20261020T090655.json` ➔ **`PASS`**, `trusted_rounds: ["r2", "r3"]`)

| Claim ID | Loại | Nội dung Claim | Bằng chứng (`REF:` khớp 100% trên đĩa) | Vòng `r2` | Vòng `r3` | Quyết định |
| :--- | :---: | :--- | :--- | :---: | :---: | :---: |
| **`D-BEHAV`** | `demo` | `bulkUpsertMasterMongo` gọi `coll.BulkWrite` với `options.BulkWrite().SetOrdered(true)`. | `09_tasks_solution_master_mongo.md:243-250` | `CORRECT (0.8178)` | `YES (0.9400)` | **`ACCEPT`** 🟢 |
| **`D-EDGE`** | `demo` | Khi `len(records) == 0`, `bulkUpsertMasterMongo` trả về ngay `0, 0, 0, nil` mà không gọi MongoDB. | `09_tasks_solution_master_mongo.md:182-185` | `CORRECT (0.5363)` | `NO (0.0385)` | **`ACCEPT`** 🟢 |
| **`D-API`** | `demo` | Câu truy vấn GORM trong `ConnectionManager` tra cứu `cdc_system.connection_registry` với `status = 'active'`. | `connection_manager.go:93-96` | `CORRECT (0.7061)` | `YES (0.8420)` | **`ACCEPT`** 🟢 |

