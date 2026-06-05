# 04_decisions_lww_guard — Architecture Decision Record

## ADR-001 — Phương án LWW Guard: chọn D (full backport + clusterTime + discriminator)

### Context

4 phương án đã đề xuất ở chat audit:

| ID | Phương án | Effort | Coverage | Đụng schema |
|---|---|---|---|---|
| A | Force `_source='snapshot'` discriminator | Low | Chỉ chặn snapshot-clobber-realtime cùng ms, KHÔNG fix race wall-clock | Không |
| B | Sống dậy `_version` thành Lamport hack | Medium | Hack, dễ hiểu sai semantic của `_version` | Không |
| C | Backport `_source_ts` xuống V1 cdcCols + Mongo `clusterTime` | Medium | Fix race chính, nhưng cùng ms vẫn non-deterministic | Migration ADD COLUMN |
| D | C + discriminator `_source` tiebreaker | Medium+ | Fix toàn diện: oplog ts + tiebreaker cùng ms | Migration ADD COLUMN + code 1 WHERE clause |

### Decision

**Chọn D**.

### Rationale

1. **R1-R4 yêu cầu strong eventual consistency** — A/B không đạt được. C đạt 95% case nhưng cùng ms race vẫn non-deterministic.
2. **Effort delta D vs C nhỏ**: chỉ thêm 1 condition trong `WHERE` clause của `BuildUpsertSQLInSchema` + 1 chỗ set `_source` ở snapshot path. Cost ≈ +30 phút.
3. **Field `_source` đã tồn tại** (`schema_adapter.go:197`, VARCHAR(20) DEFAULT 'airbyte') — chỉ chưa được khai thác làm discriminator. Tận dụng infra có sẵn = đẹp hơn thêm cột mới.
4. **Tuân thủ L-OCC-preserve**: giữ `_source_ts` semantic (đã có ở V2/sinkworker), không thay bằng COALESCE ad-hoc. Backport sang V1 = đồng bộ V1↔V2.
5. **Tuân thủ L-V2-anchor**: khi đổi schema gen, port ĐỦ logic. Migration ADD COLUMN + cdcCols map + DDL inline phải nhất quán — đã liệt kê trong `09_tasks_solution_lww_guard.md`.

### Consequences

✅ **Positive**:
- Shadow V1 và V2 đồng bộ về OCC pattern.
- Race condition cùng ms được giải quyết.
- Trace evidence trong shadow (`_source` + `_source_ts`) — debug dễ hơn.
- Migration forward-compat (column NULL cho row cũ, guard `IS NULL` đã handle).

⚠️ **Negative / trade-off**:
- 1 column extra (8 bytes BIGINT/row). Với 50M row → ~400MB extra storage. Chấp nhận.
- WHERE clause dài hơn → planner overhead ~negligible (constant string compare).
- Cần Mongo replica set để có `clusterTime`. Mongo standalone → fallback chain.

### Alternatives rejected

- **A (chỉ discriminator)**: không fix wall-clock race chính. Chỉ patch triệu chứng.
- **B (`_version` Lamport)**: vi phạm SRP — `_version` semantic hiện tại là row revision counter, không phải logical clock. Hack-y.
- **Trigger DB-side fill `_source_ts`**: vi phạm L-cheat-DB direction — Go pass sai ID, không FORCE DB. Source-of-truth là application code.

---

## ADR-002 — Mongo `clusterTime` capture method

### Context

Snapshot envelope cần oplog timestamp thật. Có 3 cách lấy:

| Method | API | Pros | Cons |
|---|---|---|---|
| **db.hello()** | `Database.RunCommand(ctx, bson.D{{"hello", 1}})`, đọc `$clusterTime.clusterTime` | Read-only, fast, standard | Cần replica set; standalone Mongo không trả `$clusterTime` |
| **replSetGetStatus** | `Database.RunCommand(ctx, bson.D{{"replSetGetStatus", 1}})`, đọc `optimes.appliedOpTime.ts` | Trả oplog ts chính xác | Require `clusterMonitor` role; verbose response |
| **Read oplog.rs last entry** | `local.oplog.rs.find().sort({ts:-1}).limit(1)` | Trả ts thực tế cuối oplog | Require read trên `local` DB; permission khó |

### Decision

**Primary**: `db.hello()` → `$clusterTime.clusterTime` (BSON Timestamp `t,i`).
**Fallback chain**: `db.hello() fail/missing` → `replSetGetStatus` → `time.Now()` + log WARN + skip M4 enforcement.

### Rationale

- `db.hello()` là API recommended bởi Mongo, low-overhead, accessible với basic read role.
- `clusterTime` là logical clock của replica set — chính xác hơn wall-clock của bất kỳ node nào.
- Fallback chain đảm bảo snapshot không bị block hoàn toàn nếu cluster đặc biệt (standalone test).

### Code conversion

```go
// BSON Timestamp {t: <secs>, i: <ordinal>} → ms epoch.
// Snapshot envelope dùng ms để khớp Debezium source.ts_ms.
clusterTimeMs := int64(ts.T) * 1000  // t là Unix seconds
// i (ordinal) dùng làm tiebreaker phụ nếu cần, nhưng phase này skip
```

---

## ADR-003 — Snapshot `_source` value canonical

### Context

Cần phân hoá rõ snapshot vs realtime ở column `_source` (VARCHAR(20)). Hiện tại các giá trị thực:

- `'airbyte'` (default DDL, legacy)
- `'debezium-v125'` (sinkworker V2 path)
- `'cdc-legacy'` (V1 legacy command_handler)
- `'debezium-transmute'` (transmuter)

### Decision

Snapshot v2 dùng `_source = 'snapshot:v2'` (16 ký tự, fit VARCHAR(20)).

### Rationale

- Khớp với `source` field trong CDCEvent envelope (`snapshot_runner_handler.go:503` đã dùng `"source":"snapshot:v2"`).
- Tiebreaker WHERE clause: `tbl._source = 'snapshot:v2' AND EXCLUDED._source <> 'snapshot:v2'` → realtime override snapshot.
- Format `<source>:<version>` future-proof: nếu sau này có snapshot v3 → `'snapshot:v3'`.

---

## ADR-004 — Migration strategy: ADD COLUMN + Backfill (KHÔNG cheat-DB)

### Context

Bảng shadow đã có data hiện trường. Cần thêm column `_source_ts BIGINT`.

### Decision

- **Source-of-truth**: `cdcCols` map (`schema_adapter.go:195-204`) + DDL inline (`createShadowTableV1WithCols:314-325`). Sửa code = mọi bảng mới có column.
- **Existing tables**: migration file `060_v1_add_source_ts_to_shadow.sql`:
  - `ALTER TABLE ... ADD COLUMN IF NOT EXISTS "_source_ts" BIGINT NULL` (metadata-only, instant trên PG ≥ 11).
  - Backfill: `UPDATE ... SET _source_ts = EXTRACT(EPOCH FROM _synced_at) * 1000 WHERE _source_ts IS NULL` — chỉ apply cho NULL.
- Migration file MUST be **forward-only** (KHÔNG có DROP COLUMN rollback inline).

### Anti-pattern explicitly avoided

❌ KHÔNG copy `ALTER TABLE ADD COLUMN` command vào `report_lww_guard_2026-05-21.md` làm "manual fix script". Lesson L-cheat-DB-ALTER-in-report cấm.

✅ Report chỉ document:
- File source thay đổi (`schema_adapter.go` line X-Y).
- Migration file mới (`060_*.sql` — đường dẫn relative).
- SQL verify count trước/sau (kết quả thực, không mock).
- Hướng dẫn user run migration qua **migration runner đã có** (cdc-cms-service migration init), KHÔNG hướng dẫn psql ALTER tay.

---

## ADR-005 — Test strategy: Unit + Integration + Race smoke

### Decision

3 lớp test:

1. **Unit test SQL generator** (fast, deterministic):
   - File mới hoặc append `internal/service/schema_adapter_test.go`.
   - Cover 4 case WHERE clause: ts mới>cũ, ts cũ<mới, ts bằng+snapshot vs realtime, ts bằng+realtime vs realtime.
   - Snapshot test SQL string (golden file pattern).

2. **Integration test với PG container** (medium, real DB):
   - Spin up PG, apply migration 060, insert 2 row race scenario, verify final state.
   - Skip nếu CI không có Docker — chạy local hoặc nightly.

3. **Race smoke test** (manual, real cluster):
   - Coordination với user: trigger snapshot trên `source_object_id=18`, đồng thời update 1 record Mongo manual, verify shadow giữ realtime data.
   - Log evidence + SQL count → report.

### Rationale

- Unit test đảm bảo SQL shape đúng (cheap, fast feedback).
- Integration test đảm bảo behavior thực với PG (catch type cast, index, constraint issue).
- Race smoke test đảm bảo end-to-end pipeline (catch BatchBuffer timing, Kafka lag, NATS publish drift).
