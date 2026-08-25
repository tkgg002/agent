# 01 Requirements: Bridge Oplog Gap Recovery & Metrics Audit

## Scope
1. Trích xuất biến động Oplog / Change Stream từ MongoDB trong khoảng thời gian `[StartTimeSec, EndTimeSec]`.
2. Ghi bù dữ liệu bị gap vào PostgreSQL Shadow Table.
3. Đo đạc và ghi vết minh bạch 2 chỉ số: `oplog_fetched` và `shadow_written` vào `cdc_system.cdc_activity_log` trên System DB (`db`).
4. Tôn trọng 100% `PrimaryKeyField` từ Config (`tc.PrimaryKeyField`), không tự ý đoán mò tên cột.
5. Truyền vết OpenTelemetry W3C TraceContext giữa CMS và Worker (100% Trace ID Propagation).

## Definition of Done (DoD)
- [x] Trace Context nối từ UI -> CMS -> NATS -> Worker.
- [x] Activity Log ghi vào `cdc_system.cdc_activity_log` với `Operation = bridge-oplog`.
- [x] Log chi tiết metrics `oplog_fetched` và `shadow_written` trong `details` JSON.
- [x] Multi-row batch upsert không bị ngắt cả transaction khi gặp lỗi single row.
- [x] Đã khởi tạo đầy đủ bộ hồ sơ tài liệu tại `agent/memory/workspaces/BridgeOplogFix`.
