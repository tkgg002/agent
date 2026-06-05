# 08_tasks_release.md — Task Breakdown (Release Prod)

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-03
> Trạng thái: ⬜ todo · 🟡 in-progress · ✅ done · ⛔ blocked

## Phase A — Master/Transmute finalize (06-03→06-05)
- ⬜ A1 — FE P1 Sync Modal + fix bug `.find` (lọc theo master_table)
- ⬜ A2 — FE P3 tooltip TransmuteSchedules
- ⬜ A3 — Revise doc P2 (bỏ DB-plumbing sai)
- ⬜ A4 — BE P2 sinkworker post_ingest gate
- ⬜ A5 — E2E 3 mode (run_now/cron/post_ingest) + evidence
- ⬜ A6 — Security gate Phase A

## Phase B — SystemHealth audit+fix (06-05→06-09)
- ⬜ B1 — Audit /api/system/health + worker probes + 4 PG ping
- ⬜ B2 — Audit slow-SQL guards (no regress)
- ⬜ B3 — Audit dependency health (Kafka/SchemaReg/NATS/Redis/Debezium)
- ⬜ B4 — Gap analysis + fix HIGH/MED + build/vet/test
- ⬜ B5 — E2E degraded-path (tắt 1 dep)
- ⬜ B6 — Security gate

## Phase C — Reconcile audit+fix (06-09→06-13)
- ⬜ C1 — Audit 3-tier hash window
- ⬜ C2 — Audit healer OCC + Mongo primary read
- ⬜ C3 — Verify fix reconcile-mongodb-not-configured trên V2
- ⬜ C4 — Audit scheduler cycle + fencing + multi-source
- ⬜ C5 — Gap + fix + build/vet/test
- ⬜ C6 — E2E drift→detect→heal→verify
- ⬜ C7 — Security gate (no PII leak)

## Phase D — Monitor audit+fix (06-13→06-16)
- ⬜ D1 — Audit JobMonitor close-loop (no miss/dup)
- ⬜ D2 — Audit Dashboard V2 (số khớp DB)
- ⬜ D3 — Audit snapshot monitor + activity metrics
- ⬜ D4 — Audit trace aggregation 5 luồng + OTel DNS
- ⬜ D5 — Gap + fix + build/test
- ⬜ D6 — E2E transmute→dashboard update + trace
- ⬜ D7 — Security gate

## Phase E — Auth + E2E + Hardening + Staging (06-16→06-22)
- ⬜ E1 — cdc-auth-service test cơ bản + chạy local (BLOCKER) — *song song từ Phase B*
- ⬜ E2 — E2E full 5 luồng (qa-agent/Playwright)
- ⬜ E3 — Dựng/validate Staging (phụ thuộc Boss Q2)
- ⬜ E4 — Migration review + rollback script
- ⬜ E5 — Load/perf smoke theo SLA (Q3)
- ⬜ E6 — Observability hardening (OTel DNS + alert)
- ⬜ E7 — Security review toàn repo (/security-review)

## Phase F — Prod readiness + Release (06-22→06-25)
- ⬜ F1 — Runbook
- ⬜ F2 — Rollback plan
- ⬜ F3 — Go/No-Go checklist (R2+R3)
- ⬜ F4 — Deploy prod + smoke
- ⬜ F5 — Post-release watch 24-48h

## Blocker hiện tại
- ⛔ Q1–Q4 (R5) chờ Boss trả lời → mới chốt được ngày chính xác Phase E/F.
- ⛔ Staging chưa xác nhận tồn tại → Phase E3 có thể nở thời gian.
