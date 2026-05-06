# Phase 2 v2 — Implementation — 4 Pillar CQRS Decoupling

> **Plan ref**: `02_plan_phase2_decoupling.md`
> **Task ref**: `08_tasks_phase2_decoupling.md`
> **Snapshot date**: 2026-05-07 (closed §7 Full Doc Set)

---

## 1. Pillar status (snapshot)

| Pillar | Plan DoD | Reality | Status |
|---|---|---|---|
| P1 — Skeleton + interface contract | `domain/`, `app/{ports,queries,commands}/`, `infra/` exist; build PASS | Đầy đủ 5 domain (`mapping/source/master/reconciliation/job`); `app/ports/` 4 interface; `infra/{persistence,messaging,cache,http}/` | ✅ DONE |
| P2 — Read paths → `app/queries/` | 15 endpoint READ qua QueryBus; 0 raw SQL trong handler GET/List | 23 query file (`list_*`, `get_*` + read models); `infra/persistence/*_read_repo_gorm.go` 10 file | ✅ DONE (bonus +8 query so plan 15) |
| P3 — Write paths → `app/commands/` + NATS bus | 24 endpoint WRITE qua CommandBus; 2 INLINE (Master Swap + V2 Sync) → worker; 0 hit `ALTER TABLE.*RENAME` / `SyncFromLegacy` trong api/app | 34 command file qua CommandBus; Master Swap → worker ✅; **`SyncFromLegacy` còn 3 hit** (G-A) | ⚠️ **90% DONE** |
| P4 — `infra/persistence/` chuẩn hóa, eliminate raw SQL, drain `internal/service/` | `grep db.Raw` = 0 trong api/app; `internal/service/` empty; `internal/repository/` migrated | `db.Raw` còn 10 hit / 3 file (G-B); `internal/service/` 22 logic file (G-C); `internal/repository/` 6 file legacy (G-D) | ⚠️ **~50% DONE** |

---

## 2. Commit chain (cdc-cms-service repo)

### P3 destructive migration scope (7 đợt, 27 endpoint / 24 handler)

| Đợt | Commit | Subject |
|---|---|---|
| 1 | (P3 Đợt 1) | 4 sync metadata cmd: ack_alert, mapping_rule × 2, reject_master |
| 2 | (P3 Đợt 2) | 4 cmd: create_master, create_wizard, patch_wizard, +1 |
| 3 | (P3 Đợt 3) | 4 cmd source_async path |
| 4 | (P3 Đợt 4) | 4 cmd recon path |
| 5 | (P3 Đợt 5) | 4 cmd system_async path |
| 6 | (P3 Đợt 6) | 1 cmd transmute_run |
| 7 | `2f5040b` | 6 system_connectors endpoint qua bus |

### T13–T17 hardening + test uplift (workspace closure)

| Task | Commit | Subject |
|---|---|---|
| T13 P3 | `7ea23d7` | ActivityLog helper extraction |
| T14 P4 | `084a4a1` | V1↔V2 dedup |
| T15 P5 | `477ba19` | Health collector probe split |
| T16 P6 | `4a2a6e7` | V2 sync atomicity |
| T17 P7 đợt 1 | `48567b8` | Pure-fn tests state machine + health compute |
| T17 P7 đợt 2 | `25f645d` | Shadow ident + alert rules + URL sanitizer tests |
| T17 P7 đợt 3 | `5c95ab5` | HTTP probe wire-contract via httptest stub |
| T17 P7 đợt 4 | `2f6b3b1` | Provisioning trace helpers + kafka-exporter probe |
| T17 P7 đợt 5 | (close-out) | DoD audit + lesson abstraction |

**Coverage delta T17 P7**: service 20.0% → 21.4%, **service/health/probes 0% → 82.8%**, 60 test mới qua 4 sqlmock-free lane (pure-fn / httptest / nil-receiver / trace-context).

---

## 3. Worker side evidence (live verify 2026-05-07)

- Container `gpay-cdc-worker` UP 38h (no restart needed).
- 6/6 enabled schedule fire mỗi 60s top-of-minute, all `last_status='success'`.
- JobMonitor wildcard subscribe `cdc.evt.*.completed` healthy — log `"job monitor: schedule closed"` xuất hiện sau mỗi transmute event.
- Bridge E2E verified ngày 2026-05-06: source `goopay_source.public.orders` → Debezium → `shadow_goopay_source.orders` → cron tick → `goopay_dest:dw_orders.orders_fact` (markers id 79-83 land cả 5/5).
- Sched #1 sample tick: `scanned=32, inserted=31, skipped=1, duration_ms=39` ✅.

→ **Worker side P3 (cmd/evt subscribe + JobMonitor) FULLY OPERATIONAL**.

---

## 4. BE-side gap matrix — Phase 2 v2 chưa close 100%

### G-A — `SyncFromLegacy` còn 3 hit (P3 DoD vi phạm)

```
internal/api/registry_handler.go:147  if err := h.v2sync.SyncFromLegacy(c.Context(), &created); err != nil {
internal/api/registry_handler.go:263  if syncErr := h.v2sync.SyncFromLegacy(c.Context(), updated); syncErr != nil {
internal/api/registry_handler.go:321  if err := h.v2sync.SyncFromLegacy(c.Context(), &e); err != nil {
```

→ V2SyncCommand chưa wrap 3 inline call. Plan P3 spec: phải qua `cdc.cmd.v2-sync` subject + worker-side handler. Pending **Task #18**.

### G-B — `db.Raw` còn 10 hit / 3 file (P4 DoD vi phạm)

| File | Hit count | Highlight |
|---|---|---|
| `internal/api/reconciliation_handler.go` | 4 | `:180, :518, :701, :702` — count + scope query |
| `internal/api/source_object_actions_handler.go` | 5 | `:59, :532, :550, :553, :559` — sample preview, raw_data count |
| `internal/api/registry_handler.go` | 1 | `:490` — table_exists check via `information_schema` |

→ Pending **Task #18** (gộp với G-A vì cùng file `registry_handler.go`).

### G-C — `internal/service/` chưa drain (43 file = 22 logic + 21 test)

Files cần migrate sang `app/commands/` hoặc `domain/`:
- `master_swap.go` → `app/commands/master_swap.go` (logic) + `domain/master/swap.go` (state)
- `provisioning_orchestrator.go` + `provisioning_state_machine.go` → `domain/master/provisioning/`
- `source_object_v2_sync.go` → `app/commands/v2_sync.go` (sync với G-A migrate)
- `shadow_automator.go` → `app/commands/shadow_automate.go`
- `system_health_collector.go` + `system_health_compute.go` + `system_health_alerts.go` + `system_health_queries.go` → `app/queries/system_health/` + `infra/persistence/system_health_*.go`
- `activity_logger.go` → `infra/persistence/activity_log_writer.go`
- `alert_manager.go` → `app/commands/alert_*.go` (đã có `ack_alert`, `silence_alert`)
- `stuck_job_reaper.go` → `app/commands/job_reaper.go` hoặc `infra/messaging/stuck_job_reaper.go` (background worker)
- `prom_client.go` → `infra/http/prom_client.go`
- `approval_service.go`, `reconciliation_service.go` → `app/commands/`

→ Pending **Task #19**.

### G-D — `internal/repository/` 6 file legacy duplicate với `infra/persistence/`

| Legacy file | Đã có replacement? |
|---|---|
| `mapping_rule_repo.go` | ✅ `infra/persistence/mapping_rule_repo_gorm.go` (P3 đợt 1 created) |
| `registry_repo.go` | ⚠️ partial — `source_object_read_repo_gorm.go` (read only, write còn ở repository/) |
| `source_repo.go` | ⚠️ partial — same as above |
| `wizard_repo.go` | ❌ chưa có replacement trong `infra/persistence/` |
| `pending_field_repo.go` | ❌ chưa có replacement |
| `schema_log_repo.go` | ❌ chưa có replacement |

→ Pending **Task #19** (gộp).

---

## 5. Backlog mở để close 100%

| Task | Subject | Effort | Blocks |
|---|---|---|---|
| **#18** — V2SyncCommand + 3 file db.Raw migrate | G-A + G-B fix | ~1.5d | P3 + P4 close |
| **#19** — Service layer drainage + repository legacy migrate | G-C + G-D fix | ~2d | P4 close |

Sau Task #18 + #19: Phase 2 v2 plan-vs-code drift = 0; workspace `feature-cdc-system-refactor` truly closed.

---

## 6. Lessons captured (đã append `agent/memory/global/lessons.md`)

- "Audit-only side-effects không cần CommandBus" (Lesson #1298, commit `a0f9eb3`).
- "Test uplift dưới project-convention no sqlmock — 4 sqlmock-free lane" với A/B/X/Y variables (T17 P7 close-out).
- "Repository adapter layer ≠ unit test target — testcontainers thay sqlmock" (T18 backlog spawn).

---

## 7. Verification commands (re-run any time để re-check DoD)

```bash
cd cdc-cms-service

# G-A check
grep -rn "SyncFromLegacy" internal/api/ internal/app/ | wc -l   # expect 0 sau Task #18

# G-B check
grep -rn "db\.Raw\|db\.Exec" internal/api/ internal/app/ | wc -l   # expect 0 sau Task #18

# G-C check
find internal/service -type f -name "*.go" -not -name "*_test.go" | wc -l   # expect 0 sau Task #19

# G-D check
find internal/repository -type f -name "*.go" | wc -l   # expect 0 sau Task #19

# Build + test gate
go build ./... && go test ./... -count=1 -short
```

---

## 8. Governance checklist (CLAUDE.md)

- §0 tiếng việt + planning + skill ✓
- §3 Plan & Verify ✓ (live BE health probe + DB row + worker log evidence)
- §7 Full Doc Set ✓ (01/02/**03** mới tạo file này/08/09 đầy đủ)
- §11 APPEND-only memory ✓ (file này NEW; không overwrite progress)
- §13 Lesson abstract ✓ (3 Global Pattern lesson đã append)
- §14 Pre-flight ✓ (BE :8083 verified up trước khi snapshot)
