# 02_plan — Detailed implementation plan

## BE-1: Reset status to pending on scan
**File**: `cdc-cms-service/internal/api/introspection_handler.go`
**File**: `cdc-cms-service/internal/server/server.go` (inject `db`)

### Code demo
```go
// IntrospectionHandler nhận thêm db (control plane) + shadowDB (data plane).
type IntrospectionHandler struct {
    natsClient *natsconn.NatsClient
    db         *gorm.DB
    shadowDB   *gorm.DB
}

func NewIntrospectionHandler(natsClient *natsconn.NatsClient, db, shadowDB *gorm.DB) *IntrospectionHandler {
    return &IntrospectionHandler{natsClient: natsClient, db: db, shadowDB: shadowDB}
}

func (h *IntrospectionHandler) ScanRawData(c *fiber.Ctx) error {
    targetTable := c.Params("table")
    // ... (existing NATS subscribe/publish/wait) ...

    var res map[string]interface{}
    json.Unmarshal(msg.Data, &res)

    // Reset mapping rules to pending — scan = re-discover handshake.
    // Operator yêu cầu rõ: mỗi lần scan, status quay về pending để tránh
    // carry over từ lifecycle cũ của shadow table (rebuild, drop+create).
    username, _ := c.Locals("username").(string)
    if h.db != nil {
        h.db.WithContext(c.Context()).Exec(`
            UPDATE cdc_system.mapping_rule_v2
            SET status = 'pending', is_active = false, updated_by = ?, updated_at = NOW()
            WHERE source_object_id IN (
                SELECT sb.source_object_id
                FROM cdc_system.shadow_binding sb
                WHERE sb.shadow_table = ? AND sb.is_active = TRUE
            )`, username, targetTable)
    }

    return c.JSON(res)
}
```

## BE-2: New endpoint shadow-columns
**File**: `cdc-cms-service/internal/api/introspection_handler.go` (cùng handler, dùng `shadowDB`)
**File**: `cdc-cms-service/internal/router/router.go`

### Code demo
```go
func (h *IntrospectionHandler) ShadowColumns(c *fiber.Ctx) error {
    targetTable := c.Params("table")
    schema := c.Query("schema")
    if schema == "" {
        return c.Status(400).JSON(fiber.Map{"error": "schema query param required"})
    }
    var cols []string
    if err := h.shadowDB.WithContext(c.Context()).Raw(`
        SELECT column_name FROM information_schema.columns
        WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position
    `, schema, targetTable).Scan(&cols).Error; err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.JSON(fiber.Map{
        "schema":  schema,
        "table":   targetTable,
        "columns": cols,
    })
}
```

Route: `dualGet(shared, "/introspection/shadow-columns/:table", introspectionHandler.ShadowColumns)`

## FE-1: Expand SYSTEM_DEFAULT_FIELDS to 11 + in_shadow indicator
**File**: `cdc-cms-web/src/pages/MappingFieldsPage.tsx`

### Code demo
```ts
const SYSTEM_DEFAULT_FIELDS = [
  { field: '<PK>', type: '<auto>', description: 'Primary key (renamed from source PK)' },
  { field: 'source_id', type: 'TEXT', description: 'V2 ON CONFLICT key (UNIQUE)' },
  { field: '_raw_data', type: 'JSONB', description: 'Full raw event data from source' },
  { field: '_source', type: 'VARCHAR', description: 'Data source identifier (debezium)' },
  { field: '_source_ts', type: 'BIGINT', description: 'OCC anchor — older-wins guard' },
  { field: '_synced_at', type: 'TIMESTAMP', description: 'When the record was last synced' },
  { field: '_version', type: 'BIGINT', description: 'Record version for conflict resolution' },
  { field: '_hash', type: 'VARCHAR', description: 'SHA256 hash for dedup detection' },
  { field: '_deleted', type: 'BOOLEAN', description: 'Soft delete flag' },
  { field: '_created_at', type: 'TIMESTAMP', description: 'Record creation time in DW' },
  { field: '_updated_at', type: 'TIMESTAMP', description: 'Last update time in DW' },
];
```

Pick PK dynamically from `registry.primary_key_field`. Render `id` if PK is `_id`.

State: `shadowColumns: string[]` (lowercase set for case-insensitive check). Fetch via new endpoint after registry resolved + after scan success.

Render: dùng `<Tag color="green">in shadow</Tag>` hoặc `<Tag>not yet</Tag>` ở System Default Fields tile + cột mới trong rules table.

## FE-2: Refetch sau scan
Trong `handleScan`, sau `setNewFields(...)`, gọi `fetchRules()` + `fetchShadowColumns()`.

## Order of execution
1. Write workspace docs ✅
2. BE inject db/shadowDB into IntrospectionHandler + Constructor + Server wiring
3. BE add reset logic in ScanRawData + ShadowColumns endpoint
4. BE add router for ShadowColumns
5. FE update SYSTEM_DEFAULT_FIELDS + fetch shadow columns + indicators + refetch on scan
6. Build BE worker + BE cms + FE (tsc)
7. Append progress + Write report_*.md
