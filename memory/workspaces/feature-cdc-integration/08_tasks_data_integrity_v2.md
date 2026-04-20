# Tasks: Data Integrity (v2)

> Date: 2026-04-16
> Source: User's requirements + plan v2

## Phase 1: Database + Models
- [ ] T1: Migration `008_reconciliation.sql`
- [ ] T2: Models: reconciliation_report.go + failed_sync_log.go (Worker + CMS)
- [ ] T3: AutoMigrate

## Phase 2: Agents
- [ ] T4: Go MongoDB driver + `pkgs/mongodb/client.go` + config
- [ ] T5: `recon_source_agent.go` (count + ID set + hash)
- [ ] T6: `recon_dest_agent.go` (count + ID set + hash)

## Phase 3: Core + Heal
- [ ] T7: `recon_core.go` (orchestrate + compare + report + schedule)
- [ ] T8: Heal logic (MongoDB → Postgres bypass Kafka)

## Phase 4: Worker Hardening
- [ ] T9: BatchBuffer error → `failed_sync_logs` table
- [ ] T10: Prometheus counters `cdc_sync_success_total` / `cdc_sync_failed_total`

## Phase 5: CMS
- [ ] T11: API endpoints (report, check, heal, failed logs)
- [ ] T12: FE Data Integrity Dashboard
- [ ] T13: FE failed_sync_logs viewer

## Phase 6: Verify
- [ ] T14: Detect lệch hiện tại → heal → count match
- [ ] T15: Progress + docs update
