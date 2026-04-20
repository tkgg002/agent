# Plan: Data Integrity — Reconciliation System

> Date: 2026-04-16
> Phase: data_integrity

## Architecture

```
┌─────────────────────────────────────────┐
│ Reconciliation Worker (Go, periodic)    │
│                                         │
│ 1. Connect MongoDB source (direct)      │
│ 2. Connect Postgres destination         │
│ 3. Per-table comparison:                │
│    - Count source vs dest               │
│    - ID set diff → find missing         │
│    - Sample hash → find corrupted       │
│ 4. Report → cdc_reconciliation_report   │
│ 5. Auto-heal:                           │
│    - Missing → fetch from source + insert│
│    - Stale → flag for re-sync           │
│ 6. Alert if drift > threshold           │
└─────────────────────────────────────────┘
```

## Hiện trạng data lệch (đã confirm)

```
export_jobs:     MongoDB = ? | Postgres = 115 (102 Airbyte + 13 Debezium)
refund_requests: MongoDB = ? | Postgres = 1713 (1712 Airbyte + 1 Debezium)
```
→ Cần query MongoDB count để biết chênh lệch bao nhiêu.

## Tasks

### T1: Migration — cdc_reconciliation_report table
```sql
CREATE TABLE cdc_reconciliation_report (
    id BIGSERIAL PRIMARY KEY,
    target_table VARCHAR(200),
    source_count BIGINT,
    dest_count BIGINT,
    diff BIGINT,
    missing_ids JSONB,          -- IDs có ở source nhưng không có ở dest
    status VARCHAR(20),          -- ok, drift, error
    check_type VARCHAR(20),      -- count, id_set, hash
    duration_ms INT,
    checked_at TIMESTAMP DEFAULT NOW()
);
```

### T2: Worker — Reconciliation Service
- File: `internal/service/reconciliation_worker.go`
- Connect MongoDB direct (Go MongoDB driver)
- Per-table: count + ID set comparison
- Write report → `cdc_reconciliation_report`
- Chạy theo schedule (cdc_worker_schedule table)

### T3: Auto-heal — missing records
- Missing IDs → fetch full document from MongoDB → insert Postgres
- Bypass Kafka (trực tiếp MongoDB → Postgres) cho reconciliation
- Log vào activity_log

### T4: CMS API
- `GET /api/reconciliation/report` — latest report per table
- `POST /api/reconciliation/check` — trigger check ngay
- `POST /api/reconciliation/heal/:table` — trigger auto-heal

### T5: CMS FE — Data Integrity Dashboard
- Page: `/data-integrity`
- Table comparison: source vs dest count, diff, status
- Action: Check Now, Heal, View Missing IDs

### T6: Go MongoDB driver dependency
- `go.mongodb.org/mongo-driver`
- Config: mongodb connection string trong config-local.yml

## Execution order
```
T6 (dependency) → T1 (migration) → T2 (service) → T3 (auto-heal) → T4 (API) → T5 (FE)
```

## Definition of Done
- [ ] Dashboard hiện source vs dest count per table
- [ ] Detect missing records (IDs có ở source, không có ở dest)
- [ ] Auto-heal: fetch missing → insert Postgres
- [ ] After heal: count match
- [ ] Schedule: auto-check mỗi 5 phút
