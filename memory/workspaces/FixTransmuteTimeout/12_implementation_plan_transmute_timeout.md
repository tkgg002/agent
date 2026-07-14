# Kế hoạch triển khai - Tối ưu hóa Toàn diện Hiệu năng Transmuter (Hạn chế Timeout)

Kế hoạch này đề xuất giải pháp hệ thống toàn diện để tối ưu hóa hiệu năng đồng bộ dữ liệu shadow-to-master (Transmute) cho các bảng lớn (100M+ dòng), loại bỏ tận gốc các nguyên nhân gây ra lỗi `context deadline exceeded` (300s).

## User Review Required

> [!IMPORTANT]
> - **Tối ưu hóa câu lệnh Incremental/Heal**: Khi có danh sách `_source_ids` cụ thể, câu lệnh SQL sẽ được tối ưu bỏ qua phân trang con trỏ `_gpay_id` và `ORDER BY 1 LIMIT 2000` (vốn là nguyên nhân ép Postgres quét toàn bộ index PK trong bảng 100M dòng).
> - **Tự động hóa Index trên Shadow**: Khi Transmuter chạy, hệ thống sẽ kiểm tra xem đã có index non-partial trên `_source_id` của bảng Shadow chưa. Nếu chưa, tiến hành tạo bất đồng bộ:
>   `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_<table_name>_source_id ON <schema>.<table_name> (_source_id)`
>   Điều này giúp các câu lệnh chữa lành/real-time sync chạy ở độ phức tạp $O(1)$ thay vì quét toàn bộ bảng.
> - **Cơ chế Checkpoint/Resume (Bản chất giải pháp cho Full Sync)**: Tận dụng cột `last_cursor_json` trong bảng `cdc_system.sync_runtime_state`. Tiến trình Full Sync khi chạy sẽ bắt đầu từ checkpoint cũ. Sau mỗi lô (batch) 2000 dòng được ghi thành công, checkpoint `last_gpay_id` sẽ được cập nhật vào DB. Khi hoàn thành toàn bộ bảng, checkpoint sẽ được reset về `{}`. Nếu tiến trình bị gián đoạn (timeout/restart), lần chạy sau sẽ tiếp tục từ checkpoint thay vì chạy lại từ đầu.
> - **Chạy bất đồng bộ (Background Goroutine)**: Chuyển toàn bộ luồng xử lý chính trong `HandleTransmute` sang background goroutine, gán timeout context lớn (30 phút cho incremental/heal, 24 giờ cho full sync).

## Proposed Changes

### Centralized Data Service (`centralized-data-service`)

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Sửa hàm `Run`:
  - Khi bắt đầu Full Sync (`len(onlySourceIDs) == 0`), tải checkpoint `lastGpayID` từ `SyncRuntimeState.LastCursorJSON`.
  - Trong vòng lặp batch, nếu `len(onlySourceIDs) > 0`, thực hiện thoát vòng lặp sau lượt chạy đầu tiên (vì đã lấy toàn bộ ID được yêu cầu).
  - Sau mỗi lô `processBatch` và `bulkUpsertMaster` thành công, lưu lại `lastGpayID` vào `LastCursorJSON`.
  - Khi hoàn thành vòng lặp thành công, reset `LastCursorJSON` về `{}`.
- Sửa hàm `fetchShadowBatch`:
  - Tách biệt logic khi `len(onlyIDs) > 0`:
    - Truy vấn trực tiếp `WHERE _source_id IN (?)` mà không gán `_gpay_id > ?`, không `ORDER BY` và không `LIMIT`.
  - Tự động kiểm tra và tạo index `CONCURRENTLY` cho `_source_id` dưới nền nếu chưa tồn tại.

#### [MODIFY] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
- Sửa hàm `HandleTransmute` chạy tiến trình chính qua background goroutine.
- Khởi tạo detached context từ `context.Background()` và gán timeout (30m cho incremental, 24h cho full sync).
- Quản lý đóng OTel span con (`cdc.worker.transmute.process`) đúng cách bên trong goroutine.

## Verification Plan

### Automated Tests
- Chạy unit tests liên quan đến transmuter:
  `go test -v ./internal/service/master/...`
- Kiểm thử biên dịch:
  `go build -v ./...`

### CDC CMS Web (`cdc-cms-web`)

#### [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
- Đặt mặc định khoảng thời gian là 30 ngày gần nhất (thay vì 7 ngày) khi chọn chế độ "Tùy chỉnh khoảng thời gian".

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Sửa hàm `useTableHistory` để nhận tham số tùy chọn `pageSize` (mặc định 30) giúp lấy lượng dữ liệu lịch sử lớn hơn khi cần.

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Tích hợp thêm tab "Phiên đối soát đã xử lý" bên cạnh tab "Phiên chưa xử lý" bằng component `Tabs` từ antd.
- Gọi hook `useTableHistory` lấy 100 bản ghi lịch sử, lọc ra danh sách các phiên đã xử lý thành công (`healed_at != null`).
- Thiết kế các cột hiển thị thông tin kết quả heal (`healed_count`, `pruned_missing_src_count`) và thời gian xử lý cụ thể.
