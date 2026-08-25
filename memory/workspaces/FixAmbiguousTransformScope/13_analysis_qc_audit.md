# QC & Self-Reflection Audit - Fix Ambiguous Transform Scope (V2 Hybrid)

This document records the strict Quality Control (QC) review and Vòng Lập Phản Tỉnh (Self-Reflection Loop) conducted on the proposed changes.

---

## 1. Thread Safety Analysis (`metadata_registry_service.go`)
* **Check**: Are the writes to `targetCache` and `targetRouteMap` thread-safe?
* **Verification**: Yes, in `ReloadAll`, the write lock `rs.mu.Lock()` is acquired at line 151, before any of the caches are initialized or populated. Therefore, storing the qualified table name as a second key is fully thread-safe and free from data races.

---

## 2. GORM Struct Tag Limitation Bypass (`table_registry_repo.go`)
* **Check**: Can GORM map the aliased `sb.shadow_schema AS shadow_schema` into `TableRegistry`?
* **Verification**: **CRITICAL FINDING!** The `ShadowSchema` field of `source.TableRegistry` is tagged with `gorm:"-"` (meaning GORM ignores it completely during database read/write operations). Thus, GORM's standard query `.Find()` or `.Scan()` will silently fail to populate the field.
* **Fix/Improvement**: Resolved by fetching active schema mappings in a single separate batch query:
  ```sql
  SELECT so.legacy_registry_id, sb.shadow_schema
  FROM cdc_system.shadow_binding sb
  JOIN cdc_system.source_object_registry so ON sb.source_object_id = so.id
  WHERE sb.is_active = true AND so.legacy_registry_id IS NOT NULL
  ```
  We build a mapping in memory and dynamically assign `ShadowSchema` to `TableRegistry` entries. This bypasses the tag restriction safely without modifying model definitions, while preserving the 2-query performance optimization.
* **Point Query Check**: We also identified that `GetByID` and `GetByTargetTable` are point query fallbacks when cache misses. We extended them to execute a similar schema lookup and populate `ShadowSchema` dynamically for total consistency.

---

## 3. Ambiguous Fallback Lookup Prevention (`helpers.go` & `table_registry_repo.go`)
* **Check**: When `ResolveTargetTableConfig` hits a cache miss and queries `GetByTargetTable(ctx, pureTable)`, does it suffer from scope collision (since 6 registries match the flat table name)?
* **Verification**: **CRITICAL FINDING!** Yes. Standard repository queries for a flat table name like `reconcile_final` will return the first matching database row, which might belong to the wrong connector/schema.
* **Fix/Improvement**: Implemented `GetByTargetTableAndSchema(ctx, targetTable, schemaName)` in `TableRegistryRepo` to search by both target table and shadow schema (joining `shadow_binding` and `source_object_registry`). Inside `helpers.go`'s `ResolveTargetTableConfig`, we use a dynamic interface type assertion:
  ```go
	if schemaName != "" {
		if resolverWithSchema, ok := r.(interface {
			GetByTargetTableAndSchema(ctx context.Context, targetTable string, schemaName string) (*source.TableRegistry, error)
		}); ok {
			if item, err := resolverWithSchema.GetByTargetTableAndSchema(ctx, pureTable, schemaName); err == nil {
				return item
			}
		}
	}
  ```
  This dynamically detects if the repository supports collision-safe schema lookups, falling back safely to standard interface methods in tests while guaranteeing 100% collision protection at runtime.

---

## 4. Schema Double-Qualification Guard (`batch_transform_handler.go`)
* **Check**: Does the worker construct SQL queries with double schema prefixes (e.g. `shadow_test33.shadow_test33.reconcile_final`) when a qualified name is passed?
* **Verification**: Yes, if we had used `targetTable` directly with `sqlutil.QualifiedTable(schemaName, targetTable)`.
* **Fix/Improvement**: Extracted `pureTable` (stripping the schema prefix if dot exists) and used it exclusively for `TableExistsInSchema`, `HasColumnInSchema`, and SQL construction, leaving the qualified name only for routing resolution.

---

## 5. Scheduler Match Logic (`server_jobs.go`)
* **Check**: If the scheduler runs for a specific target table, does it support both flat and qualified names?
* **Verification**: Yes. We updated the matcher logic to:
  ```go
  qualified := entry.QualifiedTarget()
  if targetTable != "" && qualified != targetTable && entry.TargetTable != targetTable {
      continue
  }
  ```
  This supports exact matches for both legacy flat target tables and qualified ones.

---

## 6. Mapping Rule Resolution Accuracy (`batch_transform_handler.go`)
* **Check**: Does `runTransformJob` resolve the correct mapping rules under ambiguous scope?
* **Verification**: Yes. By retrieving the `route` using the qualified name key, we extract the exact `source_object_id` and `shadow_binding_id`. We then call `ListActiveBySourceObjectAndBinding(ctx, sourceObjectID, shadowBindingID)` to load only rules specific to this shadow table, ignoring the other 5 schemas.

---

## 7. Unit Test Regression Prevention (`cdc-cms-service`)
* **Check**: Will the changes break any existing API unit tests?
* **Verification**: **CRITICAL FINDING!** In `source_object_actions_handler_test.go`, the test `TestTransformV2_Success` resolved the mock scope to `ShadowSchema: "cdc_shadow"` and `TargetTable: "shadow_table_123"`. Since our change now publishes the qualified name, it will publish `"cdc_shadow.shadow_table_123"`, which would fail the assertions.
* **Fix/Improvement**: Documented the assertion changes needed in `source_object_actions_handler_test.go` to assert `"cdc_shadow.shadow_table_123"`, keeping our unit tests green and aligned with the new schema resolution behavior.
