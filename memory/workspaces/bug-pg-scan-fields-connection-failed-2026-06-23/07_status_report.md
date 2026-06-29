# Status Report: bug-pg-scan-fields-connection-failed-2026-06-23

Báo cáo trạng thái và kết quả xử lý lỗi kết nối `pg_dev` trong quá trình `scan-fields`.

## 1. Thông tin chung
- **Thời gian xác minh**: 2026-06-23
- **Mục tiêu**: Khắc phục lỗi `SQL source returned 0 columns or connection failed` xảy ra khi trigger NATS command `scan-fields` cho bảng `cdc_data_testing.failed_sync_logs` thông qua registry `pg_dev`.
- **Trạng thái**: Đã được khắc phục hoàn toàn và kiểm thử thành công qua NATS.

## 2. Chi tiết vấn đề & Giải pháp
### Nguyên nhân lỗi
- **Bối cảnh**: Tiến trình CDC worker (`centralized-data-service` worker) chạy nền trong hệ thống (PID 70313) đã hoạt động liên tục từ ngày 4 tháng 6 năm 2026.
- **Vấn đề**: File cấu hình override local `config/config-local.yml` chứa cấu hình connection override cho `pg_dev` (chỉ định DSN cụ thể sang cổng local `5435`) được chỉnh sửa sau thời điểm worker này khởi chạy.
- **Root Cause**: Do worker cũ không tự động hot-reload config, nó tiếp tục sử dụng cấu hình cũ không có DSN override cho `pg_dev`. Khi nhận command `scan-fields`, worker kết nối thất bại tới source DB và trả về lỗi `SQL source returned 0 columns or connection failed`.

### Giải pháp áp dụng
1. **Kiểm tra kết nối trực tiếp**: Viết script test kết nối độc lập `test_conn.go` xác nhận PostgreSQL chạy trên cổng 5435 phản hồi tốt và tìm thấy 21 columns của bảng `failed_sync_logs`.
2. **Khởi động lại tiến trình worker**: Thực hiện kill tiến trình worker cũ (PID 70313) và chạy worker mới ở background (`nohup go run cmd/worker/main.go > worker.log 2>&1 &`) để nạp cấu hình override mới nhất.
3. **Trigger command**: Dùng script `pub_scan.go` gửi trực tiếp command qua NATS subject `cdc.cmd.scan-fields`. Worker mới nạp cấu hình thành công đã xử lý yêu cầu hoàn tất và lưu trữ 20 mapping rules vào database.

## 3. Các file thay đổi & Dòng code chênh lệch
Vì đây là task xử lý vận hành (ops/restart service) và nạp cấu hình local, **không có file source code nào trong codebase dự án bị thay đổi**.

| File | Loại thay đổi | Số dòng thêm (+) | Số dòng xóa (-) | Tổng dòng chênh lệch (Net) |
|---|---|---|---|---|
| *Không có file codebase* | - | 0 | 0 | 0 |
| `agent/memory/workspaces/bug-pg-scan-fields-connection-failed-2026-06-23/scratch/test_conn.go` | [NEW] Script test DB connection | 46 | 0 | +46 |
| `agent/memory/workspaces/bug-pg-scan-fields-connection-failed-2026-06-23/scratch/read_registry.go` | [NEW] Script check registry db | 58 | 0 | +58 |
| `agent/memory/workspaces/bug-pg-scan-fields-connection-failed-2026-06-23/scratch/pub_scan.go` | [NEW] Script trigger NATS command | 81 | 0 | +81 |

## 4. Kết quả kiểm thử & Xác minh thực tế
- Bằng chứng kiểm thử chi tiết và kết quả thực tế của các Test Cases được lưu trữ đầy đủ tại [06_validation.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/bug-pg-scan-fields-connection-failed-2026-06-23/06_validation.md).
- Kết quả NATS response trả về thành công:
  ```json
  {"status":"success","message":"success","data":{"rules_count":20}}
  ```
