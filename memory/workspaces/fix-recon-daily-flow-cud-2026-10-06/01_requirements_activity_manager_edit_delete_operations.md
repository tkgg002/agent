# Yêu Cầu Kỹ Thuật: Hoàn Thiện Edit, Delete và Danh Mục Tác Vụ trên Activity Manager

---

## 1. MỤC TIÊU & BỐI CẢNH
Người dùng yêu cầu hoàn thiện trang Quản lý tác vụ ngầm `/activity-manager` (`cdc-cms-web/src/pages/ActivityManager.tsx`):
1. **Bổ sung tính năng Sửa (Edit):** Cho phép Quản trị viên chỉnh sửa lịch trình đã tạo (Chu kỳ phút, Giờ chạy hằng ngày, Trạng thái bật/tắt, Ghi chú, Tham số `params` cho Source Count và Confluence).
2. **Bổ sung tính năng Xoá (Delete):** Cho phép Quản trị viên xoá bỏ các lịch trình không còn sử dụng với Popconfirm cảnh báo an toàn.
3. **Chuẩn hóa Danh mục Tác vụ (`ALL_OPERATIONS`):**
   - **Loại bỏ:** `drop-gin-index` (Xoá GIN Index) và `create-default-columns` (Tạo field mặc định) khỏi danh sách lựa chọn.
   - **Bổ sung:** `reconcile` (Đối soát dữ liệu) và `bridge` (Đồng bộ cầu nối) vào danh sách lựa chọn.

---

## 2. PHÂN TÍCH YÊU CẦU CHI TIẾT THEO TẦNG

### A. Tầng Backend CMS (`cdc-cms-service`)
1. **API Xoá Lịch Trình (`DELETE /api/worker-schedule/:id`):**
   - Hiện tại router chỉ có `GET`, `POST`, `PATCH`. Chưa có `DELETE /worker-schedule/:id`.
   - Cần bổ sung method `Delete(ctx context.Context, id int64) error` trong `ports.WorkerScheduleRepo` và `worker_schedule_repo_gorm.go`.
   - Tạo CQRS Command `DeleteWorkerScheduleCommand` và `DeleteWorkerScheduleHandler` xử lý nghiệp vụ xoá.
   - Bổ sung handler `Delete(c *fiber.Ctx) error` trong `schedule_handler.go`.
   - Đăng ký route `dual("DELETE", admin, "/worker-schedule/:id", h.Scheduler.WorkerSchedule.Delete)` trong `router.go`.
2. **Nâng cấp API Chỉnh Sửa (`PATCH /api/worker-schedule/:id`):**
   - Đảm bảo `WorkerScheduleUpdateRequest` và `UpdateScheduleCommand` cho phép cập nhật: `interval_minutes`, `is_enabled`, `notes`, `run_at_time`, `params`, `target_table`.

### B. Tầng Frontend Web (`cdc-cms-web`)
1. **Chuẩn hóa Danh sách `ALL_OPERATIONS`:**
   - `source_count`: "Đếm bản ghi ngày tạo (Source Count)" (color: `cyan`)
   - `confluence`: "Báo cáo Confluence Daily (Confluence Report)" (color: `blue`)
   - `reconcile`: "Đối soát dữ liệu (Reconcile)" (color: `green`)
   - `bridge`: "Đồng bộ cầu nối (Bridge)" (color: `orange`)
   - `transform`: "Chuyển đổi field (Transform)" (color: `purple`)
   - `field-scan`: "Quét field mới (Field Scan)" (color: `geekblue`)
   - `partition-check`: "Kiểm tra partition" (color: `lime`)
2. **Cột Thao tác (`Table` Columns):**
   - Tăng độ rộng cột `action` từ `110px` lên `200px` (fixed: 'right').
   - Bao gồm 3 nút hành động:
     - **Chạy ngay** (`PlayCircleOutlined`): Gọi `PATCH /api/worker-schedule/:id` với `{ run_now: true }`.
     - **Sửa** (`EditOutlined`): Mở Modal Chỉnh sửa với toàn bộ thông tin lịch trình được load sẵn vào form.
     - **Xoá** (`DeleteOutlined`, danger): Bọc trong `Popconfirm` xác nhận an toàn, gọi `DELETE /api/worker-schedule/:id`.
3. **Modal Form Chỉnh Sửa (Edit Schedule Modal):**
   - Hiển thị tiêu đề động: `"Chỉnh sửa lịch trình #" + record.id + " (" + record.operation + ")"`.
   - Tự động điền giá trị hiện tại của lịch trình.
   - Hỗ trợ lưu lại các thay đổi `params` phù hợp với từng loại tác vụ (`source_count`, `confluence`, `reconcile`, `bridge`...).
4. **Tái cấu trúc Triệt để Bộ chọn Cascading Selector 5 Tầng cho Source Count:**
   - **Tầng 1 (Layer Type):** `source` | `shadow` | `master`.
   - **Tầng 2 (Connection):**
     - Lọc danh sách kết nối theo đúng `role_type` của Layer đã chọn.
     - **Tự động gán `engine_type`** theo Connection (`postgresql`, `mongodb`, `mysql`, `sftp`). Khóa read-only.
     - **Tự động gán `object_type`** theo Engine (`mongodb` -> `collection`, SQL -> `table`, `sftp` -> `file_batch`).
   - **Tầng 3 (Container / Namespace):**
     - Đổi từ Input text sang `<Select showSearch>` lấy dữ liệu thực tế:
       - Source: Lấy `source_db` từ `source_object_registry`.
       - Shadow: Lấy `shadow_schema` thực tế từ `shadow_bindings`.
       - Master: Lấy `master_schema` thực tế từ `master_bindings`.
   - **Tầng 4 (Object Name):**
     - Dropdown `<Select showSearch>` lọc danh sách Table / Collection thuộc đúng Namespace ở Tầng 3.
   - **Tầng 5 (Date Field):**
     - Tự động điền `timestamp_field` từ metadata của đối tượng đã chọn.

