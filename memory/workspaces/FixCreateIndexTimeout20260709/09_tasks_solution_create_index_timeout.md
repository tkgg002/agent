# Giải pháp kỹ thuật chi tiết - Tối ưu hóa bất đồng bộ Create/Drop Index & Khắc phục lock-storm trong transmuter

## 1. Các file cần thay đổi

### File 1: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go`

Thay thế hàm `HandleCreateIndex` và `HandleDropIndex` để chạy bất đồng bộ bằng goroutine, tránh block NATS request và gây timeout trên UI:

```go
func (h *IndexHandler) HandleCreateIndex(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleCreateIndex")

	var payload struct {
		Schema      string   `json:"schema"`
		Table       string   `json:"table"`
		Columns     []string `json:"columns"`
		Plane       string   `json:"plane"`
		IsUnique    bool     `json:"is_unique"`
		IsPartial   bool     `json:"is_partial"`
		WhereClause string   `json:"where_clause"`
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: "invalid payload"})
		return
	}

	var targetDB *gorm.DB
	var err error
	if strings.ToLower(payload.Plane) == "master" {
		targetDB, err = h.connMgr.GetMasterDB(ctx, "default")
	} else {
		targetDB, err = h.connMgr.GetShadowDB(ctx, "default")
	}

	if err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}

	// Phản hồi ngay lập tức cho NATS client để tránh timeout
	h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", TargetTable: payload.Table, Status: "success"})

	// Chạy tạo index bất đồng bộ dưới nền
	go func() {
		defer span.End()
		bgCtx := observability.ExtractNATSHeader(context.Background(), msg.Header)
		bgCtx, childSpan := observability.ChildSpan(bgCtx, "nats.HandleCreateIndex.async")
		defer childSpan.End()

		err = h.indexManager.CreateIndexConcurrently(bgCtx, targetDB, payload.Schema, payload.Table, payload.Columns, payload.IsUnique, payload.IsPartial, payload.WhereClause)
		if err != nil {
			h.Logger.Error("failed to create index concurrently under background",
				zap.String("schema", payload.Schema),
				zap.String("table", payload.Table),
				zap.Strings("columns", payload.Columns),
				zap.Error(err))
		} else {
			h.Logger.Info("successfully created index concurrently under background",
				zap.String("schema", payload.Schema),
				zap.String("table", payload.Table),
				zap.Strings("columns", payload.Columns))
		}
	}()
}

func (h *IndexHandler) HandleDropIndex(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleDropIndex")

	var payload struct {
		Schema    string `json:"schema"`
		IndexName string `json:"index_name"`
		Plane     string `json:"plane"`
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: "invalid payload"})
		return
	}

	var targetDB *gorm.DB
	var err error
	if strings.ToLower(payload.Plane) == "master" {
		targetDB, err = h.connMgr.GetMasterDB(ctx, "default")
	} else {
		targetDB, err = h.connMgr.GetShadowDB(ctx, "default")
	}

	if err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}

	// Phản hồi ngay lập tức cho NATS client để tránh timeout
	h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "success"})

	// Chạy drop index bất đồng bộ dưới nền
	go func() {
		defer span.End()
		bgCtx := observability.ExtractNATSHeader(context.Background(), msg.Header)
		bgCtx, childSpan := observability.ChildSpan(bgCtx, "nats.HandleDropIndex.async")
		defer childSpan.End()

		err = h.indexManager.DropIndexConcurrently(bgCtx, targetDB, payload.Schema, payload.IndexName)
		if err != nil {
			h.Logger.Error("failed to drop index concurrently under background",
				zap.String("schema", payload.Schema),
				zap.String("index", payload.IndexName),
				zap.Error(err))
		} else {
			h.Logger.Info("successfully dropped index concurrently under background",
				zap.String("schema", payload.Schema),
				zap.String("index", payload.IndexName))
		}
	}()
}
```

### File 2: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go`

Tối ưu hóa `ensureShadowSourceIDIndex` bằng cách cache lại trạng thái kiểm tra index của shadow table để tránh tạo ra vòng lặp vô hạn `DROP/CREATE INDEX CONCURRENTLY` (gây lock storm và làm chậm transmute):

1. Thêm trường `ensuredShadowIndexes` vào `TransmuterModule`:

```go
type TransmuterModule struct {
	systemDB             *gorm.DB
	connMgr              *source.ConnectionManager
	runtimeRepo          *repomaster.SyncRuntimeStateRepo
	ddlEnsurer           MasterDestinationEnsurer
	typeRes              *shadow.TypeResolver
	logger               *zap.Logger
	batchSize            int
	mu                   sync.RWMutex
	cache                map[string]cachedRules
	cacheTTL             time.Duration
	shadowCache          map[string]shadowState
	ensuredMasters       map[string]bool
	ensuredShadowIndexes map[string]bool // Thêm cache cho index của shadow table
}
```

2. Khởi tạo trường này trong `NewTransmuterModule`:

```go
func NewTransmuterModule(
	systemDB *gorm.DB,
	connMgr *source.ConnectionManager,
	runtimeRepo *repomaster.SyncRuntimeStateRepo,
	ddlEnsurer MasterDestinationEnsurer,
	typeRes *shadow.TypeResolver,
	logger *zap.Logger,
) *TransmuterModule {
	return &TransmuterModule{
		systemDB:             systemDB,
		connMgr:              connMgr,
		runtimeRepo:          runtimeRepo,
		ddlEnsurer:           ddlEnsurer,
		typeRes:              typeRes,
		logger:               logger,
		batchSize:            2000,
		cache:                make(map[string]cachedRules),
		shadowCache:          make(map[string]shadowState),
		ensuredMasters:       make(map[string]bool),
		ensuredShadowIndexes: make(map[string]bool), // Khởi tạo cache
		cacheTTL:             60 * time.Second,
	}
}
```

3. Sửa đổi hàm `ensureShadowSourceIDIndex`:

```go
func (t *TransmuterModule) ensureShadowSourceIDIndex(ctx context.Context, shadowDB *gorm.DB, row *masterBindingRuntime) {
	if shadowDB.Dialector.Name() != "postgres" {
		return
	}

	indexName := fmt.Sprintf("idx_%s_source_id", row.ShadowTable)

	// RLock để check cache xem đã ensure chưa
	t.mu.RLock()
	var isEnsured bool
	if t.ensuredShadowIndexes != nil {
		isEnsured = t.ensuredShadowIndexes[indexName]
	}
	t.mu.RUnlock()
	if isEnsured {
		return
	}

	var validCount int64
	errValid := shadowDB.WithContext(ctx).Raw(`
		SELECT COUNT(*) 
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = ? AND t.relname = ? AND c.relname = ? AND i.indisvalid = true`,
		row.ShadowSchema, row.ShadowTable, indexName).Scan(&validCount).Error

	if errValid == nil && validCount > 0 {
		// Index hợp lệ đã tồn tại, cập nhật cache và bỏ qua
		t.mu.Lock()
		if t.ensuredShadowIndexes == nil {
			t.ensuredShadowIndexes = make(map[string]bool)
		}
		t.ensuredShadowIndexes[indexName] = true
		t.mu.Unlock()
		return
	}

	if errValid == nil {
		var existCount int64
		_ = shadowDB.WithContext(ctx).Raw(`
			SELECT COUNT(*) 
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			JOIN pg_class t ON t.oid = i.indrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			WHERE n.nspname = ? AND t.relname = ? AND c.relname = ?`,
			row.ShadowSchema, row.ShadowTable, indexName).Scan(&existCount).Error

		// Đánh dấu true vào cache ngay lập tức để tránh các thread song song spawn trùng goroutine drop/create
		t.mu.Lock()
		if t.ensuredShadowIndexes == nil {
			t.ensuredShadowIndexes = make(map[string]bool)
		}
		t.ensuredShadowIndexes[indexName] = true
		t.mu.Unlock()

		// Tạo index CONCURRENTLY bất đồng bộ dưới nền để không block transmuter
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			if existCount > 0 {
				t.logger.Warn("transmuter: invalid index found, dropping it first",
					zap.String("schema", row.ShadowSchema),
					zap.String("table", row.ShadowTable),
					zap.String("index", indexName))
				dropSql := fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS %s.%s`,
					quoteTransmuteIdent(row.ShadowSchema),
					quoteTransmuteIdent(indexName))
				if errDrop := shadowDB.WithContext(bgCtx).Exec(dropSql).Error; errDrop != nil {
					t.logger.Error("transmuter: failed to drop invalid index",
						zap.String("index", indexName),
						zap.Error(errDrop))
					return // Tránh chạy CREATE INDEX khi chưa drop thành công
				}
			}

			sqlText := fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (_source_id)`,
				quoteTransmuteIdent(indexName),
				quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable))
			t.logger.Info("transmuter: creating missing non-partial index on _source_id concurrently",
				zap.String("schema", row.ShadowSchema),
				zap.String("table", row.ShadowTable),
				zap.String("index", indexName))
			if errCreate := shadowDB.WithContext(bgCtx).Exec(sqlText).Error; errCreate != nil {
				t.logger.Error("transmuter: failed to create concurrent index on _source_id",
					zap.String("index", indexName),
					zap.Error(errCreate))
			} else {
				t.logger.Info("transmuter: successfully created concurrent index on _source_id",
					zap.String("schema", row.ShadowSchema),
					zap.String("table", row.ShadowTable),
					zap.String("index", indexName))
			}
		}()
	}
}
```
