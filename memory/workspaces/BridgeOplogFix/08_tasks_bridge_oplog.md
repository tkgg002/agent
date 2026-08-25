# 08 Tasks: Bridge Oplog Implementation Checklist

## Checklist

- [x] **Task 1**: Kiểm tra & sửa `server_setup.go:279` truyền đúng `db` (System DB) cho `NewBridgeHandler`.
- [x] **Task 2**: Cập nhật `resolveCollection` trong `bridge_handler.go` tôn trọng 100% `tc.PrimaryKeyField` từ Config.
- [x] **Task 3**: Cập nhật `batchUpsert` chạy các query độc lập qua `h.db.WithContext(ctx)` tránh kẹt block `25P02`.
- [x] **Task 4**: Đo đạc & ghi nhận minh bạch 2 chỉ số `oplog_fetched` và `shadow_written` vào `cdc_system.cdc_activity_log`.
- [x] **Task 5**: Đồng bộ bộ tài liệu Workspace `BridgeOplogFix` tại `agent/memory/workspaces/BridgeOplogFix`.
