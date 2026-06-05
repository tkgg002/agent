# Danh sách Task: Snapshot Monitor

## M1. Khởi tạo Read Models & Filters (Backend)
- [ ] Tạo file `internal/app/queries/snapshot_progress_read_models.go`.
- [ ] Khai báo struct `SnapshotProgressRow`.
- [ ] Tạo file `internal/app/queries/list_snapshot_progress.go`.
- [ ] Định nghĩa `SnapshotProgressFilter`, `SnapshotProgressReader` interface và `ListSnapshotProgressQuery`.

## M2. Triển khai Repository & Handler (Backend)
- [ ] Tạo `internal/infra/persistence/snapshot_progress_read_repo_gorm.go`.
- [ ] Viết lệnh GORM Join bảng `cdc_system.snapshot_progress` và `cdc_source_objects`.
- [ ] Khai báo endpoint HTTP handler trong `internal/api/snapshot_progress_handler.go`.
- [ ] Đăng ký Route `GET /api/snapshot-progress` vào `internal/router/router.go`.
- [ ] Gắn Dependency Injection (DI) tại `internal/server/server.go`.

## M3. Tạo trang Monitor (Frontend)
- [x] Tạo component `src/pages/SnapshotMonitor.tsx`.
- [x] Gọi api `cmsApi.get('/api/snapshot-progress')`.
- [x] Render Table với các cột quy định, format timestamp.
- [x] Parse param query `source_database` và `source_table` qua thư viện `react-router-dom` (useLocation).

## M4. Liên kết từ Activity Log (Frontend)
- [x] Sửa đổi `<Route>` trong `src/App.tsx`.
- [x] Thêm Button "View Progress" vào `src/pages/ActivityLog.tsx` đối với operation `snapshot.v2`.
- [x] Chạy lệnh build backend & frontend để đảm bảo an toàn.
