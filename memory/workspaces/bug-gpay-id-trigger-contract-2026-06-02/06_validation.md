# 06_validation.md — Validation & Test Plan

> Mapping AC (`01_requirements.md`) → command thực thi.

---

## 1. AC → Verify command

| AC | Verify command | Expected |
|---|---|---|
| AC-1 (reproduce) | `go test -run TestReproduceGpayIDNullBug -v` (TRƯỚC fix) | FAIL với SQLSTATE 23502 |
| AC-2 (fix pass) | `go test -run TestBatchUpsert_V2Shadow_NoExplicitGpayID -v` (SAU fix) | PASS |
| AC-3 (heal existing) | `psql -c "SELECT atthasdef FROM pg_attribute WHERE attrelid='data_hub.tokens'::regclass AND attname='_gpay_id'"` | `t` (true) |
| AC-4 (e2e batch) | `go test -count=3 -race -run TestBatchUpsert_V2Shadow` | PASS 3/3 |
| AC-5 (comment-code) | `grep -c "sonyflake trigger fills" internal/handler/batch_buffer.go` + `grep -c "cdc_internal.sf_nextval" internal/handler/batch_buffer.go` | `0` và `≥ 1` |
| AC-6 (idempotent) | Apply migration 2 lần liên tiếp | Không lỗi |
| AC-7 (sonyflake decode) | `TestSonyflakeIDDecode` | `machine-id` match session var |
| AC-8 (perf) | `go test -bench BenchmarkBatchUpsert_5000 -count=5` | Delta latency ≤ +5% |

---

## 2. Gate verification per phase

### Gate G0 (sau P0)
```bash
go test -v ./internal/handler/ -run TestReproduceGpayIDNullBug 2>&1 | grep "null value in column \"_gpay_id\""
# expect: match (reproduce confirmed)
```

### Gate G1 (sau P1 migration)
```bash
# Function exists
psql $DB -c "\df cdc_internal.sf_nextval" | grep sf_nextval

# Idempotent
psql $DB -f migrations/schema/ids/019_sonyflake_default_fill.sql
psql $DB -f migrations/schema/ids/019_sonyflake_default_fill.sql
echo "Exit: $?"  # expect: 0

# Heal verified
psql $DB -tAc "SELECT count(*) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE a.attname='_gpay_id' AND a.atthasdef=false AND n.nspname NOT IN ('pg_catalog','information_schema','cdc_internal','cdc_system')"
# expect: 0 (all healed)
```

### Gate G2 (sau P2 Go DDL)
```bash
cd data-hub/centralized-data-service
grep -c 'BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval()' internal/sinkworker/schema_manager.go
# expect: 1
go build ./... && go vet ./...
go test ./internal/sinkworker/ -run TestCreateShadowTable_HasDefault -v
```

### Gate G3 (sau P3 comment)
```bash
grep -c "sonyflake trigger fills" internal/handler/batch_buffer.go        # expect: 0
grep -c "cdc_internal.sf_nextval" internal/handler/batch_buffer.go        # expect: ≥ 1
```

### Gate G4 (sau P4 integration test)
```bash
go test -v -race -count=3 ./internal/handler/ -run TestBatchUpsert_V2Shadow
go test -v -race ./internal/handler/ -run TestSonyflakeIDDecode
go test -bench BenchmarkBatchUpsert_5000 -benchmem -count=5 ./internal/handler/
```

### Gate G5 (sau P5 deploy prod)
```bash
# Migration applied
psql $PROD -c "\d+ data_hub.tokens" | grep "DEFAULT cdc_internal.sf_nextval"

# 0 errors trong 30 phút
kubectl logs deploy/centralized-data-service --since=30m | grep -c "null value in column \"_gpay_id\""
# expect: 0

# Throughput stable
kubectl exec deploy/centralized-data-service -- curl localhost:9090/metrics | grep batch_upsert_total
```

### Gate G6 (sau 24h prod)
- Grafana: error_rate flat
- Grafana: latency_p99 ≤ baseline + 5%
- Grafana: throughput ≥ baseline - 5%

---

## 3. Regression checklist

Sau khi apply tất cả patch, verify các flow KHÔNG bị break:

- [ ] V1 table (per-table sequence) — INSERT vẫn OK
- [ ] V2 shadow table — INSERT không chỉ định `_gpay_id` → OK
- [ ] V2 shadow table — INSERT có chỉ định `_gpay_id` → OK (DEFAULT bị override)
- [ ] ON CONFLICT `_source_id` partial UNIQUE — UPSERT idempotent vẫn OK
- [ ] Fencing trigger vẫn raise nếu session var không match
- [ ] Soft-delete (`_deleted = true`) — INSERT new row cùng `_source_id` vẫn OK (partial UNIQUE)

---

## 4. Negative tests

| Test | Command | Expected |
|---|---|---|
| `sf_nextval()` không có session var | `psql -c "RESET app.fencing_machine_id; SELECT cdc_internal.sf_nextval();"` | EXCEPTION `app.fencing_machine_id session var not set` |
| `sf_nextval()` machine_id out-of-range | `psql -c "SET app.fencing_machine_id=99999; SELECT cdc_internal.sf_nextval();"` | EXCEPTION `machine_id out of range` |
| Migration trên schema không có `_gpay_id` | Apply migration trên DB không có V2 shadow | No-op, không lỗi |

---

## 5. Performance baseline

### Baseline trước fix (lấy từ benchmark Go test)
```
BenchmarkBatchUpsert_5000-8    10    150ms/op    50000 allocs/op
```

### Expected sau fix (acceptable)
```
BenchmarkBatchUpsert_5000-8    10    ≤157ms/op  (+5% max)   50000 allocs/op
```

### Cách đo
```bash
# Trước fix
git stash  # nếu có local changes
go test -bench BenchmarkBatchUpsert_5000 -count=5 ./internal/handler/ > before.txt

# Apply fix
git stash pop  # hoặc checkout branch fix
go test -bench BenchmarkBatchUpsert_5000 -count=5 ./internal/handler/ > after.txt

# Compare
benchstat before.txt after.txt
```

---

## 6. CI integration

### GitHub Actions matrix
```yaml
- name: Run V2 shadow tests
  run: |
    go test -race -count=3 ./internal/handler/ -run TestBatchUpsert_V2Shadow

- name: Contract drift gate
  run: |
    ! grep "sonyflake trigger fills" internal/handler/batch_buffer.go
    grep "cdc_internal.sf_nextval" internal/handler/batch_buffer.go

- name: Migration idempotency
  run: |
    psql $TEST_DB -f migrations/schema/ids/019_sonyflake_default_fill.sql
    psql $TEST_DB -f migrations/schema/ids/019_sonyflake_default_fill.sql
```

---

## 7. Production monitoring

### Metrics to watch
| Metric | Alert threshold |
|---|---|
| `batch_upsert_error_total{reason="null_gpay_id"}` | > 0 in 5min → page on-call |
| `batch_upsert_duration_seconds_p99` | > baseline × 1.10 in 15min → warning |
| `sink_lag_seconds` | > 60s sustained 5min → warning |

### Log queries
```bash
# Loki / Splunk
{app="centralized-data-service"} |= "null value in column \"_gpay_id\""
# expect: 0 hits after deploy
```

---

## 8. Sign-off matrix

| Gate | Role | Sign |
|---|---|---|
| G0-G4 (test) | Muscle | Pass screenshot trong `05_progress.md` |
| G5 (deploy) | Muscle + User | Append `[User] APPROVE DEPLOY` |
| G6 (24h) | Muscle | Append `[Muscle] T6.1 DONE — metrics stable` |
| Final | User | Append `[User] APPROVE WORKSPACE COMPLETE` |
