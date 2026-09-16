# Hồ Sơ Giải Pháp Kỹ Thuật: Hỗ Trợ Master Multi-Connection Toàn Diện Cho Toàn Bộ Subsystem Recon & Xóa Bỏ Triệt Để Hardcode default_master

**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Ngày:** 2026-09-15  
**Tác giả:** Brain (Architect)  
**Mục tiêu:** Xóa bỏ hoàn toàn sự phụ thuộc vào `default_master` (`RoleDestination`) và các câu query `LIMIT 1` mò mẫm trong toàn bộ các thành phần Recon:
1. Recon Smoke (`recon_smoke.go`)
2. Recon Segment B Tier B (`recon_tier_b.go`)
3. Recon Chunk Stream Bucket Engine (`recon_stream_bucket_engine.go`)
4. Recon Setup Bootstrap (`server_setup.go`)
5. Phân giải an toàn Fallback Transaction Aborted trong `CountRows` (`recon_dest_query.go`)

---

## 1. Giải Pháp 1: Triệt Tiêu Lỗi Fallback Transaction Aborted (`CountRows`)

### Tệp: `centralized-data-service/internal/service/recon/recon_dest_query.go`
- **Nguyên lý:** Transaction PostgreSQL khi gặp lỗi sẽ rơi vào trạng thái Aborted (`25P02`). Muốn chạy fallback sang `SELECT COUNT(*)`, bắt buộc phải gọi `tx.Rollback()` để đóng transaction hỏng, rồi mở transaction read-only mới.
- **Tối ưu:** Trong `scanExact` (`recon_smoke.go`), khi `kind == "master"`, truyền `pkCol = ""` để chạy thẳng `SELECT COUNT(*)`, không thử cột nội bộ `_gpay_id` của Shadow.

```go
func (da *ReconDestAgent) CountRows(ctx context.Context, tableName, pkColumn string) (int64, error) {
	ctx, span := observability.ChildSpan(ctx, "pg.count_rows",
		attribute.String("db.table", tableName),
		attribute.String("db.pk_column", pkColumn),
	)
	defer span.End()

	if err := validateIdent(tableName); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, da.cfg.QueryTimeout)
	defer cancel()

	result, err := da.breaker.Execute(func() (interface{}, error) {
		var count int64
		// 1. Thử COUNT(pkColumn) nếu có pkColumn hợp lệ
		if pkColumn != "" && validateIdent(pkColumn) == nil {
			tx := da.readOnlyDB(ctx)
			sql := fmt.Sprintf(`SELECT COUNT(%s) FROM %s`, quoteIdent(pkColumn), quoteRelation(tableName))
			err := tx.Raw(sql).Scan(&count).Error
			tx.Rollback() // Đảm bảo đóng tx ngay lập tức
			if err == nil {
				return count, nil
			}
		}

		// 2. Chạy COUNT(*) trên một transaction read-only mới hoàn toàn
		tx := da.readOnlyDB(ctx)
		defer tx.Rollback()
		sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName))
		if err := tx.Raw(sql).Scan(&count).Error; err != nil {
			return nil, err
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}
```

---

## 2. Giải Pháp 2: Mở Rộng Model & Query Master Connection Code

### Tệp: `centralized-data-service/internal/service/recon/recon_engine_segment_b.go`

```go
type MasterBindingRef struct {
	ID                  int64  `gorm:"column:id"`
	MasterSchema        string `gorm:"column:master_schema"`
	MasterTable         string `gorm:"column:master_table"`
	ShadowSchema        string `gorm:"column:shadow_schema"`
	ShadowTable         string `gorm:"column:shadow_table"`
	MasterConnectionKey string `gorm:"column:master_connection_key"`
	RunID               string `gorm:"-"`
}

func (rc *ReconCore) ListActiveMasterBindings(ctx context.Context) []MasterBindingRef {
	var out []MasterBindingRef
	err := rc.db.WithContext(ctx).Raw(`
		SELECT mb.id, mb.master_schema, mb.master_table, sb.shadow_schema, sb.shadow_table,
		       COALESCE(cr_ms.connection_code, 'default') AS master_connection_key
		  FROM cdc_system.master_binding mb
		  JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		  JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id AND sor.is_active = true
		  JOIN cdc_system.connection_registry cr_src ON cr_src.id = sor.source_connection_id AND cr_src.status = 'active'
		  JOIN cdc_system.connection_registry cr_sh ON cr_sh.id = sb.shadow_connection_id AND cr_sh.status = 'active'
		  LEFT JOIN cdc_system.connection_registry cr_ms ON cr_ms.id = mb.master_connection_id AND cr_ms.status = 'active'
		 WHERE mb.is_active = true AND mb.schema_status = 'approved' AND sb.is_active = true
		 ORDER BY mb.master_schema, mb.master_table`).Scan(&out).Error
	if err != nil {
		rc.logger.Error("recon segment B: list master bindings failed", zap.Error(err))
		return nil
	}
	return out
}
```

---

## 3. Giải Pháp 3: Quản Lý Pool Động `ReconDestAgent` Cho ReconCore & Xóa Hardcode `default_master`

### Tệp: `centralized-data-service/internal/service/recon/recon_engine.go`

```go
type ReconCore struct {
	sourceAgent   *ReconSourceAgent
	destAgent     *ReconDestAgent
	masterAgent   *ReconDestAgent // Giữ lại làm fallback default nếu connection_code là 'default'
	shadowPlane   *gorm.DB
	masterPlane   *gorm.DB
	db            *gorm.DB
	mongoClient   *mongo.Client
	schemaAdapter *shadow.SchemaAdapter
	registryRepo  *reposource.TableRegistryRepo
	smokeRepo     *reporecon.ReconSmokeRepo
	metadata      metadata.MetadataRegistry
	redis         *rediscache.RedisCache
	natsPub       NatsPublisher
	cfg           ReconCoreConfig
	logger        *zap.Logger
	drillDownSem  chan struct{}
	smokeCountCache sync.Map

	// Pool quản lý kết nối động tới các Master DB đích
	connMgr        *servicesource.ConnectionManager
	masterAgentsMu sync.RWMutex
	masterAgents   map[string]*ReconDestAgent
}

func (rc *ReconCore) SetConnectionManager(mgr *servicesource.ConnectionManager) {
	rc.connMgr = mgr
}

func (rc *ReconCore) GetMasterAgent(ctx context.Context, connectionKey string) (*ReconDestAgent, error) {
	connectionKey = strings.TrimSpace(connectionKey)
	if connectionKey == "" || connectionKey == "default" {
		if rc.masterAgent != nil {
			return rc.masterAgent, nil
		}
	}

	rc.masterAgentsMu.RLock()
	if agent, ok := rc.masterAgents[connectionKey]; ok {
		rc.masterAgentsMu.RUnlock()
		return agent, nil
	}
	rc.masterAgentsMu.RUnlock()

	rc.masterAgentsMu.Lock()
	defer rc.masterAgentsMu.Unlock()
	if agent, ok := rc.masterAgents[connectionKey]; ok {
		return agent, nil
	}

	if rc.connMgr == nil {
		if rc.masterAgent != nil {
			return rc.masterAgent, nil
		}
		return nil, fmt.Errorf("connection manager not configured in reconCore")
	}

	targetDB, err := rc.connMgr.GetMasterDB(ctx, connectionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get master db for %s: %w", connectionKey, err)
	}

	agent := NewReconDestAgentWithConfig(targetDB, targetDB, ReconDestAgentConfig{}, rc.logger)
	if rc.masterAgents == nil {
		rc.masterAgents = make(map[string]*ReconDestAgent)
	}
	rc.masterAgents[connectionKey] = agent
	return agent, nil
}
```

---

## 4. Giải Pháp 4: Xóa Bỏ Hardcode `default_master` Trong Toàn Bộ Recon Smoke (`recon_smoke.go`)

### Tệp: `centralized-data-service/internal/service/recon/recon_smoke.go`
- `ScanTarget` thêm `TargetKey string`.
- `smokeCountCache` cache theo `targetKey` (`kind:connKey:rel`).
- Trong `CheckAllUnified`: Lấy `msAgent, _ := rc.GetMasterAgent(ctx, r.MasterConnectionKey)`.
- Trong `RunTotalOnlyB`: Sử dụng `msAgent` tương ứng với `ref.MasterConnectionKey`, không dùng `rc.masterAgent`.
- Trong `reconDrillDownCheckB`: Dùng `msAgent` để query `BucketCounts`.

```go
	// Đẩy shadow + master relations từ Segment B
	for _, r := range validRefs {
		msAgent, err := rc.GetMasterAgent(ctx, r.MasterConnectionKey)
		if err != nil || msAgent == nil {
			msAgent = rc.masterAgent
		}
		shKey := "shadow:default:" + r.ShadowRel()
		msKey := fmt.Sprintf("master:%s:%s", r.MasterConnectionKey, r.MasterRel())

		addTarget(shKey, r.ShadowRel(), rc.destAgent, "shadow", r.ShadowTable)
		addTarget(msKey, r.MasterRel(), msAgent, "master", r.ShadowTable)
	}
```

---

## 5. Giải Pháp 5: Xóa Bỏ Hardcode `default_master` Trong Recon Tier B (`recon_tier_b.go`)

### Tệp: `centralized-data-service/internal/service/recon/recon_tier_b.go`
- Trong `RunTierB` và `RunFastLookbackSegmentB`:
  Thay thế `rc.masterAgent` bằng dynamic agent:
  ```go
  msAgent, err := rc.GetMasterAgent(ctx, ref.MasterConnectionKey)
  if err != nil || msAgent == nil {
      msAgent = rc.masterAgent
  }
  if msAgent == nil {
      observability.Ctx(ctx, rc.logger).Error("recon segment B: msAgent not wired — skip",
          zap.String("table", ref.runName()))
      return nil
  }
  ```
- Toàn bộ các lời gọi `rc.masterAgent.HashWindow`, `BucketCounts`, `ListIDTsInWindow`, `MaxWindowTs` đều chuyển sang dùng `msAgent`.

---

## 6. Giải Pháp 6: Xóa Bỏ `LIMIT 1` Mò Mẫm & Hardcode `default_master` Trong Chunk Stream Bucket Engine (`recon_stream_bucket_engine.go`)

### Tệp: `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go`
1. Bổ sung `ConnectionManager` và dynamic master agents pool cho `ChunkStreamBucketEngine`:
   ```go
   type ChunkStreamBucketEngine struct {
       sourceAgent *ReconSourceAgent
       destAgent   *ReconDestAgent
       masterAgent *ReconDestAgent
       db          *gorm.DB
       jobRepo     ReconJobRepository
       logger      *zap.Logger

       connMgr        *servicesource.ConnectionManager
       masterAgentsMu sync.RWMutex
       masterAgents   map[string]*ReconDestAgent
   }

   func (e *ChunkStreamBucketEngine) WithConnectionManager(mgr *servicesource.ConnectionManager) *ChunkStreamBucketEngine {
       e.connMgr = mgr
       return e
   }

   func (e *ChunkStreamBucketEngine) GetMasterAgent(ctx context.Context, connectionKey string) (*ReconDestAgent, error) {
       // Tương tự ReconCore
   }
   ```
2. **Xóa bỏ `ORDER BY ... LIMIT 1` mò mẫm trong `lookupMasterRef`**:
   - `lookupMasterRefExact(ctx, masterSchema, masterTable)`: SELECT thêm `COALESCE(cr_ms.connection_code, 'default') AS master_connection_key` với `LEFT JOIN cdc_system.connection_registry cr_ms`.
   - Trong `executeSegmentB`: Phân giải `msAgent` theo `ref.MasterConnectionKey`.
   - Trong `checkDayChunkB`: Dùng `msAgent` thay vì `e.masterAgent` cho `HashWindow` và `ListIDTsInWindow`.

---

## 7. Giải Pháp 7: Cập Nhật Setup Bootstrap (`server_setup.go`)

### Tệp: `centralized-data-service/internal/server/server_setup.go`
- Inject `connectionManager` cho cả `reconCore` và `chunkEngine`:
```go
reconCore.SetConnectionManager(connectionManager)
chunkEngine.WithConnectionManager(connectionManager)
```
- Loại bỏ hoàn toàn sự phụ thuộc cứng vào `registry.GetDB(database.RoleDestination)` trong quá trình kiểm tra Segment B.
