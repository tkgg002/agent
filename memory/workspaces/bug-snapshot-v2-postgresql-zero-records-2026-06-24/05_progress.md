# Progress Log: bug-snapshot-v2-postgresql-zero-records-2026-06-24

## 1. Phân tích Gốc rễ (Root Cause) lỗi vi phạm quy trình Governance
- **Lỗi vi phạm**: Tự thiết kế một logic đồng bộ dữ liệu PostgreSQL Snapshot V2 mới (`scanPGXRows`, kiểm tra và parse `PrimaryKeyType` phức tạp dựa trên registry) thay vì tái sử dụng cơ chế schema-less shadow table phẳng (lưu vào cột `_raw_data`) đã được chuẩn hóa và hoạt động tốt của MongoDB.
- **Nguyên nhân gốc rễ**: 
  1. Chưa thấu hiểu sâu sắc triết lý thiết kế của hệ thống Core: Toàn bộ shadow tables (bao gồm cả PostgreSQL shadow) được thiết kế schema-less phẳng để chứa dữ liệu JSON thô, không cần khớp schema chặt chẽ.
  2. Over-engineering logic phân trang & scan: Cố gắng map kiểu dữ liệu Postgres nguyên bản sang kiểu Go, dẫn đến lỗi "operator does not exist: bigint > text" khi phân trang và lỗi "Nil Pointer Dereference Panic" do snapshot envelope thiếu metadata.
  3. Thứ tự ưu tiên giải mã DSN bị đảo ngược: Gọi `buildDSNFromFields` trước khi giải mã `crypto.DecryptAES(conn.SecretRef)`, dẫn đến việc sử dụng DSN thiếu credentials và gây lỗi SASL authentication failed trên môi trường thực tế.
- **Biện pháp khắc phục**: Đơn giản hóa toàn bộ: PostgreSQL snapshot sẽ query dữ liệu dưới dạng JSON thô (`row_to_json`), đưa thẳng qua `HandleRaw` để đồng bộ giống hệt MongoDB. Sắp xếp lại thứ tự giải mã DSN.

## 2. Nhật ký tiến độ
- `[2026-06-24T09:22:15+07:00] [Brain:Antigravity]` Khởi động phiên làm việc. Đọc lessons.md và active_plans.md.
- `[2026-06-24T09:22:15+07:00] [Brain:Antigravity]` Xác nhận workspace `bug-snapshot-v2-postgresql-zero-records-2026-06-24` đã được khởi tạo.
- `[2026-06-24T09:40:00+07:00] [Brain:Antigravity]` Cập nhật kế hoạch dựa trên phản hồi của user (bảng failed_sync_logs có 5 rows dữ liệu). Cử subagent `research` viết và chạy script debug.
- `[2026-06-24T09:47:00+07:00] [Brain:Antigravity]` Xác định nguyên nhân lỗi panic pointer dereference nil của `event.Data.Source` trong `event_handler.go`.
- `[2026-06-24T09:58:00+07:00] [Muscle:Antigravity]` Thực hiện sửa đổi code trong `event_handler.go` và `snapshot_runner_handler.go` (bản cũ).
- `[2026-06-24T10:10:00+07:00] [Brain:Antigravity]` Đọc và phân tích phản hồi của User về việc tái sử dụng luồng Mongo (đóng gói phẳng `row_to_json`, không check PK type) và lỗi connect SASL auth.
- `[2026-06-24T10:15:00+07:00] [Brain:Antigravity]` Tìm ra lỗi thứ tự nạp DSN trong `resolveSourceURIFromConn` gây mất user/password.
- `[2026-06-24T10:19:00+07:00] [Brain:Antigravity]` Lập Implementation Plan mới để đơn giản hóa hoàn toàn logic snapshot và sửa lỗi DSN. Cập nhật `task.md` và `05_progress.md` (bao gồm Root Cause Analysis). Chờ User phê duyệt kế hoạch trước khi tiến hành Execution.
- `[2026-06-24T10:25:00+07:00] [Muscle:Antigravity]` Bắt đầu phiên làm việc của Muscle. Đọc spec và chuẩn bị chỉnh sửa DSN, đơn giản hóa Postgres Snapshot sang `row_to_json` và cập nhật Unit Tests.
- `[2026-06-24T10:35:00+07:00] [Muscle:Antigravity]` Đã thực hiện chỉnh sửa thành công:
  1. Thay đổi thứ tự check DSN trong `metadata_registry_service.go`.
  2. Xóa scan cũ, thêm `scanPGXRowsAsJSON` vào `snapshot_runner_utils.go`.
  3. Áp dụng query `row_to_json` và convert `lastSeen` trong `snapshot_runner_handler.go`.
  4. Cập nhật `mockRows.Scan` và queryFunc trong `snapshot_runner_test.go`.
- `[2026-06-24T10:40:00+07:00] [Muscle:Antigravity]` Loại bỏ các unused imports (`encoding/json`, `pgtype`) trong `snapshot_runner_utils.go`.
- `[2026-06-24T10:45:00+07:00] [Brain:Antigravity]` Tiến hành chạy unit tests trên codebase, tất cả các test của metadata registry service (`internal/service/source`) và snapshot runner (`internal/handler/orchestration`) đều **PASS** 100%.
- `[2026-06-24T10:55:00+07:00] [Brain:Antigravity]` Tiếp nhận feedback mới từ User về lỗi kết nối của object 55 và yêu cầu fallback default_schema thay vì set cứng "public". Khởi tạo 01_requirements, 02_plan, 03_implementation, 08_tasks, 09_tasks_solution cho task default_schema. Cử Muscle thực thi.
- `[2026-06-24T11:00:00+07:00] [Muscle:Antigravity]` Thực hiện thành công các thay đổi: tạo `buildDSNFromFieldsPatched` để chèn `search_path` trong DSN Postgres, sửa logic schema fallback trong `snapshot_runner_handler.go` và cập nhật unit test.
- `[2026-06-24T11:02:00+07:00] [Muscle:Antigravity]` Khắc phục lỗi compile thiếu import `"encoding/json"`.
- `[2026-06-24T11:05:00+07:00] [Brain:Antigravity]` Chạy unit tests kiểm chứng lần cuối. Tất cả các test đều **PASS 100%**. Code đã sẵn sàng bàn giao.

## 3. Trạng thái các bước
- [x] Task 1: Khắc phục lỗi Panic pointer dereference nil trong `event_handler.go` (Đã hoàn thành)
- [x] Task 2: Khắc phục lỗi kết nối Postgres (Sửa thứ tự giải mã DSN) (Đã hoàn thành)
- [x] Task 3: Đơn giản hóa Postgres Snapshot sang `row_to_json` (Đã hoàn thành)
- [x] Task 4: Cập nhật Unit Tests để bao phủ query `row_to_json` (Đã hoàn thành)
- [x] Task 5: Fallback Default Schema từ Connection Registry (Đã hoàn thành)
- [ ] Task 6: Chạy kiểm thử thủ công và verify trên môi trường thực tế (Chờ kiểm chứng thực tế)
