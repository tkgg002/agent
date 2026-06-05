# 09_tasks_solution_default_collections — Code Demo chi tiết

> **Phase**: `default_collections`
> ⚠️ **Brain Code Prohibition (CLAUDE.md §12)**: File này là **GIẢI PHÁP MẪU**. Brain TUYỆT ĐỐI KHÔNG tự apply. Muscle nhận lệnh "execute" mới được edit code.
> ⚠️ **User directive**: "đảm bảo ko sửa code rồi hẵng chạy tiếp néh" — Edit dưới đây CHƯA được apply ở phase planning.

---

## Edit #1 — `SourceConnectors.tsx` Form.Item Collections

**File**: `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx`
**Location**: line ~966-969 (verify chính xác qua T1.1 grep trước khi edit)
**Risk**: LOW

### Before (current code)

```tsx
<Form.Item name="collectionNames" label="Collections">
  <Input placeholder="users,orders,payments" />
</Form.Item>
```

### After (proposed)

```tsx
<Form.Item
  name="collectionNames"
  label="Collections"
  extra="Để trống nếu muốn CDC toàn bộ collections của database. Phân cách bằng dấu phẩy nếu chỉ muốn CDC một số collection cụ thể (vd: users,orders)."
>
  <Input placeholder="users,orders (để trống = tất cả)" />
</Form.Item>
```

### Diff summary

- Thêm prop `extra` với hint text tiếng Việt.
- Update `placeholder` để consistent (optional — có thể giữ cũ nếu user thích).
- KHÔNG thay đổi `name`, KHÔNG thêm `rules`, KHÔNG đổi component `<Input>`.

### Edit instruction cho Muscle

Dùng tool `Edit`:

```yaml
file_path: data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx
old_string: |
  <Form.Item name="collectionNames" label="Collections">
    <Input placeholder="users,orders,payments" />
  </Form.Item>
new_string: |
  <Form.Item
    name="collectionNames"
    label="Collections"
    extra="Để trống nếu muốn CDC toàn bộ collections của database. Phân cách bằng dấu phẩy nếu chỉ muốn CDC một số collection cụ thể (vd: users,orders)."
  >
    <Input placeholder="users,orders (để trống = tất cả)" />
  </Form.Item>
```

**Note**: Nếu line indent của file khác (tabs vs spaces), Muscle PHẢI Read file trước, copy exact indent.

---

## Edit #2 — List view fallback `(All collections)`

**File**: Xác định ở M1.2. Hypothesis: cùng `SourceConnectors.tsx` (table columns section), HOẶC component riêng.
**Risk**: LOW

### Pattern: tìm column definition `'Collections'` hoặc `dataIndex: ['config', 'collection.include.list']`

### Before (giả định)

```tsx
{
  title: 'Collections',
  dataIndex: ['config', 'collection.include.list'],
  key: 'collections',
  render: (value: string | undefined) => value || '-',
}
```

### After (proposed)

```tsx
{
  title: 'Collections',
  dataIndex: ['config', 'collection.include.list'],
  key: 'collections',
  render: (value: string | undefined) => {
    if (!value) {
      return (
        <span style={{ color: '#999', fontStyle: 'italic' }}>
          (All collections)
        </span>
      );
    }
    return value;
  },
}
```

### Variants tùy actual code

**Variant A** — Nếu column dùng `record` thay vì `value`:

```tsx
render: (_: unknown, record: ConnectorRecord) => {
  const value = record.config?.['collection.include.list'];
  if (!value) {
    return <span style={{ color: '#999', fontStyle: 'italic' }}>(All collections)</span>;
  }
  return value;
}
```

**Variant B** — Nếu list view dùng `<Descriptions>` (detail view):

```tsx
<Descriptions.Item label="Collections">
  {config['collection.include.list'] || (
    <span style={{ color: '#999', fontStyle: 'italic' }}>(All collections)</span>
  )}
</Descriptions.Item>
```

### Diff summary

- Wrap render với conditional fallback.
- Style: italic gray (`#999`) để de-emphasize.
- KHÔNG đổi `dataIndex`, KHÔNG đổi data flow.

---

## Edit #3 (Optional) — i18n key extraction

**Conditional**: Chỉ apply nếu T1.4 confirm project có i18n setup.

### File `data-hub/cdc-cms-web/src/locales/vi.json` (HOẶC tương đương)

```diff
 {
   "connector": {
     "form": {
+      "collections": {
+        "extra": "Để trống nếu muốn CDC toàn bộ collections của database. Phân cách bằng dấu phẩy nếu chỉ muốn CDC một số collection cụ thể (vd: users,orders).",
+        "placeholder": "users,orders (để trống = tất cả)"
+      }
+    },
+    "list": {
+      "collections": {
+        "all": "(All collections)"
+      }
     }
   }
 }
```

### File `data-hub/cdc-cms-web/src/locales/en.json` (nếu có)

```diff
 {
   "connector": {
     "form": {
+      "collections": {
+        "extra": "Leave empty to CDC all collections of the database. Separate by comma to CDC specific collections only (e.g.: users,orders).",
+        "placeholder": "users,orders (empty = all)"
+      }
+    },
+    "list": {
+      "collections": {
+        "all": "(All collections)"
+      }
     }
   }
 }
```

### Use trong component

```tsx
import { useTranslation } from 'react-i18next';

const { t } = useTranslation();

<Form.Item
  name="collectionNames"
  label="Collections"
  extra={t('connector.form.collections.extra')}
>
  <Input placeholder={t('connector.form.collections.placeholder')} />
</Form.Item>
```

---

## NON-edit references — KHÔNG chỉnh sửa

### `compactConfig` — line ~131-133

```tsx
function compactConfig(cfg: Record<string, string>) {
  return Object.fromEntries(Object.entries(cfg).filter(([, value]) => value !== ''));
}
```

**Lý do giữ**: Đây chính là cơ chế tự nhiên drop empty key → BE không nhận → Debezium default. KHÔNG đụng.

### `buildConnectorConfig` — line ~160-166

```tsx
function buildConnectorConfig({ collectionNames, ... }: FormValues): Record<string, string> {
  const cfg: Record<string, string> = {
    // ... other keys
  };
  if (collectionNames) {
    cfg['collection.include.list'] = collectionNames;
  }
  return compactConfig(cfg);
}
```

**Lý do giữ**: Conditional `if (collectionNames)` đã đảm bảo không set key khi empty. Logic đúng. KHÔNG đụng.

### BE handler `system_connectors_handler.go:168-171`

```go
var req struct {
    Name   string            `json:"name"`
    Config map[string]string `json:"config"`
}
```

**Lý do giữ**: Accept map as-is, không inject default, forward to Kafka Connect. Đúng theo ADR-005. KHÔNG đụng.

---

## Test sketches

### Unit test Edit #1 (Jest + RTL)

**File**: `data-hub/cdc-cms-web/src/pages/__tests__/SourceConnectors.test.tsx` (CREATE nếu chưa có; nếu project chưa setup test → skip, dựa M4 smoke).

```tsx
import { render, screen } from '@testing-library/react';
import SourceConnectors from '../SourceConnectors';

describe('SourceConnectors form Collections field', () => {
  it('renders hint text below Collections input', () => {
    render(<SourceConnectors />);
    // Open create modal/page
    // ... user.click(screen.getByText('Create')) etc.
    expect(
      screen.getByText(/Để trống nếu muốn CDC toàn bộ collections/),
    ).toBeInTheDocument();
  });

  it('placeholder hints at empty = all', () => {
    render(<SourceConnectors />);
    const input = screen.getByPlaceholderText(/để trống = tất cả/);
    expect(input).toBeInTheDocument();
  });
});
```

### Unit test Edit #2 (list view render)

```tsx
describe('Collections column render', () => {
  it('renders (All collections) when value is empty', () => {
    const value = '';
    // ... call render function
    expect(result).toHaveTextContent('(All collections)');
  });

  it('renders (All collections) when value is undefined', () => {
    const value = undefined;
    expect(result).toHaveTextContent('(All collections)');
  });

  it('renders explicit list when value is provided', () => {
    const value = 'users,orders';
    expect(result).toHaveTextContent('users,orders');
    expect(result).not.toHaveTextContent('(All collections)');
  });
});
```

---

## Checklist trước khi commit

- [ ] Read file đầy đủ context, KHÔNG copy-paste blind.
- [ ] Verify exact line + indent qua Read trước Edit.
- [ ] Sau Edit, Read lại file để verify diff đúng.
- [ ] `pnpm build` PASS local.
- [ ] `pnpm lint` PASS local.
- [ ] `pnpm tsc --noEmit` PASS local.
- [ ] Manual smoke trên FE dev server.
- [ ] APPEND `05_progress.md` cho mỗi milestone.
- [ ] KHÔNG `git commit --no-verify`.
- [ ] KHÔNG `git push --force`.
- [ ] KHÔNG sửa BE / Debezium config.
- [ ] KHÔNG cheat DB.
