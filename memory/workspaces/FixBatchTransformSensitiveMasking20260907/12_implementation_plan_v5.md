# 12_implementation_plan_v5 — Sensitive Masking Scale 50M–500M
**Workspace:** FixBatchTransformSensitiveMasking20260907
**Phiên:** 2026-09-08 | **Status:** Chờ APPROVE

---

## Bối cảnh & Vấn đề

`BatchTransformHandler.runTransformJob` xử lý HMAC/AES-GCM qua SQL expression — không scale ở volume lớn:

| Bottleneck | Hậu quả |
|---|---|
| Row-by-row UPDATE (N+1 round-trips) | 50M rows / 2000 QPS = 7–9 giờ |
| Open cursor toàn bảng | Ghim MVCC xmin, chặn AUTOVACUUM → WAL/Table bloat |
| json.Unmarshal → map[string]interface{} | GC thrashing ở volume lớn |
| finishJob(COMPLETED) trước masking | Downstream đọc data chưa mask |
| unchunked: hardcode pkCol="" | Silent skip sensitive masking |
| batchSize×numCols không giới hạn | Vượt 65535 PostgreSQL params |

**Giải pháp:** Keyset pagination + bulk UPDATE FROM VALUES + gjson zero-alloc + batchSize clamp.

---

## Files thay đổi

| File | Action | Lines ± |
|---|---|---|
| `internal/handler/shadow/batch_transform_handler.go` | Sửa + thêm hàm | +~185 |
| `internal/server/server_setup.go` | +1 dòng | +1 |
| `internal/service/metadata/mapping_utils.go` | Không sửa | 0 |

---

## PHẦN 1 — Import block (thay thế lines 3–22 trong file hiện tại)

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
	"github.com/tidwall/gjson"       // [NEW] zero-alloc JSON field extraction — gjson v1.18.0 có sẵn go.mod
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

> - `encoding/json` giữ nguyên — dùng cho NATS payload unmarshal.
> - Không thêm `sync`, `runtime`, `golang.org/x/sync/errgroup` — HMAC tuần tự (2000 × 1.5µs = 3ms, goroutine overhead > 3ms).

---

## PHẦN 2 — Package-level type (thêm sau imports, trước struct, trước line 24)

```go
// maskedRowData giữ PK native type và các giá trị đã masked cho bulkUpdateMasked.
// Native PK type (int64/string/[16]byte) được giữ nguyên để pgx bind đúng type → B-Tree index.
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
	sensitiveBatchSize int                       // [NEW] batch size cho sensitive masking, default 2000
	metadataRegistry   metadata.MetadataRegistry
	registryRepo       metadata.RegistryResolver
	transformJobRepo   *repository.TransformJobRepo
	hmacKey            string
	maskingSvc         *governance.MaskingService // [NEW] Go-level HMAC/AES masking
}
```

---

## PHẦN 4 — Setters (thêm sau SetTransformJobRepo, trước HandleBatchTransform — sau line 84)

```go
// SetMaskingService injects Go-level masking service cho sensitive fields.
func (h *BatchTransformHandler) SetMaskingService(svc *governance.MaskingService) {
	h.maskingSvc = svc
}

// SetSensitiveBatchSize đặt số records mỗi batch cho sensitive masking.
// Mặc định 2000 nếu không set hoặc set <= 0.
func (h *BatchTransformHandler) SetSensitiveBatchSize(n int) {
	if n > 0 {
		h.sensitiveBatchSize = n
	}
}

// getSensitiveBatchSize trả về batch size thực tế, đã clamp để không vượt 65535 PostgreSQL params.
// numCols = len(sensitiveRules) + 1 (1 cho pk column trong VALUES list)
func (h *BatchTransformHandler) getSensitiveBatchSize(numCols int) int {
	base := h.sensitiveBatchSize
	if base <= 0 {
		base = 2000
	}
	// 60000 < 65535 — buffer an toàn tránh vượt giới hạn PostgreSQL protocol (uint16)
	if maxSafe := 60000 / numCols; base > maxSafe {
		return maxSafe
	}
	return base
}
```

---

## PHẦN 5 — For loop trong runTransformJob (thay thế lines 195–235)

Thay đổi: tách `sensitiveRules`, force filter TRƯỚC sensitive check.

```go
var setClauses []string
var whereClauses []string
var sensitiveRules []mastermodel.MappingRuleV2 // [NEW] rules cần Go-level masking
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
	// Force-mode filter: bỏ qua field không thuộc force_fields (áp dụng TRƯỚC sensitive check)
	if payload.Force {
		if _, inForce := forceSet[colKey]; !inForce {
			continue
		}
	}
	// [NEW] Sensitive field: tách sang Go-level masking, KHÔNG đưa vào SQL bulk
	if rule.IsSensitiveField && h.maskingSvc != nil {
		strategy := strings.ToLower(strings.TrimSpace(rule.MaskStrategy))
		if strategy != "" && strategy != "none" {
			sensitiveRules = append(sensitiveRules, rule)
			continue // KHÔNG append setClauses, KHÔNG append whereClauses
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

Thay đổi: `hasNonSensitiveRules`, `sensitiveWhere`, sensitive-only early return.

```go
// Không có rule nào cả
if len(setClauses) == 0 && len(sensitiveRules) == 0 {
	h.publishAndFinishJob(ctx, jobID, "success", 0, 0, "no active rules to transform", rules)
	return
}

// QUAN TRỌNG: tính hasNonSensitiveRules TRƯỚC KHI append _updated_at vào setClauses.
// Nếu tính sau: len(setClauses) luôn >= 1 → sensitive-only path không bao giờ kích hoạt.
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

// sensitiveWhere: WHERE clause riêng cho sensitive phase.
// KHÔNG dùng whereExpr (whereExpr là IS NULL check cho non-sensitive cols).
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

// Sensitive-only path: bảng không có non-sensitive rule nào → bỏ qua bulk SQL + COUNT + loop
if !hasNonSensitiveRules {
	pkCol, _ := h.detectPrimaryKey(execDB, schemaName, pureTable)
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive-only masking failed",
			zap.String("table", targetTable), zap.Error(err))
		h.finishJob(ctx, jobID, "FAILED", 0, 0, err.Error())
	} else {
		h.finishJob(ctx, jobID, "COMPLETED", n, n, "")
		h.publishTransmuteTrigger(ctx, schemaName, pureTable)
	}
	return
}
```

---

## PHẦN 7 — Unchunked success path (thay thế lines 303–310)

Thay đổi: finishJob SAU masking; pkCol từ line 273 (không hardcode "").

```go
observability.Ctx(ctx, h.Logger).Info("batch transform completed (unchunked)",
	zap.String("table", targetTable),
	zap.Int64("rows_affected", result.RowsAffected),
)
if h.DB != nil {
	act := governance.NewActivityLogger(h.DB, h.Logger)
	act.Quick("cmd-batch-transform", targetTable, "nats-command", "success",
		result.RowsAffected, map[string]interface{}{"trace_id": traceID}, "")
}
// [FIX] Sensitive masking TRƯỚC finishJob.
// pkCol từ line 273: nếu rỗng (no PK), runSensitiveMasking return error rõ ràng — không skip silent.
var sensitiveUpdated int64
if len(sensitiveRules) > 0 && h.maskingSvc != nil {
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive masking failed (unchunked)",
			zap.String("table", targetTable), zap.Error(err))
		h.publishAndFinishJob(ctx, jobID, "error", result.RowsAffected, totalPendingRows, err.Error(), rules)
		return
	}
	sensitiveUpdated = n
}
// [FIX] finishJob chỉ gọi khi CẢ HAI phase hoàn thành
h.finishJob(ctx, jobID, "COMPLETED", result.RowsAffected+sensitiveUpdated, totalPendingRows, "")
h.publishTransmuteTrigger(ctx, schemaName, pureTable)
return
```

---

## PHẦN 8 — Chunked success path (thay thế lines 442–448)

Thay đổi: finishJob SAU masking. pkCol hợp lệ (chunked path chỉ đến khi pkErr==nil && pkCol!="").

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
// [FIX] Sensitive masking TRƯỚC finishJob.
// pkCol hợp lệ — chunked path chỉ đến đây khi pkErr == nil && pkCol != "".
var sensitiveUpdated int64
if len(sensitiveRules) > 0 && h.maskingSvc != nil {
	n, err := h.runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, sensitiveWhere)
	if err != nil {
		observability.Ctx(ctx, h.Logger).Error("sensitive masking failed (chunked)",
			zap.String("table", targetTable), zap.Error(err))
		h.publishAndFinishJob(ctx, jobID, "error", totalRows, totalPendingRows, err.Error(), rules)
		h.publishTransmuteTrigger(ctx, schemaName, pureTable)
		return
	}
	sensitiveUpdated = n
}
// [FIX] finishJob chỉ gọi khi CẢ HAI phase hoàn thành
h.finishJob(ctx, jobID, "COMPLETED", totalRows+sensitiveUpdated, totalPendingRows, "")
h.publishTransmuteTrigger(ctx, schemaName, pureTable)
```

---

## PHẦN 9 — runSensitiveMasking (hàm mới, thêm sau publishTransmuteTrigger)

```go
// runSensitiveMasking áp dụng Go-native HMAC/AES masking cho sensitive fields.
//
// Thiết kế cho 50M–500M records:
//   - Keyset pagination trên native PK type (không cast ::text — giữ B-Tree index, tránh sort lexicographic sai)
//   - gjson.GetBytes: zero-alloc JSON field extraction (không unmarshal toàn bộ map[string]interface{})
//   - Bulk UPDATE FROM (VALUES ...): 1 DB round-trip cho toàn chunk
//   - HMAC/AES xử lý tuần tự (2000 × 1.5µs = 3ms — goroutine overhead > 3ms, không cần goroutine)
//   - batchSize clamp: min(2000, 60000/(numRules+1)) để không vượt 65535 params PostgreSQL
//   - rows.Close() ngay sau mỗi chunk — giải phóng xmin MVCC, AUTOVACUUM hoạt động bình thường
//
// pkCol rỗng → return error (table thiếu PK — operator cần thêm PK trước khi chạy).
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
	numCols     := len(rules) + 1 // 1 pk + 1 col per rule
	batchSize   := h.getSensitiveBatchSize(numCols)

	type rawRow struct {
		pk      interface{}
		rawJSON []byte
	}

	var lastPK interface{} // nil cho chunk đầu tiên
	var totalUpdated int64

	for {
		// ── 1. Keyset SELECT — native PK type, không cast ::text ───────────────
		// Giữ nguyên kiểu PK gốc để pgx bind đúng type → PostgreSQL dùng B-Tree index
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
		rows.Close() // ← Đóng cursor NGAY — giải phóng xmin snapshot, AUTOVACUUM chạy bình thường

		if len(chunk) == 0 {
			break
		}

		// ── 2. Mask in-memory (tuần tự, gjson zero-alloc) ────────────────────
		// Dùng rule.SourceField trực tiếp — system dùng flat keys per BuildCastExpr pattern
		maskedRows := make([]maskedRowData, len(chunk))
		for i, r := range chunk {
			vals := make([]interface{}, len(rules))
			for j, rule := range rules {
				res     := gjson.GetBytes(r.rawJSON, rule.SourceField)
				vals[j]  = h.maskingSvc.MaskByStrategy(res.Value(), rule.MaskStrategy)
			}
			maskedRows[i] = maskedRowData{pk: r.pk, values: vals}
		}

		// ── 3. Bulk UPDATE — 1 round-trip cho toàn chunk ─────────────────────
		affected, err := h.bulkUpdateMasked(ctx, execDB, quotedTable, quotedPK, rules, maskedRows)
		if err != nil {
			return totalUpdated, fmt.Errorf("sensitive masking: bulk update chunk failed: %w", err)
		}
		totalUpdated += affected

		lastPK = chunk[len(chunk)-1].pk
		if len(chunk) < batchSize {
			break // chunk cuối — không còn dữ liệu
		}
	}

	return totalUpdated, nil
}
```

---

## PHẦN 10 — bulkUpdateMasked (hàm mới, thêm sau runSensitiveMasking)

```go
// bulkUpdateMasked thực thi UPDATE ... FROM (VALUES ...) cho toàn bộ chunk.
// Giảm N round-trips xuống 1 round-trip mỗi batchSize rows.
//
// PK comparison: t.pk = v.pk trên native type.
// pgx bind đúng type từ interface{} → PostgreSQL dùng B-Tree index lookup (không sequential scan).
func (h *BatchTransformHandler) bulkUpdateMasked(
	ctx context.Context,
	execDB *gorm.DB,
	quotedTable, quotedPK string,
	rules []mastermodel.MappingRuleV2,
	rows []maskedRowData,
) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	numCols := len(rules) + 1 // pk + 1 col per rule
	rowPHs  := make([]string, len(rows))
	args    := make([]interface{}, 0, len(rows)*numCols)

	for i, row := range rows {
		ph := make([]string, numCols)
		for j := range ph {
			ph[j] = "?"
		}
		rowPHs[i] = "(" + strings.Join(ph, ", ") + ")"
		args = append(args, row.pk)        // native type → pgx giữ đúng type, PostgreSQL dùng index
		args = append(args, row.values...) // masked values
	}

	// SET: "col0" = v.c0, "col1" = v.c1, ..., _updated_at = NOW()
	setClauses := make([]string, 0, len(rules)+1)
	for i, rule := range rules {
		setClauses = append(setClauses, fmt.Sprintf("%s = v.c%d", sqlutil.QuoteIdent(rule.TargetColumn), i))
	}
	setClauses = append(setClauses, "_updated_at = NOW()") // không có ? — không thêm arg

	// VALUES alias: pk, c0, c1, ...
	aliasCols := make([]string, numCols)
	aliasCols[0] = "pk"
	for i := range rules {
		aliasCols[i+1] = fmt.Sprintf("c%d", i)
	}

	// WHERE trên native PK type — không ::text cast → PostgreSQL giữ được B-Tree index
	updateSQL := fmt.Sprintf(
		`UPDATE %s AS t SET %s FROM (VALUES %s) AS v(%s) WHERE t.%s = v.pk`,
		quotedTable,
		strings.Join(setClauses, ", "),
		strings.Join(rowPHs, ", "),
		strings.Join(aliasCols, ", "),
		quotedPK,
	)

	result := execDB.WithContext(ctx).Exec(updateSQL, args...)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
```

---

## server_setup.go (+1 dòng, thêm sau SetMaskingService)

```go
batchTransformHandler.SetMaskingService(maskingSvc)
batchTransformHandler.SetSensitiveBatchSize(2000)   // [NEW]
```

---

## Execution Flow

```
[NATS: cdc.cmd.batch-transform]
        │
        ▼
[runTransformJob]
        │
        ├── for loop: tách sensitiveRules vs setClauses
        │
        ├── Post-loop:
        │       ├── len == 0 cả hai → publishAndFinishJob("success") return
        │       ├── hasNonSensitiveRules = len(setClauses) > 0    ← TRƯỚC append _updated_at
        │       ├── sensitiveWhere tính riêng (độc lập whereExpr)
        │       └── !hasNonSensitiveRules → sensitive-only path:
        │               detectPK → runSensitiveMasking → finishJob → trigger
        │
        ├── COUNT pending rows → RUNNING
        ├── detectPrimaryKey (line 273)
        │
        ├── No PK → Unchunked path:
        │       ├── Bulk SQL UPDATE (non-sensitive setClauses)
        │       ├── runSensitiveMasking(pkCol)  ← pkCol từ line 273, error nếu rỗng
        │       ├── finishJob(COMPLETED)         ← SAU CẢ HAI
        │       └── publishTransmuteTrigger
        │
        └── Has PK → Chunked CTE loop:
                ├── [loop: CTE UPDATE chunks dynamic size 1000–20000]
                ├── runSensitiveMasking(pkCol)  ← pkCol hợp lệ
                ├── finishJob(COMPLETED)         ← SAU CẢ HAI
                └── publishTransmuteTrigger

[runSensitiveMasking — per chunk]
  SELECT pk, _raw_data WHERE pk > lastPK LIMIT batchSize  (native PK, B-Tree index)
  ┌── for row: gjson.GetBytes → MaskByStrategy (tuần tự, ~3ms/2000 rows)
  └── bulkUpdateMasked: UPDATE t SET col=v.c0 FROM (VALUES...) WHERE t.pk=v.pk
  rows.Close() → xmin released → AUTOVACUUM ok
```

---

## Self-Audit Checklist (16 điểm)

| # | Điểm kiểm tra | Kết quả |
|---|---|---|
| 1 | gjson v1.18.0 có sẵn go.mod | ✅ |
| 2 | encoding/json giữ nguyên (NATS unmarshal) | ✅ |
| 3 | errgroup/sync/runtime KHÔNG import | ✅ |
| 4 | pk::text xóa khỏi SELECT, WHERE, VALUES JOIN | ✅ |
| 5 | finishJob sau cả hai phase (unchunked) | ✅ |
| 6 | finishJob sau cả hai phase (chunked) | ✅ |
| 7 | unchunked: pkCol từ line 273, không hardcode "" | ✅ |
| 8 | chunked: pkCol hợp lệ khi đến đây | ✅ |
| 9 | batchSize clamp: min(2000, 60000/numCols) | ✅ |
| 10 | hasNonSensitiveRules TRƯỚC append _updated_at | ✅ |
| 11 | rows.Close() sau mỗi chunk | ✅ |
| 12 | args count = ? count (NOW() không có ?) | ✅ |
| 13 | maskedRowData package-level trước struct | ✅ |
| 14 | mapping_utils.go không sửa | ✅ |
| 15 | sensitiveWhere độc lập whereExpr | ✅ |
| 16 | gjson rule.SourceField không escape dot (flat keys) | ✅ |
