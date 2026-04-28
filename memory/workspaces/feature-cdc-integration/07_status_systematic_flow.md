# Status Report — Systematic Connect→Master Flow

> Phase: `systematic_flow` · Stages 1–6 complete · 2026-04-24
> Muscle: claude-opus-4-7-1m
> Boss DoD: "Admin chọn Source từ Dropdown → click Automate Everything → data chảy vào Master."

---

## 1. What shipped

| Track | Files | Status |
|:-|:-|:-|
| T0 Migrations | `027_systematic_sources.sql`, `028_sonyflake_fallback_fn.sql` | ✅ applied on local goopay_dw |
| T1 Sources BE | `model/source.go`, `repository/source_repo.go`, `api/sources_handler.go`, edit `system_connectors_handler.go` | ✅ compile clean |
| T2 Shadow Automator | `service/shadow_automator.go`, edit `registry_handler.go` | ✅ compile clean |
| T3 Wizard + Swap BE | `model/wizard_session.go`, `repository/wizard_repo.go`, `service/master_swap.go`, `api/wizard_handler.go`, edit `master_registry_handler.go` | ✅ compile clean |
| T4 FE | rewrite `SourceToMasterWizard.tsx`, edit `TableRegistry.tsx` | ✅ `tsc --noEmit` clean |
| Wiring | `server.go`, `router.go` | ✅ compile clean |

Totals: 2 SQL + 8 new Go + 4 edited Go + 2 FE = **16 files**.

## 2. New API surface

Shared (admin|operator reads):
- `GET  /api/v1/sources`
- `GET  /api/v1/sources/:id`
- `GET  /api/v1/wizard/sessions/:id`
- `GET  /api/v1/wizard/sessions/:id/progress`

Destructive (ops-admin + idempotency + audit):
- `POST  /api/v1/wizard/sessions`
- `POST  /api/v1/wizard/sessions/:id/execute`
- `PATCH /api/v1/wizard/sessions/:id`
- `POST  /api/v1/masters/:name/swap`

## 3. Verification matrix

| AC | Requirement | Status | Evidence |
|:-|:-|:-|:-|
| AC1 | POST /connectors → sources row | Code-complete | Upsert path in `system_connectors_handler.Create` tail |
| AC2 | Register modal dropdown source + auto-fill → 202 sync | Code-complete | `TableRegistry.tsx` + `EnsureShadowTable` sync in Register |
| AC3 | `\d cdc_internal.<t>` has 8 cols + fallback trigger | ✅ verified | Smoke table `systematic_smoke` created; trigger attached by `ensure_shadow_sonyflake_trigger()` |
| AC4 | INSERT without id → Sonyflake id | ✅ verified | IDs 41049564853043215, 41049564886597648 auto-generated |
| AC5 | "Automate Everything" + session_id URL resume | Code-complete | `SourceToMasterWizard.tsx` full rewrite with `useSearchParams('session_id')` + 2s poll while running |
| AC6 | Atomic swap in 1 TX | ✅ verified | `smoke_master` ↔ `smoke_master_v2` swap with `SET LOCAL lock_timeout='3s'`; old row retained as `_old_probe` |

Go build: `go build ./...` EXIT=0. TS: `npx tsc --noEmit` EXIT=0. Migrations idempotent (safe to re-apply).

## 4. Design decisions applied (Boss conservative lock-in)

- **Q1 (schema)**: kept 8-col `create_cdc_table` layout — no downgrade.
- **Q2 (schema namespace)**: `cdc_internal.<target>` preserved.
- **Q3 (trigger mode)**: fallback-only — `tg_sonyflake_fallback()` fires only when `NEW.id IS NULL OR 0`. Go worker path remains authoritative.
- **Q4 (sources location)**: `cdc_internal.sources` (not `public.`). FK from `cdc_wizard_sessions.connector_id`.
- **Q5 (wizard)**: Option A — full rewrite, stateful via API, session_id in URL, 2s poll.
- **Atomic Swap**: `BEGIN; SET LOCAL lock_timeout='3s'; ALTER RENAME x2; COMMIT;` — lock timeout surfaces as 409 `lock_timeout`.
- **EnsureShadowTable bootstrap**: inlines `CREATE OR REPLACE` for `gen_sonyflake_id()` + trigger body + helper, so the automator self-heals even if migration 028 hasn't run yet.

## 5. What is left for runtime acceptance (Boss hands-on)

1. Deploy migrations 027 + 028 against prod goopay_dw (dryrun OK locally).
2. Restart cdc-cms-service with new binaries.
3. `npm run dev` on cdc-cms-web, open `/source-to-master`.
4. E2E smoke: create MongoDB connector via `/sources` → verify `GET /api/v1/sources` returns row → navigate to `/registry` → dropdown shows the source → submit → shadow row visible with `is_table_created=true`.
5. Optional: once a master v2 has been populated, call `POST /api/v1/masters/<name>/swap` to prove AC6 on a real master.

## 6. Risks / follow-ups

- **E2E pipeline automation**: `Execute` currently flips status→running + logs intent; the FE still drives the 11-step pipeline via existing endpoints. A next pass can move the Create-connector / Upsert-source / Register-shadow / Snapshot sequence fully server-side and stream real step-level progress. Captured implicitly in `02_plan_systematic_flow.md`.
- **Prod DB role**: verify the goopay_dw role has `CREATE FUNCTION`, `CREATE TRIGGER`, `ALTER TABLE` privileges in prod (local user has them).
- **Legacy NATS path**: `cdc.cmd.create-default-columns` still publishes on Register; Worker short-circuits when `is_table_created=true` but the publish cost remains. Can be pruned after a quiet observation window.

## 7. Files touched (absolute paths)

Migrations (create):
- `cdc-system/centralized-data-service/migrations/027_systematic_sources.sql`
- `cdc-system/centralized-data-service/migrations/028_sonyflake_fallback_fn.sql`

Go new:
- `cdc-system/cdc-cms-service/internal/model/source.go`
- `cdc-system/cdc-cms-service/internal/model/wizard_session.go`
- `cdc-system/cdc-cms-service/internal/repository/source_repo.go`
- `cdc-system/cdc-cms-service/internal/repository/wizard_repo.go`
- `cdc-system/cdc-cms-service/internal/service/shadow_automator.go`
- `cdc-system/cdc-cms-service/internal/service/master_swap.go`
- `cdc-system/cdc-cms-service/internal/api/sources_handler.go`
- `cdc-system/cdc-cms-service/internal/api/wizard_handler.go`

Go edit:
- `cdc-system/cdc-cms-service/internal/api/system_connectors_handler.go` (+ sourceRepo, fingerprint persist, soft-delete, parseFingerprint)
- `cdc-system/cdc-cms-service/internal/api/registry_handler.go` (+ automator sync call + rollback)
- `cdc-system/cdc-cms-service/internal/api/master_registry_handler.go` (+ Swap endpoint)
- `cdc-system/cdc-cms-service/internal/server/server.go` (+ wire SourceRepo, WizardRepo, ShadowAutomator, MasterSwap)
- `cdc-system/cdc-cms-service/internal/router/router.go` (+ 10 routes)

FE:
- `cdc-system/cdc-cms-web/src/pages/SourceToMasterWizard.tsx` (full rewrite)
- `cdc-system/cdc-cms-web/src/pages/TableRegistry.tsx` (source dropdown + auto-fill + collection select)

---

## 8. Mid-session fix (2026-04-24 14:42) — Wizard route re-tier

After handing the initial report to Boss, runtime test revealed `POST /v1/wizard/sessions` bootstrapped by FE failed the destructive chain (missing Idempotency-Key + later missing reason). Root cause: route classification error — Create/Patch session rows are draft mutations (zero infra side-effect) and should not sit in the destructive chain alongside Execute/Swap.

**Applied fix** (no new files; edits only):
- `router.go`: Create + Patch moved from `registerDestructive` to `admin.Post` / `admin.Patch`. Execute + Swap remain destructive.
- `SourceToMasterWizard.tsx`: Idempotency-Key removed from Create/Patch calls; Execute gained `{reason}` body + `Idempotency-Key` + `X-Action-Reason` headers.
- `agent/memory/global/lessons.md`: appended lesson "Route classification".

**E2E verified end-to-end (no Boss handholding)**:
- T1 (Create) → 201 · T2 (Patch) → 200 · T3 (Execute no reason) → 400 · T4 (Execute with reason) → 202.
- DB: `admin_actions` shows 1 row for Execute, 0 rows for Create/Patch (tier split confirmed).
- `progress_log[0]` contains `{event:"execute_started", step:1, actor:"admin"}`.

**BE rebuild + restart path for the record**: killed PID 93666 (go-build main) → `nohup go run cmd/server/main.go &` in background (task `b2q7h9j28`). `/health` 200.

**Lesson captured**: always classify mutations by real side-effect boundary (shared-infra DDL / NATS fan-out / data rename) — not by HTTP verb — before mounting middleware.
