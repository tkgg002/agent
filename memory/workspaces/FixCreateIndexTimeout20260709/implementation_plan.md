# Kế hoạch triển khai - Tối ưu hóa bất đồng bộ Create/Drop Index & Khắc phục lock-storm trong transmuter

Nhằm khắc phục việc tạo index bị `INVALID` và cải thiện hiệu năng transmute (hiện đang tốn tới 10s cho các batch nhỏ do tranh chấp khóa tạo index), chúng ta sẽ thực hiện 2 thay đổi chính:
1. Chuyển đổi cơ chế xử lý tạo/xóa index của `IndexHandler` sang bất đồng bộ (asynchronous) bằng goroutine.
2. Thêm cache `ensuredShadowIndexes` trong `TransmuterModule` để không chạy lại câu lệnh check và tạo/xóa index `CREATE INDEX CONCURRENTLY` trên shadow table liên tục mỗi khi transmute (ngăn chặn lock storm).

## Proposed Changes

### centralized-data-service

#### [MODIFY] [index_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/governance/index_handler.go)

- Cập nhật `HandleCreateIndex` và `HandleDropIndex`:
  - Phản hồi ngay lập tức cho NATS với `Status: "success"` để giải phóng client và tránh timeout.
  - Spawn goroutine chạy ngầm cho `CreateIndexConcurrently` và `DropIndexConcurrently`.
  - Sử dụng detached context cho goroutine nền.

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)

- Thêm trường `ensuredShadowIndexes map[string]bool` vào `TransmuterModule` struct và khởi tạo nó trong `NewTransmuterModule`.
- Sửa đổi `ensureShadowSourceIDIndex`:
  - RLock kiểm tra cache `ensuredShadowIndexes`. Nếu đã được check/tạo, return ngay lập tức.
  - Nếu chưa có trong cache và index hợp lệ tồn tại trong DB, lưu cache = true và return.
  - Nếu chưa có và index cần tạo mới (hoặc cần drop index invalid cũ), gán cache = true ngay lập tức trước khi chạy tiến trình ngầm để tránh tranh chấp từ các luồng song song tiếp theo, sau đó chạy drop/create trong goroutine nền.

## Verification Plan

### Automated Tests
- Chạy unit test của package `master` (đặc biệt là `transmuter_index_test.go`):
  ```bash
  go test -v ./internal/service/master/...
  ```
- Chạy biên dịch toàn bộ handler:
  ```bash
  go test -v ./internal/handler/...
  ```
