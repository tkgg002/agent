# Tasks: Data Integrity (FINAL)

> Date: 2026-04-16
> Merged: v2 + deep analysis

## Phase 1: Infra + Models ✅
- [x] T1: Migration `008_reconciliation.sql`
- [x] T2: Models (Worker + CMS)
- [x] T3: Kafka `cleanup.policy=compact` (3 topics)
- [x] T4: Debezium signal collections (2 DBs)

## Phase 2: Agents ✅
- [x] T5: Go MongoDB driver + client + config
- [x] T6: recon_source_agent.go (Tier 1-3: count + ID batch + Merkle hash)
- [x] T7: recon_dest_agent.go (Tier 1-3: count + ID batch + Merkle hash)

## Phase 3: Core + Heal ✅
- [x] T8: recon_core.go (tiered + version-aware heal + audit)
- [x] T9: Version-aware Heal (timestamp compare → UPSERT or SKIP)
- [x] T10: Audit Log (via ActivityLog)

## Phase 4: Worker Hardening ✅
- [x] T11: BatchBuffer → failed_sync_logs (DLQ + classifyError)
- [x] T12: Prometheus counters (SyncSuccess, SyncFailed, ConsumerLag, ReconDrift)
- [ ] T13: Schema version validation (Schema Registry check)

## Phase 5: CMS + FE ✅
- [x] T14: CMS API (12 endpoints: report, check tiers, heal, failed logs, retry, tools)
- [x] T15: FE Data Integrity Dashboard (2 tabs + 4 stats cards + actions)
- [x] T16: FE failed_sync_logs viewer + retry (merged into T15)

## Phase 6: Verify + Heal
- [ ] T17: Detect lệch hiện tại
- [ ] T18: Heal → count match
- [ ] T19: Progress + docs

## Phase 7: Bổ sung (từ doc 1 coverage check)
- [ ] T20: Schema History Topic — verify retention unlimited
- [ ] T21: Consumer lag Prometheus metric + alert threshold
- [ ] T22: CMS tools — Reset Debezium offset + Trigger snapshot + Reset Kafka offset (API + FE)
