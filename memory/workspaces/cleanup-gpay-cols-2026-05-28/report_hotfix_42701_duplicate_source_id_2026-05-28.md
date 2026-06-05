# Hotfix Report — SQLSTATE 42701 "source_id specified more than once"

- **Workspace**: `cleanup-gpay-cols-2026-05-28`
- **Date**: 2026-05-28
- **Severity**: P0 — production ingest fail loop (`shadow_goopay_local_ws_wallet_service.events`)
- **Trigger**: User báo log lỗi runtime sau khi tao báo "done" lần 1.
- **Root cause class**: rename-blind cross-path tạo duplicate SQL column (lesson `L-2026-05-28-rename-blind-creates-duplicate`).

---

## 1. Bug evidence (production log)

```
{"level":"error","msg":"gorm exec error",
 "error":"ERROR: column \"source_id\" specified more than once (SQLSTATE 42701)",
 "sql":""}
{"level":"error","msg":"upsert failed",
 "schema":"shadow_goopay_local_ws_wallet_service",
 "table":"events",
 "pk":"6a17c851845a375c01b28137",
 "error":"ERROR: column \"source_id\" specified more than once (SQLSTATE 42701)"}
```

Stack trace dẫn về `handler.BatchBuffer.Flush() at batch_buffer.go:184` → `BuildBatchUpsertSQLInSchema` → SQL chunk có cột `source_id` lặp 2 lần trong `INSERT INTO ... (...)`.

---

## 2. Root cause analysis

### 2.1 Pre-rename state (codebase trước hôm nay)

`internal/service/schema_adapter.go` 2 helper:
```go
func getMetadataInsertCols(schema *TableSchema) []string {
    ...
    if _, ok := schema.Columns["_gpay_source_id"]; ok {   // <-- check legacy col
        cols = append(cols, `"_gpay_source_id"`)
    }
    ...
}
```

Conditional này CHỈ dành cho V2 master Path C, nơi master schema có `_gpay_id BIGINT PK` + `_gpay_source_id TEXT` (cột business key, KHÔNG trùng PK). Path A shadow KHÔNG có column `_gpay_source_id` → conditional KHÔNG triggered → SAFE.

### 2.2 Post-rename state (rename blanket tao làm sáng nay)

```go
if _, ok := schema.Columns["source_id"]; ok {   // <-- renamed
    cols = append(cols, `"source_id"`)
}
```

Hậu rename:
- Path C V2 master: schema CÓ `source_id` → conditional triggered → add. Đúng (vì PK = `_gpay_id`, khác).
- Path A shadow: schema cũng CÓ `source_id` (từ `shadow_automator.go:80` đã có sẵn từ trước, không liên quan rename). Đồng thời `batch_buffer.go:251-256` remap `effectivePK = "source_id"`. PK + metadata branch CÙNG emit `"source_id"` → INSERT INTO X (`source_id`, ..., `source_id`, ...) → **DUPLICATE COLUMN**.

### 2.3 Vì sao build + tests PASS lần 1 mà runtime fail?

- Go compile-time KHÔNG bắt được duplicate SQL column trong literal string.
- Unit test pre-existing (`schema_adapter_test.go`) chỉ test V2 isolated case (pkField != "source_id") + V1 no-source_id case. KHÔNG test Path A pkField == "source_id".
- Verify zero-residue grep chỉ tìm `_gpay_source_id` (legacy), KHÔNG count `source_id` xuất hiện 2+ lần trong cùng SQL site.

→ Lesson: build + grep zero-residue **CHƯA ĐỦ** cho rename refactor; phải kiểm tra duplicate column trong cùng INSERT site (DB schema runtime semantic).

---

## 3. Fix

### 3.1 File 1 (CRITICAL) — `internal/service/schema_adapter.go`

Thêm `pkField` param cho 2 helper, skip `source_id` nếu PK đã chiếm column đó.

```go
func getMetadataInsertCols(schema *TableSchema, pkField string) []string {
    ...
    if _, ok := schema.Columns["source_id"]; ok && pkField != "source_id" {
        cols = append(cols, `"source_id"`)
    }
    ...
}

func getMetadataInsertPlaceholdersAndValues(
    schema *TableSchema, rawData, source, hash string,
    pkValue interface{}, sourceTsMs int64,
    pkField string,
) ([]string, []interface{}) {
    ...
    if _, ok := schema.Columns["source_id"]; ok && pkField != "source_id" {
        placeholders = append(placeholders, "?")
        vals = append(vals, fmt.Sprintf("%v", pkValue))
    }
    ...
}
```

Update 5 caller site:
- `BuildUpsertSQLInSchema:325-326` (1 site call mỗi helper).
- `BuildBatchUpsertSQLsInSchema:402` (getMetadataInsertCols).
- `BuildBatchUpsertSQLsInSchema:412` (chunk size estimate sampleMetaVals).
- `BuildBatchUpsertSQLsInSchema:448` (per-row metaPlaceholders+metaVals).

### 3.2 File 2 (DEFENSIVE) — `internal/handler/event_handler.go`

Tombstone INSERT cũng có rủi ro tương tự nếu `pgPKField == "source_id"`. Fix conditional 2 nhánh:

```go
if pgPKField == "source_id" {
    sql = `INSERT INTO X (source_id, _deleted, _created_at, _updated_at, _source)
           VALUES (?::text, TRUE, NOW(), NOW(), 'debezium')
           ON CONFLICT (source_id) DO UPDATE SET _deleted = TRUE, _updated_at = NOW()`
    args = []interface{}{pkValue}
} else {
    sql = `INSERT INTO X (pgPKField, source_id, _deleted, ...) VALUES (?, ?::text, TRUE, ...)
           ON CONFLICT (pgPKField) DO UPDATE SET ...`
    args = []interface{}{pkValue, pkValue}
}
```

### 3.3 File 3 (DEFENSIVE) — `internal/handler/command_handler.go`

Path B handler create CREATE TABLE inline cũng có rủi ro nếu `pkField == "source_id"`. Conditional 2 nhánh: promote PK trực tiếp lên `source_id PRIMARY KEY` (bỏ dòng `source_id TEXT UNIQUE` riêng).

### 3.4 File 4 (TEST) — `test/internal/service/schema_adapter_test.go`

Thêm `TestBuildUpsertSQL_PKIsSourceID_NoDuplicateColumn`:
```go
sql, _ := sa.BuildUpsertSQLInSchema(schema, ..., "source_id", "abc123", ...)
count := strings.Count(sql, `"source_id"`)
if count >= 4 {
    t.Errorf("source_id duplicate column risk: %d hits", count)
}
```

Test này regression-guard bug 42701 — invert fix ngay lập tức làm test FAIL.

---

## 4. Verify evidence

| Item | Result |
|---|---|
| `go build ./...` centralized-data-service | PASS (silent) |
| `go build ./...` cdc-cms-service | PASS (silent) |
| `go test ./internal/service/...` | `ok 0.396s` |
| `go test ./internal/handler/...` | `ok` |
| `go test ./test/internal/service/...` | `ok 0.396s` (regression test included) |
| `go test ./test/internal/sinkworker/...` | `ok` |
| `go test ./test/internal/handler/...` | `ok` |
| Regression test `TestBuildUpsertSQL_PKIsSourceID_NoDuplicateColumn` | PASS |
| `go test ./test/...` cdc-cms-service | PASS |

---

## 5. Files modified (hotfix)

| # | File | LOC delta | Type |
|---|------|-----------|------|
| 1 | `centralized-data-service/internal/service/schema_adapter.go` | +18 / -8 | CRITICAL fix |
| 2 | `centralized-data-service/internal/handler/event_handler.go` | +20 / -9 | Defensive fix |
| 3 | `centralized-data-service/internal/handler/command_handler.go` | +26 / -3 | Defensive fix |
| 4 | `centralized-data-service/test/internal/service/schema_adapter_test.go` | +37 / 0 | Regression test |
| **Tổng** | — | **+101 / -20 (NET +81)** | — |

Plus 1 lesson APPEND vào `agent/memory/global/lessons.md`: `L-2026-05-28-rename-blind-creates-duplicate` (~30 LOC).

---

## 6. Deploy considerations

- Fix code-only — KHÔNG cần thêm DB migration (migration `migration_rename_gpay_cols.sql` từ buổi sáng vẫn áp dụng nguyên).
- Deploy ngay khi user duyệt — production đang fail loop.
- Sau deploy: monitor log 5-10 phút cho `source_id specified more than once` → expect 0 occurrence.

---

## 7. Lessons APPEND today (2 lessons trong 1 ngày)

1. `L-2026-05-28-cleanup-is-not-remove` — cleanup ≠ pure remove (sáng).
2. `L-2026-05-28-rename-blind-creates-duplicate` — cleanup ≠ pure rename, phải per-site case analysis (sau bug 42701).

Cặp lesson này phải đọc cùng nhau: cleanup = mixture RENAME (target chưa có) + REMOVE (target đã có) — KHÔNG phải pure-anything.

---

## 8. Self-critique

- **Sai chính**: tao chỉ "verify build + test + grep zero-residue" mà không check duplicate column trong cùng INSERT site (đáng lẽ phải mock pkField == "source_id" + test SQL build trước khi declare done).
- **Sai phụ**: 2 lần liên tiếp trong cùng phiên — lần 1 over-defer (3 option remove), lần 2 over-correct (rename blanket). User phải sửa giữa session 2 lần.
- **Recovery**: cả 2 đều APPEND lesson vào memory global per §5 + §13.
- **DoD updated**: ngoài build + test + grep, BẮT BUỘC thêm: simulate runtime SQL với corner-case pkField để bắt duplicate column. Đã enforce qua regression test.
