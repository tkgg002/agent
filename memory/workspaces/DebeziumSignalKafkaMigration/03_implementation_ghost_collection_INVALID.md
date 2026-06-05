# 03 Implementation — Ghost Collection (PHASE INVALIDATED)

## Status
**INVALID — DO NOT USE PATTERN.** Rolled back 2026-05-20.

## Why invalidated
- Source MongoDB trên prod = **read-only** (fintech + DBA policy).
- Debezium 2.5.4 (và cả 3.x) MongoDB incremental snapshot watermark BUỘC ghi vào source connection: `MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen` insert watermark doc qua MongoClient của connector → source.
- Ghost collection pattern (tạo empty collection `cdc_system.debezium_watermarks` trên source) chỉ work trên dev (gpay-mongo local) — KHÔNG deploy được lên prod cluster `10.200.187.x` vì connector role không có quyền write.

## Actions executed (then reverted)
1. ✅ Revert docker-compose 2.7.4 → 2.5.4 (KEEP — đây là state đúng).
2. ✅ Recreate kafka-connect — plugins re-install OK (KEEP).
3. ❌ Created Mongo collection `cdc_system.debezium_watermarks` on gpay-mongo → **DROPPED** via `db.getSiblingDB('cdc_system').dropDatabase()`.
4. ❌ PATCH goopay-local config thêm `signal.data.collection` → **REVERTED** via PUT loại key này khỏi config. Confirmed `has("signal.data.collection") == false`.
5. ❌ Restart connector sau patch → KHÔNG verify state, đã restart lần nữa sau revert.

## Rollback verification
```
$ docker exec gpay-mongo mongosh --quiet --eval "printjson(db.adminCommand({listDatabases:1}).databases.map(d=>d.name))"
[ 'admin', 'centralized-export-service', 'config', 'goopay', 'local', 'market_db', 'payment-bill-service', 'phase_e_ns_1777885325' ]
# (cdc_system absent — đã drop)

$ curl -s http://127.0.0.1:18083/connectors/goopay-local/config | jq 'has("signal.data.collection")'
false
```

## Lesson recorded
`agent/memory/global/lessons.md` 2026-05-20 entry — Global Pattern `[Pipeline A đề xuất pattern B yêu cầu ghi vào store C] → nếu C có constraint read-only thì B infeasible bất kể có chạy dev không`.

## Next phase
`snapshot-custom-worker` — bypass Debezium incremental hoàn toàn cho snapshot. Debezium giữ vai trò CDC streaming (read-only).
