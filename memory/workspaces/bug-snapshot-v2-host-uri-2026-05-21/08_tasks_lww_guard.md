# 08_tasks_lww_guard — Task Checklist

> **Phase**: `lww_guard`
> **Owner Muscle**: claude-sonnet-4-6 (default) hoặc claude-opus-4-7 (nếu user override)
> **Execution order**: TUẦN TỰ (M1 → M8)

---

## M1 — Source code: cdcCols + DDL + OCC guard

- [x] **T1.1** Đọc `schema_adapter.go:195-249` để verify line numbers chính xác (có thể đã shift do edit khác).
- [x] **T1.2** Edit `cdcCols` map (line ~195-204): thêm `"_source_ts": "BIGINT"` SAU `"_source"`. Tham chiếu `09_tasks_solution_lww_guard.md` Edit #1.
- [x] **T1.3** Edit `createShadowTableV1WithCols` DDL inline (line ~314-325): thêm dòng `"_source_ts" BIGINT,` SAU `"_source" VARCHAR(20) DEFAULT 'airbyte',`. Tham chiếu Edit #2.
- [x] **T1.4** Edit `BuildUpsertSQLInSchema` WHERE clause (line ~513-518): đổi `<=` → `<` + thêm tiebreaker branch. Tham chiếu Edit #3.
- [x] **T1.5** `go build ./...` PASS, `go vet ./...` PASS.
- [x] **T1.6** APPEND `05_progress.md` "M1 done — schema_adapter.go edited at <commit/diff>".

## M2 — Migration

- [x] **T2.1** Tạo file `data-hub/cdc-cms-service/migrations/schema/core/060_v1_add_source_ts_to_shadow.sql` theo Migration #1 trong 09_tasks_solution.
- [x] **T2.2** Tạo file `data-hub/cdc-cms-service/migrations/schema/core/061_v1_snapshot_progress_cluster_time.sql` theo Migration #2.
- [x] **T2.3** Verify migration apply LOCAL: `cd cdc-cms-service && make migrate-up` (HOẶC migration runner thực tế của project — đọc Makefile để biết).
- [x] **T2.4** SQL verify column tồn tại (Gate 6 trong 09_tasks_solution checklist).
- [x] **T2.5** APPEND `05_progress.md` "M2 done — migration 060 + 061 applied locally, verify SQL count = X tables had _source_ts added".

## M3 — Mongo clusterTime capture

- [x] **T3.1** Verify Mongo driver version + API: `grep "go.mongodb.org/mongo-driver" data-hub/centralized-data-service/go.mod`.
- [x] **T3.2** Thêm helper `captureClusterTime` (Edit #5) ở file phù hợp (likely `snapshot_runner_handler.go` cùng package).
- [x] **T3.3** Update `buildSnapshotEnvelope` signature thêm param `clusterTimeMs int64` (Edit #4). Update all callers.
- [x] **T3.4** Tích hợp `captureClusterTime` vào snapshot start path: gọi 1 lần trước loop chunk, lưu vào local var + truyền vào envelope.
- [x] **T3.5** Bind `mongo_cluster_time_start_ms` + `mongo_cluster_time_capture_method` vào `snapshot_progress` row (migration 061 đã tạo column).
- [x] **T3.6** `go build` + `go vet` PASS.
- [x] **T3.7** Unit test `TestCaptureClusterTime` (mock Mongo client, verify fallback chain).
- [x] **T3.8** APPEND `05_progress.md` "M3 done — clusterTime capture wired, fallback chain tested".

## M4 — `_source` discriminator wiring

- [x] **T4.1** AUDIT trước khi edit: `grep -n "_source\"\|record.Source\|envelope.*source" data-hub/centralized-data-service/internal/handler/*.go internal/service/batch_buffer.go`.
- [x] **T4.2** Trace: envelope `source` field → record map → `BuildUpsertSQLInSchema(source=...)`.
- [x] **T4.3** Nếu pipeline đã pass envelope source qua → snapshot envelope đã `"source":"snapshot:v2"` → check downstream có dùng đúng KHÔNG.
- [x] **T4.4** Nếu KHÔNG đúng: edit minimal — parse `envelope.source` → set `record.Source` → pass vào `BuildUpsertSQLInSchema`.
- [x] **T4.5** `go build` + `go vet` PASS.
- [x] **T4.6** APPEND `05_progress.md` "M4 done — _source value end-to-end verified, trace: <file:line chain>".

## M5 — Test

- [x] **T5.1** Append unit tests vào `schema_adapter_test.go` (tham chiếu Edit #6 trong 09_tasks_solution: 2 test function).
- [x] **T5.2** Run `go test -run 'TestBuildUpsertSQLInSchema' -v` → PASS.
- [x] **T5.3** Run full suite `go test ./internal/service/... ./internal/handler/...` → no NEW regression (pre-existing fail OK, ghi rõ trong report).
- [x] **T5.4** Coverage: `go test -cover ./internal/service/` → log coverage % cho `schema_adapter.go`.
- [x] **T5.5** APPEND `05_progress.md` "M5 done — N tests PASS, coverage X%, pre-existing fails: <list>".

## M6 — Race smoke test

- [ ] **T6.1** Worker restart: K8s rollout HOẶC local `go run cmd/worker/main.go` (xem 09_tasks_solution Gate 7).
- [ ] **T6.2** Coordinate user trigger snapshot.v2 cho `source_object_id=18` qua FE.
- [ ] **T6.3** Trong lúc snapshot chạy: manual insert/update 1 record qua Mongo shell HOẶC ứng dụng:
      ```
      mongo "mongodb://10.200.187.11:27017/" --eval 'db.getSiblingDB("goopay_pbs").<collection>.updateOne({_id: ObjectId("<id>")}, {$set: {test_field: "race_marker_<timestamp>"}})'
      ```
- [ ] **T6.4** Sau snapshot done, SQL verify:
      ```sql
      SELECT _id, _source, _source_ts, test_field, _synced_at
      FROM cdc_internal.<shadow_table>
      WHERE _id = '<test_record_id>';
      ```
      Expect: `_source = 'debezium-v125'`, `test_field = 'race_marker_<timestamp>'`.
- [ ] **T6.5** Negative test: SQL verify snapshot data với row khác (không bị update bởi realtime):
      ```sql
      SELECT _source, _source_ts FROM cdc_internal.<shadow_table>
      WHERE _id = '<other_record>';
      ```
      Expect: `_source = 'snapshot:v2'`, `_source_ts = clusterTime` (gần thời điểm snapshot chạy).
- [ ] **T6.6** APPEND `05_progress.md` "M6 done — race test PASS, evidence: <SQL output snippet>".

## M7 — Security review

- [ ] **T7.1** Run `/security-agent` trên `data-hub/centralized-data-service/`.
- [ ] **T7.2** Address bất kỳ HIGH/CRITICAL finding (LOW/INFO note vào report).
- [ ] **T7.3** APPEND `05_progress.md` "M7 done — security review clean / N findings addressed".

## M8 — Report + lesson + workspace update

- [ ] **T8.1** Tạo `report_lww_guard_2026-05-21.md` ở workspace root. Tham chiếu template trong file `report_lww_guard_2026-05-21.md` đã pre-created.
- [ ] **T8.2** Fill report với SQL count thực tế, file changes thực tế, race test evidence.
- [ ] **T8.3** APPEND `agent/memory/global/lessons.md` Global Pattern về LWW + clusterTime + tiebreaker.
- [ ] **T8.4** Update `agent/memory/global/active_plans.md` workspace row: status sau khi done.
- [ ] **T8.5** APPEND `05_progress.md` "M8 done — phase lww_guard COMPLETE — report at <path>".
- [ ] **T8.6** Final pre-flight check (§14): scan tất cả file workspace, verify tồn tại vật lý.

---

## Skip conditions

- Nếu Mongo standalone không trả `clusterTime` (M3 T3.1 verify): fallback wall-clock + log WARN. M4 vẫn proceed. Race smoke M6 có thể chỉ PASS một phần.
- Nếu CI không có PG container: integration test (M5) skip. Smoke M6 vẫn yêu cầu thực tế.

## Escalation

- Stuck > 3 lần ở 1 task → dừng lại, APPEND `05_progress.md` "STUCK at T#.#, escalate Brain re-plan", chờ user verb.
