# Kế hoạch triển khai: Sửa lỗi OOM Transient & Poison Pill

Sửa lỗi nhận diện nhầm lỗi cạn kiệt tài nguyên (OOM - SQLSTATE 53200) của PostgreSQL thành Poison Pill trong `centralized-data-service`.

## User Review Required

> [!NOTE]
> Giải pháp duy nhất và tối ưu được chọn:
> 1. **Kiểm tra Lỗi OOM**: Thêm kiểm tra `"out of memory"` và `"sqlstate 53200"` vào hàm `isTransientError` để phân loại OOM là lỗi tạm thời.
> 2. **Dừng Split khi gặp Lỗi Transient**: Bổ sung check `isTransientError(err)` trong hàm `processSubBatch` để ngăn chặn việc đệ quy chia tách vô hạn làm biến dữ liệu tốt thành Poison Pill khi DB sập.

## Open Questions

Không có.

---

## Proposed Changes

### Centralized Data Service

#### [MODIFY] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
- Cập nhật hàm `isTransientError` để bổ sung check OOM.
- Cập nhật hàm `processSubBatch` để thêm điều kiện check `isTransientError` và trả về lỗi transient ngay lập tức thay vì tiếp tục đệ quy `binarySearchSplit`.

---

## Verification Plan

### Automated Tests
- Chạy unit test của package `master` trong `centralized-data-service`:
  ```bash
  go test -v ./internal/handler/master/...
  ```
