# Kế hoạch tối ưu hiệu năng Transmute và sửa lỗi hỏng/thiếu Index trên Shadow Tables

## 1. Mục tiêu & Bối cảnh
Khi transmuter chạy incremental sync hoặc heal cho các bảng shadow lớn (ví dụ `shadow_core_trans_proxy_history.trans_his`), thời gian xử lý rất chậm (45.3 giây) do PostgreSQL phải thực hiện Full Table Scan.
Nguyên nhân là thiếu index không duy nhất `idx_<table_name>_source_id` trên trường `_source_id`.
Mặc dù `ensureShadowSourceIDIndex` cố gắng tạo index này ở chế độ `CONCURRENTLY` trong nền, nhưng:
1. Hàm kiểm tra sự tồn tại của index (`pg_indexes`) chỉ kiểm tra tên index mà không kiểm tra độ hợp lệ của index (`indisvalid`). Khi một index bị lỗi lúc build (ở trạng thái `INVALID`), hệ thống coi như index đã tồn tại và bỏ qua không tạo lại. Planner của Postgres cũng không thể dùng index này.
2. Index không được khởi tạo sẵn từ đầu khi tạo shadow table, dẫn đến lag trong lần chạy đầu tiên.

### Giải thích thiết kế: Tại sao không dùng `_gpay_id` để truy vấn?
- **Nguồn kích hoạt (CDC / Reconciliation):** Khi có sự kiện CDC (từ MongoDB) bắn sang, hoặc khi hệ thống đối soát (Reconciliation) phát hiện dòng lệch cần đồng bộ/chữa lành, thông tin định danh duy nhất mà ta nhận được từ nguồn là MongoDB `_id` (được lưu ở cột `_source_id` trong shadow).
- **Tính chất của `_gpay_id`:** `_gpay_id` là Sonyflake ID được tự sinh khi ghi dữ liệu vào Postgres. Hệ thống nguồn (MongoDB) hoàn toàn không biết về ID này.
- **Bắt buộc truy vấn theo `_source_id`:** Do worker chỉ nhận được danh sách `_source_id` cần đồng bộ từ CDC hoặc báo cáo đối soát, worker bắt buộc phải truy vấn dữ liệu thô từ shadow bằng `WHERE _source_id IN (?)` chứ không thể truy vấn bằng `_gpay_id` (vì chưa biết map tương ứng).
- **Giải pháp tối ưu:** Tạo index không partial `idx_<table_name>_source_id` trên cột `_source_id` ở bảng shadow để tối ưu hóa truy vấn này xuống dưới vài mili-giây thay vì quét toàn bộ bảng.

## 2. Các thay đổi đề xuất

### centralize-data-service

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Sửa hàm `ensureShadowSourceIDIndex` để kiểm tra độ hợp lệ của index từ bảng `pg_index` bằng cách kiểm tra điều kiện `indisvalid = true`.
- Nếu index chưa tồn tại hoặc bị hỏng (`indisvalid = false`), tiến hành drop index cũ bằng `DROP INDEX CONCURRENTLY IF EXISTS` (nếu tồn tại) và tạo lại bằng `CREATE INDEX CONCURRENTLY` trong background goroutine.

#### [MODIFY] [schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go)
- Cập nhật hàm `EnsureCDCColumnsInSchema` để tạo sẵn index `idx_%s_source_id` trên cột `_source_id` khi cấu hình/nâng cấp schema cho shadow table.

#### [MODIFY] [schema_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker/schema_manager.go)
- Thêm bước tạo index `idx_<tableName>_source_id` ngay sau khi tạo shadow table.

#### [MODIFY] [schema_manager.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/sinkworker_bk/schema_manager.go)
- Thêm bước tạo index `idx_<tableName>_source_id` tương tự phiên bản backup.

---

## 3. Kế hoạch xác minh

### Kiểm thử tự động
- Chạy các unit test hiện có để đảm bảo không xảy ra lỗi regression:
  - `go test ./...`

### Xác minh thủ công
1. Khởi động lại hoặc kích hoạt chạy thử transmuter.
2. Kiểm tra trực tiếp trong PostgreSQL xem index đã được tạo và ở trạng thái hợp lệ chưa:
   - `SELECT indisvalid FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_class t ON t.oid = i.indrelid JOIN pg_namespace n ON n.oid = t.relnamespace WHERE n.nspname = 'shadow_test_ms' AND t.relname = 'merchants' AND c.relname = 'idx_merchants_source_id';`
3. So sánh hiệu năng trước và sau khi có index hợp lệ.
