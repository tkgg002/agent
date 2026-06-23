package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/model"
	"centralized-data-service/internal/naming"
	"centralized-data-service/internal/repository"
	"centralized-data-service/internal/service"
	"centralized-data-service/pkgs/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CommandHandler handles async CDC commands published from the API service via NATS.
// These commands operate on the DW (Data Warehouse) database, which only the worker can access.
type CommandHandler struct {
	db            *gorm.DB
	mappingRepo   *repository.MappingRuleRepo
	mappingV2Repo *repository.MappingRuleV2Repo
	registryRepo  *repository.RegistryRepo
	pendingRepo   *repository.PendingFieldRepo
	shadowDB      *gorm.DB // Phase A3 Hybrid — connection to cdc_shadow
	metadata      service.MetadataRegistry
	logger        *zap.Logger
	// kafkaConnectURL — base URL for Kafka Connect REST (boundary
	// refactor). Empty disables the Debezium-specific handlers.
	kafkaConnectURL string
	// natsConn — used by boundary-refactor handlers to publish async
	// result events (cdc.result.*). The NATS library does not expose the
	// connection from *nats.Subscription so we inject it explicitly.
	natsConn *nats.Conn
	// mongoSvc — for MongoDB introspection (Flow 1)
	mongoSvc *service.MongoIntrospectionService
	// mongoURL — pre-configured MongoDB connection URI from worker config.
	// Used by scanFieldsMongoSource to connect to the source cluster.
	mongoURL string
	// transformChunkSize — chunked UPDATE size for HandleBatchTransform to
	// avoid long lock + bloated WAL on large shadow tables. 0 = default 1000.
	transformChunkSize int
	// connectionOverrides — worker-side overlay keyed by connection_code.
	// Hit short-circuits scanFieldsMongoSource's host/port assembly.
	connectionOverrides map[string]string
}

// SetKafkaConnectURL injects the Kafka Connect REST base URL (used by
// boundary refactor handlers HandleRestartDebezium / HandleSyncState).
// Called from worker_server wiring to keep NewCommandHandler signature
// stable.
func (h *CommandHandler) SetKafkaConnectURL(url string) {
	h.kafkaConnectURL = url
}

// SetNATSConn injects the shared NATS connection. Required by all
// boundary-refactor handlers that publish async results. Left optional
// so unit tests can swap a mock.
func (h *CommandHandler) SetNATSConn(conn *nats.Conn) {
	h.natsConn = conn
}

func (h *CommandHandler) SetMetadataRegistry(metadata service.MetadataRegistry) {
	h.metadata = metadata
}

func (h *CommandHandler) SetMongoService(svc *service.MongoIntrospectionService) {
	h.mongoSvc = svc
}

func (h *CommandHandler) SetMongoURL(url string) {
	h.mongoURL = url
}

// SetTransformChunkSize injects chunk size for HandleBatchTransform.
// Non-positive value falls back to the internal default at call time.
func (h *CommandHandler) SetTransformChunkSize(n int) {
	h.transformChunkSize = n
}

// SetConnectionOverrides injects the worker-side URI overlay map
// (cfg.ConnectionOverrides). Pass nil/empty to keep production
// behaviour (no overrides applied).
func (h *CommandHandler) SetConnectionOverrides(overrides map[string]string) {
	h.connectionOverrides = service.NormalizeConnectionOverrides(overrides)
}

// CommandResult is the admin-facing result envelope. It must stay
// sanitized before being logged, stored in ActivityLog, or published to
// downstream control-plane consumers.
type CommandResult struct {
	Command      string `json:"command"`
	RegistryID   uint   `json:"registry_id,omitempty"`
	TargetTable  string `json:"target_table,omitempty"`
	RowsAffected int    `json:"rows_affected,omitempty"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	TraceID      string `json:"trace_id,omitempty"`
	Action       string `json:"action,omitempty"`
	Origin       string `json:"origin,omitempty"`
}

func NewCommandHandler(
	db *gorm.DB,
	mappingRepo *repository.MappingRuleRepo,
	mappingV2Repo *repository.MappingRuleV2Repo,
	registryRepo *repository.RegistryRepo,
	pendingRepo *repository.PendingFieldRepo,
	shadowDB *gorm.DB,
	logger *zap.Logger,
) *CommandHandler {
	return &CommandHandler{
		db:            db,
		mappingRepo:   mappingRepo,
		mappingV2Repo: mappingV2Repo,
		registryRepo:  registryRepo,
		pendingRepo:   pendingRepo,
		shadowDB:      shadowDB,
		logger:        logger,
	}
}

// Returns error if table doesn't exist. Safe to call multiple times (ADD COLUMN IF NOT EXISTS).
func (h *CommandHandler) ensureCDCColumns(tableName string) error {
	return h.ensureCDCColumnsInSchema("public", tableName)
}

func quoteCommandIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func quoteCommandQualifiedTable(schemaName, tableName string) string {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	return quoteCommandIdent(schemaName) + "." + quoteCommandIdent(tableName)
}

func (h *CommandHandler) ensureCDCColumnsInSchema(schemaName, tableName string) error {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	var exists bool
	h.shadowDB.Raw(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = ? AND table_schema = ?)",
		tableName,
		schemaName,
	).Scan(&exists)
	if !exists {
		return fmt.Errorf("table %s.%s does not exist", schemaName, tableName)
	}

	// Shadow Layer required system columns — TRUTH SOURCE là runtime path
	// sinkworker/schema_manager.go:225-237 (11 cols). Bootstrap phải align:
	//   _gpay_id BIGINT (system PK runtime sonyflake) — legacy ALTER ADD chỉ
	//     thêm column, KHÔNG ép PRIMARY KEY (bảng cũ đã có id PK → xung đột);
	//     bảng mới CREATE TABLE ở HandleCreateDefaultColumns set PK trực tiếp.
	//   _source_id TEXT — V2 ON CONFLICT anchor (UNIQUE).
	//   _source_ts BIGINT — OCC anchor (sinkworker/upsert.go:69-122).
	//   _deleted BOOLEAN — tombstone.
	cdcColumns := []struct{ name, def string }{
		{"_gpay_id", "BIGINT"},
		{"_source_id", "TEXT"},
		{"_raw_data", "JSONB"},
		{"_source", "VARCHAR(20) DEFAULT 'debezium'"},
		{"_source_ts", "BIGINT"},
		{"_synced_at", "TIMESTAMP DEFAULT NOW()"},
		{"_version", "BIGINT DEFAULT 1"},
		{"_hash", "VARCHAR(64)"},
		{"_deleted", "BOOLEAN DEFAULT FALSE"},
		{"_created_at", "TIMESTAMP DEFAULT NOW()"},
		{"_updated_at", "TIMESTAMP DEFAULT NOW()"},
	}
	for _, col := range cdcColumns {
		h.shadowDB.Exec(fmt.Sprintf(`ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s %s`, quoteCommandIdent(schemaName), quoteCommandIdent(tableName), col.name, col.def))
	}
	indexName := fmt.Sprintf("idx_%s_raw", tableName)
	h.shadowDB.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s USING GIN(_raw_data)`, quoteCommandIdent(indexName), quoteCommandIdent(schemaName), quoteCommandIdent(tableName)))

	// OCC sort index — sinkworker upsert lookup theo _source_ts older-wins.
	// Match sinkworker/schema_manager.go:231 contract.
	sourceTsIdx := fmt.Sprintf("idx_%s_source_ts", tableName)
	h.shadowDB.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s(_source_ts)`, quoteCommandIdent(sourceTsIdx), quoteCommandIdent(schemaName), quoteCommandIdent(tableName)))

	// Partial UNIQUE INDEX trên _source_id (V2 master ON CONFLICT key) —
	// MUST khớp sinkworker/schema_manager.go:262-268 (`WHERE NOT _deleted`).
	// Tên `ux_<table>_source_id_active` khớp đúng tên sinkworker tạo để
	// runtime EnsureShadowTable không tạo duplicate index.
	uxName := fmt.Sprintf("ux_%s_source_id_active", tableName)
	h.shadowDB.Exec(fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (_source_id) WHERE NOT _deleted`,
		quoteCommandIdent(uxName),
		quoteCommandIdent(schemaName), quoteCommandIdent(tableName),
	))
	return nil
}

// hasColumn checks if a column exists in a table
func (h *CommandHandler) hasColumn(tableName, columnName string) bool {
	return h.hasColumnInSchema(h.resolveTargetSchema(tableName), tableName, columnName)
}

func (h *CommandHandler) hasColumnInSchema(schemaName, tableName, columnName string) bool {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	var exists bool
	h.shadowDB.Raw(
		"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?)",
		schemaName,
		tableName,
		columnName,
	).Scan(&exists)
	return exists
}

// tableExists checks if a table exists
func (h *CommandHandler) tableExists(tableName string) bool {
	return h.tableExistsInSchema(h.resolveTargetSchema(tableName), tableName)
}

func (h *CommandHandler) tableExistsInSchema(schemaName, tableName string) bool {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	var exists bool
	h.shadowDB.Raw(
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = ? AND table_schema = ?)",
		tableName,
		schemaName,
	).Scan(&exists)
	return exists
}

// listShadowColumns returns the existing column names (lower-cased) for the
// given shadow table as a set. Used by HandleCreateDefaultColumns to count
// "columns actually added" vs "columns that were already there" — Postgres
// ALTER ... ADD COLUMN IF NOT EXISTS is a no-op when the column exists, so
// counting successful Exec() calls overcounts.
func (h *CommandHandler) listShadowColumns(schemaName, tableName string) map[string]struct{} {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	out := make(map[string]struct{})
	var cols []string
	h.shadowDB.Raw(
		"SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?",
		schemaName,
		tableName,
	).Scan(&cols)
	for _, c := range cols {
		out[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
	}
	return out
}

// listShadowColumnsWithType returns column_name (lower-cased) → data_type
// (lower-cased) for the shadow table. Used by HandleCreateDefaultColumns to
// detect drift between the actual PG column type and the approved
// mapping_rule_v2.data_type, then ALTER COLUMN TYPE accordingly.
func (h *CommandHandler) listShadowColumnsWithType(schemaName, tableName string) map[string]string {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	out := make(map[string]string)
	type row struct {
		ColumnName string `gorm:"column:column_name"`
		DataType   string `gorm:"column:data_type"`
	}
	var rows []row
	h.shadowDB.Raw(
		"SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = ? AND table_name = ?",
		schemaName,
		tableName,
	).Scan(&rows)
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.ColumnName))] = strings.ToLower(strings.TrimSpace(r.DataType))
	}
	return out
}

// normalizePGType collapses PG type aliases so rule.data_type (e.g. "BIGINT")
// compares cleanly against information_schema.columns.data_type (e.g.
// "bigint" or "int8"). Strips length/precision suffix.
func normalizePGType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if i := strings.IndexAny(t, "( "); i > 0 {
		// Keep "timestamp without time zone" intact — only strip on '(' or
		// on the first space when the head is a single token alias.
		head := t[:i]
		switch head {
		case "timestamp", "time", "character", "double":
			// multi-word canonical types — fall through, keep full string
		default:
			t = head
		}
	}
	switch t {
	case "int8", "bigserial":
		return "bigint"
	case "int4", "serial":
		return "integer"
	case "int2":
		return "smallint"
	case "bool":
		return "boolean"
	case "float4":
		return "real"
	case "float8":
		return "double precision"
	case "timestamptz":
		return "timestamp with time zone"
	case "timestamp":
		return "timestamp without time zone"
	case "varchar":
		return "character varying"
	}
	return t
}

// HandleStandardize subscribes to "cdc.cmd.standardize" and runs standardize_cdc_table() on the DW DB.
func (h *CommandHandler) HandleStandardize(msg *nats.Msg) {
	var payload struct {
		RegistryID   uint   `json:"registry_id"`
		TargetTable  string `json:"target_table"`
		ShadowSchema string `json:"shadow_schema"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Error("cdc.cmd.standardize: invalid payload", zap.Error(err))
		return
	}

	schemaName := strings.TrimSpace(payload.ShadowSchema)
	if schemaName == "" {
		schemaName = "public"
	}

	h.logger.Info("standardizing table", zap.String("schema", schemaName), zap.String("table", payload.TargetTable))

	if err := h.ensureCDCColumnsInSchema(schemaName, payload.TargetTable); err != nil {
		h.logger.Error("standardize failed",
			zap.String("schema", schemaName),
			zap.String("table", payload.TargetTable),
			zap.Error(err),
		)
		h.publishResult(msg, CommandResult{
			Command:     "standardize",
			RegistryID:  payload.RegistryID,
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       err.Error(),
		})
		return
	}

	h.logger.Info("standardize complete", zap.String("schema", schemaName), zap.String("table", payload.TargetTable))
	h.publishResult(msg, CommandResult{
		Command:     "standardize",
		RegistryID:  payload.RegistryID,
		TargetTable: payload.TargetTable,
		Status:      "success",
	})
}

// HandleCreateDefaultColumns creates the CDC table + adds all approved mapping rule columns.
// This is the "tạo field default" action from Luồng 1.
// Subject: "cdc.cmd.create-default-columns"
// Subject: "cdc.cmd.scan-fields" fallback
func (h *CommandHandler) scanFieldsMongoSource(ctx context.Context, registryID int64, shadowBindingID int64, sourceTable string, autoApprove bool) (int, int, error) {
	// 1. Get registry details from source_object_registry
	var registry model.SourceObjectRegistry
	if err := h.db.Where("id = ?", registryID).First(&registry).Error; err != nil {
		// Heuristic: If not found by ID, try legacy registry ID in locator
		err = h.db.Raw(`SELECT * FROM cdc_system.source_object_registry WHERE source_locator_json->>'legacy_registry_id' = ?`, strconv.FormatInt(registryID, 10)).Scan(&registry).Error
		if err != nil || registry.ID == 0 {
			return 0, 0, fmt.Errorf("failed to get source_object_registry id=%d: %v", registryID, err)
		}
	}

	// 2. Lấy Mongo DSN từ connection_registry row của source qua resolver
	// thống nhất (MetadataRegistryService.GetSourceDSN). Resolver tự cover
	// các convention: override > host-as-URI > host-as-env-ptr >
	// secret-as-URI > secret-as-env-ptr > build-from-fields > AES.
	if registry.SourceConnectionID <= 0 {
		return 0, 0, fmt.Errorf("source_object_registry id=%d has no source_connection_id; cannot resolve source", registryID)
	}
	var conn model.ConnectionRegistry
	if dbErr := h.db.WithContext(ctx).First(&conn, registry.SourceConnectionID).Error; dbErr != nil {
		return 0, 0, fmt.Errorf("failed to load connection_registry id=%d for source registry id=%d: %w",
			registry.SourceConnectionID, registryID, dbErr)
	}
	if h.metadata == nil {
		return 0, 0, fmt.Errorf("metadata registry not wired; cannot resolve DSN for connection_id=%d",
			registry.SourceConnectionID)
	}
	dsn, dsnErr := h.metadata.GetSourceDSN(ctx, conn.ConnectionCode)
	if dsnErr != nil || strings.TrimSpace(dsn) == "" {
		return 0, 0, fmt.Errorf("resolve DSN for connection_code=%s (id=%d): %w",
			conn.ConnectionCode, registry.SourceConnectionID, dsnErr)
	}

	// 3. Introspect Source
	sourceDB := ""
	if registry.SourceDatabase != nil {
		sourceDB = *registry.SourceDatabase
	}
	if sourceDB == "" {
		return 0, 0, fmt.Errorf("source_database is missing in registry id=%d", registryID)
	}

	sanitizedDSN := service.SanitizeMongoDSN(dsn)
	dispatchPath := "fallback"
	if _, hit := service.ApplyConnectionOverride(&conn, h.connectionOverrides, nil); hit {
		dispatchPath = "override"
	}
	h.logger.Info("scan-fields mongo introspect",
		zap.String("connection_code", conn.ConnectionCode),
		zap.String("dispatch_path", dispatchPath),
		zap.String("sanitized_dsn", sanitizedDSN),
		zap.String("source_db", sourceDB),
		zap.String("collection", registry.SourceObjectName),
		zap.Int64("registry_id", registryID),
	)

	fieldMap, diag, err := h.mongoSvc.IntrospectCollectionDiagnose(dsn, sourceDB, registry.SourceObjectName, 10)
	if err != nil || diag.Status == "cluster_err" {
		cause := ""
		if diag.Err != nil {
			cause = diag.Err.Error()
		} else if err != nil {
			cause = err.Error()
		}
		h.logger.Error("scan-fields cluster unreachable",
			zap.String("connection_code", conn.ConnectionCode),
			zap.String("sanitized_dsn", sanitizedDSN),
			zap.String("err", cause),
		)
		return 0, 0, fmt.Errorf("mongo cluster unreachable for connection_code=%s sanitized_dsn=%s: %s",
			conn.ConnectionCode, sanitizedDSN, cause)
	}

	switch diag.Status {
	case "db_missing":
		h.logger.Warn("scan-fields db missing",
			zap.String("connection_code", conn.ConnectionCode),
			zap.String("sanitized_dsn", sanitizedDSN),
			zap.String("source_db", sourceDB),
			zap.Strings("available_dbs", diag.AvailableDBs),
		)
		return 0, 0, fmt.Errorf("source database %q not found on connection_code=%s (sanitized_dsn=%s); available DBs on cluster: %v",
			sourceDB, conn.ConnectionCode, sanitizedDSN, diag.AvailableDBs)

	case "coll_missing":
		similar := findSimilarCollections(registry.SourceObjectName, diag.AvailableColls)
		sample := diag.AvailableColls
		if len(sample) > 50 {
			sample = sample[:50]
		}
		h.logger.Warn("scan-fields collection missing",
			zap.String("connection_code", conn.ConnectionCode),
			zap.String("sanitized_dsn", sanitizedDSN),
			zap.String("source_db", sourceDB),
			zap.String("collection", registry.SourceObjectName),
			zap.Int("available_count", len(diag.AvailableColls)),
			zap.Strings("available_collections", diag.AvailableColls),
			zap.Strings("similar_collections", similar),
		)
		if len(similar) > 0 {
			return 0, 0, fmt.Errorf("collection %q not found in database %q on connection_code=%s (sanitized_dsn=%s); did you mean: %v ? (DB has %d collections total; first 50: %v)",
				registry.SourceObjectName, sourceDB, conn.ConnectionCode, sanitizedDSN, similar, len(diag.AvailableColls), sample)
		}
		return 0, 0, fmt.Errorf("collection %q not found in database %q on connection_code=%s (sanitized_dsn=%s); DB has %d collections; first 50: %v — check worker terminal log for FULL list",
			registry.SourceObjectName, sourceDB, conn.ConnectionCode, sanitizedDSN, len(diag.AvailableColls), sample)

	case "empty":
		h.logger.Warn("scan-fields collection empty",
			zap.String("connection_code", conn.ConnectionCode),
			zap.String("sanitized_dsn", sanitizedDSN),
			zap.String("source_db", sourceDB),
			zap.String("collection", registry.SourceObjectName),
			zap.Int64("doc_count", diag.DocCount),
		)
		return 0, 0, fmt.Errorf("collection %s.%s exists on connection_code=%s but contains 0 documents (sanitized_dsn=%s); nothing to sample — load data into the source then retry",
			sourceDB, registry.SourceObjectName, conn.ConnectionCode, sanitizedDSN)

	case "no_fields":
		h.logger.Warn("scan-fields no fields",
			zap.String("connection_code", conn.ConnectionCode),
			zap.String("sanitized_dsn", sanitizedDSN),
			zap.String("source_db", sourceDB),
			zap.String("collection", registry.SourceObjectName),
			zap.Int64("doc_count", diag.DocCount),
		)
		return 0, 0, fmt.Errorf("collection %s.%s on connection_code=%s has %d documents but sampled docs contain no usable fields (only _id)",
			sourceDB, registry.SourceObjectName, conn.ConnectionCode, diag.DocCount)
	}

	// Convert fieldMap to JSON strings for reuse of existing logic
	rows := []string{}
	for _, v := range []map[string]interface{}{fieldMap} {
		b, _ := json.Marshal(v)
		rows = append(rows, string(b))
	}

	return h.processDiscoveryRows(ctx, registryID, shadowBindingID, sourceTable, rows, autoApprove)
}

// findSimilarCollections returns up to 20 collection names that share a
// token with `target` after splitting on -, _, . — case-insensitive.
// Used by scan-fields coll_missing diagnostic to nudge the user toward
// the correct collection name when only the separator/case is off
// (e.g. registry says "export-jobs" but Mongo has "exportJobs").
func findSimilarCollections(target string, all []string) []string {
	tokens := strings.FieldsFunc(strings.ToLower(target), func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	if len(tokens) == 0 {
		return nil
	}
	out := []string{}
	for _, c := range all {
		lc := strings.ToLower(c)
		for _, tok := range tokens {
			if tok != "" && strings.Contains(lc, tok) {
				out = append(out, c)
				break
			}
		}
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func (h *CommandHandler) processDiscoveryRows(ctx context.Context, registryID int64, shadowBindingID int64, sourceTable string, rows []string, autoApprove bool) (int, int, error) {
	// Per-field union of inferred types across every sampled row. Old
	// behaviour set discovered[k] from the FIRST row only — if that row
	// happened to carry an integer value for an otherwise UUID field, the
	// rule was seeded BIGINT and the shadow column locked into a numeric
	// type, breaking subsequent UUID upserts (SQLSTATE 22P02). The union +
	// resolveMongoSampledType collapse to TEXT whenever any sample is a
	// string — lossless for the heterogeneous data Mongo typically carries.
	perField := map[string]map[string]struct{}{}
	for _, row := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(row), &doc); err != nil {
			continue
		}
		for k, v := range doc {
			if k == "_raw_data" || k == "_synced_at" || k == "_source" {
				continue
			}
			t := service.InferTypeFromRawData(v)
			set, ok := perField[k]
			if !ok {
				set = map[string]struct{}{}
				perField[k] = set
			}
			set[t] = struct{}{}
		}
	}
	discovered := make(map[string]string, len(perField))
	for k, set := range perField {
		discovered[k] = resolveMongoSampledType(set)
	}

	// Dedup scope = (source_object_id, shadow_binding_id). With multi-binding
	// sources, the same source_field exists once per binding, so the
	// "already mapped" check MUST include the binding to avoid skipping
	// inserts for a freshly-created binding whose source has prior rules
	// under the other bindings.
	existingRules := []model.MappingRuleV2{}
	existingQ := h.db.Table("cdc_system.mapping_rule_v2").Where("source_object_id = ?", registryID)
	if shadowBindingID > 0 {
		existingQ = existingQ.Where("shadow_binding_id = ?", shadowBindingID)
	} else {
		existingQ = existingQ.Where("shadow_binding_id IS NULL")
	}
	existingQ.Find(&existingRules)
	mapped := make(map[string]bool)
	for _, r := range existingRules {
		mapped[r.SourceField] = true
	}

	status := "pending"
	if autoApprove {
		status = "approved"
	}

	var bindingPtr *int64
	if shadowBindingID > 0 {
		bid := shadowBindingID
		bindingPtr = &bid
	}

	added := 0
	insertErrors := 0
	for field, dataType := range discovered {
		if !mapped[field] {
			rule := model.MappingRuleV2{
				SourceObjectID:  registryID,
				ShadowBindingID: bindingPtr,
				SourceField:     field,
				TargetColumn:    field,
				DataType:        dataType,
				SourceFormat:    "raw",
				IsActive:        true,
				Status:          status,
			}
			if err := h.db.Table("cdc_system.mapping_rule_v2").Create(&rule).Error; err != nil {
				// Trước đây swallow → user thấy "scan ok 0 added" mà không
				// biết tại sao. Surface để grep theo source_object_id.
				h.logger.Warn("processDiscoveryRows: failed to insert mapping_rule_v2",
					zap.Int64("source_object_id", registryID),
					zap.Int64("shadow_binding_id", shadowBindingID),
					zap.String("source_table", sourceTable),
					zap.String("field", field),
					zap.String("data_type", dataType),
					zap.String("status", status),
					zap.Error(err),
				)
				insertErrors++
				continue
			}
			added++
		}
	}
	h.logger.Info("processDiscoveryRows summary",
		zap.Int64("source_object_id", registryID),
		zap.Int64("shadow_binding_id", shadowBindingID),
		zap.String("source_table", sourceTable),
		zap.Int("discovered_total", len(discovered)),
		zap.Int("already_mapped", len(discovered)-added-insertErrors),
		zap.Int("inserted", added),
		zap.Int("insert_errors", insertErrors),
	)

	return added, len(discovered), nil
}

func (h *CommandHandler) HandleCreateDefaultColumns(msg *nats.Msg) {
	var payload struct {
		RegistryID      uint   `json:"registry_id"`
		SourceObjectID  int64  `json:"source_object_id"`
		ShadowBindingID int64  `json:"shadow_binding_id"`
		ShadowSchema    string `json:"shadow_schema"`
		TargetTable     string `json:"target_table"`
		SourceTable     string `json:"source_table"`
		PKField         string `json:"primary_key_field"`
		PKType          string `json:"primary_key_type"`
		TraceID         string `json:"trace_id"`
		Action          string `json:"action"`
		Origin          string `json:"origin"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.publishResult(msg, CommandResult{Command: "create-default-columns", Status: "error", Error: "invalid payload"})
		return
	}
	trace := completeActionTrace(msg, payload.TraceID, payload.Action, payload.Origin, "sync_fields_to_shadow")

	schemaName := strings.TrimSpace(payload.ShadowSchema)
	if schemaName == "" {
		schemaName = "public"
	}

	h.logger.Info("action trace received",
		zap.String("trace_id", trace.TraceID),
		zap.String("action", trace.Action),
		zap.String("origin", trace.Origin),
		zap.String("subject", "cdc.cmd.create-default-columns"),
		zap.String("schema", schemaName),
		zap.String("table", payload.TargetTable),
		zap.Int64("source_object_id", payload.SourceObjectID),
		zap.Int64("shadow_binding_id", payload.ShadowBindingID),
	)
	h.logger.Info("creating default columns", zap.String("schema", schemaName), zap.String("table", payload.TargetTable))

	tableAlreadyExists := h.tableExistsInSchema(schemaName, payload.TargetTable)
	columnsAdded := 0

	// Resolve registry source-engine — vẫn cần cho scanFieldsDebezium bên dưới
	// (Mongo direct-scan fallback khi shadow table rỗng).
	preResolvedID := int64(payload.RegistryID)
	if payload.SourceObjectID > 0 {
		preResolvedID = payload.SourceObjectID
	}
	var soEngine struct {
		SourceEngineType string `gorm:"column:source_engine_type"`
	}
	h.db.Table("cdc_system.source_object_registry").Select("source_engine_type").Where("id = ?", preResolvedID).First(&soEngine)

	if !tableAlreadyExists {
		if err := h.shadowDB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, quoteCommandIdent(schemaName))).Error; err != nil {
			h.publishResult(msg, CommandResult{Command: "create-default-columns", TargetTable: payload.TargetTable, Status: "error", Error: "create schema: " + err.Error()})
			return
		}

		// Align với TRUTH SOURCE sinkworker/schema_manager.go:225-237:
		// PK = _gpay_id BIGINT (sonyflake runtime), _source_id TEXT NOT NULL (V2
		// ON CONFLICT anchor — partial UNIQUE index tạo riêng), 9 cột CDC khác.
		// Source PK (id / wallet_id / _id) là BUSINESS field — KHÔNG tạo ở
		// bootstrap; mapping rule sync sẽ ALTER ADD COLUMN với type đúng.
		// payload.PKField / PKType giữ trong struct cho FE compat nhưng không
		// còn dùng để build CREATE TABLE — tránh re-introduce 3-path divergence.
		createSQL := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s.%s (
				"_gpay_id" BIGINT PRIMARY KEY,
				"_source_id" TEXT NOT NULL,
				"_raw_data" JSONB NOT NULL DEFAULT '{}'::jsonb,
				"_source" VARCHAR(20) NOT NULL DEFAULT 'debezium',
				"_source_ts" BIGINT,
				"_synced_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				"_version" BIGINT NOT NULL DEFAULT 1,
				"_hash" VARCHAR(64),
				"_deleted" BOOLEAN NOT NULL DEFAULT FALSE,
				"_created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				"_updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			quoteCommandIdent(schemaName),
			quoteCommandIdent(payload.TargetTable),
		)
		if err := h.shadowDB.Exec(createSQL).Error; err != nil {
			h.publishResult(msg, CommandResult{Command: "create-default-columns", TargetTable: payload.TargetTable, Status: "error", Error: "create table: " + err.Error()})
			return
		}
	}

	// Always ensure CDC system columns exist
	if err := h.ensureCDCColumnsInSchema(schemaName, payload.TargetTable); err != nil {
		h.publishResult(msg, CommandResult{Command: "create-default-columns", TargetTable: payload.TargetTable, Status: "error", Error: err.Error()})
		return
	}

	// 1.5 Auto-discovery: Scan _raw_data for new fields and queue them as pending.
	// Fields discovered here are NOT auto-approved — they require explicit review
	// before the next sync will ALTER TABLE to add them to the shadow schema.
	h.logger.Info("triggering auto-discovery before sync", zap.String("table", payload.TargetTable))
	effectiveID := preResolvedID

	scanAdded, scanTotal, scanErr := h.scanFieldsDebezium(context.Background(), uint(effectiveID), payload.ShadowBindingID, payload.TargetTable, payload.SourceTable, soEngine.SourceEngineType, false)
	if scanErr != nil {
		h.logger.Warn("auto-discovery during sync failed (continuing with existing rules)",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.String("table", payload.TargetTable),
			zap.Error(scanErr),
		)
	} else {
		h.logger.Info("auto-discovery during sync ok",
			zap.String("trace_id", trace.TraceID),
			zap.String("table", payload.TargetTable),
			zap.Int("scanned_total", scanTotal),
			zap.Int("scanned_added", scanAdded),
		)
	}

	// 2. Add approved business fields (works for both new and existing tables).
	// Query theo source_object_id (identity) thay vì source_object_name (string
	// trùng được giữa các registry rows). Hai registry rows cùng tên source
	// (vd. `export_jobs` chia thành 2 master khác nhau) là use-case hợp lệ —
	// query theo NAME sẽ kéo mapping rules của registry kia → ALTER ADD
	// COLUMN cross-leak shadow. effectiveID đã resolve ở trên = V2
	// source_object_id (ưu tiên) hoặc legacy registry_id (fallback).
	var rules []model.MappingRuleV2
	var err error
	if payload.ShadowBindingID > 0 {
		rules, err = h.mappingV2Repo.ListActiveBySourceObjectAndBinding(context.Background(), effectiveID, payload.ShadowBindingID)
	} else {
		rules, err = h.mappingV2Repo.ListActiveBySourceObject(context.Background(), effectiveID)
	}
	if err != nil {
		h.logger.Error("failed to load mapping_rule_v2 for ALTER",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.Int64("source_object_id", effectiveID),
			zap.Int64("shadow_binding_id", payload.ShadowBindingID),
			zap.Error(err),
		)
	} else {
		h.logger.Info("loaded mapping_rule_v2 for ALTER",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.Int64("source_object_id", effectiveID),
			zap.Int64("shadow_binding_id", payload.ShadowBindingID),
			zap.Int("rules_count", len(rules)),
		)
		if len(rules) == 0 {
			h.logger.Warn("no active+approved mapping rules for source_object_id; check mapping_rule_v2.source_object_id + status='approved' + is_active=true",
				zap.String("trace_id", trace.TraceID),
				zap.Int64("source_object_id", effectiveID),
				zap.Int64("shadow_binding_id", payload.ShadowBindingID),
			)
		}
		// Pre-fetch existing columns với cả data_type để (a) phân biệt
		// "thật sự ADD" vs "IF NOT EXISTS no-op", (b) detect drift giữa
		// rule.DataType và actual PG column type — root cause của bug
		// `id BIGINT` trong khi rule = TEXT (CREATE TABLE đặt PK BIGINT
		// trước, ALTER ADD COLUMN IF NOT EXISTS sau là no-op nên type cũ
		// giữ nguyên). Drift → ALTER COLUMN TYPE … USING …::text::…
		existingColsWithType := h.listShadowColumnsWithType(schemaName, payload.TargetTable)
		columnsSkipped := 0
		columnsAlreadyExist := 0
		columnsAlteredType := 0
		for _, rule := range rules {
			colLower := strings.ToLower(strings.TrimSpace(rule.TargetColumn))
			if existingType, ok := existingColsWithType[colLower]; ok {
				if normalizePGType(rule.DataType) == normalizePGType(existingType) {
					columnsAlreadyExist++
					h.logger.Debug("column already exists with matching type, skipping",
						zap.String("trace_id", trace.TraceID),
						zap.String("column", rule.TargetColumn),
						zap.String("type", existingType),
					)
					continue
				}
				// Drift: column exists with a different type than the
				// approved rule. ALTER COLUMN TYPE … USING column::text::type
				// — text is the universal intermediate so PG can cast
				// numbers → text → uuid-shaped text, etc.
				h.logger.Warn("column type drift detected, altering",
					zap.String("trace_id", trace.TraceID),
					zap.String("column", rule.TargetColumn),
					zap.String("rule_type", rule.DataType),
					zap.String("existing_type", existingType),
				)
				alterTypeSQL := fmt.Sprintf(
					`ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s USING %s::text::%s`,
					quoteCommandIdent(schemaName),
					quoteCommandIdent(payload.TargetTable),
					quoteCommandIdent(rule.TargetColumn),
					rule.DataType,
					quoteCommandIdent(rule.TargetColumn),
					rule.DataType,
				)
				if err := h.shadowDB.Exec(alterTypeSQL).Error; err != nil {
					h.logger.Error("failed to ALTER COLUMN TYPE (drift fix)",
						zap.String("trace_id", trace.TraceID),
						zap.String("column", rule.TargetColumn),
						zap.String("from_type", existingType),
						zap.String("to_type", rule.DataType),
						zap.Error(err),
					)
					columnsSkipped++
					continue
				}
				columnsAlteredType++
				existingColsWithType[colLower] = strings.ToLower(rule.DataType)
				continue
			}
			alterSQL := fmt.Sprintf(`ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s %s`,
				quoteCommandIdent(schemaName), quoteCommandIdent(payload.TargetTable), quoteCommandIdent(rule.TargetColumn), rule.DataType)
			h.logger.Info("executing column sync",
				zap.String("trace_id", trace.TraceID),
				zap.String("column", rule.TargetColumn),
				zap.String("data_type", rule.DataType),
			)
			if err := h.shadowDB.Exec(alterSQL).Error; err != nil {
				h.logger.Warn("failed to add column",
					zap.String("trace_id", trace.TraceID),
					zap.String("column", rule.TargetColumn),
					zap.String("data_type", rule.DataType),
					zap.Error(err),
				)
				columnsSkipped++
				continue
			}
			existingColsWithType[colLower] = strings.ToLower(rule.DataType)
			columnsAdded++
		}
		h.logger.Info("ALTER TABLE summary",
			zap.String("trace_id", trace.TraceID),
			zap.String("table", payload.TargetTable),
			zap.Int("rules_total", len(rules)),
			zap.Int("columns_added", columnsAdded),
			zap.Int("columns_altered_type", columnsAlteredType),
			zap.Int("columns_already_exist", columnsAlreadyExist),
			zap.Int("columns_skipped", columnsSkipped),
		)
	}

	// 3. Update states
	h.db.Model(&model.TableRegistry{}).Where("target_table = ?", payload.TargetTable).Update("is_table_created", true)
	if payload.SourceObjectID > 0 {
		h.db.Table("cdc_system.shadow_binding").
			Where("source_object_id = ?", payload.SourceObjectID).
			Updates(map[string]interface{}{"ddl_status": "created", "updated_at": gorm.Expr("NOW()")})
	}

	h.logger.Info("default columns ensured",
		zap.String("trace_id", trace.TraceID),
		zap.String("action", trace.Action),
		zap.String("schema", schemaName),
		zap.String("table", payload.TargetTable),
		zap.Int("columns_processed", columnsAdded),
	)

	h.publishResult(msg, CommandResult{
		Command:      "create-default-columns",
		TargetTable:  payload.TargetTable,
		RowsAffected: columnsAdded,
		Status:       "success",
		TraceID:      trace.TraceID,
		Action:       trace.Action,
		Origin:       trace.Origin,
	})
}

// HandleDiscover subscribes to "cdc.cmd.discover" and auto-generates mapping rules
// by scanning DW table columns via information_schema.
//
// Phase D — when payload carries Provisioning=true + SourceID, this
// handler also publishes cdc.evt.provisioning.step_completed (success
// or failure) so the orchestrator can finalize mapping_pending →
// mapping_ready and fan-out the next Advance.
func (h *CommandHandler) HandleDiscover(msg *nats.Msg) {
	var payload struct {
		RegistryID    uint   `json:"registry_id"`
		TargetTable   string `json:"target_table"`
		SourceTable   string `json:"source_table"`
		Provisioning  bool   `json:"provisioning,omitempty"`
		SourceID      int64  `json:"_source_id,omitempty"`
		ReplyTo       string `json:"reply_to"`
		CorrelationID string `json:"correlation_id,omitempty"`
		TraceID       string `json:"trace_id,omitempty"`
		SpanID        string `json:"span_id,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Error("cdc.cmd.discover: invalid payload", zap.Error(err))
		return
	}

	// Phase D (Q4): defer emit so panic/early-return paths still
	// signal the orchestrator. Only fires for provisioning flow.
	var stepErr error
	defer func() {
		if !payload.Provisioning {
			return
		}
		emitStepCompleted(h.natsConn, h.logger,
			payload.SourceID, "discover", stepErr,
			payload.CorrelationID, "discover_handler",
			payload.TraceID, payload.SpanID)
	}()

	h.logger.Info("discovering mappings",
		zap.String("target", payload.TargetTable),
		zap.String("source", payload.SourceTable),
		zap.Bool("provisioning", payload.Provisioning),
	)
	schemaName := h.resolveTargetSchema(payload.TargetTable)

	// Step 1: Discover columns from information_schema
	var cols []struct {
		ColumnName string `gorm:"column:column_name"`
		DataType   string `gorm:"column:data_type"`
	}

	// Phase A3 Hybrid — targetTable is in shadowDB (cdc_shadow).
	// Normalize table name to handle hyphens (e.g. refund-requests -> refund_requests).
	normalizedTable := naming.NormalizeIdentifier(payload.TargetTable)
	discoveryDB := h.db
	if h.shadowDB != nil {
		discoveryDB = h.shadowDB
	}

	if err := discoveryDB.WithContext(context.Background()).Raw(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = ? AND table_schema = ?
		ORDER BY ordinal_position
	`, normalizedTable, schemaName).Scan(&cols).Error; err != nil {
		stepErr = fmt.Errorf("get columns: %w", err)
		h.logger.Error("discover: query information_schema failed",
			zap.String("table", normalizedTable),
			zap.String("schema", schemaName),
			zap.Error(err))
		h.publishResult(msg, CommandResult{
			Command:     "discover",
			RegistryID:  payload.RegistryID,
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       fmt.Sprintf("get columns: %v", err),
		})
		return
	}

	// 2. Get existing mapping rules to avoid duplicates
	existing, err := h.mappingRepo.GetByTable(context.Background(), payload.SourceTable)
	if err != nil {
		h.logger.Error("discover: failed to get existing rules", zap.Error(err))
	}
	existingFields := make(map[string]bool, len(existing))
	for _, r := range existing {
		existingFields[r.SourceField] = true
	}

	// 3. Create new mapping rules for unknown columns
	count := 0
	for _, col := range cols {
		// Skip CDC metadata columns
		if strings.HasPrefix(col.ColumnName, "_") {
			continue
		}
		if existingFields[col.ColumnName] {
			continue
		}
		rule := &model.MappingRule{
			SourceTable:  payload.SourceTable,
			SourceField:  col.ColumnName,
			TargetColumn: col.ColumnName,
			// data_type stored in cdc_mapping_rules has a strict CHECK
			// constraint (see migration mapping_rules_data_type_chk).
			// information_schema returns lowercase verbose names like
			// "timestamp without time zone" / "numeric" — those fail
			// the regex. Normalize to the canonical uppercase form
			// before INSERT.
			DataType:   normalizeMappingRuleDataType(col.DataType),
			IsActive:   true,
			IsEnriched: false,
			IsNullable: true,
		}
		if err := h.mappingRepo.Create(context.Background(), rule); err != nil {
			h.logger.Error("discover: failed to create rule", zap.String("field", col.ColumnName), zap.Error(err))
			continue
		}
		count++
	}

	// Phase multi_engine_unified — Cascade Liability gate (lesson L1399).
	// If the shadow table holds NO business columns (only `_*` cdc meta)
	// AND no rule existed already, the discover step is silently empty:
	// pipeline would advance mapping_pending → mapping_ready → running
	// with zero mapping coverage, master would land empty rows, user
	// catches it after a day. Refuse to cascade — fail the step so the
	// orchestrator pins state to `failed` with last_step_error set.
	totalRules := count + len(existing)
	if totalRules == 0 {
		stepErr = fmt.Errorf("discover: 0 mapping rules — shadow table %q has no business columns (cdc meta only). Likely cause: source schema not yet projected into shadow (e.g. Mongo schemaless without sample, or shadow_bind ran before any source row landed). Refusing to cascade", payload.TargetTable)
		h.logger.Error("discover: zero rules — break cascade",
			zap.String("target", payload.TargetTable),
			zap.Int("cdc_only_columns", len(cols)))
		h.publishResult(msg, CommandResult{
			Command:     "discover",
			RegistryID:  payload.RegistryID,
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       stepErr.Error(),
		})
		return
	}

	// V2 bridge — V1 cdc_mapping_rules feeds the legacy shadow ingest
	// path; mapping_rule_v2 is the sole truth for the shadow→master
	// transmute path (transmuter.go:loadRules). Auto cascade must seed V2
	// or the master table stays empty even when shadow is hot.
	if payload.Provisioning && payload.SourceID > 0 {
		if v2Created, v2Err := h.bridgeMappingRulesToV2(context.Background(), payload.SourceID, payload.SourceTable); v2Err != nil {
			h.logger.Warn("discover: V2 bridge encountered errors",
				zap.Int64("_source_id", payload.SourceID),
				zap.Int("v2_created", v2Created),
				zap.Error(v2Err))
		} else {
			h.logger.Info("discover: V2 bridge done",
				zap.Int64("_source_id", payload.SourceID),
				zap.Int("v2_created", v2Created))
		}
	}

	h.logger.Info("discover complete", zap.Int("new_rules", count), zap.String("table", payload.TargetTable))
	h.publishResult(msg, CommandResult{
		Command:      "discover",
		RegistryID:   payload.RegistryID,
		TargetTable:  payload.TargetTable,
		RowsAffected: count,
		Status:       "success",
	})
}

// bridgeMappingRulesToV2 mirrors V1 cdc_mapping_rules into V2 mapping_rule_v2
// for the given source object so the transmute path (shadow→master) has rules
// to apply. Idempotent via the ux_v2_mapping_rule_identity unique index.
//
// V2 source_format CHECK only allows {raw,jsonpath,expression}; transmuter.go
// reads source_path first when present, falling back to source_field. For
// Debezium-envelope shadows (PG/MariaDB connectors) we set source_path =
// "after.<field>". For Mongo we leave source_path NULL — the connector emits
// `after` as a JSON-encoded string so gjson cannot dereference; transmuter
// will fall back to the bare source_field path which mirrors the field name
// at the top-level of _raw_data once flattened by the ingest path.
func (h *CommandHandler) bridgeMappingRulesToV2(ctx context.Context, sourceID int64, sourceTable string) (int, error) {
	type ctxRow struct {
		Engine          string
		MasterBindingID *int64
	}
	var ctxr ctxRow
	if err := h.db.WithContext(ctx).Raw(`
		SELECT sor.source_engine_type AS engine,
		       mb.id AS master_binding_id
		  FROM cdc_system.source_object_registry sor
		  LEFT JOIN cdc_system.master_binding mb
		    ON mb.source_object_id = sor.id
		   AND mb.is_active = TRUE
		 WHERE sor.id = ?
		 ORDER BY mb.updated_at DESC NULLS LAST
		 LIMIT 1
	`, sourceID).Scan(&ctxr).Error; err != nil {
		return 0, fmt.Errorf("v2 bridge: resolve context: %w", err)
	}
	if ctxr.MasterBindingID == nil {
		return 0, fmt.Errorf("v2 bridge: source %d has no active master_binding", sourceID)
	}

	v1Rules, err := h.mappingRepo.GetByTable(ctx, sourceTable)
	if err != nil {
		return 0, fmt.Errorf("v2 bridge: fetch v1 rules: %w", err)
	}

	useEnvelope := strings.EqualFold(ctxr.Engine, "postgresql") ||
		strings.EqualFold(ctxr.Engine, "mariadb") ||
		strings.EqualFold(ctxr.Engine, "mysql")

	created := 0
	var lastErr error
	for _, r := range v1Rules {
		if !r.IsActive {
			continue
		}
		var sourcePath any
		if useEnvelope {
			sourcePath = "after." + r.SourceField
		} else {
			sourcePath = nil
		}
		// ON CONFLICT matches ux_v2_mapping_rule_identity (migration 067 —
		// adds COALESCE(shadow_binding_id, 0) to the unique key so each
		// shadow binding can carry its own target_column definitions).
		// status='approved' so transmuter immediately picks up the rule.
		res := h.db.WithContext(ctx).Exec(`
			INSERT INTO cdc_system.mapping_rule_v2
			  (source_object_id, master_binding_id, source_field, source_path,
			   target_column, data_type, source_format,
			   is_nullable, is_active, status, created_by, updated_by,
			   created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'raw',
			        TRUE, TRUE, 'approved', 'discover_handler', 'discover_handler',
			        NOW(), NOW())
			ON CONFLICT (source_object_id, COALESCE(shadow_binding_id, 0), COALESCE(master_binding_id, 0), target_column)
			DO NOTHING
		`,
			sourceID, *ctxr.MasterBindingID,
			r.SourceField, sourcePath,
			r.TargetColumn, r.DataType,
		)
		if res.Error != nil {
			lastErr = res.Error
			continue
		}
		if res.RowsAffected > 0 {
			created++
		}
	}

	// master_bind ran BEFORE V2 was seeded (orchestrator step order is
	// master_bind → discover), so the master table on disk holds only
	// `_*` cdc meta cols at this point. Republish cdc.cmd.master-create
	// so MasterDDLGenerator.Apply runs the additive ALTER ADD COLUMN
	// pass for the rules just bridged. Best-effort — failure here is
	// logged by HandleMasterCreate and the schedule_enable step will
	// proceed regardless; the gap surfaces later as a transmute upsert
	// error rather than a silent cascade.
	if created > 0 && h.natsConn != nil {
		var masterTable string
		if err := h.db.WithContext(ctx).Raw(`
			SELECT master_table FROM cdc_system.master_binding WHERE id = ?
		`, *ctxr.MasterBindingID).Scan(&masterTable).Error; err == nil && masterTable != "" {
			payload, _ := json.Marshal(map[string]any{
				"master_table":   masterTable,
				"correlation_id": fmt.Sprintf("v2bridge-src-%d", sourceID),
				"triggered_by":   "v2_bridge",
			})
			if perr := h.natsConn.Publish("cdc.cmd.master-create", payload); perr != nil {
				h.logger.Warn("v2 bridge: republish master-create failed",
					zap.String("master", masterTable), zap.Error(perr))
			}
		}
	}
	return created, lastErr
}

// HandleBackfill subscribes to "cdc.cmd.backfill" and populates target columns from _raw_data.
func (h *CommandHandler) HandleBackfill(msg *nats.Msg) {
	var payload struct {
		RegistryID   uint   `json:"registry_id"`
		TargetTable  string `json:"target_table"`
		SourceField  string `json:"source_field"`
		TargetColumn string `json:"target_column"`
		DataType     string `json:"data_type"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Error("cdc.cmd.backfill: invalid payload", zap.Error(err))
		return
	}

	h.logger.Info("backfilling column",
		zap.String("table", payload.TargetTable),
		zap.String("field", payload.SourceField),
		zap.String("column", payload.TargetColumn),
	)
	schemaName := h.resolveTargetSchema(payload.TargetTable)

	// Build type-aware extraction expression
	castExpr := buildCastExpr(payload.SourceField, payload.DataType)

	// Update NULL target column values from _raw_data JSONB
	sql := fmt.Sprintf(
		`UPDATE %s SET %s = %s WHERE %s IS NULL AND _raw_data IS NOT NULL`,
		quoteCommandQualifiedTable(schemaName, payload.TargetTable),
		quoteCommandIdent(payload.TargetColumn),
		castExpr,
		quoteCommandIdent(payload.TargetColumn),
	)
	result := h.db.WithContext(context.Background()).Exec(sql)
	if result.Error != nil {
		h.logger.Error("backfill failed",
			zap.String("table", payload.TargetTable),
			zap.Error(result.Error),
		)
		h.publishResult(msg, CommandResult{
			Command:     "backfill",
			RegistryID:  payload.RegistryID,
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       result.Error.Error(),
		})
		return
	}

	rows := int(result.RowsAffected)
	h.logger.Info("backfill complete",
		zap.String("table", payload.TargetTable),
		zap.String("column", payload.TargetColumn),
		zap.Int("rows", rows),
	)
	h.publishResult(msg, CommandResult{
		Command:      "backfill",
		RegistryID:   payload.RegistryID,
		TargetTable:  payload.TargetTable,
		RowsAffected: rows,
		Status:       "success",
	})
}

// HandleMasterSwap performs atomic RENAME for a master table.
func (h *CommandHandler) HandleMasterSwap(msg *nats.Msg) {
	var req struct {
		MasterName   string `json:"master_name"`
		NewTableName string `json:"new_table_name"`
		Reason       string `json:"reason"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.Error("cdc.cmd.master-swap: invalid payload", zap.Error(err))
		return
	}

	jobID := msg.Header.Get("Cdc-Job-Id")

	h.logger.Info("master swap starting", zap.String("master", req.MasterName), zap.String("new_table", req.NewTableName), zap.String("job_id", jobID))

	err := h.db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL lock_timeout = '3s'").Error; err != nil {
			return fmt.Errorf("set lock_timeout: %w", err)
		}

		ts := time.Now().Unix()
		oldName := fmt.Sprintf("%s_old_%d", req.MasterName, ts)

		renameCur := fmt.Sprintf(`ALTER TABLE public."%s" RENAME TO "%s"`, req.MasterName, oldName)
		if err := tx.Exec(renameCur).Error; err != nil {
			return fmt.Errorf("rename current: %w", err)
		}
		renameNew := fmt.Sprintf(`ALTER TABLE public."%s" RENAME TO "%s"`, req.NewTableName, req.MasterName)
		if err := tx.Exec(renameNew).Error; err != nil {
			return fmt.Errorf("rename new: %w", err)
		}

		details, _ := json.Marshal(map[string]string{
			"old_table": oldName,
			"new_table": req.NewTableName,
			"reason":    req.Reason,
		})
		return tx.Exec(
			`INSERT INTO cdc_activity_log
			    (operation, target_table, status, details, triggered_by, started_at, completed_at)
			 VALUES ('master_swap', ?, 'success', ?::jsonb, 'manual', NOW(), NOW())`,
			req.MasterName, string(details),
		).Error
	})

	if err != nil {
		h.logger.Error("master swap failed", zap.String("master", req.MasterName), zap.Error(err))
		if h.natsConn != nil && jobID != "" {
			errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
			resultMsg := &nats.Msg{
				Subject: "cdc.evt.master-swap.completed",
				Header:  nats.Header{"Cdc-Job-Id": []string{jobID}, "Cdc-Job-Status": []string{"failed"}},
				Data:    errPayload,
			}
			_ = h.natsConn.PublishMsg(resultMsg)
		}
		return
	}

	h.logger.Info("master swap complete", zap.String("master", req.MasterName))
	if h.natsConn != nil && jobID != "" {
		resPayload, _ := json.Marshal(map[string]string{"master_name": req.MasterName, "new_table_name": req.NewTableName})
		resultMsg := &nats.Msg{
			Subject: "cdc.evt.master-swap.completed",
			Header:  nats.Header{"Cdc-Job-Id": []string{jobID}, "Cdc-Job-Status": []string{"success"}},
			Data:    resPayload,
		}
		_ = h.natsConn.PublishMsg(resultMsg)
	}
}

// HandleDiscoverMongoDatabases handles MongoDB database discovery.
//
// Accepts EITHER full URI (preferred, preserves auth/replicaSet/TLS) OR
// legacy host+port pair (backward compat with older callers). When both
// are provided, URI wins. Raw URI is never logged — only the
// SanitizeMongoDSN form is used for logs/error reply.
func (h *CommandHandler) HandleDiscoverMongoDatabases(msg *nats.Msg) {
	var req struct {
		URI     string `json:"uri"`
		Host    string `json:"host"`
		Port    string `json:"port"`
		ReplyTo string `json:"reply_to"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}

	if h.mongoSvc == nil {
		return
	}

	uri := req.URI
	if uri == "" {
		if req.Host == "" || req.Port == "" {
			h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
				"status": "cluster_err",
				"error":  "missing connection: provide either uri or host+port",
			})
			return
		}
		uri = fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port)
	}

	sanitized := service.SanitizeMongoDSN(uri)
	h.logger.Info("introspect.mongo.databases.start",
		zap.String("sanitized_dsn", sanitized), zap.Bool("audit", true))

	dbs, err := h.mongoSvc.DiscoverDatabases(uri)
	if err != nil {
		observability.Ctx(context.Background(), h.logger).Error("introspect.mongo.databases.failed",
			observability.ErrorField(err),
			observability.Attrs(
				zap.String("sanitized_dsn", sanitized),
			),
		)
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status":        "cluster_err",
			"error":         err.Error(),
			"sanitized_dsn": sanitized,
		})
		return
	}

	h.logger.Info("introspect.mongo.databases.ok",
		zap.String("sanitized_dsn", sanitized), zap.Int("count", len(dbs)), zap.Bool("audit", true))
	h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
		"status":        "ok",
		"databases":     dbs,
		"sanitized_dsn": sanitized,
	})
}

// HandleDiscoverMongoCollections handles MongoDB collection discovery.
//
// Replies with a 5-case status field (ok / cluster_err / db_missing /
// empty / unknown) so the FE can render an actionable message instead
// of a catch-all error. db_missing payload includes available_databases
// to help the user pick the right name.
func (h *CommandHandler) HandleDiscoverMongoCollections(msg *nats.Msg) {
	var req struct {
		URI      string `json:"uri"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		DB       string `json:"db"`
		Database string `json:"database"`
		ReplyTo  string `json:"reply_to"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}

	if h.mongoSvc == nil {
		return
	}

	database := req.Database
	if database == "" {
		database = req.DB
	}
	if database == "" {
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status": "cluster_err",
			"error":  "missing database",
		})
		return
	}

	uri := req.URI
	if uri == "" {
		if req.Host == "" || req.Port == "" {
			h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
				"status": "cluster_err",
				"error":  "missing connection: provide either uri or host+port",
			})
			return
		}
		uri = fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port)
	}

	sanitized := service.SanitizeMongoDSN(uri)
	h.logger.Info("introspect.mongo.collections.start",
		zap.String("sanitized_dsn", sanitized), zap.String("database", database), zap.Bool("audit", true))

	// Probe DB existence first so we can return db_missing with the
	// candidate list instead of a vacuous "no collections" reply.
	allDBs, err := h.mongoSvc.DiscoverDatabases(uri)
	if err != nil {
		h.logger.Error("introspect.mongo.collections.cluster_err",
			zap.String("sanitized_dsn", sanitized), zap.Error(err))
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status":        "cluster_err",
			"error":         err.Error(),
			"sanitized_dsn": sanitized,
		})
		return
	}
	foundDB := false
	for _, d := range allDBs {
		if d == database {
			foundDB = true
			break
		}
	}
	if !foundDB {
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status":              "db_missing",
			"error":               fmt.Sprintf("database '%s' not found", database),
			"database":            database,
			"available_databases": allDBs,
			"sanitized_dsn":       sanitized,
		})
		return
	}

	cols, err := h.mongoSvc.DiscoverCollections(uri, database)
	if err != nil {
		h.logger.Error("introspect.mongo.collections.failed",
			zap.String("sanitized_dsn", sanitized), zap.String("database", database), zap.Error(err))
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status":        "cluster_err",
			"error":         err.Error(),
			"sanitized_dsn": sanitized,
		})
		return
	}
	if len(cols) == 0 {
		h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
			"status":        "empty",
			"collections":   []string{},
			"database":      database,
			"sanitized_dsn": sanitized,
		})
		return
	}

	h.logger.Info("introspect.mongo.collections.ok",
		zap.Bool("audit", true),
		observability.Attrs(
			zap.String("sanitized_dsn", sanitized),
			zap.String("database", database),
			zap.Int("count", len(cols)),
		),
	)
	h.replyMongoDiscovery(msg, req.ReplyTo, map[string]any{
		"status":        "ok",
		"collections":   cols,
		"database":      database,
		"sanitized_dsn": sanitized,
	})
}

func (h *CommandHandler) replyMongoDiscovery(msg *nats.Msg, replyTo string, payload map[string]any) {
	resp, _ := json.Marshal(payload)
	if replyTo != "" {
		if err := h.natsConn.Publish(replyTo, resp); err != nil {
			h.logger.Error("failed to publish discovery response",
				zap.String("reply_to", replyTo), zap.Error(err))
		}
		return
	}
	msg.Respond(resp)
}

// HandleIntrospect subscribes to "cdc.cmd.introspect" and scans a sample of _raw_data
// from the DW table to find unmapped fields. It replies via NATS Request-Reply.

// Debezium path is the sole CDC engine. Shadow→Master via TransmuterModule (R6).

// HandleBatchTransform applies mapping rules to populate typed columns from _raw_data.
// Subject: "cdc.cmd.batch-transform" (pub/sub pattern)
func (h *CommandHandler) HandleBatchTransform(msg *nats.Msg) {
	targetTable := string(msg.Data)
	h.logger.Info("batch transforming table", zap.String("table", targetTable))
	schemaName := h.resolveTargetSchema(targetTable)

	// 0. Check table exists + has _raw_data column
	if !h.tableExists(targetTable) {
		h.publishResult(msg, CommandResult{Command: "batch-transform", TargetTable: targetTable, Status: "skipped", Error: "table does not exist"})
		return
	}
	if !h.hasColumn(targetTable, "_raw_data") {
		h.publishResult(msg, CommandResult{Command: "batch-transform", TargetTable: targetTable, Status: "skipped", Error: "table has no _raw_data column yet"})
		return
	}

	// 1. Find source_table from registry
	reg := h.resolveTargetTableConfig(context.Background(), targetTable)
	var sourceTable string
	if reg != nil {
		sourceTable = reg.SourceTable
	} else {
		sourceTable = targetTable
	}

	// 2. Get active mapping rules. Reads from mapping_rule_v2 because the
	// FE/CMS approve flow writes there; the legacy cdc_mapping_rules
	// table is unused (V2 migration complete).
	rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(context.Background(), sourceTable)
	if err != nil || len(rules) == 0 {
		h.publishResult(msg, CommandResult{
			Command:     "batch-transform",
			TargetTable: targetTable,
			Status:      "error",
			Error:       fmt.Sprintf("no active mapping rules for table %s (source: %s)", targetTable, sourceTable),
		})
		return
	}

	// 3. Build UPDATE SET clause from mapping rules.
	// Dedupe by target_column: if registry has two active rules pointing to
	// the same column (e.g. mongoose `__v` mapped from two source fields),
	// Postgres rejects the UPDATE with 42601 "multiple assignments to same
	// column". First rule wins, subsequent dupes logged + skipped.
	var setClauses []string
	var whereClauses []string
	seenCols := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		colKey := strings.ToLower(strings.TrimSpace(rule.TargetColumn))
		if _, dup := seenCols[colKey]; dup {
			h.logger.Warn("batch transform: duplicate target_column in mapping rules, skipping rule",
				zap.String("table", targetTable),
				zap.String("target_column", rule.TargetColumn),
				zap.String("source_field", rule.SourceField),
			)
			continue
		}
		seenCols[colKey] = struct{}{}
		castExpr := buildCastExpr(rule.SourceField, rule.DataType)
		quotedCol := quoteCommandIdent(rule.TargetColumn)
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quotedCol))
	}

	if len(setClauses) == 0 {
		h.publishResult(msg, CommandResult{
			Command:     "batch-transform",
			TargetTable: targetTable,
			Status:      "success",
			Error:       "no active rules to transform",
		})
		return
	}

	setClauses = append(setClauses, "_updated_at = NOW()")

	// Shadow tables live in the shadow plane (cdc_shadow:5436), not the
	// destination plane. Same root cause as HandleAlterColumn fix.
	execDB := h.db
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}

	quotedTable := quoteCommandQualifiedTable(schemaName, targetTable)
	whereExpr := strings.Join(whereClauses, " OR ")
	setExpr := strings.Join(setClauses, ", ")

	// Detect primary key. Chunked path requires a single-column PK so the
	// CTE can stream id batches; composite or missing PK falls back to one
	// large UPDATE (same behaviour as before this refactor).
	pkCol, pkErr := h.detectPrimaryKey(execDB, schemaName, targetTable)

	chunkSize := h.transformChunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	if pkErr != nil || pkCol == "" {
		h.logger.Warn("batch transform: PK not detected, running unchunked UPDATE",
			zap.String("table", targetTable),
			zap.Error(pkErr),
		)
		transformSQL := fmt.Sprintf(`UPDATE %s SET %s WHERE _raw_data IS NOT NULL AND (%s)`,
			quotedTable, setExpr, whereExpr,
		)
		result := execDB.Exec(transformSQL)
		if result.Error != nil {
			h.publishResult(msg, CommandResult{
				Command:     "batch-transform",
				TargetTable: targetTable,
				Status:      "error",
				Error:       result.Error.Error(),
			})
			return
		}
		h.logger.Info("batch transform completed (unchunked)",
			zap.String("table", targetTable),
			zap.Int64("rows_affected", result.RowsAffected),
		)
		h.publishResult(msg, CommandResult{
			Command:      "batch-transform",
			TargetTable:  targetTable,
			RowsAffected: int(result.RowsAffected),
			Status:       "success",
		})
		return
	}

	// Chunked path: cursor-based loop. Each iteration UPDATEs at most
	// chunkSize rows whose PK is greater than the previous chunk's max PK.
	// Cursor (vs unbounded WHERE) is required because the SET clause may
	// not change WHERE-match state (e.g. rule produces NULL → SET NULL →
	// still NULL → row keeps matching). Without a cursor the next chunk
	// would re-select the same rows and never make progress.
	// Termination: chunk returns 0 rows (cursor passed the last PK).
	quotedPK := quoteCommandIdent(pkCol)
	const maxIterations = 100000
	var totalRows int64
	var productiveIters int
	var lastPK interface{}
	hasCursor := false
	for i := 0; i < maxIterations; i++ {
		cursorClause := ""
		var args []interface{}
		if hasCursor {
			cursorClause = fmt.Sprintf(" AND %s > ?", quotedPK)
			args = append(args, lastPK)
		}
		chunkSQL := fmt.Sprintf(
			`WITH chunk AS (
                SELECT %s AS pk FROM %s
                WHERE _raw_data IS NOT NULL AND (%s)%s
                ORDER BY %s LIMIT %d
            )
            UPDATE %s s SET %s
            FROM chunk c
            WHERE s.%s = c.pk
            RETURNING s.%s`,
			quotedPK, quotedTable, whereExpr, cursorClause, quotedPK, chunkSize,
			quotedTable, setExpr, quotedPK, quotedPK,
		)
		rows, err := execDB.Raw(chunkSQL, args...).Rows()
		if err != nil {
			h.publishResult(msg, CommandResult{
				Command:      "batch-transform",
				TargetTable:  targetTable,
				RowsAffected: int(totalRows),
				Status:       "error",
				Error:        err.Error(),
			})
			return
		}
		var chunkCount int64
		var maxPK interface{}
		for rows.Next() {
			var v interface{}
			if err := rows.Scan(&v); err != nil {
				rows.Close()
				h.publishResult(msg, CommandResult{
					Command:      "batch-transform",
					TargetTable:  targetTable,
					RowsAffected: int(totalRows),
					Status:       "error",
					Error:        err.Error(),
				})
				return
			}
			maxPK = v
			chunkCount++
		}
		rows.Close()
		totalRows += chunkCount
		if chunkCount == 0 {
			break
		}
		productiveIters++
		hasCursor = true
		lastPK = maxPK
		if chunkCount < int64(chunkSize) {
			break
		}
	}

	h.logger.Info("batch transform completed (chunked)",
		zap.String("table", targetTable),
		zap.String("pk", pkCol),
		zap.Int("chunk_size", chunkSize),
		zap.Int("iterations", productiveIters),
		zap.Int64("rows_affected", totalRows),
	)
	h.publishResult(msg, CommandResult{
		Command:      "batch-transform",
		TargetTable:  targetTable,
		RowsAffected: int(totalRows),
		Status:       "success",
	})
}

// detectPrimaryKey returns the single PK column name for schemaName.tableName.
// Returns "" when the table has no PK or a composite PK (chunking unsupported).
func (h *CommandHandler) detectPrimaryKey(execDB *gorm.DB, schemaName, tableName string) (string, error) {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	var cols []string
	sql := `
        SELECT a.attname
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = (?::text)::regclass AND i.indisprimary
        ORDER BY array_position(i.indkey, a.attnum)`
	qualified := schemaName + "." + tableName
	if err := execDB.Raw(sql, qualified).Scan(&cols).Error; err != nil {
		return "", err
	}
	if len(cols) != 1 {
		return "", nil
	}
	return cols[0], nil
}

// HandleScanRawData scans _raw_data JSONB column to find fields not yet mapped.
// Subject: "cdc.cmd.scan-raw-data" (request-reply pattern)
func (h *CommandHandler) HandleScanRawData(msg *nats.Msg) {
	var payload struct {
		TargetTable string `json:"target_table"`
		ReplyTo     string `json:"reply_to"`
	}
	// Backward compatibility: if not JSON, treat as raw targetTable string
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		payload.TargetTable = string(msg.Data)
	}

	targetTable := payload.TargetTable
	h.logger.Info("scanning _raw_data for unmapped fields", zap.String("table", targetTable))
	schemaName := h.resolveTargetSchema(targetTable)

	// 0. Check table + _raw_data exists
	if !h.tableExists(targetTable) || !h.hasColumn(targetTable, "_raw_data") {
		res, _ := json.Marshal(map[string]interface{}{"status": "skipped", "reason": "table or _raw_data column not found"})
		h.nats_publish(msg, "cdc.result.scan-raw-data", res)
		return
	}

	// 1. Get all distinct keys and their JSON type from _raw_data JSONB (sample from recent rows)
	var rawFields []struct {
		Key  string
		Type string
	}
	sql := fmt.Sprintf(
		`SELECT key, MAX(type) as type FROM (
			SELECT key, jsonb_typeof(_raw_data->key) as type
			FROM %s, jsonb_object_keys(_raw_data) as key
			WHERE _raw_data IS NOT NULL AND _raw_data != '{}'::jsonb
			LIMIT 1000
		) sub WHERE type != 'null' GROUP BY key ORDER BY key`, quoteCommandQualifiedTable(schemaName, targetTable))
	if err := h.db.Raw(sql).Scan(&rawFields).Error; err != nil {
		res := map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("failed to scan _raw_data: %s", err.Error()),
		}
		resBytes, _ := json.Marshal(res)
		h.nats_publish(msg, "cdc.result.scan-raw-data", resBytes)
		return
	}

	// 2. Fetch shadow_binding to know the scope
	var binding struct {
		ID             int64
		SourceObjectID int64
	}
	h.db.Table("cdc_system.shadow_binding").Select("id, source_object_id").Where("shadow_table = ?", targetTable).Limit(1).Scan(&binding)

	var existingV2 []struct {
		SourceField string
	}
	if binding.ID > 0 {
		h.db.Table("cdc_system.mapping_rule_v2").
			Select("source_field").
			Where("shadow_binding_id = ?", binding.ID).
			Scan(&existingV2)
	} else if binding.SourceObjectID > 0 {
		h.db.Table("cdc_system.mapping_rule_v2").
			Select("source_field").
			Where("source_object_id = ?", binding.SourceObjectID).
			Scan(&existingV2)
	}

	mappedFields := make(map[string]bool)
	for _, r := range existingV2 {
		mappedFields[r.SourceField] = true
	}

	// 3. Also skip system/internal fields
	skipFields := map[string]bool{
		"_id": true, "_raw_data": true, "_source": true, "_synced_at": true,
		"_version": true, "_hash": true, "_deleted": true, "_created_at": true, "_updated_at": true,
		"id": true,
	}

	// 4. Find unmapped fields and auto-insert to V2
	var unmappedFields []string
	for _, f := range rawFields {
		key := f.Key
		if !mappedFields[key] && !skipFields[key] {
			unmappedFields = append(unmappedFields, key)

			pgType := "TEXT"
			switch f.Type {
			case "number":
				pgType = "NUMERIC"
			case "boolean":
				pgType = "BOOLEAN"
			case "object", "array":
				pgType = "JSONB"
			}

			if binding.ID > 0 {
				sourceType := f.Type
				sys := "system-scan"
				rule := model.MappingRuleV2{
					SourceObjectID:  binding.SourceObjectID,
					ShadowBindingID: &binding.ID,
					SourceField:     key,
					TargetColumn:    key,
					DataType:        pgType,
					SourceDataType:  &sourceType,
					SourceFormat:    "raw",
					IsActive:        false,
					Status:          "pending",
					IsNullable:      true,
					CreatedBy:       &sys,
				}
				h.db.Create(&rule)
			}
		}
	}

	res := map[string]interface{}{
		"status":         "ok",
		"table":          targetTable,
		"source_table":   targetTable,
		"total_raw_keys": len(rawFields),
		"mapped_count":   len(existingV2),
		"new_fields":     unmappedFields,
	}
	res = sanitizeAdminResultMap(res)
	resBytes, _ := json.Marshal(res)
	h.nats_publish(msg, "cdc.result.scan-raw-data", resBytes)

	h.logger.Info("_raw_data scan completed",
		zap.String("table", targetTable),
		zap.Int("raw_keys", len(rawFields)),
		zap.Int("unmapped", len(unmappedFields)),
	)
}

// validScanIdent guards an explode_path segment so it can be embedded into the
// jsonb #> path literal without SQL injection. PG identifier rules: starts with
// letter/underscore, then letter/digit/underscore, max 63 bytes.
func validScanIdent(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// explodePathToPGPath converts a flatten explode_path ("after.items[*]", "items")
// into a Postgres jsonb text-array path literal "{after,items}" for the #>
// operator. Every segment is validated as a strict identifier so the literal is
// safe to embed in SQL. Multi-level array iteration ("a[*].b[*]") is rejected —
// matches the V3 single-level explode limit.
func explodePathToPGPath(path string) (string, bool, error) {
	cleaned := strings.TrimSpace(path)
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, ".")
	if cleaned == "" {
		return "", false, fmt.Errorf("explode_path is empty")
	}

	segs := strings.Split(cleaned, ".")
	out := make([]string, 0, len(segs))
	var isArrayExplode bool

	for i, seg := range segs {
		seg = strings.TrimSpace(seg)
		hadStar := strings.HasSuffix(seg, "[*]")

		// Cú pháp kiểm tra đa tầng array lồng nhau
		if hadStar {
			if i != len(segs)-1 {
				return "", false, fmt.Errorf("multi-level array explode unsupported: [*] only allowed on last segment")
			}
			seg = strings.TrimSuffix(seg, "[*]")
			isArrayExplode = true // Đánh dấu lại để tầng trên biết đường Unnest JSONB
		}

		if !validScanIdent(seg) {
			return "", false, fmt.Errorf("invalid explode_path segment %q", seg)
		}
		out = append(out, seg)
	}

	// Trả về dạng {field1,field2} kèm flag báo mảng
	return "{" + strings.Join(out, ",") + "}", isArrayExplode, nil
}

// HandleScanArrayFields is the discovery step of the flatten sync type. It scans
// the array at explode_path inside _raw_data and proposes one PENDING
// mapping_rule_v2 per element field (status=pending, is_active=false), so an
// operator reviews/approves the candidate columns BEFORE the master table is
// created. No auto-DDL on the master side. Mirrors HandleScanRawData but
// navigates into the array via jsonb_array_elements + jsonb_each.
// Subject: "cdc.cmd.scan-array" (request-reply).
func (h *CommandHandler) HandleScanArrayFields(msg *nats.Msg) {
	var payload struct {
		TargetTable     string `json:"target_table"`
		ExplodePath     string `json:"explode_path"`
		MasterBindingID *int64 `json:"master_binding_id"`
		ReplyTo         string `json:"reply_to"`
		Mode            string `json:"mode"` // "sample" hoặc "full"
	}
	h.logger.Info("cmd: scan_array_fields", zap.Any("payload", payload))
	replySubject := msg.Reply
	if replySubject == "" && payload.ReplyTo != "" {
		replySubject = payload.ReplyTo
	} else if replySubject == "" {
		replySubject = "cdc.result.scan-array"
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "bad_json"})
		h.nats_publish(msg, replySubject, res)
		return
	}

	if payload.Mode == "" {
		payload.Mode = "sample"
	}

	targetTable := payload.TargetTable
	schemaName := h.resolveTargetSchema(targetTable)

	if payload.MasterBindingID != nil {
		var sb struct {
			ShadowSchema string `gorm:"column:shadow_schema"`
			ShadowTable  string `gorm:"column:shadow_table"`
		}
		if err := h.db.Raw(`
            SELECT sb.shadow_schema, sb.shadow_table
              FROM cdc_system.master_binding mb
              JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
             WHERE mb.id = ?`, *payload.MasterBindingID).Scan(&sb).Error; err == nil {
			if strings.TrimSpace(sb.ShadowTable) != "" {
				targetTable = sb.ShadowTable
			}
			if strings.TrimSpace(sb.ShadowSchema) != "" {
				schemaName = sb.ShadowSchema
			}
		}
	}

	if !h.tableExistsInSchema(schemaName, targetTable) || !h.hasColumnInSchema(schemaName, targetTable, "_raw_data") {
		res, _ := json.Marshal(map[string]interface{}{"status": "skipped", "reason": "table or _raw_data column not found"})
		h.nats_publish(msg, replySubject, res)
		return
	}

	pgPath, isArrayExplode, err := explodePathToPGPath(payload.ExplodePath)
	if err != nil {
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "invalid_explode_path: " + err.Error()})
		h.nats_publish(msg, replySubject, res)
		return
	}

	var pathType string
	errPathType := h.shadowDB.Raw(fmt.Sprintf(
		`SELECT jsonb_typeof(_raw_data #> ?::text[]) FROM %s WHERE _raw_data #> ?::text[] IS NOT NULL LIMIT 1`,
		quoteCommandQualifiedTable(schemaName, targetTable)), pgPath, pgPath,
	).Scan(&pathType).Error

	if errPathType != nil {
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "db_query_error: " + errPathType.Error()})
		h.nats_publish(msg, replySubject, res)
		return
	}

	if pathType == "string" {
		res, _ := json.Marshal(map[string]interface{}{"status": "skipped", "reason": "Dữ liệu đang là 'string' (stringified JSON)."})
		h.nats_publish(msg, replySubject, res)
		return
	}

	isObject := pathType == "object"
	basePath := strings.TrimSuffix(strings.TrimSpace(payload.ExplodePath), "[*]")
	basePath = strings.TrimSuffix(basePath, ".")

	// -------------------------------------------------------------------------
	// ENGINE ĐỆ QUY GO THAY THẾ CHO SQL JSONB_EACH
	// -------------------------------------------------------------------------
	sampleClause := ""
	limitClause := ""
	if payload.Mode == "sample" {
		sampleClause = "TABLESAMPLE SYSTEM(1)"
		limitClause = "LIMIT 1000"
	}

	sqlQuery := fmt.Sprintf(`
		SELECT (_raw_data #> ?::text[])::text AS node_data
		FROM %s %s 
		WHERE _raw_data #> ?::text[] IS NOT NULL 
		  AND jsonb_typeof(_raw_data #> ?::text[]) != 'null'
		%s`,
		quoteCommandQualifiedTable(schemaName, targetTable), sampleClause, limitClause)

	rows, err := h.shadowDB.Raw(sqlQuery, pgPath, pgPath, pgPath).Rows()
	if err != nil {
		h.logger.Error("Lỗi stream data đệ quy", zap.Error(err))
		return
	}
	defer rows.Close()

	// Map lưu trữ: "test.test1" -> "TEXT" (chống trùng lặp và xác định type)
	discoveredFields := make(map[string]string)
	var scannedCount int64

	for rows.Next() {
		var nodeData sql.NullString
		// BẮT LỖI SCAN: Thay vì lờ đi, ta in ra xem chết ở đâu
		if err := rows.Scan(&nodeData); err != nil {
			h.logger.Error("Lỗi Scan SQL row", zap.Error(err))
			continue
		}

		if nodeData.Valid && nodeData.String != "" && nodeData.String != "{}" {
			var parsed interface{}
			// BẮT LỖI UNMARSHAL: Xem JSON có bị dị dạng không
			if err := json.Unmarshal([]byte(nodeData.String), &parsed); err != nil {
				h.logger.Error("Lỗi parse JSON", zap.Error(err), zap.String("data", nodeData.String))
				continue
			}

			scannedCount++
			if isArray, ok := parsed.([]interface{}); ok {
				for _, item := range isArray {
					flattenJSONWithTypes("", item, discoveredFields)
				}
			} else {
				flattenJSONWithTypes("", parsed, discoveredFields)
			}
		}
	}

	h.logger.Info("DEBUG - KẾT QUẢ ĐỆ QUY GO",
		zap.Int("rows_scanned_success", int(scannedCount)),
		zap.Any("discovered_fields", discoveredFields))

	// -------------------------------------------------------------------------

	var binding struct {
		ID             int64
		SourceObjectID int64
	}
	h.db.Table("cdc_system.shadow_binding").Select("id, source_object_id").
		Where("shadow_table = ?", targetTable).Limit(1).Scan(&binding)

	existing := make(map[string]bool)
	if isObject && payload.MasterBindingID != nil {
		var existingRows []struct{ TargetColumn string }
		h.db.Table("cdc_system.mapping_rule_master").Select("target_column").
			Where("master_binding_id = ?", *payload.MasterBindingID).Scan(&existingRows)
		for _, r := range existingRows {
			existing[r.TargetColumn] = true
		}
	} else {
		var existingRows []struct{ SourceField string }
		q := h.db.Table("cdc_system.mapping_rule_v2").Select("source_field")
		if payload.MasterBindingID != nil {
			q = q.Where("master_binding_id = ?", *payload.MasterBindingID)
		} else if binding.ID > 0 {
			q = q.Where("shadow_binding_id = ? AND master_binding_id IS NULL", binding.ID)
		}
		q.Scan(&existingRows)
		for _, r := range existingRows {
			existing[r.SourceField] = true
		}
	}

	skipFields := map[string]bool{
		"_id": true, "_raw_data": true, "_source": true, "_synced_at": true,
		"_source_ts": true, "_deleted": true, "_created_at": true, "_updated_at": true,
		"_parent_source_id": true, "_array_index": true, "id": true,
	}

	type CreatedRuleDTO struct {
		ID          int64  `json:"id"`
		SourceField string `json:"source_field"`
		DataType    string `json:"data_type"`
	}
	var newFields []string
	var rulesToInsert []model.MappingRuleV2
	type masterNestedRow struct{ target, srcField, srcPath, dataType string }
	var masterRows []masterNestedRow
	var newRules []CreatedRuleDTO

	// Lặp qua kết quả đệ quy từ Go thay vì rawFields của SQL
	for relPath, pgType := range discoveredFields {
		if skipFields[relPath] {
			continue
		}

		if isObject && payload.MasterBindingID != nil {
			// BIẾN ĐỔI CHUẨN XÁC VỚI RECORD BẠN CUNG CẤP:
			// basePath = "params" | relPath = "test.test1"
			// objTarget = "params_test_test1"
			// srcPath = "params.test.test1"
			objTarget := strings.ReplaceAll(basePath+"."+relPath, ".", "_")
			objTarget = strings.ReplaceAll(objTarget, "[*]", "") // Clear ký tự array nếu có trong tên cột

			if existing[objTarget] {
				continue
			}
			newFields = append(newFields, relPath)
			masterRows = append(masterRows, masterNestedRow{
				target:   objTarget,
				srcField: basePath,
				srcPath:  basePath + "." + relPath,
				dataType: pgType,
			})
		} else if binding.ID > 0 {
			if existing[relPath] {
				continue
			}
			newFields = append(newFields, relPath)
			sys := "system-scan-array"
			var srcPath *string
			if isArrayExplode {
				np := basePath + "[*]." + relPath
				srcPath = &np
			}
			rulesToInsert = append(rulesToInsert, model.MappingRuleV2{
				SourceObjectID:  binding.SourceObjectID,
				ShadowBindingID: &binding.ID,
				MasterBindingID: payload.MasterBindingID,
				SourceField:     relPath,
				TargetColumn:    strings.ReplaceAll(relPath, ".", "_"), // Tránh lỗi invalid column name
				DataType:        pgType,
				SourcePath:      srcPath,
				SourceFormat:    "raw",
				IsActive:        false,
				Status:          "pending",
				IsNullable:      true,
				CreatedBy:       &sys,
			})
		}
	}

	// THỰC THI TRANSACTION INSERT ...
	errTx := h.db.Transaction(func(tx *gorm.DB) error {
		if len(rulesToInsert) > 0 {
			if err := tx.Create(&rulesToInsert).Error; err != nil {
				return err
			}
			for _, r := range rulesToInsert {
				newRules = append(newRules, CreatedRuleDTO{ID: r.ID, SourceField: r.SourceField, DataType: r.DataType})
			}
		}

		if len(masterRows) > 0 && payload.MasterBindingID != nil {
			var parentV2ID int64
			tx.Raw(`
                SELECT v2.id FROM cdc_system.mapping_rule_v2 v2
                  JOIN cdc_system.master_binding mb ON mb.shadow_binding_id = v2.shadow_binding_id
                 WHERE mb.id = ? AND v2.target_column = ? LIMIT 1`,
				*payload.MasterBindingID, basePath).Scan(&parentV2ID)

			for _, mr := range masterRows {
				var newID int64
				insErr := tx.Raw(`
                    INSERT INTO cdc_system.mapping_rule_master
                      (master_binding_id, mapping_v2_id, target_column, source_field, source_path,
                       data_type, is_active, status, created_by, updated_by, created_at, updated_at)
                    SELECT ?, NULLIF(?, 0), ?, ?, ?, ?, true, 'pending', 'system-scan-array', 'system-scan-array', NOW(), NOW()
                    WHERE NOT EXISTS (
                      SELECT 1 FROM cdc_system.mapping_rule_master
                       WHERE master_binding_id = ? AND target_column = ?)
                    RETURNING id`,
					*payload.MasterBindingID, parentV2ID, mr.target, mr.srcField, mr.srcPath, mr.dataType,
					*payload.MasterBindingID, mr.target,
				).Scan(&newID).Error

				if insErr == nil && newID > 0 {
					newRules = append(newRules, CreatedRuleDTO{ID: newID, SourceField: mr.srcField, DataType: mr.dataType})
				}
			}
		}
		return nil
	})

	if errTx != nil {
		h.logger.Error("Lỗi insert rules đệ quy", zap.Error(errTx))
	}

	res := map[string]interface{}{
		"status":       "ok",
		"table":        targetTable,
		"explode_path": payload.ExplodePath,
		"total_keys":   len(discoveredFields),
		"new_fields":   newFields,
		"new_rules":    newRules,
		"mode":         payload.Mode,
	}

	res = sanitizeAdminResultMap(res)
	resBytes, _ := json.Marshal(res)
	h.nats_publish(msg, replySubject, resBytes)

	h.logger.Info("scan-array-recursive completed",
		zap.String("table", targetTable),
		zap.Int("discovered", len(discoveredFields)),
		zap.Int("inserted", len(newFields)))
}

func flattenJSONWithTypes(prefix string, value interface{}, result map[string]string) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenJSONWithTypes(newPrefix, val, result)
		}
	case []interface{}:
		newPrefix := prefix + "[*]"
		if len(v) == 0 {
			result[newPrefix] = "JSONB"
		} else {
			for _, item := range v {
				flattenJSONWithTypes(newPrefix, item, result)
			}
		}
	case float64:
		result[prefix] = "NUMERIC"
	case bool:
		result[prefix] = "BOOLEAN"
	case string:
		// BÍ QUYẾT Ở ĐÂY: Phát hiện Stringified JSON
		trimmed := strings.TrimSpace(v)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var parsed interface{}
			// Thử ép kiểu nó thành JSON xem có được không
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				// Thành công! Nó thực chất là JSON ẩn. Đệ quy đào sâu vào trong nó:
				flattenJSONWithTypes(prefix, parsed, result)
				return // Xong việc thì thoát, không set nó là TEXT nữa
			}
		}
		// Nếu không thể Unmarshal (không phải JSON), thì nó chỉ là string bình thường
		result[prefix] = "TEXT"
	case nil:
		if _, exists := result[prefix]; !exists {
			result[prefix] = "TEXT"
		}
	default:
		result[prefix] = "TEXT"
	}
}

// HandlePeriodicScan scans _raw_data for all active tables and auto-creates pending mapping rules.
// Subject: "cdc.cmd.periodic-scan" (pub/sub, triggered by scheduler)
func (h *CommandHandler) HandlePeriodicScan(msg *nats.Msg) {
	entries := h.listActiveTableConfigs(context.Background())
	if len(entries) == 0 {
		h.logger.Warn("periodic scan: no active table configs available")
		return
	}

	totalNew := 0
	for _, entry := range entries {
		schemaName := h.resolveTargetSchema(entry.TargetTable)
		// Skip tables that don't exist or don't have _raw_data
		if !h.tableExists(entry.TargetTable) || !h.hasColumn(entry.TargetTable, "_raw_data") {
			continue
		}

		// Scan _raw_data keys and infer type
		var rawFields []struct {
			Key  string
			Type string
		}
		sql := fmt.Sprintf(
			`SELECT key, MAX(type) as type FROM (
				SELECT key, jsonb_typeof(_raw_data->key) as type FROM %s
				WHERE _raw_data IS NOT NULL AND _raw_data != '{}'::jsonb LIMIT 1000
			) sub WHERE type != 'null' GROUP BY key`, quoteCommandQualifiedTable(schemaName, entry.TargetTable))
		if err := h.db.Raw(sql).Scan(&rawFields).Error; err != nil {
			continue
		}

		// 2. Fetch shadow_binding to know the scope
		var binding struct {
			ID             int64
			SourceObjectID int64
		}
		h.db.Table("cdc_system.shadow_binding").Select("id, source_object_id").Where("shadow_table = ?", entry.TargetTable).Limit(1).Scan(&binding)

		// Get existing rules
		var existingV2 []struct {
			SourceField string
		}
		if binding.ID > 0 {
			h.db.Table("cdc_system.mapping_rule_v2").
				Select("source_field").
				Where("shadow_binding_id = ?", binding.ID).
				Scan(&existingV2)
		} else if binding.SourceObjectID > 0 {
			h.db.Table("cdc_system.mapping_rule_v2").
				Select("source_field").
				Where("source_object_id = ?", binding.SourceObjectID).
				Scan(&existingV2)
		}

		mapped := make(map[string]bool)
		for _, r := range existingV2 {
			mapped[r.SourceField] = true
		}

		skip := map[string]bool{
			"_id": true, "_raw_data": true, "_source": true, "_synced_at": true,
			"_version": true, "_hash": true, "_deleted": true, "_created_at": true, "_updated_at": true,
			"id": true, "_debezium_ab_id": true, "_debezium_emitted_at": true, "_debezium_extracted_at": true, "_debezium_meta": true,
		}

		for _, f := range rawFields {
			key := f.Key
			if mapped[key] || skip[key] {
				continue
			}

			pgType := "TEXT"
			switch f.Type {
			case "number":
				pgType = "NUMERIC"
			case "boolean":
				pgType = "BOOLEAN"
			case "object", "array":
				pgType = "JSONB"
			}

			if binding.ID > 0 {
				sourceType := f.Type
				sys := "system-scan"
				rule := model.MappingRuleV2{
					SourceObjectID:  binding.SourceObjectID,
					ShadowBindingID: &binding.ID,
					SourceField:     key,
					TargetColumn:    key,
					DataType:        pgType,
					SourceDataType:  &sourceType,
					SourceFormat:    "raw",
					IsActive:        false,
					Status:          "pending",
					IsNullable:      true,
					CreatedBy:       &sys,
				}
				if err := h.db.Create(&rule).Error; err == nil {
					totalNew++
				}
			}
		}
	}

	h.logger.Info("periodic scan completed", zap.Int("new_rules_created", totalNew), zap.Int("tables_scanned", len(entries)))
}

// buildCastExpr builds a PostgreSQL expression to extract a typed value from _raw_data JSONB.
// JSONB/JSON targets must use the `->` operator (returns jsonb) rather than `->>` (returns text);
// otherwise UPDATE fails with 42804 "expression is of type text" against a jsonb column.
//
// MongoDB BSON values arrive in JSONB in several shapes depending on which encoder wrote them:
//   - Number / string scalars: plain JSON primitives.
//   - BSON Date: either a raw epoch-ms number (Go-driver default), or an Extended-JSON object
//     `{"$date": "ISO8601"}` / `{"$date": {"$numberLong": "epoch-ms"}}` (canonical encoder).
//   - BSON Int64: either a raw number, or `{"$numberLong": "123"}` (canonical encoder).
//
// Without handling the Extended-JSON object forms, cmd-batch-transform on older rows blows up
// with 22007 "invalid input syntax for type timestamp: {\"$date\": ...}".
func buildCastExpr(field, dataType string) string {
	switch strings.ToLower(dataType) {
	case "jsonb":
		return fmt.Sprintf("(_raw_data->'%s')", field)
	case "json":
		return fmt.Sprintf("((_raw_data->'%s')::JSON)", field)
	}
	base := fmt.Sprintf("(_raw_data->>'%s')", field)
	switch strings.ToLower(dataType) {
	case "integer", "int", "int4", "smallint":
		return fmt.Sprintf("(%s)::INTEGER", base)
	case "int8", "bigint":
		return fmt.Sprintf(
			"(CASE "+
				"WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
				"AND jsonb_typeof(_raw_data->'%[1]s'->'$numberLong') = 'string' "+
				"THEN (_raw_data->'%[1]s'->>'$numberLong')::BIGINT "+
				"ELSE (_raw_data->>'%[1]s')::BIGINT "+
				"END)",
			field)
	case "numeric", "decimal", "float", "float8", "double precision":
		return fmt.Sprintf("(%s)::NUMERIC", base)
	case "boolean", "bool":
		return fmt.Sprintf("(%s)::BOOLEAN", base)
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return fmt.Sprintf(
			"(CASE "+
				"WHEN jsonb_typeof(_raw_data->'%[1]s') = 'number' "+
				"THEN to_timestamp((_raw_data->>'%[1]s')::BIGINT / 1000.0) AT TIME ZONE 'UTC' "+
				"WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
				"AND jsonb_typeof(_raw_data->'%[1]s'->'$date') = 'string' "+
				"THEN (_raw_data->'%[1]s'->>'$date')::TIMESTAMPTZ "+
				"WHEN jsonb_typeof(_raw_data->'%[1]s') = 'object' "+
				"AND jsonb_typeof(_raw_data->'%[1]s'->'$date') = 'object' "+
				"AND jsonb_typeof(_raw_data->'%[1]s'->'$date'->'$numberLong') = 'string' "+
				"THEN to_timestamp((_raw_data->'%[1]s'->'$date'->>'$numberLong')::BIGINT / 1000.0) AT TIME ZONE 'UTC' "+
				"ELSE (_raw_data->>'%[1]s')::TIMESTAMP "+
				"END)",
			field)
	default:
		return fmt.Sprintf("(%s)::TEXT", base)
	}
}

// publishResult publishes a sanitized command result to reply-to (if
// present), then writes the same sanitized payload into ActivityLog.
func (h *CommandHandler) publishResult(msg *nats.Msg, result CommandResult) {
	safeResult := result
	safeResult.Error = sanitizeAdminError(result.Error)
	data, _ := json.Marshal(safeResult)
	h.nats_publish(msg, "cdc.result."+result.Command, data)
	h.logCommandResult(safeResult)

	// Activity Log — every command result
	now := time.Now()
	var errPtr *string
	if safeResult.Error != "" {
		e := safeResult.Error
		errPtr = &e
	}
	h.db.Create(&model.ActivityLog{
		Operation:    "cmd-" + safeResult.Command,
		TargetTable:  safeResult.TargetTable,
		Status:       safeResult.Status,
		RowsAffected: int64(safeResult.RowsAffected),
		ErrorMessage: errPtr,
		Details:      data,
		TriggeredBy:  activity.TriggeredByNATSCommand.String(),
		StartedAt:    now,
		CompletedAt:  &now,
	})
}

// HandleDropGINIndex drops the GIN index on _raw_data after a table is fully transformed.
// This reclaims significant storage (GIN indexes on JSONB are large).
// Subject: "cdc.cmd.drop-gin-index"
func (h *CommandHandler) HandleDropGINIndex(msg *nats.Msg) {
	targetTable := string(msg.Data)
	h.logger.Info("checking GIN index cleanup eligibility", zap.String("table", targetTable))
	schemaName := h.resolveTargetSchema(targetTable)

	if !h.tableExists(targetTable) {
		h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "skipped", Error: "table does not exist"})
		return
	}

	// 1. Check if table is fully transformed (no pending rows)
	var pendingRows int64
	reg := h.resolveTargetTableConfig(context.Background(), targetTable)
	if reg == nil {
		h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "error", Error: "registry entry not found"})
		return
	}

	sourceTable := reg.SourceTable
	rules, _ := h.mappingRepo.GetByTable(context.Background(), sourceTable)
	if len(rules) == 0 {
		h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "error", Error: "no mapping rules"})
		return
	}

	// Find first active mapped column as proxy
	var firstCol string
	for _, r := range rules {
		if r.IsActive {
			firstCol = r.TargetColumn
			break
		}
	}
	if firstCol == "" {
		h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "error", Error: "no active mapping rules"})
		return
	}

	h.db.Raw(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE _raw_data IS NOT NULL AND %s IS NULL`,
		quoteCommandQualifiedTable(schemaName, targetTable),
		quoteCommandIdent(firstCol),
	)).Scan(&pendingRows)
	if pendingRows > 0 {
		h.publishResult(msg, CommandResult{
			Command:     "drop-gin-index",
			TargetTable: targetTable,
			Status:      "skipped",
			Error:       fmt.Sprintf("%d rows still pending transform", pendingRows),
		})
		return
	}

	// 2. Drop GIN index
	indexName := "idx_" + targetTable + "_raw"
	result := h.db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %s.%s`, quoteCommandIdent(schemaName), quoteCommandIdent(indexName)))
	if result.Error != nil {
		h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "error", Error: result.Error.Error()})
		return
	}

	h.logger.Info("GIN index dropped", zap.String("table", targetTable), zap.String("index", indexName))
	h.publishResult(msg, CommandResult{Command: "drop-gin-index", TargetTable: targetTable, Status: "success"})
}

// ============================================================================
// Boundary-refactor handlers (workspace feature-cdc-integration).
// Each handler owns an ex-CMS operation so the CMS API stays thin-proxy.
// Subject catalogue:
//   1. cdc.cmd.scan-fields             -> HandleScanFields
//   2. cdc.cmd.scan-source             -> HandleScanSource
//   3. cdc.cmd.refresh-catalog         -> HandleRefreshCatalog
//   5. cdc.cmd.sync-register           -> HandleSyncRegister
//   6. cdc.cmd.sync-state              -> HandleSyncState
//   7. cdc.cmd.restart-debezium        -> HandleRestartDebezium
//   8. cdc.cmd.alter-column            -> HandleAlterColumn
//   9. cdc.cmd.import-streams          -> HandleImportStreams
// Result subjects mirror cdc.result.<subject-tail>.
// ============================================================================

// systemFieldSet — same list used by the legacy CMS ScanFields flow.
func systemFieldSet() map[string]bool {
	return map[string]bool{
		"_raw_data": true, "_source": true, "_created_at": true,
		"_updated_at": true, "_deleted": true, "_hash": true,
		"_synced_at": true, "_version": true,
		"_debezium_raw_id": true, "_debezium_extracted_at": true, "_debezium_meta": true,
		"_debezium_ab_id": true, "_debezium_emitted_at": true, "_debezium_generation_id": true,
	}
}

// migrated logic stays byte-identical (Rule #4 no-improvement port).
func inferSQLTypeFromLegacyCatalogProp(prop interface{}) string {
	propMap, ok := prop.(map[string]interface{})
	if !ok {
		return "TEXT"
	}
	legacyType, _ := propMap["type"].(string)
	if types, ok := propMap["type"].([]interface{}); ok && len(types) > 0 {
		for _, t := range types {
			if tStr, ok := t.(string); ok && tStr != "null" {
				legacyType = tStr
				break
			}
		}
	}
	switch legacyType {
	case "integer":
		return "BIGINT"
	case "number":
		return "NUMERIC"
	case "boolean":
		return "BOOLEAN"
	case "array", "object":
		return "JSONB"
	default:
		if format, _ := propMap["format"].(string); format == "date-time" {
			return "TIMESTAMP"
		}
		return "TEXT"
	}
}

// scanFieldsDebezium samples _raw_data JSONB (last 100 rows) to infer
// new fields for Debezium-backed tables.
// autoApprove=true will set discovered fields to 'approved' status immediately.
func (h *CommandHandler) scanFieldsDebezium(ctx context.Context, registryID uint, shadowBindingID int64, targetTable, sourceTable, sourceType string, autoApprove bool) (int, int, error) {
	v2ObjectID := int64(registryID)
	// Heuristic: If registryID > 0 but no V2 object found, try to resolve via legacy locator
	if h.mappingV2Repo != nil && v2ObjectID > 0 {
		_, err := h.mappingV2Repo.ListBySourceObject(ctx, v2ObjectID)
		if err != nil {
			// Try lookup via legacy locator
			var soID int64
			err := h.db.Raw(`SELECT id FROM cdc_system.source_object_registry WHERE source_locator_json->>'legacy_registry_id' = ?`, strconv.FormatUint(uint64(registryID), 10)).Scan(&soID).Error
			if err == nil && soID > 0 {
				v2ObjectID = soID
			}
		}
	}

	schemaName := h.resolveTargetSchema(targetTable)
	if !h.tableExists(targetTable) || !h.hasColumn(targetTable, "_raw_data") {
		// Fallback for MongoDB: if shadow doesn't exist yet, scan source directly
		if sourceType == "mongodb" {
			zap.S().Infof("Shadow table %s does not exist, falling back to direct MongoDB source scanning (v2ID=%d)", targetTable, v2ObjectID)
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		return 0, 0, fmt.Errorf("table %s has no _raw_data column in shadow db", targetTable)
	}
	type sampleRow struct {
		Raw json.RawMessage `gorm:"column:_raw_data"`
	}
	var rows []sampleRow
	sql := fmt.Sprintf(`SELECT _raw_data FROM %s WHERE _raw_data IS NOT NULL AND _raw_data != '{}'::jsonb ORDER BY _synced_at DESC LIMIT 100`, quoteCommandQualifiedTable(schemaName, targetTable))
	if err := h.shadowDB.WithContext(ctx).Raw(sql).Scan(&rows).Error; err != nil {
		return 0, 0, fmt.Errorf("sample raw_data: %w", err)
	}

	if len(rows) == 0 {
		// Fallback for MongoDB: if shadow is empty, try to scan source directly
		if sourceType == "mongodb" {
			zap.S().Infof("Shadow table %s is empty, falling back to direct MongoDB source scanning (v2ID=%d)", targetTable, v2ObjectID)
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		return 0, 0, fmt.Errorf("shadow table %s is empty; wait for Debezium to sync some data before scanning fields", targetTable)
	}

	rawJSONs := make([]string, len(rows))
	for i, r := range rows {
		rawJSONs[i] = string(r.Raw)
	}

	return h.processDiscoveryRows(ctx, v2ObjectID, shadowBindingID, sourceTable, rawJSONs, autoApprove)
}

// HandleScanFields implements boundary-refactor #1 (subject
// Debezium _raw_data sampling based on sync_engine.
func (h *CommandHandler) HandleScanFields(msg *nats.Msg) {
	var payload struct {
		RegistryID      uint   `json:"registry_id"`
		SourceObjectID  int64  `json:"source_object_id"`
		ShadowBindingID int64  `json:"shadow_binding_id"`
		TargetTable     string `json:"target_table"`
		SourceTable     string `json:"source_table"`
		SyncEngine      string `json:"sync_engine"`
		SourceType      string `json:"source_type"`
		LegacySourceID  string `json:"legacy_source_id"`
		ReplyTo         string `json:"reply_to"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Error("cdc.cmd.scan-fields: invalid payload", zap.Error(err))
		h.publishResultWithSubject(msg, "cdc.result.scan-fields", CommandResult{Command: "scan-fields", Status: "error", Error: "invalid payload"})
		return
	}

	// Resolve effective registry ID: V2 source_object_id takes priority over legacy registry_id
	effectiveID := payload.RegistryID
	if payload.SourceObjectID > 0 {
		effectiveID = uint(payload.SourceObjectID)
	}

	ctx := context.Background()
	engine := strings.ToLower(strings.TrimSpace(payload.SyncEngine))
	h.logger.Info("scan-fields dispatch",
		zap.String("target", payload.TargetTable),
		zap.String("source", payload.SourceTable),
		zap.String("engine", engine),
		zap.Uint("effective_id", effectiveID),
		zap.Int64("source_object_id", payload.SourceObjectID),
		zap.Int64("shadow_binding_id", payload.ShadowBindingID),
	)

	var added, total int
	var sourceUsed string = "debezium"
	var err error
	_ = engine
	added, total, err = h.scanFieldsDebezium(ctx, effectiveID, payload.ShadowBindingID, payload.TargetTable, payload.SourceTable, payload.SourceType, false)

	if err != nil {
		h.publishResultWithSubject(msg, "cdc.result.scan-fields", CommandResult{
			Command: "scan-fields", RegistryID: payload.RegistryID, TargetTable: payload.TargetTable,
			Status: "error", Error: err.Error(),
		})
		return
	}

	// Detailed result event (Activity Log is hit by publishResult too).
	result := map[string]interface{}{
		"command":      "scan-fields",
		"registry_id":  payload.RegistryID,
		"target_table": payload.TargetTable,
		"added":        added,
		"total":        total,
		"source_used":  sourceUsed,
		"status":       "success",
	}
	result = sanitizeAdminResultMap(result)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, "cdc.result.scan-fields", data)
	h.writeActivity("scan-fields", payload.TargetTable, "success", int64(added), result, "")
}

// HandleScanSource implements boundary-refactor #2 (subject
// missing registry rows (is_active=false, matching legacy CMS logic).

// HandleRefreshCatalog implements boundary-refactor #3 (subject
// drift into schema_changes_log when the catalogue changes.

// HandleSyncRegister implements boundary-refactor #5 (subject
// cdc.cmd.sync-register). Routes the "sync registry entry with sync
func (h *CommandHandler) HandleSyncRegister(msg *nats.Msg) {
	var payload struct {
		RegistryID  uint   `json:"registry_id"`
		TargetTable string `json:"target_table"`
		SourceTable string `json:"source_table"`
		SourceType  string `json:"source_type"`
		SyncEngine  string `json:"sync_engine"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.publishResultWithSubject(msg, "cdc.result.sync-register", CommandResult{Command: "sync-register", Status: "error", Error: "invalid payload"})
		return
	}
	ctx := context.Background()
	_ = payload.SyncEngine
	debeziumOK := false
	var debeziumErr string
	if err := h.verifyDebeziumConnector(ctx); err != nil {
		debeziumErr = err.Error()
	} else {
		debeziumOK = true
	}

	status := "success"
	if !debeziumOK {
		status = "error"
	}
	result := map[string]interface{}{
		"command":        "sync-register",
		"registry_id":    payload.RegistryID,
		"target_table":   payload.TargetTable,
		"sync_engine":    "debezium",
		"debezium_ok":    debeziumOK,
		"debezium_error": debeziumErr,
		"status":         status,
	}
	result = sanitizeAdminResultMap(result)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, "cdc.result.sync-register", data)
	h.writeActivity("sync-register", payload.TargetTable, status, 0, result, "")
}

// path. Keeps the stream config minimal (selected=true).

// verifyDebeziumConnector probes Kafka Connect for the configured
// connector. Returns nil on 2xx, an error otherwise. Circuit-breaker:
// 10s timeout + single retry (per boundary refactor Rule #3).
func (h *CommandHandler) verifyDebeziumConnector(ctx context.Context) error {
	if h.kafkaConnectURL == "" {
		return fmt.Errorf("kafka connect url not configured")
	}
	// The actual connector name is injected by worker_server via config.
	// We hit the /connectors endpoint which returns a JSON list — any
	// 2xx means the control plane is reachable.
	url := strings.TrimRight(h.kafkaConnectURL, "/") + "/connectors"
	return h.connectGET(ctx, url)
}

// HandleSyncState implements boundary-refactor #6 (subject
// pauses/resumes the Debezium connector via Kafka Connect REST.
func (h *CommandHandler) HandleSyncState(msg *nats.Msg) {
	var payload struct {
		RegistryID uint   `json:"registry_id"`
		Action     string `json:"action"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.publishResultWithSubject(msg, "cdc.result.sync-state", CommandResult{Command: "sync-state", Status: "error", Error: "invalid payload"})
		return
	}
	action := strings.ToLower(payload.Action)
	if action != "activate" && action != "deactivate" {
		h.publishResultWithSubject(msg, "cdc.result.sync-state", CommandResult{Command: "sync-state", Status: "error", Error: "action must be activate|deactivate"})
		return
	}
	ctx := context.Background()
	entry := h.resolveTableConfigByID(ctx, payload.RegistryID)
	if entry == nil {
		h.publishResultWithSubject(msg, "cdc.result.sync-state", CommandResult{Command: "sync-state", Status: "error", Error: "registry not found"})
		return
	}

	debeziumStatus := "skipped"
	var firstErr string

	// Debezium path — pause / resume the connector.
	if service.ShouldUseDebezium(entry) && h.kafkaConnectURL != "" {
		connector := h.detectConnectorName(entry)
		if connector == "" {
			debeziumStatus = "error"
			firstErr = fmt.Sprintf("debezium: cannot resolve connector for source_db=%q source_table=%q", entry.SourceDB, entry.SourceTable)
		} else {
			verb := "pause"
			if action == "activate" {
				verb = "resume"
			}
			url := fmt.Sprintf("%s/connectors/%s/%s", strings.TrimRight(h.kafkaConnectURL, "/"), connector, verb)
			if err := h.connectPUT(ctx, url); err != nil {
				debeziumStatus = "error"
				if firstErr == "" {
					firstErr = "debezium: " + err.Error()
				}
			} else {
				debeziumStatus = "ok"
			}
		}
	}

	status := "success"
	if firstErr != "" && debeziumStatus != "ok" {
		status = "error"
	}
	result := map[string]interface{}{
		"command":         "sync-state",
		"registry_id":     payload.RegistryID,
		"target_table":    entry.TargetTable,
		"action":          action,
		"debezium_status": debeziumStatus,
		"error":           firstErr,
		"status":          status,
	}
	result = sanitizeAdminResultMap(result)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, "cdc.result.sync-state", data)
	h.writeActivity("sync-state", entry.TargetTable, status, 0, result, firstErr)
}

// HandleRestartDebezium implements boundary-refactor #7 (subject
// cdc.cmd.restart-debezium). Calls Kafka Connect REST restart endpoint.
func (h *CommandHandler) HandleRestartDebezium(msg *nats.Msg) {
	var payload struct {
		ConnectorName string `json:"connector_name"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.publishResultWithSubject(msg, "cdc.result.restart-debezium", CommandResult{Command: "restart-debezium", Status: "error", Error: "invalid payload"})
		return
	}
	if h.kafkaConnectURL == "" {
		h.publishResultWithSubject(msg, "cdc.result.restart-debezium", CommandResult{Command: "restart-debezium", Status: "error", Error: "kafka_connect_url not configured"})
		return
	}
	connector := strings.TrimSpace(payload.ConnectorName)
	if connector == "" {
		h.publishResultWithSubject(msg, "cdc.result.restart-debezium", CommandResult{Command: "restart-debezium", Status: "error", Error: "connector_name is required (connectors are dynamically registered, no default available)"})
		return
	}
	url := fmt.Sprintf("%s/connectors/%s/restart?includeTasks=true&onlyFailed=false",
		strings.TrimRight(h.kafkaConnectURL, "/"), connector)
	if err := h.connectPOST(context.Background(), url); err != nil {
		h.publishResultWithSubject(msg, "cdc.result.restart-debezium", CommandResult{Command: "restart-debezium", Status: "error", Error: err.Error()})
		h.writeActivity("restart-debezium", connector, "error", 0, nil, err.Error())
		return
	}
	result := map[string]interface{}{
		"command":        "restart-debezium",
		"connector_name": connector,
		"status":         "success",
	}
	result = sanitizeAdminResultMap(result)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, "cdc.result.restart-debezium", data)
	h.writeActivity("restart-debezium", connector, "success", 0, result, "")
}

// HandleAlterColumn implements boundary-refactor #8 (subject
// cdc.cmd.alter-column). Executes ALTER TABLE ... ADD|DROP|ALTER COLUMN.
// Identifiers are validated (whitelist of [A-Za-z0-9_-]) before being
// interpolated (Rule #3 security gate — Postgres does not allow bound
// identifiers).
func (h *CommandHandler) HandleAlterColumn(msg *nats.Msg) {
	var payload struct {
		TargetSchema string `json:"target_schema"`
		TargetTable  string `json:"target_table"`
		ColumnName   string `json:"column_name"`
		DataType     string `json:"data_type"`
		Action       string `json:"action"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", Status: "error", Error: "invalid payload"})
		return
	}
	if !isSafeIdent(payload.TargetTable) || !isSafeIdent(payload.ColumnName) {
		h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: "invalid identifier"})
		return
	}
	if payload.TargetSchema != "" && !isSafeIdent(payload.TargetSchema) {
		h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: "invalid schema identifier"})
		return
	}
	// Schema-qualify when provided so DDL does not depend on session
	// search_path (lesson 2026-04-28 / 2026-05-11).
	qualifiedTable := fmt.Sprintf(`"%s"`, payload.TargetTable)
	if payload.TargetSchema != "" {
		qualifiedTable = fmt.Sprintf(`"%s"."%s"`, payload.TargetSchema, payload.TargetTable)
	}
	var sql string
	switch strings.ToLower(payload.Action) {
	case "add":
		if !isSafeType(payload.DataType) {
			h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: "invalid data_type"})
			return
		}
		sql = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "%s" %s`, qualifiedTable, payload.ColumnName, payload.DataType)
	case "drop":
		sql = fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS "%s"`, qualifiedTable, payload.ColumnName)
	case "alter_type":
		if !isSafeType(payload.DataType) {
			h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: "invalid data_type"})
			return
		}
		sql = fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN "%s" TYPE %s USING "%s"::%s`,
			qualifiedTable, payload.ColumnName, payload.DataType, payload.ColumnName, payload.DataType)
	default:
		h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: "action must be add|drop|alter_type"})
		return
	}
	// Shadow tables live in the shadow plane (gpay-postgres-shadow:5436
	// cdc_shadow), not the destination plane. Other handlers in this file
	// already use h.shadowDB for shadow DDL (see line 150).
	execDB := h.db
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}
	if err := execDB.Exec(sql).Error; err != nil {
		h.publishResultWithSubject(msg, "cdc.result.alter-column", CommandResult{Command: "alter-column", TargetTable: payload.TargetTable, Status: "error", Error: err.Error()})
		h.writeActivity("alter-column", payload.TargetTable, "error", 0, nil, err.Error())
		return
	}
	result := map[string]interface{}{
		"command":      "alter-column",
		"target_table": payload.TargetTable,
		"column_name":  payload.ColumnName,
		"action":       payload.Action,
		"status":       "success",
	}
	result = sanitizeAdminResultMap(result)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, "cdc.result.alter-column", data)
	h.writeActivity("alter-column", payload.TargetTable, "success", 0, result, "")
}

// HandleImportStreams implements boundary-refactor #9 (subject
// cdc.cmd.import-streams). Inserts registry rows, calls
// create_cdc_table() on DW, and seeds default mapping rules per stream.

// workspace and imports missing registry rows + default mapping rules.
// Heavy operation — logs once per connection.

// ---------------------------------------------------------------------------
// Helpers used across boundary-refactor handlers.
// ---------------------------------------------------------------------------

// nats_publish emits a sanitized result event to the configured subject
// when the caller used pub/sub; falls back to request-reply otherwise.
func (h *CommandHandler) nats_publish(msg *nats.Msg, subject string, data []byte) {
	var payload struct {
		ReplyTo string `json:"reply_to"`
	}
	_ = json.Unmarshal(msg.Data, &payload)

	if payload.ReplyTo != "" && h.natsConn != nil {
		if err := h.natsConn.Publish(payload.ReplyTo, data); err != nil {
			h.logger.Warn("publish result failed (reply_to)", zap.String("subject", payload.ReplyTo), zap.Error(err))
		}
		return
	}

	if msg.Reply != "" {
		_ = msg.Respond(data)
		return
	}
	if h.natsConn != nil {
		if err := h.natsConn.Publish(subject, data); err != nil {
			h.logger.Warn("publish result failed", zap.String("subject", subject), zap.Error(err))
		}
		return
	}
	var result CommandResult
	if err := json.Unmarshal(data, &result); err == nil {
		h.logCommandResult(result, zap.String("subject", subject))
		return
	}
	h.logger.Info("command result", zap.String("subject", subject), zap.Int("bytes", len(data)))
}

// publishResultWithSubject is the error-path variant; it keeps
// ActivityLog and outbound result subjects aligned on the same
// sanitized payload shape.
func (h *CommandHandler) publishResultWithSubject(msg *nats.Msg, subject string, result CommandResult) {
	result.Error = sanitizeAdminError(result.Error)
	data, _ := json.Marshal(result)
	h.nats_publish(msg, subject, data)
	h.logger.Error("command failed", zap.String("command", result.Command), zap.String("error", result.Error))
	h.writeActivity(result.Command, result.TargetTable, result.Status, int64(result.RowsAffected), nil, result.Error)
}

// writeActivity mirrors publishResult's ActivityLog side-effect in a
// reusable form and never stores unsanitized admin-facing details.
func (h *CommandHandler) writeActivity(op, table, status string, rows int64, details map[string]interface{}, errMsg string) {
	now := time.Now()
	var errPtr *string
	if errMsg = sanitizeAdminError(errMsg); errMsg != "" {
		e := errMsg
		errPtr = &e
	}
	var detailsJSON []byte
	if details != nil {
		details = sanitizeAdminResultMap(details)
		detailsJSON, _ = json.Marshal(details)
	}
	h.db.Create(&model.ActivityLog{
		Operation:    op,
		TargetTable:  table,
		Status:       status,
		RowsAffected: rows,
		ErrorMessage: errPtr,
		Details:      detailsJSON,
		TriggeredBy:  activity.TriggeredByNATSCommand.String(),
		StartedAt:    now,
		CompletedAt:  &now,
	})
}

// isSafeIdent allows [A-Za-z0-9_-] only — no quoting escape surface.
func isSafeIdent(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// isSafeType restricts the DDL TYPE slot to a small allowlist to keep
// HandleAlterColumn safe.
func isSafeType(t string) bool {
	u := strings.ToUpper(strings.TrimSpace(t))
	switch u {
	case "TEXT", "VARCHAR", "VARCHAR(255)", "BIGINT", "INTEGER", "SMALLINT",
		"NUMERIC", "DECIMAL", "REAL", "DOUBLE PRECISION", "BOOLEAN",
		"TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMPTZ",
		"DATE", "TIME", "JSONB", "JSON", "UUID":
		return true
	}
	return false
}

// detectConnectorName resolves the Debezium connector name dynamically
// from connection_registry.connection_code via source_object_registry.
// Returns empty string nếu không tra được — caller phải xử lý fallback/error.
func (h *CommandHandler) detectConnectorName(entry *model.TableRegistry) string {
	if entry == nil || h.db == nil {
		return ""
	}
	return service.ResolveConnectorNameBySource(context.Background(), h.db, entry.SourceDB, entry.SourceTable)
}

// connectGET issues a short-timeout GET with a single retry.
func (h *CommandHandler) connectGET(ctx context.Context, url string) error {
	return h.connectCall(ctx, http.MethodGet, url, nil)
}

func (h *CommandHandler) connectPOST(ctx context.Context, url string) error {
	return h.connectCall(ctx, http.MethodPost, url, nil)
}

func (h *CommandHandler) connectPUT(ctx context.Context, url string) error {
	return h.connectCall(ctx, http.MethodPut, url, nil)
}

// connectCall centralises Kafka Connect REST calls — 10s timeout, 1
// retry on transport/5xx error (Rule #3 circuit-breaker lite).
func (h *CommandHandler) connectCall(ctx context.Context, method, url string, body []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("kafka-connect %d: %s", resp.StatusCode, string(respBody))
			continue
		}
		return fmt.Errorf("kafka-connect %d: %s", resp.StatusCode, string(respBody))
	}
	return lastErr
}

func (h *CommandHandler) logCommandResult(result CommandResult, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("command", result.Command),
		zap.String("target_table", result.TargetTable),
		zap.String("status", result.Status),
		zap.Int("rows_affected", result.RowsAffected),
	}
	if result.RegistryID > 0 {
		baseFields = append(baseFields, zap.Uint("registry_id", result.RegistryID))
	}
	if result.Error != "" {
		baseFields = append(baseFields, zap.String("error", sanitizeAdminError(result.Error)))
	}
	baseFields = append(baseFields, fields...)
	h.logger.Info("command result", baseFields...)
}

func sanitizeAdminError(errMsg string) string {
	return service.SanitizeFreeformText(errMsg, 240)
}

func sanitizeAdminResultMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return map[string]interface{}{}
	}
	allowed := map[string]struct{}{
		"command": {}, "registry_id": {}, "target_table": {}, "table": {}, "rows_affected": {},
		"status": {}, "error": {}, "reason": {}, "added": {}, "total": {},
		"source_used": {}, "source_table": {}, "mapped_count": {}, "new_fields": {},
		"total_raw_keys": {}, "sync_engine": {}, "debezium_ok": {}, "debezium_error": {},
		"debezium_status": {}, "action": {}, "connector_name": {}, "column_name": {},
		"total_keys": {}, "mode": {}, "note": {}, "explode_path": {},
		"new_rules": {}, // CreatedRuleDTO[] (id/source_field/data_type) — FE review modal cần
	}
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		if _, ok := allowed[key]; !ok {
			continue
		}
		switch key {
		case "error", "reason", "debezium_error":
			out[key] = sanitizeAdminError(fmt.Sprintf("%v", value))
		case "new_fields":
			out[key] = sanitizeAdminFields(value)
		default:
			out[key] = value
		}
	}
	return out
}

func sanitizeAdminFields(value interface{}) []string {
	list, ok := value.([]string)
	if ok {
		out := append([]string(nil), list...)
		sort.Strings(out)
		return out
	}
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func (h *CommandHandler) resolveTargetTableConfig(ctx context.Context, targetTable string) *model.TableRegistry {
	if h.metadata != nil {
		if item := h.metadata.GetTableConfig(targetTable); item != nil {
			return item
		}
	}
	if h.registryRepo == nil {
		return nil
	}
	item, err := h.registryRepo.GetByTargetTable(ctx, targetTable)
	if err != nil {
		return nil
	}
	return item
}

func (h *CommandHandler) resolveTargetRoute(targetTable string) *service.ResolvedSourceRoute {
	if h.metadata == nil {
		return nil
	}
	return h.metadata.ResolveTargetRoute(targetTable)
}

// normalizeMappingRuleDataType maps raw PG information_schema.data_type
// values into the canonical uppercase form accepted by the
// `mapping_rules_data_type_chk` CHECK constraint. Anything outside the
// safe-list lands as TEXT (lossless fallback that always passes the
// regex). Caller has already filtered `_*` cdc meta cols.
func normalizeMappingRuleDataType(dt string) string {
	switch strings.ToLower(strings.TrimSpace(dt)) {
	case "smallint", "int2":
		return "SMALLINT"
	case "integer", "int", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "real", "float4":
		return "REAL"
	case "double precision", "float8":
		return "DOUBLE PRECISION"
	case "boolean", "bool":
		return "BOOLEAN"
	case "date":
		return "DATE"
	case "time", "time without time zone", "time with time zone":
		return "TIME"
	case "timestamp", "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamptz", "timestamp with time zone":
		return "TIMESTAMPTZ"
	case "interval":
		return "INTERVAL"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	case "uuid":
		return "UUID"
	case "inet":
		return "INET"
	case "cidr":
		return "CIDR"
	case "macaddr":
		return "MACADDR"
	case "bytea":
		return "BYTEA"
	case "text", "character varying", "varchar", "character", "char":
		// information_schema strips length for CHARACTER VARYING — we
		// can't recover (P,N) shapes without querying
		// character_maximum_length. TEXT is a safe upcast for both.
		return "TEXT"
	case "numeric", "decimal":
		// No precision available from raw lookup — store as TEXT to
		// pass the constraint (NUMERIC(P,S) is required by the regex).
		return "TEXT"
	default:
		return "TEXT"
	}
}

func (h *CommandHandler) resolveTargetSchema(targetTable string) string {
	route := h.resolveTargetRoute(targetTable)
	if route != nil && route.ShadowBinding != nil {
		if v := strings.TrimSpace(route.ShadowBinding.ShadowSchema); v != "" {
			return v
		}
	}
	return "public"
}

func (h *CommandHandler) resolveTableConfigByID(ctx context.Context, id uint) *model.TableRegistry {
	if h.metadata != nil {
		if item := h.metadata.GetTableConfigByID(id); item != nil {
			return item
		}
	}
	if h.registryRepo == nil {
		return nil
	}
	item, err := h.registryRepo.GetByID(ctx, id)
	if err != nil {
		return nil
	}
	return item
}

func (h *CommandHandler) listActiveTableConfigs(ctx context.Context) []model.TableRegistry {
	if h.metadata != nil {
		items := h.metadata.ListTableConfigs()
		if len(items) > 0 {
			return items
		}
	}
	if h.registryRepo == nil {
		return nil
	}
	items, err := h.registryRepo.GetAllActive(ctx)
	if err != nil {
		return nil
	}
	return items
}

// SanitizeAdminErrorForTest exposes sanitizeAdminError for the out-of-package test suite.
func SanitizeAdminErrorForTest(errMsg string) string { return sanitizeAdminError(errMsg) }

// SanitizeAdminResultMapForTest exposes sanitizeAdminResultMap for the out-of-package test suite.
func SanitizeAdminResultMapForTest(input map[string]interface{}) map[string]interface{} {
	return sanitizeAdminResultMap(input)
}

// BuildCastExprForTest exposes buildCastExpr for the out-of-package test suite.
func BuildCastExprForTest(field, dataType string) string { return buildCastExpr(field, dataType) }

// PublishResultWithSubjectForTest exposes CommandHandler.publishResultWithSubject for integration tests.
func (h *CommandHandler) PublishResultWithSubjectForTest(msg *nats.Msg, subject string, result CommandResult) {
	h.publishResultWithSubject(msg, subject, result)
}

// WriteActivityForTest exposes CommandHandler.writeActivity for integration tests.
func (h *CommandHandler) WriteActivityForTest(op, table, status string, rows int64, details map[string]interface{}, errMsg string) {
	h.writeActivity(op, table, status, rows, details, errMsg)
}
