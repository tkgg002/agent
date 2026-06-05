# 02_plan.md — Execution Plan

> Tham chiếu `01_requirements.md` (FR/NFR/AC) + `04_decisions.md` (ADR).
> Plan này dành cho Muscle thực thi sau khi User approve.

---

## Tổng quan

- **Mục tiêu:** Fix Contract Drift `_gpay_id` qua 1 migration SQL + 1 dòng Go DDL + 1 comment.
- **Tổng effort ước tính:** 2-4 giờ (1 dev senior).
- **Risk:** LOW (patch tối thiểu, idempotent, defense-in-depth).
- **Rollback:** DROP FUNCTION + ALTER TABLE DROP DEFAULT (idempotent reverse).

---

## Phase plan

| Phase | Mục tiêu | Effort | Gate |
|---|---|---|---|
| **P0** | Reproduce bug local (testcontainers) | 0.5h | Test FAIL như expect |
| **P1** | Migration `019_sonyflake_default_fill.sql` | 1h | psql apply OK, idempotent OK |
| **P2** | Go DDL fix `schema_manager.go` | 0.3h | Unit test pass |
| **P3** | Comment fix `batch_buffer.go` | 0.1h | grep gate pass |
| **P4** | Integration test mới | 1h | AC-1 → AC-4 pass |
| **P5** | Deploy migration prod + restart sink | 0.5h | `\d+` show DEFAULT, batch_upsert log clean |
| **P6** | Post-deploy verify + lesson append | 0.5h | Metrics stable 24h |

**Total: ~4h** (không tính deploy window prod).

---

## Dependency graph

```
P0 (reproduce) ─┐
                ├─→ P1 (migration) ─→ P4 (integration test) ─→ P5 (deploy) ─→ P6 (verify)
P3 (comment) ──┐│
P2 (Go DDL)  ──┴┘
```

P2, P3 có thể parallel với P1 (cùng file domain).
P4 cần P1+P2+P3 done.

---

## Phase chi tiết

### P0 — Reproduce bug local
**Goal:** Có repro chắc chắn trước khi sửa (lesson "verify before claim done").

**Steps:**
1. Spawn testcontainers Postgres
2. Chạy migrations đến `018_sonyflake_v125_foundation.sql`
3. CREATE V2 shadow table `tokens_v2_test` đúng layout production (gồm `_gpay_id BIGINT PK`)
4. Set session vars `app.fencing_machine_id=1, app.fencing_token=...`
5. `INSERT INTO tokens_v2_test (_source_id, _raw_data) VALUES ('src-1', '{}'::jsonb)`
6. Expect: `ERROR: null value in column "_gpay_id" ... (SQLSTATE 23502)`

**DoD:** Test `TestReproduceGpayIDNullBug` FAIL với đúng SQLSTATE 23502.

---

### P1 — Migration `019_sonyflake_default_fill.sql`

**Goal:** Tạo `cdc_internal.sf_nextval()` + ALTER tất cả V2 shadow tables existing.

**Steps:**
1. Tạo file `data-hub/cdc-cms-service/migrations/schema/ids/019_sonyflake_default_fill.sql`
2. Implement `cdc_internal.sf_nextval()` function (xem `03_implementation.md` §1)
3. `DO $$ ... $$` block iterate `information_schema.tables` WHERE table có column `_gpay_id` AND `atthasdef = false` → `ALTER TABLE SET DEFAULT`
4. Đảm bảo idempotent: `CREATE OR REPLACE FUNCTION`, `ALTER TABLE` luôn safe
5. Test local: chạy 2 lần liên tiếp → không lỗi
6. Verify: `SELECT atthasdef, ... FROM pg_attribute WHERE attname='_gpay_id'`

**DoD:** AC-3 + AC-6 pass.

---

### P2 — Go DDL fix `schema_manager.go`

**Goal:** Mọi V2 shadow table tạo bởi Go runtime từ giờ trở đi có DEFAULT.

**Steps:**
1. Edit `data-hub/centralized-data-service/internal/sinkworker/schema_manager.go:226`
2. Đổi:
   ```go
   `"_gpay_id" BIGINT PRIMARY KEY`,
   ```
   thành:
   ```go
   `"_gpay_id" BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval()`,
   ```
3. Build + unit test

**DoD:** Build pass; unit test `TestCreateShadowTable_HasDefault` pass.

---

### P3 — Comment fix `batch_buffer.go`

**Goal:** Comment khớp implementation thật.

**Steps:**
1. Edit `data-hub/centralized-data-service/internal/handler/batch_buffer.go:246-250`
2. Rewrite theo ADR-05 (xem `03_implementation.md` §3)
3. Add CI grep gate (optional, có thể skip)

**DoD:** AC-5 grep pass.

---

### P4 — Integration test mới

**Goal:** AC-1 → AC-4 + AC-7 + AC-8 pass.

**Steps:**
1. Tạo `data-hub/centralized-data-service/internal/handler/batch_buffer_v2shadow_test.go`
2. 3 test case:
   - `TestBatchUpsert_V2Shadow_NoExplicitGpayID` — happy path
   - `TestBatchUpsert_V2Shadow_5000Rows_Perf` — benchmark
   - `TestSonyflakeIDDecode` — verify ID decode đúng machine
3. Setup testcontainers chạy full migration chain

**DoD:** Tất cả pass 3 lần liên tiếp (non-flaky).

---

### P5 — Deploy prod

**Pre-check:**
- ✅ Backup DB snapshot
- ✅ Verify migration trên staging trước
- ✅ Communicate downtime (dù = 0 do metadata-only ALTER)

**Steps:**
1. Apply migration 019 qua `make migrate`
2. Verify: `\d+ data_hub.tokens` show `_gpay_id BIGINT NOT NULL DEFAULT cdc_internal.sf_nextval()`
3. Deploy Go binary mới (chứa P2 + P3)
4. Restart sink workers
5. Tail log: `kubectl logs -f deploy/centralized-data-service | grep "batch upsert"` — không còn `null value in column "_gpay_id"`

**Rollback (nếu fail):**
```sql
-- Reverse migration
ALTER TABLE data_hub.tokens ALTER COLUMN _gpay_id DROP DEFAULT;
-- ... repeat cho mỗi V2 shadow table
DROP FUNCTION cdc_internal.sf_nextval();
```
+ Redeploy Go binary cũ.

**DoD:** 30 phút sau deploy, 0 lỗi `_gpay_id`, throughput batch_upsert bình thường.

---

### P6 — Post-deploy verify + lesson

**Steps:**
1. Monitor 24h: throughput, error rate, latency p99
2. Append `agent/memory/global/lessons.md` lesson:
   > **Pattern [Code A claim contract X, Migration B claim contract Y, không nơi nào implement X+Y]** → Result: hidden bug bóc lộ khi thay đổi upstream caller. Đúng: Trước mỗi claim trong comment, grep verify function/trigger tồn tại; CI check comment-vs-symbol consistency.
3. Update `05_progress.md` final entry

**DoD:** Lesson abstract đúng format CLAUDE.md §13, áp dụng được ≥ 3 dự án khác.

---

## Risk → Mitigation

| Risk | Phase | Mitigation |
|---|---|---|
| R1 (ALTER lock prod) | P5 | Metadata-only ALTER (no rewrite); test trên staging trước |
| R2 (Sonyflake collision) | P1 | `sf_nextval()` dùng UNIQUE check + retry trong function |
| R4 (Comment lệch lần nữa) | P3 | grep gate CI optional; ADR-05 ghi rõ tham chiếu migration |
| R5 (Local dev drift) | P5 | Doc step: restart Go sink để `createShadowTable` re-eval |

---

## Stop Rule (CLAUDE.md §8)

Nếu Muscle fail > 3 lần ở bất kỳ phase:
1. **STOP** ngay
2. Append `05_progress.md` entry ESCALATE
3. Notify Brain → re-plan

---

## Gate verification script

```bash
# Gate G1 (sau P1)
psql -c "\df cdc_internal.sf_nextval"
psql -c "\d+ data_hub.tokens" | grep "DEFAULT"

# Gate G2 (sau P2)
go build ./internal/sinkworker/...
go test ./internal/sinkworker/... -run TestCreateShadowTable -count=3

# Gate G3 (sau P3)
grep -c "cdc_internal.sf_nextval" internal/handler/batch_buffer.go  # ≥ 1

# Gate G4 (sau P4)
go test ./internal/handler/... -run TestBatchUpsert_V2Shadow -count=3 -race

# Gate G5 (sau P5, prod)
kubectl logs deploy/centralized-data-service --since=30m | grep -c "null value in column" # = 0
```

---

## Timeline

```
T+0h    P0 reproduce
T+0.5h  P1 migration  ┐
T+1.5h  P2 Go DDL     ├─ parallel
T+1.5h  P3 comment    ┘
T+1.8h  P4 integration test
T+2.8h  P5 deploy (window quy ước)
T+3.3h  P6 verify
        DONE
```

---

## Communication protocol

- **Trước P5 (deploy):** Muscle phải xin User confirm.
- **Sau P5:** Muscle báo log + metrics, User sign-off trong `05_progress.md`.
- **Nếu rollback:** STOP, escalate Brain.
