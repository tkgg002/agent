# Giải pháp kỹ thuật: Phân tách transmuter.go (Gom nhóm theo Flow)

Để đảm bảo code dễ bảo trì, chúng tôi đề xuất phân tách file `transmuter.go` (903 dòng) thành đúng **3 file** lớn dựa trên flow logic nghiệp vụ:

---

## 1. `transmuter.go` (MODIFY)
Chứa package, imports, các core interfaces & structs, constructor, và các helper methods ghi nhận runtime state.

```go
package master

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	mastermodel "centralized-data-service/internal/model/master"
	repomaster "centralized-data-service/internal/repository/master"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/internal/service/shadow"
	"centralized-data-service/internal/service/source"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MasterDestinationEnsurer interface {
	EnsureMaster(ctx context.Context, masterName string) error
}

type TransmuterModule struct {
	systemDB    *gorm.DB
	connMgr     *source.ConnectionManager
	runtimeRepo *repomaster.SyncRuntimeStateRepo
	ddlEnsurer  MasterDestinationEnsurer
	typeRes     *shadow.TypeResolver
	logger      *zap.Logger
	batchSize   int
	mu          sync.Mutex
	cache       map[string]cachedRules
	cacheTTL    time.Duration
	shadowCache map[string]shadowState
}

type cachedRules struct {
	rules    []mappingRuleRow
	loadedAt time.Time
}

type shadowState struct {
	isActive      bool
	profileStatus string
	loadedAt      time.Time
}

type mappingRuleRow struct {
	ID              int64   `gorm:"column:id"`
	SourceObjectID  int64   `gorm:"column:source_object_id"`
	MasterBindingID *int64  `gorm:"column:master_binding_id"`
	SourceField     string  `gorm:"column:source_field"`
	TargetColumn    string  `gorm:"column:target_column"`
	DataType        string  `gorm:"column:data_type"`
	SourceFormat    string  `gorm:"column:source_format"`
	SourcePath      *string `gorm:"column:source_path"`
	TransformFn     *string `gorm:"column:transform_fn"`
	IsNullable      bool    `gorm:"column:is_nullable"`
	DefaultValue    *string `gorm:"column:default_value"`
}

type masterBindingRuntime struct {
	ID                  int64  `gorm:"column:id"`
	SourceObjectID      int64  `gorm:"column:source_object_id"`
	ShadowBindingID     *int64 `gorm:"column:shadow_binding_id"`
	MasterConnectionKey string `gorm:"column:master_connection_key"`
	MasterSchema        string `gorm:"column:master_schema"`
	MasterTable         string `gorm:"column:master_table"`
	PhysicalTableFQN    string `gorm:"column:physical_table_fqn"`
	TransformType       string `gorm:"column:transform_type"`
	TransformSpec       []byte `gorm:"column:transform_spec"`
	IsActive            bool   `gorm:"column:is_active"`
	SchemaStatus        string `gorm:"column:schema_status"`
	ShadowConnectionKey string `gorm:"column:shadow_connection_key"`
	ShadowSchema        string `gorm:"column:shadow_schema"`
	ShadowTable         string `gorm:"column:shadow_table"`
	ShadowIsActive      bool   `gorm:"column:shadow_is_active"`
	SourceProfileStatus string `gorm:"column:source_profile_status"`
	SourcePK            string `gorm:"column:source_pk"`
	ShadowPK            string `gorm:"column:shadow_pk"`
}

type shadowBatchRow struct {
	GpayID   int64  `gorm:"column:_gpay_id"`
	SourceID string `gorm:"column:_source_id"`
	RawData  []byte `gorm:"column:_raw_data"`
	SourceTs int64  `gorm:"column:_source_ts"`
	Deleted  bool   `gorm:"column:_deleted"`
}

type batchOutcome struct {
	scanned, inserted, updated, skipped, ruleMisses, typeErrors, lastGpayID int64
}

type TransmuteResult struct {
	Master     string `json:"master"`
	Source     string `json:"source"`
	Scanned    int64  `json:"scanned"`
	Inserted   int64  `json:"inserted"`
	Updated    int64  `json:"updated"`
	Skipped    int64  `json:"skipped"`
	RuleMisses int64  `json:"rule_misses"`
	TypeErrors int64  `json:"type_errors"`
	DurationMs int64  `json:"duration_ms"`
	ActiveGate string `json:"active_gate,omitempty"`
}

func NewTransmuterModule(
	systemDB *gorm.DB,
	connMgr *source.ConnectionManager,
	runtimeRepo *repomaster.SyncRuntimeStateRepo,
	ddlEnsurer MasterDestinationEnsurer,
	typeRes *shadow.TypeResolver,
	logger *zap.Logger,
) *TransmuterModule {
	return &TransmuterModule{
		systemDB:    systemDB,
		connMgr:     connMgr,
		runtimeRepo: runtimeRepo,
		ddlEnsurer:  ddlEnsurer,
		typeRes:     typeRes,
		logger:      logger,
		batchSize:   500,
		cache:       make(map[string]cachedRules),
		shadowCache: make(map[string]shadowState),
		cacheTTL:    60 * time.Second,
	}
}

// --- Runtime State Persist Helpers ---

func (t *TransmuterModule) markRuntimeSuccess(ctx context.Context, masterBindingID int64, res TransmuteResult) {
	stats, _ := json.Marshal(map[string]any{
		"master":      res.Master,
		"source":      res.Source,
		"scanned":     res.Scanned,
		"inserted":    res.Inserted,
		"updated":     res.Updated,
		"skipped":     res.Skipped,
		"rule_misses": res.RuleMisses,
		"type_errors": res.TypeErrors,
		"duration_ms": res.DurationMs,
		"active_gate": res.ActiveGate,
	})
	t.persistRuntimeState(ctx, masterBindingID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
		item.LastSuccessAt = &now
		item.LastErrorAt = nil
		item.LastErrorMessage = nil
		item.StatsJSON = stats
	})
}

func (t *TransmuterModule) markRuntimeFailure(ctx context.Context, masterBindingID int64, opErr error) {
	if opErr == nil {
		return
	}
	msg := governance.SanitizeFreeformText(opErr.Error(), 2000)
	t.persistRuntimeState(ctx, masterBindingID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
		item.LastErrorAt = &now
		item.LastErrorMessage = &msg
	})
}

func (t *TransmuterModule) markRuntimeSkipped(ctx context.Context, masterBindingID int64, reason string) {
	stats, _ := json.Marshal(map[string]any{
		"active_gate": reason,
		"status":      "skipped",
	})
	t.persistRuntimeState(ctx, masterBindingID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
		item.LastSuccessAt = &now
		item.LastErrorAt = nil
		item.LastErrorMessage = nil
		item.StatsJSON = stats
	})
}

func (t *TransmuterModule) persistRuntimeState(ctx context.Context, masterBindingID int64, mutate func(*mastermodel.SyncRuntimeState, time.Time)) {
	if t.runtimeRepo == nil || masterBindingID == 0 {
		return
	}
	item, err := t.runtimeRepo.GetByMasterBinding(ctx, masterBindingID)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.logger.Warn("transmuter runtime lookup failed",
			zap.Int64("master_binding_id", masterBindingID),
			zap.Error(err))
		return
	}
	now := time.Now().UTC()
	if err == gorm.ErrRecordNotFound || item == nil || item.ID == 0 {
		item = &mastermodel.SyncRuntimeState{
			MasterBindingID: &masterBindingID,
			RuntimeScope:    "master",
			StatsJSON:       []byte(`{}`),
		}
	}
	item.UpdatedAt = now
	mutate(item, now)
	if item.ID == 0 {
		if createErr := t.runtimeRepo.Create(ctx, item); createErr != nil {
			t.logger.Warn("transmuter runtime create failed",
				zap.Int64("master_binding_id", masterBindingID),
				zap.Error(createErr))
		}
		return
	}
	if updateErr := t.runtimeRepo.Update(ctx, item); updateErr != nil {
		t.logger.Warn("transmuter runtime update failed",
			zap.Int64("master_binding_id", masterBindingID),
			zap.Error(updateErr))
	}
}
```

---

## 2. `transmuter_run.go` (NEW)
Chứa core execution logic bao gồm `Run`, `loadMaster`, `shadowActive`, `loadRules`, `fetchShadowBatch`, `InvalidateRuleCache`.

```go
package master

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string) (TransmuteResult, error) {
	start := time.Now()
	res := TransmuteResult{Master: masterName}

	masterRow, err := t.loadMaster(ctx, masterName)
	if err != nil {
		t.markRuntimeFailure(ctx, 0, err)
		return res, fmt.Errorf("master lookup: %w", err)
	}
	if !masterRow.IsActive || masterRow.SchemaStatus != "approved" {
		res.ActiveGate = fmt.Sprintf("master gate: is_active=%v schema_status=%s", masterRow.IsActive, masterRow.SchemaStatus)
		t.markRuntimeSkipped(ctx, masterRow.ID, res.ActiveGate)
		return res, nil
	}
	res.Source = masterRow.ShadowTable

	if ok, reason := t.shadowActive(masterRow); !ok {
		res.ActiveGate = fmt.Sprintf("shadow gate: %s", reason)
		t.markRuntimeSkipped(ctx, masterRow.ID, res.ActiveGate)
		return res, nil
	}
	if t.ddlEnsurer != nil {
		if err := t.ddlEnsurer.EnsureMaster(ctx, masterRow.MasterTable); err != nil {
			t.markRuntimeFailure(ctx, masterRow.ID, err)
			return res, fmt.Errorf("ensure master destination: %w", err)
		}
	}

	rules, err := t.loadRules(ctx, masterRow)
	if err != nil {
		t.markRuntimeFailure(ctx, masterRow.ID, err)
		return res, fmt.Errorf("rule load: %w", err)
	}
	if len(rules) == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
		t.markRuntimeSuccess(ctx, masterRow.ID, res)
		return res, nil
	}

	var lastGpayID int64
	for {
		shadowRows, err := t.fetchShadowBatch(ctx, masterRow, lastGpayID, onlySourceIDs)
		if err != nil {
			t.markRuntimeFailure(ctx, masterRow.ID, err)
			return res, fmt.Errorf("fetch shadow batch: %w", err)
		}
		if len(shadowRows) == 0 {
			break
		}
		batchRes := t.processBatch(ctx, masterRow, rules, shadowRows)
		res.Scanned += batchRes.scanned
		res.Inserted += batchRes.inserted
		res.Updated += batchRes.updated
		res.Skipped += batchRes.skipped
		res.RuleMisses += batchRes.ruleMisses
		res.TypeErrors += batchRes.typeErrors
		lastGpayID = batchRes.lastGpayID
		if len(onlySourceIDs) > 0 {
			break
		}
	}
	res.DurationMs = time.Since(start).Milliseconds()
	// Data-safety guard (P0): đã quét được dòng shadow nhưng KHÔNG dòng nào tới
	// master (toàn bộ skip do transform error / emits rỗng / type error / upsert
	// fail) = FALSE-POSITIVE → báo failed kèm lý do, KHÔNG markSuccess.
	// (scanned=0 = shadow rỗng hợp lệ → vẫn success; all-unchanged → inserted>0.)
	if res.Scanned > 0 && res.Inserted == 0 && res.Updated == 0 {
		degraded := fmt.Errorf("transmute degraded: scanned=%d nhưng 0 dòng ghi master (skipped=%d rule_misses=%d type_errors=%d) — kiểm tra transform_spec/mapping rules",
			res.Scanned, res.Skipped, res.RuleMisses, res.TypeErrors)
		t.markRuntimeFailure(ctx, masterRow.ID, degraded)
		return res, degraded
	}
	t.markRuntimeSuccess(ctx, masterRow.ID, res)
	return res, nil
}

func (t *TransmuterModule) loadMaster(ctx context.Context, name string) (*masterBindingRuntime, error) {
	var row masterBindingRuntime
	err := t.systemDB.WithContext(ctx).Raw(`
		SELECT mb.id,
		       mb.source_object_id,
		       mb.shadow_binding_id,
		       COALESCE(mc.connection_code, 'default') AS master_connection_key,
		       mb.master_schema,
		       mb.master_table,
		       mb.physical_table_fqn,
		       mb.transform_type,
		       COALESCE(mb.transform_spec, '{}'::jsonb) AS transform_spec,
		       mb.is_active,
		       mb.schema_status,
		       COALESCE(sc.connection_code, 'default') AS shadow_connection_key,
		       sb.shadow_schema,
		       sb.shadow_table,
		       sb.is_active AS shadow_is_active,
		       sor.profile_status AS source_profile_status,
		       COALESCE(sor.primary_key_field, 'id') AS source_pk,
		       '_gpay_id' AS shadow_pk
		  FROM cdc_system.master_binding mb
		  LEFT JOIN cdc_system.connection_registry mc ON mc.id = mb.master_connection_id
		  LEFT JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
		  LEFT JOIN cdc_system.connection_registry sc ON sc.id = sb.shadow_connection_id
		  LEFT JOIN cdc_system.source_object_registry sor ON sor.id = mb.source_object_id
		 WHERE mb.master_table = ?
		 LIMIT 1`, name).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.MasterTable == "" {
		return nil, fmt.Errorf("master %q not registered", name)
	}
	return &row, nil
}

func (t *TransmuterModule) shadowActive(row *masterBindingRuntime) (bool, string) {
	if row == nil {
		return false, "missing master row"
	}
	cacheKey := row.ShadowConnectionKey + "|" + row.ShadowSchema + "|" + row.ShadowTable
	t.mu.Lock()
	if e, ok := t.shadowCache[cacheKey]; ok && time.Since(e.loadedAt) < t.cacheTTL {
		t.mu.Unlock()
		if !e.isActive {
			return false, "is_active=false"
		}
		if e.profileStatus != "active" {
			return false, fmt.Sprintf("profile_status=%s", e.profileStatus)
		}
		return true, ""
	}
	t.mu.Unlock()

	state := shadowState{
		isActive:      row.ShadowIsActive,
		profileStatus: row.SourceProfileStatus,
		loadedAt:      time.Now(),
	}
	t.mu.Lock()
	t.shadowCache[cacheKey] = state
	t.mu.Unlock()
	if !state.isActive {
		return false, "is_active=false"
	}
	if state.profileStatus != "active" {
		return false, fmt.Sprintf("profile_status=%s", state.profileStatus)
	}
	return true, ""
}

func (t *TransmuterModule) loadRules(ctx context.Context, row *masterBindingRuntime) ([]mappingRuleRow, error) {
	cacheKey := fmt.Sprintf("%d|%s", row.ID, row.MasterTable)
	t.mu.Lock()
	if e, ok := t.cache[cacheKey]; ok && time.Since(e.loadedAt) < t.cacheTTL {
		defer t.mu.Unlock()
		return e.rules, nil
	}
	t.mu.Unlock()

	var rules []mappingRuleRow
	err := t.systemDB.WithContext(ctx).Raw(`
		SELECT m.id, ?::bigint AS source_object_id, m.master_binding_id,
		       COALESCE(m.source_field, v2.source_field) AS source_field,
		       m.target_column,
		       COALESCE(m.data_type, v2.data_type)       AS data_type,
		       COALESCE(v2.source_format, '')            AS source_format,
		       COALESCE(m.source_path, v2.source_path)   AS source_path,
		       v2.transform_fn,
		       COALESCE(v2.is_nullable, true)            AS is_nullable,
		       v2.default_value
		  FROM cdc_system.mapping_rule_master m
		  LEFT JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
		 WHERE m.master_binding_id = ?
		   AND m.is_active = true
		   AND m.status = 'approved'
		 ORDER BY m.id`, row.SourceObjectID, row.ID).Scan(&rules).Error
	if err != nil {
		return nil, err
	}
	valid := rules[:0]
	for _, r := range rules {
		if r.TransformFn != nil && !IsTransformWhitelisted(*r.TransformFn) {
			continue
		}
		if !t.typeRes.Validate(r.DataType) {
			continue
		}

		if strings.ContainsRune(r.TargetColumn, '$') {
			t.logger.Warn("loadRules: skipping rule with invalid PG identifier in target_column (contains '$')",
				zap.String("master", row.MasterTable),
				zap.String("target_column", r.TargetColumn),
				zap.String("source_field", r.SourceField),
			)
			continue
		}
		valid = append(valid, r)
	}
	t.mu.Lock()
	t.cache[cacheKey] = cachedRules{rules: valid, loadedAt: time.Now()}
	t.mu.Unlock()
	return valid, nil
}

func (t *TransmuterModule) InvalidateRuleCache(bindingID int64, masterTable string) {
	cacheKey := fmt.Sprintf("%d|%s", bindingID, masterTable)
	t.mu.Lock()
	_, existed := t.cache[cacheKey]
	delete(t.cache, cacheKey)
	t.mu.Unlock()
	t.logger.Info("transmuter rule cache invalidated after DDL apply",
		zap.Int64("master_binding_id", bindingID),
		zap.String("master_table", masterTable),
		zap.Bool("had_entry", existed))
}

func (t *TransmuterModule) fetchShadowBatch(ctx context.Context, row *masterBindingRuntime, cursor int64, onlyIDs []string) ([]shadowBatchRow, error) {
	shadowDB, err := t.connMgr.GetShadowDB(ctx, row.ShadowConnectionKey)
	if err != nil {
		return nil, err
	}
	var rows []shadowBatchRow
	pkIdent := quoteTransmuteIdent(row.ShadowPK)
	pkSelect := pkIdent
	if row.ShadowPK != "_gpay_id" {
		pkSelect = fmt.Sprintf("%s::bigint AS _gpay_id", pkIdent)
	}

	qt := fmt.Sprintf(`SELECT %s, _source_id, _raw_data, _source_ts, _deleted
		FROM %s
		WHERE %s > ?`, pkSelect, quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable),
		fmt.Sprintf("(%s)::bigint", pkIdent))

	args := []any{cursor}
	if len(onlyIDs) > 0 {
		qt += ` AND _source_id IN (?)`
		args = append(args, onlyIDs)
	}
	qt += ` ORDER BY 1 LIMIT ?`
	args = append(args, t.batchSize)
	err = shadowDB.WithContext(ctx).Raw(qt, args...).Scan(&rows).Error
	return rows, err
}
```

---

## 3. `transmuter_extract.go` (NEW)
Chứa logic mapping, transform doc JSON, coerce types, Postgres upsert logic và các generic format converters.

```go
package master

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"centralized-data-service/internal/service/master/transmute"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (t *TransmuterModule) processBatch(ctx context.Context, binding *masterBindingRuntime, rules []mappingRuleRow, rows []shadowBatchRow) batchOutcome {
	out := batchOutcome{}
	strat, ok := transmute.Get(binding.TransformType)
	if !ok {
		t.logger.Error("no transform strategy registered",
			zap.String("master", binding.MasterTable),
			zap.String("transform_type", binding.TransformType))
		out.skipped += int64(len(rows))
		out.scanned += int64(len(rows))
		if n := len(rows); n > 0 {
			out.lastGpayID = rows[n-1].GpayID
		}
		return out
	}
	rc := transmute.RunContext{
		Rules:          toTransmuteRules(rules),
		Spec:           binding.TransformSpec,
		ExtractColumns: t.extractColumnsFn(ctx),
		ExtractArray:   explodeArrayElements,
	}
	for _, row := range rows {
		out.scanned++
		out.lastGpayID = row.GpayID
		emits, st, err := strat.BuildEmits(rc, transmute.ShadowRow{
			Raw:      row.RawData,
			SourceID: row.SourceID,
			SourceTs: row.SourceTs,
			Deleted:  row.Deleted,
		})
		out.ruleMisses += st.RuleMisses
		out.typeErrors += st.TypeErrors
		if err != nil {
			t.logger.Error("transform strategy failed",
				zap.String("master", binding.MasterTable),
				zap.String("transform_type", binding.TransformType),
				zap.String("_source_id", row.SourceID),
				zap.Error(err))
			out.skipped++
			continue
		}
		if len(emits) == 0 {
			out.skipped++
			continue
		}
		for _, e := range emits {
			record := e.Cols
			sourceID := row.SourceID + e.KeySuffix
			record["_gpay_id"] = deterministicGpayID(row.GpayID, e.KeySuffix)
			record["_source_ts"] = row.SourceTs
			record["_deleted"] = row.Deleted

			outcome, err := t.upsertMaster(ctx, binding, record)
			if err != nil {
				t.logger.Error("master upsert failed", zap.String("master", binding.MasterTable), zap.String("_source_id", sourceID), zap.Error(err))
				out.skipped++
				continue
			}
			switch outcome {
			case upsertInserted:
				out.inserted++
			case upsertUpdated:
				out.updated++
			default:
				out.skipped++
			}
		}
	}
	return out
}

func toTransmuteRules(rules []mappingRuleRow) []transmute.MappingRule {
	out := make([]transmute.MappingRule, len(rules))
	for i, r := range rules {
		out[i] = transmute.MappingRule{
			SourceField:  r.SourceField,
			TargetColumn: r.TargetColumn,
			DataType:     r.DataType,
			SourceFormat: r.SourceFormat,
			SourcePath:   r.SourcePath,
			TransformFn:  r.TransformFn,
			IsNullable:   r.IsNullable,
			DefaultValue: r.DefaultValue,
		}
	}
	return out
}

func (t *TransmuterModule) extractColumnsFn(ctx context.Context) transmute.ColumnExtractor {
	return func(raw []byte, rules []transmute.MappingRule) (map[string]any, int64, int64, bool) {
		return t.extractColumns(ctx, raw, rules)
	}
}

func (t *TransmuterModule) extractColumns(ctx context.Context, raw []byte, rules []transmute.MappingRule) (map[string]any, int64, int64, bool) {
	rec := make(map[string]any, len(rules)+4)
	rawStr := string(raw)
	var missCount, typeErrCount int64
	for _, r := range rules {
		path := r.SourceField
		if r.SourcePath != nil && *r.SourcePath != "" {
			path = *r.SourcePath
		} else if r.SourceFormat == "debezium_after" {
			path = "after." + r.SourceField
		}
		gres := gjson.Get(rawStr, path)
		if !gres.Exists() {
			if r.IsNullable {
				rec[r.TargetColumn] = nil
			} else if r.DefaultValue != nil {
				rec[r.TargetColumn] = *r.DefaultValue
			} else {
				missCount++
				return nil, missCount, typeErrCount, false
			}
			continue
		}
		val := gjsonValueToGo(gres)
		val = unwrapMongoExtJSON(val)
		if r.TransformFn != nil && *r.TransformFn != "" {
			converted, err := ApplyTransform(*r.TransformFn, val)
			if err != nil {
				typeErrCount++
				if r.IsNullable {
					rec[r.TargetColumn] = nil
					continue
				}
				return nil, missCount, typeErrCount, false
			}
			val = converted
		}
		if violation := t.typeRes.ValidateValue(ctx, r.DataType, val); violation != "" {
			typeErrCount++
			if r.IsNullable {
				rec[r.TargetColumn] = nil
				continue
			}
			return nil, missCount, typeErrCount, false
		}
		rec[r.TargetColumn] = coerceForColumn(val, r.DataType)
	}
	return rec, missCount, typeErrCount, true
}

const (
	upsertSkipped  = 0
	upsertInserted = 1
	upsertUpdated  = 2
)

func (t *TransmuterModule) upsertMaster(ctx context.Context, binding *masterBindingRuntime, record map[string]any) (int, error) {
	masterDB, err := t.connMgr.GetMasterDB(ctx, binding.MasterConnectionKey)
	if err != nil {
		return upsertSkipped, err
	}
	keys := sortedKeysAny(record)
	cols := make([]string, len(keys))
	placeholders := make([]string, len(keys))
	values := make([]any, len(keys))
	sets := make([]string, 0, len(keys))
	for i, k := range keys {
		cols[i] = quoteTransmuteIdent(k)
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = sqlBindValueTransmute(record[k])
		if k == "_gpay_id" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", quoteTransmuteIdent(k), quoteTransmuteIdent(k)))
	}
	sets = append(sets, `"_updated_at" = NOW()`)
	qt := quoteTransmuteQualified(binding.MasterSchema, binding.MasterTable)
	sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)
ON CONFLICT (_gpay_id) DO UPDATE SET %s
WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0)
RETURNING (xmax = 0) AS inserted`,
		qt,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(sets, ", "),
		qt,
	)
	var rows []struct {
		Inserted bool `gorm:"column:inserted"`
	}
	if err := masterDB.WithContext(ctx).Raw(sqlText, values...).Scan(&rows).Error; err != nil {
		return upsertSkipped, err
	}
	if len(rows) == 0 {
		return upsertSkipped, nil
	}
	if rows[0].Inserted {
		return upsertInserted, nil
	}
	return upsertUpdated, nil
}

func gjsonValueToGo(r gjson.Result) any {
	switch r.Type {
	case gjson.Null:
		return nil
	case gjson.False:
		return false
	case gjson.True:
		return true
	case gjson.Number:
		if r.Raw != "" && strings.ContainsAny(r.Raw, ".eE") {
			return r.Float()
		}
		return r.Int()
	case gjson.String:
		return r.String()
	case gjson.JSON:
		var v any
		_ = json.Unmarshal([]byte(r.Raw), &v)
		return v
	}
	return r.Value()
}

func unwrapMongoExtJSON(v any) any {
	m, ok := v.(map[string]any)
	if !ok || len(m) != 1 {
		return v
	}
	if s, ok := m["$oid"].(string); ok {
		return s
	}
	if s, ok := m["$numberDecimal"].(string); ok {
		return s
	}
	if n, ok := m["$numberLong"]; ok {
		return mongoNumberToGo(n)
	}
	if n, ok := m["$numberInt"]; ok {
		return mongoNumberToGo(n)
	}
	if n, ok := m["$numberDouble"]; ok {
		if s, ok := n.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
		return n
	}
	if d, ok := m["$date"]; ok {
		switch dv := d.(type) {
		case string:
			return dv
		case float64:
			return time.UnixMilli(int64(dv)).UTC()
		case map[string]any:
			if nl, ok := dv["$numberLong"].(string); ok {
				if ms, err := strconv.ParseInt(nl, 10, 64); err == nil {
					return time.UnixMilli(ms).UTC()
				}
			}
		}
	}
	return v
}

func mongoNumberToGo(n any) any {
	if s, ok := n.(string); ok {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		return s
	}
	return n
}

func isJSONColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return dt == "json" || dt == "jsonb" || strings.HasPrefix(dt, "json ") || strings.HasPrefix(dt, "jsonb ")
}

func coerceForColumn(v any, dataType string) any {
	if v == nil {
		return nil
	}
	if isTimestampColumnType(dataType) {
		switch n := v.(type) {
		case int64:
			return epochToTime(n)
		case float64:
			return epochToTime(int64(n))
		}
		return v
	}
	_, isMap := v.(map[string]any)
	_, isSlice := v.([]any)
	composite := isMap || isSlice
	if isJSONColumnType(dataType) || composite {
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return v
}

func isTimestampColumnType(dataType string) bool {
	dt := strings.ToLower(strings.TrimSpace(dataType))
	return strings.HasPrefix(dt, "timestamp") || dt == "datetime"
}

func epochToTime(n int64) time.Time {
	if n >= 1_000_000_000_000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}

func deterministicGpayID(shadowGpayID int64, keySuffix string) int64 {
	if keySuffix == "" {
		return shadowGpayID
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
	_, _ = h.Write([]byte(keySuffix))
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}

func sortedKeysAny(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func sqlBindValueTransmute(v any) any {
	switch v.(type) {
	case map[string]any, []any, []map[string]any:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
	return v
}

func quoteTransmuteIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func quoteTransmuteQualified(schemaName, tableName string) string {
	return quoteTransmuteIdent(schemaName) + "." + quoteTransmuteIdent(tableName)
}
```
