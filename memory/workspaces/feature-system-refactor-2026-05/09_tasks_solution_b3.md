# B3 — Tasks Solution (concrete diff per task)

## Task 105 — `system_health_collector.go:267`

**Before**:
```go
url := strings.TrimRight(c.cfg.WorkerURL, "/") + "/health"
```

**After**:
```go
url := strings.TrimRight(c.cfg.WorkerURL, "/") + "/healthz"
```

Lý do: admin-api `/healthz` (no-auth) trả `{"ok":true}` dùng cho dev probe; `/health` đã auth-gate Phase F1 → 401. Đổi sang `/healthz` để probe không bị 401.

## Task 106 — `prom_client.go:200` graceful 401

**Before** (đoạn xử lý error):
```go
req, err := http.NewRequestWithContext(ctxHTTP, http.MethodGet, p.workerURL+"/metrics", nil)
// ... fetch + parse logic
if resp.StatusCode >= 400 {
    return nil, fmt.Errorf("worker /metrics http %d", resp.StatusCode)
}
```

**After**:
```go
req, err := http.NewRequestWithContext(ctxHTTP, http.MethodGet, p.workerURL+"/metrics", nil)
// ... fetch + parse logic
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    p.logger.Debug("worker /metrics requires auth (Phase F1) — fallback to empty stats",
        zap.Int("status", resp.StatusCode))
    return &Stats{}, nil
}
if resp.StatusCode >= 400 {
    return nil, fmt.Errorf("worker /metrics http %d", resp.StatusCode)
}
```

Lý do: chấp nhận 401 thay vì biến thành "critical alert". Khi sau Brain wire admin token thì có thể bỏ branch này.

## Task 107 — `cdc-cms-service/Makefile` migrate target

Cần inspect Makefile hiện tại trước khi viết diff. Solution placeholder:

**Before** (suy đoán theo lesson auth Makefile drift):
```makefile
migrate:
	docker exec -i gpay-postgres psql -U user -d goopay_dw < migrations/001_init.sql
```

**After**:
```makefile
migrate:
	docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw < migrations/001_init.sql
```

(Áp credentials đúng theo config-local.yml: PG cdc-metadata 5433, db `cdc_dw`, user `gpay_admin`. Sẽ confirm trong execution step.)

## Task 108 — `config-local.yml` clear kafkaExporterUrl

**Before**:
```yaml
system:
  ...
  kafkaExporterUrl: "http://localhost:9308/metrics"
```

**After**:
```yaml
system:
  ...
  # kafka_exporter sidecar deferred to B4. Redpanda Console v2.7.2
  # at :18088 hiển thị consumer lag UI; metric automation defer.
  kafkaExporterUrl: ""
```

## Task 109 — `schema_adapter.go::PrepareForCDCInsertInSchema` auto-CREATE

**Insert before existing nil-check error return** (per plan curried-waddling-spindle P2):

```go
func (sa *SchemaAdapter) PrepareForCDCInsertInSchema(schemaName, tableName, pkColumn string) error {
    schema, err := sa.GetSchemaInSchema(schemaName, tableName)
    if err != nil {
        return err
    }
    if schema == nil {
        if err := sa.createShadowTableV1(schemaName, tableName, pkColumn); err != nil {
            return fmt.Errorf("create shadow table %s.%s: %w", schemaName, tableName, err)
        }
        schema, err = sa.loadSchemaInSchema(schemaName, tableName)
        if err != nil {
            return err
        }
    }
    // ... existing ALTER ADD COLUMN IF NOT EXISTS loop unchanged ...
}

func (sa *SchemaAdapter) createShadowTableV1(schemaName, tableName, pkColumn string) error {
    ddl := []string{
        fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %q`, schemaName),
        fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %q.%q (
            %q TEXT,
            "_raw_data" JSONB,
            "_source" VARCHAR(20) DEFAULT 'debezium',
            "_synced_at" TIMESTAMP DEFAULT NOW(),
            "_version" BIGINT DEFAULT 1,
            "_hash" VARCHAR(64),
            "_deleted" BOOLEAN DEFAULT FALSE,
            "_created_at" TIMESTAMP DEFAULT NOW(),
            "_updated_at" TIMESTAMP DEFAULT NOW()
        )`, schemaName, tableName, pkColumn),
    }
    for _, stmt := range ddl {
        if err := sa.db.Exec(stmt).Error; err != nil {
            return err
        }
    }
    return nil
}
```

PK type = TEXT (V1 conservative — V2 path đã có SchemaManager.createShadowTable typed).

## Task 110 — `036_prune_legacy_v1.sql`

```sql
-- 036_prune_legacy_v1.sql — Phase B3 (System Refactor 2026-05).
-- Idempotent prune of V1 legacy seed rows in source_object_registry +
-- shadow_binding (migration 035 inserts 10 rows legacy_1..legacy_10).
--
-- Eliminates first-write-wins routeCache collisions when V2 source name
-- == V1 source name. Re-runnable: only touches rows still active.

BEGIN;

WITH legacy_src AS (
    SELECT id FROM cdc_system.source_object_registry
     WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
)
UPDATE cdc_system.shadow_binding sb
   SET is_active = false,
       updated_at = NOW()
  FROM legacy_src ls
 WHERE sb.source_object_id = ls.id
   AND sb.is_active = true;

WITH legacy_src AS (
    SELECT id FROM cdc_system.source_object_registry
     WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
)
UPDATE cdc_system.master_binding mb
   SET is_active = false,
       updated_at = NOW()
  FROM legacy_src ls
 WHERE mb.source_object_id = ls.id
   AND mb.is_active = true;

UPDATE cdc_system.source_object_registry
   SET is_active = false,
       notes = COALESCE(notes || E'\n', '') ||
               '[pruned by 036_prune_legacy_v1.sql at ' || NOW()::text || ']',
       updated_at = NOW()
 WHERE object_code LIKE 'legacy\_%' ESCAPE '\'
   AND is_active = true;

SELECT
    (SELECT count(*) FROM cdc_system.source_object_registry
      WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active = false) AS pruned_sources,
    (SELECT count(*) FROM cdc_system.shadow_binding sb
      JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id
      WHERE sor.object_code LIKE 'legacy\_%' ESCAPE '\' AND sb.is_active = false) AS pruned_shadow_bindings;

COMMIT;
```

## Task 111 — Inventory Operator flow (separate file `03_implementation_b3_operator_flow.md`)

Will be written during Step 3.

## Task 112 — Smoke 3 engine

Test scripts inline in execution step. Each engine:
1. Insert source row (mongo/mariadb/pg)
2. Curl admin-api `/v1/source-objects/register` (or via cms-service `/api/v1/source-objects/register`)
3. Curl cms `/api/v1/schema-proposals` → approve
4. Curl cms `/api/v1/masters` POST → create master
5. Wait cron tick 60s
6. Query shadow + master row count
7. Document evidence
