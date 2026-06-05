# Kế hoạch thực thi (High-level Plan): Snapshot Monitor

## Chiến lược
- **Phân tách trách nhiệm (CQRS)**: Tạo luồng đọc (Read Model) riêng biệt, hoàn toàn không chạm vào luồng ghi (Command) của Snapshot Worker.
- **Tiêu chuẩn Frontend**: Dùng React + Ant Design, tái sử dụng các layout và Table design pattern từ ActivityLog.

## Các bước thực thi chính
1. **Khởi tạo Read Models & CQRS Queries (Backend)**
   - Tạo file `snapshot_progress_read_models.go`.
   - Tạo interface `SnapshotProgressReader`.
   - Định nghĩa `ListSnapshotProgressQuery`.
2. **Triển khai Repository (Backend)**
   - Tạo `snapshot_progress_read_repo_gorm.go` trong thư mục `infra/persistence`.
   - Viết GORM query join giữa bảng `snapshot_progress` và `cdc_source_objects`.
3. **Mở Endpoint (Backend)**
   - Tạo `snapshot_progress_handler.go` ở `infra/http` (hoặc `api`).
   - Đăng ký route `GET /api/snapshot-progress`.
4. **Phát triển Màn hình Monitor (Frontend)**
   - Tạo `src/pages/SnapshotMonitor.tsx` với Ant Design Table.
   - Thêm route vào `src/App.tsx`.
   - Bổ sung menu item vào mục "Operate".
5. **Liên kết luồng (Frontend)**
   - Sửa cột Operation/Details trong `ActivityLog.tsx` để render Link trỏ về `/snapshot-monitor`.
6. **Kiểm thử**
   - Đảm bảo Backend build pass và test pass.
   - Đảm bảo Frontend compile thành công và routing chính xác.
