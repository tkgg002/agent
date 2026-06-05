# 02_plan.md — Execution Plan

> Tham chiếu `01_requirements.md` (FR/NFR/AC) + `04_decisions.md` (ADR 01-08).

---

## Tổng quan

- **Mục tiêu**: Fix G1 root + G2 amplifier + G6 observability. 3-layer defense (CMS publish always, dispatch pre-publish, worker post-reload check).
- **Tổng effort**: 3-5 giờ (1 dev senior).
- **Risk**: LOW (patch tối thiểu, idempotent, backwards-compat).
- **Rollback**: Revert 4 commit nhỏ (CMS approve, CMS dispatch, worker preflight, NATS payload schema).

---

## Phase plan

| Phase | Mục tiêu | Effort | Gate |
|---|---|---|---|
| **P0** | Reproduce race local (testcontainers) | 0.5h | Test FAIL như expect |
| **P1** | Fix G1 — CMS approve always publish (ADR-01 + ADR-04) | 0.7h | Grep gate, unit test |
| **P2** | Fix G2 — Worker post-reload count check (ADR-02) | 0.5h | Drift scenario log |
| **P3** | Defense — CMS dispatch publish reload trước (ADR-03) | 0.3h | NATS msg order |
| **P4** | Observability — 2 metric + log line (ADR-05) | 0.5h | `/metrics` show |
| **P5** | CI grep gate (ADR-07) + integration test | 0.5h | CI green |
| **P6** | Deploy + verify prod | 0.5h | 0 lỗi mapping miss 24h |
| **P7** | Lesson append (ADR-06) + workspace status update | 0.2h | Lesson visible global |

**Total: ~3.7h** (không tính 24h prod monitoring).

---

## Dependency graph

```
P0 (reproduce) ─┐
                ├─→ P1 (CMS publish always) ─┐
                │                              │
                ├─→ P2 (worker preflight)   ─┼─→ P5 (CI + integration test) ─→ P6 (deploy) ─→ P7 (lesson)
                │                              │
                └─→ P3 (CMS dispatch)       ─┤
                                              │
P4 (observability) ──────────────────────────┘
```

P1, P2, P3, P4 có thể parallel sau P0 (file độc lập).

---

## Phase chi tiết

### P0 — Reproduce race local
**Goal**: Test deterministic reproduce race trước khi sửa.

**Steps**:
1. Spawn testcontainers: Postgres (`cdc_system` schema) + NATS + Mongo (mock source).
2. Seed: source_object_id=66, shadow_binding active, 0 mapping_rule.
3. Goroutine A approve rule: `UPDATE mapping_rule_v2 SET status='approved' WHERE id=X` + commit.
4. Goroutine B (5ms sau): NATS publish `cdc.cmd.snapshot.v2 {source_object_id:66}`.
5. Wait worker drain snapshot job.
6. Query shadow table → assert column `<approved field>` exists.

**DoD**: Test `TestSnapshotV2_ApproveRace_BeforeFix` FAIL với assertion missing column.

---

### P1 — Fix G1 — CMS approve always publish

**File**: `cdc-cms-service/internal/app/commands/update_mapping_rule.go`

**Steps**:
1. Edit line 177-179 (xem `03_implementation.md` §1).
2. Đổi NATS payload schema (`internal/infra/messaging/nats_publisher.go`) — thêm `source_object_id` field.
3. Unit test approve handler:
   - Case A: `ShadowTable != nil` → publish OK với cả 2 fields.
   - Case B: `ShadowTable == nil` → vẫn publish với `source_object_id`.
4. Log line `mapping_rule.reload_signal action=... source_object_id=... published=true reason=...`.

**DoD**: AC-3 + AC-4 pass. `go test ./internal/app/commands/...` green.

---

### P2 — Fix G2 — Worker post-reload count check

**File**: `centralized-data-service/internal/handler/snapshot_runner_handler.go` (line 328-335)

**Steps**:
1. Sau `r.registrySvc.ReloadAll(ctx)`, query `dbCount` via `mappingV2Repo.CountActiveBySourceObject(ctx, sourceObjectID)`.
2. Lấy `cacheCount` via `r.registrySvc.MappingCacheSize(bindingID)` (helper mới).
3. Nếu `dbCount > cacheCount`:
   - Log WARN `snapshot.preflight.cache_drift db=%d cache=%d source=%d`.
   - Sleep 200ms.
   - Call `ReloadAll(ctx)` lần 2.
   - Re-check; nếu vẫn drift → log ERROR + metric increment + continue.
4. Add helper `MappingCacheSize(bindingID int64) int` vào `MetadataRegistryService` interface.

**DoD**: AC-5 pass. Trace test scenario có log drift line + retry behavior.

---

### P3 — Defense — CMS dispatch publish reload trước snapshot

**File**: `cdc-cms-service/internal/api/source_object_actions_handler.go:561 SnapshotV2`

**Steps**:
1. Trước khi `bus.Dispatch(SnapshotV2Command)`, gọi `h.nats.PublishReload(sourceObjectID=X, reason="snapshot_v2_dispatch")`.
2. `time.Sleep(50 * time.Millisecond)` (giải thích trong comment với link ADR-03).
3. Tiếp tục dispatch command.

**DoD**: AC-7 pass. NATS subject order verify trong integration test.

---

### P4 — Observability — 2 metric + log line

**Files**:
- `centralized-data-service/internal/service/metadata_registry_service.go` — register Prometheus metric + update on ReloadAll.
- `centralized-data-service/internal/observability/metrics.go` — declare metrics.

**Steps**:
1. Declare:
   ```go
   var MappingCacheSize = promauto.NewGaugeVec(...,"mapping_cache_size","Number of cached mapping rules per source", []string{"source_object_id"})
   var MappingCacheLastReloadedSeconds = promauto.NewGauge(...,"mapping_cache_last_reloaded_seconds",...)
   ```
2. Trong `ReloadAll` cuối hàm: cập nhật metric cho mỗi source_object_id.
3. Set `MappingCacheLastReloadedSeconds.Set(float64(time.Now().Unix()))` cuối ReloadAll.

**DoD**: AC-6 pass. `curl localhost:9090/metrics | grep mapping_cache_size` show value.

---

### P5 — CI grep gate + integration test

**Files**:
- `.github/workflows/ci.yml` (hoặc tương đương) — thêm step grep.
- `centralized-data-service/internal/handler/snapshot_runner_race_test.go` (new) — integration test.

**Steps**:
1. CI step (xem `03_implementation.md` §5):
   ```yaml
   - name: Anti-pattern reload gate
     run: scripts/check_no_conditional_publish.sh
   ```
2. Integration test `TestSnapshotV2_AfterApprove_NoRace`:
   - Reuse harness P0.
   - SAU FIX: assert column exists.

**DoD**: AC-1 (TRƯỚC fix) FAIL; AC-2 (SAU fix) PASS. CI green.

---

### P6 — Deploy + verify prod

**Pre-deploy**:
- Backup DB snapshot.
- Verify cả 2 service đều build pass + test pass.

**Steps**:
1. Deploy `cdc-cms-service` mới (CMS approve handler + dispatch + payload schema).
2. Deploy `centralized-data-service` mới (worker post-reload check + metrics).
3. Smoke test:
   - Approve 1 mapping rule mới cho source test.
   - Trigger snapshot.v2.
   - Assert mapping field xuất hiện shadow + master.
4. Monitor 24h.

**Rollback** (nếu fail):
- Revert commit P1+P2+P3+P4, redeploy.
- Schema NATS payload backwards-compat → safe rollback (worker mới handle msg cũ).

**DoD**: 24h sau deploy, 0 lỗi mapping miss, metric `mapping_cache_size` flat.

---

### P7 — Lesson append (ADR-06)

**File**: `agent/memory/global/lessons.md` (APPEND, không overwrite).

**Steps**:
1. View tail file để xác định điểm append.
2. APPEND entry mới (xem `09_tasks_solution.md` §lesson):
   - Pattern: producer-conditional inverse của L-3110.
   - Global Pattern A/B/X/Y format.
3. Update `05_progress.md` cuối workspace.
4. Set `07_status.md` → COMPLETED.

**DoD**: `grep "producer-conditional" agent/memory/global/lessons.md` show match.

---

## Risk → Mitigation

| Risk | Phase | Mitigation |
|---|---|---|
| R1 (NATS spam bulk approve) | P1 | NATS throughput thoải mái; worker ReloadAll idempotent + có lock guard |
| R2 (worker cũ chưa restart, key string vs int64) | P6 | Deploy doc: rolling restart all worker pod SAU merge `0289fe4` + này |
| R5 (race chưa eliminate 100%) | P2 | Post-reload count + retry; log ERROR cho residual race operator detect |
| R6 (integration test flaky) | P5 | Use `chan` synchronize, không `time.Sleep` |

---

## Stop Rule (CLAUDE.md §8)

Muscle fail > 3 lần ở 1 phase:
1. **STOP**.
2. Append `05_progress.md` ESCALATE.
3. Notify Brain → re-plan.

---

## Gate verification script

```bash
# Gate G0 (sau P0)
go test -v ./internal/handler/ -run TestSnapshotV2_ApproveRace_BeforeFix
# expect: FAIL với "column missing"

# Gate G1 (sau P1)
grep -rn "ShadowTable.*nil.*PublishReload" cdc-cms-service/internal/ # expect: 0
go test -v ./internal/app/commands/ -run TestUpdateMappingRule_AlwaysPublish

# Gate G2 (sau P2)
go test -v ./internal/handler/ -run TestSnapshotPreflight_DriftRetry

# Gate G3 (sau P3)
go test -v ./internal/api/ -run TestSnapshotV2Dispatch_PublishReloadFirst

# Gate G4 (sau P4)
curl localhost:9090/metrics | grep "mapping_cache_size\|mapping_cache_last_reloaded"

# Gate G5 (sau P5)
gh pr checks <PR> # all green
go test -v ./internal/handler/ -run TestSnapshotV2_AfterApprove_NoRace -count=5
# expect: PASS 5/5

# Gate G6 (sau P6, prod)
kubectl logs deploy/centralized-data-service --since=30m | grep "snapshot.preflight.cache_drift_unresolved"
# expect: 0
```

---

## Timeline

```
T+0h      P0 reproduce
T+0.5h    P1 CMS publish always   ┐
T+1.0h    P2 worker preflight     ├─ parallel
T+1.5h    P3 CMS dispatch         │
T+2.0h    P4 observability        ┘
T+2.5h    P5 CI + integration test
T+3.0h    P6 deploy
T+3.5h    P7 lesson append
          DONE
```

---

## Communication protocol

- **Trước P6 deploy**: Muscle xin User confirm.
- **Sau P6**: Muscle báo log + metric, User sign-off `05_progress.md`.
- **Rollback**: STOP, escalate Brain.
