# Plan: Debezium Delete Flow Debugging

## Phase 1: Research & Diagnosis
1. **Kiểm tra cấu hình Debezium**:
   - Xem file `deployments/debezium/pg-source-connector.json` và các cấu hình connector khác để kiểm tra xem có thuộc tính liên quan đến delete hay không.
2. **Kiểm tra mã nguồn Sink Worker**:
   - Tìm nơi xử lý Kafka message trong `centralized-data-service`.
   - Xem cách handler xử lý các hoạt động (operation type: `c`, `u`, `d`, `r`). Đặc biệt kiểm tra xem `d` (delete) có bị bỏ qua không, hoặc logic xử lý delete có lỗi không.
   - Kiểm tra việc xử lý Tombstone messages (message có value `nil` hoặc chứa thông tin delete).
3. **Xác định Root Cause**:
   - Đưa ra kết luận vì sao delete không chạy (do Debezium không gửi event, hay do Sink Worker skip event, hay do shadow table mapping logic).

## Phase 2: Implementation Plan
- Viết `implementation_plan.md` chi tiết về cách khắc phục và gửi cho User duyệt.

## Phase 3: Execution & Verification
- Thực hiện sửa đổi sau khi được duyệt.
- Chạy unit tests và integration tests (nếu có) để xác thực.
