# Giải pháp kỹ thuật: Phân tách recon_handler.go (Gom nhóm theo Flow)

Để tránh băm nhỏ code thành quá nhiều file helper vụn vặt gây phân tán logic, chúng tôi đề xuất gom nhóm toàn bộ nội dung của `recon_handler.go` (844 dòng) thành đúng **3 file** mạch lạc theo flow logic:

---

## 1. `recon_handler.go` (MODIFY)
Chứa core struct `ReconHandler`, interface `NatsPublisher`, constructors, configurations, và các resolver helpers dùng chung (`resolveTargetTableConfig`, `resolveTableConfigByID`, `logActivity`).

```go
package recon

import (
	"centralized-data-service/internal/activity"
	"centralized-data-service/internal/model/source"
	"centralized-data-service/internal/model/system"
	servicerecon "centralized-data-service/internal/service/recon"
	"centralized-data-service/internal/service/shadow"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/internal/service/governance"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ReconHandler handles reconciliation NATS commands
type ReconHandler struct {
	reconCore  *servicerecon.ReconCore
	healer     *servicerecon.ReconHealer // plan WORKER task #9 — Phase 2/3 heal (legacy bypass; heal chính = re-trigger V4)
	db         *gorm.DB
	shadowDB   *gorm.DB // Recon V4 P2 — map _gpay_id→_source_id cho heal-B (WithShadowDB)
	schema     *shadow.SchemaAdapter
	metadata   metadata.MetadataRegistry
	masking    *governance.MaskingService
	backfill   *governance.BackfillSourceTsService
	tsDetector *governance.TimestampDetector // Migration 017 — manual re-detect
	signal     *governance.DebeziumSignalClient
	natsPub    NatsPublisher
	logger     *zap.Logger
}

// NatsPublisher is the minimal surface of nats.Conn the handler needs
// to publish result events. Kept as an interface so unit tests can
// substitute a no-op / in-memory mock.
type NatsPublisher interface {
	Publish(subject string, data []byte) error
}

func NewReconHandler(reconCore *servicerecon.ReconCore, db *gorm.DB, schema *shadow.SchemaAdapter, logger *zap.Logger) *ReconHandler {
	return &ReconHandler{reconCore: reconCore, db: db, schema: schema, logger: logger}
}

// WithBackfill wires the backfill service. Separated from the
// constructor so existing call sites that don't use backfill keep
// working.
func (h *ReconHandler) WithBackfill(b *governance.BackfillSourceTsService, pub NatsPublisher) *ReconHandler {
	h.backfill = b
	h.natsPub = pub
	return h
}

// WithHealer wires the v3 ReconHealer so HandleReconHeal routes through
// HealWindow (Phase 2/3 signal + batched direct heal) instead of the
// legacy ReconCore.Heal path.
func (h *ReconHandler) WithHealer(healer *servicerecon.ReconHealer) *ReconHandler {
	h.healer = healer
	return h
}

func (h *ReconHandler) WithMaskingService(masking *governance.MaskingService) *ReconHandler {
	h.masking = masking
	return h
}

// WithTimestampDetector wires the Migration 017 auto-detect service so
// HandleDetectTimestampField can sample Mongo and update registry rows.
// Separated from the constructor so legacy call sites keep compiling.
func (h *ReconHandler) WithTimestampDetector(td *governance.TimestampDetector) *ReconHandler {
	h.tsDetector = td
	return h
}

func (h *ReconHandler) WithMetadataRegistry(metadata metadata.MetadataRegistry) *ReconHandler {
	h.metadata = metadata
	return h
}

func (h *ReconHandler) WithSignalClient(s *governance.DebeziumSignalClient) *ReconHandler {
	h.signal = s
	return h
}

// --- Helpers ---

func (h *ReconHandler) resolveTargetTableConfig(targetTable string) *source.TableRegistry {
	// 1. Try Metadata Registry (cached)
	if h.metadata != nil {
		if item := h.metadata.GetTableConfig(targetTable); item != nil {
			return item
		}
		// Try with sd_ prefix fallback for V1
		if !strings.HasPrefix(targetTable, "sd_") {
			if item := h.metadata.GetTableConfig("sd_" + targetTable); item != nil {
				return item
			}
		}
		// 2. Try resolving by SourceTable (V2 modern behavior: lookup shadow by source name)
		if item := h.metadata.GetTableConfigBySource(targetTable); item != nil {
			return item
		}
	}

	// 3. Try V1 cdc_table_registry
	var v1 source.TableRegistry
	if err := h.db.Where("target_table = ?", targetTable).First(&v1).Error; err == nil {
		return &v1
	}
	if !strings.HasPrefix(targetTable, "sd_") {
		if err := h.db.Where("target_table = ?", "sd_"+targetTable).First(&v1).Error; err == nil {
			return &v1
		}
	}

	return nil
}

func (h *ReconHandler) resolveTableConfigByID(id uint) *source.TableRegistry {
	if h.metadata != nil {
		if item := h.metadata.GetTableConfigByID(id); item != nil {
			return item
		}
	}
	var entry source.TableRegistry
	if err := h.db.Where("id = ?", id).First(&entry).Error; err != nil {
		return nil
	}
	return &entry
}

func (h *ReconHandler) logActivity(operation, table, status string, rows int64, err error) {
	now := time.Now()
	var errPtr *string
	if err != nil {
		e := err.Error()
		errPtr = &e
	}
	h.db.Create(&system.ActivityLog{
		Operation:    operation,
		TargetTable:  table,
		Status:       status,
		RowsAffected: rows,
		ErrorMessage: errPtr,
		TriggeredBy:  activity.TriggeredByNATSCommand.String(),
		StartedAt:    now,
		CompletedAt:  &now,
	})
}
```

---

## 2. `recon_handler_run.go` (NEW)
Gom nhóm toàn bộ flow thực thi Check và Heal chính của Recon (Segment A & Segment B).

```go
package recon

import (
	"centralized-data-service/internal/model/system"
	"centralized-data-service/pkgs/observability"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// HandleReconCheck — subscribe "cdc.cmd.recon-check"
func (h *ReconHandler) HandleReconCheck(msg *nats.Msg) {
	var payload struct {
		Tier    string `json:"tier"`
		Table   string `json:"table"`
		Segment string `json:"segment"` // ""|"source_shadow" = A (mặc định); "shadow_master" = B
		Deep    bool   `json:"deep"`    // P4: B per-table → chạy thêm L3-B row/field-diff
	}
	json.Unmarshal(msg.Data, &payload)

	h.logger.Info("recon check received", zap.String("tier", payload.Tier),
		zap.String("table", payload.Table), zap.String("segment", payload.Segment),
		zap.Bool("deep", payload.Deep))

	ctx, span := observability.ChildSpan(context.Background(), "cdc.recon.check",
		attribute.String("recon.tier", payload.Tier),
		attribute.String("recon.table", payload.Table),
		attribute.String("recon.segment", payload.Segment),
		attribute.Bool("recon.deep", payload.Deep),
	)
	defer observability.EndSpan(span, nil)

	// Recon V4 — Segment B (shadow↔master, transmute path).
	if payload.Segment == "shadow_master" {
		h.handleReconCheckSegmentB(ctx, msg, payload.Table, payload.Deep)
		return
	}

	// Orphan-prune (Segment A): soft-delete ghost shadow row không còn ở source.
	// Gốc của "shadow > source" khi source drop/re-seed (Mongo không emit per-doc delete).
	if payload.Tier == "prune" {
		if payload.Table == "*" || payload.Table == "" {
			reports := h.reconCore.PruneAllOrphans(ctx)
			total := 0
			for _, r := range reports {
				if r != nil {
					total += r.StaleCount
				}
			}
			result, _ := json.Marshal(map[string]interface{}{"status": "success", "tables": len(reports), "pruned": total})
			if msg.Reply != "" {
				msg.Respond(result)
			}
			h.logActivity("recon-prune-all", "*", "success", int64(total), nil)
			return
		}
		entry := h.resolveTargetTableConfig(payload.Table)
		if entry == nil {
			h.logActivity("recon-prune", payload.Table, "error", 0, fmt.Errorf("registry not found: %s", payload.Table))
			if msg.Reply != "" {
				msg.Respond([]byte(`{"status":"error","error":"registry_not_found"}`))
			}
			return
		}
		report := h.reconCore.RunOrphanPrune(ctx, *entry)
		pruned := 0
		if report != nil {
			pruned = report.StaleCount
		}
		result, _ := json.Marshal(map[string]interface{}{"status": "success", "table": payload.Table, "pruned": pruned})
		if msg.Reply != "" {
			msg.Respond(result)
		}
		h.logActivity("recon-prune", payload.Table, "success", int64(pruned), nil)
		return
	}

	if payload.Table == "*" || payload.Table == "" {
		// Check all. 0 bảng checked = wiring/materialise hỏng, KHÔNG phải
		// success (chống false-positive "success tables_checked=0" câm lặng).
		reports := h.reconCore.CheckAll(ctx)
		status := "success"
		if len(reports) == 0 {
			status = "warning"
		}
		result, _ := json.Marshal(map[string]interface{}{"status": status, "tables_checked": len(reports)})
		if msg.Reply != "" {
			msg.Respond(result)
		}
		h.logActivity("recon-check-all", "*", status, int64(len(reports)), nil)
		return
	}

	// Single table
	entry := h.resolveTargetTableConfig(payload.Table)
	if entry == nil {
		h.logActivity("recon-check", payload.Table, "error", 0, fmt.Errorf("registry not found: %s", payload.Table))
		return
	}

	var report *system.ReconciliationReport
	switch payload.Tier {
	case "2":
		report = h.reconCore.RunTier2(ctx, *entry)
	case "3":
		report = h.reconCore.RunTier3(ctx, *entry)
	default:
		report = h.reconCore.RunTier1(ctx, *entry)
	}

	h.logActivity("recon-check", payload.Table, report.Status, report.Diff, nil)

	if msg.Reply != "" {
		result, _ := json.Marshal(report)
		msg.Respond(result)
	}
}

// handleReconCheckSegmentB — Recon V4: đối soát shadow↔master. table="*" =
// mọi master binding active+approved; table cụ thể = resolve theo master_table.
func (h *ReconHandler) handleReconCheckSegmentB(ctx context.Context, msg *nats.Msg, table string, deep bool) {
	if table == "*" || table == "" {
		reports := h.reconCore.CheckAllSegmentB(ctx)
		status := "success"
		if len(reports) == 0 {
			status = "warning" // 0 checked ≠ success (chống false-positive câm lặng)
		}
		result, _ := json.Marshal(map[string]interface{}{
			"status": status, "segment": "shadow_master", "tables_checked": len(reports)})
		if msg.Reply != "" {
			msg.Respond(result)
		}
		h.logActivity("recon-check-b-all", "*", status, int64(len(reports)), nil)
		return
	}
	report := h.reconCore.RunSegmentBFor(ctx, table, deep)
	if report == nil {
		h.logActivity("recon-check-b", table, "error", 0,
			fmt.Errorf("master binding not found or not approved/active: %s", table))
		if msg.Reply != "" {
			res, _ := json.Marshal(map[string]string{"status": "error", "error": "master_binding_not_found"})
			msg.Respond(res)
		}
		return
	}
	h.logActivity("recon-check-b", table, report.Status, report.Diff, nil)
	if msg.Reply != "" {
		result, _ := json.Marshal(report)
		msg.Respond(result)
	}
}

func (h *ReconHandler) HandleReconHeal(msg *nats.Msg) {
	var payload struct {
		Table   string `json:"table"`
		Segment string `json:"segment"` // ""/"source_shadow" = A; "shadow_master" = B
		Legacy  bool   `json:"legacy"`  // true = ép đường bypass cũ (escape hatch, sẽ gỡ sau P4)
	}
	json.Unmarshal(msg.Data, &payload)

	h.logger.Info("recon heal received", zap.String("table", payload.Table),
		zap.String("segment", payload.Segment), zap.Bool("legacy", payload.Legacy))

	ctx, span := observability.ChildSpan(context.Background(), "cdc.recon.heal",
		attribute.String("recon.table", payload.Table),
		attribute.String("recon.segment", payload.Segment),
		attribute.Bool("recon.legacy", payload.Legacy),
	)
	defer observability.EndSpan(span, nil)

	// Recon V4 P2 — heal chính = RE-TRIGGER qua pipeline chuẩn (không bypass).
	if !payload.Legacy {
		if payload.Segment == "shadow_master" {
			h.healSegmentB(ctx, msg, payload.Table)
		} else {
			h.healSegmentA(ctx, msg, payload.Table)
		}
		return
	}

	if h.healer == nil {
		err := fmt.Errorf("v3 healer not wired — worker_server init is broken; refusing to fall back to legacy heal")
		h.logger.Error("recon heal rejected", zap.String("table", payload.Table), zap.Error(err))
		h.logActivity("recon-heal", payload.Table, "error", 0, err)
		if msg.Reply != "" {
			result, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
			msg.Respond(result)
		}
		return
	}

	entry := h.resolveTargetTableConfig(payload.Table)
	if entry == nil {
		h.logActivity("recon-heal", payload.Table, "error", 0, fmt.Errorf("registry not found"))
		return
	}

	// Get latest Tier 2 report with missing IDs
	var report system.ReconciliationReport
	if err := h.db.Where("target_table = ? AND tier = 2 AND missing_count > 0", payload.Table).
		Order("checked_at DESC").First(&report).Error; err != nil {
		// No Tier 2 report → run Tier 2 first
		h.logger.Info("no tier 2 report, running tier 2 first", zap.String("table", payload.Table))
		newReport := h.reconCore.RunTier2(ctx, *entry)
		if newReport.MissingCount == 0 {
			h.logActivity("recon-heal", payload.Table, "success", 0, nil)
			return
		}
		report = *newReport
	}

	var missingIDs []string
	json.Unmarshal(report.MissingIDs, &missingIDs)

	// Window bounds for Phase A signal: derive from the report's
	// check time backwards by the default recon lookback. Debezium
	// uses these as the updated_at filter for incremental snapshots.
	tHi := report.CheckedAt
	if tHi.IsZero() {
		tHi = time.Now()
	}
	tLo := tHi.Add(-7 * 24 * time.Hour)

	res, healErr := h.healer.HealWindow(ctx, *entry, tLo, tHi, missingIDs)
	var healedCount int
	if res != nil {
		healedCount = res.Upserted
		h.logger.Info("recon heal via v3 healer",
			zap.String("table", payload.Table),
			zap.Int("upserted", res.Upserted),
			zap.Int("skipped", res.Skipped),
			zap.Int("errored", res.Errored),
			zap.Bool("used_signal", res.UsedSignal),
			zap.String("signal_id", res.SignalID),
		)
	}

	if healErr != nil {
		h.logActivity("recon-heal", payload.Table, "error", 0, healErr)
	} else {
		h.logActivity("recon-heal", payload.Table, "success", int64(healedCount), nil)
	}

	if msg.Reply != "" {
		result, _ := json.Marshal(map[string]interface{}{"healed": healedCount, "total_missing": len(missingIDs)})
		msg.Respond(result)
	}
}
```

---

## 3. `recon_handler_ops.go` (NEW)
Gom nhóm toàn bộ các luồng vận hành phụ trợ (ops & maintenance commands).

```go
package recon

import (
	"centralized-data-service/internal/model/shadow"
	sourcemodel "centralized-data-service/internal/model/source"
	"centralized-data-service/internal/service/source"
	"centralized-data-service/pkgs/natsconn"
	"centralized-data-service/pkgs/observability"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// HandleRetryFailed — subscribe "cdc.cmd.retry-failed"
func (h *ReconHandler) HandleRetryFailed(msg *nats.Msg) {
	var payload struct {
		FailedLogID uint64 `json:"failed_log_id"`
		TargetTable string `json:"target_table"`
		RecordID    string `json:"record_id"`
		RawJSON     string `json:"raw_json"`
	}
	json.Unmarshal(msg.Data, &payload)

	h.logger.Info("retry failed received", zap.Uint64("id", payload.FailedLogID), zap.String("table", payload.TargetTable))

	sanitizedRawJSON := h.sanitizeRetryRawJSON(payload.TargetTable, payload.RawJSON)

	// Parse raw JSON → map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(sanitizedRawJSON), &data); err != nil {
		h.updateFailedLog(payload.FailedLogID, "failed", err.Error())
		return
	}

	// Get registry for PK field
	entry := h.resolveTargetTableConfig(payload.TargetTable)
	if entry == nil {
		h.updateFailedLog(payload.FailedLogID, "failed", "registry not found")
		return
	}

	// Upsert via SchemaAdapter
	schema := h.schema.GetSchema(payload.TargetTable)
	if schema == nil {
		h.schema.PrepareForCDCInsert(payload.TargetTable, entry.PrimaryKeyField)
		schema = h.schema.GetSchema(payload.TargetTable)
	}
	if schema == nil {
		h.updateFailedLog(payload.FailedLogID, "failed", "schema not found")
		return
	}

	// Retry path: unknown Debezium ts_ms → pass 0 so the OCC guard on
	// _source_ts is skipped (hash guard still applies).
	query, values := h.schema.BuildUpsertSQL(schema, payload.TargetTable, entry.PrimaryKeyField, payload.RecordID, data, sanitizedRawJSON, "retry", "", 0)
	if err := h.db.Exec(query, values...).Error; err != nil {
		h.updateFailedLog(payload.FailedLogID, "failed", err.Error())
		return
	}

	h.updateFailedLog(payload.FailedLogID, "resolved", "")
	h.logActivity("retry-failed", payload.TargetTable, "success", 1, nil)
}

// HandleDebeziumSignal — subscribe "cdc.cmd.debezium-signal" + "cdc.cmd.debezium-snapshot"
func (h *ReconHandler) HandleDebeziumSignal(msg *nats.Msg) {
	var payload struct {
		Type       string `json:"type"`
		Database   string `json:"database"`
		Collection string `json:"collection"`
		Table      string `json:"table"`
		Filter     string `json:"filter"` // optional incremental snapshot filter
		TraceID    string `json:"trace_id"`
		Action     string `json:"action"`
		Origin     string `json:"origin"`
	}
	json.Unmarshal(msg.Data, &payload)
	trace := natsconn.CompleteActionTrace(msg, payload.TraceID, payload.Action, payload.Origin, "snapshot_now")

	h.logger.Info("debezium signal received",
		zap.String("trace_id", trace.TraceID),
		zap.String("action", trace.Action),
		zap.String("origin", trace.Origin),
		zap.String("type", payload.Type),
		zap.String("table", payload.Table),
		zap.String("database", payload.Database),
		zap.String("collection", payload.Collection),
	)

	// 1. Resolve source database + collection
	db := payload.Database
	collection := payload.Collection
	if payload.Table != "" && db == "" {
		// Lookup from registry
		if entry := h.resolveTargetTableConfig(payload.Table); entry != nil {
			db = entry.SourceDB
			collection = entry.SourceTable
		}
	}

	if db == "" || collection == "" {
		h.logger.Warn("action trace failed",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.String("reason", "database or collection could not be resolved"),
			zap.String("hint", "FE phải truyền database+collection, hoặc cdc_table_registry phải có row khớp payload.Table"),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0, fmt.Errorf("database or collection could not be resolved"))
		return
	}

	// 2. Dispatch via Kafka SignalClient. Source DB is NEVER written.
	if h.signal == nil || !h.signal.IsConfigured() {
		err := fmt.Errorf("debezium signal client not configured (kafka brokers missing); refusing to write to source DB")
		h.logger.Warn("action trace failed",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.String("dispatch_path", "signal_client"),
			zap.String("database", db),
			zap.String("collection", collection),
			zap.Error(err),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0, err)
		return
	}

	h.logger.Info("debezium signal: using SignalClient path",
		zap.String("trace_id", trace.TraceID),
		zap.String("database", db),
		zap.String("collection", collection),
	)
	engine := source.ResolveEngineTypeBySource(context.Background(), h.db, db, collection)
	connectorName := source.ResolveConnectorNameBySource(context.Background(), h.db, db, collection)
	if connectorName == "" {
		err := fmt.Errorf("no connector registered for %s.%s — cannot resolve topic.prefix for signal key", db, collection)
		h.logger.Warn("action trace failed",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.String("dispatch_path", "signal_client"),
			zap.String("database", db),
			zap.String("collection", collection),
			zap.Error(err),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0, err)
		return
	}
	signalID, err := h.signal.TriggerIncrementalSnapshot(context.Background(), connectorName, engine, db, collection, payload.Filter)
	if err != nil {
		h.logger.Warn("action trace failed",
			zap.String("trace_id", trace.TraceID),
			zap.String("action", trace.Action),
			zap.String("dispatch_path", "signal_client"),
			zap.String("database", db),
			zap.String("collection", collection),
			zap.Error(err),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0, err)
		return
	}

	h.logger.Info("debezium signal dispatched",
		zap.String("trace_id", trace.TraceID),
		zap.String("action", trace.Action),
		zap.String("dispatch_path", "signal_client"),
		zap.String("signal_id", signalID),
		zap.String("table", payload.Table),
		zap.String("database", db),
		zap.String("collection", collection),
	)

	connHealth, healthErr := h.signal.CheckConnectorHealth(context.Background(), connectorName)
	if healthErr != nil {
		h.logger.Error("debezium signal published BUT connector status probe failed",
			zap.String("trace_id", trace.TraceID),
			zap.String("signal_id", signalID),
			zap.String("table", payload.Table),
			zap.String("database", db),
			zap.String("collection", collection),
			zap.String("connector_name", connectorName),
			zap.Error(healthErr),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0,
			fmt.Errorf("signal published to kafka but connector status probe failed (connector=%q): %w", connectorName, healthErr))
		return
	}
	if !connHealth.Healthy {
		h.logger.Error("debezium signal published BUT connector not ready — snapshot will NOT execute",
			zap.String("trace_id", trace.TraceID),
			zap.String("signal_id", signalID),
			zap.String("table", payload.Table),
			zap.String("database", db),
			zap.String("collection", collection),
			zap.String("connector_name", connectorName),
			zap.String("connector_state", connHealth.State),
			zap.Int("task_count", connHealth.TaskCount),
			zap.String("task_state", connHealth.TaskState),
			zap.String("reason", connHealth.Reason),
		)
		h.logActivity("debezium-signal", payload.Table, "error", 0,
			fmt.Errorf("signal published to kafka but connector %q not ready: state=%s task_count=%d task_state=%s reason=%s",
				connectorName, connHealth.State, connHealth.TaskCount, connHealth.TaskState, connHealth.Reason))
		return
	}

	h.logger.Info("debezium signal end-to-end ready",
		zap.String("trace_id", trace.TraceID),
		zap.String("signal_id", signalID),
		zap.String("connector_name", connectorName),
		zap.String("connector_state", connHealth.State),
		zap.Int("task_count", connHealth.TaskCount),
	)
	h.logActivity("debezium-signal", payload.Table, "success", 1, nil)
}

// HandleBackfillSourceTs — subscribe "cdc.cmd.recon-backfill-source-ts".
func (h *ReconHandler) HandleBackfillSourceTs(msg *nats.Msg) {
	var payload struct {
		Table     string `json:"table"`
		RunID     string `json:"run_id"`
		BatchSize int    `json:"batch_size"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Warn("backfill: invalid payload", zap.Error(err))
		return
	}

	if h.backfill == nil {
		h.logger.Warn("backfill: service not configured")
		h.logActivity("recon-backfill-source-ts", payload.Table, "error", 0, fmt.Errorf("backfill service not configured"))
		return
	}

	ctx, span := observability.ChildSpan(context.Background(), "cdc.recon.backfill_source_ts",
		attribute.String("recon.table", payload.Table),
	)
	defer observability.EndSpan(span, nil)
	var tables []string
	if payload.Table != "" {
		tables = []string{payload.Table}
	}

	h.logger.Info("backfill: run starting",
		zap.String("run_id", payload.RunID),
		zap.Strings("tables", tables),
		zap.Int("batch_size", payload.BatchSize),
	)

	results, err := h.backfill.BackfillAll(ctx, payload.RunID, tables)
	outcome := "success"
	if err != nil {
		outcome = "error"
		h.logger.Error("backfill: run failed", zap.Error(err))
	}

	h.backfill.WriteActivity(ctx, payload.RunID, results, outcome)

	if h.natsPub != nil {
		result, _ := json.Marshal(map[string]interface{}{
			"run_id":  payload.RunID,
			"outcome": outcome,
			"results": results,
		})
		_ = h.natsPub.Publish("cdc.result.recon-backfill-source-ts", result)
	}

	if msg.Reply != "" {
		result, _ := json.Marshal(map[string]interface{}{
			"run_id":  payload.RunID,
			"outcome": outcome,
			"results": results,
		})
		msg.Respond(result)
	}
}

// HandleDetectTimestampField — subscribe "cdc.cmd.detect-timestamp-field".
func (h *ReconHandler) HandleDetectTimestampField(msg *nats.Msg) {
	var payload struct {
		RegistryID  uint   `json:"registry_id"`
		TargetTable string `json:"target_table"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.logger.Warn("detect-timestamp-field: bad payload", zap.Error(err))
		return
	}

	if h.tsDetector == nil {
		h.logger.Warn("detect-timestamp-field: detector not wired")
		if msg.Reply != "" {
			resp, _ := json.Marshal(map[string]interface{}{"error": "detector not configured"})
			msg.Respond(resp)
		}
		return
	}

	ctx, span := observability.ChildSpan(context.Background(), "cdc.recon.detect_timestamp_field",
		attribute.String("recon.table", payload.TargetTable),
	)
	defer observability.EndSpan(span, nil)
	var entry *sourcemodel.TableRegistry
	if payload.RegistryID > 0 {
		entry = h.resolveTableConfigByID(payload.RegistryID)
	} else if payload.TargetTable != "" {
		entry = h.resolveTargetTableConfig(payload.TargetTable)
	} else {
		h.logger.Warn("detect-timestamp-field: missing registry_id and target_table")
		if msg.Reply != "" {
			resp, _ := json.Marshal(map[string]interface{}{"error": "registry_id or target_table required"})
			msg.Respond(resp)
		}
		return
	}
	if entry == nil {
		h.logger.Warn("detect-timestamp-field: registry lookup failed",
			zap.Uint("registry_id", payload.RegistryID),
			zap.String("target_table", payload.TargetTable))
		if msg.Reply != "" {
			resp, _ := json.Marshal(map[string]interface{}{"error": "registry not found"})
			msg.Respond(resp)
		}
		return
	}

	candidates := entry.GetCandidates()
	result, err := h.tsDetector.DetectForCollection(ctx, entry.SourceDB, entry.SourceTable, candidates, 100)
	if err != nil {
		h.logger.Warn("detect-timestamp-field: detection failed",
			zap.String("target_table", entry.TargetTable),
			zap.Error(err))
		h.logActivity("detect-timestamp-field", entry.TargetTable, "error", 0, err)
		if msg.Reply != "" {
			resp, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
			msg.Respond(resp)
		}
		return
	}

	sourceType := "auto"
	if entry.TimestampFieldSource != nil {
		sourceType = *entry.TimestampFieldSource
	}
	if sourceType == "admin_override" {
		h.logger.Info("detect-timestamp-field: admin_override — registry not mutated",
			zap.String("target_table", entry.TargetTable),
			zap.String("detected_field", result.Field),
			zap.String("confidence", result.Confidence),
		)
	} else {
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"timestamp_field":             result.Field,
			"timestamp_field_detected_at": now,
			"timestamp_field_confidence":  result.Confidence,
			"timestamp_field_source":      "auto",
		}
		if err := h.db.WithContext(ctx).Model(&sourcemodel.TableRegistry{}).
			Where("id = ?", entry.ID).Updates(updates).Error; err != nil {
			h.logger.Warn("detect-timestamp-field: registry update failed",
				zap.Uint("id", entry.ID), zap.Error(err))
		}
	}

	h.logActivity("detect-timestamp-field", entry.TargetTable, "success", 0, nil)

	if msg.Reply != "" {
		resp, _ := json.Marshal(map[string]interface{}{
			"registry_id":  entry.ID,
			"target_table": entry.TargetTable,
			"result":       result,
			"source":       sourceType,
		})
		msg.Respond(resp)
	}
}

func (h *ReconHandler) updateFailedLog(id uint64, status, errMsg string) {
	updates := map[string]interface{}{"status": status}
	if status == "resolved" {
		now := time.Now()
		updates["resolved_at"] = now
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	updates["last_retry_at"] = time.Now()
	h.db.Model(&shadow.FailedSyncLog{}).Where("id = ?", id).Updates(updates)
}

func (h *ReconHandler) sanitizeRetryRawJSON(table, raw string) string {
	if h.masking != nil {
		return string(h.masking.MaskJSONPayload(table, []byte(raw)))
	}
	if json.Valid([]byte(raw)) {
		return raw
	}
	wrapped, _ := json.Marshal(map[string]string{"raw": raw})
	return string(wrapped)
}

// SanitizeRetryRawJSONForTest exposes ReconHandler.sanitizeRetryRawJSON for the test suite.
func (h *ReconHandler) SanitizeRetryRawJSONForTest(table, raw string) string {
	return h.sanitizeRetryRawJSON(table, raw)
}
```
