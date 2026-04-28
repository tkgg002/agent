# Task Checklist — Systematic Flow

> Stage 4 · Phase: `systematic_flow` · 2026-04-24
> Status legend: `[ ]` pending · `[~]` in-progress · `[x]` done · `[!]` blocked

## Track 0 — Migrations

- [x] **T0.1** Create `centralized-data-service/migrations/027_systematic_sources.sql` (cdc_internal.sources + cdc_internal.cdc_wizard_sessions).
- [x] **T0.2** Create `centralized-data-service/migrations/028_sonyflake_fallback_fn.sql` (`gen_sonyflake_id()` + `ensure_shadow_sonyflake_trigger()`).
- [x] **T0.3** Dryrun migration on local goopay_dw — verify no conflict with 001–026.

## Track 1 — BE Sources (Task 1 Boss)

- [x] **T1.1** `internal/model/source.go` — Source struct + `TableName()`.
- [x] **T1.2** `internal/repository/source_repo.go` — `Upsert`, `List`, `GetByID`, `GetByConnectorName`, `MarkDeleted`.
- [x] **T1.3** `internal/api/sources_handler.go` — `NewSourcesHandler`, `List`, `Get`.
- [x] **T1.4** Edit `internal/api/system_connectors_handler.go`.
- [x] **T1.5** Edit `internal/server/server.go`.
- [x] **T1.6** Edit `internal/router/router.go`.

## Track 2 — BE ShadowAutomator (Task 2 Boss)

- [x] **T2.1** `internal/service/shadow_automator.go`.
- [x] **T2.2** Edit `internal/api/registry_handler.go` — sync EnsureShadowTable + rollback on fail.
- [x] **T2.3** Edit `internal/server/server.go` — ShadowAutomator wired.
- [x] **T2.4** Local test: ID auto-gen confirmed (AC3 + AC4).

## Track 3 — BE Wizard + Atomic Swap (Task 3 Boss)

- [x] **T3.1** `internal/model/wizard_session.go`.
- [x] **T3.2** `internal/repository/wizard_repo.go`.
- [x] **T3.3** `internal/service/master_swap.go`.
- [x] **T3.4** `internal/api/wizard_handler.go`.
- [x] **T3.5** Edit `internal/api/master_registry_handler.go` — Swap handler added.
- [x] **T3.6** Edit `internal/router/router.go` — wizard + swap routes landed.
- [x] **T3.7** Edit `internal/server/server.go`.

## Track 4 — Frontend (Option A)

- [x] **T4.1** Rewrite `SourceToMasterWizard.tsx`.
- [x] **T4.2** Edit `TableRegistry.tsx` — dropdown + auto-fill.

## Track 5 — Verify (Stage 6)

- [x] **T5.1** `go build ./...` green.
- [x] **T5.2** `npx tsc --noEmit` green.
- [x] **T5.3** SQL dryrun on local goopay_dw (027 + 028) — COMMIT.
- [x] **T5.4** Manual smoke: Sonyflake trigger + atomic swap — IDs + renames verified.
- [x] **T5.5** Summary report + progress append — see `07_status_systematic_flow.md`.

## Exit Criteria

All `[ ]` → `[x]` + Stage 6 checks green + AC1–AC6 (01_requirements §5) demo-able. Then Stage 7 report.
