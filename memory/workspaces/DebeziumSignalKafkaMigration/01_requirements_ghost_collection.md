# 01 Requirements — Ghost Collection Workaround (Bug C)

## Context
- Phase trước (`snapshot-signal-kafka-key-fix`) fix xong Bug A (worker key) + Bug B (CMS Vite placeholder leak). Debezium đã nhận signal: log `Requested 'INCREMENTAL' snapshot of data collections '[centralized-export-service.export-jobs]'`.
- Bug C: Debezium 2.5.4 Mongo connector NPE tại `MongoDbIncrementalSnapshotChangeEventSource.lambda$emitWindowOpen$2(:228)` → snapshot fail ngay tại chunk emit, shadow rows = 0.
- Phase bump 2.5.4 → 2.7.4 đã FAIL: Confluent Hub catalog không có 2.7.x. 20 lần retry plugins=0 (log `/tmp/.../blec2sj9c.output`).
- User quyết: KHÔNG bump (giữ 2.5.4 vì sẽ thay Confluent về sau), áp dụng workaround "Ghost Collection".

## Decision (user)
> "Giải pháp 2: Sử dụng giải pháp tình thế 'Bộ sưu tập ma' (Ghost Collection)... Tạo một bộ sưu tập rỗng hoàn toàn, đặt tên là cdc_system.debezium_watermarks. Bổ sung thêm dòng này vào cấu hình để lừa Debezium: 'signal.data.collection': 'cdc_system.debezium_watermarks'... giữ 2.5.4 lúc này."

## Why Ghost Collection fixes NPE
- `MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen` ghi WATERMARK doc vào `signal.data.collection` để đánh dấu chunk boundary.
- Debezium 2.5.4 KHÔNG có default + KHÔNG validate khi config vắng → NPE khi resolve collection handle.
- Cung cấp empty collection thật → connector ghi watermark vào đó → snapshot chunks flow bình thường.

## Functional Requirements
1. Revert docker-compose Debezium plugins về `2.5.4` (sau khi bump 2.7.4 fail).
2. Recreate `gpay-kafka-connect` container, đảm bảo 3 plugin (mongo/pg/mysql) install OK + REST API up.
3. Tạo empty collection `debezium_watermarks` trong DB `cdc_system` trên Mongo source `gpay-mongo` (port 17017 host).
4. PATCH config 2 connector `goopay-local`, `goopay-dev` để thêm:
   - `"signal.data.collection": "cdc_system.debezium_watermarks"`
   - Giữ nguyên: `signal.kafka.topic`, `topic.prefix`, các include.list.
5. Restart cả 2 connector + verify state RUNNING/RUNNING.
6. Trigger snapshot `centralized-export-service.export-jobs` qua worker (NATS publish), verify:
   - Connect log có `Requested INCREMENTAL snapshot` (đã có từ phase trước).
   - Connect log KHÔNG còn NPE `lambda$emitWindowOpen$2`.
   - Shadow PG count tăng đúng = source Mongo count (delta-verified).

## Non-Functional Requirements
- KHÔNG bump Debezium version.
- KHÔNG block source DB (snapshot phải là incremental, không phải blocking).
- KHÔNG cheat: shadow count phải capture before/after, report delta thực tế.
- Worker code (Bug A fix) KHÔNG đổi.
- CMS code (Bug B fix) KHÔNG đổi.

## Out of Scope
- Migrate goopay-dev khỏi Mongo cluster 10.200.187.x (production-ish, user sẽ tự handle).
- Việc loại bỏ Confluent Hub (user note: "tao sẽ nâng cấp lên và bỏ mẹ cái confluent đi sau").
- Cleanup watermark docs trong ghost collection (Debezium tự manage, không cần TTL).

## Definition of Done
- [x] docker-compose 2.5.4 (revert applied)
- [ ] kafka-connect container UP + 3 plugin install OK
- [ ] mongo `cdc_system.debezium_watermarks` tồn tại (empty)
- [ ] 2 connector config có `signal.data.collection`
- [ ] 2 connector state RUNNING/RUNNING
- [ ] Trigger snapshot qua worker NATS → shadow delta > 0
- [ ] Connect log: `Requested INCREMENTAL` + zero NPE trong 60s sau trigger
- [ ] Report file `report_2026-05-20_snapshot-ghost-collection.md` với số liệu thực
- [ ] APPEND `05_progress.md`
- [ ] APPEND lesson Global Pattern vào `lessons.md`
