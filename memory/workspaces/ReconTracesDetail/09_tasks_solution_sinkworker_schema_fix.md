# Hồ sơ Giải pháp Kỹ thuật Sửa đổi Logic Shadow Schema & Bổ sung Activity Log cho SinkWorker

Hồ sơ này hướng dẫn chi tiết Muscle thực hiện sửa đổi mã nguồn Go trong `internal/sinkworker/worker.go`.

---

## 1. Sửa đổi `internal/sinkworker/worker.go`
1. **Thêm imports:**
   ```go
   "centralized-data-service/internal/model/system"
   "centralized-data-service/internal/service/governance"
   ```
2. **Cập nhật struct `SinkWorker`:**
   ```go
   type SinkWorker struct {
       db            *gorm.DB
       schemaManager *SchemaManager
       avro          *avroDecoder
       machineID     int
       fencingToken  int64
       source        string
       logger        *zap.Logger
       natsConn      *nats.Conn
       masking       *governance.MaskingService
       activity      *governance.ActivityLogger // Ghi nhận activity log
   
       piCacheMu       sync.RWMutex
       postIngestCache map[string]piCacheEntry
   
       bindingCacheMu sync.RWMutex
       bindingCache   map[string]bindingCacheEntry
   }
   ```
3. **Cập nhật hàm `New`:**
   ```go
   func New(cfg Config) *SinkWorker {
       src := cfg.Source
       if src == "" {
           src = "debezium-v125"
       }
       var act *governance.ActivityLogger
       if cfg.DB != nil {
           act = governance.NewActivityLogger(cfg.DB, cfg.Logger)
       }
       return &SinkWorker{
           db:              cfg.DB,
           schemaManager:   cfg.SchemaManager,
           avro:            newAvroDecoder(cfg.SchemaRegistryURL),
           machineID:       cfg.MachineID,
           fencingToken:    cfg.FencingToken,
           source:          src,
           logger:          cfg.Logger,
           natsConn:        cfg.NATSConn,
           masking:         cfg.Masking,
           activity:        act,
           postIngestCache: make(map[string]piCacheEntry),
           bindingCache:    make(map[string]bindingCacheEntry),
       }
   }
   ```
4. **Thêm hàm `resolveShadowTarget` làm method của `SinkWorker`:**
   ```go
   // resolveShadowTarget maps a Debezium topic name to shadow schema and table by querying the database.
   // If mapping is not found in database, it returns an error instead of guessing.
   func (w *SinkWorker) resolveShadowTarget(ctx context.Context, topic string) (string, string, error) {
       parts := strings.Split(topic, ".")
       if len(parts) < 4 {
           return "", "", fmt.Errorf("invalid topic name structure %q", topic)
       }
       sourceDB := parts[len(parts)-2]
       sourceTable := parts[len(parts)-1]
   
       var bind struct {
           ShadowSchema string `gorm:"column:shadow_schema"`
           ShadowTable  string `gorm:"column:shadow_table"`
       }
   
       // Tra cứu mapping trong database
       err := w.db.WithContext(ctx).Raw(`
           SELECT sb.shadow_schema, sb.shadow_table
             FROM cdc_system.shadow_binding sb
             JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id
            WHERE (sor.source_database = ? OR sor.source_database = ?)
              AND (sor.source_object_name = ? OR sor.source_object_name = ?)
            LIMIT 1`,
           sourceDB, strings.ReplaceAll(sourceDB, "_", "-"),
           sourceTable, strings.ReplaceAll(sourceTable, "_", "-"),
       ).Scan(&bind).Error
   
       if err != nil {
           return "", "", fmt.Errorf("query shadow binding: %w", err)
       }
   
       if bind.ShadowSchema == "" || bind.ShadowTable == "" {
           return "", "", fmt.Errorf("shadow binding not found in DB for topic %q (source_db=%s, source_table=%s)", 
               topic, sourceDB, sourceTable)
       }
   
       return bind.ShadowSchema, bind.ShadowTable, nil
   }
   ```
5. **Cập nhật `HandleMessage`:**
   Thay thế đoạn tự derive cũ:
   ```go
   shadowSchema, table := extractShadowTarget(msg.Topic)
   if table == "" || shadowSchema == "" {
       handleErr = fmt.Errorf("cannot derive table from topic %q", msg.Topic)
       return handleErr
   }
   ```
   Bằng:
   ```go
   shadowSchema, table, err := w.resolveShadowTarget(ctx, msg.Topic)
   if err != nil {
       handleErr = err
       return handleErr
   }
   
   targetFQN := shadowSchema + "." + table
   var logEntry *system.ActivityLog
   if w.activity != nil {
       logEntry = w.activity.Start("sink-upsert", targetFQN, "kafka-consumer")
   }
   
   defer func() {
       if w.activity != nil && logEntry != nil {
           if handleErr != nil {
               w.activity.Fail(logEntry, handleErr.Error())
           } else {
               details := map[string]any{
                   "topic":     msg.Topic,
                   "partition": msg.Partition,
                   "offset":    msg.Offset,
                   "source_id": sourceID,
                   "snap":      snap,
               }
               w.activity.Complete(logEntry, 1, details)
           }
       }
   }()
   ```

---

## 2. Kiểm thử
- Đảm bảo `go test -v ./internal/sinkworker/...` PASS 100%.
- Biên dịch thành công: `go build ./cmd/...`.
