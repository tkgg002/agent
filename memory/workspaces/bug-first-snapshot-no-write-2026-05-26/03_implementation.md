# 03_implementation — snapshot.v2 first-run no write

## File thay đổi

### 1. `centralized-data-service/internal/handler/event_handler.go`
Lines 84-99 (block "routes empty"):
- Debug log → Warn log.
- Thêm context: subject, source_db, source_table.
- Giữ nguyên return signature `(0, nil)` để không break Kafka consumer
  flow (silent-skip vẫn là behavior hợp lệ ở stream realtime nhiều topic).

### 2. `centralized-data-service/internal/handler/snapshot_runner_handler.go`
3 chỗ thay đổi nằm trong `runSnapshot`:

**(a) Pre-flight reload + assert route — sau khi mở mongo client (L279–307)**:
- Type-assert `r.registrySvc` về interface có `ReloadAll(ctx) error`
  để force reload không phụ thuộc NATS signal.
- Sau reload, gọi `ResolveSourceRoutes(srcDB, srcColl)` — nếu vẫn rỗng
  thì `markProgressError` + return error có message chỉ đích danh
  cờ `is_active` cần kiểm tra.

**(b) Khai báo `batchWritten` cùng `batchErrors` — L390-392**:
- `int64` counter accumulate trong inner loop.

**(c) Inner doc loop — L461-495**:
- `written, err := r.eventHandler.HandleRaw(...)` thay vì `_, err`.
- Nhánh `err != nil`: giữ recordDocError như cũ.
- Thêm nhánh `written == 0`: route silently empty → recordDocError
  với stage "route empty" + message chỉ rõ nguyên nhân. CB
  (consecutiveErrors / batch ratio) sẽ trip ngay batch đầu.
- Happy-path: `batchWritten += int64(written)`.

**(d) `rowsTotal += batchWritten` (L521) thay vì `rowsTotal += int64(len(batch))`**:
- activity_log.rows_affected giờ phản ánh số doc thực sự được route + add
  vào batch buffer, không phải số doc đã đọc từ Mongo Find.

## Verify
- `go build ./...` ở centralized-data-service → EXIT=0.
- `go vet ./internal/handler/...` → EXIT=0.
- `go test ./internal/handler/ -count=1` → PASS (0.9s).

## Side-effect & Risk audit
- **Buffer.Flush() vẫn nuốt lỗi**: lỗi UPSERT từ batchUpsert vẫn không
  propagate về snapshot_runner. Đã kiểm — fallback path ghi vào
  `failed_sync_logs` nên forensics vẫn có. Để fix triệt để cần BatchBuffer
  expose flush stats — DEFER sang follow-up.
- **Pre-flight reload tăng 1 query GetActive + 1 query GetAll connections +
  1 query ListBySourceObject mỗi source mỗi lần dispatch**: snapshot.v2
  là operator-driven (không phải hot-path streaming) → chi phí chấp nhận được.
- **Strict mode**: vẫn fail-on-first-error ở `recordDocError`. Non-strict:
  CB sẽ trip ở ngưỡng 100 consecutive HOẶC 50% batch ratio.

## Diff summary (số dòng)
- event_handler.go: +9 / -2 (block routes-empty log).
- snapshot_runner_handler.go: +33 / -3 (pre-flight + written tracking).
