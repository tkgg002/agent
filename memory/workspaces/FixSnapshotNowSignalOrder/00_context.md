# Context — Fix "Snapshot Now không chạy" do BSON field order

## User report (2026-05-12 ICT)
UI trigger "Snapshot Now" cho `export-jobs` → activity_log thấy `success`, nhưng worker
Snapshot Now không thật sự chạy snapshot. CDC log:
```
debezium signal received  type="" table=export-jobs
debezium signal inserted  database=centralized-export-service collection=export-jobs signal_id=ObjectID(6a02dd52a4ef18f55599b693)
debezium signal dispatched signal_id=ObjectID(6a02dd52a4ef18f55599b693) table=export-jobs
```

## Quy tắc user áp dụng cho phiên này
1. Đọc `agent/memory/global/lessons.md` trước (CLAUDE.md §0, §7).
2. Đọc `work/agent/GEMINI.md` cho role/skill.
3. Chỉ làm đúng yêu cầu — **tìm root cause, KHÔNG tự fix**.
4. Report dựa trên kết quả thực tế (no lies).
5. Verify service work trước khi báo done.
6. Tạo `report_*.md` ghi lại.

## Service stack được kiểm tra
- cms-service (8083): admin endpoint dispatch DebeziumSnapshotCommand qua NATS.
- centralized-data-service worker: subscribe `cdc.cmd.debezium-snapshot` + `cdc.cmd.debezium-signal`.
- gpay-mongo (27017): replica set `rs0`, source DB `centralized-export-service`.
- gpay-kafka-connect (18083): connector `goopay-mongodb-cdc` (MongoDbConnector) — RUNNING.
- Debezium signal collection: `centralized-export-service.debezium_signal`.
