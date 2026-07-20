# Kế Hoạch Triển Khai: Sửa Lỗi Drift Giả và Đồng Bộ Trạng Thái Đối Soát

Kế hoạch này giải quyết việc trạng thái đối soát (reconciliation status) báo "drift" giả do độ trễ đồng bộ tạm thời (replication lag) hoặc lỗi scan tạm thời làm phát sinh `driftedWindows > 0`, nhưng kết quả drill-down thực tế không tìm thấy chênh lệch (`mismatches == 0`). Đồng thời đồng bộ logic báo lỗi scan (status "error") giữa Segment A và Segment B, và chuẩn hóa cấu trúc truy vấn `ListIDTsInWindow` như một biện pháp phòng ngừa bất đối xứng cấu trúc (lọc bản ghi đã xóa mềm và timestamp NULL) trong tương lai.

## Những Thay Đổi Đề Xuất

### 1. Component: Recon Dest Agent Query
Cập nhật phương thức `ListIDTsInWindow` để áp dụng cùng một bộ lọc loại bỏ các hàng xóa mềm (`_deleted = true`) và timestamp NULL, đồng bộ hoàn toàn với logic tính `HashWindow`.

#### [MODIFY] [recon_dest_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go)
- **Hàm `ListIDTsInWindow` (nhánh `_source_ts`):**
  Thêm điều kiện `AND "_source_ts" IS NOT NULL AND NOT "_deleted"` vào câu SQL.
  ```sql
  SELECT %s::text AS id, "_source_ts" AS ts FROM %s
   WHERE "_source_ts" >= ? AND "_source_ts" < ?
     AND "_source_ts" IS NOT NULL
     AND NOT "_deleted"
  ```
- **Hàm `ListIDTsInWindow` (nhánh Domain TS):**
  Thêm điều kiện `AND %s IS NOT NULL AND NOT "_deleted"` vào câu SQL.
  ```sql
  SELECT %s::text AS id, %s AS ts FROM %s
   WHERE %s >= ? AND %s < ?
     AND %s IS NOT NULL
     AND NOT "_deleted"
  ```

### 2. Component: Recon Segment A Engine
Đồng bộ logic gán trạng thái đối soát `statusStr` và cập nhật thông tin chạy `finishRun` tương tự Segment B.

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- **Hàm `RunHashWindowCheck`:**
  Khai báo biến `var errMsg string` ở đầu hàm. Cập nhật defer block để truyền `errMsg` vào `finishRun(ctx, handle, status, errMsg)`.
  Ở cuối hàm, cập nhật logic gán `statusStr`:
  ```go
  statusStr := "ok"
  if driftedWindows > 0 {
      statusStr = "drift"
  }
  if handle.mismatches == 0 && driftedWindows > 0 {
      statusStr = "error"
      status = "failed"
      errMsg = fmt.Sprintf("%d windows scanned but no diff (possible scan error)", driftedWindows)
  }
  ```

### 3. Component: Unit Tests
Cập nhật lại các chuỗi kỳ vọng (ExpectQuery) trong unit test của `ReconDestAgent` và `ReconCore` để khớp với câu SQL mới.

#### [MODIFY] [recon_dest_agent_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent_test.go)
- Cập nhật truy vấn mock trong `TestDestAgent_ListIDTsInWindow_DomainTS` và `TestDestAgent_ListIDTsInWindow_Default`.

#### [MODIFY] [recon_tier_a_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a_test.go)
- Cập nhật truy vấn mock cho `da.ListIDTsInWindow` trong `TestRunHashWindowCheck_GlobalMismatch_FallbackToLoop`.

## Kế Hoạch Xác Minh

### Kiểm thử tự động (Automated Tests)
- Chạy unit test của package `recon`:
  `go test -v ./internal/service/recon/...`
