# Giải pháp kỹ thuật: Phân tách scan_handler.go (Gom nhóm theo Flow)

Để đảm bảo code dễ bảo trì, chúng tôi đề xuất phân tách file `scan_handler.go` (766 dòng) thành đúng **3 file** lớn dựa trên flow logic nghiệp vụ:

---

## 1. `scan_handler.go` (MODIFY)
Chứa core struct `ScanHandler`, constructor, setters và các static/generic helpers liên quan đến parse path/flatten dữ liệu.

```go
package recon

import (
	"encoding/json"
	"fmt"
	"strings"

	"centralized-data-service/internal/handler/base"
	repomaster "centralized-data-service/internal/repository/master"
	"centralized-data-service/internal/service/metadata"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScanHandler struct {
	base.BaseHandler
	mappingV2Repo      *repomaster.MappingRuleV2Repo
	shadowDB           *gorm.DB
	transformChunkSize int
	metadataRegistry   metadata.MetadataRegistry
	registryRepo       metadata.RegistryResolver
}

func NewScanHandler(
	db *gorm.DB,
	natsConn *nats.Conn,
	logger *zap.Logger,
	mappingV2Repo *repomaster.MappingRuleV2Repo,
	shadowDB *gorm.DB,
	metadataRegistry metadata.MetadataRegistry,
	registryRepo metadata.RegistryResolver,
) *ScanHandler {
	return &ScanHandler{
		BaseHandler:        base.NewBaseHandler(db, natsConn, logger),
		mappingV2Repo:      mappingV2Repo,
		shadowDB:           shadowDB,
		transformChunkSize: 1000,
		metadataRegistry:   metadataRegistry,
		registryRepo:       registryRepo,
	}
}

func (h *ScanHandler) SetTransformChunkSize(n int) {
	h.transformChunkSize = n
}

func (h *ScanHandler) SetMetadataRegistry(m metadata.MetadataRegistry) {
	h.metadataRegistry = m
}

func (h *ScanHandler) SetRegistryResolver(r metadata.RegistryResolver) {
	h.registryRepo = r
}

// --- Generic / Static Helpers ---

func validScanIdent(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

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

	return "{" + strings.Join(out, ",") + "}", isArrayExplode, nil
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
		trimmed := strings.TrimSpace(v)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				flattenJSONWithTypes(prefix, parsed, result)
				return
			}
		}
		result[prefix] = "TEXT"
	case nil:
		if _, exists := result[prefix]; !exists {
			result[prefix] = "TEXT"
		}
	default:
		result[prefix] = "TEXT"
	}
}
```

---

## 2. `scan_handler_backfill.go` (NEW)
Chứa duy nhất flow logic và handler xử lý backfill dữ liệu cho shadow columns.

```go
package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"centralized-data-service/internal/handler/base"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/pkgs/sqlutil"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// HandleBackfill — subscribe "cdc.cmd.backfill"
func (h *ScanHandler) HandleBackfill(msg *nats.Msg) {
	var payload struct {
		TargetTable string `json:"target_table"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.PublishResult(context.Background(), msg, base.CommandResult{Command: "backfill", Status: "error", Error: "invalid payload"})
		return
	}

	h.Logger.Info("backfilling table", zap.String("table", payload.TargetTable), zap.Int("limit", payload.Limit))
	schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, payload.TargetTable)

	ctx := context.Background()

	if !h.TableExistsInSchema(ctx, h.shadowDB, schemaName, payload.TargetTable) {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "backfill", TargetTable: payload.TargetTable, Status: "skipped", Error: "table does not exist"})
		return
	}
	if !h.HasColumnInSchema(ctx, h.shadowDB, schemaName, payload.TargetTable, "_raw_data") {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "backfill", TargetTable: payload.TargetTable, Status: "skipped", Error: "table has no _raw_data column yet"})
		return
	}

	reg := metadata.ResolveTargetTableConfig(ctx, h.metadataRegistry, h.registryRepo, payload.TargetTable)
	var sourceTable string
	if reg != nil {
		sourceTable = reg.SourceTable
	} else {
		sourceTable = payload.TargetTable
	}

	rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(ctx, sourceTable)
	if err != nil || len(rules) == 0 {
		h.PublishResult(ctx, msg, base.CommandResult{
			Command:     "backfill",
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       fmt.Sprintf("no active mapping rules for table %s (source: %s)", payload.TargetTable, sourceTable),
		})
		return
	}

	var setClauses []string
	var whereClauses []string
	seenCols := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		colKey := strings.ToLower(strings.TrimSpace(rule.TargetColumn))
		if _, dup := seenCols[colKey]; dup {
			continue
		}
		seenCols[colKey] = struct{}{}
		castExpr := metadata.BuildCastExpr(rule.SourceField, rule.DataType)
		quotedCol := sqlutil.QuoteIdent(rule.TargetColumn)
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quotedCol))
	}

	if len(setClauses) == 0 {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "backfill", TargetTable: payload.TargetTable, Status: "success", Error: "no active rules to backfill"})
		return
	}

	setClauses = append(setClauses, "_updated_at = NOW()")

	execDB := h.DB
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}

	limitClause := ""
	if payload.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", payload.Limit)
	}

	quotedTable := sqlutil.QualifiedTable(schemaName, payload.TargetTable)
	whereExpr := strings.Join(whereClauses, " OR ")
	setExpr := strings.Join(setClauses, ", ")

	backfillSQL := fmt.Sprintf(
		`UPDATE %[1]s SET %[2]s
		 WHERE _gpay_id IN (
			SELECT _gpay_id FROM %[1]s
			WHERE _raw_data IS NOT NULL AND (%[3]s)
			%[4]s
		 )`,
		quotedTable, setExpr, whereExpr, limitClause,
	)

	result := execDB.Exec(backfillSQL)
	if result.Error != nil {
		h.PublishResult(ctx, msg, base.CommandResult{
			Command:     "backfill",
			TargetTable: payload.TargetTable,
			Status:      "error",
			Error:       result.Error.Error(),
		})
		return
	}

	h.Logger.Info("backfill completed", zap.String("table", payload.TargetTable), zap.Int64("rows_affected", result.RowsAffected))
	h.PublishResult(ctx, msg, base.CommandResult{
		Command:      "backfill",
		TargetTable:  payload.TargetTable,
		RowsAffected: int(result.RowsAffected),
		Status:       "success",
	})
}
```

---

## 3. `scan_handler_discover.go` (NEW)
Gom các luồng quét phát hiện schema trường động mới và quét định kỳ.

```go
package recon

import (
	"centralized-data-service/internal/handler/base"
	mastermodel "centralized-data-service/internal/model/master"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/pkgs/sqlutil"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HandleScanRawData — subscribe "cdc.cmd.scan-raw-data"
func (h *ScanHandler) HandleScanRawData(msg *nats.Msg) {
	var payload struct {
		TargetTable string `json:"target_table"`
		Limit       int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		payload.TargetTable = string(msg.Data)
	}

	if !validScanIdent(payload.TargetTable) {
		h.PublishResult(context.Background(), msg, base.CommandResult{Command: "scan-raw-data", TargetTable: payload.TargetTable, Status: "error", Error: "invalid table name"})
		return
	}

	h.Logger.Info("scanning _raw_data JSONB", zap.String("table", payload.TargetTable))
	schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, payload.TargetTable)

	ctx := context.Background()

	if !h.TableExistsInSchema(ctx, h.shadowDB, schemaName, payload.TargetTable) {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "scan-raw-data", TargetTable: payload.TargetTable, Status: "skipped", Error: "table does not exist"})
		return
	}
	if !h.HasColumnInSchema(ctx, h.shadowDB, schemaName, payload.TargetTable, "_raw_data") {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "scan-raw-data", TargetTable: payload.TargetTable, Status: "skipped", Error: "table has no _raw_data column yet"})
		return
	}
	execDB := h.DB
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}

	var rawFields []struct {
		Key  string
		Type string
	}
	sqlQuery := fmt.Sprintf(
		`SELECT key, MAX(type) as type FROM (
			SELECT key, jsonb_typeof(_raw_data->key) as type
			FROM %s, jsonb_object_keys(_raw_data) as key
			WHERE _raw_data IS NOT NULL AND _raw_data != '{}'::jsonb
			LIMIT 1000
		) sub WHERE type != 'null' GROUP BY key ORDER BY key`, sqlutil.QualifiedTable(schemaName, payload.TargetTable))

	if err := execDB.Raw(sqlQuery).Scan(&rawFields).Error; err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "scan-raw-data", TargetTable: payload.TargetTable, Status: "error", Error: err.Error()})
		return
	}

	var binding struct {
		ID             int64
		SourceObjectID int64
	}
	h.DB.Table("cdc_system.shadow_binding").Select("id, source_object_id").Where("shadow_table = ?", payload.TargetTable).Limit(1).Scan(&binding)

	var existingV2 []struct {
		SourceField string
	}
	if binding.ID > 0 {
		h.DB.Table("cdc_system.mapping_rule_v2").
			Select("source_field").
			Where("shadow_binding_id = ?", binding.ID).
			Scan(&existingV2)
	} else if binding.SourceObjectID > 0 {
		h.DB.Table("cdc_system.mapping_rule_v2").
			Select("source_field").
			Where("source_object_id = ?", binding.SourceObjectID).
			Scan(&existingV2)
	}

	mappedFields := make(map[string]bool)
	for _, r := range existingV2 {
		mappedFields[r.SourceField] = true
	}

	skipFields := map[string]bool{
		"_id": true, "_raw_data": true, "_source": true, "_synced_at": true,
		"_version": true, "_hash": true, "_deleted": true, "_created_at": true, "_updated_at": true,
		"id": true, "_debezium_ab_id": true, "_debezium_emitted_at": true, "_debezium_extracted_at": true, "_debezium_meta": true,
	}

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
				rule := mastermodel.MappingRuleV2{
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
				h.DB.Create(&rule)
			}
		}
	}

	res := map[string]interface{}{
		"status":         "ok",
		"table":          payload.TargetTable,
		"source_table":   payload.TargetTable,
		"total_raw_keys": len(rawFields),
		"mapped_count":   len(existingV2),
		"new_fields":     unmappedFields,
	}
	res = base.SanitizeAdminResultMap(res)
	resBytes, _ := json.Marshal(res)
	h.NatsPublish(msg, "cdc.result.scan-raw-data", resBytes)

	if h.OnPublishResult != nil {
		h.OnPublishResult(ctx, base.CommandResult{
			Command:      "scan-raw-data",
			TargetTable:  payload.TargetTable,
			RowsAffected: len(unmappedFields),
			Status:       "success",
		})
	}

	h.Logger.Info("_raw_data scan completed",
		zap.String("table", payload.TargetTable),
		zap.Int("raw_keys", len(rawFields)),
		zap.Int("unmapped", len(unmappedFields)),
	)
}

// HandleScanArrayFields — subscribe "cdc.cmd.scan-array"
func (h *ScanHandler) HandleScanArrayFields(msg *nats.Msg) {
	var payload struct {
		TargetTable     string `json:"target_table"`
		ExplodePath     string `json:"explode_path"`
		MasterBindingID *int64 `json:"master_binding_id"`
		ReplyTo         string `json:"reply_to"`
		Mode            string `json:"mode"` // "sample" hoặc "full"
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		replySubject := msg.Reply
		if replySubject == "" {
			replySubject = "cdc.result.scan-array"
		}
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "bad_json"})
		h.NatsPublish(msg, replySubject, res)
		return
	}
	h.Logger.Info("cmd: scan_array_fields", zap.Any("payload", payload))
	replySubject := msg.Reply
	if replySubject == "" && payload.ReplyTo != "" {
		replySubject = payload.ReplyTo
	} else if replySubject == "" {
		replySubject = "cdc.result.scan-array"
	}

	if payload.Mode == "" {
		payload.Mode = "sample"
	}

	targetTable := payload.TargetTable
	schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, targetTable)

	ctx := context.Background()

	if payload.MasterBindingID != nil {
		var sb struct {
			ShadowSchema string `gorm:"column:shadow_schema"`
			ShadowTable  string `gorm:"column:shadow_table"`
		}
		if err := h.DB.Raw(`
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

	execDB := h.DB
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}

	if !h.TableExistsInSchema(ctx, execDB, schemaName, targetTable) || !h.HasColumnInSchema(ctx, execDB, schemaName, targetTable, "_raw_data") {
		res, _ := json.Marshal(map[string]interface{}{"status": "skipped", "reason": "table or _raw_data column not found"})
		h.NatsPublish(msg, replySubject, res)
		return
	}

	pgPath, isArrayExplode, err := explodePathToPGPath(payload.ExplodePath)
	if err != nil {
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "invalid_explode_path: " + err.Error()})
		h.NatsPublish(msg, replySubject, res)
		return
	}

	var pathType string
	errPathType := execDB.Raw(fmt.Sprintf(
		`SELECT jsonb_typeof(_raw_data #> ?::text[]) FROM %s WHERE _raw_data #> ?::text[] IS NOT NULL LIMIT 1`,
		sqlutil.QualifiedTable(schemaName, targetTable)), pgPath, pgPath,
	).Scan(&pathType).Error

	if errPathType != nil {
		res, _ := json.Marshal(map[string]interface{}{"status": "error", "error": "db_query_error: " + errPathType.Error()})
		h.NatsPublish(msg, replySubject, res)
		return
	}

	if pathType == "string" {
		res, _ := json.Marshal(map[string]interface{}{"status": "skipped", "reason": "Dữ liệu đang là 'string' (stringified JSON)."})
		h.NatsPublish(msg, replySubject, res)
		return
	}

	isObject := pathType == "object"
	basePath := strings.TrimSuffix(strings.TrimSpace(payload.ExplodePath), "[*]")
	basePath = strings.TrimSuffix(basePath, ".")

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
		sqlutil.QualifiedTable(schemaName, targetTable), sampleClause, limitClause)

	rows, err := execDB.Raw(sqlQuery, pgPath, pgPath, pgPath).Rows()
	if err != nil {
		h.Logger.Error("Lỗi stream data đệ quy", zap.Error(err))
		return
	}
	defer rows.Close()

	discoveredFields := make(map[string]string)
	var scannedCount int64

	for rows.Next() {
		var nodeData sql.NullString
		if err := rows.Scan(&nodeData); err != nil {
			h.Logger.Error("Lỗi Scan SQL row", zap.Error(err))
			continue
		}

		if nodeData.Valid && nodeData.String != "" && nodeData.String != "{}" {
			var parsed interface{}
			if err := json.Unmarshal([]byte(nodeData.String), &parsed); err != nil {
				h.Logger.Error("Lỗi parse JSON", zap.Error(err), zap.String("data", nodeData.String))
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

	h.Logger.Info("DEBUG - KẾT QUẢ ĐỆ QUY GO",
		zap.Int("rows_scanned_success", int(scannedCount)),
		zap.Any("discovered_fields", discoveredFields))

	var binding struct {
		ID             int64
		SourceObjectID int64
	}
	h.DB.Table("cdc_system.shadow_binding").Select("id, source_object_id").
		Where("shadow_table = ?", targetTable).Limit(1).Scan(&binding)

	existing := make(map[string]bool)
	if isObject && payload.MasterBindingID != nil {
		var existingRows []struct{ TargetColumn string }
		h.DB.Table("cdc_system.mapping_rule_master").Select("target_column").
			Where("master_binding_id = ?", *payload.MasterBindingID).Scan(&existingRows)
		for _, r := range existingRows {
			existing[r.TargetColumn] = true
		}
	} else {
		var existingRows []struct{ SourceField string }
		q := h.DB.Table("cdc_system.mapping_rule_v2").Select("source_field")
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
	var rulesToInsert []mastermodel.MappingRuleV2
	type masterNestedRow struct{ target, srcField, srcPath, dataType string }
	var masterRows []masterNestedRow
	var newRules []CreatedRuleDTO

	for relPath, pgType := range discoveredFields {
		if skipFields[relPath] {
			continue
		}

		if isObject && payload.MasterBindingID != nil {
			objTarget := strings.ReplaceAll(basePath+"."+relPath, ".", "_")
			objTarget = strings.ReplaceAll(objTarget, "[*]", "")

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
			rulesToInsert = append(rulesToInsert, mastermodel.MappingRuleV2{
				SourceObjectID:  binding.SourceObjectID,
				ShadowBindingID: &binding.ID,
				MasterBindingID: payload.MasterBindingID,
				SourceField:     relPath,
				TargetColumn:    strings.ReplaceAll(relPath, ".", "_"),
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

	errTx := h.DB.Transaction(func(tx *gorm.DB) error {
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
		h.Logger.Error("Lỗi insert rules đệ quy", zap.Error(errTx))
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

	res = base.SanitizeAdminResultMap(res)
	resBytes, _ := json.Marshal(res)
	h.NatsPublish(msg, replySubject, resBytes)

	if h.OnPublishResult != nil {
		h.OnPublishResult(ctx, base.CommandResult{
			Command:      "scan-array-fields",
			TargetTable:  targetTable,
			RowsAffected: len(newFields),
			Status:       "success",
		})
	}

	h.Logger.Info("scan-array-recursive completed",
		zap.String("table", targetTable),
		zap.Int("discovered", len(discoveredFields)),
		zap.Int("inserted", len(newFields)))
}

// HandlePeriodicScan — subscribe "cdc.cmd.scan-periodic"
func (h *ScanHandler) HandlePeriodicScan(msg *nats.Msg) {
	h.Logger.Info("periodic scan triggered")
	if h.metadataRegistry == nil {
		h.Logger.Warn("periodic scan: metadata registry is nil")
		return
	}
	entries := h.metadataRegistry.ListTableConfigs()
	if len(entries) == 0 {
		h.Logger.Warn("periodic scan: no active table configs available")
		return
	}

	execDB := h.DB
	if h.shadowDB != nil {
		execDB = h.shadowDB
	}

	ctx := context.Background()
	totalScanned := 0

	for _, entry := range entries {
		if !entry.IsActive {
			continue
		}
		schemaName := metadata.ResolveTargetSchema(h.metadataRegistry, entry.TargetTable)
		if !h.TableExistsInSchema(ctx, execDB, schemaName, entry.TargetTable) || !h.HasColumnInSchema(ctx, execDB, schemaName, entry.TargetTable, "_raw_data") {
			continue
		}

		h.Logger.Info("periodic scan: checking table", zap.String("table", entry.TargetTable))
		subMsg := &nats.Msg{
			Subject: "cdc.cmd.scan-raw-data",
			Data:    []byte(fmt.Sprintf(`{"target_table":"%s","limit":50}`, entry.TargetTable)),
		}
		h.HandleScanRawData(subMsg)
		totalScanned++
	}

	h.Logger.Info("periodic scan completed", zap.Int("tables_scanned", totalScanned))
}
```
