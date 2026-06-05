# 01_requirements_lww_guard — LWW Guard cho race Snapshot v2 ↔ Realtime

> **Phase**: `lww_guard` (extension của workspace `bug-snapshot-v2-host-uri-2026-05-21`)
> **Author**: Brain (claude-opus-4-7)
> **Date**: 2026-05-21
> **Status**: 📋 Plan — chờ user approve để Muscle execute

---

## 1. Background

Sau khi snapshot.v2 DSN-resolver fix (workspace phase trước), 2 luồng ghi vào shadow chạy **song song**:

- **Luồng Realtime**: `Mongo oplog → Debezium → Kafka → kafka_consumer.go → EventHandler.HandleRaw → BatchBuffer.batchUpsert → schema_adapter.BuildUpsertSQLInSchema → PG shadow`.
- **Luồng Snapshot v2** (Path B custom worker): `Mongo find() chunk → snapshot_runner_handler.buildSnapshotEnvelope → EventHandler.HandleRaw → ... → PG shadow` (cùng pipeline, khác entry point).

Audit phát hiện 2 luồng race **không có hàng rào đủ mạnh** trên bảng shadow V1 hiện hành:

| Lỗ hổng | File:line | Hệ quả |
|---|---|---|
| `cdcCols` V1 thiếu `_source_ts BIGINT` | `centralized-data-service/internal/service/schema_adapter.go:195-204` | OCC guard `WHERE _source_ts <= EXCLUDED._source_ts` (line 514-518) **skipped** vì `hasSourceTs=false`. Bảng V1 fall through, mọi upsert ghi đè vô điều kiện. |
| Snapshot envelope dùng `time.Now().UnixMilli()` thay vì Mongo `clusterTime` | `snapshot_runner_handler.go:506` (`buildSnapshotEnvelope`) | Wall clock của worker > oplog ts thật → snapshot **clobber** realtime data mới hơn. |
| Khi `_source_ts` bằng nhau ms, OCC `<=` cho phép ghi đè không thứ tự | `schema_adapter.go:516` | Race nội-luồng và race chéo-luồng cùng ms → kết quả non-deterministic. |
| `_source` column không phân hoá snapshot vs realtime (cùng giá trị `'debezium-v125'` / default `'airbyte'`) | `schema_adapter.go:197` default + caller chains | Không có discriminator để tiebreak khi ts bằng nhau. |

## 2. Functional Requirements

**R1 — Strong Eventual Consistency**: với mọi record có cùng `_id` (PK source), trạng thái cuối cùng trong shadow PHẢI khớp với event mới nhất theo **oplog timestamp của source store** (không phải wall clock của worker).

**R2 — Snapshot không bao giờ clobber realtime**: nếu một record có realtime event ts = T1, và snapshot quét được record đó tại thời điểm worker clock T2 (T2 > T1 do clock skew hoặc snapshot chạy sau), shadow PHẢI giữ realtime data, KHÔNG ghi snapshot data lên.

**R3 — Realtime out-of-order vẫn dùng LWW theo oplog ts**: 2 event realtime cùng record, ts1 < ts2, dù tới shadow theo thứ tự nào, cuối cùng shadow PHẢI giữ data của ts2.

**R4 — Tie-breaker khi `_source_ts` bằng nhau** (cùng millisecond): realtime LUÔN thắng snapshot. Realtime vs realtime: chấp nhận order-of-arrival (rare, không materially nguy hiểm).

**R5 — Backward compatibility**: bảng shadow V1 đã có data hiện trường PHẢI vẫn upsert được sau migration (không break runtime trong lúc deploy).

**R6 — Backfill cho row cũ**: row đã có sẵn trong shadow trước migration có `_source_ts = NULL` → guard `IS NULL OR <=` cho phép update lần đầu (đã sẵn ở `schema_adapter.go:516`).

## 3. Non-Functional Requirements

**N1 — Core systems direction**: KHÔNG cheat DB bằng `ALTER ADD COLUMN` ad-hoc trong report (vi phạm L-cheat-DB-ALTER-in-report). Sửa SOURCE: `cdcCols` map + `createShadowTableV1WithCols` DDL. Migration file là source-of-truth cho prod DB. Local DB align bằng drop-replay hoặc apply migration đúng quy trình.

**N2 — CDC golden rule (read-only source)**: KHÔNG ghi gì vào source Mongo. `db.hello()` (đọc clusterTime) là read-only command → OK. KHÔNG dùng `signal.data.collection` hay tương đương.

**N3 — Single source of truth**: chỉ 1 chỗ define column list (`cdcCols` map). DDL inline + migration ADD COLUMN cùng tham chiếu chung manifest. KHÔNG fork pipeline.

**N4 — Idempotent migration**: `IF NOT EXISTS` ở DDL, backfill `_source_ts` từ `_synced_at` chỉ apply cho row NULL — re-run an toàn.

**N5 — Trace evidence**: mỗi row shadow phải có thể answer "data này từ snapshot hay realtime, oplog ts khi nào" qua `_source` + `_source_ts`.

## 4. Out of Scope

- ❌ KHÔNG đụng V2 sinkworker path (`internal/sinkworker/*`) — đã có guard riêng, scope sau.
- ❌ KHÔNG refactor BatchBuffer / EventHandler shape.
- ❌ KHÔNG đụng master layer (transmute / OCC ở master) — fix shadow trước.
- ❌ KHÔNG tự ý archive `cdc-system/` stale tree (defer; lesson L-dual-tree-drift).
- ❌ KHÔNG fix config reload storm + topic typo `centrallized` (defer phase khác).

## 5. Definition of Done (DoD)

| Gate | Acceptance |
|---|---|
| Code change | `cdcCols` có `_source_ts`, snapshot envelope dùng clusterTime, `BuildUpsertSQLInSchema` có tiebreaker `_source` discriminator |
| Migration | `060_v1_add_source_ts_to_shadow.sql` apply CLEAN trên DB metadata + DB shadow (`cdc-metadata` 5433) |
| Build gate | `go build ./...` PASS, `go vet ./...` PASS |
| Test gate | Unit test mới cho `BuildUpsertSQLInSchema` cover 4 case: (a) ts mới>cũ, (b) ts cũ<mới, (c) ts bằng + snapshot vs realtime, (d) ts bằng + realtime vs snapshot |
| Smoke gate | Restart worker (K8s rollout HOẶC `go run` local), trigger snapshot.v2 cho `source_object_id=18` (goopay-pbs); SQL verify `SELECT _source, _source_ts FROM cdc_internal.<shadow> ORDER BY _source_ts DESC LIMIT 10` có giá trị non-NULL + phân hoá đúng `snapshot:v2` vs `debezium-v125` |
| Race test | Inject 1 realtime event `op=u` với ts=T1 cho 1 record, sau đó force snapshot quét record đó. Verify shadow giữ realtime data (record `_source='debezium-v125'`, `_source_ts=T1`). |
| Report | `report_lww_guard_2026-05-21.md` ghi đủ: file thay đổi, SQL count verify trước/sau, race test result, rollback steps. Không chứa "manual ALTER" làm repair script. |
| Lesson | APPEND `agent/memory/global/lessons.md` Global Pattern về LWW + clusterTime + tiebreaker discriminator. |

## 6. Risks & Mitigation

| Risk | Likelihood | Mitigation |
|---|---|---|
| Mongo driver Go không expose `clusterTime` từ `db.hello()` | Medium | Verify lib `go.mongodb.org/mongo-driver` `Database.RunCommand(ctx, bson.D{{"hello", 1}})` trả `Result.Raw()` chứa `$clusterTime.clusterTime` (BSON Timestamp `t,i`). Fallback: dùng `replSetGetStatus` lấy `optimes.appliedOpTime.ts`. |
| Backfill `_source_ts = _synced_at*1000` cho row cũ tạo ts "giả" cao bất thường | Low | Acceptable: ts giả từ NOW của lần ingest đầu — vẫn < bất kỳ realtime ts mới nào (vì worker clock chưa drift). Nếu lo, có thể set `_source_ts = 0` để fall vào `IS NULL` branch của guard. |
| Migration ADD COLUMN trên bảng shadow lớn (>1M row) lock lâu | Medium | Postgres `ADD COLUMN` không default = metadata-only (instant). Backfill chạy `UPDATE WHERE _source_ts IS NULL` thành batch nếu row count > 100K. Migration file ghi rõ pattern này. |
| Realtime path đang set `_source` thành gì? Có override snapshot không? | High (cần verify) | Task L-2.5 audit: trace `_source` value qua `kafka_consumer → eventHandler → batchBuffer → schemaAdapter`. Nếu realtime đang set `'debezium-v125'` và snapshot không override → tiebreaker fail. PHẢI force snapshot path set `_source='snapshot:v2'` ở envelope hoặc record map. |

## 7. References

- Plan source code paths: `data-hub/centralized-data-service/...` (CORRECT tree, lesson L-dual-tree-drift).
- Lesson áp dụng: L-OCC-preserve (1037), L-V2-anchor (1744), L-cheat-DB-ALTER-in-report (2962), L-CDC-golden-rule (3671), L-Path-B-pattern (3706), L-dual-tree-drift (3809).
- Audit reports (chat history): xem `05_progress.md` Followup #5 + Followup #6.
