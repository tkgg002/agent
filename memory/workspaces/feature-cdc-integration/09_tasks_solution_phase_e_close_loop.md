# 09 — Tasks Solution (diff hints + commands) Phase E

> Brain prohibition §12 — file này là **đề xuất diff cho Muscle**, KHÔNG phải code change của Brain.

---

## E1 — `internal/admin/helpers.go` diff hint

### New helper: `extendDatabaseList`

```go
// extendDatabaseList appends `value` to `config[key]` if not already present.
// Returns updated value + wasAdded flag (true nếu thực sự append).
// Idempotent: gọi 2 lần với cùng value → lần 2 wasAdded=false.
func extendDatabaseList(config map[string]string, key, value string) (string, bool) {
    current := strings.TrimSpace(config[key])
    if current == "" {
        config[key] = value
        return value, true
    }
    parts := strings.Split(current, ",")
    seen := make(map[string]struct{}, len(parts))
    for _, p := range parts {
        seen[strings.TrimSpace(p)] = struct{}{}
    }
    if _, ok := seen[value]; ok {
        return current, false
    }
    next := current + "," + value
    config[key] = next
    return next, true
}
```

### Refactor `extendDebeziumInclude` (signature change — return wasAdded)

```go
// Result of extending an include list. WasAdded* indicates whether each tier
// actually changed (used by handler to emit warnings to operator).
type ExtendResult struct {
    DatabaseTierAdded   bool
    CollectionTierAdded bool
    UpdatedConfig       map[string]string
}

func extendDebeziumInclude(
    config map[string]string,
    sourceType, databaseName, namespaceName string,
) (*ExtendResult, error) {
    res := &ExtendResult{UpdatedConfig: config}
    switch sourceType {
    case "mongodb":
        _, res.DatabaseTierAdded = extendDatabaseList(config, "database.include.list", databaseName)
        _, res.CollectionTierAdded = extendDatabaseList(config, "collection.include.list", databaseName+"."+namespaceName)
    case "mysql", "mariadb":
        _, res.DatabaseTierAdded = extendDatabaseList(config, "database.include.list", databaseName)
        _, res.CollectionTierAdded = extendDatabaseList(config, "table.include.list", databaseName+"."+namespaceName)
    case "postgres":
        // PG single-database constraint — fail-fast nếu mismatch (per L-cascade-liability)
        if cur := strings.TrimSpace(config["database.dbname"]); cur != "" && cur != databaseName {
            return nil, fmt.Errorf("pg connector locked to db=%s, requested db=%s", cur, databaseName)
        }
        // namespaceName cho PG là "schema.table"
        _, res.CollectionTierAdded = extendDatabaseList(config, "table.include.list", namespaceName)
    default:
        return nil, fmt.Errorf("unsupported source_type: %s", sourceType)
    }
    return res, nil
}
```

---

## E1 — `internal/admin/types.go` diff hint

```go
type RegisterSourceResponse struct {
    Status   string   `json:"status"`
    Steps    []string `json:"steps"`
    Errors   []string `json:"errors,omitempty"`
    Warnings []string `json:"warnings,omitempty"`  // NEW
}
```

---

## E1 — `internal/admin/source_register.go` diff hint

Trong handler, sau khi gọi `extendDebeziumInclude`:

```go
result, err := extendDebeziumInclude(connectorConfig, req.SourceType, req.DatabaseName, namespace)
if err != nil {
    // step 2 fail → 207 Multi-Status path
    ...
}
if result.DatabaseTierAdded {
    resp.Warnings = append(resp.Warnings, fmt.Sprintf(
        "database '%s' was just added to debezium include — first event from new namespace may be delayed; connector task may need a moment to snapshot",
        req.DatabaseName))
}
```

---

## E1 — `internal/admin/server_test.go` diff hint

```go
func TestExtendDatabaseList_NewValue(t *testing.T) {
    cfg := map[string]string{"database.include.list": "a,b"}
    val, added := extendDatabaseList(cfg, "database.include.list", "c")
    assert.Equal(t, "a,b,c", val)
    assert.True(t, added)
}

func TestExtendDatabaseList_AlreadyPresent(t *testing.T) {
    cfg := map[string]string{"database.include.list": "a,b"}
    val, added := extendDatabaseList(cfg, "database.include.list", "b")
    assert.Equal(t, "a,b", val)
    assert.False(t, added)
}

func TestExtendDatabaseList_EmptyConfig(t *testing.T) {
    cfg := map[string]string{}
    val, added := extendDatabaseList(cfg, "database.include.list", "a")
    assert.Equal(t, "a", val)
    assert.True(t, added)
}

func TestExtendDebeziumInclude_Mongo_BothTiers(t *testing.T) {
    cfg := map[string]string{
        "database.include.list":   "service-a",
        "collection.include.list": "service-a.col1",
    }
    res, err := extendDebeziumInclude(cfg, "mongodb", "service-b", "col2")
    assert.NoError(t, err)
    assert.True(t, res.DatabaseTierAdded)
    assert.True(t, res.CollectionTierAdded)
    assert.Equal(t, "service-a,service-b", cfg["database.include.list"])
    assert.Equal(t, "service-a.col1,service-b.col2", cfg["collection.include.list"])
}

func TestExtendDebeziumInclude_Mongo_DBExistsCollNew(t *testing.T) {
    cfg := map[string]string{
        "database.include.list":   "service-a",
        "collection.include.list": "service-a.col1",
    }
    res, err := extendDebeziumInclude(cfg, "mongodb", "service-a", "col2")
    assert.NoError(t, err)
    assert.False(t, res.DatabaseTierAdded)
    assert.True(t, res.CollectionTierAdded)
}

func TestExtendDebeziumInclude_PG_DBLockMismatch(t *testing.T) {
    cfg := map[string]string{"database.dbname": "goopay_source"}
    _, err := extendDebeziumInclude(cfg, "postgres", "other_db", "public.tbl")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "pg connector locked to db=goopay_source")
}
```

---

## E3 — Prune V1 legacy commands

```bash
# Lần 1
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  < deployments/sql/cdc/prune_legacy_v1_bindings.sql 2>&1 | tee /tmp/prune_run1.log

# Lần 2 (idempotency check)
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  < deployments/sql/cdc/prune_legacy_v1_bindings.sql 2>&1 | tee /tmp/prune_run2.log

# Verify
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT count(*) FROM cdc_system.source_object_registry
    WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active=true;"
# Expect: 0
```

---

## E4 — G4 audit commands

```bash
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw <<'SQL' 2>&1 | tee /tmp/g4_audit.log
\echo === 1. Binding state ===
SELECT mb.id, mb.binding_code, mb.is_active, mb.write_mode, sor.object_code, sor.is_active AS src_active, sor.provisioning_state
  FROM cdc_system.master_binding mb
  JOIN cdc_system.source_object_registry sor ON sor.id = mb.source_object_id
 WHERE mb.binding_code ILIKE '%orders_addtest%' OR sor.object_code ILIKE '%orders_addtest%';

\echo === 2. Schedule state ===
SELECT ts.id, ts.master_binding_id, ts.mode, ts.is_enabled, ts.last_status, ts.last_run_at, ts.last_error
  FROM cdc_system.transmute_schedule ts
  JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
 WHERE mb.binding_code ILIKE '%orders_addtest%';

\echo === 3. Master DDL exists ===
\dt dw_src_local_pg_source.*

\echo === 4. Shadow row count ===
SELECT count(*) FROM shadow_src_local_pg_source.orders_addtest;

\echo === 5. Recent activity log ===
SELECT timestamp, action_type, payload->>'binding_code' AS binding, payload->>'status' AS status
  FROM cdc_system.cdc_activity_log
 WHERE payload::text ILIKE '%orders_addtest%'
 ORDER BY timestamp DESC LIMIT 20;
SQL
```

Sau đó write `report_g4_diag_<ts>.md` với:
- 5 query output snapshot
- Root cause classification:
  - Class A: schedule `is_enabled=false` → recommend `UPDATE … SET is_enabled=true`
  - Class B: binding `is_active=false` → recommend re-trigger provisioning
  - Class C: master DDL chưa tạo → cascade chưa hoàn thành (cần re-trigger ở step master_bind)
  - Class D: schedule `last_status='failed'` lặp lại → debug `last_error`
  - Class E: schedule `last_run_at` advance bình thường nhưng master = 0 → bug transmuter (dependency với D1 schema schism / B6 hardcode `_gpay_id`)
- Recommendation: chỉ ra step cần làm tiếp (KHÔNG fix trong phase E).

---

## E5 — `/security-agent` invocation

```
Brain invokes Skill tool with skill="security-agent" and args targeting:
  - cmd/admin-api/main.go
  - internal/admin/server.go
  - internal/admin/source_register.go
  - internal/admin/helpers.go
  - internal/admin/types.go

Expected output:
  - file report_security_agent_admin_api_<ts>.md
  - Severity-tagged findings (HIGH/MED/LOW)
  - Threat model coverage: auth, rate-limit, replay, audit, error disclosure, secrets handling, race condition
```

---

## Rollback procedures

| Step | Rollback |
|---|---|
| E3 prune | `UPDATE cdc_system.source_object_registry SET is_active=true WHERE object_code LIKE 'legacy\_%' AND notes ILIKE '%pruned by deployments%';` (chỉ chạy nếu cần khôi phục) |
| E4 audit | N/A (read-only) |
| E1 code | `git revert <commit>` + restart admin-api |
| E1 connector config | PUT `/tmp/before.json` qua Kafka Connect REST |
| E2 smoke | DELETE source doc + admin-api UPDATE registry `is_active=false` |
| E5 sec gate | N/A (audit only) |
