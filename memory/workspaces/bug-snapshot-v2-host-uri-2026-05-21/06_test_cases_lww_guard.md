# 06_test_cases_lww_guard — Test Plan

## Test Matrix

| TC | Layer | Scenario | Input | Expected Output |
|---|---|---|---|---|
| TC-U-01 | Unit | OCC guard: realtime ts mới > shadow ts cũ | shadow `_source_ts=100`, EXCLUDED `_source_ts=200`, `_source='debezium-v125'` | UPDATE applied, shadow `_source_ts=200` |
| TC-U-02 | Unit | OCC guard: snapshot ts cũ < shadow ts mới | shadow `_source_ts=200, _source='debezium-v125'`, EXCLUDED `_source_ts=100, _source='snapshot:v2'` | UPDATE skipped, shadow giữ nguyên |
| TC-U-03 | Unit | Tiebreaker: ts bằng, snapshot vs realtime | shadow `_source_ts=150, _source='snapshot:v2'`, EXCLUDED `_source_ts=150, _source='debezium-v125'` | UPDATE applied (realtime thắng) |
| TC-U-04 | Unit | Tiebreaker: ts bằng, realtime vs snapshot | shadow `_source_ts=150, _source='debezium-v125'`, EXCLUDED `_source_ts=150, _source='snapshot:v2'` | UPDATE skipped |
| TC-U-05 | Unit | Tiebreaker: ts bằng, snapshot vs snapshot | shadow `_source_ts=150, _source='snapshot:v2'`, EXCLUDED `_source_ts=150, _source='snapshot:v2'` | UPDATE skipped (no-op) |
| TC-U-06 | Unit | Tiebreaker: ts bằng, realtime vs realtime | shadow `_source_ts=150, _source='debezium-v125'`, EXCLUDED same | UPDATE skipped (no-op, idempotent) |
| TC-U-07 | Unit | Row legacy: shadow `_source_ts=NULL` | shadow `_source_ts=NULL`, EXCLUDED `_source_ts=100` | UPDATE applied (IS NULL branch) |
| TC-U-08 | Unit | Fallback hash dedup khi `sourceTsMs=0` | EXCLUDED `_source_ts` placeholder NULL | WHERE clause dùng `_hash IS DISTINCT FROM` |
| TC-U-09 | Unit | Schema thiếu `_source` col | Columns không có `_source`, có `_source_ts` | WHERE clause chỉ guard theo ts (no tiebreaker reference) |
| TC-U-10 | Unit | Schema thiếu cả `_source` và `_source_ts` (legacy) | Columns không có cả 2 | WHERE clause fall hash dedup |

| TC | Layer | Scenario | Input | Expected Output |
|---|---|---|---|---|
| TC-I-01 | Integration (PG) | Migration 060 apply trên DB sạch | empty `cdc_internal.test_t` | Column `_source_ts BIGINT NULL` thêm thành công |
| TC-I-02 | Integration (PG) | Migration 060 idempotent re-run | apply 060 2 lần | Lần 2 no-op, không error |
| TC-I-03 | Integration (PG) | UPSERT race: realtime then snapshot | row R: INSERT ts=100, snapshot ts=50 | shadow giữ realtime data, `_source_ts=100` |
| TC-I-04 | Integration (PG) | UPSERT race: snapshot then realtime | row R: INSERT snapshot ts=100, realtime ts=200 | shadow update theo realtime, `_source_ts=200` |
| TC-I-05 | Integration (PG) | UPSERT race: cùng ts, snapshot before realtime | INSERT snapshot ts=100; UPSERT realtime ts=100 | shadow `_source='debezium-v125'` |
| TC-I-06 | Integration (PG) | UPSERT race: cùng ts, realtime before snapshot | INSERT realtime ts=100; UPSERT snapshot ts=100 | shadow `_source='debezium-v125'` (giữ nguyên) |

| TC | Layer | Scenario | Steps | Pass criteria |
|---|---|---|---|---|
| TC-E-01 | E2E smoke | Snapshot v2 trên `source_object_id=18` cluster live | 1. Worker restart sau build. 2. FE trigger snapshot. 3. Wait done. 4. SQL count. | `SELECT count(*) FROM cdc_internal.<shadow> WHERE _source='snapshot:v2'` > 0 |
| TC-E-02 | E2E smoke | Race: snapshot đang chạy + realtime update | 1. Trigger snapshot (chunk 1000 row, ETA 30s). 2. Sau 10s: manual `db.collection.updateOne({_id: X}, {$set: {f: 'race_marker'}})`. 3. Wait snapshot done. 4. SQL verify row X. | Row X: `_source='debezium-v125'`, `f='race_marker'`, `_source_ts = oplog ts > clusterTime snapshot start` |
| TC-E-03 | E2E smoke | Clock skew simulation | Set worker clock +1h, trigger snapshot, đồng thời realtime event với ts thật | shadow giữ realtime data (vì clusterTime của Mongo không bị skew) |
| TC-E-04 | E2E smoke | Mongo standalone fallback | Trigger snapshot trên Mongo standalone (no replica set) | Log có `snapshot.v2 clusterTime capture fallback`, snapshot vẫn complete, shadow data có `_source_ts = wall_clock` |

## Test Data Setup

### Mongo source (cluster `goopay-pbs`, registry_id=18)

```js
// Connect: mongodb://10.200.187.11:27017,10.200.187.12:27017,10.200.187.13:27017/?replicaSet=goopay&authSource=admin
// DB: goopay-pbs (hoặc tương ứng connection)

// Setup 3 test record cho TC-E-02
db.test_collection.insertMany([
    { _id: ObjectId("000000000000000000000001"), name: "record_A", v: 0 },
    { _id: ObjectId("000000000000000000000002"), name: "record_B", v: 0 },
    { _id: ObjectId("000000000000000000000003"), name: "record_C", v: 0 },
]);
```

### PG verify queries

```sql
-- Sau snapshot run + race update
SELECT _id, _source, _source_ts, name, v
FROM cdc_internal.test_collection
WHERE _id IN (
    '000000000000000000000001',
    '000000000000000000000002',
    '000000000000000000000003'
)
ORDER BY _id;

-- Distribution check
SELECT _source, COUNT(*) FROM cdc_internal.test_collection GROUP BY _source;
-- Expect: cả 'snapshot:v2' và 'debezium-v125' (mix tuỳ race scenario).
```

## Pass / Fail criteria tổng

| Criterion | Pass | Fail |
|---|---|---|
| Unit test PASS rate | 10/10 | <10/10 |
| Integration test PASS rate | 6/6 | <6/6 |
| E2E smoke | TC-E-01 + TC-E-02 PASS bắt buộc; TC-E-03 + TC-E-04 SHOULD | Bất kỳ MUST PASS test fail |
| Build/Vet | EXIT 0 cho cả 2 | EXIT non-zero |
| Pre-existing test regression | KHÔNG có test mới fail | Test trước PASS, giờ fail |
| Migration apply | clean, < 5s per table | Lock > 30s hoặc error |

## Test artifacts

Lưu vào `/tmp/lww_guard_*.log` rồi attach evidence vào `report_lww_guard_2026-05-21.md`:
- `/tmp/lww_guard_build.log`
- `/tmp/lww_guard_vet.log`
- `/tmp/lww_guard_test.log`
- `/tmp/lww_guard_full_test.log`
- `/tmp/lww_guard_migrate.log`
- `/tmp/lww_guard_race_smoke.log`
- `/tmp/lww_guard_security.log`
