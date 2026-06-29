# Audit Report: Debezium Delete Flow, Avro Key Decoding & Soft-Delete 2-Step Fixes (Bản cuối)

**Workspace**: `bug-debezium-delete-not-working-2026-06-25`  
**Date**: 2026-06-25  
**Auditor**: Brain (Antigravity)  
**Status**: ✅ Approved & Pass  

---

## 1. Mục tiêu Audit
Đánh giá toàn diện các thay đổi mã nguồn đã thực hiện trong workspace này để giải quyết sự cố luồng DELETE Debezium không hoạt động, lỗi rác dữ liệu, và lỗi ghi đè mất thông tin lịch sử `_raw_data` khi UPDATE xóa mềm.

---

## 2. Kết quả đối chiếu với Kế hoạch thực thi (Implementation Plan)

### Lỗi 1: Fallback PK từ Kafka Key & Giải mã Avro Key (Môi trường thực tế)
- **Fix**: Tự động giải mã Avro-encoded key trong `kafka_consumer.go` (check byte 0 và len > 5) rồi map JSON string vào `kafka_key`. `EventHandler` thực hiện fallback trích xuất PK từ key khi `before` rỗng. Kết quả: Đã hoạt động chính xác.

### Lỗi 2: Trực quan hóa rác dữ liệu khi DELETE (Soft-Delete 2 bước)
- **Fix**: Sửa đổi `batch_buffer.go` để chia chunk thành các sub-chunks liên tiếp theo `IsDelete`. Thực thi Soft-Delete 2 bước: UPDATE trước, nếu RowsAffected == 0 thì INSERT tombstone sau (ngăn chặn dòng tombstone rác).

### Lỗi 3: Ghi đè cột `_raw_data` làm mất dữ liệu lịch sử
- **Fix**: Chỉnh sửa phương thức `BuildSoftDeleteUpdateSQL` tại [schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go). Thay vì gán `_raw_data = ?` làm ghi đè toàn bộ cột, chúng ta thực hiện merge thuộc tính `"_deleted": true` vào JSONB cũ trực tiếp trên database bằng toán tử `||`:
  ```sql
  _raw_data = COALESCE(_raw_data, '{}'::jsonb) || '{"_deleted": true}'::jsonb
  ```
- **Xác thực**: Loại bỏ việc truyền `r.RawData` vào args của query UPDATE xóa mềm để tránh truyền thừa.

---

## 3. Đánh giá tính tuân thủ Core Patterns & Architecture
1. **JSONB Concat Pattern**: Sử dụng toán tử `||` gốc của Postgres đảm bảo tính tối giản (Simplicity First), hạn chế tối đa data payload truyền qua mạng và giữ lại 100% dữ liệu lịch sử.
2. **Unit Test Coverage**: Đã viết thêm 2 unit test cases `TestBuildSoftDeleteUpdateSQL` và `TestBuildSoftDeleteInsertSQL` vào [schema_adapter_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/schema_adapter_test.go) để kiểm chứng SQL sinh ra.

---

## 4. Kết luận
Toàn bộ integration tests (`test/internal/handler/...`) và service unit tests (`test/internal/service/...`) **đều PASS 100%**. Không có lỗi hồi sinh dữ liệu hay mất dữ liệu lịch sử. Đóng workspace an toàn.
