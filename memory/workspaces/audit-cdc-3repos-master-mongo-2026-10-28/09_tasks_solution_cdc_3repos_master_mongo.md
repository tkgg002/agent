# 09 Tasks Solution: Tích hợp đồng thời 2 Feature (Event Date Split & Batching + Master MongoDB Destination) trên Codebase `2026-10-28`

- **Workspace**: `audit-cdc-3repos-master-mongo-2026-10-28`
- **Suffix**: `cdc_3repos_master_mongo`
- **Tier**: `T2`
- **Date**: `2026-10-28`

---

## 1. Kết quả Audit 3 Stash (`stash@{0..2}` ngày 22/10) đối chiếu với Codebase hiện tại (`2026-10-28`, `HEAD 786db91`)

### 1.1. Tại sao `stash@{0}`, `stash@{1}`, `stash@{2}` tuyệt đối KHÔNG ĐƯỢC `git stash apply`?
1. **3 stash thực chất là các snapshot trên cùng 1 commit gốc `43e5da7` (`2026-10-22`):**
   - `stash@{1}` và `stash@{2}` giống hệt nhau (`9 files changed, 599 insertions(+), 132 deletions(-)`) và **mất sạch 3 file untracked** (`mongo_type_converter.go`, `transmuter_mongo.go`, `recon_dest_mongo.go`).
   - `stash@{0}` chứa 9 file tracked + 5 file untracked nhưng được chụp ở commit `43e5da7` (ngày `2026-10-22`).
2. **Từ `43e5da7` (`22/10`) đến `786db91` (`28/10`) có 29 file thay đổi (`+2,977 / -949` dòng) cộng thêm working tree của Feature 1 (*Event Date Split & Batching*):**
   - Nếu chạy `git stash apply stash@{0}`, git sẽ tạo conflict hoặc đè ngược làm **mất/hỏng Feature 1** (`transmuter_events_batch.go`, `transmute_event_date_split.go`) và **gây 10 lỗi logic/panic nghiêm trọng (`OB-01` → `OB-10`)** dưới đây.

### 1.2. Danh sách 10 Bug & Điểm Outdated (`OB-01` → `OB-10`) phát hiện khi Audit Code ngày 22/10 so với Codebase 28/10

| Mã lỗi | File & Dòng trên Codebase `2026-10-28` | Mô tả lỗi trong `stash@{0}` (22/10) | Cách khắc phục triệt để trong Plan mới |
|---|---|---|---|
| **`OB-01`** | `internal/service/recon/recon_stream_bucket_engine.go:134-176, 720-728`<br>`internal/service/recon/recon_engine_segment_b.go:56`<br>`internal/service/recon/recon_dest_query.go:92-124` | Giữa 22/10 và 28/10, Recon Segment B (`2h/7d/30d`) đã chuyển sang dùng `ChunkStreamBucketEngine`. Engine này có hàm `ChunkStreamBucketEngine.GetMasterAgent` riêng (chỉ gọi `GetMasterDB` của Postgres) và gọi `msAgent.ColumnExists(ctx, masterRel, col)`. Bản stash 22/10 **bỏ sót hoàn toàn** `ChunkStreamBucketEngine.GetMasterAgent` và `ColumnExists` / `GetRealColumnName` không có nhánh `if da.IsMongo()`, dẫn tới **nil-pointer panic** (`da.replica == nil`) hoặc lỗi kết nối! | Cập nhật `ChunkStreamBucketEngine.GetMasterAgent` kiểm tra `e.connMgr.IsMongoMasterConnection` → trả về `NewReconDestMongoAgent`. Bổ sung nhánh `if da.IsMongo()` cho `ColumnExists` và `GetRealColumnName`. |
| **`OB-02`** | `internal/service/recon/recon_dest_query.go:350-450`<br>`internal/service/recon/recon_tier_b.go:389, 612-676, 774-848` | `RunDeepCheckB` gọi `msAgent.BucketCounts(...)` nhưng `BucketCounts` trong `recon_dest_query.go` không có nhánh `if da.IsMongo()`. Đồng thời `RunRowDiffB` (`recon_tier_b.go:612`) và `TimeBoundedDiffMissingFromMaster` (`recon_tier_b.go:774`) gọi thẳng `rc.connMgr.GetMasterDB` và chạy raw SQL `SELECT * FROM ...` mà không kiểm tra MongoDB Master. | Bổ sung `da.mongoBucketCounts` trong `recon_dest_mongo.go` + nhánh `if da.IsMongo()` ở `BucketCounts`. Bổ sung nhánh MongoDB trong `RunRowDiffB` và `TimeBoundedDiffMissingFromMaster`. |
| **`OB-03`** | `internal/handler/recon/recon_execute_heal_segb.go:100-140, 340-360, 528-546` | Commit `786db91` đã tách toàn bộ logic Heal Segment B từ `recon_execute_heal_handler.go` sang file mới `recon_execute_heal_segb.go`. Ngoài ra, ở dòng `528-546` (`resolveSourceIDsForSegmentB` khi `effectiveIDType == "gpay"` và row đã bị xóa cứng khỏi Shadow), code chỉ gọi `h.resolveMasterDBForRef` (PostgreSQL) mà thiếu nhánh đọc `_gpay_id` từ MongoDB Master! | Đưa `isMongoMasterRef` và `softDeleteMongoMasterOrphans` vào `recon_execute_heal_segb.go`, đồng thời thêm nhánh `if h.isMongoMasterRef(ctx, ref)` tại `resolveSourceIDsForSegmentB` (`528-546`) để tra cứu `pkCol` theo `_gpay_id` trên MongoDB Master. |
| **`OB-04`** | `internal/service/recon/recon_dest_mongo.go:103-112` | `mongoCountInWindow` truyền `deletedFilter = nil` vào `buildMongoWindowFilter(timestampField, tLo, tHi, nil)`, khiến các document đã xóa mềm (`_deleted: true`) vẫn bị đếm là active trong cửa sổ thời gian! | Truyền `bson.M{"_deleted": bson.M{"$ne": true}}` vào `mongoCountInWindow`. |
| **`OB-05`** | `internal/service/recon/recon_dest_mongo.go:74-96, 263-294` | `buildMongoWindowFilter` giả định trường timestamp nghiệp vụ (khác `_source_ts`) trên MongoDB Master chỉ lưu kiểu BSON `Date`. Nếu bảng lưu epoch ms (`int64`) hoặc ISO string thì query window trả về 0 document! | Trong `buildMongoWindowFilter`, khi `tsCol != "_source_ts"`, dùng `$or` kết hợp cả phạm vi `primitive.NewDateTimeFromTime` (BSON Date) và `[tLo.UnixMilli(), tHi.UnixMilli())` (`int64` epoch ms) để tương thích mọi schema. |
| **`OB-06`** | `internal/service/master/transmuter.go:296`<br>`internal/service/master/transmuter_identity.go:14-125` | Commit `3369c31` đã tách `transmuter_identity.go` và nâng cấp orphan cleanup dùng `deterministicMasterFlattenGpayID(masterRow.ID, row.SourceID, row.GpayID, suffix)`. Bản `stash@{0}` ngày 22/10 vẫn gọi hàm cũ `deterministicGpayID(row.GpayID, suffix)`! | Tuyệt đối giữ nguyên `transmuter_identity.go` và lời gọi `deterministicMasterFlattenGpayID` của codebase `2026-10-28`. |
| **`OB-07`** | `internal/service/master/transmuter_mongo.go:195-248`<br>`internal/service/master/transmuter_identity.go:15-17` | Theo thiết kế Sonyflake (`transmuter_identity.go:15-17`), khi `conflictTarget != "_gpay_id"` (VD: PK là `_id` hoặc `id`), mỗi lần chạy Transmuter sẽ sinh một `nextMasterSonyflakeGpayID()` mới; trên PostgreSQL `ON CONFLICT (<pk>) DO UPDATE` không bao giờ ghi đè `_gpay_id` của row đã tồn tại. Thế nhưng `BuildMongoOCCBulkModels` trong `stash@{0}` lại đưa `_gpay_id` vào `$set` thông thường, làm **mỗi lần update document trên MongoDB Master bị đổi mất `_gpay_id`**, hoặc văng lỗi duplicate key trên unique index `_gpay_id`! | Trong `BuildMongoOCCBulkModels`: khi `conflictTarget != "_gpay_id"`, ở nhánh pipeline update dùng `setStage["_gpay_id"] = bson.M{"$ifNull": bson.A{"$_gpay_id", v}}` (chỉ gán khi insert mới, giữ nguyên `$_gpay_id` cũ khi update); ở nhánh non-OCC (`isFullSync \|\| incomingTs == 0`) tách `_gpay_id` sang `"$setOnInsert": bson.M{"_gpay_id": v}`! |
| **`OB-08`** | `internal/service/master/master_ddl_generator.go:510-546` | (1) `applyMongoCollectionAndIndexes` thiếu tạo index trên `conflict_target` (business PK lấy từ `reg.TransformSpec`), khiến `BulkWrite` upsert theo `{conflictTarget: keyVal}` bị `COLLSCAN` trên bảng lớn.<br>(2) Câu lệnh `UPDATE cdc_system.mapping_rule_master SET in_master = true` ở dòng `543` thiếu điều kiện `AND status = 'approved'`. | (1) Trích xuất `pk` từ `reg.TransformSpec`; nếu `pk != "" && pk != "_gpay_id" && pk != "_id"`, tạo thêm unique index `idx_master_pk_uniq` trên `pk`.<br>(2) Thêm `AND status = 'approved'` vào câu `UPDATE mapping_rule_master`. |
| **`OB-09`** | `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go:604-625, 648-654, 710-725`<br>`cdc-cms-web/src/pages/MasterRegistry.tsx:402-419` | `CreateConnection` và `UpdateConnection` trong `master_repo_gorm.go` chỉ ghi `sslmode` vào `options_json`, làm rơi mất `auth_source` và `replica_set`. `ListConnectionsWithFilter` và `openEditConnection` trên `MasterRegistry.tsx` cũng chưa nạp lại 2 trường này. | Lưu `auth_source` & `replica_set` vào `options_json` khi Create/Update; đọc lại trong `ListConnectionsWithFilter`; gán vào `connForm` trong `openEditConnection`. |
| **`OB-10`** | `internal/service/sourcecount/source_count_runner.go:165-176` | `SourceCountRunner` ở nhánh `case "master":` chỉ gọi `r.connMgr.GetMasterDB` và `r.countSQL(...)`, sẽ lỗi nếu chạy daily count trên bảng Master MongoDB. | Kiểm tra `if strings.ToLower(params.EngineType) == "mongodb" \|\| (r.connMgr != nil && r.connMgr.IsMongoMasterConnection(queryCtx, params.ConnectionCode))` thì gọi `r.countMongoDB` (hoặc đếm qua `GetMasterMongoClient`). |

---

## 2. Kiến trúc Hợp nhất 2 Feature trên Codebase `2026-10-28` (1 Phương án Tối ưu Duy nhất)

### 2.1. Sơ đồ Tương tác giữa Feature 1 (Event Date Split & Batching) và Feature 2 (Master MongoDB Destination)

```
NATS JetStream (CDC Events)
   │
   ▼
[Feature 1] splitEventPayloadsBySourceDay (transmute_event_date_split.go)
   │  Chia nhỏ dải ngày + truyền EventItems vào TransmuterModule.RunEvents
   ▼
TransmuterModule.RunEvents (transmuter.go:640-713)
   ├── Kiểm tra Table/Collection tồn tại:
   │     ├─ PostgreSQL: verifyTargetTableExists(ctx, masterDB, schema, table)
   │     └─ MongoDB:    verifyMongoTargetCollectionExists(ctx, masterRow)  <-- [Feature 2]
   │
   ▼
[Feature 1] t.runEventBatches (transmuter_events_batch.go:26-203)
   ├── Gom lô Upsert events (tối đa transmuteEventBatchSize = 200)
   │     └─ Gọi t.bulkUpsertMaster(ctx, masterRow, recs, colTypes, false)
   │           ├─ PostgreSQL: INSERT ... ON CONFLICT DO UPDATE WHERE _source_ts <= EXCLUDED._source_ts
   │           └─ MongoDB:    t.bulkUpsertMongoMaster(...)                 <-- [Feature 2]
   │                            └─ BuildMongoOCCBulkModels ($cond OCC + $ifNull _gpay_id)
   │
   └── Xử lý Delete events
         └─ Gọi t.deleteFromMasterBySourceAndGpayIDs(ctx, masterRow, colTypes, sourceIDs, gpayIDs)
               ├─ PostgreSQL: UPDATE ... SET _deleted = true
               └─ MongoDB:    t.deleteFromMongoMasterBySourceAndGpayIDs(...) <-- [Feature 2]
```

**Điểm mấu chốt giúp 2 Feature kết hợp hoàn hảo 100%:**
- Feature 1 (`transmuter_events_batch.go`) gom batch ở tầng điều phối (`runEventBatches`) rồi gọi xuống 2 hàm lõi `t.bulkUpsertMaster` và `t.deleteFromMasterBySourceAndGpayIDs`.
- Feature 2 (`transmuter_mongo.go`) đặt điểm rẽ nhánh `if t.isMongoMaster(ctx, masterRow)` ngay đầu `bulkUpsertMaster` và `deleteFromMasterBySourceAndGpayIDs`.
- Nhờ vậy, **không cần sửa bất kỳ dòng logic gom batch nào trong `transmuter_events_batch.go`** mà toàn bộ luồng Event Date Split & Batching tự động chạy batch `BulkWrite` thẳng xuống MongoDB Master!

---

## 3. Code Demo Chi tiết cho từng Component (Đã fix trọn vẹn `OB-01` → `OB-10`)

### 3.1. `cdc-cms-service` & `cdc-cms-web` (Fix `OB-09` — Giữ nguyên `activity_log_read_repo_gorm.go`)

#### File 1: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_repo_gorm.go`
```go
// Trong CreateConnection (dòng 604-625):
func (r *GormMasterRepo) CreateConnection(ctx context.Context, row MasterConnectionRow) (int64, error) {
	optionsMap := map[string]string{
		"sslmode": row.SSLMode,
	}
	if strings.TrimSpace(row.AuthSource) != "" {
		optionsMap["auth_source"] = strings.TrimSpace(row.AuthSource)
	}
	if strings.TrimSpace(row.ReplicaSet) != "" {
		optionsMap["replica_set"] = strings.TrimSpace(row.ReplicaSet)
	}
	optionsBytes, _ := json.Marshal(optionsMap)
	optionsJSON := string(optionsBytes)
	// ... giữ nguyên INSERT INTO cdc_system.connection_registry ...
}

// Trong ListConnectionsWithFilter (dòng 648-658):
	for i := range rows {
		if rows[i].OptionsJSON != "" {
			var opts map[string]interface{}
			if err := json.Unmarshal([]byte(rows[i].OptionsJSON), &opts); err == nil {
				if ssl, ok := opts["sslmode"].(string); ok {
					rows[i].SSLMode = ssl
				}
				if authSource, ok := opts["auth_source"].(string); ok {
					rows[i].AuthSource = authSource
				}
				if replicaSet, ok := opts["replica_set"].(string); ok {
					rows[i].ReplicaSet = replicaSet
				}
			}
		}
	}

// Trong UpdateConnection (dòng 710-725):
func (r *GormMasterRepo) UpdateConnection(ctx context.Context, row MasterConnectionRow) error {
	optionsMap := map[string]string{
		"sslmode": row.SSLMode,
	}
	if strings.TrimSpace(row.AuthSource) != "" {
		optionsMap["auth_source"] = strings.TrimSpace(row.AuthSource)
	}
	if strings.TrimSpace(row.ReplicaSet) != "" {
		optionsMap["replica_set"] = strings.TrimSpace(row.ReplicaSet)
	}
	optionsBytes, _ := json.Marshal(optionsMap)
	optionsJSON := string(optionsBytes)
	// ... giữ nguyên UPDATE cdc_system.connection_registry ...
}
```

#### File 2: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx`
```tsx
// Trong openEditConnection (dòng 402-419):
  const openEditConnection = (c: MasterConnection) => {
    setEditingConn(c);
    setConnForm({
      connection_code: c.connection_code,
      display_name: c.display_name || '',
      engine_type: c.engine_type === 'mongodb' ? 'mongodb' : 'postgresql',
      host: c.host || '',
      port: String(c.port || (c.engine_type === 'mongodb' ? 27017 : 5432)),
      default_database: c.default_database || '',
      default_schema: c.default_schema || (c.engine_type === 'mongodb' ? '' : 'public'),
      username: c.username || '',
      password: '',
      sslmode: c.sslmode || 'disable',
      auth_source: c.auth_source || 'admin',
      replica_set: c.replica_set || '',
    });
    setTestResult(null);
    setShowConnModal(true);
  };
```

---

### 3.2. `centralized-data-service` — `ConnectionManager` & `MasterDDLGenerator` (Fix `OB-08`)

#### File 3: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go`
- Bổ sung `MasterMongoHandle{Client *mongo.Client, DefaultDatabase string, EngineType string}`, `masterMongoPool map[string]*MasterMongoHandle`, `masterEngineMap map[string]string`.
- Bổ sung `IsMongoMasterConnection(ctx, key string) bool` và `GetMasterMongoClient(ctx, key string) (*MasterMongoHandle, error)` hỗ trợ đọc `auth_source`, `replica_set` từ `options_json` hoặc `secret_ref`.
- Trong `GetMasterDB(ctx, key)`: nếu `strings.EqualFold(row.EngineType, "mongodb")` thì lưu `m.masterEngineMap[key] = "mongodb"` và trả lỗi tường minh `"connection '%s' is a MongoDB master destination; use GetMasterMongoClient instead of GetMasterDB"`.

#### File 4: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go` (Fix `OB-08`)
```go
func (g *MasterDDLGenerator) applyMongoCollectionAndIndexes(ctx context.Context, reg masterRegisterRow) (DDLResult, error) {
	res := DDLResult{MasterName: reg.MasterTable}
	handle, err := g.connMgr.GetMasterMongoClient(ctx, reg.MasterConnectionKey)
	if err != nil {
		res.Err = err.Error()
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return res, fmt.Errorf("get master mongo client (%s): %w", reg.MasterConnectionKey, err)
	}

	dbName := strings.TrimSpace(reg.MasterSchema)
	if dbName == "" || dbName == "public" {
		dbName = handle.DefaultDatabase
	}
	if dbName == "" {
		dbName = "cdc_master"
	}

	mongoDB := handle.Client.Database(dbName)
	if cErr := mongoDB.CreateCollection(ctx, reg.MasterTable); cErr != nil {
		errLower := strings.ToLower(cErr.Error())
		if !strings.Contains(errLower, "already exists") && !strings.Contains(errLower, "namespaceexists") {
			res.Err = cErr.Error()
			g.markDDLStatus(ctx, reg.ID, "failed", cErr)
			return res, fmt.Errorf("create mongo collection %s.%s: %w", dbName, reg.MasterTable, cErr)
		}
	} else {
		res.TableCreated = true
	}

	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_gpay_id", Value: 1}},
			Options: options.Index().SetName("idx_master_gpay_id_uniq").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "_source_id", Value: 1}},
			Options: options.Index().SetName("idx_master_source_id"),
		},
		{
			Keys:    bson.D{{Key: "_source_ts", Value: -1}},
			Options: options.Index().SetName("idx_master_source_ts"),
		},
		{
			Keys:    bson.D{{Key: "_deleted", Value: 1}},
			Options: options.Index().SetName("idx_master_deleted"),
		},
	}

	// [FIX OB-08]: Tạo unique/lookup index trên business PK (conflict_target) nếu khác _gpay_id và _id
	if len(reg.TransformSpec) > 0 {
		var spec struct {
			PK string `json:"pk"`
		}
		if err := json.Unmarshal(reg.TransformSpec, &spec); err == nil {
			pk := strings.TrimSpace(spec.PK)
			if pk != "" && pk != "_gpay_id" && pk != "_id" {
				indexModels = append(indexModels, mongo.IndexModel{
					Keys:    bson.D{{Key: pk, Value: 1}},
					Options: options.Index().SetName("idx_master_pk_" + pk),
				})
			}
		}
	}

	if _, iErr := mongoDB.Collection(reg.MasterTable).Indexes().CreateMany(ctx, indexModels); iErr != nil {
		res.Err = iErr.Error()
		g.markDDLStatus(ctx, reg.ID, "failed", iErr)
		return res, fmt.Errorf("create mongo indexes on %s.%s: %w", dbName, reg.MasterTable, iErr)
	}

	// [FIX OB-08]: Chỉ đánh dấu in_master = true cho các rule đã approved!
	if g.systemDB != nil {
		_ = g.systemDB.WithContext(ctx).Exec(
			`UPDATE cdc_system.mapping_rule_master SET in_master = true WHERE master_binding_id = ? AND is_active = true AND status = 'approved'`,
			reg.ID,
		).Error
	}
	g.markDDLStatus(ctx, reg.ID, "created", nil)
	if g.cacheInvalidator != nil {
		g.cacheInvalidator(reg.ID, reg.MasterTable)
	}
	return res, nil
}
```

---

### 3.3. `centralized-data-service` — `transmuter_mongo.go` & `transmuter.go` (Fix `OB-06`, `OB-07` & Hợp nhất Feature 1 + Feature 2)

#### File 5: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go` (Fix `OB-07`)
```go
// BuildMongoOCCBulkModels constructs MongoDB WriteModels for bulkUpsertMaster.
//
// [FIX OB-07]: Khi conflictTarget != "_gpay_id" (ví dụ conflictTarget == "_id" hoặc "id"),
// _gpay_id được sinh mới bởi nextMasterSonyflakeGpayID() cho mỗi record.
// Trên PostgreSQL, ON CONFLICT (<pk>) DO UPDATE không bao giờ ghi đè _gpay_id của row đã tồn tại.
// Trên MongoDB Master:
//   - Nhánh Pipeline Update (OCC): dùng "$ifNull": ["$_gpay_id", v] để giữ nguyên _gpay_id hiện có
//     nếu document đã tồn tại, và chỉ gán v khi document được insert mới!
//   - Nhánh Standard Update (isFullSync || incomingTs == 0): tách _gpay_id sang "$setOnInsert"
//     để không bao giờ ghi đè _gpay_id của document đã tồn tại!
func BuildMongoOCCBulkModels(
	records []masterRecord,
	colTypes map[string]string,
	conflictTarget string,
	now time.Time,
	isFullSync bool,
) ([]mongo.WriteModel, error) {
	if conflictTarget == "" {
		conflictTarget = "_gpay_id"
	}
	models := make([]mongo.WriteModel, 0, len(records))
	nowDate := primitive.NewDateTimeFromTime(now.UTC())

	for _, rec := range records {
		doc := bson.M{
			"_gpay_id":     rec.GpayID,
			"_source_id":   rec.SourceID,
			"_source_ts":   rec.SourceTs,
			"_ingested_at": nowDate,
			"_deleted":     rec.Deleted,
		}

		for col, rawVal := range rec.Values {
			targetType := colTypes[col]
			converted, err := ConvertValueForMongoMaster(rawVal, targetType)
			if err != nil {
				return nil, fmt.Errorf("convert field %q (type=%s): %w", col, targetType, err)
			}
			doc[col] = converted
		}

		var keyVal interface{}
		if conflictTarget == "_gpay_id" {
			keyVal = rec.GpayID
		} else {
			keyVal = doc[conflictTarget]
			if keyVal == nil || keyVal == "" {
				keyVal = rec.GpayID
			}
		}

		filter := bson.M{conflictTarget: keyVal}
		incomingTs := rec.SourceTs

		if !isFullSync && incomingTs > 0 {
			occCond := bson.M{
				"$lte": bson.A{
					bson.M{"$ifNull": bson.A{"$_source_ts", int64(0)}},
					incomingTs,
				},
			}
			setStage := bson.M{}
			for k, v := range doc {
				if k == "_id" && conflictTarget == "_id" {
					continue
				}
				// [FIX OB-07]: Bảo toàn _gpay_id của document đã tồn tại khi conflictTarget != "_gpay_id"
				if k == "_gpay_id" && conflictTarget != "_gpay_id" {
					setStage[k] = bson.M{
						"$ifNull": bson.A{"$_gpay_id", v},
					}
					continue
				}
				setStage[k] = bson.M{
					"$cond": bson.A{
						occCond,
						v,
						"$" + k,
					},
				}
			}
			pipeline := mongo.Pipeline{
				{{Key: "$set", Value: setStage}},
			}
			model := mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(pipeline).
				SetUpsert(true)
			models = append(models, model)
		} else {
			setDoc := bson.M{}
			setOnInsertDoc := bson.M{}
			for k, v := range doc {
				if k == "_id" && conflictTarget == "_id" {
					continue
				}
				if k == "_gpay_id" && conflictTarget != "_gpay_id" {
					setOnInsertDoc[k] = v
					continue
				}
				setDoc[k] = v
			}
			updateDoc := bson.M{"$set": setDoc}
			if len(setOnInsertDoc) > 0 {
				updateDoc["$setOnInsert"] = setOnInsertDoc
			}
			model := mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(updateDoc).
				SetUpsert(true)
			models = append(models, model)
		}
	}

	return models, nil
}
```

---

### 3.4. `centralized-data-service` — `ReconDestAgent`, `ChunkStreamBucketEngine`, `ReconCore`, `recon_tier_b.go`, `source_count_runner.go` (Fix `OB-01`, `OB-02`, `OB-04`, `OB-05`, `OB-10`)

#### File 6: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go` (Fix `OB-01`)
```go
func (e *ChunkStreamBucketEngine) GetMasterAgent(ctx context.Context, connectionKey string) (*ReconDestAgent, error) {
	connectionKey = strings.TrimSpace(connectionKey)
	if connectionKey == "" || connectionKey == "default" {
		if e.masterAgent != nil {
			return e.masterAgent, nil
		}
	}

	e.masterAgentsMu.RLock()
	if agent, ok := e.masterAgents[connectionKey]; ok && agent != nil {
		e.masterAgentsMu.RUnlock()
		return agent, nil
	}
	e.masterAgentsMu.RUnlock()

	e.masterAgentsMu.Lock()
	defer e.masterAgentsMu.Unlock()
	if agent, ok := e.masterAgents[connectionKey]; ok && agent != nil {
		return agent, nil
	}

	if e.connMgr == nil {
		if e.masterAgent != nil {
			return e.masterAgent, nil
		}
		return nil, fmt.Errorf("connection manager not configured in chunkStreamBucketEngine")
	}

	// [FIX OB-01]: Kiểm tra MongoDB Master connection trước khi gọi GetMasterDB
	if e.connMgr.IsMongoMasterConnection(ctx, connectionKey) {
		handle, err := e.connMgr.GetMasterMongoClient(ctx, connectionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get master mongo client for %s: %w", connectionKey, err)
		}
		agent := NewReconDestMongoAgent(handle.Client, handle.DefaultDatabase, ReconDestAgentConfig{}, e.logger)
		if e.masterAgents == nil {
			e.masterAgents = make(map[string]*ReconDestAgent)
		}
		e.masterAgents[connectionKey] = agent
		return agent, nil
	}

	targetDB, err := e.connMgr.GetMasterDB(ctx, connectionKey)
	if err != nil {
		if e.masterAgent != nil {
			return e.masterAgent, nil
		}
		return nil, fmt.Errorf("failed to get master db for %s: %w", connectionKey, err)
	}

	agent := NewReconDestAgentWithConfig(targetDB, targetDB, ReconDestAgentConfig{}, e.logger)
	if e.masterAgents == nil {
		e.masterAgents = make(map[string]*ReconDestAgent)
	}
	e.masterAgents[connectionKey] = agent
	return agent, nil
}
```

#### File 7: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_mongo.go` & `recon_dest_query.go` (Fix `OB-01`, `OB-02`, `OB-04`, `OB-05`)
```go
// Trong recon_dest_query.go:
// 1. GetRealColumnName (được gọi bởi ColumnExists):
func (da *ReconDestAgent) GetRealColumnName(ctx context.Context, tableName, columnName string) (string, error) {
	if da.IsMongo() {
		return da.mongoGetRealColumnName(ctx, tableName, columnName)
	}
	// ... giữ nguyên nhánh PostgreSQL ...
}

// 2. BucketCounts (được gọi bởi RunDeepCheckB):
func (da *ReconDestAgent) BucketCounts(ctx context.Context, tableName, pkColumn, timestampField string, tLo, tHi time.Time) (map[int64]BucketStat, error) {
	if da.IsMongo() {
		return da.mongoBucketCounts(ctx, tableName, pkColumn, timestampField, tLo, tHi)
	}
	// ... giữ nguyên nhánh PostgreSQL ...
}

// Trong recon_dest_mongo.go:
// [FIX OB-01]: mongoGetRealColumnName kiểm tra field trên MongoDB Master
func (da *ReconDestAgent) mongoGetRealColumnName(ctx context.Context, tableName, columnName string) (string, error) {
	col := strings.TrimSpace(columnName)
	if col == "" {
		return "", nil
	}
	switch col {
	case "_id", "_gpay_id", "_source_id", "_source_ts", "_ingested_at", "_deleted":
		return col, nil
	}
	coll := da.mongoCollection(tableName)
	if coll == nil {
		return col, nil
	}
	qCtx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()
	var doc bson.M
	err := coll.FindOne(qCtx, bson.M{}).Decode(&doc)
	if err != nil {
		// Nếu collection đang rỗng (mongo.ErrNoDocuments), cho phép dùng tên cột từ transform_spec/entry
		if err == mongo.ErrNoDocuments {
			return col, nil
		}
		return col, nil
	}
	for k := range doc {
		if strings.EqualFold(k, col) {
			return k, nil
		}
	}
	// Đối với MongoDB schemaless, nếu field là PK được cấu hình trong transform_spec hoặc _source_ts
	return col, nil
}

// [FIX OB-04]: mongoCountInWindow chỉ đếm document active (_deleted != true)
func (da *ReconDestAgent) mongoCountInWindow(ctx context.Context, tableName, timestampField string, tLo, tHi time.Time) (int64, error) {
	coll := da.mongoCollection(tableName)
	if coll == nil {
		return 0, fmt.Errorf("mongo client not initialized on ReconDestAgent")
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()
	filter := buildMongoWindowFilter(timestampField, tLo, tHi, bson.M{"_deleted": bson.M{"$ne": true}})
	return coll.CountDocuments(ctx, filter)
}

// [FIX OB-05]: buildMongoWindowFilter hỗ trợ cả BSON Date và int64 epoch-ms cho trường timestamp nghiệp vụ
func buildMongoWindowFilter(timestampField string, tLo, tHi time.Time, deletedFilter bson.M) bson.M {
	tsCol := strings.TrimSpace(timestampField)
	if tsCol == "" {
		tsCol = "_source_ts"
	}
	loMs := tLo.UnixMilli()
	hiMs := tHi.UnixMilli()
	var filter bson.M
	if tsCol == "_source_ts" {
		filter = bson.M{
			"_source_ts": bson.M{
				"$gte": loMs,
				"$lt":  hiMs,
			},
		}
	} else {
		filter = bson.M{
			"$or": bson.A{
				bson.M{
					tsCol: bson.M{
						"$gte": primitive.NewDateTimeFromTime(tLo.UTC()),
						"$lt":  primitive.NewDateTimeFromTime(tHi.UTC()),
					},
				},
				bson.M{
					tsCol: bson.M{
						"$gte": loMs,
						"$lt":  hiMs,
					},
				},
			},
		}
	}
	for k, v := range deletedFilter {
		filter[k] = v
	}
	return filter
}
```

---

### 3.5. `centralized-data-service` — `recon_execute_heal_segb.go` (Fix `OB-03`)

#### File 8: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb.go`
```go
// 1. Tại executeHealSegB (dòng 100):
	if len(orphanIDs) > 0 {
		if h.isMongoMasterRef(ctx, ref) {
			affected, err := h.softDeleteMongoMasterOrphans(ctx, ref, orphanIDs, idType)
			if err != nil {
				return 0, fmt.Errorf("heal-b prune mongo master orphans (%s): %w", masterRel, err)
			}
			totalHealed += int(affected)
		} else {
			// ... giữ nguyên nhánh PostgreSQL ...
		}
	}

// 2. Tại resolveSourceIDsForSegmentB (dòng 528-546, khi row đã bị xóa cứng khỏi Shadow):
	if len(missingGpay) > 0 && pkCol != "" && pkCol != "_gpay_id" && validateHealIdent(pkCol) == nil {
		if h.isMongoMasterRef(ctx, ref) {
			if mongoIDs, errM := h.resolvePKByGpayIDsFromMongoMaster(ctx, ref, pkCol, missingGpay); errM == nil {
				out = append(out, mongoIDs...)
			}
		} else if masterDB := h.resolveMasterDBForRef(ctx, ref); masterDB != nil {
			masterRel := quoteHealRelation(ref.MasterSchema, ref.MasterTable)
			queryMaster := fmt.Sprintf(
				`SELECT DISTINCT %s::text FROM %s WHERE "_gpay_id"::text IN ? AND %s IS NOT NULL`,
				quoteHealIdent(pkCol),
				masterRel,
				quoteHealIdent(pkCol),
			)
			var masterIDs []string
			if err := masterDB.WithContext(ctx).Raw(queryMaster, missingGpay).Scan(&masterIDs).Error; err == nil {
				out = append(out, masterIDs...)
			}
		}
	}
```

---

### 3.6. Giải pháp & Code Demo cho `TASK-M11` (Fix lỗi `ReplicaSetNoPrimary` / `lookup gpay-mongo: no such host` ở Bước 1)

- **Phát hiện gốc rễ (`F11` — `VERIFIED`):**
  1. Container `gpay-mongo` (`mongo:6.0`, port mapping `0.0.0.0:17017->27017/tcp`) chạy ReplicaSet `rs0` với member internal hostname trong `rs.status().members` là `gpay-mongo:27017`.
  2. Bản ghi `id=406` (`connection_code = mongo_local`) trong `cdc_system.connection_registry` có `host = localhost`, `port = 17017`, `default_database = devtest` và `options_json` lưu `"dsn": "mongodb://localhost:17017/devtest?connectTimeoutMS=5000"`.
  3. Với `go.mongodb.org/mongo-driver v1.17.9`, khi kết nối tới URI đơn host (`mongodb://localhost:17017/...`) mà không có `directConnection=true`, sau lệnh handshake `hello` đầu tiên nhận về `setName: "rs0"` và `hosts: ["gpay-mongo:27017"]`, driver tự động chuyển topology sang `ReplicaSetNoPrimary`, loại bỏ `localhost:17017` và dial tới `gpay-mongo:27017` → gây lỗi `dial tcp: lookup gpay-mongo: no such host` trên máy Mac host.

#### File 9: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler_connections.go`
```go
func formatMongoHostList(rawHost string, defaultPort int) string {
	if defaultPort <= 0 {
		defaultPort = 27017
	}
	parts := strings.Split(rawHost, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		h := strings.TrimSpace(p)
		if h == "" {
			continue
		}
		if strings.Contains(h, ":") {
			out = append(out, h)
		} else {
			out = append(out, fmt.Sprintf("%s:%d", h, defaultPort))
		}
	}
	return strings.Join(out, ",")
}

func normalizeMasterMongoURI(rawURI string) string {
	trimmed := strings.TrimSpace(rawURI)
	if !strings.HasPrefix(trimmed, "mongodb://") {
		return trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return trimmed
	}
	// Với kết nối đơn host (không chứa dấu phẩy multi-host), mặc định bật directConnection=true
	// nếu chưa được chỉ định tường minh để tránh mongo-driver tự nhảy sang internal Docker hostname
	// (ví dụ gpay-mongo:27017) khi node là thành viên của single-node/port-forwarded ReplicaSet.
	if !strings.Contains(u.Host, ",") {
		q := u.Query()
		if q.Get("directConnection") == "" {
			q.Set("directConnection", "true")
			u.RawQuery = q.Encode()
		}
	}
	return u.String()
}

func buildMasterConnectionDSN(engineType, host string, port int, username, password, database, sslmode, authSource, replicaSet string) string {
	engineType = normalizeMasterEngineType(engineType)
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)
	database = strings.TrimSpace(database)
	sslmode = strings.TrimSpace(sslmode)
	authSource = strings.TrimSpace(authSource)
	replicaSet = strings.TrimSpace(replicaSet)

	if engineType == "mongodb" {
		if strings.HasPrefix(host, "mongodb://") || strings.HasPrefix(host, "mongodb+srv://") {
			return normalizeMasterMongoURI(host)
		}
		if port <= 0 {
			port = 27017
		}
		if database == "" {
			database = "admin"
		}
		hostSegment := formatMongoHostList(host, port)
		q := url.Values{}
		if authSource == "" && username != "" {
			authSource = "admin"
		}
		if authSource != "" {
			q.Set("authSource", authSource)
		}
		if replicaSet != "" {
			q.Set("replicaSet", replicaSet)
		}
		if sslmode == "require" || sslmode == "verify-full" || sslmode == "true" {
			q.Set("tls", "true")
		}
		if !strings.Contains(hostSegment, ",") {
			q.Set("directConnection", "true")
		}
		q.Set("connectTimeoutMS", "5000")
		if username != "" {
			return normalizeMasterMongoURI(fmt.Sprintf("mongodb://%s:%s@%s/%s?%s",
				url.QueryEscape(username),
				url.QueryEscape(password),
				hostSegment,
				database,
				q.Encode(),
			))
		}
		return normalizeMasterMongoURI(fmt.Sprintf("mongodb://%s/%s?%s", hostSegment, database, q.Encode()))
	}
	// ... giữ nguyên nhánh PostgreSQL ...
}
```
Và tại `TestConnection` (`master_registry_handler_connections.go:194`):
```go
	if engineType == "mongodb" || strings.HasPrefix(dsn, "mongodb://") || strings.HasPrefix(dsn, "mongodb+srv://") {
		dsn = normalizeMasterMongoURI(dsn)
		clientOpts := options.Client().
			ApplyURI(dsn).
			SetConnectTimeout(5 * time.Second).
			SetServerSelectionTimeout(5 * time.Second)
```

#### File 10: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/connection_manager.go`
```go
func normalizeMasterMongoURI(rawURI string) string {
	trimmed := strings.TrimSpace(rawURI)
	if !strings.HasPrefix(trimmed, "mongodb://") {
		return trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return trimmed
	}
	if !strings.Contains(u.Host, ",") {
		q := u.Query()
		if q.Get("directConnection") == "" {
			q.Set("directConnection", "true")
			u.RawQuery = q.Encode()
		}
	}
	return u.String()
}

// Cuối hàm resolveMasterMongoURI (dòng 248):
	return normalizeMasterMongoURI(uri), defaultDB
```

---

### 3.7. Giải pháp & Code Demo cho `TASK-M12` (Xử lý lỗi `FATAL: permission denied to start WAL sender` ở Bước 2 — Thêm PostgreSQL Connector `devcoretrans`)

- **Phát hiện gốc rễ (`F12` — `VERIFIED`):**
  1. Connector `devcoretrans` (`id=100` trong `cdc_system.sources`, `id=407` trong `cdc_system.connection_registry`) kết nối tới PostgreSQL `10.200.187.110:5432`, database `core-trans-proxy-history-service` bằng tài khoản `database.user = "core-trans-proxy-history-user"`, `plugin.name = "pgoutput"`, `slot.name = "devcoretrans_slot"`, `publication.name = "devcoretrans_pub"`.
  2. Truy vấn thực tế trên `10.200.187.110:5432` (`core-trans-proxy-history-service`):
     - `SHOW wal_level;` trả về `logical` (hợp lệ).
     - `core-trans-proxy-history-user` là owner (`datdba`) của DB `core-trans-proxy-history-service` và toàn bộ 4 bảng trong schema `public` (`cdc_trans_his_test`, `cdc_transaction_history`, `transaction_history`, `transaction_history_payload`).
     - Tuy nhiên trong `pg_roles`, tài khoản `core-trans-proxy-history-user` đang có **`rolreplication = f` (`false`) và `rolsuper = f` (`false`)** (chỉ có 2 role `postgres` và `replicator` có `rolreplication = t`).
  3. Khi Debezium (`PostgresConnectorTask.tryToCreateSlot`) mở kết nối giao thức Logical Replication (`walsender`) để tạo slot `devcoretrans_slot`, PostgreSQL từ chối ngay ở tầng xác thực `walsender` với lỗi:
     `FATAL: permission denied to start WAL sender — Detail: Only roles with the REPLICATION attribute may start a WAL sender process.`

- **Giải pháp Duy nhất (`TASK-M12`):**
  1. **Trên Database nguồn `10.200.187.110:5432`:** Dùng tài khoản superuser `postgres` chạy lệnh cấp quyền `REPLICATION` cho `core-trans-proxy-history-user`:
     ```sql
     ALTER ROLE "core-trans-proxy-history-user" WITH REPLICATION;
     ```
     Sau đó bấm **Restart Task** cho connector `devcoretrans` trên CMS UI (hoặc gọi `POST /api/v1/system/connectors/devcoretrans/tasks/0/restart`).
  2. **Trên `cdc-cms-service` (`internal/api/source/system_connectors_handler.go`):** Bổ sung hàm `validatePostgresConnectorPrereqs(ctx, req.Config)` tại `Create` và `UpdateConfig` để kiểm tra trước `rolreplication || rolsuper` và `wal_level == "logical"`, trả về thông báo lỗi tường minh kèm câu lệnh SQL `ALTER ROLE "<user>" WITH REPLICATION;` ngay khi cấu hình thiếu quyền.

#### File 11: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go`
```go
func validatePostgresConnectorPrereqs(ctx context.Context, cfg map[string]string) error {
	if !strings.Contains(cfg["connector.class"], "PostgresConnector") {
		return nil
	}
	host := strings.TrimSpace(cfg["database.hostname"])
	port := strings.TrimSpace(cfg["database.port"])
	if port == "" {
		port = "5432"
	}
	user := strings.TrimSpace(cfg["database.user"])
	pass := cfg["database.password"]
	dbname := strings.TrimSpace(cfg["database.dbname"])
	if host == "" || user == "" || dbname == "" || pass == "" || pass == "***" {
		return nil
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=5",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, url.QueryEscape(dbname))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	defer sqlDB.Close()

	var roleInfo struct {
		RolSuper       bool `gorm:"column:rolsuper"`
		RolReplication bool `gorm:"column:rolreplication"`
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT rolsuper, rolreplication FROM pg_roles WHERE rolname = current_user LIMIT 1`,
	).Scan(&roleInfo).Error; err == nil {
		if !roleInfo.RolSuper && !roleInfo.RolReplication {
			return fmt.Errorf(
				"PostgreSQL user %q on %s:%s lacks REPLICATION privilege required by Debezium WAL sender. Run as superuser (postgres): ALTER ROLE %q WITH REPLICATION;",
				user, host, port, user,
			)
		}
	}
	return nil
}
```

---

### 3.8. `TASK-M13` — Check Connection (Kiểm tra điều kiện Debezium) & Quét danh sách Table theo Schema cho PostgreSQL Connector trong Modal New Connect

- **Phát hiện (`F13` — `VERIFIED`):**
  1. Trong `cdc-cms-web/src/pages/SourceConnectors.tsx:1601-1710`, hiện chỉ có `dbKind === 'mongodb'` được trang bị nút **Check Connection** (`useCheckMongoConnection` → `POST /api/introspection/mongo/collections`) và ô chọn nhiều `<Select mode="multiple">` cho `collectionNames` kèm khoá nút `Create` cho tới khi Check PASS.
  2. Trong `SourceConnectors.tsx:1727-1754`, khi `dbKind === 'postgresql'`, cả `schemaIncludeList` và `tableIncludeList` đều đang là `<Input />` nhập tay thuần tuý:
     - Không có nút **Check Connection** để kiểm tra trước các điều kiện bắt buộc của Debezium PostgreSQL (`wal_level = 'logical'`, role có `rolreplication = true` hoặc `rolsuper = true`, tồn tại database và schema).
     - Không tự động quét danh sách bảng (`schema.table`) theo `Schema Include List` vừa nhập để người vận hành chọn/bỏ chọn nhanh như bên MongoDB.

- **Giải pháp Duy nhất (`TASK-M13`):**
  1. **Backend (`cdc-cms-service`):**
     - Tạo file đơn trách nhiệm `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler_postgres.go` cài đặt `DiscoverPostgresTablesPost(c *fiber.Ctx)` và đăng ký `dual("POST", shared, "/introspection/postgres/tables", h.System.Introspection.DiscoverPostgresTablesPost)` tại `internal/router/router.go:347`.
     - `DiscoverPostgresTablesPost` kết nối tới PostgreSQL nguồn (`connect_timeout=5`) và kiểm tra tuần tự:
       - Kết nối & xác thực (`cluster_err`, `auth_err`, `db_missing`).
       - `SHOW wal_level`: bắt buộc `wal_level == "logical"`, nếu khác trả về `status: "wal_level_err"` kèm câu lệnh `ALTER SYSTEM SET wal_level = 'logical';`.
       - `SELECT rolsuper, rolreplication FROM pg_roles WHERE rolname = current_user LIMIT 1`: bắt buộc `rolsuper || rolreplication`, nếu thiếu trả về `status: "replication_err"` kèm câu lệnh `ALTER ROLE "<user>" WITH REPLICATION;`.
       - Kiểm tra sự tồn tại của các schema trong `schemaIncludeList` (mặc định `public`) qua `information_schema.schemata`; nếu không tồn tại trả về `status: "schema_missing"` kèm `available_schemas`.
       - Quét toàn bộ bảng (`relkind IN ('r', 'p') AND NOT relispartition`) thuộc các schema đã nhập từ `pg_catalog.pg_class` + `pg_catalog.pg_namespace`, trả về `tables: ["schema.table1", ...]` (`status: "ok"` hoặc `"empty"`).
     - Đồng bộ `validatePostgresConnectorPrereqs` tại `internal/api/source/system_connectors_handler.go` kiểm tra cả `SHOW wal_level` (`logical`) lẫn `rolsuper || rolreplication`.
  2. **Frontend (`cdc-cms-web`):**
     - Mở rộng `src/services/connectorCheck.ts` (`CheckPostgresTablesResponse`, `checkPostgresTables`, `mapPostgresCheckStatusToVi`) và `src/hooks/useConnectorCheck.ts` (`useCheckPostgresConnection`).
     - Trong `src/pages/SourceConnectors.tsx`:
       - Khai báo `Form.useWatch` cho `host`, `port`, `username`, `password`, `schemaIncludeList` và `pgCheckHook = useCheckPostgresConnection()` ở **top-level** của `SourceConnectors` (tuân thủ nghiêm ngặt `L042` React Rules of Hooks).
       - Tự động đổi `port` từ `3306` sang `5432` khi user chọn `dbKind === 'postgresql'` ở chế độ tạo mới.
       - Khi `dbKind === 'postgresql'`: hiển thị `Schema Include List` (mặc định `public`), nút **Check Connection & Quét Table**, `<Alert>` hiển thị chi tiết trạng thái Debezium (`wal_level=logical`, `REPLICATION` OK, số lượng bảng tìm thấy hoặc gợi ý lệnh SQL copyable nếu thiếu quyền), và `<Select mode="multiple">` cho `Table Include List` tự động chọn toàn bộ bảng `schema.table` vừa quét được khi Check PASS.
       - Khoá nút `Create` khi `editorMode === 'create' && dbKind === 'postgresql'` cho tới khi `pgCheckHook.result?.status === 'ok'`.

#### File 12: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler_postgres.go`
```go
package system

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type DiscoverPostgresTablesRequest struct {
	Host     string `json:"host"`
	Port     any    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Schemas  string `json:"schemas"`
	SSLMode  string `json:"sslmode"`
}

func parsePostgresSchemaList(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return []string{"public"}
	}
	return out
}

func classifyPostgresConnectError(err error) string {
	if err == nil {
		return "ok"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "28p01") ||
		strings.Contains(msg, "28000") ||
		strings.Contains(msg, "password authentication failed") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "no pg_hba.conf entry"):
		return "auth_err"
	case strings.Contains(msg, "3d000") ||
		(strings.Contains(msg, "database") && strings.Contains(msg, "does not exist")):
		return "db_missing"
	default:
		return "cluster_err"
	}
}

// DiscoverPostgresTablesPost — POST /api/introspection/postgres/tables
// Kiểm tra kết nối PostgreSQL, điều kiện tiên quyết Debezium (wal_level=logical,
// rolreplication || rolsuper) và quét danh sách bảng (schema.table) theo schemas.
func (h *IntrospectionHandler) DiscoverPostgresTablesPost(c *fiber.Ctx) error {
	var body DiscoverPostgresTablesRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body: " + err.Error()})
	}
	host := strings.TrimSpace(body.Host)
	port := strings.TrimSpace(fmt.Sprintf("%v", body.Port))
	if port == "" || port == "<nil>" || port == "0" {
		port = "5432"
	}
	user := strings.TrimSpace(body.Username)
	pass := body.Password
	database := strings.TrimSpace(body.Database)
	sslmode := strings.TrimSpace(body.SSLMode)
	if sslmode == "" {
		sslmode = "disable"
	}
	if host == "" || user == "" || database == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "cluster_err",
			"error":  "missing host, username, or database",
		})
	}

	schemas := parsePostgresSchemaList(body.Schemas)
	sanitizedDSN := fmt.Sprintf("postgres://%s:***@%s:%s/%s?sslmode=%s",
		url.QueryEscape(user), host, port, url.QueryEscape(database), sslmode)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&connect_timeout=5",
		url.QueryEscape(user), url.QueryEscape(pass), host, port, url.QueryEscape(database), sslmode)

	ctx, cancel := context.WithTimeout(c.UserContext(), 8*time.Second)
	defer cancel()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return c.JSON(fiber.Map{
			"status":        classifyPostgresConnectError(err),
			"error":         err.Error(),
			"database":      database,
			"sanitized_dsn": sanitizedDSN,
		})
	}
	sqlDB, err := db.DB()
	if err != nil {
		return c.JSON(fiber.Map{
			"status":        "cluster_err",
			"error":         err.Error(),
			"database":      database,
			"sanitized_dsn": sanitizedDSN,
		})
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return c.JSON(fiber.Map{
			"status":        classifyPostgresConnectError(err),
			"error":         err.Error(),
			"database":      database,
			"sanitized_dsn": sanitizedDSN,
		})
	}

	// 1. Kiểm tra wal_level == 'logical'
	var walLevel string
	if err := db.WithContext(ctx).Raw("SHOW wal_level").Scan(&walLevel).Error; err == nil {
		walLevel = strings.TrimSpace(strings.ToLower(walLevel))
		if walLevel != "" && walLevel != "logical" {
			return c.JSON(fiber.Map{
				"status":        "wal_level_err",
				"wal_level":     walLevel,
				"database":      database,
				"sanitized_dsn": sanitizedDSN,
				"sql_hint":      "ALTER SYSTEM SET wal_level = 'logical';",
				"error": fmt.Sprintf(
					"PostgreSQL wal_level hiện tại là %q (Debezium yêu cầu wal_level = 'logical'). Chạy bằng superuser: ALTER SYSTEM SET wal_level = 'logical'; và restart PostgreSQL.",
					walLevel,
				),
			})
		}
	}

	// 2. Kiểm tra quyền REPLICATION hoặc SUPERUSER trên role hiện tại
	var roleInfo struct {
		RolSuper       bool `gorm:"column:rolsuper"`
		RolReplication bool `gorm:"column:rolreplication"`
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT rolsuper, rolreplication FROM pg_roles WHERE rolname = current_user LIMIT 1`,
	).Scan(&roleInfo).Error; err == nil {
		if !roleInfo.RolSuper && !roleInfo.RolReplication {
			sqlHint := fmt.Sprintf(`ALTER ROLE "%s" WITH REPLICATION;`, strings.ReplaceAll(user, `"`, `""`))
			return c.JSON(fiber.Map{
				"status":         "replication_err",
				"wal_level":      walLevel,
				"rolsuper":       roleInfo.RolSuper,
				"rolreplication": roleInfo.RolReplication,
				"database":       database,
				"sanitized_dsn":  sanitizedDSN,
				"sql_hint":       sqlHint,
				"error": fmt.Sprintf(
					"User %q trên %s:%s thiếu quyền REPLICATION để Debezium tạo replication slot và đọc WAL. Chạy bằng superuser (postgres): %s",
					user, host, port, sqlHint,
				),
			})
		}
	}

	// 3. Kiểm tra sự tồn tại của các schema trong danh sách
	var existingSchemas []string
	if err := db.WithContext(ctx).Raw(
		`SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ? ORDER BY schema_name`,
		schemas,
	).Scan(&existingSchemas).Error; err == nil && len(existingSchemas) == 0 {
		var availableSchemas []string
		_ = db.WithContext(ctx).Raw(
			`SELECT schema_name FROM information_schema.schemata
			 WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			   AND schema_name NOT LIKE 'pg_temp_%'
			   AND schema_name NOT LIKE 'pg_toast_temp_%'
			 ORDER BY schema_name`,
		).Scan(&availableSchemas).Error
		return c.JSON(fiber.Map{
			"status":            "schema_missing",
			"database":          database,
			"schemas":           schemas,
			"available_schemas": availableSchemas,
			"wal_level":         walLevel,
			"rolsuper":          roleInfo.RolSuper,
			"rolreplication":    roleInfo.RolReplication,
			"sanitized_dsn":     sanitizedDSN,
			"error":             fmt.Sprintf("Schema %q không tồn tại trong database %q", strings.Join(schemas, ", "), database),
		})
	}

	// 4. Quét danh sách bảng (schema.table) trong các schema đã chọn
	var tables []string
	if err := db.WithContext(ctx).Raw(
		`SELECT n.nspname || '.' || c.relname AS full_name
		 FROM pg_catalog.pg_class c
		 JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname IN ?
		   AND c.relkind IN ('r', 'p')
		   AND NOT c.relispartition
		 ORDER BY n.nspname, c.relname`,
		schemas,
	).Scan(&tables).Error; err != nil {
		return c.JSON(fiber.Map{
			"status":        "cluster_err",
			"error":         "failed to list tables: " + err.Error(),
			"database":      database,
			"sanitized_dsn": sanitizedDSN,
		})
	}

	if len(tables) == 0 {
		return c.JSON(fiber.Map{
			"status":         "empty",
			"tables":         []string{},
			"schemas":        schemas,
			"database":       database,
			"wal_level":      walLevel,
			"rolsuper":       roleInfo.RolSuper,
			"rolreplication": roleInfo.RolReplication,
			"sanitized_dsn":  sanitizedDSN,
		})
	}

	return c.JSON(fiber.Map{
		"status":         "ok",
		"tables":         tables,
		"schemas":        schemas,
		"database":       database,
		"wal_level":      walLevel,
		"rolsuper":       roleInfo.RolSuper,
		"rolreplication": roleInfo.RolReplication,
		"sanitized_dsn":  sanitizedDSN,
	})
}
```

#### File 13: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/services/connectorCheck.ts` & `src/hooks/useConnectorCheck.ts`
```ts
// Trong src/services/connectorCheck.ts:
export type PostgresCheckStatus =
  | 'ok'
  | 'cluster_err'
  | 'auth_err'
  | 'db_missing'
  | 'wal_level_err'
  | 'replication_err'
  | 'schema_missing'
  | 'empty'
  | 'timeout'
  | 'unknown';

export interface CheckPostgresTablesResponse {
  status: PostgresCheckStatus;
  tables?: string[];
  schemas?: string[];
  available_schemas?: string[];
  database?: string;
  wal_level?: string;
  rolsuper?: boolean;
  rolreplication?: boolean;
  sql_hint?: string;
  error?: string;
  sanitized_dsn?: string;
}

export async function checkPostgresTables(payload: {
  host: string;
  port: number | string;
  username: string;
  password?: string;
  database: string;
  schemas?: string;
}): Promise<CheckPostgresTablesResponse> {
  const { data } = await cmsApi.post<CheckPostgresTablesResponse>(
    '/api/introspection/postgres/tables',
    payload,
  );
  return data;
}
```

---

## 4. Kế hoạch Thực thi & Kiểm định sau khi có `APPROVE` (S4 → S6)
1. **Không chạy `git stash apply`:** Giữ nguyên toàn bộ code hiện tại trên working tree của `centralized-data-service` (Feature 1: Event Date Split & Batching) và `cdc-cms-service` (`activity_log_read_repo_gorm.go`).
2. **Tạo Restore-Point an toàn:** Backup trạng thái diff hiện tại trước khi áp dụng từng chỉnh sửa theo đúng thiết kế ở Mục 3.
3. **Thực thi `TASK-M01` → `TASK-M10`:** Cập nhật `cdc-cms-service`, `cdc-cms-web`, `centralized-data-service` và bổ sung đầy đủ unit/concurrency tests (`transmuter_mongo_test.go`, `recon_dest_mongo_test.go`).
4. **Kiểm định `S5` & `S6`:** Chạy `go vet` + `go test -race` qua `evidence_run.py` lấy biên lai `EV-...`, chạy `laya_gate.py run --stage dg3`, xuất `audit_report_cdc_3repos_master_mongo.md` và chạy `verify_governance.py`.
