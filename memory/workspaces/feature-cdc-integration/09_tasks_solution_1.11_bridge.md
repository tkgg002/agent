# Solution: v1.11 Track E — Airbyte Bridge + Transform

> **Date**: 2026-04-08
> **Decision**: Hướng A — Bridge copy `_airbyte_raw_*` → CDC `_raw_data`

---

## E0: HandleAirbyteBridge SQL

```sql
-- Bridge: Copy from Airbyte raw table → CDC table
INSERT INTO {target_table} ({pk_field}, _raw_data, _source, _synced_at, _hash, _version)
SELECT 
  COALESCE(
    _airbyte_data->>'{pk_field}',          -- Try configured PK
    _airbyte_data->>'_id',                  -- MongoDB fallback
    _airbyte_data->>'id',                   -- Generic fallback
    _airbyte_ab_id::text                    -- Airbyte ID as last resort
  ),
  _airbyte_data,                            -- Full JSON → _raw_data
  'airbyte',
  COALESCE(_airbyte_emitted_at, NOW()),
  md5(_airbyte_data::text),
  1
FROM {airbyte_raw_table}
WHERE _airbyte_emitted_at > $1              -- Incremental: only new rows
ON CONFLICT ({pk_field}) DO UPDATE SET
  _raw_data = EXCLUDED._raw_data,
  _synced_at = EXCLUDED._synced_at,
  _hash = EXCLUDED._hash,
  _version = {target_table}._version + 1,
  _updated_at = NOW()
WHERE {target_table}._hash IS DISTINCT FROM EXCLUDED._hash;
```

## E1: HandleBatchTransform SQL

```sql
-- Transform: Apply mapping rules to populate typed columns
-- Generated dynamically per table from mapping_rules
UPDATE {target_table} SET
  {col1} = (_raw_data->>'{field1}')::{type1},
  {col2} = (_raw_data->>'{field2}')::{type2},
  ...
  _updated_at = NOW()
WHERE _raw_data IS NOT NULL
  AND ({col1} IS NULL OR {col2} IS NULL ...);
```

## Airbyte Raw Table Detection

Convention: `_airbyte_raw_{source_table}`
Verify: `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)`
