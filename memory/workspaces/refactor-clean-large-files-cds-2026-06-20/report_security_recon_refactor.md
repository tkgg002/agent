## Security Report - Recon Refactor (Source & Dest Agents)

### Scan Summary
| Category | Issues Found | Severity | Status |
|----------|-------------|----------|--------|
| Input Validation & Injection Prevention | 0 | None | ✅ PASS |
| Data Mutation Protection (Read-Only Guard) | 0 | None | ✅ PASS |
| Secrets / PII Exposure | 0 | None | ✅ PASS |
| Overload Guard (Breaker & Limiter) | 0 | None | ✅ PASS |

### Vulnerabilities Found
Không tìm thấy lỗ hổng bảo mật nào trong các file thuộc package `recon` vừa được refactor.

### Security Mechanisms Implemented

#### 1. Input Validation & SQL/BSON Injection Prevention
- **ReconSourceAgent (Mongo)**:
  - Tên trường timestamp được kiểm tra và lọc qua regex an toàn `governance.CandidateNameRE.MatchString` trước khi đưa vào query.
  - Hàm `resolveTimestampField` giới hạn độ dài ký tự tối đa là 64 và chỉ cho phép chữ cái, chữ số, dấu gạch dưới.
- **ReconDestAgent (Postgres)**:
  - Tất cả các tên bảng, tên cột động đi qua hàm kiểm tra phòng thủ `validateIdent` (kiểm tra rỗng, giới hạn độ dài <= 128 ký tự, từ chối ký tự điều khiển như `\x00`, `\n`, `\r`).
  - Sử dụng hàm `quoteIdent` và `quoteRelation` để bọc các định danh SQL trong double quotes `"` và nhân đôi bất kỳ ký tự `"` nào bên trong để ngăn chặn hoàn toàn các cuộc tấn công SQL Injection thông qua tên bảng hoặc cột.

#### 2. Data Mutation Prevention (Read-Only Guard)
- **ReconDestAgent**:
  - Toàn bộ các API đọc đi qua `readOnlyDB(ctx)`. Hàm này khởi tạo một transaction Postgres và lập tức gọi `SET TRANSACTION READ ONLY`.
  - Điều này đảm bảo rằng ngay cả khi có bất kỳ sai sót logic hay query SQL nào bị chèn mã độc hại, Postgres cũng sẽ từ chối việc ghi/sửa đổi/xóa dữ liệu.

#### 3. Secrets & Credentials Protection
- **ReconSourceAgent**:
  - Hàm helper `redactURL` được sử dụng để lọc và xóa bỏ thông tin mật khẩu/credentials khỏi connection string MongoDB trước khi ghi log lỗi.
  - Không có token, khóa bí mật hay cấu hình nhạy cảm nào bị hardcode.

#### 4. Overload & Resource Exhaustion Protection
- **Cả Source & Dest Agents**:
  - Tích hợp rate limiter của `golang.org/x/time/rate` trên các luồng lặp stream rows để tránh làm nghẹt kết nối mạng và bộ nhớ.
  - Tích hợp circuit breaker `gobreaker` bao bọc mọi truy vấn cơ sở dữ liệu để nhanh chóng ngắt kết nối khi phát hiện DB liên tục gặp timeout/lỗi, bảo vệ hệ thống khỏi cascade failures.

### Verdict
✅ PASS
