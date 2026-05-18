# 09 — Solution: schema-qualify ALTER COLUMN

## Strategy
Truyền `shadow_schema` xuyên suốt chain command → NATS payload → worker SQL.
Không động vào search_path/DSN (per lesson 2026-05-11 — role-level setting nguy hiểm).

## Diff dự kiến

### 1. `cdc-cms-service/internal/app/commands/source_async.go`
```go
type AlterColumnCommand struct {
	ports.AsyncCommandMixin
	TargetSchema string `json:"target_schema,omitempty"` // NEW
	TargetTable  string `json:"target_table"`
	ColumnName   string `json:"column_name"`
	DataType     string `json:"data_type"`
	Action       string `json:"action"`
}
```

### 2. `cdc-cms-service/internal/api/mapping_rule_handler_batch.go:48`
```go
schema := ""
if rule.ShadowSchema != nil { schema = *rule.ShadowSchema }
h.bus.Dispatch(ctx, commands.AlterColumnCommand{
	TargetSchema: schema,
	TargetTable:  *rule.ShadowTable,
	ColumnName:   rule.TargetColumn,
	DataType:     rule.DataType,
	Action:       "add",
})
```

### 3. `centralized-data-service/internal/handler/command_handler.go:1672`
- Add `TargetSchema string \`json:"target_schema"\`` vào payload struct
- Validate `isSafeIdent(payload.TargetSchema)` (cho phép empty để backward-compat)
- Build SQL với prefix khi schema có giá trị:
```go
qualified := fmt.Sprintf(`"%s"`, payload.TargetTable)
if payload.TargetSchema != "" {
	qualified = fmt.Sprintf(`"%s"."%s"`, payload.TargetSchema, payload.TargetTable)
}
sql = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS "%s" %s`, qualified, payload.ColumnName, payload.DataType)
```

## Backward compatibility
- Old payload (no `target_schema`) vẫn hoạt động — fallback về bare name (giữ behavior cũ).
- Mới: khi `target_schema` có, dùng qualified.

## Test plan
1. Build cả 2 service.
2. Restart worker (vì có file watcher đã restart auto khi save Go file).
3. Restart cms-service (manual — không có watcher đã verified ở session trước).
4. POST `/api/mapping-rules/batch` với 1 rule status=approved.
5. Đọc activity log → `alter-column success`.
6. Query `information_schema.columns` xác nhận cột mới tồn tại.
