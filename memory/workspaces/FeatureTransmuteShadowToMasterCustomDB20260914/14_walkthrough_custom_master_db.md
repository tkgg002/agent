# 14_walkthrough_custom_master_db.md - Hướng Dẫn Walkthrough Tính Năng

## 1. Mục tiêu tính năng
- Cho phép người vận hành chuyển dữ liệu (Transmute) trực tiếp từ **Shadow Table sang Master Table** với khả năng chọn **Target PostgreSQL Database Connection độc lập** (ví dụ DB Analytics, DB Reporting, hoặc DB Master phụ).
- Đảm bảo **DDL Safety Gate**: Ngăn chặn tuyệt đối việc chạy Transmute khi Master Table chưa hoàn tất vòng đời DDL (`schema_status != 'approved'`, hoặc chưa có bảng vật lý và index trên database đích).

---

## 2. Các điểm chạm người dùng (User Touchpoints & Walkthrough)

### 2.1 Tạo Master Table với Target Database Connection tùy chọn
1. Truy cập trang **Master Registry** (`/masters`).
2. Nhấn nút **Tạo Master Table** (Create Master).
3. Trong Modal:
   - Nhập thông tin bảng: `Table Name`, `Shadow Table`, `Description`.
   - Chọn mục **Target Database Connection** (dropdown tự động tải từ danh sách các connection active loại PostgreSQL). Người dùng có thể giữ `Mặc định (default)` hoặc chọn instance mong muốn (ví dụ: `pg-analytics-01`, `pg-reporting-prod`).
4. Nhấn **Xác nhận**: Hệ thống lưu `master_connection_code` và liên kết `master_connection_id` tương ứng vào `cdc_system.master_binding`.

### 2.2 Quản trị và Quan sát Target Connection
- Trên bảng danh sách **Master Registry**, cột **DB Master** hiển thị thông tin rõ ràng:
  - Tên schema và bảng: `public.orders_master`
  - Tag connection instance: `<DatabaseOutlined /> default` hoặc `<DatabaseOutlined /> pg-analytics-01`.

### 2.3 Cơ chế DDL Safety Gate trên Giao diện
- Nút **Sync** (Transmute) tại mỗi hàng:
  - Nếu Master Table đang ở trạng thái `pending_review` hoặc `rejected`: Nút **Sync** sẽ bị **Disable** (màu xám), kèm tooltip giải thích: *"Cần approve master và tạo DDL trước khi sync"*.
  - Nếu Master Table đã ở trạng thái `approved`: Nút **Sync** sáng và cho phép người vận hành mở modal đồng bộ.
  - Trong Modal Sync: Nút **Chạy ngay** cũng được gắn cờ bảo vệ `disabled={syncRow?.schema_status !== 'approved'}`.

### 2.4 Cơ chế DDL Safety Gate tại Backend Engine (`centralized-data-service`)
- Khi Worker Engine nhận job Transmute:
  - Trước khi batch fetch dữ liệu từ Shadow table và thực hiện upsert, Engine kết nối đến `masterDB` (đã được resolve động và cache theo connection key).
  - Thực hiện pre-flight query:
    ```sql
    SELECT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = ? AND table_name = ?
    )
    ```
  - Nếu bảng chưa tồn tại trên database đích: Job ngay lập tức bị từ chối với lỗi rõ ràng:
    `master table <schema>.<table> does not exist on target database (DDL not created or pending approval)`.
  - Giúp triệt tiêu hoàn toàn lỗi crash runtime `relation does not exist` và không làm gián đoạn pipeline CDC.

---

## 3. Kết quả Kiểm thử & Nghiệm thu
- **Centralized Data Service (Go Engine)**:
  - `TestConnectionManager_GetMasterDB_OverrideAndCache`: **PASS** (xác nhận dynamic connection resolution, GORM pool caching thread-safe, không leak goroutine).
  - Toàn bộ suite `TestConnectionManager_*`: **PASS (3/3 tests)**.
- **CDC CMS Service (API Backend)**:
  - `go build ./cmd/server`: **PASS (compile sạch 100%)**.
- **CDC CMS Web (Frontend)**:
  - `npm run build` (`tsc -b && vite build`): **PASS (build sạch 100%)**.
