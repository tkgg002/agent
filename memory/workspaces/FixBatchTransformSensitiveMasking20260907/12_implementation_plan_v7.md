# 12_implementation_plan_v7 — Sensitive Masking Scale 50M–500M
**Workspace:** FixBatchTransformSensitiveMasking20260907
**Phiên:** 2026-09-08 | **Status:** Chờ APPROVE

---

## So sánh v6 → v7 (3 điểm sửa cuối)

| # | Rủi ro v6 | Fix v7 |
|---|---|---|
| Risk 1 | Keyset SELECT: `lastPK` bind as `text` → `uuid > text` → sập | Thêm `?::pkType` cho `lastPK` trong WHERE clause |
| Risk 2 | nil guard `nil → ""` gây data semantic corruption (NULL → empty string, vỡ CHECK constraint, sai incremental) | Xóa nil guard, giữ `nil` là SQL NULL; `?::text` ở row 0 của VALUES đã đủ để PostgreSQL infer type |
| Risk 3 | `ph[0]` (PK) row 0 không cast → type inference bất định cho cột `v.pk` | Row 0: `ph[0] = "?::pkType"` nếu pkType != "" |

---

## Files thay đổi

| File | Lines ± |
|---|---|
| `internal/handler/shadow/batch_transform_handler.go` | +~200 |
| `internal/server/server_setup.go` | +1 |
| `internal/service/metadata/mapping_utils.go` | 0 (không sửa) |

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

> - `encoding/json` giữ nguyên — dùng cho NATS payload unmarshal & `publishTransmuteTrigger`.
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
	// Sensitive field: tách sang Go-level masking, KHÔNG đưa vào SQL bulk
	if rule.IsSensitiveField && h.maskingSvc != nil {
		strategy := strings.ToLower(strings.TrimSpace(rule.MaskStrategy))
		if strategy != "" && strategy != "none" {
			sensitiveRules = append(sensitiveRules, rule)
			continue
		}
	}
	// Non-sensitive: SQL expression (giữ nguyên behavior cũ)
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

```go
if len(setClauses) == 0 && len(sensitiveRules) == 0 {
	h.publishAndFinishJob(ctx, jobID, "success", 0, 0, "no active rules to transform", rules)
	return
}

// QUAN TRỌNG: tính hasNonSensitiveRules TRƯỚC KHI append _updated_at
// Nếu tính sau: len(setClauses) luôn >= 1 → sensitive-only path không bao giờ kích hoạt
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

// Sensitive-only path: bảng không có non-sensitive rule nào
if !hasNonSensitiveRules {
	// Fail-fast PK check trước khi chạy bất kỳ thứ gì
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

```go
pkCol, pkErr := h.detectPrimaryKey(execDB, schemaName, pureTable)
chunkSize := h.transformChunkSize
if chunkSize <= 0 {
	chunkSize = 1000
}

// Fail-fast: sensitive masking yêu cầu PK
// Tránh dirty state: non-sensitive đã UPDATE xong, sensitive fail → data ghi dở dang không rollback được
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
	// sensitiveRules guaranteed empty tại đây (fail-fast ở trên đã handle)
	h.finishJob(ctx, jobID, "COMPLETED", result.RowsAffected, totalPendingRows, "")
	h.publishTransmuteTrigger(ctx, schemaName, pureTable)
	return
}
```

---

## PHẦN 8 — Chunked success path (thay thế lines 442–448)

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

// Sensitive masking TRƯỚC finishJob
var sensitiveUpdated int64
if len(sensitiveRules) > 0 && h.maskingSvc != nil {
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive masking failed (chunked)",
			zap.String("table", targetTable), zap.Error(err))
		h.publishAndFinishJob(ctx, jobID, "error", totalRows, totalPendingRows, err.Error(), rules)
		// KHÔNG gọi publishTransmuteTrigger khi masking failed
		// Tránh transmute worker đồng bộ data chưa mask sang bảng đích
		return
	}
	sensitiveUpdated = n
}

// rows_affected = số record thực tế được xử lý, không cộng dồn
// max(bulk, sensitive): hai phase UPDATE cùng tập rows — không phải rows riêng biệt
finalAffected := totalRows
if sensitiveUpdated > finalAffected {
	finalAffected = sensitiveUpdated
}
h.finishJob(ctx, jobID, "COMPLETED", finalAffected, totalPendingRows, "")
h.publishTransmuteTrigger(ctx, schemaName, pureTable)
```

---

## PHẦN 9 — `runSensitiveMasking` (hàm mới, thêm sau `publishTransmuteTrigger`)

**[FIX Risk 1]** Keyset SELECT cast `lastPK` với `?::pkType`.
**[FIX Risk 2]** Xóa nil guard `nil → ""` — giữ nguyên `nil` là SQL NULL.

```go
// runSensitiveMasking áp dụng Go-native HMAC/AES masking cho sensitive fields.
//
// Scale 50M–500M records:
//   - Keyset pagination native PK type (không ::text — B-Tree index ok)
//   - lastPK cast ?::pkType trong WHERE để tránh "operator does not exist: uuid > text"
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

	// Detect PK type để:
	// (1) cast ?::pkType trong WHERE keyset → tránh "uuid > text"
	// (2) cast ph[0] trong VALUES row 0 → type inference chuẩn cho v.pk
	pkType := h.detectPrimaryKeyType(execDB, schemaName, pureTable, pkCol)

	type rawRow struct {
		pk      interface{}
		rawJSON []byte
	}

	var lastPK interface{}
	var totalUpdated int64

	for {
		// ── 1. Keyset SELECT ────────────────────────────────────────────────
		var selectSQL string
		var queryArgs []interface{}

		if lastPK == nil {
			selectSQL = fmt.Sprintf(
				`SELECT %s, _raw_data FROM %s WHERE _raw_data IS NOT NULL AND (%s) ORDER BY %s ASC LIMIT %d`,
				quotedPK, quotedTable, sensitiveWhere, quotedPK, batchSize,
			)
		} else {
			// [FIX Risk 1] Cast lastPK sang pkType để tránh lỗi:
			// "operator does not exist: uuid > text" khi PK là UUID
			// Không cast lastPK → driver bind dưới dạng text → PostgreSQL không tìm được operator uuid > text
			pkCastPH := "?"
			if pkType != "" {
				pkCastPH = fmt.Sprintf("?::%s", pkType)
			}
			selectSQL = fmt.Sprintf(
				`SELECT %s, _raw_data FROM %s WHERE %s > %s AND _raw_data IS NOT NULL AND (%s) ORDER BY %s ASC LIMIT %d`,
				quotedPK, quotedTable, quotedPK, pkCastPH, sensitiveWhere, quotedPK, batchSize,
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
				res := gjson.GetBytes(r.rawJSON, rule.SourceField)
				// [FIX Risk 2] Giữ nguyên nil (SQL NULL), KHÔNG ép thành ""
				// Lý do: nil → "" sẽ:
				//   (a) biến cột DB từ NULL thành empty string — data corruption
				//   (b) phá CHECK constraint (phone regex, length)
				//   (c) sai incremental: sensitiveWhere (IS NULL) sẽ không nhận diện được records này nữa
				// Việc phòng "could not determine data type" đã được xử lý bởi
				// ph[j] = "?::text" ở row 0 của bulkUpdateMasked — PostgreSQL đã biết type là text
				vals[j] = h.maskingSvc.MaskByStrategy(res.Value(), rule.MaskStrategy)
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

## PHẦN 10 — `bulkUpdateMasked` (hàm mới, thêm sau `runSensitiveMasking`)

**[FIX Risk 3]** Row 0: `ph[0] = "?::pkType"` để type inference chuẩn cho cột `v.pk`.

```go
// bulkUpdateMasked thực thi UPDATE ... FROM (VALUES ...) cho toàn bộ chunk.
// 1 round-trip thay vì N round-trips.
//
// pkType: PostgreSQL type name của PK column ("bigint", "uuid", v.v.)
//   - Row 0: ph[0] = "?::pkType" → PostgreSQL infers type cho v.pk chuẩn ngay từ đầu
//   - Row 0: ph[j≥1] = "?::text" → PostgreSQL infers type cho cột data là text
//   - WHERE: t.pk = v.pk::pkType → cast v.pk (VALUES alias) về đúng type, B-Tree index của t.pk giữ nguyên
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
		ph := make([]string, numCols)
		if i == 0 {
			// [FIX Risk 3] Row 0: định hình type cho toàn bộ virtual table "v" ngay từ dòng đầu
			// ph[0] = "?::pkType" → type của v.pk được infer đúng (bigint, uuid, ...) không phải text
			// ph[j] = "?::text"   → type của v.c0, v.c1, ... được infer là text
			// Điều này loại bỏ "could not determine data type" và "uuid = text" mismatch
			if pkType != "" {
				ph[0] = fmt.Sprintf("?::%s", pkType)
			} else {
				ph[0] = "?"
			}
			for j := 1; j < numCols; j++ {
				ph[j] = "?::text"
			}
		} else {
			// Row i>0: dùng ? thuần — PostgreSQL đã biết type từ row 0
			for j := 0; j < numCols; j++ {
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

	// Cast v.pk về đúng type của PK column trong WHERE
	// Tránh "operator does not exist: uuid = text" khi PK là UUID
	// Cast phía v.pk (VALUES alias) — KHÔNG cast t.pk (table column) → B-Tree index giữ nguyên
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

## PHẦN 11 — `detectPrimaryKeyType` (hàm mới, thêm sau `detectPrimaryKey` ~line 496)

```go
// detectPrimaryKeyType trả về PostgreSQL type name của cột pk.
// Ví dụ: "bigint", "uuid", "integer", "character varying", "text".
// Dùng để:
//   - Cast ?::pkType khi so sánh lastPK trong keyset SELECT (tránh uuid > text)
//   - Cast ph[0] = "?::pkType" ở row 0 của VALUES (type inference chuẩn)
//   - Cast v.pk::pkType trong WHERE (tránh uuid = text)
// Trả về "" nếu không detect được → fallback không cast (hành vi tương thích cũ).
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

## PHẦN 12 — `server_setup.go` (+1 dòng)

```go
batchTransformHandler.SetMaskingService(maskingSvc)
batchTransformHandler.SetSensitiveBatchSize(2000)   // NEW
```

---

## Execution Flow (Final)

```
[NATS: cdc.cmd.batch-transform]
        │
[runTransformJob]
        │
        ├── for loop → tách sensitiveRules vs setClauses
        │
        ├── Post-loop:
        │     ├── len==0 cả hai → publishAndFinishJob("success") return
        │     ├── hasNonSensitiveRules = len(setClauses) > 0  ← TRƯỚC append _updated_at
        │     ├── sensitiveWhere tính riêng (độc lập whereExpr)
        │     └── !hasNonSensitiveRules → sensitive-only path:
        │           ├── detectPK → fail-fast if empty
        │           ├── runSensitiveMasking
        │           ├── Activity Logger
        │           ├── publishAndFinishJob("success")
        │           └── publishTransmuteTrigger
        │
        ├── COUNT pending rows → set RUNNING
        ├── detectPrimaryKey
        │
        ├── [FAIL-FAST] sensitiveRules>0 && pkCol=="" → publishAndFinishJob("error")
        │     Tránh dirty state (bulk SQL chưa chạy)
        │
        ├── No PK → Unchunked:
        │     ├── Bulk SQL (sensitiveRules guaranteed empty)
        │     ├── finishJob(COMPLETED)
        │     └── publishTransmuteTrigger
        │
        └── Has PK → Chunked CTE loop:
              ├── [loop: CTE UPDATE chunks 1000–20000]
              ├── runSensitiveMasking(pkCol):
              │     ├── detectPrimaryKeyType → pkType
              │     ├── SELECT: chunk 1: WHERE (IS NULL)
              │     │          chunk 2+: WHERE pk > ?::pkType [FIX Risk 1]
              │     ├── gjson zero-alloc
              │     ├── MaskByStrategy → nil giữ nguyên nil [FIX Risk 2]
              │     └── bulkUpdateMasked(pkType):
              │           ├── row 0: ph[0]="?::pkType", ph[j]="?::text" [FIX Risk 3]
              │           ├── row i>0: ph[j]="?"
              │           └── WHERE t.pk = v.pk::pkType
              ├── err → publishAndFinishJob("error") + return (NO trigger)
              ├── finalAffected = max(totalRows, sensitiveUpdated)
              ├── finishJob(COMPLETED, finalAffected)
              └── publishTransmuteTrigger
```

---

## Self-Audit Checklist (22 điểm)

| # | Điểm kiểm tra | Kết quả |
|---|---|---|
| 1 | gjson v1.18.0 có sẵn go.mod | ✅ |
| 2 | encoding/json giữ nguyên | ✅ |
| 3 | errgroup/sync/runtime KHÔNG import | ✅ |
| 4 | pk::text xóa khỏi SELECT keyset (dùng ?::pkType thay) | ✅ |
| 5 | Chunked error: publishTransmuteTrigger KHÔNG gọi | ✅ |
| 6 | Sensitive-only: publishAndFinishJob thay finishJob | ✅ |
| 7 | Sensitive-only: Activity Logger trong success path | ✅ |
| 8 | Sensitive-only: fail-fast PK check trước khi run | ✅ |
| 9 | Fail-fast TRƯỚC bulk SQL | ✅ |
| 10 | Unchunked: sensitiveRules guaranteed empty | ✅ |
| 11 | Unchunked: không có unreachable sensitive block | ✅ |
| 12 | Chunked: finalAffected = max(totalRows, sensitiveUpdated) | ✅ |
| 13 | runSensitiveMasking: pkCastPH = "?::pkType" cho lastPK [Risk 1] | ✅ |
| 14 | runSensitiveMasking: nil guard đã xóa — nil giữ nguyên [Risk 2] | ✅ |
| 15 | bulkUpdateMasked: row 0 ph[0] = "?::pkType" [Risk 3] | ✅ |
| 16 | bulkUpdateMasked: row 0 ph[j≥1] = "?::text" | ✅ |
| 17 | bulkUpdateMasked: row i>0 ph[j] = "?" | ✅ |
| 18 | bulkUpdateMasked: WHERE t.pk = v.pk::pkType | ✅ |
| 19 | detectPrimaryKeyType: hàm mới sau detectPrimaryKey | ✅ |
| 20 | hasNonSensitiveRules TRƯỚC append _updated_at | ✅ |
| 21 | rows.Close() sau mỗi chunk | ✅ |
| 22 | batchSize clamp: min(2000, 60000/numCols) | ✅ |
