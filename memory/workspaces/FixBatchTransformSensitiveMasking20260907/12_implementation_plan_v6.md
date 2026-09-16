# 12_implementation_plan_v6 — Sensitive Masking Scale 50M–500M
**Workspace:** FixBatchTransformSensitiveMasking20260907
**Phiên:** 2026-09-08 | **Status:** Chờ APPROVE

---

## So sánh v5 → v6 (6 điểm sửa)

| # | Lỗi v5 | Fix v6 |
|---|---|---|
| Bug 1 | Chunked error path gọi `publishTransmuteTrigger` → leak data unmasked | Xóa trigger khỏi error block |
| Bug 2 | Sensitive-only path dùng `finishJob` trực tiếp, thiếu Activity Logger, thiếu consistency | Dùng `publishAndFinishJob` + Activity Logger; fail-fast PK check |
| Bug 3 | VALUES type inference: nil → "could not determine data type"; UUID PK → "operator does not exist" | `::text` cast cho first-row cols; `detectPrimaryKeyType` + `v.pk::pkType` trong WHERE |
| Design 1 | Dirty state: bulk SQL chạy thành công, sensitive fail → data ghi dở dang | Fail-fast TRƯỚC bulk SQL nếu `sensitiveRules > 0 && pkCol == ""` |
| Design 2 | Double-count: 50M bulk + 50M sensitive = 100M báo về job (200%) | `finalAffected = max(totalRows, sensitiveUpdated)` |
| Design 3 | Unchunked path vẫn có sensitive masking block (unreachable sau fail-fast) | Xóa block unreachable, simplify |

---

## Files thay đổi

| File | Action | Lines ± |
|---|---|---|
| `internal/handler/shadow/batch_transform_handler.go` | Sửa + thêm hàm | +~200 |
| `internal/server/server_setup.go` | +1 dòng | +1 |
| `internal/service/metadata/mapping_utils.go` | Không sửa | 0 |

---

## PHẦN 1 — Import block (thay thế lines 3–22)

```go
import (
	"centralized-data-service/internal/handler/base"
	mastermodel "centralized-data-service/internal/model/master"
	repomaster "centralized-data-service/internal/repository/master"
	"centralized-data-service/internal/repository"
	"centralized-data-service/internal/service/governance"
	"centralized-data-service/internal/service/metadata"
	"centralized-data-service/pkgs/observability"
	"centralized-data-service/pkgs/sqlutil"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"       // zero-alloc JSON field extraction — gjson v1.18.0 có sẵn go.mod
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

> - `encoding/json` giữ nguyên — dùng cho NATS payload unmarshal & publishTransmuteTrigger.
> - Không thêm `sync`, `runtime`, `errgroup`.

---

## PHẦN 2 — Package-level type (thêm sau imports, trước line 24)

```go
// maskedRowData giữ PK native type và các giá trị đã masked cho bulkUpdateMasked.
// Native PK type (int64/string/[16]byte) được giữ nguyên để pgx bind đúng type.
type maskedRowData struct {
	pk     interface{}
	values []interface{} // len = len(sensitiveRules), thứ tự tương ứng với rules
}
```

---

## PHẦN 3 — Struct (thay thế lines 39–48)

```go
type BatchTransformHandler struct {
	base.BaseHandler
	mappingV2Repo      *repomaster.MappingRuleV2Repo
	shadowDB           *gorm.DB
	transformChunkSize int
	sensitiveBatchSize int                       // batch size cho sensitive masking, default 2000
	metadataRegistry   metadata.MetadataRegistry
	registryRepo       metadata.RegistryResolver
	transformJobRepo   *repository.TransformJobRepo
	hmacKey            string
	maskingSvc         *governance.MaskingService // Go-level HMAC/AES masking
}
```

---

## PHẦN 4 — Setters (thêm sau SetTransformJobRepo, trước HandleBatchTransform)

```go
func (h *BatchTransformHandler) SetMaskingService(svc *governance.MaskingService) {
	h.maskingSvc = svc
}

func (h *BatchTransformHandler) SetSensitiveBatchSize(n int) {
	if n > 0 {
		h.sensitiveBatchSize = n
	}
}

// getSensitiveBatchSize clamp batchSize để không vượt 65535 PostgreSQL protocol params.
// numCols = len(sensitiveRules) + 1 (1 cho pk trong VALUES list)
func (h *BatchTransformHandler) getSensitiveBatchSize(numCols int) int {
	base := h.sensitiveBatchSize
	if base <= 0 {
		base = 2000
	}
	// 60000 < 65535 — buffer an toàn tránh edge case
	if maxSafe := 60000 / numCols; base > maxSafe {
		return maxSafe
	}
	return base
}
```

---

## PHẦN 5 — For loop trong runTransformJob (thay thế lines 195–235)

Không thay đổi so với v5.

```go
var setClauses []string
var whereClauses []string
var sensitiveRules []mastermodel.MappingRuleV2
seenCols := make(map[string]struct{}, len(rules))

for _, rule := range rules {
	if !rule.IsActive {
		continue
	}
	colKey := strings.ToLower(strings.TrimSpace(rule.TargetColumn))
	if _, dup := seenCols[colKey]; dup {
		observability.Ctx(ctx, h.Logger).Warn("batch transform: duplicate target_column in mapping rules, skipping rule",
			zap.String("table", targetTable),
			zap.String("target_column", rule.TargetColumn),
			zap.String("source_field", rule.SourceField),
		)
		continue
	}
	seenCols[colKey] = struct{}{}
	if !h.HasColumnInSchema(ctx, execDB, schemaName, pureTable, rule.TargetColumn) {
		observability.Ctx(ctx, h.Logger).Warn("batch transform: target_column does not exist in db, skipping rule",
			zap.String("table", targetTable),
			zap.String("target_column", rule.TargetColumn),
			zap.String("source_field", rule.SourceField),
		)
		continue
	}
	// Force-mode filter TRƯỚC sensitive check
	if payload.Force {
		if _, inForce := forceSet[colKey]; !inForce {
			continue
		}
	}
	// Sensitive field: tách sang Go-level masking
	if rule.IsSensitiveField && h.maskingSvc != nil {
		strategy := strings.ToLower(strings.TrimSpace(rule.MaskStrategy))
		if strategy != "" && strategy != "none" {
			sensitiveRules = append(sensitiveRules, rule)
			continue
		}
	}
	// Non-sensitive: SQL expression
	castExpr := metadata.BuildCastExprWithRule(rule, h.hmacKey)
	quotedCol := sqlutil.QuoteIdent(rule.TargetColumn)
	if payload.Force {
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
	} else {
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quotedCol))
	}
}
```

---

## PHẦN 6 — Post-loop block (thay thế lines 237–251)

**[FIX Bug 2]** Sensitive-only path: dùng `publishAndFinishJob` + Activity Logger. Fail-fast PK check.

```go
if len(setClauses) == 0 && len(sensitiveRules) == 0 {
	h.publishAndFinishJob(ctx, jobID, "success", 0, 0, "no active rules to transform", rules)
	return
}

// QUAN TRỌNG: tính hasNonSensitiveRules TRƯỚC KHI append _updated_at
hasNonSensitiveRules := len(setClauses) > 0

setClauses = append(setClauses, "_updated_at = NOW()")
quotedTable := sqlutil.QualifiedTable(schemaName, pureTable)
var whereExpr string
if payload.Force {
	whereExpr = "TRUE"
} else {
	whereExpr = strings.Join(whereClauses, " OR ")
}
setExpr := strings.Join(setClauses, ", ")

// sensitiveWhere: WHERE riêng cho sensitive phase, độc lập với whereExpr
var sensitiveWhere string
if payload.Force {
	sensitiveWhere = "TRUE"
} else if len(sensitiveRules) > 0 {
	sc := make([]string, 0, len(sensitiveRules))
	for _, sr := range sensitiveRules {
		sc = append(sc, fmt.Sprintf("%s IS NULL", sqlutil.QuoteIdent(sr.TargetColumn)))
	}
	sensitiveWhere = strings.Join(sc, " OR ")
}

// Sensitive-only path
if !hasNonSensitiveRules {
	// [FIX Bug 2] Fail-fast PK check trước khi chạy bất kỳ thứ gì
	pkColOnly, pkErrOnly := h.detectPrimaryKey(execDB, schemaName, pureTable)
	if pkErrOnly != nil || pkColOnly == "" {
		errStr := fmt.Sprintf("sensitive masking: table %s.%s requires a primary key", schemaName, pureTable)
		observability.Ctx(ctx, h.Logger).Error(errStr, zap.Error(pkErrOnly))
		h.publishAndFinishJob(ctx, jobID, "error", 0, 0, errStr, rules)
		return
	}
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkColOnly, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive-only masking failed",
			zap.String("table", targetTable), zap.Error(err))
		h.publishAndFinishJob(ctx, jobID, "error", 0, 0, err.Error(), rules)
		return
	}
	// [FIX Bug 2] Activity Logger cho sensitive-only success (nhất quán với other paths)
	if h.DB != nil {
		act := governance.NewActivityLogger(h.DB, h.Logger)
		act.Quick("cmd-batch-transform", targetTable, "nats-command", "success",
			n, map[string]interface{}{"trace_id": traceID, "mode": "sensitive-only"}, "")
	}
	h.publishAndFinishJob(ctx, jobID, "success", n, n, "", rules)
	h.publishTransmuteTrigger(ctx, schemaName, pureTable)
	return
}
```

---

## PHẦN 7 — Fail-fast + Unchunked path (thay thế lines 273–311)

**[FIX Design 1]** Fail-fast TRƯỚC bulk SQL. **[FIX Design 3]** Unchunked path simplified (sensitiveRules guaranteed empty).

```go
pkCol, pkErr := h.detectPrimaryKey(execDB, schemaName, pureTable)
chunkSize := h.transformChunkSize
if chunkSize <= 0 {
	chunkSize = 1000
}

// [FIX Design 1] Fail-fast: sensitive masking yêu cầu PK.
// Phải check TRƯỚC khi bulk SQL chạy để tránh dirty state:
// (non-sensitive đã UPDATE xong nhưng sensitive chưa mask → data ghi dở dang)
if len(sensitiveRules) > 0 && (pkErr != nil || pkCol == "") {
	observability.Ctx(ctx, h.Logger).Error("batch transform: sensitive masking requires primary key — aborting before bulk SQL",
		zap.String("table", targetTable), zap.Error(pkErr))
	h.publishAndFinishJob(ctx, jobID, "error", 0, totalPendingRows,
		fmt.Sprintf("sensitive masking: table %s.%s requires a primary key", schemaName, pureTable), rules)
	return
}

if pkErr != nil || pkCol == "" {
	// Unchunked fallback — bảng không có PK
	observability.Ctx(ctx, h.Logger).Warn("batch transform: PK not detected, running unchunked UPDATE",
		zap.String("table", targetTable),
		zap.Error(pkErr),
	)
	transformSQL := fmt.Sprintf(`UPDATE %s SET %s WHERE _raw_data IS NOT NULL AND (%s)`,
		quotedTable, setExpr, whereExpr,
	)
	result := execDB.Exec(transformSQL)
	if result.Error != nil {
		enrichedErr := h.enrichCastingError(result.Error, rules)
		observability.Ctx(ctx, h.Logger).Error("batch transform query failed",
			zap.String("table", targetTable),
			zap.Error(result.Error),
			zap.String("enriched_error", enrichedErr),
		)
		h.publishAndFinishJob(ctx, jobID, "error", 0, totalPendingRows, enrichedErr, rules)
		return
	}
	observability.Ctx(ctx, h.Logger).Info("batch transform completed (unchunked)",
		zap.String("table", targetTable),
		zap.Int64("rows_affected", result.RowsAffected),
	)
	if h.DB != nil {
		act := governance.NewActivityLogger(h.DB, h.Logger)
		act.Quick("cmd-batch-transform", targetTable, "nats-command", "success",
			result.RowsAffected, map[string]interface{}{"trace_id": traceID}, "")
	}
	// [FIX Design 3] sensitiveRules guaranteed empty tại đây (fail-fast ở trên đã xử lý)
	// Không cần gọi runSensitiveMasking
	h.finishJob(ctx, jobID, "COMPLETED", result.RowsAffected, totalPendingRows, "")
	h.publishTransmuteTrigger(ctx, schemaName, pureTable)
	return
}
```

---

## PHẦN 8 — Chunked success path (thay thế lines 442–448)

**[FIX Bug 1]** Xóa `publishTransmuteTrigger` khỏi error block.
**[FIX Design 2]** `finalAffected = max(totalRows, sensitiveUpdated)` tránh double-count.

```go
observability.Ctx(ctx, h.Logger).Info("batch transform completed (chunked)",
	zap.String("table", targetTable),
	zap.String("pk", pkCol),
	zap.Int("chunk_size", chunkSize),
	zap.Int("iterations", productiveIters),
	zap.Int64("rows_affected", totalRows),
	zap.Int64("total_pending", totalPendingRows),
)
if h.DB != nil {
	act := governance.NewActivityLogger(h.DB, h.Logger)
	act.Quick("cmd-batch-transform", targetTable, "nats-command", "success",
		totalRows, map[string]interface{}{"trace_id": traceID, "iterations": productiveIters}, "")
}

// [FIX Bug 1 + Design 2] Sensitive masking TRƯỚC finishJob
var sensitiveUpdated int64
if len(sensitiveRules) > 0 && h.maskingSvc != nil {
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive masking failed (chunked)",
			zap.String("table", targetTable), zap.Error(err))
		h.publishAndFinishJob(ctx, jobID, "error", totalRows, totalPendingRows, err.Error(), rules)
		// [FIX Bug 1] KHÔNG gọi publishTransmuteTrigger khi masking failed
		// Tránh transmute worker đồng bộ data chưa mask sang bảng đích
		return
	}
	sensitiveUpdated = n
}

// [FIX Design 2] rows_affected = số record thực tế được xử lý (không cộng dồn)
// max(bulk, sensitive) vì hai phase UPDATE cùng tập rows — không phải rows riêng biệt
finalAffected := totalRows
if sensitiveUpdated > finalAffected {
	finalAffected = sensitiveUpdated
}
h.finishJob(ctx, jobID, "COMPLETED", finalAffected, totalPendingRows, "")
h.publishTransmuteTrigger(ctx, schemaName, pureTable)
```

---

## PHẦN 9 — runSensitiveMasking (hàm mới, thêm sau publishTransmuteTrigger)

**[FIX Bug 3]** Thêm `detectPrimaryKeyType` + nil guard cho MaskByStrategy.

```go
// runSensitiveMasking áp dụng Go-native HMAC/AES masking cho sensitive fields.
//
// Scale 50M–500M records:
//   - Keyset pagination native PK type (không ::text — B-Tree index ok)
//   - gjson.GetBytes zero-alloc (không unmarshal map[string]interface{})
//   - Bulk UPDATE FROM VALUES: 1 round-trip/chunk
//   - HMAC tuần tự: 2000 × 1.5µs = 3ms (goroutine overhead > 3ms — không cần)
//   - batchSize clamp: min(2000, 60000/(numRules+1))
//   - rows.Close() ngay sau chunk — giải phóng xmin, AUTOVACUUM hoạt động
func (h *BatchTransformHandler) runSensitiveMasking(
	ctx context.Context,
	execDB *gorm.DB,
	schemaName, pureTable, pkCol string,
	rules []mastermodel.MappingRuleV2,
	sensitiveWhere string,
) (int64, error) {
	if pkCol == "" {
		return 0, fmt.Errorf("sensitive masking: table %s.%s requires a primary key", schemaName, pureTable)
	}
	if len(rules) == 0 || sensitiveWhere == "" {
		return 0, nil
	}

	quotedTable := sqlutil.QualifiedTable(schemaName, pureTable)
	quotedPK    := sqlutil.QuoteIdent(pkCol)
	numCols     := len(rules) + 1
	batchSize   := h.getSensitiveBatchSize(numCols)

	// [FIX Bug 3] Detect PK type để bulkUpdateMasked cast đúng trong WHERE
	// Tránh: "operator does not exist: uuid = text" khi PK là UUID
	pkType := h.detectPrimaryKeyType(execDB, schemaName, pureTable, pkCol)

	type rawRow struct {
		pk      interface{}
		rawJSON []byte
	}

	var lastPK interface{}
	var totalUpdated int64

	for {
		// ── 1. Keyset SELECT — native PK type, không ::text cast ───────────────
		var selectSQL string
		var queryArgs []interface{}

		if lastPK == nil {
			selectSQL = fmt.Sprintf(
				`SELECT %s, _raw_data FROM %s WHERE _raw_data IS NOT NULL AND (%s) ORDER BY %s ASC LIMIT %d`,
				quotedPK, quotedTable, sensitiveWhere, quotedPK, batchSize,
			)
		} else {
			selectSQL = fmt.Sprintf(
				`SELECT %s, _raw_data FROM %s WHERE %s > ? AND _raw_data IS NOT NULL AND (%s) ORDER BY %s ASC LIMIT %d`,
				quotedPK, quotedTable, quotedPK, sensitiveWhere, quotedPK, batchSize,
			)
			queryArgs = append(queryArgs, lastPK)
		}

		rows, err := execDB.WithContext(ctx).Raw(selectSQL, queryArgs...).Rows()
		if err != nil {
			return totalUpdated, fmt.Errorf("sensitive masking: fetch chunk failed: %w", err)
		}

		var chunk []rawRow
		for rows.Next() {
			var r rawRow
			if err := rows.Scan(&r.pk, &r.rawJSON); err != nil {
				rows.Close()
				return totalUpdated, fmt.Errorf("sensitive masking: scan row failed: %w", err)
			}
			chunk = append(chunk, r)
		}
		rows.Close() // Đóng cursor NGAY — giải phóng xmin, AUTOVACUUM hoạt động bình thường

		if len(chunk) == 0 {
			break
		}

		// ── 2. Mask in-memory (tuần tự, gjson zero-alloc) ────────────────────
		maskedRows := make([]maskedRowData, len(chunk))
		for i, r := range chunk {
			vals := make([]interface{}, len(rules))
			for j, rule := range rules {
				res       := gjson.GetBytes(r.rawJSON, rule.SourceField)
				maskedVal := h.maskingSvc.MaskByStrategy(res.Value(), rule.MaskStrategy)
				// [FIX Bug 3] nil guard: tránh "could not determine data type of parameter"
				// khi field không tồn tại trong JSON và MaskByStrategy trả về nil
				if maskedVal == nil {
					maskedVal = ""
				}
				vals[j] = maskedVal
			}
			maskedRows[i] = maskedRowData{pk: r.pk, values: vals}
		}

		// ── 3. Bulk UPDATE — 1 round-trip/chunk ─────────────────────────────
		affected, err := h.bulkUpdateMasked(ctx, execDB, quotedTable, quotedPK, pkType, rules, maskedRows)
		if err != nil {
			return totalUpdated, fmt.Errorf("sensitive masking: bulk update chunk failed: %w", err)
		}
		totalUpdated += affected

		lastPK = chunk[len(chunk)-1].pk
		if len(chunk) < batchSize {
			break
		}
	}

	return totalUpdated, nil
}
```

---

## PHẦN 10 — bulkUpdateMasked (hàm mới, thêm sau runSensitiveMasking)

**[FIX Bug 3]** Thêm `pkType` param; `::text` cast cho first-row non-pk cols; `v.pk::pkType` trong WHERE.

```go
// bulkUpdateMasked thực thi UPDATE ... FROM (VALUES ...) cho toàn bộ chunk.
// 1 round-trip thay vì N round-trips.
//
// pkType: PostgreSQL type name của PK column ("bigint", "uuid", v.v.)
//   - Dùng để cast v.pk trong WHERE tránh lỗi type mismatch (uuid vs text)
//   - Cast v.pk KHÔNG ảnh hưởng B-Tree index của t.pk (index trên table column, không VALUES)
func (h *BatchTransformHandler) bulkUpdateMasked(
	ctx context.Context,
	execDB *gorm.DB,
	quotedTable, quotedPK string,
	pkType string,
	rules []mastermodel.MappingRuleV2,
	rows []maskedRowData,
) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	numCols := len(rules) + 1
	rowPHs  := make([]string, len(rows))
	args    := make([]interface{}, 0, len(rows)*numCols)

	for i, row := range rows {
		ph    := make([]string, numCols)
		ph[0]  = "?"
		for j := 1; j < numCols; j++ {
			if i == 0 {
				// [FIX Bug 3] Cast row đầu tiên → PostgreSQL infers "text" cho toàn VALUES list
				// Tránh: "could not determine data type of parameter" khi vals chứa nil/empty
				ph[j] = "?::text"
			} else {
				ph[j] = "?"
			}
		}
		rowPHs[i] = "(" + strings.Join(ph, ", ") + ")"
		args = append(args, row.pk)
		args = append(args, row.values...)
	}

	setClauses := make([]string, 0, len(rules)+1)
	for i, rule := range rules {
		setClauses = append(setClauses, fmt.Sprintf("%s = v.c%d", sqlutil.QuoteIdent(rule.TargetColumn), i))
	}
	setClauses = append(setClauses, "_updated_at = NOW()") // không có ? — không thêm arg

	aliasCols := make([]string, numCols)
	aliasCols[0] = "pk"
	for i := range rules {
		aliasCols[i+1] = fmt.Sprintf("c%d", i)
	}

	// [FIX Bug 3] Cast v.pk về đúng type của PK column trong WHERE
	// Tránh: "operator does not exist: uuid = text" khi PK là UUID
	// Cast phía v.pk (VALUES alias) — KHÔNG cast t.pk (table column) → B-Tree index an toàn
	var wherePK string
	if pkType != "" {
		wherePK = fmt.Sprintf("t.%s = v.pk::%s", quotedPK, pkType)
	} else {
		wherePK = fmt.Sprintf("t.%s = v.pk", quotedPK)
	}

	updateSQL := fmt.Sprintf(
		`UPDATE %s AS t SET %s FROM (VALUES %s) AS v(%s) WHERE %s`,
		quotedTable,
		strings.Join(setClauses, ", "),
		strings.Join(rowPHs, ", "),
		strings.Join(aliasCols, ", "),
		wherePK,
	)

	result := execDB.WithContext(ctx).Exec(updateSQL, args...)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
```

---

## PHẦN 11 — detectPrimaryKeyType (hàm mới, thêm sau detectPrimaryKey ~line 496)

```go
// detectPrimaryKeyType trả về PostgreSQL type name của cột pk.
// Ví dụ: "bigint", "uuid", "integer", "character varying", "text".
// Dùng để cast v.pk trong bulkUpdateMasked tránh type mismatch.
// Trả về "" nếu không detect được → bulkUpdateMasked fallback không cast.
func (h *BatchTransformHandler) detectPrimaryKeyType(execDB *gorm.DB, schemaName, tableName, pkCol string) string {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	var typeName string
	sql := `
		SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ? AND c.relname = ? AND a.attname = ? AND a.attnum > 0`
	if err := execDB.Raw(sql, schemaName, tableName, pkCol).Scan(&typeName).Error; err != nil {
		return ""
	}
	return typeName
}
```

---

## PHẦN 12 — server_setup.go (+1 dòng)

```go
batchTransformHandler.SetMaskingService(maskingSvc)
batchTransformHandler.SetSensitiveBatchSize(2000)   // NEW
```

---

## Execution Flow

```
[NATS: cdc.cmd.batch-transform]
        │
[runTransformJob]
        │
        ├── for loop → tách sensitiveRules vs setClauses
        ├── Post-loop:
        │     ├── len==0 cả hai → publishAndFinishJob("success") return
        │     ├── hasNonSensitiveRules = len(setClauses) > 0  ← TRƯỚC append _updated_at
        │     ├── sensitiveWhere tính riêng
        │     └── !hasNonSensitiveRules → sensitive-only path:
        │           ├── detectPK → fail-fast if empty
        │           ├── runSensitiveMasking
        │           ├── Activity Logger
        │           ├── publishAndFinishJob("success")
        │           └── publishTransmuteTrigger
        │
        ├── COUNT pending rows → set RUNNING
        ├── detectPrimaryKey (line 273)
        │
        ├── [NEW FAIL-FAST] sensitiveRules>0 && pkCol=="" → publishAndFinishJob("error") return
        │     Tránh dirty state: không để bulk SQL chạy khi biết sensitive sẽ fail
        │
        ├── No PK → Unchunked:
        │     ├── Bulk SQL UPDATE (non-sensitive only — sensitiveRules guaranteed empty)
        │     ├── finishJob(COMPLETED)
        │     └── publishTransmuteTrigger
        │
        └── Has PK → Chunked CTE loop:
              ├── [loop: CTE UPDATE chunks 1000–20000, dynamic size]
              ├── runSensitiveMasking(pkCol)
              │     ├── detectPrimaryKeyType → pkType
              │     ├── SELECT keyset native PK (B-Tree index ok)
              │     ├── gjson zero-alloc + nil guard
              │     └── bulkUpdateMasked(pkType):
              │           ├── first-row cols: ?::text (type inference)
              │           └── WHERE t.pk = v.pk::pkType (no index hit on t.pk)
              ├── err → publishAndFinishJob("error") + return [NO trigger]
              ├── finalAffected = max(totalRows, sensitiveUpdated)
              ├── finishJob(COMPLETED, finalAffected)
              └── publishTransmuteTrigger
```

---

## Self-Audit Checklist (19 điểm)

| # | Điểm kiểm tra | Kết quả |
|---|---|---|
| 1 | gjson v1.18.0 có sẵn go.mod | ✅ |
| 2 | encoding/json giữ nguyên | ✅ |
| 3 | errgroup/sync/runtime KHÔNG import | ✅ |
| 4 | pk::text xóa khỏi SELECT, WHERE keyset | ✅ |
| 5 | publishTransmuteTrigger KHÔNG gọi khi chunked masking error | ✅ |
| 6 | Sensitive-only: publishAndFinishJob thay finishJob | ✅ |
| 7 | Sensitive-only: Activity Logger trong success path | ✅ |
| 8 | Sensitive-only: fail-fast PK check trước khi run | ✅ |
| 9 | Fail-fast TRƯỚC bulk SQL (Design 1) | ✅ |
| 10 | Unchunked: sensitiveRules guaranteed empty sau fail-fast | ✅ |
| 11 | Unchunked: không có sensitive masking block | ✅ |
| 12 | Chunked: finalAffected = max(totalRows, sensitiveUpdated) | ✅ |
| 13 | bulkUpdateMasked: thêm pkType param | ✅ |
| 14 | bulkUpdateMasked: first-row cols ?::text | ✅ |
| 15 | bulkUpdateMasked: WHERE t.pk = v.pk::pkType | ✅ |
| 16 | runSensitiveMasking: nil guard cho MaskByStrategy | ✅ |
| 17 | detectPrimaryKeyType: thêm hàm mới sau detectPrimaryKey | ✅ |
| 18 | hasNonSensitiveRules TRƯỚC append _updated_at | ✅ |
| 19 | rows.Close() sau mỗi chunk | ✅ |
