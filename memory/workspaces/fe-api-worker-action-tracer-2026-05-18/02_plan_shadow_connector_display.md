# Plan — Shadow Page Connector Display

**Phase**: fe-api-worker-action-tracer-2026-05-18 / shadow_connector_display
**Date**: 2026-05-19
**Status**: PLANNED (chờ apply)

## Vấn đề (Symptom)

`http://localhost:5173/shadow` panel header hiển thị `"Source Database: centralized-export-service · 1 objects"`. Không có thông tin connector nào sở hữu source DB này. 2 connector cùng `source_db` sẽ collapse vào 1 panel duy nhất (hệ quả: identity-tier-discriminator fix backend đã xong nhưng UX layer không phản ánh được).

## Root cause

1. **Backend list response** (`/api/v1/source-objects` + `/api/v1/shadow-bindings`) projection thiếu `source_connection_id` + `source_connection_code` → FE không có data.
2. **FE grouping key** chỉ dùng `source_db` (file `TableRegistry.tsx:300, 310`) → multi-connector cùng db merge vào 1 group.
3. **FE columns** không có column "Connector" → row-level cũng không phân biệt.

## Files thay đổi (4 file)

### Backend Go (3 file)

#### B1. `cdc-cms-service/internal/app/queries/source_objects_read_models.go`

Thêm 2 field vào `SourceObjectListItem` (giữa `ObjectCode` và `SourceDB` để duy trì hierarchical reading):

```go
type SourceObjectListItem struct {
    ID                   int64     `json:"id"`
    RegistryID           *uint     `json:"registry_id,omitempty"`
    ShadowBindingID      *int64    `json:"shadow_binding_id,omitempty"`
    ObjectCode           string    `json:"object_code"`
    SourceConnectionID   *int64    `json:"source_connection_id,omitempty"`    // NEW
    SourceConnectionCode string    `json:"source_connection_code,omitempty"`  // NEW
    SourceDB             string    `json:"source_db"`
    // ... rest unchanged
}
```

#### B2. `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`

- Mở rộng `listBaseFromWhere` (line 36): thêm `LEFT JOIN cdc_system.connection_registry cn ON cn.id = so.source_connection_id` (sau JOIN shadow_binding).
- Mở rộng projection trong `ListEnriched` (line 108): bổ sung 2 cột:
  ```sql
  so.source_connection_id,
  COALESCE(cn.connection_code, '') AS source_connection_code,
  ```
- `ORDER BY` (line 152): đổi thành `ORDER BY COALESCE(cn.connection_code, ''), so.source_database, so.source_object_name` để rows stable per connector.

#### B3. `cdc-cms-service/internal/api/source_objects_handler.go`

- Mở rộng struct `ShadowBindingRow` (line 59): thêm `SourceConnectionID *int64` + `SourceConnectionCode string` cùng JSON tag.
- Mở rộng `ListShadowBindings` SQL (line 268): thêm `LEFT JOIN cdc_system.connection_registry cn ON cn.id = so.source_connection_id`; project `so.source_connection_id, COALESCE(cn.connection_code, '') AS source_connection_code`; `ORDER BY` bao gồm `cn.connection_code`.

### FE TS (2 file)

#### F1. `cdc-cms-web/src/types/index.ts`

`SourceObjectRow` (line 54) + `ShadowBindingRow` (line 93) — mỗi interface thêm 2 optional field:

```ts
source_connection_id?: number | null;
source_connection_code?: string | null;
```

#### F2. `cdc-cms-web/src/pages/TableRegistry.tsx`

**(F2a) Grouping logic** (line 297-315):

```ts
const groupedData = useMemo(() => {
  const groups: Record<string, TRegistry[]> = {};
  filteredByEngine.forEach(item => {
    const conn = item.source_connection_code || '(unassigned)';
    const db = item.source_db || 'unknown';
    const key = `${conn}::${db}`;
    if (!groups[key]) groups[key] = [];
    groups[key].push(item);
  });
  return groups;
}, [filteredByEngine]);
```

Same pattern cho `groupedBindings`.

**(F2b) Panel header** (line 786, 823):

```tsx
const [conn, db] = key.split('::', 2);
return (
  <Panel header={
    <Space>
      <DatabaseOutlined style={{ color: '#1890ff' }} />
      <span style={{ fontWeight: 600 }}>Connector:</span>
      <Tag color="purple">{conn}</Tag>
      <span style={{ fontWeight: 600 }}>Source Database:</span>
      <Tag color="geekblue">{db}</Tag>
      <Tag color="blue">{tables.length} objects</Tag>
    </Space>
  } key={key} ...>
```

**(F2c) Column "Connector"** thêm vào `columns` (trước `Source DB`):

```tsx
{ title: 'Connector', dataIndex: 'source_connection_code', width: 140,
  render: (v?: string | null) => v ? <Tag color="purple">{v}</Tag> : <Tag>(unassigned)</Tag>
},
```

Same column thêm vào `bindingColumns`.

## Verification

| Gate | Command | Expected |
|---|---|---|
| CMS build | `cd cdc-cms-service && go build ./...` | EXIT=0 |
| CMS vet | `go vet ./...` | EXIT=0 |
| CMS test | `go test -count=1 ./internal/infra/persistence/... ./internal/api/... ./internal/app/queries/...` | PASS |
| FE typecheck | `cd cdc-cms-web && npx tsc --noEmit -p tsconfig.app.json` | không có lỗi mới ngoài pre-existing (Upload, UploadOutlined, handleBulkImport) |
| FE lint | `npx eslint src/pages/TableRegistry.tsx src/types/index.ts` | không có lỗi mới |

## Non-goal (out of scope)

- Không apply migration mới (đã xong trong phase trước).
- Không re-fix FE register form (đã xong).
- Không sửa worker (worker không touch /shadow page).

## Backout

Revert 4 file (3 Go + 1 TS + 1 TSX). Backend JSON tag mới `omitempty` → backwards-compat: client cũ ignore unknown field.
