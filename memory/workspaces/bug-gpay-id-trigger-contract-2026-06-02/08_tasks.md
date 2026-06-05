# 08_tasks.md — Muscle Task List

> Granular task breakdown. Mỗi task có DoD + verify command.
> Tham chiếu `02_plan.md` (phase) + `03_implementation.md` (patch spec).

---

## Phase P0 — Reproduce bug

### T0.1 — Setup testcontainers Postgres harness
- **File:** `internal/handler/testutil/pg_container.go` (CREATE nếu chưa có)
- **DoD:** Function `SpawnPG(t)` trả về `*pgxpool.Pool` + auto cleanup
- **Verify:** `go test -run TestPGContainerSpawn` pass

### T0.2 — Reproduce test `TestReproduceGpayIDNullBug`
- **File:** `internal/handler/batch_buffer_v2shadow_test.go` (CREATE)
- **DoD:** Test FAIL với SQLSTATE 23502 đúng intent (negative test trước fix)
- **Verify:** `go test -run TestReproduceGpayIDNullBug` → FAIL có lỗi `null value in column "_gpay_id"`

---

## Phase P1 — Migration

### T1.1 — Tạo migration file 019
- **File:** `migrations/schema/ids/019_sonyflake_default_fill.sql` (CREATE)
- **DoD:** File apply qua `make migrate` không lỗi
- **Verify:** `psql -c "\df cdc_internal.sf_nextval"` show function

### T1.2 — Test idempotent
- **DoD:** Chạy migration lần 2 không raise lỗi
- **Verify:**
  ```bash
  psql -f migrations/schema/ids/019_sonyflake_default_fill.sql
  psql -f migrations/schema/ids/019_sonyflake_default_fill.sql  # 2nd time
  ```

### T1.3 — Test ALTER heal existing
- **DoD:** Sau migration, table V2 shadow đã exist có `_gpay_id` với DEFAULT
- **Verify:**
  ```bash
  psql -c "SELECT attname, atthasdef FROM pg_attribute WHERE attrelid='data_hub.tokens'::regclass AND attname='_gpay_id'"
  # expect: atthasdef = true
  ```

### T1.4 — Test function with missing session var
- **DoD:** `sf_nextval()` raise rõ lỗi khi `app.fencing_machine_id` chưa set
- **Verify:** Test SQL `RESET app.fencing_machine_id; SELECT cdc_internal.sf_nextval();` → expect EXCEPTION

---

## Phase P2 — Go DDL

### T2.1 — Edit `schema_manager.go:226`
- **File:** `internal/sinkworker/schema_manager.go`
- **DoD:** Đổi 1 dòng theo `03_implementation.md` §2
- **Verify:**
  ```bash
  grep "BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval()" internal/sinkworker/schema_manager.go
  # expect: 1 match
  ```

### T2.2 — Unit test `TestCreateShadowTable_HasDefault`
- **File:** `internal/sinkworker/schema_manager_test.go` (EDIT/ADD test)
- **DoD:** Test verify DDL emit có chuỗi DEFAULT
- **Verify:** `go test -run TestCreateShadowTable_HasDefault -v`

### T2.3 — Go build + vet
- **Verify:**
  ```bash
  cd data-hub/centralized-data-service
  go build ./...
  go vet ./...
  ```

---

## Phase P3 — Comment fix

### T3.1 — Edit comment `batch_buffer.go:246-250`
- **File:** `internal/handler/batch_buffer.go`
- **DoD:** Comment rewrite theo `03_implementation.md` §3
- **Verify:**
  ```bash
  grep -c "sonyflake trigger fills" internal/handler/batch_buffer.go  # = 0
  grep -c "cdc_internal.sf_nextval" internal/handler/batch_buffer.go  # ≥ 1
  ```

---

## Phase P4 — Integration test

### T4.1 — `TestBatchUpsert_V2Shadow_NoExplicitGpayID` (happy path)
- **File:** `internal/handler/batch_buffer_v2shadow_test.go` (mở rộng từ T0.2)
- **DoD:** Test PASS sau khi P1 + P2 applied
- **Verify:** `go test -count=3 -race -run TestBatchUpsert_V2Shadow_NoExplicitGpayID`

### T4.2 — `TestSonyflakeIDDecode`
- **DoD:** Generated ID decode đúng machine_id = session var
- **Verify:** `go test -run TestSonyflakeIDDecode -v`

### T4.3 — Benchmark `BenchmarkBatchUpsert_5000`
- **DoD:** Latency delta ≤ +5% vs baseline trước fix
- **Verify:** `go test -bench BenchmarkBatchUpsert_5000 -benchmem -count=5`

### T4.4 — CI green gate
- **DoD:** GitHub Actions CI pass
- **Verify:** `gh pr checks <PR>` all green

---

## Phase P5 — Deploy

### T5.1 — Backup snapshot prod DB
- **DoD:** `pg_dump --schema-only` file saved with timestamp
- **Verify:** `ls -la backup_*.sql`

### T5.2 — Test migration trên staging
- **DoD:** Migration apply OK trên staging, smoke test pass
- **Verify:** `psql $STAGING -c "\d+ data_hub.tokens" | grep DEFAULT`

### T5.3 — Apply migration prod
- **DoD:** Migration applied, không lỗi, không lock dài
- **Verify:**
  ```bash
  time psql $PROD -f migrations/schema/ids/019_sonyflake_default_fill.sql
  # expect: < 5s
  ```

### T5.4 — Deploy Go binary mới
- **DoD:** Rollout success, pods Ready
- **Verify:** `kubectl rollout status deploy/centralized-data-service --timeout=120s`

### T5.5 — Smoke test prod
- **DoD:** 0 lỗi `_gpay_id` trong 5 phút đầu sau deploy
- **Verify:**
  ```bash
  kubectl logs -f deploy/centralized-data-service --since=5m | grep -c "null value in column \"_gpay_id\""
  # expect: 0
  ```

---

## Phase P6 — Verify + Lesson

### T6.1 — Monitor 24h metrics
- **DoD:** Error rate, throughput, latency p99 không regression
- **Verify:** Grafana dashboard `centralized-data-service` 24h window

### T6.2 — Append lesson `agent/memory/global/lessons.md`
- **DoD:** Lesson abstract đúng format CLAUDE.md §13
- **Verify:** `grep "Contract Drift" agent/memory/global/lessons.md`

### T6.3 — Final entry `05_progress.md`
- **DoD:** User sign-off entry appended
- **Verify:** `tail -5 05_progress.md` show APPROVE

### T6.4 — Move workspace status → COMPLETED
- **DoD:** `07_status.md` updated
- **Verify:** `head -10 07_status.md` show `COMPLETED`

---

## Task graph

```
T0.1 → T0.2 ─┐
              ├─→ T1.1 → T1.2 → T1.3 → T1.4 ─┐
T2.1 → T2.2 → T2.3 ──────────────────────────┤
T3.1 ────────────────────────────────────────┴─→ T4.1 → T4.2 → T4.3 → T4.4
                                                                       │
                                                                       ▼
                                                  T5.1 → T5.2 → T5.3 → T5.4 → T5.5
                                                                                │
                                                                                ▼
                                                           T6.1 → T6.2 → T6.3 → T6.4
```

---

## Summary table

| Task | Effort | Priority | Owner | Gate |
|---|---|---|---|---|
| T0.1 | 0.2h | P0 | Muscle | - |
| T0.2 | 0.3h | P0 | Muscle | Reproduce confirmed |
| T1.1 | 0.5h | P0 | Muscle | Function exists |
| T1.2 | 0.1h | P0 | Muscle | Idempotent |
| T1.3 | 0.2h | P0 | Muscle | Heal verified |
| T1.4 | 0.2h | P1 | Muscle | Error msg clear |
| T2.1 | 0.1h | P0 | Muscle | Code change |
| T2.2 | 0.2h | P1 | Muscle | Unit test |
| T2.3 | 0.05h | P0 | Muscle | Build |
| T3.1 | 0.05h | P0 | Muscle | Grep gate |
| T4.1 | 0.3h | P0 | Muscle | AC-1+2 |
| T4.2 | 0.2h | P1 | Muscle | AC-7 |
| T4.3 | 0.3h | P2 | Muscle | AC-8 |
| T4.4 | 0.2h | P0 | Muscle | CI green |
| T5.1 | 0.1h | P0 | Muscle+User | Backup |
| T5.2 | 0.2h | P0 | Muscle | Staging OK |
| T5.3 | 0.1h | P0 | Muscle+User | Prod migration |
| T5.4 | 0.2h | P0 | Muscle | Rollout |
| T5.5 | 0.2h | P0 | Muscle | 0 errors |
| T6.1 | 24h elapsed | P1 | Muscle | Metrics |
| T6.2 | 0.2h | P0 | Brain | Lesson |
| T6.3 | 0.05h | P0 | Brain+User | Sign-off |
| T6.4 | 0.05h | P0 | Brain | Status |

**Total active effort:** ~4h (excluding 24h monitoring window)
