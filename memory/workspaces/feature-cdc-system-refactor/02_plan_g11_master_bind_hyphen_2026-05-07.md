# Plan — G-11 Master-bind hyphen blocker

> **Author**: max-Brain | **Date**: 2026-05-07 ICT | **Loop iter#10**
> **Workspace**: `feature-cdc-system-refactor`
> **Source of truth**: real-evidence audit at `report_flow1_loop_iter9_2026-05-07.md` + queries chạy trong iter#10
> **Pre-req docs**: `01_requirements_flow1_e2e_2026-05-07.md`, `02_plan_flow1_e2e_2026-05-07.md`, `08_tasks_flow1_e2e_2026-05-07.md`
> **Boss directive**: "bằng mọi giá phải lên đc flow1 này"

---

## A. Tóm tắt blocker

**G-11 (NEW iter#9, root-caused iter#10)** — Mongo collection có hyphen (e.g. `refund-requests`) không qua được master DDL apply.

### A.1 — Evidence (real query)

```
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c
  "SELECT id, source_object_id, master_schema, master_table, schema_status, is_active
   FROM cdc_system.master_binding WHERE source_object_id = 44;"
```

```
 id | source_object_id |         master_schema         |  master_table   | schema_status | is_active
----+------------------+-------------------------------+-----------------+---------------+-----------
 31 |               44 | dw_mongo_payment_bill_default | refund-requests | approved      | t
```

`master_table='refund-requests'` → có hyphen.

### A.2 — Failure point

`cdc-system/centralized-data-service/internal/service/master_ddl_generator.go:47,62`:

```go
var ddlIdentRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
...
func (g *MasterDDLGenerator) Generate(ctx context.Context, masterName string) (*MasterDDLResult, error) {
    if !ddlIdentRe.MatchString(masterName) {
        return nil, fmt.Errorf("invalid master_name: %q", masterName)   // ← G-11 stops here
    }
```

Hyphen không khớp regex → DDL gen abort → master_pending → failed → state machine không advance → src 44 stuck `provisioning_state='failed'`.

### A.3 — Root-cause chain

```
[register API]            → INSERT source_object_registry(source_object_name='refund-requests')
[orchestrator step seedMasterBindingForAdvance @ line 409]
                          → masterTable := src.SourceObjectName    // RAW, no normalization
                          → INSERT master_binding(master_table='refund-requests')
                          → publish nats cdc.cmd.master.bind {master_table:'refund-requests'}
[worker handler master_ddl_handler.go:87]
                          → gen.Apply(ctx, 'refund-requests')
                          → generator regex reject → "invalid master_name"
[orchestrator]            → step master_bind = failed
                          → provisioning_state = 'failed'
```

Single-line root cause: `provisioning_orchestrator.go:409` truyền raw external identifier vào downstream PG DDL pipeline mà không normalize.

---

## B. 3 Phương án + so sánh

| Phương án | Vị trí sửa | Ưu | Nhược | Effort | Recommendation |
|-----------|-----------|----|-------|--------|----------------|
| **X — Normalize at orchestrator boundary** | `provisioning_orchestrator.go:409` | 1 chỗ sửa, nhất quán cho mọi engine, không đụng worker, không đụng API | Cần document `master_table ≠ source_object_name`; cần backfill 1 row hiện tại | ~30 phút | ✅ **Recommended** |
| Y — Relax DDL gen regex | `master_ddl_generator.go:47` + quoteDDLIdent calls | Giữ nguyên external naming | PG identifier với hyphen phải luôn quote → fragile; tooling khác (psql, dbt, BI) phải biết → tăng surface lỗi | ~1 giờ | ❌ Reject |
| Z — Reject at register API | `cdc-cms-service/internal/api/source_register_handler.go` | Fail-fast | User-hostile (Mongo collection name là external constraint, dev không rename được); chặn use-case hợp lệ | ~20 phút | ❌ Reject |

---

## C. Phương án X — Detail (Recommended)

### C.1 — Code change (Muscle scope, not Brain)

File: `cdc-system/centralized-data-service/internal/service/provisioning_orchestrator.go`

**Step 1**: Add helper (top of file, near package-level vars):

```go
// normalizePGIdent converts an external identifier (e.g. Mongo collection name with hyphen)
// to a PG-safe lowercase snake_case form. Mirrors shadow_binding shadow_table normalization.
var pgIdentReplacer = strings.NewReplacer("-", "_", ".", "_", " ", "_")

func normalizePGIdent(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    s = pgIdentReplacer.Replace(s)
    // strip remaining invalid chars; collapse repeats
    var b strings.Builder
    prevUnderscore := false
    for i, r := range s {
        valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
        // first char must be letter or underscore
        if i == 0 && (r >= '0' && r <= '9') {
            b.WriteByte('_')
            prevUnderscore = true
        }
        if !valid { continue }
        if r == '_' {
            if prevUnderscore { continue }
            prevUnderscore = true
        } else {
            prevUnderscore = false
        }
        b.WriteRune(r)
    }
    out := b.String()
    if len(out) > 63 {
        out = out[:63]
    }
    if out == "" {
        out = "_invalid"
    }
    return out
}
```

**Step 2**: Apply at line 409:

```go
// BEFORE
masterTable := src.SourceObjectName

// AFTER
masterTable := normalizePGIdent(src.SourceObjectName)
if masterTable != src.SourceObjectName {
    o.logger.Warn("provisioning: master_table normalized",
        zap.Int64("source_id", sourceID),
        zap.String("source_object_name", src.SourceObjectName),
        zap.String("master_table", masterTable))
}
```

**Step 3**: Mirror at shadow path nếu chưa có (kiểm tra `shadow_binding` insert site — `internal/admin/source_register.go` step 2). Nếu shadow_table đã normalize ở đó → để nguyên. Nếu chưa → áp dụng cùng helper.

### C.2 — Backfill SQL (one-shot, idempotent)

File: `centralized-data-service/deployments/sql/cdc/fix_g11_master_table_hyphen_2026-05-07.sql`

```sql
-- G-11 backfill: normalize hyphen → underscore in master_binding.master_table.
-- Idempotent: chỉ update row có hyphen.
BEGIN;

UPDATE cdc_system.master_binding
SET master_table = REPLACE(master_table, '-', '_'),
    physical_table_fqn = REPLACE(physical_table_fqn, '-', '_'),
    updated_at = NOW()
WHERE master_table LIKE '%-%';

-- Reset src 44 state để re-trigger master_bind step
UPDATE cdc_system.source_object_registry
SET provisioning_state = 'master_pending',
    last_step_error = NULL,
    updated_at = NOW()
WHERE id = 44 AND provisioning_state = 'failed';

COMMIT;

-- Verify
SELECT id, master_table FROM cdc_system.master_binding WHERE source_object_id = 44;
SELECT id, provisioning_state FROM cdc_system.source_object_registry WHERE id = 44;
```

### C.3 — Unit test

File: `centralized-data-service/internal/service/provisioning_orchestrator_test.go` (existing? new?)

```go
func TestNormalizePGIdent(t *testing.T) {
    cases := []struct{ in, out string }{
        {"refund-requests", "refund_requests"},
        {"Refund-Requests", "refund_requests"},
        {"order.items", "order_items"},
        {"_v2_orders", "_v2_orders"},
        {"123abc", "_123abc"},
        {"users", "users"},
        {"a---b", "a_b"},
        {"name with space", "name_with_space"},
        {"", "_invalid"},
    }
    for _, c := range cases {
        if got := normalizePGIdent(c.in); got != c.out {
            t.Errorf("normalizePGIdent(%q) = %q, want %q", c.in, got, c.out)
        }
    }
}
```

### C.4 — Smoke verify (post-fix, post-restart worker)

```bash
# 1. Apply backfill SQL
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw \
  < centralized-data-service/deployments/sql/cdc/fix_g11_master_table_hyphen_2026-05-07.sql

# 2. Restart worker (Boss-approved) — picks up new binary with normalize helper
# kill <worker-pid>; nohup ... &

# 3. Wait 30s for orchestrator tick (default poll interval)

# 4. Verify state advanced
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT id, source_object_name, provisioning_state, last_step_error
   FROM cdc_system.source_object_registry WHERE id = 44;"

# Expected: provisioning_state ∈ {master_active, mapping_pending, active}, last_step_error=NULL

# 5. Verify physical master table exists on Path A
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "\d dw_mongo_payment_bill_default.refund_requests"

# 6. Verify worker log — không còn 'invalid master_name'
docker logs gpay-cdc-worker --since 5m 2>&1 | grep -E 'master_ddl|invalid master_name'
# Expected: no 'invalid master_name' lines after backfill apply timestamp
```

### C.5 — Acceptance criteria

- [ ] AC-1: `normalizePGIdent` unit test PASS toàn bộ 9 case
- [ ] AC-2: Backfill SQL chạy idempotent (rerun = 0 rows updated)
- [ ] AC-3: Worker rebuild PASS (`go build ./...`)
- [ ] AC-4: Worker test PASS (`go test ./...`)
- [ ] AC-5: src 44 advance khỏi `failed` trong 60s sau khi restart worker + apply backfill
- [ ] AC-6: Master table physical `dw_mongo_payment_bill_default.refund_requests` (underscore, no hyphen) tồn tại trên Path A
- [ ] AC-7: Worker log không có `invalid master_name` trong 5 phút post-fix

---

## D. Risk & rollback

### D.1 — Risk

| Risk | Mitigation |
|------|-----------|
| Normalized `master_table` collide với existing row khác (UNIQUE constraint `(master_connection_id, master_schema, master_table)`) | Backfill query check: `SELECT count(*) FROM master_binding WHERE master_table LIKE '%-%' AND EXISTS (SELECT 1 FROM master_binding mb2 WHERE mb2.master_connection_id=master_binding.master_connection_id AND mb2.master_schema=master_binding.master_schema AND mb2.master_table=REPLACE(master_binding.master_table,'-','_') AND mb2.id != master_binding.id);` — nếu >0 → halt, manual resolve |
| Mapping rules trỏ tới `master_binding.id=31` (chưa bị invalidate) — đổi master_table có invalidate mapping không? | `mapping_rule_v2_master_binding_id_fkey ON DELETE CASCADE` — không CASCADE on UPDATE; mapping rules giữ FK ổn |
| User-facing API trả `master_table` với hyphen sang FE/BI tools | Document trong `04_decisions_*` rằng `master_table` là PG-safe; FE hiển thị `source_object_name` (giữ hyphen) ra UI |
| Backfill chạy trên prod khi đang có request ghi master_binding | Backfill là single transaction; lock bằng PG row lock (UPDATE WHERE) — request cùng row sẽ wait/serialize |

### D.2 — Rollback (nếu fail)

```sql
-- Revert backfill (chỉ trong cùng phiên — không có audit của giá trị cũ)
-- Nếu cần khôi phục, dùng pg_dump trước backfill.
-- Rollback code: revert orchestrator.go:409 về `masterTable := src.SourceObjectName`,
-- rebuild worker.
```

Boss approve trước khi apply prod.

---

## E. Out of scope (defer)

- ddl_status=pending vs provisioning_state=active drift (audit trong iter#10): không phải G-11. Sources 35/37/42 đã active — không liên quan src 44 master_bind. Plan riêng cho dedup/sync ddl_status sau khi G-11 unblocked.
- Phương án Y/Z: reject. Lý do trong bảng B.
- Multi-row shadow_binding cho src 44 (id 52 + 53): cleanup riêng — sau khi G-11 fix, run dedup query loại row có hyphen.

---

## F. Workflow gate

1. ✅ max-Brain output `02_plan_g11_*` (this file) + APPEND `coordination_max_x2_*` + `05_progress`
2. ⏳ **Boss approve plan** — đặc biệt §C.2 backfill SQL chạy prod
3. ⏳ **Muscle (CC CLI / x2 nếu Boss assign worker-lane override)** thi công:
   - C.1 code change (provisioning_orchestrator.go + helper)
   - C.2 backfill SQL file
   - C.3 unit test
4. ⏳ **Build + test PASS** trước commit
5. ⏳ **Boss approve restart worker**
6. ⏳ Apply backfill + restart worker + verify §C.4
7. ⏳ APPEND `05_progress.md` với raw output 7 AC
8. ⏳ Mark G-11 CLOSED trong `coordination_max_x2_*`

---

## G. Files

- This plan: `02_plan_g11_master_bind_hyphen_2026-05-07.md`
- Pending Muscle artifacts:
  - `centralized-data-service/internal/service/provisioning_orchestrator.go` (modify lines ~408–410, add helper top of file)
  - `centralized-data-service/internal/service/provisioning_orchestrator_test.go` (add TestNormalizePGIdent)
  - `centralized-data-service/deployments/sql/cdc/fix_g11_master_table_hyphen_2026-05-07.sql` (new)

— max-Brain, iter#10
