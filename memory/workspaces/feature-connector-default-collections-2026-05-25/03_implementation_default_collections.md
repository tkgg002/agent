# 03_implementation_default_collections — Technical Design

> **Phase**: `default_collections`
> **Strategy**: Phương án A (FE-only hint)
> **Audience**: Muscle thực thi + reviewer.

---

## 1. High-level data flow (giữ NGUYÊN, KHÔNG đổi runtime)

```
┌────────────────────────────────────────────────────────────────────┐
│  User UI (cdc-cms-web SourceConnectors.tsx)                        │
│                                                                    │
│  Form.Item name="collectionNames" (free-text, optional)            │
│    ↓ user submits                                                  │
│  buildConnectorConfig({ collectionNames, ... })                    │
│    ↓ collectionNames === '' → key 'collection.include.list' = ''   │
│  compactConfig(cfg)                                                │
│    ↓ DROP entries where value === ''                               │
│  POST /api/system-connectors  body: { name, config: {...} }        │
│         (NO 'collection.include.list' key when user left empty)    │
└────────────────────────────────────────┬───────────────────────────┘
                                         ▼
┌────────────────────────────────────────────────────────────────────┐
│  BE cdc-cms-service system_connectors_handler.go                   │
│                                                                    │
│  - Accept config map as-is                                         │
│  - NO inject default                                               │
│  - Forward to Kafka Connect REST                                   │
└────────────────────────────────────────┬───────────────────────────┘
                                         ▼
┌────────────────────────────────────────────────────────────────────┐
│  Kafka Connect REST → Debezium Mongo connector instance            │
│                                                                    │
│  config.collection.include.list = (MISSING)                        │
│    ↓ Debezium default behavior                                     │
│  CDC ALL collections of database matching database.include.list    │
└────────────────────────────────────────────────────────────────────┘
```

**Key invariant**: Runtime correctness DEPENDS ON ba điều kiện:
1. FE `compactConfig` filter empty value (đã có, KHÔNG đụng).
2. BE handler accept map as-is, không inject default (đã đúng, KHÔNG đụng).
3. Debezium connector `collection.include.list` mặc định = all khi missing (cần M0 verify version).

Nếu cả 3 đúng → R2 (CDC all when empty) hoạt động ngay. Phase này CHỈ thêm UX hint.

## 2. Code changes (FE only)

### 2.1 `SourceConnectors.tsx` — Form.Item Collections (line 966-969)

**Before**:
```tsx
<Form.Item name="collectionNames" label="Collections">
  <Input placeholder="users,orders,payments" />
</Form.Item>
```

**After (proposed — xem `09_tasks_solution` Edit #1)**:
```tsx
<Form.Item
  name="collectionNames"
  label="Collections"
  extra="Để trống nếu muốn CDC toàn bộ collections của database. Phân cách bằng dấu phẩy nếu chỉ muốn CDC một số collection cụ thể (vd: users,orders)."
>
  <Input placeholder="users,orders (để trống = tất cả)" />
</Form.Item>
```

**Rationale**:
- `extra` là Antd-native API, render helper text bên dưới input, đã có ARIA association sẵn (N5).
- Placeholder updated nhẹ để consistent với hint (không bắt buộc).
- KHÔNG dùng `tooltip` vì user phải hover mới thấy → giảm UX.

### 2.2 List view component — display fallback

**Audit task M1.2**: chưa biết exact file. Hypothesis common candidate:
- `SourceConnectors.tsx` table column "Collections"
- HOẶC component riêng `ConnectorDetail.tsx` / `ConnectorList.tsx`

**Pattern edit** (apply khi tìm thấy):

**Before** (giả định):
```tsx
{
  title: 'Collections',
  dataIndex: ['config', 'collection.include.list'],
  render: (value: string | undefined) => value || '-',
}
```

**After** (proposed — xem `09_tasks_solution` Edit #2):
```tsx
{
  title: 'Collections',
  dataIndex: ['config', 'collection.include.list'],
  render: (value: string | undefined) => {
    if (!value) {
      return <span style={{ color: '#999', fontStyle: 'italic' }}>(All collections)</span>;
    }
    return value;
  },
}
```

**Rationale**:
- Phân biệt visual rõ ràng giữa "chưa cấu hình filter" (intentional all) vs "đã filter".
- KHÔNG đụng data flow, chỉ render.

## 3. Schema changes

**KHÔNG có**. Phase này không đụng DB.

## 4. Migration changes

**KHÔNG có**. Phase này không tạo migration.

## 5. Backward compatibility

| Aspect | Status |
|---|---|
| Connector cũ với explicit `collection.include.list = "a,b,c"` | ✅ Render giữ nguyên (value truthy, fallback không trigger) |
| Connector cũ với `collection.include.list = ""` (string rỗng) | ✅ Fallback render `(All collections)` — semantically đúng |
| Connector cũ với `collection.include.list = null` / undefined | ✅ Fallback render `(All collections)` |
| Form edit existing connector | ✅ `extra` text vô hại |
| API contract BE / Kafka Connect | ✅ KHÔNG đổi |

## 6. Performance impact

| Aspect | Impact |
|---|---|
| FE render | Negligible (1 extra string per row) |
| BE request size | KHÔNG đổi |
| Network | KHÔNG đổi |
| Debezium runtime | KHÔNG đổi |

## 7. Observability

KHÔNG cần thêm log / metric. UI-only change.

## 8. Security

- `extra` text là plain string, Antd auto-escape → KHÔNG có XSS risk.
- `render` function nhận `value` từ config map (controlled by Ops admin) → không user-untrusted input.
- KHÔNG đụng auth / authz / DSN / credential.
- `/security-agent` run ở M5 vẫn bắt buộc theo CLAUDE.md §8.

## 9. i18n consideration

- Codebase audit M1.4 sẽ xác định.
- Nếu có i18n: thêm key `connector.form.collections.extra` + `connector.list.collections.all`.
- Nếu chưa có: hardcode tiếng Việt theo CLAUDE.md §0.

## 10. Verification commands (cho Muscle khi thực thi)

```bash
# M3 build
cd data-hub/cdc-cms-web
pnpm install
pnpm build 2>&1 | tee /tmp/default_collections_build.log
pnpm lint 2>&1 | tee /tmp/default_collections_lint.log
pnpm tsc --noEmit 2>&1 | tee /tmp/default_collections_tsc.log

# M4 smoke (manual, cần local stack)
# 1. Start FE dev
pnpm dev

# 2. Verify connector config qua Kafka Connect REST
curl -s http://localhost:8083/connectors/<connector-name>/config | jq '.["collection.include.list"]'
# Expect: null

# 3. Mongo insert vào collection mới
mongosh "<uri>" --eval 'db.brand_new_test_coll.insertOne({_id:"test1", marker:"default_collections_smoke"})'

# 4. Verify topic
kafkacat -b localhost:9092 -t cdc.<server>.<db>.brand_new_test_coll -C -e | head -5
```

## 11. Rollback plan

- **Code rollback**: `git revert <commit_hash>` trên `data-hub/cdc-cms-web/`.
- **No DB rollback** (không có migration).
- **No infra rollback** (không đụng Kafka Connect / Debezium).
- Behavior fall về cũ (placeholder + không có hint) — runtime vẫn đúng.

## 12. Future work (defer phase sau)

- Phase `connector-collection-picker`: gọi BE endpoint `GET /api/mongo/collections?uri=...` → render multi-select checkbox.
- Phase `connector-filter-validate`: validate format `db.collection,db.collection` (chuẩn Debezium 1.x).
- Phase `connector-default-display-unified`: extend pattern `(All X)` cho các field default-all khác (vd `database.include.list`).
