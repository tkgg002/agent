# Báo cáo thay đổi mã nguồn - Khắc phục logic tự sửa đổi index trong Transmuter & Bổ sung đề xuất trên UI

## 1. Danh sách file đã thay đổi

| Đường dẫn file | Mô tả thay đổi | Số dòng ảnh hưởng |
| :--- | :--- | :--- |
| `centralized-data-service/internal/service/master/transmuter.go` | Loại bỏ các câu lệnh drop/create index `CONCURRENTLY` ngầm trong `ensureShadowSourceIDIndex`, thay bằng log `Warn` và cache trạng thái. | ~70 dòng |
| `centralized-data-service/internal/service/master/transmuter_index_test.go` | Loại bỏ mock mong đợi câu lệnh Exec drop/create index và check existence cho test case missing và invalid. | ~50 dòng |
| `centralized-data-service/internal/service/governance/index_manager.go` | Thêm struct `IndexRecommendation` và hàm `GetRecommendations` để kiểm tra đề xuất thiếu index `_source_id` và `_deleted`. | ~55 dòng |
| `centralized-data-service/internal/service/governance/index_manager_test.go` | Bổ sung unit test `TestIndexManager_GetRecommendations` kiểm tra toàn bộ 4 trường hợp logic recommendations. | ~70 dòng |
| `centralized-data-service/internal/handler/governance/index_handler.go` | Trong `HandleIntrospectIndexes`, gọi `GetRecommendations` và trả về recommendations trong payload NATS. | ~15 dòng |

## 2. Chi tiết các thay đổi chính

### 2.1 Transmuter
- **Mục tiêu:** Tránh lock-storm và deadlock khi transmuter tự động chạy DDL `CREATE INDEX CONCURRENTLY` tại runtime.
- **Thực thi:** Gỡ bỏ hoàn toàn logic goroutine chạy Exec DDL. Hàm `ensureShadowSourceIDIndex` giờ chỉ check index trong database, nếu chưa có thì ghi log `Warn` hướng dẫn người dùng tự tạo index trước đó thông qua UI và lưu cache kiểm tra để tránh spam log.

### 2.2 Index Manager & Recommendations
- **Mục tiêu:** Cho phép người dùng kiểm tra và tạo index chủ động từ CMS UI trước khi chạy sync.
- **Thực thi:**
  - Struct `IndexRecommendation` định nghĩa cấu trúc đề xuất index gồm: `index_name`, `columns`, `is_unique`, `is_partial`, `where_clause`, `description`.
  - Hàm `GetRecommendations` kiểm tra danh sách index hiện tại của bảng shadow. Nếu thiếu index core trên `_source_id` (idx_`<table_name>`_source_id) hoặc thiếu partial index trên `_deleted` (idx_`<table_name>`_deleted_partial), hàm sẽ trả về đề xuất tương ứng.
  - Handler NATS cập nhật để đính kèm các đề xuất này vào trường `recommendations` của command `introspect-indexes`.

## 3. Kết quả kiểm thử
- Chạy unit test thành công cho các package:
  - `internal/service/master/...` - PASS
  - `internal/service/governance/...` - PASS
  - `internal/handler/...` - PASS
