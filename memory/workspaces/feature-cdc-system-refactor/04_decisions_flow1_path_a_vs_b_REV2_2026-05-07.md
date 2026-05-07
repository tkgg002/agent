# 04 — Decisions: Flow 1 Path A vs Path B (REV2 — supersede iter#3)

> **Author**: max-Brain | **Date**: 2026-05-07 ICT loop iter#5
> **Trigger**: x2 double-verification iter#3+iter#4 phát hiện iter#3 doc §1.5 SAI fact + A1 cleanup destructive
> **Audience**: Boss approve gate + x2 cms-lane
> **Predecessor**: `04_decisions_flow1_path_a_vs_b_2026-05-07.md` (iter#3) — **REVOKE**
> **Cross-ref**: `09_tasks_solution_flow1_x2_*.md §7§8` (x2 evidence collection)

---

## §0 Summary

REV2 supersede iter#3 doc. Recommendation đảo chiều:

| Iter#3 (REVOKED) | REV2 iter#5 (CURRENT) |
|---|---|
| **A4** status quo Path A only | ❌ REJECT (Path A = orphan) |
| **A1** drop `gpay-postgres-shadow` cleanup | ❌ REJECT (sẽ destroy 1720 rows Boss output Flow 1) |
| **A2** adopt Path B (deprecate Path A) | ⚠️ Candidate (cần re-route control plane reads) |
| **A3** hybrid dual-cluster (Path A control + Path B data) | ✅ **RECOMMENDED** — match worker runtime intent |

→ Boss approve gate: **HOLD iter#3 A1 escalation**. New escalation = **A3 hybrid + cms `ShadowAutomator` align**.

---

## §1 Iter#3 doc errata (where iter#3 went wrong)

| Iter#3 claim | Reality (verified iter#4+iter#5) | Source |
|---|---|---|
| §1.5: "Path B 5436 cdc_shadow KHÔNG phải production data path, là test artifact" | Path B = production shadow data target, _synced_at matches Flow 1 iter#0 timestamp | `09_tasks_solution §7.3` + max iter#4 re-verify |
| §1.3: "Cả worker + cms đều configured shadow=5433 cdc_dw" | Worker `.env:7` deliberately overrides → `CDC_SHADOW_DB_URL=...gpay-postgres-shadow:5432/cdc_shadow`. Worker docker-compose default cdc_dw NOT loaded | `09_tasks_solution §8.1` |
| §1.4: "Không có process active nào đang write xuống 5436 trong thời điểm iter#3" | Worker `netstat`: 1 active TCP conn 172.26.0.18:5432 (gpay-postgres-shadow). KHÔNG dead | `09_tasks_solution §7.2` |
| §2 Recommendation A4 + A1 cleanup | Sẽ destroy 1720 rows Boss output Flow 1 + break worker connection pool | §1.5 reasoning vô hiệu hóa |

→ **Root cause iter#3 sai**: Decision doc chỉ trust static `*.yml` comment + grep code config, KHÔNG verify `docker inspect ENV` runtime + `netstat/lsof active conn` + actual data row count.

---

## §2 Consolidated evidence (REV2 authoritative)

### §2.1 2 PG instance độc lập (unchanged from iter#3)

| Cluster | Container | Port | Database | Role REV2 |
|---|---|---|---|---|
| **A** | gpay-postgres-cdc | 5433 | cdc_dw | **Control plane** (registry + bindings + admin metadata) |
| **B** | gpay-postgres-shadow | 5436 | cdc_shadow | **Shadow data plane** (worker writes shadow_<src>.* tables) |

### §2.2 Worker runtime ENV (3 separate DSN)

```
docker inspect gpay-cdc-worker --format '{{range .Config.Env}}{{println .}}{{end}}'
CDC_SYSTEM_DB_URL=postgres://...postgres-cdc:5432/cdc_dw           ← Path A control
CDC_CONTROL_PLANE_URL=postgres://...postgres-cdc:5432/cdc_dw       ← Path A control
CDC_SHADOW_DB_URL=postgres://...gpay-postgres-shadow:5432/cdc_shadow  ← Path B data
```

### §2.3 Worker code architectural support

- `internal/service/connection_manager.go:33-89` — pattern multi-shadow `RoleShadow`, `getNamedDB`, fallback nếu `CDC_SHADOW_DB_URL` unset.
- `internal/handler/event_handler.go:178-179` — `h.connMgr.GetShadowDB(...)` separate pool cho shadow writes.

→ Architectural intent: Path A control + Path B data. KHÔNG phải hack.

### §2.4 Worker `.env` deliberate

- `centralized-data-service/.env:7`: `CDC_SHADOW_DB_URL=...gpay-postgres-shadow:5432/cdc_shadow` (active).
- `.env.example:19,22`: documents BOTH options (cdc_dw default + cdc_shadow alternative). Operator chọn Path B.

### §2.5 Path B data fingerprint (production)

```sql
-- Path B 5436 cdc_shadow
SELECT count(*), min(_synced_at), max(_synced_at) FROM shadow_payment_bill_service.refund_requests;
1720 | 2026-05-07 03:23:44.527350 | 2026-05-07 03:23:45.031237
```

→ Match cửa sổ iter#0 Flow 1 run (8s Debezium snapshot ingest). KHÔNG phải orphan.

### §2.6 Path A orphan (cms config drift)

```sql
-- Path A 5433 cdc_dw
SELECT count(*) FROM shadow_payment_bill_service.refund_requests;
0
```

CMS code grep `CDC_SHADOW_DB_URL\|ShadowDB\|cdc_shadow` → **0 hits**. CMS missing shadow DSN block. ShadowAutomator dùng global gorm session = control plane Path A.

→ CMS tạo physical table tại Path A (sai), worker ghi data tại Path B (đúng) → orphan.

---

## §3 Decision options REVISED

### A3 — Hybrid dual-cluster (RECOMMENDED REV2)

**Architecture**: Path A control plane + Path B shadow data, đúng worker runtime intent.

**Implementation steps**:
1. **CMS config** add `shadowDb:` block trỏ Path B 5436 cdc_shadow (và optionally `shadowDb.url` env override).
2. **CMS code** `ShadowAutomator` (cms-lane) inject `*gorm.DB` riêng (parameter, không global) thay vì dùng default control-plane session.
3. **CMS server boot** init 2 gorm session: `controlPlaneDB` (Path A) + `shadowDB` (Path B). Inject vào ShadowAutomator constructor.
4. **Migration**: 0-row orphan tables tại Path A → DROP an toàn. 1720-row Path B keep nguyên.
5. **Verify smoke**: re-Register source qua Phương án Z → cms ShadowAutomator tạo table tại Path B → worker Kafka consumer ingest tại Path B (đã có) → AC-5/6/7/8 PASS.

**Effort**: 4-6h x2 (cms-lane).
**Risk**: Medium (touch cms hexagonal infra/persistence + server boot wiring).
**Pre-req**: Boss approve A3 + spec lock cho cms config schema split.
**Lane**: x2 (cms-lane).

### A2 — Adopt Path B (deprecate Path A control plane shift)

Move ALL cms reads (registry, bindings, admin) sang Path B 5436 cdc_shadow. Drop Path A.

**Effort**: 6-8h (touch worker `CDC_CONTROL_PLANE_URL` + cms config + migration registry/bindings/master_binding tables A→B).
**Risk**: High (control plane migration is invasive).
**Verdict**: ❌ Reject — không cần invasive vì A3 đã đủ giải quyết drift.

### A1 — Drop Path B (REVOKED iter#3)

❌ REVOKED. Sẽ destroy 1720 rows Boss output Flow 1 + break worker `CDC_SHADOW_DB_URL` pool.

### A4 — Status quo Path A only (REVOKED iter#3)

❌ REVOKED. Path A = orphan, không phải single source of truth.

---

## §4 Recommendation matrix REV2

| Criterion | A1 (REVOKED) | A2 | A3 (REC) | A4 (REVOKED) |
|---|---|---|---|---|
| Effort | 30 min | 6-8h | 4-6h | 0 |
| Risk | Low | High | Medium | Low |
| Match worker runtime intent | ✗ break | partial | ✅ | ✗ ignore |
| Match `.env` deliberate setup | ✗ | partial | ✅ | ✗ |
| Preserve 1720 rows Boss output | ✗ destroy | keep | ✅ keep | keep (orphan irrelevant) |
| Production-ready post-fix | ✗ | needs migration | ✅ | ✗ orphan |
| Block Phương án Y | No | Yes | No | No |

→ **REV2 recommendation**: **A3 hybrid**.

---

## §5 A3 implementation plan (high-level, x2-lane execution)

### §5.1 CMS config (read-only by max-Brain — x2 thi công)

```yaml
# cdc-cms-service/config/config-local.yml
masterDb:
  port: 5433
  database: cdc_dw
  url: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable

# NEW: separate shadow data plane
shadowDb:
  default: postgres://gpay_admin:gpay_pass@localhost:5436/cdc_shadow?sslmode=disable
  # optional: env override `CDC_SHADOW_DB_URL`
```

### §5.2 CMS server boot (x2-lane code)

```go
// cdc-cms-service/internal/server/server.go (or wherever DB init lives)
controlPlaneDB := gormOpen(cfg.MasterDb.URL)  // Path A 5433 cdc_dw — control plane
shadowDB       := gormOpen(cfg.ShadowDb.URL)  // Path B 5436 cdc_shadow — data plane

// Wire ShadowAutomator with explicit shadowDB (NOT controlPlaneDB)
shadowAutomator := persistence.NewShadowAutomator(shadowDB, ...)
```

### §5.3 CMS ShadowAutomator (x2-lane code)

Constructor injection thay vì grab global session. Existing constructor signature có thể đã accept `*gorm.DB` — confirm trong x2 investigation.

### §5.4 Migration (DBA-lane, ops-only)

```sql
-- Path A 5433 cdc_dw orphan cleanup (0-row tables only)
DROP SCHEMA shadow_payment_bill_service CASCADE;
-- ... grep all shadow_<src>_<table> với row count=0 → DROP
-- 1720-row Path B tables keep nguyên
```

### §5.5 Smoke verify post-deployment

1. Boss approve G-7 worker `PROVISIONING_ORCHESTRATOR_ENABLED=1` + restart.
2. x2.D rebuild + restart cms binary để pickup `adc6faf` G-10 + new shadowDb wiring.
3. Phương án Z: `POST /api/v1/source-objects/register` + `POST /api/v1/cms/sources/:id/provisioning/advance` → verify state machine advance to `shadow_active`.
4. Verify shadow table tạo tại Path B 5436 (không phải Path A 5433).
5. Verify Path B refund_requests row count grow.

---

## §6 Boss decision matrix REV2

| Item | Iter#3 verdict | REV2 verdict |
|---|---|---|
| **G-7** worker enable PROVISIONING_ORCHESTRATOR_ENABLED + restart | P0 highest leverage | ✅ unchanged — **approve sớm nhất** |
| **G-8** Path A vs B architecture | A4 + A1 cleanup | ❌ REVOKED. ✅ **A3 hybrid** (cms config + ShadowAutomator inject `*gorm.DB` riêng) |
| **A1** drop gpay-postgres-shadow cleanup | "30 min low risk" | ⛔ **REVOKE escalation** — destructive 1720 rows |
| **Phương án Y** refactor admin endpoint | P2 unchanged | unchanged |
| **Backfill 4 phantom rows** | P2 unchanged | unchanged |

---

## §7 Open questions cho Boss

1. **Q-1**: Approve A3 hybrid? (Effort 4-6h x2-lane).
2. **Q-2**: Approve x2.G investigation result là sufficient input cho A3 plan? (x2 đã collect §8 evidence).
3. **Q-3**: Approve order: G-7 trước hay A3 trước? (Recommend G-7 trước vì worker advance không cần A3 depend).
4. **Q-4**: Approve migration drop orphan Path A tables (0-row only)?

---

## §8 Lesson candidate (chờ Boss confirm REV2)

**L-DECISION-DOC-FACT-CHECK-DRIFT** — Brain decision doc PHẢI cite runtime evidence:
- `docker inspect gpay-cdc-worker --format ENV` (env var actual loaded)
- `docker exec ... netstat -tn` (active TCP connection)
- `psql ... -c "SELECT count(*), min/max timestamp..."` (actual data)

KHÔNG chỉ trust static `*.yml` comment hoặc grep code default — runtime override có thể flip toàn bộ data path.

→ Muscle có quyền pause Brain decision khi runtime ≠ doc. Brain phải REVOKE recommendation cũ + ship REV2 doc, không bảo vệ doc cũ.

— max-Brain (REV2 iter#5)
