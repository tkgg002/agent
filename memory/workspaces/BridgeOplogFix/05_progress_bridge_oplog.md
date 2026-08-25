# 05 Audit Log & Progress: Bridge Oplog Fix

## Audit Trail (Append Only)

- `[2026-08-05T13:15:00+07:00] [Brain:Gemini]` Khởi tạo workspace `BridgeOplogFix` tuân thủ Quy tắc 4 (GEMINI Core Rules).
- `[2026-08-05T13:15:10+07:00] [Brain:Gemini]` Phân tích Root Cause:
  1. `server_setup.go:279` truyền nhầm `shadowDB` thay vì `db` (System DB) làm `ActivityLogger` không tìm thấy `cdc_system.cdc_activity_log`.
  2. `bridge_handler.go` không lấy đúng `tc.PrimaryKeyField` từ Config registry mà tự ý override `_id -> id`.
  3. `batchUpsert` bọc trong `tx.Transaction` gây lỗi PostgreSQL `25P02: current transaction is aborted`.
- `[2026-08-05T13:15:20+07:00] [Muscle:ChiefEngineer]` Sửa `server_setup.go` truyền `db`, sửa `bridge_handler.go` dùng `tc.PrimaryKeyField` và đo đạc `oplog_fetched` vs `shadow_written`.
- `[2026-08-05T13:15:30+07:00] [Muscle:ChiefEngineer]` Biên dịch thành công 100% 3 layer (Web, CMS, Worker).

## Phiên 2026-08-05T14:00–17:00+07:00

- `[2026-08-05T14:55:00+07:00] [Brain:Gemini]` CHECKPOINT: Phát hiện 3 bug mới qua runtime trace:
  1. **Bug**: `BridgeHandler.h.db = main DB` → `batchUpsert` exec INSERT vào shadow schema trên sai DB → `42P01 relation "shadow_testces.export_jobs" does not exist`.
  2. **Bug**: Fallback cứng `_id→id` trong `resolveCollection` L318-321 override giá trị PK đúng từ config → `pg_pk = id` sai.
  3. **Bug**: `actLogger` dùng `h.db` (sau fix thành shadowDB) → không ghi được ActivityLog vào system DB.

- `[2026-08-05T15:00:00+07:00] [Muscle:ChiefEngineer]` Fix Bug #2: Xóa hardcoded fallback `_id→id` ở `bridge_handler.go:318-321`. Build OK.

- `[2026-08-05T15:15:00+07:00] [Muscle:ChiefEngineer]` Fix Bug #1: Sửa `server_setup.go:279` truyền `shadowDB` vào `NewBridgeHandler` thay vì `db`. Build OK.

- `[2026-08-05T16:15:00+07:00] [Muscle:ChiefEngineer]` Fix Bug #3: Tách `systemDB` riêng — thêm field `systemDB *gorm.DB` vào `BridgeHandler` struct. Cập nhật constructor nhận 2 DB. Thêm `FailWithDetails()` vào `ActivityLogger` để ghi metrics cả khi error. Build OK.

- `[2026-08-05T16:50:00+07:00] [Brain:Gemini]` QC Audit Self-Improvement Loop. Phát hiện các GAP:
  - **D1 CLOSED**: `sync_handler.go` giữ `StartOpTimeSec` trong Worker payload struct — ĐÚNG, CMS không còn gửi field này. Không bug.
  - **D2 OPEN**: Fiber OTel middleware trace propagation chưa verify E2E.
  - **D3 OPEN**: ActivityLog record chưa verify thực tế sau khi restart CDS với đủ 3 fix.
  - **D4 OPEN**: Chưa chạy Bridge Oplog cho collection `export_jobs` sau fix.

- `[2026-08-05T16:50:00+07:00] [Brain:Gemini]` Files modified trong phiên này:
  - `centralized-data-service/internal/handler/source/bridge_handler.go` (xóa fallback _id→id, tách systemDB)
  - `centralized-data-service/internal/server/server_setup.go` (shadowDB + db vào NewBridgeHandler)
  - `centralized-data-service/internal/service/governance/activity_logger.go` (thêm FailWithDetails)
  - `agent/memory/global/lessons.md` (append 4 lessons)
