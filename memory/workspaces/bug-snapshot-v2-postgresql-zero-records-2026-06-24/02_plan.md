# Plan: bug-snapshot-v2-postgresql-zero-records-2026-06-24

## 1. Mục tiêu
- Điều tra tại sao quá trình chạy snapshot.v2 trên PostgreSQL cho bảng `failed_sync_logs` (source_object_id=52) không xử lý được dòng nào (0 rows processed).
- Sửa lỗi và kiểm chứng để snapshot chạy thành công và đưa dữ liệu sang shadow table.

## 2. Kế hoạch điều tra & thực thi
### Phase 1: Research & Triage
- Kiểm tra dữ liệu thực tế trong DB nguồn `cdc_data_testing.failed_sync_logs` để xem có dữ liệu không.
- Kiểm tra đăng ký cấu hình của source_object 52 (`failed_sync_logs`) để xem PrimaryKeyField, PrimaryKeyType là gì.
- Kiểm tra source code của `snapshot_runner_handler.go` và `snapshot_runner_utils.go` (đặc biệt là logic phân trang của PostgreSQL).
- Xác định nguyên nhân gốc rễ (Root Cause): câu SQL query phân trang thực tế được sinh ra là gì, tại sao nó trả về 0 dòng.

### Phase 2: Sửa lỗi (qua Muscle Subagent)
- Muscle subagent sẽ thực hiện sửa lỗi trong code Go.
- Cập nhật unit test để bao phủ trường hợp lỗi này (nếu cần thiết).

### Phase 3: Kiểm chứng (Verification)
- Chạy unit tests.
- Chạy thử nghiệm thực tế (hoặc chạy thử flow snapshot thông qua trigger/command) để xem dữ liệu có được đồng bộ qua shadow table hay không.

## 3. Lịch trình công việc
- [ ] Task 1: Research cấu trúc bảng `failed_sync_logs` và cấu hình source_object 52.
- [ ] Task 2: Đọc code snapshot_runner logic phân trang PostgreSQL.
- [ ] Task 3: Xác định câu query SQL thực tế sinh ra và chạy thử để tìm nguyên nhân.
- [ ] Task 4: Propose fix và delegate cho Muscle subagent thực hiện sửa đổi.
- [ ] Task 5: Chạy unit tests và kiểm chứng end-to-end.
