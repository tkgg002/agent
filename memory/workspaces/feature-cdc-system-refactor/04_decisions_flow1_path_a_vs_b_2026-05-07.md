# 04 — Decisions: Flow 1 Path A vs Path B architectural drift (G-8)

> **Author**: max-Brain | **Date**: 2026-05-07 ICT loop iter#3
> **Trigger**: Boss directive "bằng mọi giá phải lên đc flow1" + iter#1 verify discovered Path A 0 rows vs Path B 1720 rows
> **Audience**: Boss approve gate + x2 cms-lane awareness
> **Predecessor**: `04_decisions_flow1_root_cause_2026-05-07.md` (Phương án Y), `report_flow1_loop_iter1_2026-05-07.md`

---

## 1. Evidence (verified queries iter#1 + iter#3)

### 1.1 2 PG instance độc lập

| Cluster | Container | Port | Database | Owner | Role intended |
|---|---|---|---|---|---|
| **A** | gpay-postgres-cdc | 5433 | cdc_dw | gpay_admin | "control plane + shadow co-located" (per worker `config-local.yml` comment) |
| **B** | gpay-postgres-shadow | 5436 | cdc_shadow | gpay_admin | (mục đích chưa rõ trong code/config) |

### 1.2 Schema drift (cùng table tên, structure khác)

```sql
-- Cluster A 5433 cdc_dw
\d shadow_payment_bill_service.refund_requests
→ 10 cols: id, source_id, _raw_data, _source, _synced_at, _version, _hash,
   _deleted, _created_at, _updated_at + UNIQUE(source_id) + 3 indexes
   row count = 0

-- Cluster B 5436 cdc_shadow
\d shadow_payment_bill_service.refund_requests
→ 9 cols (1 ít hơn — không xác minh được col nào missing trong iter này)
   row count = 1720 (= source Mongo `payment-bill-service.refund-requests` count)
```

### 1.3 Config evidence

**Worker `config-local.yml`** (lines 9-36):
```yaml
# control-plane connection (registry reads + shadow writes), so it
# points at gpay-postgres-cdc / cdc_dw.
masterDb:
  port: 5433
  database: cdc_dw
  url: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable

# Logical role: shadow_<src>.* — co-located with control plane in cdc_dw.
shadowDb:
  default: postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw?sslmode=disable
```

**Worker `docker-compose.yml:67`**:
```
- CDC_SHADOW_DB_URL=${CDC_SHADOW_DB_URL:-postgres://gpay_admin:gpay_pass@postgres-cdc:5432/cdc_dw?sslmode=disable}
```

**Cms `config-local.yml`** (lines 8-11):
```yaml
port: 5433
database: cdc_dw
```

→ **Cả worker + cms đều configured shadow=5433 cdc_dw**. Path B (5436 cdc_shadow) KHÔNG có configuration source-of-truth nào trỏ tới.

### 1.4 Process/connector evidence iter#3

- Kafka Connect 3 connector: `cdc-pg-source` + `cdc-mariadb-source` + `goopay-mongodb-cdc` — đều là Debezium SOURCE, KHÔNG có sink connector (PG sink) đăng ký.
- cms PID 64511 lsof: chỉ connect 5433 (pyrrho), KHÔNG tới 5436.
- worker PID 23565 docker-compose env trỏ 5433.
- `cdc-admin-api-f3v2` PID 21133 ADMIN_DB_URL trỏ 5433.

→ **Không có process active nào đang write xuống 5436 trong thời điểm iter#3**.

### 1.5 Implication: Path B 1720 rows = orphan data từ session test trước

Khả năng cao:
- Một session worker hoặc binary trước đó (có thể `/tmp/cdc-cms-service-flow1` build giai đoạn x2 test) chạy với env override `CDC_SHADOW_DB_URL=postgres://...:5436/cdc_shadow` để test → ghi 1720 rows → exit.
- HOẶC x2 dùng manual `psql + COPY` từ Kafka topic dump xuống 5436 để verify functional output.
- `gpay-postgres-shadow` container alive 47h nhưng không có active writer → snapshot stable.

→ **Path B KHÔNG phải production data path**, là test artifact.

## 2. Decision options

### Phương án A1 — Deprecate Path B (recommended)

- Boss confirm 5436 cdc_shadow là test cluster, không cần consolidate.
- Drop container `gpay-postgres-shadow` (hoặc giữ standalone test rig).
- Path A 5433 cdc_dw = single source of truth (theo intent config + comment).
- AC-8 verify: shadow row count > 0 sẽ phải dùng 5433 (sau khi G-7 worker enable + state→shadow_active + Debezium consumer ingest).

**Effort**: 30 min (Boss confirm + cleanup).
**Risk**: Low (drop test cluster, không ảnh hưởng production).
**Pre-req**: Boss approve drop hoặc giữ làm test rig (không Merge với production path).

### Phương án A2 — Adopt Path B (deprecate Path A)

- Boss decide 5436 cdc_shadow là production target (per "physical separation control plane vs shadow data" architecture pattern).
- Update worker `config-local.yml` + `docker-compose.yml` shadowDb URL từ 5433 → 5436.
- Update cms `config-local.yml` shadow connection target.
- Migrate existing shadow tables từ 5433 → 5436 (data backfill).
- Update `04_decisions_flow1_root_cause` Phương án Y reflect new target.

**Effort**: 4h (config change + migration script + smoke).
**Risk**: Medium (config drift cần align worker + cms + tests; có thể impact existing master_swap path tới 5434 dest).
**Pre-req**: Boss approve architectural shift + budget cho migration.

### Phương án A3 — Hybrid 2-cluster (control plane vs shadow data)

- Boss confirm pattern: 5433 cdc_dw = control plane (registry + bindings + admin metadata only); 5436 cdc_shadow = shadow data tables.
- Update cms `ShadowAutomator` connection từ control plane GORM → separate `shadowConnPool` trỏ 5436.
- Update worker `shadowDb.default` từ 5433 → 5436.
- Split config: `controlPlane.dsn` (5433) vs `shadowDb.dsn` (5436).
- Migrate existing shadow tables 5433 → 5436 (or accept Path A is empty + fresh start tại Path B).

**Effort**: 6h (config schema split + cms ShadowAutomator refactor + worker config refactor + migration + smoke).
**Risk**: Medium-High (touch cms hexagonal layer infra/persistence + worker connection pool init + tests).
**Pre-req**: Boss approve dual-cluster pattern + Phase 2 spec freeze (block other refactor).

### Phương án A4 — Status quo (Path A only, ignore Path B)

- Document Path B 5436 là legacy artifact, không sửa.
- Tất cả future work target Path A 5433 cdc_dw.
- Path B container có thể giữ standalone hoặc drop tùy Boss.
- Phương án Y + G-7 thi công như đã plan trong `04_decisions_flow1_root_cause`.

**Effort**: 0 (no change).
**Risk**: Low (chỉ là confirmation).
**Pre-req**: Boss confirm.

## 3. Recommendation matrix

| Criterion | A1 | A2 | A3 | A4 |
|---|---|---|---|---|
| Effort | 30 min | 4h | 6h | 0 |
| Risk | Low | Medium | Medium-High | Low |
| Match config intent | ✓ | ✗ | partial | ✓ |
| Match comment "co-located cdc_dw" | ✓ | ✗ | ✗ | ✓ |
| Production-ready | ✓ | needs migration | needs refactor | ✓ |
| Block Phase 2 Y | No | Yes | Yes | No |

→ **max recommendation**: **A4 (status quo)** kết hợp **A1 cleanup**. Lý do:
1. Config + comment intent rõ ràng: shadow tại 5433 cdc_dw.
2. Path B 1720 rows = test artifact, không phải architectural decision đã chốt.
3. Phương án Y (Phase 2 fix admin endpoint) target 5433, A4 không block.
4. Nếu Boss muốn dual-cluster sau (A3), có thể migrate sau khi production stable.

## 4. Open questions cho Boss

1. **Q-1**: Path B 5436 cdc_shadow tồn tại có chủ đích (architectural plan dual-cluster) HOẶC test artifact?
2. **Q-2**: Approve A4 + A1 cleanup (drop gpay-postgres-shadow container)?
3. **Q-3**: Nếu A3 (dual-cluster), Boss approve effort 6h Phase 3 refactor sau khi Phase 2 Y land?

## 5. Impact lên existing plans

### Phương án Y (Phase 2 từ `04_decisions_flow1_root_cause`)

- Không đổi nếu chọn A4. Refactor `admin/source_register.go:92` Step 5 → `orchestrator.Advance()` target 5433 cdc_dw.
- Đổi nếu chọn A2/A3: phải update connection target trong worker init + `ShadowAutomator` GORM session.

### G-7 (worker enable PROVISIONING_ORCHESTRATOR_ENABLED)

- Không đổi (env var change, không touch DB target).
- Sau khi enable, worker fire `cdc.cmd.shadow.bind` handler sẽ ghi shadow tables tại 5433 (current config).

### G-9 (worker auto-fire kafka.refresh-topics)

- Không đổi (NATS handler logic, không touch DB target).

### G-10 (cms normalize pk_type)

- ✅ x2 đã thi công working tree iter#3 (`register_registry.go:75 entry.PrimaryKeyType = normalizePKType(entry.PrimaryKeyType)` + new func + unit test).
- Không đổi target.

## 6. Decision gate

**max approve**: A4 + A1 cleanup khi Boss approve.
**Boss approve needed for**:
- A1: Drop `gpay-postgres-shadow` container (hoặc giữ standalone test).
- A2/A3: Approve effort + spec lock cho dual-cluster pattern.

— max-Brain (loop iter#3 Brain commitment delivered)
