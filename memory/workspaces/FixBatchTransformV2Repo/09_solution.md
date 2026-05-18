# 09_solution — FixBatchTransformV2Repo

## Tổng quan
1 lệnh user (`cmd-batch-transform` lỗi "no active mapping rules") → khi
fix lộ ra cascade 4 bug khác (cùng pattern với fix sáng cùng ngày
`FixAlterColumnShadowSchema`). Tổng cộng **5 edit** trong **1 file**
(`centralized-data-service/internal/handler/command_handler.go`).

## Root causes
| # | Root cause | Symptom | Fix |
|---|---|---|---|
| 1 | Worker query V1 table (`cdc_mapping_rules`), CMS ghi V2 (`mapping_rule_v2`) | `no active mapping rules` (nhưng V2 có 18 rule) | Đổi sang `mappingV2Repo.GetActiveRulesBySourceTable` |
| 2 | UPDATE trên `h.db` (dest plane 5434, không có schema shadow) | `relation "shadow_centralized_export_service.sd_export_jobs" does not exist` | Switch `execDB := h.shadowDB if h.shadowDB != nil else h.db` |
| 3 | Bare identifier `exportType` → PG fold lowercase | `column "exporttype" does not exist` | Wrap với `quoteCommandIdent()` cho cả SET và WHERE |
| 4 | JSONB column dùng cast `::TEXT` (qua `->>`) | `column "params" is of type jsonb but expression is of type text (42804)` | `buildCastExpr` thêm nhánh `jsonb`/`json` dùng `->` (returns jsonb) |
| 5 | Mongo BSON Date stored as JSON number (epoch-ms) | `timestamp out of range: "1778482050803" (22008)` khi `(text)::TIMESTAMP` | `buildCastExpr` nhánh timestamp → CASE WHEN jsonb_typeof='number' THEN to_timestamp(BIGINT/1000) ELSE ::TIMESTAMP END |

## Code changes (5 edit / 1 file)

### File: `centralized-data-service/internal/handler/command_handler.go`

**Edit #1 — line ~993**: `HandleBatchTransform` rule lookup
```go
// Trước:
rules, err := h.mappingRepo.GetActiveRulesByTargetTable(context.Background(), targetTable)
// Sau:
rules, err := h.mappingV2Repo.GetActiveRulesBySourceTable(context.Background(), sourceTable)
```

**Edit #2 — line ~1037**: dùng đúng DB connection
```go
execDB := h.db
if h.shadowDB != nil {
    execDB = h.shadowDB
}
result := execDB.Exec(transformSQL)
```

**Edit #3 — quote identifiers trong setClauses/whereClauses**:
```go
quotedCol := quoteCommandIdent(rule.TargetColumn)
setClauses = append(setClauses, fmt.Sprintf("%s = %s", quotedCol, castExpr))
whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quotedCol))
```

**Edit #4 — `buildCastExpr` thêm nhánh JSONB/JSON**:
```go
switch strings.ToLower(dataType) {
case "jsonb":
    return fmt.Sprintf("(_raw_data->'%s')", field)
case "json":
    return fmt.Sprintf("((_raw_data->'%s')::JSON)", field)
}
```

**Edit #5 — `buildCastExpr` nhánh timestamp + tách int4/int8**:
```go
case "integer", "int", "int4", "smallint":
    return fmt.Sprintf("(%s)::INTEGER", base)
case "int8", "bigint":
    return fmt.Sprintf("(%s)::BIGINT", base)
...
case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
    return fmt.Sprintf(
        "(CASE WHEN jsonb_typeof(_raw_data->'%s') = 'number' "+
            "THEN to_timestamp((%s)::BIGINT / 1000.0) AT TIME ZONE 'UTC' "+
            "ELSE (%s)::TIMESTAMP END)",
        field, base, base)
```

## Verify (5 lần test)

| Test | Trigger | Kết quả |
|---|---|---|
| #1 | NATS pub batch-transform | `no active mapping rules` (pre-fix #1) |
| #2 | sau Edit #1 | `relation does not exist` (pre-fix #2) |
| #3 | sau Edit #2 | `column "exporttype" does not exist` (pre-fix #3) |
| #4 | sau Edit #3 | `params is of type jsonb` (pre-fix #4) |
| #5 | sau Edit #4 | `timestamp out of range` (pre-fix #5) |
| #6 | sau Edit #5 | `success`, rows=129, no error ✓ |

### Activity log (post-fix, từ DB)
```
id | operation           | target_table   | status  | rows | error
34 | cmd-batch-transform | sd_export_jobs | success | 129  | (null)
33 | cmd-batch-transform | sd_export_jobs | error   | 0    | timestamp out of range
32 | cmd-batch-transform | sd_export_jobs | error   | 0    | jsonb but expression is of type text
```

### DB shape verify
```sql
-- params là JSONB thật:
SELECT pg_typeof(params), jsonb_typeof(params), params LIMIT 2;
-- jsonb | string | "testValue"

-- createdAt decode đúng từ epoch-ms:
SELECT "createdAt" LIMIT 2;
-- 2026-04-16 01:58:45.976  (1778482050803ms = 1778482050.803s = 2026-04-16T01:57:30.803Z UTC)
```

## Service health (post-fix)
| Service | PID | Port | Health |
|---|---|---|---|
| centralized-data-service worker | 11709 | 8082, 9090 | `{"service":"cdc-worker","status":"ok"}` |
| cdc-cms-service | (giữ nguyên session trước) | 8083 | `{"service":"cdc-cms","status":"ok"}` |

## Skills / Tools đã dùng
- **Read**: trace `HandleBatchTransform`, `buildCastExpr`,
  `mappingV2Repo.GetActiveRulesBySourceTable`,
  `quoteCommandIdent`, `mapping_rule_v2.go` model.
- **Edit**: 5 edit trong 1 file Go.
- **Bash**: `go build`, `kill`, `nohup go run`, `lsof`, `curl /health`,
  `nats pub`, `docker exec psql` (verify column type / value /
  activity log).
- **TaskCreate / TaskUpdate**: track 3 task.
- **Governance**:
  - §3 Verify Before Done — chứng minh bằng activity_log entry id=34
    + DB row count + pg_typeof.
  - §6 Minimal scope — chỉ chạm 1 file, 5 edit nhỏ; không refactor
    rộng.
  - §7 Workspace prefix bắt buộc (00, 05, 09, report).
  - §11 Memory APPEND only.
  - §12 Muscle CC tự thực thi code (Brain không touch).
  - §13 Lesson abstract thành Global Pattern (xem report).

## Lessons cần append vào `agent/memory/global/lessons.md`
3 Global Pattern (xem `report_2026-05-13_1004.md` mục Lessons).

## Next actions (suggest)
- Audit các handler khác trong `command_handler.go` còn dùng V1 repo
  hay không (`grep -n "h\.mappingRepo\." command_handler.go`).
- Audit các handler khác còn dùng `h.db.Exec` cho shadow target
  (`grep -nE "h\.db\.Exec.*shadow|h\.db\.Exec.*UPDATE" command_handler.go`).
- Backfill rule data_type=TIMESTAMP cho các bảng Mongo khác (cùng
  vấn đề epoch-ms).
- Append 3 Global Pattern vào `agent/memory/global/lessons.md`.
