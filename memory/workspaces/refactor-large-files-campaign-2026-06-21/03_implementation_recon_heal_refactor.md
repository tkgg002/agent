# Giải pháp kỹ thuật chi tiết: Phân tách recon_heal.go

Tài liệu này chứa mã nguồn chi tiết cho các file helper mới được phân tách từ `recon_heal.go`.

---

## 1. `recon_heal_models.go`
```go
package recon

import "time"

// HealResult captures the outcome of a single HealMissingIDs call.
type HealResult struct {
	Table         string
	Requested     int
	Upserted      int
	Deleted       int
	Skipped       int // OCC guard rejected, or doc missing in source
	Errored       int
	UsedSignal    bool // true when Debezium signal took the lead path
	DurationMs    int64
	SignalID      string
	AuditFlushCnt int
	RunID         string // heal run correlation id — echoes into audit rows
}

// ReconHealerConfig exposes the handful of tunables we need. All
// optional.
type ReconHealerConfig struct {
	BatchSize      int           // default 500 (plan v3 §6)
	AuditFlushSize int           // default 100 (plan v3 §6)
	QueryTimeout   time.Duration // per-chunk ctx timeout, default 60s
	// ForceDirect — when true, skip the Signal path even when a healthy
	// connector is available. Useful for tests and targeted re-heal.
	ForceDirect bool
	// SensitiveFieldMask provides fallback default keywords only when a
	// shared MaskingService instance is not injected by the caller.
	SensitiveFieldMask []string
}

func (c *ReconHealerConfig) applyDefaults() {
	if c.BatchSize <= 0 {
		c.BatchSize = 500
	}
	if c.AuditFlushSize <= 0 {
		c.AuditFlushSize = 100
	}
	if c.QueryTimeout <= 0 {
		c.QueryTimeout = 60 * time.Second
	}
}
```

---

## 2. `recon_heal_audit.go`
```go
package recon

import (
	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/model/system"
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// healAuditBatcher buffers per-record audit entries and flushes them in
// multi-row INSERT batches. Critical for prod scale (50M records → used
// to be 50M audit rows; now capped at ≤ 102 rows per heal run via the
// sampling + summary strategy below).
type healAuditBatcher struct {
	db       *gorm.DB
	logger   *zap.Logger
	mu       sync.Mutex
	buf      []system.ActivityLog
	maxBatch int
	// maxSampleUpsert caps how many "upsert" rows we persist per run.
	// Errors are always persisted (no sampling).
	maxSampleUpsert int

	table        string
	runID        string
	startedAt    time.Time
	upsertCount  int
	skipCount    int
	errorCount   int
	upsertLogged int // sampled-through count
	flushCount   int
}

func newHealAuditBatcher(db *gorm.DB, logger *zap.Logger, maxBatch, maxSampleUpsert int) *healAuditBatcher {
	if maxBatch <= 0 {
		maxBatch = 100
	}
	if maxSampleUpsert <= 0 {
		maxSampleUpsert = 100
	}
	return &healAuditBatcher{
		db:              db,
		logger:          logger,
		maxBatch:        maxBatch,
		maxSampleUpsert: maxSampleUpsert,
		buf:             make([]system.ActivityLog, 0, maxBatch),
	}
}

// Begin writes the run_started summary row. Not batched — one row per
// heal run is cheap and gives operators a head marker.
func (b *healAuditBatcher) Begin(ctx context.Context, runID, table string) {
	b.mu.Lock()
	b.runID = runID
	b.table = table
	b.startedAt = time.Now()
	b.mu.Unlock()

	details, _ := json.Marshal(map[string]interface{}{
		"action": "run_started",
		"run_id": runID,
	})
	now := time.Now()
	row := system.ActivityLog{
		Operation:   activity.OperationReconHeal.String(),
		TargetTable: table,
		Status:      "running",
		Details:     details,
		TriggeredBy: activity.TriggeredByReconHealer.String(),
		StartedAt:   now,
		CompletedAt: &now,
	}
	if err := b.db.WithContext(ctx).Create(&row).Error; err != nil && b.logger != nil {
		b.logger.Warn("heal audit run_started insert failed",
			zap.String("table", table), zap.String("run_id", runID), zap.Error(err))
	}
}

// Record classifies one heal action. Skips never produce rows; upserts
// are sampled; errors are always persisted. Counters update
// unconditionally.
func (b *healAuditBatcher) Record(ctx context.Context, action, recordID string, sourceTsMs int64, errMsg string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch action {
	case "skip":
		b.skipCount++
		return // aggregate only — NO per-record row
	case "upsert":
		b.upsertCount++
		if b.upsertLogged >= b.maxSampleUpsert {
			return // sampled out — counter keeps growing, row suppressed
		}
		b.upsertLogged++
	case "error":
		b.errorCount++
	default:
		// Unknown action — log it anyway; cheap and rare.
	}

	detailsMap := map[string]interface{}{
		"action":    action,
		"record_id": recordID,
		"run_id":    b.runID,
	}
	if sourceTsMs > 0 {
		detailsMap["source_ts_ms"] = sourceTsMs
	}
	if errMsg != "" {
		detailsMap["error"] = errMsg
	}
	details, _ := json.Marshal(detailsMap)

	status := "success"
	if action == "error" {
		status = "error"
	}
	now := time.Now()
	b.buf = append(b.buf, system.ActivityLog{
		Operation:   activity.OperationReconHeal.String(),
		TargetTable: b.table,
		Status:      status,
		Details:     details,
		TriggeredBy: activity.TriggeredByReconHealer.String(),
		StartedAt:   now,
		CompletedAt: &now,
	})

	if len(b.buf) >= b.maxBatch {
		b.flushLocked(ctx)
	}
}

// Flush drains whatever is buffered. Safe to call at any point.
func (b *healAuditBatcher) Flush(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked(ctx)
}

func (b *healAuditBatcher) flushLocked(ctx context.Context) {
	if len(b.buf) == 0 {
		return
	}
	if err := b.db.WithContext(ctx).CreateInBatches(&b.buf, b.maxBatch).Error; err != nil && b.logger != nil {
		b.logger.Warn("heal audit batch flush failed",
			zap.Int("batch", len(b.buf)),
			zap.String("table", b.table),
			zap.Error(err),
		)
	}
	b.flushCount++
	b.buf = b.buf[:0]
}

// End drains the buffer and writes the run_completed summary. Must be
// called exactly once per Begin.
func (b *healAuditBatcher) End(ctx context.Context, status string, runErr error, usedSignal bool, signalID string) {
	b.Flush(ctx)

	b.mu.Lock()
	startedAt := b.startedAt
	runID := b.runID
	table := b.table
	upsertCount := b.upsertCount
	skipCount := b.skipCount
	errorCount := b.errorCount
	flushCount := b.flushCount
	b.mu.Unlock()

	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
	}
	duration := time.Since(startedAt)
	durMs := duration.Milliseconds()

	detailsMap := map[string]interface{}{
		"action":         "run_completed",
		"run_id":         runID,
		"status":         status,
		"upserted_count": upsertCount,
		"skipped_count":  skipCount,
		"errored_count":  errorCount,
		"duration_ms":    durMs,
		"audit_flushes":  flushCount,
		"used_signal":    usedSignal,
	}
	if signalID != "" {
		detailsMap["signal_id"] = signalID
	}
	if errMsg != "" {
		detailsMap["error"] = errMsg
	}
	details, _ := json.Marshal(detailsMap)

	durInt := int(durMs)
	now := time.Now()
	row := system.ActivityLog{
		Operation:   activity.OperationReconHeal.String(),
		TargetTable: table,
		Status:      status,
		Details:     details,
		TriggeredBy: activity.TriggeredByReconHealer.String(),
		StartedAt:   startedAt,
		CompletedAt: &now,
		DurationMs:  &durInt,
	}
	if errMsg != "" {
		row.ErrorMessage = &errMsg
	}
	if err := b.db.WithContext(ctx).Create(&row).Error; err != nil && b.logger != nil {
		b.logger.Warn("heal audit run_completed insert failed",
			zap.String("table", table), zap.String("run_id", runID), zap.Error(err))
	}
}

// FlushCount returns the number of batch inserts performed.
func (b *healAuditBatcher) FlushCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushCount
}
```

---

## 3. `recon_heal_action.go`
```go
package recon

import (
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/service/shadow"
	"centralized-data-service/pkgs/metrics"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// HealMissingIDs performs the v3 heal pipeline on a batch of missing
// primary-key strings.
func (rh *ReconHealer) HealMissingIDs(
	ctx context.Context,
	entry source.TableRegistry,
	ids []string,
) (*HealResult, error) {
	batcher := newHealAuditBatcher(rh.db, rh.logger, rh.cfg.AuditFlushSize, rh.cfg.AuditFlushSize)
	runID := newHealRunID()
	batcher.Begin(ctx, runID, entry.TargetTable)

	res, err := rh.healMissingIDsWithBatcher(ctx, entry, ids, batcher, runID)
	if res != nil {
		res.RunID = runID
	}
	status := "success"
	if err != nil {
		status = "error"
	}
	batcher.End(ctx, status, err, false, "")
	if res != nil {
		res.AuditFlushCnt = batcher.FlushCount()
	}
	return res, err
}

func (rh *ReconHealer) healMissingIDsWithBatcher(
	ctx context.Context,
	entry source.TableRegistry,
	ids []string,
	batcher *healAuditBatcher,
	runID string,
) (*HealResult, error) {
	if len(ids) == 0 {
		return &HealResult{Table: entry.TargetTable, RunID: runID}, nil
	}
	if rh.mongoClient == nil {
		return nil, fmt.Errorf("heal: mongo client not configured")
	}
	if rh.schemaAdapter == nil {
		return nil, fmt.Errorf("heal: schema adapter not configured")
	}

	schema := rh.schemaAdapter.GetSchema(entry.TargetTable)
	if schema == nil {
		return nil, fmt.Errorf("heal: schema not found for %s", entry.TargetTable)
	}

	start := time.Now()
	res := &HealResult{Table: entry.TargetTable, Requested: len(ids), RunID: runID}

	coll := rh.mongoClient.
		Database(entry.SourceDB).
		Collection(
			entry.SourceTable,
			options.Collection().SetReadPreference(readpref.Primary()),
		)

	for _, chunk := range chunkStrings(ids, rh.cfg.BatchSize) {
		chunkCtx, cancel := context.WithTimeout(ctx, rh.cfg.QueryTimeout)

		inVals := make([]interface{}, 0, len(chunk))
		for _, s := range chunk {
			if oid, err := primitive.ObjectIDFromHex(s); err == nil {
				inVals = append(inVals, oid)
			} else {
				inVals = append(inVals, s)
			}
		}

		cursor, err := coll.Find(chunkCtx, bson.M{"_id": bson.M{"$in": inVals}})
		if err != nil {
			cancel()
			res.Errored += len(chunk)
			metrics.ReconHealActions.
				WithLabelValues(entry.TargetTable, "error").
				Add(float64(len(chunk)))
			rh.logger.Warn("heal: mongo find chunk failed",
				zap.String("table", entry.TargetTable),
				zap.Int("chunk_size", len(chunk)),
				zap.Error(err),
			)
			continue
		}

		var docs []bson.M
		for cursor.Next(chunkCtx) {
			var d bson.M
			if err := cursor.Decode(&d); err != nil {
				rh.logger.Warn("heal: decode failed", zap.Error(err))
				continue
			}
			docs = append(docs, d)
		}
		cursor.Close(chunkCtx)
		cancel()

		fetched := make(map[string]bool, len(docs))

		for _, doc := range docs {
			idStr := extractDocIDString(doc)
			if idStr == "" {
				res.Errored++
				metrics.ReconHealActions.
					WithLabelValues(entry.TargetTable, "error").
					Inc()
				continue
			}
			fetched[idStr] = true

			action, err := rh.applyOne(ctx, schema, entry, doc, idStr)
			if err != nil {
				res.Errored++
				metrics.ReconHealActions.
					WithLabelValues(entry.TargetTable, "error").
					Inc()
				rh.logger.Warn("heal: upsert failed",
					zap.String("table", entry.TargetTable),
					zap.String("id", idStr),
					zap.Error(err),
				)
				batcher.Record(ctx, "error", idStr, extractSourceTsFromDoc(doc), err.Error())
				continue
			}

			switch action {
			case "upsert":
				res.Upserted++
			case "skip":
				res.Skipped++
			}
			metrics.ReconHealActions.
				WithLabelValues(entry.TargetTable, action).
				Inc()

			batcher.Record(ctx, action, idStr, extractSourceTsFromDoc(doc), "")
		}

		for _, s := range chunk {
			if !fetched[s] {
				res.Skipped++
				metrics.ReconHealActions.
					WithLabelValues(entry.TargetTable, "skip").
					Inc()
				batcher.Record(ctx, "skip", s, 0, "")
			}
		}
	}

	res.DurationMs = time.Since(start).Milliseconds()

	rh.logger.Info("heal batch completed",
		zap.String("table", entry.TargetTable),
		zap.Int("requested", res.Requested),
		zap.Int("upserted", res.Upserted),
		zap.Int("skipped", res.Skipped),
		zap.Int("errored", res.Errored),
		zap.Int64("duration_ms", res.DurationMs),
	)
	return res, nil
}

// HealOrphanedIDs removes destination rows that no longer exist in the Mongo source.
func (rh *ReconHealer) HealOrphanedIDs(
	ctx context.Context,
	entry source.TableRegistry,
	missingFromSrc []string,
) (*HealResult, error) {
	if len(missingFromSrc) == 0 {
		return &HealResult{Table: entry.TargetTable}, nil
	}
	if err := validateIdent(entry.TargetTable); err != nil {
		return nil, err
	}
	if err := validateIdent(entry.PrimaryKeyField); err != nil {
		return nil, err
	}

	start := time.Now()
	res := &HealResult{
		Table:     entry.TargetTable,
		Requested: len(missingFromSrc),
	}

	for _, chunk := range chunkStrings(missingFromSrc, rh.cfg.BatchSize) {
		args := make([]interface{}, 0, len(chunk)+1)
		placeholders := make([]string, 0, len(chunk))
		for _, id := range chunk {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		query := fmt.Sprintf(
			`DELETE FROM %s WHERE %s IN (%s)`,
			quoteIdent(entry.TargetTable),
			quoteIdent(entry.PrimaryKeyField),
			strings.Join(placeholders, ", "),
		)
		execRes := rh.db.WithContext(ctx).Exec(query, args...)
		if execRes.Error != nil {
			res.Errored += len(chunk)
			rh.logger.Warn("heal orphaned delete failed",
				zap.String("table", entry.TargetTable),
				zap.Int("chunk_size", len(chunk)),
				zap.Error(execRes.Error),
			)
			continue
		}
		res.Deleted += int(execRes.RowsAffected)
	}

	res.DurationMs = time.Since(start).Milliseconds()
	rh.logger.Info("heal orphaned IDs completed",
		zap.String("table", entry.TargetTable),
		zap.Int("requested", res.Requested),
		zap.Int("deleted", res.Deleted),
		zap.Int("errored", res.Errored),
		zap.Int64("duration_ms", res.DurationMs),
	)
	return res, nil
}

// HealWindow is the orchestrated heal flow.
func (rh *ReconHealer) HealWindow(
	ctx context.Context,
	entry source.TableRegistry,
	tLo, tHi time.Time,
	missingIDs []string,
) (*HealResult, error) {
	runID := newHealRunID()
	res := &HealResult{Table: entry.TargetTable, Requested: len(missingIDs), RunID: runID}
	start := time.Now()

	batcher := newHealAuditBatcher(rh.db, rh.logger, rh.cfg.AuditFlushSize, rh.cfg.AuditFlushSize)
	batcher.Begin(ctx, runID, entry.TargetTable)

	var runErr error
	defer func() {
		status := "success"
		if runErr != nil || res.Errored > 0 {
			status = "error"
			if runErr == nil && res.Errored > 0 {
				runErr = fmt.Errorf("%d record(s) errored during heal", res.Errored)
			}
		}
		res.DurationMs = time.Since(start).Milliseconds()
		res.AuditFlushCnt = batcher.FlushCount()
		batcher.End(ctx, status, runErr, res.UsedSignal, res.SignalID)
	}()

	// Phase A — Debezium signal, best-effort.
	if rh.signal != nil && rh.signal.IsConfigured() && !rh.cfg.ForceDirect {
		connectorName := ResolveConnectorNameBySource(ctx, rh.db, entry.SourceDB, entry.SourceTable)
		healthy, err := rh.signal.IsConnectorHealthy(ctx, connectorName)
		if err != nil {
			rh.logger.Warn("heal: connector status probe failed, skipping signal path",
				zap.String("connector_name", connectorName),
				zap.Error(err),
			)
		} else if healthy {
			filter := BuildUpdatedAtRangeFilter(tLo, tHi)
			engine := ResolveEngineTypeBySource(ctx, rh.db, entry.SourceDB, entry.SourceTable)
			signalID, err := rh.signal.TriggerIncrementalSnapshot(
				ctx, connectorName, engine, entry.SourceDB, entry.SourceTable, filter,
			)
			if err != nil {
				rh.logger.Warn("heal: debezium signal insert failed, falling back to direct",
					zap.Error(err),
				)
			} else {
				res.UsedSignal = true
				res.SignalID = signalID
				rh.logger.Info("heal: debezium incremental snapshot requested",
					zap.String("table", entry.TargetTable),
					zap.String("filter", filter),
					zap.String("signal_id", signalID),
				)
			}
		} else {
			rh.logger.Info("heal: connector NOT running, skipping signal path",
				zap.String("table", entry.TargetTable),
			)
		}
	}

	// Phase B — direct heal for the exact list.
	if len(missingIDs) > 0 {
		direct, err := rh.healMissingIDsWithBatcher(ctx, entry, missingIDs, batcher, runID)
		if err != nil {
			runErr = err
			return res, err
		}
		if direct != nil {
			res.Upserted += direct.Upserted
			res.Skipped += direct.Skipped
			res.Errored += direct.Errored
		}
	}

	return res, nil
}

// applyOne builds + executes the OCC upsert for a single Mongo doc.
func (rh *ReconHealer) applyOne(
	ctx context.Context,
	schema *shadow.TableSchema,
	entry source.TableRegistry,
	doc bson.M,
	idStr string,
) (string, error) {
	data := make(map[string]interface{}, len(doc))
	for k, v := range doc {
		data[k] = unwrapBSONValue(v)
	}

	rawJSON := rh.buildMaskedRawJSON(entry.ID, data)
	srcTsMs := extractSourceTsFromDoc(doc)

	query, values := rh.schemaAdapter.BuildUpsertSQL(
		schema,
		entry.TargetTable,
		entry.PrimaryKeyField,
		idStr,
		data,
		string(rawJSON),
		"recon-heal",
		md5Hex(rawJSON),
		srcTsMs,
	)

	execRes := rh.db.WithContext(ctx).Exec(query, values...)
	if err := execRes.Error; err != nil {
		return "error", err
	}
	if execRes.RowsAffected == 0 {
		return "skip", nil
	}
	return "upsert", nil
}
```

---

## 4. `recon_heal_utils.go`
```go
package recon

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (rh *ReconHealer) buildMaskedRawJSON(key interface{}, data map[string]interface{}) []byte {
	maskedData := data
	if rh.masking != nil {
		maskedData = rh.masking.MaskTableData(key, data)
	}
	rawJSON, _ := json.Marshal(maskedData)
	return rawJSON
}

func newHealRunID() string {
	now := time.Now().UTC()
	return fmt.Sprintf("heal-%s-%06x",
		now.Format("20060102T150405.000"),
		uint32(now.UnixNano())&0xFFFFFF,
	)
}

func chunkStrings(ids []string, size int) [][]string {
	if size <= 0 {
		return [][]string{ids}
	}
	s := make([]string, len(ids))
	copy(s, ids)
	sort.Strings(s)

	out := make([][]string, 0, (len(s)+size-1)/size)
	for i := 0; i < len(s); i += size {
		j := i + size
		if j > len(s) {
			j = len(s)
		}
		out = append(out, s[i:j])
	}
	return out
}

func extractDocIDString(doc bson.M) string {
	v, ok := doc["_id"]
	if !ok {
		return ""
	}
	switch vv := v.(type) {
	case primitive.ObjectID:
		return vv.Hex()
	case string:
		return vv
	case int32:
		return fmt.Sprintf("%d", vv)
	case int64:
		return fmt.Sprintf("%d", vv)
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func extractSourceTsFromDoc(doc bson.M) int64 {
	if v, ok := doc["updated_at"]; ok {
		switch tv := v.(type) {
		case time.Time:
			return tv.UnixMilli()
		case primitive.DateTime:
			return tv.Time().UnixMilli()
		}
	}
	if v, ok := doc["_id"]; ok {
		if oid, ok := v.(primitive.ObjectID); ok {
			return oid.Timestamp().UnixMilli()
		}
	}
	return 0
}
```

---

## 5. `recon_heal_legacy.go`
```go
package recon

import (
	"centralized-data-service/internal/service/governance"
	"context"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

// NewReconHealerForTest returns a ReconHealer with only the masking
// service wired up, used by external test/ files.
func NewReconHealerForTest(masking *governance.MaskingService) *ReconHealer {
	return &ReconHealer{masking: masking}
}

// BuildMaskedRawJSONForTest exposes buildMaskedRawJSON for tests.
func (rh *ReconHealer) BuildMaskedRawJSONForTest(targetTable string, data map[string]interface{}) []byte {
	return rh.buildMaskedRawJSON(targetTable, data)
}

// ExtractSourceTsFromDocForTest exposes extractSourceTsFromDoc for tests.
func ExtractSourceTsFromDocForTest(doc map[string]interface{}) int64 {
	return extractSourceTsFromDoc(doc)
}

// NewHealRunIDForTest exposes newHealRunID for tests.
func NewHealRunIDForTest() string { return newHealRunID() }

// HealAuditBatcherForTest narrows healAuditBatcher to the methods used
// by external tests.
type HealAuditBatcherForTest interface {
	Begin(ctx context.Context, runID, table string)
	Record(ctx context.Context, action, recordID string, sourceTsMs int64, errMsg string)
	End(ctx context.Context, status string, runErr error, usedSignal bool, signalID string)
}

// NewHealAuditBatcherForTest exposes newHealAuditBatcher for tests.
func NewHealAuditBatcherForTest(db *gorm.DB, logger *zap.Logger, maxBatch, maxSampleUpsert int) HealAuditBatcherForTest {
	return newHealAuditBatcher(db, logger, maxBatch, maxSampleUpsert)
}

// ChunkStringsForTest exposes chunkStrings for tests.
func ChunkStringsForTest(ids []string, size int) [][]string { return chunkStrings(ids, size) }
```

---

## 6. `recon_heal.go` (Rút gọn)
```go
package recon

import (
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/internal/service/shadow"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ReconHealer orchestrates the heal phases.
type ReconHealer struct {
	mongoClient   *mongo.Client
	db            *gorm.DB
	schemaAdapter *shadow.SchemaAdapter
	signal        *governance.DebeziumSignalClient // optional — nil disables Signal path
	masking       *governance.MaskingService
	logger        *zap.Logger
	cfg           ReconHealerConfig
}

// NewReconHealer wires the heal pipeline.
func NewReconHealer(
	db *gorm.DB,
	mongoClient *mongo.Client,
	schemaAdapter *shadow.SchemaAdapter,
	signal *governance.DebeziumSignalClient,
	masking *governance.MaskingService,
	cfg ReconHealerConfig,
	logger *zap.Logger,
) *ReconHealer {
	cfg.applyDefaults()
	if masking == nil {
		masking = governance.NewMaskingService(db, logger, cfg.SensitiveFieldMask...)
	}
	return &ReconHealer{
		db:            db,
		mongoClient:   mongoClient,
		schemaAdapter: schemaAdapter,
		signal:        signal,
		masking:       masking,
		logger:        logger,
		cfg:           cfg,
	}
}

// InvalidateMaskCache drops the cache — no-op.
func (rh *ReconHealer) InvalidateMaskCache() {
	// Cache is centralized in MetadataRegistryService; this is now a no-op.
}
```
