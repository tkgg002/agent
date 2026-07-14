# Danh sách Task chi tiết - Tối ưu hiệu năng Transmute và sửa lỗi thiếu/hỏng Index trên Shadow Tables

## Task: Tối ưu hiệu năng Transmute và sửa lỗi thiếu/hỏng Index trên Shadow Tables
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: centralized-data-service
- **Mô tả**: Triển khai các sửa đổi trên Golang source code để kiểm tra độ hợp lệ của index, sửa đổi logic tự động tạo index trong schema adapter và schema manager.
- **Trạng thái**: [/] DOING

### [Context]
- Current state: Đã được phê duyệt kế hoạch (implementation plan approved). Bảng `trans-his` chạy transmute rất chậm do thiếu index `idx_<tableName>_source_id`. Hàm `ensureShadowSourceIDIndex` hiện tại chỉ check tên trong `pg_indexes` mà không check `indisvalid`.
- Dependencies: `internal/service/master/transmuter.go`, `internal/service/shadow/schema_adapter.go`, `internal/sinkworker/schema_manager.go`, `internal/sinkworker_bk/schema_manager.go`.
- ADR liên quan: Không có.
- Logs/Error: Transmute batch sync cho `trans-his` mất tới 45.3 giây.

### [Definition of Done]
- [ ] Điều kiện 1: Sửa đổi `ensureShadowSourceIDIndex` trong `transmuter.go` để kiểm tra độ hợp lệ `indisvalid = true` từ `pg_index` và tự động `DROP INDEX CONCURRENTLY IF EXISTS` + `CREATE INDEX CONCURRENTLY` nếu index hỏng/thiếu.
- [ ] Điều kiện 2: Thêm logic tạo index `idx_<tableName>_source_id` tại `schema_adapter.go` (`EnsureCDCColumnsInSchema`) và `schema_manager.go` / `schema_manager_bk.go` (`createShadowTable`).
- [ ] **[QA Gate]**: Chạy toàn bộ test suite của dự án bằng lệnh `go test ./...` và đảm bảo kết quả PASS.
- [ ] **[Security Gate]**: Rà soát xem có rò rỉ secret hoặc API key nào không trước khi commit.
- [ ] Blast radius verified: Không ảnh hưởng đến các service khác ngoài centralized-data-service.
- [ ] Model Tracking: Ghi nhận task vào `05_progress_transmute_slow_index.md` với tag model.
