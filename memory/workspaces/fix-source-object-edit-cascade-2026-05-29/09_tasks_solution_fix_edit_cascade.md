# 09_tasks_solution_fix_edit_cascade

## Solution: FE-only patch trong `TableRegistry.tsx`

### Patch site 1 — useMemo firstRowIndexBySource

**Vị trí**: sau state `actionLoadingId` (component `TableRegistry`).

```tsx
// Index của row đầu tiên cho mỗi source_object_id trong `data`.
// is_active + các field source-level cascade lên TẤT CẢ binding của
// cùng source, nên nút "Sửa" chỉ cần xuất hiện 1 lần/source.
const firstRowIndexBySource = useMemo(() => {
  const map = new Map<number, number>();
  data.forEach((r, i) => {
    if (r.id != null && !map.has(r.id)) map.set(r.id, i);
  });
  return map;
}, [data]);
```

### Patch site 2 — Column "Thao tác" render

**Vị trí**: column `Thao tác` (cuối columns array).

```tsx
render: (_, record, index) => {
  const showEdit = record.id != null && firstRowIndexBySource.get(record.id) === index;
  const sourceActive = Boolean(record.is_active);
  const sourceDisabledHint = sourceActive ? undefined : 'Bật "Kích hoạt debezium Sync" để dùng';
  return (
    <Space orientation="vertical" size={4} onClick={e => e.stopPropagation()}>
      <Space wrap>
        {showEdit && (
          <Button size="small" icon={<EditOutlined />} onClick={(e) => openEdit(e, record)}>Sửa</Button>
        )}
        <Tooltip title={sourceDisabledHint}>
          <Button size="small" icon={<ThunderboltOutlined />} type="primary" ghost
            disabled={!sourceActive}
            loading={actionLoadingId === record.id}
            onClick={(e) => handleSnapshot(e, record)}>Snapshot</Button>
        </Tooltip>
        <Tooltip title={sourceDisabledHint}>
          <Button size="small" icon={<RocketOutlined />}
            disabled={!sourceActive}
            onClick={(e) => { /* navigate masters */ }}>
            Manage Masters
          </Button>
        </Tooltip>
      </Space>
    </Space>
  );
}
```

### Patch site 3 — AsyncRowActions effectiveActive

**Vị trí**: trong `AsyncRowActions` sau `canUseScan`.

```tsx
// Yêu cầu binding "Trạng thái table" đang bật mới quét field được.
const effectiveActive = record.shadow_binding_id != null
  ? Boolean(record.shadow_binding_is_active)
  : Boolean(record.is_active);
```

### Patch site 4 — Quét field button

```tsx
<Tooltip title={!effectiveActive ? 'Bật "Trạng thái table" để quét field' : undefined}>
  <Button
    size="small"
    icon={<SearchOutlined />}
    disabled={!canUseScan || !effectiveActive}
    loading={scanBusy}
    onClick={(e) => { e.stopPropagation(); openConfirm('scan-fields'); }}
  >
    Quét field
  </Button>
</Tooltip>
// ...
{!effectiveActive && canUseScan && <Tag color="default">Binding chưa active</Tag>}
```

## Files changed
| # | File | LOC delta |
|---|------|-----------|
| 1 | `cdc-cms-web/src/pages/TableRegistry.tsx` | +52 / -25 (NET +27) |

## Verify
- `npm run build` PASS — `built in 742ms`, TypeScript clean.
- Bundle `TableRegistry-DlqC4fpm.js` 24.38 kB gzip 8.07 kB (size healthy).
