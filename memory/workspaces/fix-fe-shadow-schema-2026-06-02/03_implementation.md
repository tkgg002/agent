# 03_implementation — Fix FE shadow_schema (Architectural)

> Code demo cho 6 file: 2 BE + 4 FE. Tổng LOC: ~−45 (xoá nhiều hơn thêm).

## §1 — BE: `internal/infra/persistence/source_object_read_repo_gorm.go`

**EDIT** dòng 207-208 trong SQL:
```diff
-			sb.shadow_schema,
-			sb.physical_table_fqn,
+			COALESCE(sb.shadow_schema, '') AS shadow_schema,
+			COALESCE(sb.physical_table_fqn, '') AS physical_table_fqn,
```

## §2 — BE: `internal/app/queries/source_objects_read_models.go`

**EDIT** dòng 70-71:
```diff
-	ShadowSchema     *string   `json:"shadow_schema,omitempty"`
-	PhysicalTableFQN *string   `json:"physical_table_fqn,omitempty"`
+	ShadowSchema     string    `json:"shadow_schema"`
+	PhysicalTableFQN string    `json:"physical_table_fqn"`
```

**Note**: nếu có nơi khác đọc `*ReadModel.ShadowSchema` rồi deref `*model.ShadowSchema` → cần grep + sửa. Dự kiến 0 site vì model chỉ dùng cho c.JSON.

## §3 — FE: `src/pages/TableRegistry.tsx`

**DELETE** dòng 75-81 (function `normalizeShadowSchema`).

**EDIT** dòng 85:
```diff
-  const schema = record.shadow_schema || normalizeShadowSchema(record.source_db);
+  const schema = record.shadow_schema || '';
```

**EDIT** dòng 780:
```diff
-            schema={record.shadow_schema || normalizeShadowSchema(record.source_db)}
+            schema={record.shadow_schema || ''}
```

**EDIT** dòng 884:
```diff
-                      shadow_schema: record.shadow_schema || normalizeShadowSchema(record.source_db),
+                      shadow_schema: record.shadow_schema || '',
```

## §4 — FE: `src/pages/MappingFieldsPage.tsx`

**DELETE** dòng 14-17 (function `normalizeShadowSchema`).

**EDIT** dòng 24:
```diff
-  return `${registry.shadow_schema || normalizeShadowSchema(registry.source_db)}.${registry.target_table}`;
+  return `${registry.shadow_schema || ''}.${registry.target_table}`;
```

**EDIT** dòng 112:
```diff
-        shadow_schema: registry.shadow_schema || normalizeShadowSchema(registry.source_db),
+        shadow_schema: registry.shadow_schema || '',
```

**EDIT** dòng 198:
```diff
-        params.shadow_schema = registry.shadow_schema || normalizeShadowSchema(registry.source_db);
+        params.shadow_schema = registry.shadow_schema || '';
```

**EDIT** dòng 213 (`fetchShadowColumns` — critical bug site):
```diff
   const fetchShadowColumns = useCallback(async () => {
     if (!registry) return;
+    const schema = registry.shadow_schema || '';
+    if (!schema) {
+      setShadowColumns(new Set());
+      return;
+    }
     try {
-      const schema = registry.shadow_schema || normalizeShadowSchema(registry.source_db);
       const { data } = await cmsApi.get(`/api/introspection/shadow-columns/${registry.target_table}`, {
         params: { schema },
       });
```

**EDIT** dòng 357:
```diff
-          shadow_schema: registry?.shadow_schema || (registry ? normalizeShadowSchema(registry.source_db) : undefined),
+          shadow_schema: registry?.shadow_schema || undefined,
```

**EDIT** dòng 372:
```diff
-      shadow_schema: registry.shadow_schema || normalizeShadowSchema(registry.source_db),
+      shadow_schema: registry.shadow_schema || '',
```

**EDIT** dòng 572 (Descriptions item):
```diff
-          <Descriptions.Item label="Shadow Schema"><Text code>{registry.shadow_schema || normalizeShadowSchema(registry.source_db)}</Text></Descriptions.Item>
+          <Descriptions.Item label="Shadow Schema">
+            {registry.shadow_schema
+              ? <Text code>{registry.shadow_schema}</Text>
+              : <Text type="secondary">(chưa có)</Text>}
+          </Descriptions.Item>
```

## §5 — FE: `src/pages/DataIntegrity.tsx`

**DELETE** dòng 56-62 (function `normalizeShadowSchema`).

**EDIT** dòng 65:
```diff
-  return `${normalizeShadowSchema(record.source_db)}.${record.target_table}`;
+  return `${record.shadow_schema || ''}.${record.target_table}`;
```

**EDIT** dòng 81:
```diff
-  return `${normalizeShadowSchema(record.source_db || 'unknown')}.${record.target_table}`;
+  return `${record.shadow_schema || ''}.${record.target_table}`;
```

## §6 — FE: `src/pages/ActivityManager.tsx`

**DELETE** dòng 65 (inline arrow `normalizeShadowSchema`).

**EDIT** dòng 135:
```diff
-        label: `${row.source_db}.${row.source_table} -> ${(row.shadow_schema || normalizeShadowSchema(row.source_db))}.${row.target_table}`,
+        label: `${row.source_db}.${row.source_table} -> ${(row.shadow_schema || '')}.${row.target_table}`,
```

**EDIT** dòng 138:
```diff
-        shadowSchema: row.shadow_schema || normalizeShadowSchema(row.source_db),
+        shadowSchema: row.shadow_schema || '',
```

**EDIT** dòng 176:
```diff
-            <Text type="secondary" code>{normalizeShadowSchema(meta.source_db)}.{meta.target_table}</Text>
+            <Text type="secondary" code>{(meta.shadow_schema || '')}.{meta.target_table}</Text>
```
> Note: `meta` cần có field `shadow_schema` — nếu type chưa có, thêm `shadow_schema?: string` ở interface meta dòng ~30.

## §7 — FE: `src/types/index.ts`

**EDIT** dòng 112:
```diff
-  shadow_schema: string;
+  shadow_schema: string; // always present from BE (may be empty)
```
> Field đã `string` rồi — chỉ thêm comment để rõ contract. Dòng 67 và 134 giữ `string | null` cho ReconReport.

## LOC summary
| File | Delete | Add | Net |
|------|--------|-----|-----|
| source_object_read_repo_gorm.go | 2 | 2 | 0 |
| source_objects_read_models.go | 2 | 2 | 0 |
| TableRegistry.tsx | 10 | 3 | −7 |
| MappingFieldsPage.tsx | 12 | 8 | −4 |
| DataIntegrity.tsx | 9 | 2 | −7 |
| ActivityManager.tsx | 4 | 3 | −1 |
| types/index.ts | 0 | 1 | +1 |
| **Total** | **39** | **21** | **−18** |
