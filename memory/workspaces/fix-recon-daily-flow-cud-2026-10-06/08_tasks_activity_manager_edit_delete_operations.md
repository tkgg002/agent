# Danh Sách Nhiệm Vụ: Hoàn Thiện Edit, Delete và Danh Mục Tác Vụ trên Activity Manager

---

## 1. PHÂN RÃ NHIỆM VỤ CHI TIẾT

### Giai đoạn 1: Backend CMS Service (`cdc-cms-service`)
- [x] **Task 1.1:** Mở rộng `ports.WorkerScheduleRepo` và struct `workerScheduleRepoGorm`: Bổ sung method `Delete(ctx context.Context, id int64) error`.
- [x] **Task 1.2:** Tạo CQRS Command `DeleteWorkerScheduleCommand` và Handler `DeleteWorkerScheduleHandler` trong `internal/app/commands/scheduler/delete_worker_schedule.go`.
- [x] **Task 1.3:** Nâng cấp `UpdateScheduleCommand`: Bổ sung trường `TargetTable *string` cho phép đổi bảng đích khi sửa lịch trình.
- [x] **Task 1.4:** Bổ sung method `Delete(c *fiber.Ctx) error` và cập nhật `WorkerScheduleUpdateRequest` trong `internal/api/scheduler/schedule_handler.go`.
- [x] **Task 1.5:** Đăng ký command handler trong `internal/server/server.go` và đăng ký route `dual("DELETE", admin, "/worker-schedule/:id", ...)` trong `internal/router/router.go`.
- [x] **Task 1.6:** Viết unit test cho Delete handler và verify build `go build ./cmd/server` đạt Exit 0.

### Giai đoạn 2: Frontend Web UI (`cdc-cms-web`)
- [x] **Task 2.1:** Cập nhật `ALL_OPERATIONS` và `opColors` trong `src/pages/ActivityManager.tsx`:
  - Loại bỏ: `drop-gin-index`, `create-default-columns`.
  - Bổ sung: `reconcile` (Đối soát dữ liệu), `bridge` (Đồng bộ cầu nối).
- [x] **Task 2.2:** Cập nhật cột Thao tác (`Table` Columns):
  - Tăng width lên `220px` (fixed: 'right').
  - Thêm nút Sửa (`EditOutlined`) và nút Xoá (`DeleteOutlined`) bọc `Popconfirm`.
- [x] **Task 2.3:** Xây dựng Modal và Form Chỉnh sửa Lịch trình (Edit Schedule Modal):
  - Load form values từ record được chọn (`source_count`, `confluence`, các tác vụ tiêu chuẩn).
  - Gửi request `PATCH /api/worker-schedule/:id` với updates.
- [x] **Task 2.4:** Thêm hàm `handleDeleteSchedule(record: Schedule)` gọi `DELETE /api/worker-schedule/:id`, hiển thị thông báo thành công và refresh data.
- [x] **Task 2.5:** Tái cấu trúc Triệt để Bộ chọn Cascading Dynamic Selector 5 Tầng cho Source Count:
  - Tầng 1 (Layer Type): `source` | `shadow` | `master`.
  - Tầng 2 (Connection): Lọc theo `role_type`, tự động gán và khóa `engine_type`, tự động gán `object_type` theo engine.
  - Tầng 3 (Container / Namespace): Đổi sang `<Select showSearch>` lấy dữ liệu thực tế từ `source_db`, `shadow_schema`, `master_schema`.
  - Tầng 4 (Object Name): `<Select showSearch>` lọc danh sách Table / Collection theo đúng Namespace.
  - Tầng 5 (Date Field): Tự động điền `timestamp_field` từ metadata của đối tượng đã chọn.
- [x] **Task 2.6:** Chạy `npm run build` (`tsc -b && vite build`) xác nhận không có lỗi biên dịch.

### Giai đoạn 3: Kiểm thử & Nghiệm thu Live
- [x] **Task 3.1:** Thao tác tạo thử một lịch trình `reconcile` hoặc `bridge` trên UI.
- [x] **Task 3.2:** Thao tác chỉnh sửa (Edit) lịch trình đó (đổi chu kỳ / ghi chú).
- [x] **Task 3.3:** Thao tác xoá (Delete) lịch trình đó và xác nhận DB sạch sẽ.
- [x] **Task 3.4:** Ghi nhận báo cáo vào `11_report_*.md` và audit log `05_progress.md`.
