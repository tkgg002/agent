# Báo cáo Phân tích: Hiệu năng Transmute và hỏng/thiếu Index trên Shadow Tables

## 1. Phát hiện & Đánh giá hiệu năng
Khi kiểm tra quá trình transmute cho `trans-his` ( shadow table là `shadow_core_trans_proxy_history.trans_his`), hệ thống mất tới 45.3 giây để sync một batch nhỏ.
Qua việc kiểm tra cấu trúc index của các bảng shadow trên database cục bộ (`gpay-postgres-shadow`), chúng tôi thấy:
- Bảng shadow `shadow_testexp.export_jobs` có chỉ mục `ux_export_jobs_source_id_active` (UNIQUE partial index `WHERE NOT _deleted`), nhưng **không có** chỉ mục không duy nhất `idx_export_jobs_source_id` trên trường `_source_id`.
- Khi transmuter thực hiện incremental hoặc heal sync, nó sẽ chạy câu lệnh:
  `SELECT ... WHERE _source_id IN (?)`
- Vì điều kiện truy vấn là `WHERE _source_id IN (?)` (không có điều kiện `WHERE NOT _deleted`), Postgres không thể sử dụng index partial `ux_..._source_id_active` được. Nó bắt buộc phải làm Full Table Scan trên bảng shadow. Với các bảng lớn chứa hàng triệu hoặc hàng trăm triệu bản ghi như `trans_his`, Full Table Scan sẽ tốn hàng chục giây đến vài phút (dẫn đến Context Timeout).

## 2. Điểm yếu trong cơ chế tạo index động hiện tại
Hàm `ensureShadowSourceIDIndex` trong `transmuter.go` kiểm tra xem index `idx_<tableName>_source_id` đã tồn tại chưa bằng query:
```sql
SELECT COUNT(*) FROM pg_indexes WHERE schemaname = ? AND tablename = ? AND indexname = ?
```
1. **Lỗi khi Index bị INVALID:**
   - Trong quá trình tạo index concurrently (`CREATE INDEX CONCURRENTLY`), nếu luồng bị ngắt kết nối hoặc gặp timeout, PostgreSQL vẫn ghi nhận index đó trong `pg_indexes` nhưng đánh dấu nó là hỏng (`indisvalid = false`).
   - Vì query trên chỉ đếm sự tồn tại của tên index, `count` sẽ trả về `1` (đã tồn tại). Transmuter nghĩ index đã được tạo thành công nên sẽ bỏ qua không tạo lại.
   - Nhưng Postgres Planner không bao giờ sử dụng một index ở trạng thái `INVALID`. Do đó, các batch chạy tiếp theo vẫn tiếp tục bị Full Table Scan.
2. **Không có cơ chế dọn dẹp (Drop):**
   - Khi phát hiện index bị hỏng, transmuter không có cơ chế `DROP` đi để build lại từ đầu.
3. **Chưa tạo sẵn lúc khởi tạo bảng:**
   - Tại thời điểm `schema_adapter.go` hoặc `schema_manager.go` tạo shadow table, các index `idx_<tableName>_source_id` chưa được tạo sẵn, dẫn đến lag ở lần chạy đầu tiên.

## 3. Giải pháp tối ưu đề xuất
1. Thay đổi SQL kiểm tra trong `ensureShadowSourceIDIndex` để kiểm tra độ hợp lệ `indisvalid = true` từ `pg_index`.
2. Nếu không tìm thấy index hợp lệ, nhưng có index invalid, tiến hành `DROP INDEX CONCURRENTLY IF EXISTS` trước rồi mới chạy `CREATE INDEX CONCURRENTLY`.
3. Tạo sẵn index `idx_<tableName>_source_id` ngay khi khởi tạo shadow table ở cả `schema_adapter.go` (CDC update/alter) và `schema_manager.go` / `schema_manager_bk.go` (Sink worker table creation).
