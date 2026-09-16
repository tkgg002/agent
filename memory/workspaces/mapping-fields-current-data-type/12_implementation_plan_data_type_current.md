# Implementation Plan — Add "Data Type Current" Column to MappingFieldsPage

**Workspace:** `mapping-fields-current-data-type`  
**URL đích:** `http://localhost:5173/shadow/72/mappings?binding_id=190`  
**Date:** 2026-08-27

---

## 1. Phân tích Hiện trạng

### Cột hiện có trong bảng Mapping Rules:
| Cột | Nguồn dữ liệu | Ghi chú |
|-----|---------------|---------|
| Source Field | `rule.source_field` | Field từ source DB |
| Target Column | `rule.target_column` | Column trên shadow table |
| **Source Data Type** | `rule.source_data_type` | Type ở source (PostgreSQL/MongoDB) |
| **Data Type Target** | `rule.data_type` (editable) | Type đã configure trong mapping rule |
| Rule Type | `rule.rule_type` | system/discovered/mapping |
| Status | `rule.status` | pending/approved/rejected |
| In Shadow | từ `fetchShadowColumns()` | Check xem column đã tồn tại chưa |
| Active | `rule.is_active` | Toggle |
| Sensitive | `rule.is_sensitive_field` | Toggle |
| Mask Strategy | `rule.mask_strategy` | Select |

### Vấn đề:
- **Thiếu cột "Data Type Current"** = data type **thực tế đang có trên shadow column** trong PostgreSQL DB
- Hiện tại FE chỉ biết column có tồn tại không ("In Shadow") nhưng không biết type thực tế là gì
- Operator cần biết để so sánh với "Data Type Target" → phát hiện drift cần ALTER

### Nguồn dữ liệu:
`information_schema.columns` trên shadow DB → `SELECT column_name, data_type, udt_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?`

---

## 2. Phương án Thực thi

### Tầng Backend — `cdc-cms-service`

#### 2.1 Mở rộng `ShadowSchemaReader` interface
**File:** `internal/app/ports/repository.go` (line 285-287)

Thêm method mới **`GetColumnsWithTypes`** trả về `map[string]string` (column_name → data_type):

```go
type ShadowSchemaReader interface {
    GetColumns(ctx context.Context, schema, table string) ([]string, error)
    GetColumnsWithTypes(ctx context.Context, schema, table string) (map[string]string, error)
}
```

#### 2.2 Implement `GetColumnsWithTypes`
**File:** `internal/infra/persistence/shadow/shadow_schema_reader_gorm.go`

```go
func (r *shadowSchemaReaderGorm) GetColumnsWithTypes(ctx context.Context, schema, table string) (map[string]string, error) {
    var rows []struct {
        ColumnName string `gorm:"column:column_name"`
        DataType   string `gorm:"column:data_type"`
        UdtName    string `gorm:"column:udt_name"`
    }
    err := r.db.WithContext(ctx).Raw(`
        SELECT column_name,
               data_type,
               udt_name
        FROM information_schema.columns
        WHERE table_schema = ? AND table_name = ?
        ORDER BY ordinal_position`, schema, table).Scan(&rows).Error
    if err != nil {
        return nil, err
    }
    m := make(map[string]string, len(rows))
    for _, row := range rows {
        // udt_name cho custom type (e.g. "int8" thay vì "bigint")
        if row.DataType == "USER-DEFINED" || row.DataType == "ARRAY" {
            m[row.ColumnName] = row.UdtName
        } else {
            m[row.ColumnName] = strings.ToUpper(row.DataType)
        }
    }
    return m, nil
}
```

#### 2.3 Thêm endpoint mới: `GET /api/introspection/shadow-columns-with-types/:table?schema=...`
**File:** `internal/api/system/introspection_handler.go`

Thêm handler `ShadowColumnsWithTypes`:
```go
func (h *IntrospectionHandler) ShadowColumnsWithTypes(c *fiber.Ctx) error {
    targetTable := c.Params("table")
    schema := c.Query("schema")
    if schema == "" {
        return c.Status(400).JSON(fiber.Map{"error": "schema query param required"})
    }
    if h.shadowReader == nil {
        return c.Status(500).JSON(fiber.Map{"error": "shadow DB reader not configured"})
    }
    types, err := h.shadowReader.GetColumnsWithTypes(c.UserContext(), schema, targetTable)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.JSON(fiber.Map{
        "schema":  schema,
        "table":   targetTable,
        "columns": types,  // map[column_name]data_type
    })
}
```

#### 2.4 Đăng ký route mới
**File:** (file routes của server — cần kiểm tra, thường là `server.go` hoặc `router.go`)

```go
introspectionGroup.Get("/shadow-columns-with-types/:table", introspectionHandler.ShadowColumnsWithTypes)
```

---

### Tầng Frontend — `cdc-cms-web`

#### 2.5 Cập nhật `fetchShadowColumns` trong `MappingFieldsPage.tsx`

Thêm state mới `shadowColumnTypes: Record<string, string>` và fetch từ endpoint mới:

```typescript
const [shadowColumnTypes, setShadowColumnTypes] = useState<Record<string, string>>({});

const fetchShadowColumns = useCallback(async () => {
  if (!registry) return;
  const schema = registry.shadow_schema || '';
  if (!schema) {
    setShadowColumns(new Set());
    setShadowColumnTypes({});
    return;
  }
  try {
    // Gọi endpoint mới lấy column + type cùng lúc
    const { data } = await cmsApi.get(`/api/introspection/shadow-columns-with-types/${registry.target_table}`, {
      params: { schema },
    });
    const colsMap: Record<string, string> = data?.columns || {};
    setShadowColumnTypes(colsMap);
    setShadowColumns(new Set(Object.keys(colsMap).map(c => c.toLowerCase())));
  } catch {
    setShadowColumns(new Set());
    setShadowColumnTypes({});
  }
}, [registry]);
```

#### 2.6 Thêm cột "Data Type Current" vào table columns

Chèn sau cột "Data Type Target" (trước "Rule Type"):

```typescript
{
  title: (
    <Tooltip title="Kiểu dữ liệu thực tế đang tồn tại trên shadow column (lấy từ information_schema.columns). So sánh với 'Data Type Target' để phát hiện drift cần ALTER.">
      <span>Data Type Current</span>
    </Tooltip>
  ),
  key: 'current_data_type',
  width: 160,
  render: (_: unknown, record: MappingRule) => {
    const colKey = (record.target_column || '').toLowerCase();
    const currentType = shadowColumnTypes[colKey] || shadowColumnTypes[record.target_column || ''];
    if (!currentType) {
      return <Text type="secondary">—</Text>;
    }
    const targetType = (record.data_type || '').toUpperCase();
    const isDrift = currentType.toUpperCase() !== targetType;
    return (
      <Tag color={isDrift ? 'orange' : 'green'}>
        {currentType}
        {isDrift && <Tooltip title={`Target là ${targetType}, nhưng shadow column hiện là ${currentType}. Cần "Sync Fields" để ALTER.`}> ⚠️</Tooltip>}
      </Tag>
    );
  },
},
```

---

## 3. Điểm Kỹ thuật Quan trọng

### Drift Detection (bonus)
- Cột "Data Type Current" tự động highlight **màu orange** nếu `current_type ≠ target_type` → operator nhìn liền biết cần ALTER
- Màu **green** nếu đã khớp

### Backward Compatibility
- `GetColumns` giữ nguyên — `ShadowColumns` endpoint cũ không bị thay đổi
- FE gọi endpoint MỚI `/shadow-columns-with-types/...` thay vì endpoint cũ `/shadow-columns/...`
- Nếu endpoint mới lỗi → fallback về `{}` (state rỗng, cột "Data Type Current" hiển thị `—`)

### Lookup key
- Map key từ backend là **lowercase** (PostgreSQL column names case-insensitive)
- FE lookup: `shadowColumnTypes[colKey]` (lowercase), fallback `shadowColumnTypes[record.target_column]`

---

## 4. Files thay đổi

| File | Thay đổi | Service |
|------|----------|---------|
| `internal/app/ports/repository.go` | Thêm `GetColumnsWithTypes` vào interface | `cdc-cms-service` |
| `internal/infra/persistence/shadow/shadow_schema_reader_gorm.go` | Implement `GetColumnsWithTypes` | `cdc-cms-service` |
| `internal/api/system/introspection_handler.go` | Thêm `ShadowColumnsWithTypes` handler | `cdc-cms-service` |
| `internal/server/server.go` (hoặc router file) | Đăng ký route mới | `cdc-cms-service` |
| `src/pages/MappingFieldsPage.tsx` | Thêm state + fetch + column mới | `cdc-cms-web` |

**Tổng: 5 files** — không có DB migration, không có breaking change.

---

## 5. Verification Plan

1. `go build ./...` trong `cdc-cms-service` → pass
2. Restart `cdc-cms-service` → gọi `GET /api/introspection/shadow-columns-with-types/<table>?schema=<schema>` trực tiếp → verify response `{ columns: { col_name: "TEXT", ... } }`
3. Mở `http://localhost:5173/shadow/72/mappings?binding_id=190` → thấy cột "Data Type Current" mới
4. Verify: cột nào có `current ≠ target` → hiển thị orange tag + tooltip
5. Cột chưa có trong shadow → hiển thị `—`

---

## 6. Open Questions

Không có — phương án rõ ràng, không cần user lựa chọn.
