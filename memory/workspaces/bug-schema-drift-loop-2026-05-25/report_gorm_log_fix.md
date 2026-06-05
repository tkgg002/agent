# Báo cáo: Sửa đổi Định dạng Log GORM & Giảm Thiểu Log Spam

**Ngày thực hiện**: 2026-05-26  
**Dự án**: `centralized-data-service`  
**Workspace**: `bug-schema-drift-loop-2026-05-25`  
**Trạng thái**: Hoàn thành ✅  

---

## 1. Nội dung Thay đổi (Changes Implemented)

### Tệp đã sửa đổi:
- `pkgs/database/gorm_logger.go` (Sửa đổi cấu hình Zap logger của GORM và phân loại mức độ log trong `Trace`).
- `pkgs/database/gorm_logger_test.go` (Tạo mới: Unit test kiểm thử logic hạ log level khi có lỗi expected).

### Chi tiết logic sửa đổi:
1. **Tắt Stacktrace của GORM logger**:
   Cấu hình Zap logger trong `NewCDCLogger` được bổ sung cấu hình `zap.AddStacktrace(zap.FatalLevel)`. Điều này ngăn việc in ra các khối stacktrace cồng kềnh cho các lỗi ở mức `Error` hoặc `Warn`, giúp giữ log JSON ngắn gọn và dễ theo dõi.
   ```go
   func NewCDCLogger(level gormlogger.LogLevel) gormlogger.Interface {
       z, _ := zap.NewProduction(
           zap.AddCallerSkip(3),
           zap.AddStacktrace(zap.FatalLevel),
       )
       return &cdcLogger{level: level, zap: z}
   }
   ```

2. **Hạ log level cho lỗi dự kiến (Expected errors)**:
   Trong phương thức `Trace` của `cdcLogger`, chúng ta kiểm tra lỗi SQL. Nếu lỗi chứa các mã hoặc thông điệp liên quan đến trùng khóa (`23505`, `duplicate key`) hoặc deadlock (`40P01`, `deadlock`), log level sẽ được hạ xuống `Warn` vì các lỗi này đã được CDC Worker xử lý an toàn bằng cơ chế sequential fallback.
   ```go
   errStr := err.Error()
   isExpected := strings.Contains(errStr, "23505") || strings.Contains(errStr, "40P01") ||
       strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "deadlock")

   if isExpected {
       l.zap.Warn("gorm exec expected error (will fallback/retry)", ...)
   } else {
       l.zap.Error("gorm exec error", ...)
   }
   ```

---

## 2. Kết quả Xác minh (Verification Results)

### A. Kiểm thử Unit Test
Chúng ta đã viết bổ sung 3 test case để kiểm thử hành vi phân loại lỗi của GORM logger:
- `TestCDCLogger_Trace_UnexpectedError`: Kiểm tra lỗi không mong muốn (ví dụ lỗi cú pháp SQL) vẫn được ghi nhận ở mức `Error`.
- `TestCDCLogger_Trace_ExpectedUniqueConstraintError`: Kiểm tra lỗi trùng khóa (`23505`) được ghi nhận ở mức `Warn`.
- `TestCDCLogger_Trace_ExpectedDeadlockError`: Kiểm tra lỗi deadlock (`40P01`) được ghi nhận ở mức `Warn`.

**Kết quả chạy test trong thư mục `pkgs/database`**:
```bash
=== RUN   TestCDCLogger_Trace_UnexpectedError
--- PASS: TestCDCLogger_Trace_UnexpectedError (0.00s)
=== RUN   TestCDCLogger_Trace_ExpectedUniqueConstraintError
--- PASS: TestCDCLogger_Trace_ExpectedUniqueConstraintError (0.00s)
=== RUN   TestCDCLogger_Trace_ExpectedDeadlockError
--- PASS: TestCDCLogger_Trace_ExpectedDeadlockError (0.00s)
=== RUN   TestRegistry_SeparatePoolsPerRole
--- PASS: TestRegistry_SeparatePoolsPerRole (0.08s)
=== RUN   TestRegistry_GetDBIsCached
--- PASS: TestRegistry_GetDBIsCached (0.02s)
=== RUN   TestRegistry_ConcurrentGetDBOpensExactlyOnePool
--- PASS: TestRegistry_ConcurrentGetDBOpensExactlyOnePool (0.01s)
=== RUN   TestRegistry_GetPgxPoolIsCached
--- PASS: TestRegistry_GetPgxPoolIsCached (0.00s)
=== RUN   TestRegistry_RejectsUnknownRole
--- PASS: TestRegistry_RejectsUnknownRole (0.00s)
PASS
ok  	centralized-data-service/pkgs/database	0.590s
```
**Kết quả**: Tất cả các test cases biên dịch thành công và vượt qua kiểm thử với mã thoát (exit code) bằng 0.

### B. Kiểm thử tích hợp toàn bộ repository
Chạy `go test ./...` đảm bảo không có bất kỳ regression nào trong toàn bộ dự án:
- `centralized-data-service/internal/handler`: OK (cached)
- `centralized-data-service/internal/service`: OK (0.854s)
- `centralized-data-service/pkgs/database`: OK (0.329s)

---

## 3. Đánh giá Tác động & Khuyên dùng (Impact & Recommendation)

- **Ngăn chặn ngập log (Anti-spam)**: Log stdout sẽ không còn bị lấp bởi các bản ghi JSON khổng lồ chứa stacktrace dài vô ích của các transaction lỗi trùng khóa.
- **Giảm cảnh báo ảo (Alert Optimization)**: Hạ cấp mức độ log xuống `Warn` giúp các hệ thống log parsing (như FluentBit, Vector) gửi dữ liệu về Elasticsearch/SigNoz không gắn cờ `Error` cho lỗi trùng khóa nữa, do đó ngăn chặn các cảnh báo ảo (Slack/PagerDuty alerts) kích hoạt ngoài ý muốn.
