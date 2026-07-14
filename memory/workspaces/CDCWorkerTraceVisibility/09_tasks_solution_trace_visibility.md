# Hồ Sơ Giải Pháp Kỹ Thuật - Tối ưu Visibility Traces & Đặt tên Span Động (Toàn diện)

Tài liệu hướng dẫn Muscle sửa đổi mã nguồn trong các dự án `centralized-data-service` và `cdc-cms-service`.

## 1. Bổ sung helper `ChildSpanWithLinks`
File: [trace_helpers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/observability/trace_helpers.go)

Thêm hàm sau:
```go
// ChildSpanWithLinks starts a child span from ctx, linked to multiple parent spans.
func ChildSpanWithLinks(ctx context.Context, name string, links []oteltrace.Link, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	opts := []oteltrace.SpanStartOption{
		oteltrace.WithAttributes(attrs...),
		oteltrace.WithLinks(links...),
	}
	return Tracer().Start(ctx, name, opts...)
}
```

## 2. Cập nhật struct `UpsertRecord`
File: [cdc_event.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/shadow/cdc_event.go)

Thêm trường `TraceContext` vào struct `UpsertRecord`:
```go
type UpsertRecord struct {
	TableName        string
	SchemaName       string
	ConnectionRole   string
	ConnectionKey    string
	PhysicalTableFQN string
	PrimaryKeyField  string
	PrimaryKeyValue  string
	MappedData       map[string]interface{}
	RawData          string
	Source           string
	Hash             string
	SourceTsMs       int64
	IsDelete         bool
	TraceContext     context.Context `gorm:"-" json:"-"` // Không lưu xuống DB, không serialize
}
```

## 3. Gán Trace Context khi đẩy record vào Buffer
File: [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)

Trong hàm `HandleRaw` (hoặc nơi khởi tạo `UpsertRecord`), gán trường `TraceContext`:
```go
		record := &shadow.UpsertRecord{
			TableName:        targetTable,
			SchemaName:       shadowSchemaName(route),
			ConnectionRole:   "shadow",
			ConnectionKey:    route.ShadowConnectionKey,
			PhysicalTableFQN: shadowPhysicalTable(route),
			PrimaryKeyField:  pgPKField,
			PrimaryKeyValue:  pkValue,
			MappedData:       mappedColumns,
			RawData:          rawJSON,
			Source:           sourceStr,
			Hash:             hash,
			SourceTsMs:       event.Data.SourceTsMs,
			IsDelete:         isDelete,
			TraceContext:     ctx, // Gán context hiện tại chứa span cdc.event_handle
		}
```

## 4. Tích hợp OTel Links & Tên Span Động trong BatchBuffer
### A. Trong `batchUpsert`
File: [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)

Sửa logic khởi tạo span:
```go
	// Thu thập trace links từ các record
	var links []oteltrace.Link
	for _, r := range records {
		if r.TraceContext != nil {
			if span := oteltrace.SpanFromContext(r.TraceContext); span.SpanContext().IsValid() {
				links = append(links, oteltrace.Link{SpanContext: span.SpanContext()})
			}
		}
	}

	// Đổi tên span thành động: cdc.batchbuffer.upsert: <table_name>
	spanName := fmt.Sprintf("cdc.batchbuffer.upsert: %s", tableName)
	ctx, span := observability.ChildSpanWithLinks(ctx, spanName, links,
		attribute.Int("cdc.batch_size", len(records)),
		attribute.String("cdc.target_table", tableName),
		attribute.String("cdc.target_schema", schemaName),
	)
```

### B. Trong `publishTransmuteTrigger`
File: [batch_buffer_fanout.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go)

Sửa logic khởi tạo span:
```go
	var links []oteltrace.Link
	for _, r := range bb.records { 
		if r.TraceContext != nil {
			if s := oteltrace.SpanFromContext(r.TraceContext); s.SpanContext().IsValid() {
				links = append(links, oteltrace.Link{SpanContext: s.SpanContext()})
			}
		}
	}

	spanName := fmt.Sprintf("cdc.batchbuffer.fanout: %s", shadowTable)
	ctx, span := observability.ChildSpanWithLinks(ctx, spanName, links,
		attribute.String("cdc.shadow_table", shadowTable),
		attribute.Int("cdc.source_ids_count", len(sourceIDs)),
	)
```

## 5. Tên Span Động ở các chặng CDC Flow chính

### A. Trong `kafka_consumer.go`
File: [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)

- Chặng `kafka.consume`:
```go
			spanName := fmt.Sprintf("kafka.consume: %s", msg.Topic)
			spanCtx, span := observability.StartSpan(parentCtx, spanName,
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination", msg.Topic),
				attribute.Int("messaging.partition", msg.Partition),
				attribute.Int64("messaging.offset", msg.Offset),
			)
```
- Chặng `cdc.process_message`:
```go
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) (rows int, err error) {
	spanName := fmt.Sprintf("cdc.process_message: %s", msg.Topic)
	ctx, span := observability.ChildSpan(ctx, spanName,
		attribute.Int("cdc.value_size_bytes", len(msg.Value)),
	)
```

### B. Trong `event_handler.go`
File: [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)

- Chặng `cdc.event_handle`:
```go
	spanName := fmt.Sprintf("cdc.event_handle: %s", sourceTable)
	ctx, span := observability.ChildSpan(ctx, spanName,
		attribute.String("cdc.subject", subject),
		attribute.String("cdc.source_table", sourceTable),
	)
```

### C. Trong `schema_inspector.go`
File: [schema_inspector.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/governance/schema_inspector.go)

- Chặng `cdc.schema_inspect`:
```go
func (si *SchemaInspector) InspectEvent(ctx context.Context, tableName, sourceDB string, eventData map[string]interface{}) (drift *SchemaDrift, err error) {
	spanName := fmt.Sprintf("cdc.schema_inspect: %s", tableName)
	ctx, span := observability.ChildSpan(ctx, spanName,
		attribute.String("cdc.table", tableName),
		attribute.String("cdc.source_db", sourceDB),
	)
```

### D. Trong `transmute_handler.go`
File: [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)

- Chặng `nats.HandleTransmuteShadow`:
```go
func (h *TransmuteHandler) HandleTransmuteShadow(msg *nats.Msg) {
	var temp struct {
		ShadowTable string `json:"shadow_table"`
	}
	_ = json.Unmarshal(msg.Data, &temp)

	spanName := "nats.HandleTransmuteShadow"
	if temp.ShadowTable != "" {
		spanName = fmt.Sprintf("nats.HandleTransmuteShadow: %s", temp.ShadowTable)
	}

	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, spanName)
```
- Chặng `cdc.worker.transmute.process`:
```go
func (h *TransmuteHandler) HandleTransmute(msg *nats.Msg) {
	var req TransmuteRequest
	_ = json.Unmarshal(msg.Data, &req)

	spanName := "cdc.worker.transmute.process"
	if req.MasterTable != "" {
		spanName = fmt.Sprintf("cdc.worker.transmute.process: %s", req.MasterTable)
	}

	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, spanName)
```
- Sửa đứt gãy context trong goroutine bất đồng bộ:
```diff
-		bgCtx, cancel := context.WithTimeout(observability.ExtractNATSHeader(context.Background(), msg.Header), timeout)
+		bgCtx, cancel := context.WithTimeout(ctx, timeout)
```

### E. Trong `transmuter.go`
File: [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)

- Chặng `cdc.service.transmute`:
```go
func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string) (res TransmuteResult, err error) {
	spanName := fmt.Sprintf("cdc.service.transmute: %s", masterName)
	ctx, span := observability.ChildSpan(ctx, spanName,
		attribute.String("cdc.master_table", masterName),
		attribute.Int("cdc.incremental_ids_count", len(onlySourceIDs)),
	)
```

### F. Trong `internal/sinkworker/worker.go`
File: [worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker/worker.go)

- Chặng `kafka.consume.sink`:
```go
	spanName := fmt.Sprintf("kafka.consume.sink: %s", msg.Topic)
	ctx, span := observability.ChildSpan(parentCtx, spanName,
		attribute.String("kafka.topic", msg.Topic),
		attribute.Int("kafka.partition", msg.Partition),
		attribute.Int64("kafka.offset", msg.Offset),
		attribute.String("kafka.key", string(msg.Key)),
	)
```

---

## 6. Đặt tên Span động cho tất cả các Handler DDL, index, scan, provisioning và orchestration khác

Để tránh các trace span có tên tĩnh chung chung, cần đổi tên động cho các hàm sau:

### A. DDL Handlers (`schema_ddl_handler.go` & `master_ddl_handler.go`)
- `schema_ddl_handler.go`:
  - `nats.HandleStandardize` -> `nats.HandleStandardize: <shadow_table>`
  - `nats.HandleCreateDefaultColumns` -> `nats.HandleCreateDefaultColumns: <shadow_table>`
  - `nats.HandleProvisionShadowTable` -> `nats.HandleProvisionShadowTable: <shadow_table>`
  - `nats.HandleAlterColumn` -> `nats.HandleAlterColumn: <shadow_table>`
  - `nats.HandleDropGinIndex` -> `nats.HandleDropGinIndex: <shadow_table>`
- `master_ddl_handler.go`:
  - `nats.HandleMasterAlterColumn` -> `nats.HandleMasterAlterColumn: <master_table>`
  - `nats.HandleMasterCreate` -> `nats.HandleMasterCreate: <master_table>`
  - `nats.HandleMasterSwap` -> `nats.HandleMasterSwap: <master_table>`

### B. Provisioning & Integration Handlers
- `batch_transform_handler.go`:
  - `nats.HandleBatchTransform` -> `nats.HandleBatchTransform: <shadow_table>`
- `provisioning_shadow_bind.go`:
  - `nats.HandleShadowBind` -> `nats.HandleShadowBind: <shadow_table>`
- `discover_handler.go`:
  - `nats.HandleDiscover` -> `nats.HandleDiscover: <source_db>` 
  - `nats.HandleScanFields` -> `nats.HandleScanFields: <table_name>`
- `sync_handler.go`:
  - `nats.HandleRestartDebezium` -> `nats.HandleRestartDebezium: <connector_name>`
  - `nats.HandleSyncRegister` -> `nats.HandleSyncRegister: <connector_name>`
  - `nats.HandleSyncState` -> `nats.HandleSyncState: <connector_name>`
- `mongo_discover_handler.go`:
  - `nats.HandleDiscoverMongoDatabases` -> `nats.HandleDiscoverMongoDatabases: <connection_key>`
  - `nats.HandleDiscoverMongoCollections` -> `nats.HandleDiscoverMongoCollections: <database>`
- `scan_handler.go`:
  - `nats.HandleScanRawData` -> `nats.HandleScanRawData: <table_name>`
  - `nats.HandleScanArrayFields` -> `nats.HandleScanArrayFields: <table_name>`

### C. Index Handlers (`index_handler.go`)
- `nats.HandleIntrospectIndexes` -> `nats.HandleIntrospectIndexes: <table_name>`
- `nats.HandleCreateIndex` -> `nats.HandleCreateIndex: <table_name>`
- `nats.HandleCreateIndex.async` -> `nats.HandleCreateIndex.async: <table_name>`
- `nats.HandleDropIndex` -> `nats.HandleDropIndex: <table_name>`
- `nats.HandleDropIndex.async` -> `nats.HandleDropIndex.async: <table_name>`

### D. Reconciliation Handlers
- `recon_execute_heal_handler.go`:
  - `nats.HandleExecuteHeal` -> `nats.HandleExecuteHeal: <table_name>`
- `recon_check_handler.go`:
  - `nats.HandleReconCheck` -> `nats.HandleReconCheck: <table_name>`
  - `cdc.recon.check` -> `cdc.recon.check: <table_name>`
- `recon_check_heal_handler.go`:
  - `nats.HandleReconHeal` -> `nats.HandleReconHeal: <table_name>`
  - `cdc.recon.heal` -> `cdc.recon.heal: <table_name>`
- `recon_sysops_handler.go`:
  - `nats.HandleRetryFailed` -> `nats.HandleRetryFailed: <table_name>`
  - `nats.HandleDebeziumSignal` -> `nats.HandleDebeziumSignal: <table_name>`
  - `nats.HandleBackfillSourceTs` -> `nats.HandleBackfillSourceTs: <table_name>`
  - `cdc.recon.backfill_source_ts` -> `cdc.recon.backfill_source_ts: <table_name>`
  - `nats.HandleDetectTimestampField` -> `nats.HandleDetectTimestampField: <table_name>`
  - `cdc.recon.detect_timestamp_field` -> `cdc.recon.detect_timestamp_field: <table_name>`

### E. Scheduler & Orchestration
- `provisioning_schedule_enable.go`:
  - `nats.HandleScheduleEnable` -> `nats.HandleScheduleEnable: <schedule_id>`
- `provisioning_handler.go`:
  - `nats.HandleStepCompleted` -> `nats.HandleStepCompleted: <step>`

---

## 7. Đặt tên Span động cho Reconciliation Core Engine

Để việc phân tích log/trace đối soát (recon) chuẩn xác, cần đổi tên động cho các span lõi trong `internal/service/recon/`:

### A. Trong `recon_tier_a.go` & `recon_tier_b.go`
- `cdc.recon.run_hash_window_check_a` -> `cdc.recon.run_hash_window_check_a: <table_name>`
- `cdc.recon.pick_scan_range` -> `cdc.recon.pick_scan_range: <table_name>`
- `cdc.recon.verify_global_range` -> `cdc.recon.verify_global_range: <table_name>`
- `cdc.recon.verify_global_blocks` -> `cdc.recon.verify_global_blocks: <table_name>`
- `cdc.recon.window_loop` -> `cdc.recon.window_loop: <table_name>`
- `cdc.recon.drift_drill_down` -> `cdc.recon.drift_drill_down: <table_name>`
- `cdc.recon.cross_check_shadow` -> `cdc.recon.cross_check_shadow: <table_name>`
- `cdc.recon.run_deep_check_a` -> `cdc.recon.run_deep_check_a: <table_name>`
- `pg.query.count_rows` -> `pg.query.count_rows: <table_name>`
- `mongo.bucket_hash` -> `mongo.bucket_hash: <table_name>`
- `pg.bucket_hash` -> `pg.bucket_hash: <table_name>`
- `cdc.recon.time_bounded_diff` -> `cdc.recon.time_bounded_diff: <table_name>`
- `pg.query.shadow_ids` -> `pg.query.shadow_ids: <table_name>`
- `mongo.stream.source_ids` -> `mongo.stream.source_ids: <table_name>`
- **Tier B tương ứng**:
  - `cdc.recon.run_hash_window_check_b` -> `cdc.recon.run_hash_window_check_b: <table_name>`
  - `cdc.recon.verify_global_range_b` -> `cdc.recon.verify_global_range_b: <table_name>`
  - `cdc.recon.verify_global_blocks_b` -> `cdc.recon.verify_global_blocks_b: <table_name>`
  - `cdc.recon.window_loop_b` -> `cdc.recon.window_loop_b: <table_name>`
  - `cdc.recon.drift_drill_down_b` -> `cdc.recon.drift_drill_down_b: <table_name>`
  - `cdc.recon.run_deep_check_b` -> `cdc.recon.run_deep_check_b: <table_name>`
  - `cdc.recon.time_bounded_diff_b` -> `cdc.recon.time_bounded_diff_b: <table_name>`

### B. Trong `recon_stream.go`
- `recon.source.list_ids_in_window` -> `recon.source.list_ids_in_window: <collection_name>`
- `recon.source.list_all_ids` -> `recon.source.list_all_ids: <collection_name>`
- `recon.source.stream_all_ids` -> `recon.source.stream_all_ids: <collection_name>`
- `mongo.find_batch` -> `mongo.find_batch: <collection_name>`
- `pg.list_ids_in_window` -> `pg.list_ids_in_window: <table_name>`
- `pg.stream_all_ids` -> `pg.stream_all_ids: <table_name>`
- `pg.select_batch` -> `pg.select_batch: <table_name>`
- `recon.source.list_idts_in_window` -> `recon.source.list_idts_in_window: <collection_name>`
- `pg.list_idts_in_window` -> `pg.list_idts_in_window: <table_name>`
- `recon.source.stream_ids_in_time_range` -> `recon.source.stream_ids_in_time_range: <collection_name>`
- `pg.stream_ids_in_time_range` -> `pg.stream_ids_in_time_range: <table_name>`

### C. Trong `recon_query.go` & `recon_dest_query.go`
- `recon.source.count_documents` -> `recon.source.count_documents: <collection_name>`
- `recon.source.estimated_count` -> `recon.source.estimated_count: <collection_name>`
- `recon.source.bucket_counts` -> `recon.source.bucket_counts: <collection_name>`
- `recon.source.count_in_window` -> `recon.source.count_in_window: <collection_name>`
- `recon.source.max_window_ts` -> `recon.source.max_window_ts: <collection_name>`
- `pg.count_documents` -> `pg.count_documents: <table_name>`
- `pg.estimated_count` -> `pg.estimated_count: <table_name>`
- `pg.count_in_window` -> `pg.count_in_window: <table_name>`
- `pg.bucket_counts` -> `pg.bucket_counts: <table_name>`
- `pg.max_window_ts` -> `pg.max_window_ts: <table_name>`
- `pg.count_rows` -> `pg.count_rows: <table_name>`
- `pg.count_deleted_rows` -> `pg.count_deleted_rows: <table_name>`
- `pg.estimated_count_rows` -> `pg.estimated_count_rows: <table_name>`
- `pg.list_idts_in_window` -> `pg.list_idts_in_window: <table_name>`

### D. Trong `recon_dest_hash.go` & `recon_hash.go`
- `pg.hash_window` -> `pg.hash_window: <table_name>`
- `pg.bucket_hash` -> `pg.bucket_hash: <table_name>`
- `recon.source.hash_window` -> `recon.source.hash_window: <collection_name>`
- `recon.source.bucket_hash` -> `recon.source.bucket_hash: <collection_name>`

### E. Trong `recon_smoke.go` & `recon_engine_run.go`
- `recon.smoke.pg_get_counts` -> `recon.smoke.pg_get_counts: <table_name>`
- `recon.smoke.segment_a` -> `recon.smoke.segment_a: <table_name>`
- `recon.smoke.segment_b` -> `recon.smoke.segment_b: <table_name>`
- `recon.smoke.mongo_get_counts` -> `recon.smoke.mongo_get_counts: <collection_name>`
- `recon.smoke.cycle_unified` -> `recon.smoke.cycle_unified: <table_name>`
- `cdc.service.reap` -> `cdc.service.reap: <table_name>`

---

## 8. Đặt tên Span động cho chu kỳ nền & công việc (Scheduler & Cycles)

### A. Trong `server_jobs.go`
File: [server_jobs.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_jobs.go)
- `cdc.worker.transform_cycle` -> `cdc.worker.transform_cycle: <target_table>`
- `cdc.worker.field_scan_cycle` -> `cdc.worker.field_scan_cycle: <target_table>`

---

## 9. Tối ưu hóa traces trong `cdc-cms-service`

### A. NATS Command Bus (`nats_command_bus.go`)
File: [nats_command_bus.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats_command_bus.go)
- `command_bus.execute` -> `command_bus.execute: <command.Type()>`
- `command_bus.dispatch` -> `command_bus.dispatch: <command.Type()>`

### B. Saga Orchestrator (`saga.go`)
File: [saga.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/saga/saga.go)
- `saga.step` -> `saga.step: <step_name>` (ở dòng 67)
  ```go
  spanName := fmt.Sprintf("saga.step: %s", step.Name)
  stepCtx, stepSpan := observability.StartSpan(ctx, spanName, ...)
  ```
- `saga.compensate` -> `saga.compensate: <step_name>` (ở dòng 109)
  ```go
  spanName := fmt.Sprintf("saga.compensate: %s", s.Name)
  _, compSpan := observability.StartSpan(ctx, spanName, ...)
  ```
