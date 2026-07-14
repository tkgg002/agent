# Yêu cầu: Tối ưu hiệu năng Transmute và sửa lỗi thiếu/hỏng Index trên Shadow Tables

## 1. Bối cảnh
Khi chạy transmute cho `core-trans-proxy-history-service.trans-his` ( shadow table là `shadow_core_trans_proxy_history.trans_his`), thời gian xử lý tốn tới 45.3s cho batch chỉ có vài records.
Nguyên nhân gốc rễ:
- Thiếu index `idx_trans_his_source_id` trên trường `_source_id` trong shadow table (hoặc index này bị `INVALID` do quá trình tạo bất đồng bộ trước đó bị gián đoạn/kẹt).
- Cơ chế kiểm tra sự tồn tại của index trong `ensureShadowSourceIDIndex` (file `transmuter.go`) chỉ đếm số lượng index trong `pg_indexes` mà không kiểm tra độ hợp lệ (`indisvalid`). Khi index ở trạng thái `INVALID`, hệ thống bỏ qua không tạo lại, nhưng PostgreSQL Planner không thể sử dụng index này, dẫn đến Full Table Scan trên bảng shadow lớn (hàng chục triệu records).
- Tại thời điểm khởi tạo shadow table (trong `schema_adapter.go` và `schema_manager.go`), index không duy nhất trên `_source_id` (tức `idx_<table_name>_source_id`) chưa được khai báo tạo sẵn, dẫn đến lần đầu tiên thực hiện incremental/heal sync sẽ bị lag rất nặng vì phải tạo index động.

## 2. Yêu cầu chi tiết
- **Yêu cầu 1:** Sửa logic kiểm tra index trong `ensureShadowSourceIDIndex` (`internal/service/master/transmuter.go`). Cần kiểm tra trạng thái hoạt động (`indisvalid = true`) từ `pg_index` thay vì chỉ đếm trong `pg_indexes`.
- **Yêu cầu 2:** Nếu index tồn tại nhưng ở trạng thái `INVALID`, tiến hành drop index cũ bằng `DROP INDEX CONCURRENTLY IF EXISTS` (trong goroutine nền để tránh block transmuter), sau đó tạo lại bằng `CREATE INDEX CONCURRENTLY`.
- **Yêu cầu 3:** Khai báo và tạo sẵn index `idx_<tableName>_source_id` ngay khi khởi tạo hoặc cập nhật cấu trúc shadow table trong `internal/service/shadow/schema_adapter.go` (hàm `EnsureCDCColumnsInSchema`).
- **Yêu cầu 4:** Khai báo và tạo sẵn index `idx_<tableName>_source_id` trong `internal/sinkworker/schema_manager.go` và `internal/sinkworker_bk/schema_manager.go` (hàm `createShadowTable`).
- **Yêu cầu 5:** Kiểm thử, chạy linter quy trình và đảm bảo chất lượng.
