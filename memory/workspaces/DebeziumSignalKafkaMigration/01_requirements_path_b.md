# 01 — Requirements: Path B — Custom Snapshot Worker (bypass Debezium signal)

## Vì sao có file này
Path A (đợt trước) chỉ vá: strip `signal.data.collection` khỏi connector config
→ Debezium NPE silent-fail ở `MongoDbIncrementalSnapshotChangeEventSource.emitWindowOpen`.
Snapshot button của user IM lặng, không write source DB nữa (đúng) nhưng cũng
KHÔNG materialize được dữ liệu sang shadow PG (sai — chức năng mất).

Boss verb: **"go path B"** = build worker tự đọc Mongo bypass Debezium signal,
giữ rule **source DB read-only**.

## Mục tiêu (Definition of Done)
1. Một nút "Snapshot" trên CMS (hoặc API `POST /api/sources/:id/snapshot.v2`)
   trigger backfill collection sang shadow PG.
2. Source DB MongoDB tuyệt đối **chỉ có Find** (read-only). Không insert
   `debezium_signals`, không tạo `__debezium_snapshot_open/close`.
3. Pipeline tái sử dụng: snapshot → `EventHandler.HandleRaw` → DynamicMapper
   → BatchBuffer → shadow upsert (cùng path với realtime CDC stream).
4. Resumable: crash giữa chừng → restart tiếp tục từ checkpoint `last_seen_id`,
   không double-write (`_gpay_source_id` UPSERT ON CONFLICT đã idempotent).
5. Có audit row trong `cdc_system.snapshot_progress` để CMS hiển thị tiến độ.
6. Trace_id propagate end-to-end (CMS → NATS header → worker log → progress row).

## Constraints / nguyên tắc bất di
- KHÔNG ghi vào source DB (rule vàng CDC, đã ghi `lessons.md` line 3640+).
- KHÔNG bỏ Debezium realtime — Path B chỉ thay đợt backfill snapshot, oplog
  streaming vẫn Debezium-driven.
- KHÔNG xoá `cdc.cmd.debezium-snapshot` ngay — giữ alias trống/no-op để FE
  cũ không 404. Phase sau dọn.
- Plan minimal: dùng lại registry, connection_manager, eventHandler đang có.
  Tuyệt đối không tạo abstraction mới khi chưa cần.

## Phạm vi
- Worker: thêm 1 handler + 1 NATS subscriber.
- CMS: 1 command struct + 1 RegisterSubject + 1 handler method (hoặc thay
  body của `TriggerSnapshot`).
- Schema: 1 migration `058_v1_snapshot_progress.sql`.

## Out of scope
- Throttling / rate-limit (mặc định batch_size 1000, để CMS truyền nếu cần).
- Multi-collection 1-shot — gọi từng source_object_id, scheduler tự fan-out.
- Cross-cluster Mongo (replica set vs standalone) — tái dụng Mongo URI từ
  `connection_registry`, ai connect được Debezium thì connect được snapshot.
