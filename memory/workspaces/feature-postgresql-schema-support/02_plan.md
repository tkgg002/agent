# Plan - PostgreSQL Schema Support

Kế hoạch triển khai chi tiết:

1. **Backend (`cdc-cms-service`)**:
   - Chỉnh sửa `internal/infra/persistence/source/source_object_v2_sync.go` (hàm `SyncFromLegacyTx`):
     - Truy vấn `default_database` từ `connection_registry` của connector PostgreSQL liên kết.
     - Truyền `source_schema = entry.SourceDB` và `source_namespace = entry.SourceDB`.
     - Update câu SQL INSERT / UPDATE với tham số `source_schema`.
2. **Frontend (`cdc-cms-web`)**:
   - Chỉnh sửa `src/pages/TableRegistry.tsx` (Form Register):
     - Chuyển đổi label dynamic dựa trên `source_type` (Postgres vs Mongo).
     - Cho phép chỉnh sửa `source_db` (nhãn Source Schema) khi là Postgres (mặc định `'public'`).
     - Tự động gán PK field/type tương ứng khi chọn Postgres connector.
3. **Verify**:
   - Chạy manual test và kiểm tra database sync V2.
