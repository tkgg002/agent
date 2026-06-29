# connection_failed_fix (failed_sync_logs scan-fields)

> Date: 2026-06-23
> Trigger: nats_command SQL source returned 0 columns or connection failed khi thực hiện scan-fields cho bảng failed_sync_logs thuộc pg_dev
> Status: ✅ RESOLVED

## 1. Symptom
- **Error**: `nats_command: SQL source returned 0 columns or connection failed` khi gửi lệnh `cdc.cmd.scan-fields` qua NATS cho connection `pg_dev`, target table `shadow_pg_dev.failed_sync_logs`.
- **Tần suất**: 100% xảy ra khi chạy lệnh.
- **Scope**: CDC Worker (`centralized-data-service`).

## 2. Iteration Timeline
- **23:05** Discovery: Kiểm tra workspace, thiết lập context.
- **23:06** Hypothesis 1: Cổng Postgres `5435` không kết nối được hoặc registry mapping sai.
  - Viết `test_conn.go` và `read_registry.go` để verify. Kết quả: host kết nối bình thường, registry ID 52 trỏ đúng `pg_dev`.
- **23:06** Root cause identified: Tiến trình worker cũ chạy nền (PID 70313) đã hoạt động từ ngày 4 tháng 6, trước khi tệp `config-local.yml` được cập nhật cấu hình override DSN cho `pg_dev`.
- **23:06** Fix applied: Kill tiến trình worker cũ và khởi chạy tiến trình worker mới để nạp config override mới nhất.
- **23:07** Verified: Chạy `pub_scan.go` gửi NATS command. Worker xử lý thành công, tự động map 20 rules mới cho bảng `failed_sync_logs`.

## 3. Root Cause
CDC Worker khi khởi chạy sẽ nạp tệp cấu hình override local `config/config-local.yml` và lưu trữ connection pool. Do tiến trình worker chạy nền cũ (PID 70313) hoạt động liên tục từ trước khi file config này được cập nhật, worker không có thông tin kết nối thực tế (override DSN) cho `pg_dev`, dẫn tới kết nối thất bại khi nhận lệnh NATS.

## 4. Fix
- Không có thay đổi mã nguồn trong codebase để giảm thiểu Blast Radius (Simplicity First).
- Chỉ thực hiện ops: restart worker nền.
  ```bash
  kill 70313
  nohup go run cmd/worker/main.go > worker.log 2>&1 &
  ```

## 5. Verify
- **Before**: Lệnh `scan-fields` báo lỗi `SQL source returned 0 columns or connection failed`, 0 rules được cập nhật.
- **After**: Lệnh `scan-fields` hoàn thành thành công:
  ```json
  {"status":"success","message":"success","data":{"rules_count":20}}
  ```
- **Reduction**: 100% lỗi kết nối được xử lý triệt để.

## 6. Related lessons
- **GP-244** ( lessons.md): Không restart tiến trình worker/service nền dẫn đến không nạp tệp cấu hình (override config) mới nhất.

## 7. Follow-ups
- Đề xuất bổ sung cơ chế hot-reload cấu hình kết nối khi phát hiện thay đổi trong registry hoặc tệp config để tránh phải restart service thủ công trong tương lai.
