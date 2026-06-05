# 01_requirements.md — Functional / Non-functional / AC

> Tham chiếu `00_context.md` cho bối cảnh bug Contract Drift `_gpay_id`.

---

## 1. Functional Requirements

| ID | Requirement | Layer | Acceptance signal |
|---|---|---|---|
| **FR-1** | `_gpay_id` PHẢI được auto-fill khi INSERT mọi V2 shadow table | DB + Go | INSERT không chỉ định `_gpay_id` → row có giá trị unique BIGINT |
| **FR-2** | Comment trong `batch_buffer.go:246-250` PHẢI khớp implementation thực tế | Doc | grep "trigger fills" → reference đúng function tồn tại HOẶC bỏ claim |
| **FR-3** | DDL trong `schema_manager.go:createShadowTable` PHẢI tạo cột `_gpay_id` có cơ chế auto-fill | Go runtime | `pg_dump` table show `DEFAULT` clause HOẶC trigger fill |
| **FR-4** | Migration mới PHẢI heal tất cả existing V2 shadow tables (NULL DEFAULT) | SQL | Sau migration, mọi V2 shadow table có DEFAULT/trigger fill |
| **FR-5** | Fix PHẢI idempotent — chạy lại migration không lỗi | SQL | `psql -f migration.sql` lần 2 không raise exception |
| **FR-6** | Giải pháp PHẢI giữ Sonyflake ID semantic (BIGINT, k8s-aware machine ID, monotonic) | Cross | Generated ID decode được qua sonyflake decomposer |
| **FR-7** | Bug bị reproduce trong CI test trước fix — pass sau fix | Test | Integration test `TestBatchUpsert_V2Shadow_NoExplicitGpayID` |

---

## 2. Non-functional Requirements

| ID | Requirement | Rationale |
|---|---|---|
| **NFR-1** | KHÔNG đổi business logic (`_source_id` UNIQUE, ON CONFLICT, fencing) | Tránh regression fix vòng trước |
| **NFR-2** | KHÔNG đổi tableV1 path (per-table sequence DEFAULT giữ nguyên) | Out-of-scope, migration 003 stable |
| **NFR-3** | KHÔNG dùng UUID / Snowflake / serial — giữ Sonyflake | Design intent v1.25 |
| **NFR-4** | Migration phải chạy được trên prod table có data hiện hữu (no rewrite) | Downtime tối thiểu |
| **NFR-5** | Latency UPSERT batch không tăng > 5% sau fix | Hot path, perf-sensitive |
| **NFR-6** | KHÔNG thêm dependency mới (giữ pkgs/idgen hiện có) | Anti-over-abstraction |
| **NFR-7** | Patch tối thiểu — chỉ chạm file cần thiết | Lesson "Simplicity First" CLAUDE.md §6 |
| **NFR-8** | Defense-in-depth: 2 lớp fill (Go-side + DB-side fallback) | Tránh tái diễn nếu 1 lớp bị bypass |

---

## 3. Acceptance Criteria

### AC-1: Reproduce bug trước fix
```bash
# Setup: fresh DB, chạy migrations cũ (chưa apply fix)
psql -c "CREATE SCHEMA test_v2; CREATE TABLE test_v2.tokens (_gpay_id BIGINT PRIMARY KEY, _source_id TEXT NOT NULL, ...);"
# INSERT giống batch_upsert builder
psql -c "INSERT INTO test_v2.tokens (_source_id, _raw_data) VALUES ('src-1', '{}');"
# → ERROR: null value in column "_gpay_id"
```
**Expect:** lỗi `23502` REPRODUCED.

### AC-2: Fix pass cùng kịch bản
Sau apply migration + Go fix:
```bash
psql -c "INSERT INTO test_v2.tokens (_source_id, _raw_data) VALUES ('src-1', '{}');"
psql -c "SELECT _gpay_id FROM test_v2.tokens WHERE _source_id = 'src-1';"
# → BIGINT non-null (sonyflake ID)
```

### AC-3: Heal existing prod table
- Trước migration: `pg_dump --schema-only` show `_gpay_id BIGINT PRIMARY KEY` (no DEFAULT).
- Sau migration: cùng table show `DEFAULT cdc_internal.sf_nextval()` HOẶC trigger AFTER ALTER.

### AC-4: Batch upsert end-to-end
```bash
go test ./internal/handler/... -run TestBatchUpsert_V2Shadow_NoExplicitGpayID -count=3
```
- Trước fix: FAIL với `null value in column "_gpay_id"`.
- Sau fix: PASS 3/3 lần, không flaky.

### AC-5: Comment ↔ Code consistency
```bash
grep -n "sonyflake trigger fills" internal/handler/batch_buffer.go
# → comment update reflect đúng (chỉ trigger DDL function nào, file nào)
```

### AC-6: Idempotent migration
```bash
psql -f migrations/schema/ids/019_sonyflake_default_fill.sql  # lần 1
psql -f migrations/schema/ids/019_sonyflake_default_fill.sql  # lần 2 — không lỗi
```

### AC-7: Sonyflake decode
```go
id := <fetched _gpay_id>
decoded := sonyflake.Decompose(id)
assert.NotZero(decoded["machine-id"])
assert.NotZero(decoded["time"])
```

### AC-8: Perf regression check
Benchmark `BenchmarkBatchUpsert_5000` — delta latency ≤ +5% so với baseline trước fix.

---

## 4. Out-of-scope (re-confirm từ 00_context §6)

| Out | Lý do |
|---|---|
| Thay đổi `_source_id` UNIQUE / ON CONFLICT | Fix trước đã xử |
| Migrate sang Snowflake / UUID | Giữ Sonyflake intent |
| Refactor toàn bộ `BatchBuffer` | Out-of-scope, patch tối thiểu |
| Đụng tableV1 (003 migration) | V1 path stable |
| Hexagonal refactor | Track riêng workspace v2 |

---

## 5. Risk Register

| ID | Risk | Severity | Mitigation |
|---|---|---|---|
| R1 | Migration ALTER TABLE lock production table | HIGH | Dùng `ALTER TABLE ... SET DEFAULT` (metadata-only lock, fast) thay vì rewrite |
| R2 | Sonyflake collision khi 2 pod cùng machine ID | MED | Defense-in-depth: DB-side `sf_nextval()` có UNIQUE check + retry |
| R3 | Trigger fill conflict với fencing guard | LOW | Cùng function chain: fencing check trước, fill `_gpay_id` sau (cùng BEFORE INSERT) |
| R4 | Comment fix lệch lần nữa nếu sửa Go code không sửa comment | MED | AC-5 grep gate trong CI |
| R5 | Local dev sau fix vẫn dùng schema cũ → flaky local | MED | Migration up + restart Go sink để createShadowTable re-eval |
| R6 | pgx.Batch không hỗ trợ `RETURNING _gpay_id` style → app code không lấy được ID | LOW | Không cần RETURNING — DB fill đủ |
| R7 | Migration `pgcrypto`/`extension` dependency thiếu | LOW | Sonyflake không cần extension — pure plpgsql |

---

## 6. Definition of Done (toàn workspace)

- [ ] Tất cả 7 AC pass
- [ ] 3 layer (comment + DDL + migration) align về 1 implementation
- [ ] CI integration test green
- [ ] Prod DB sau migration: `\d+ <v2_table>` show DEFAULT cho `_gpay_id`
- [ ] `05_progress.md` log đầy đủ Brain plan + Muscle execute
- [ ] User sign-off trong `05_progress.md`
