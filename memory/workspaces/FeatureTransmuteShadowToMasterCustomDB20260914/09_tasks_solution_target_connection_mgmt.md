# 09_tasks_solution_target_connection_mgmt.md - Hồ sơ giải pháp kỹ thuật Target Connection Management Suite

## 1. Thiết kế Mô hình Dữ liệu & Lưu Trữ
Bảng catalog: `cdc_system.connection_registry`.
Khi lưu Target Master Connection:
- `connection_code`: String (bắt buộc, ví dụ `pg-analytics-01`), regex `^[a-zA-Z0-9_-]+$`.
- `display_name`: String (ví dụ `Postgres Analytics Cluster`).
- `role_type`: Cố định `'master'`.
- `engine_type`: Cố định `'postgresql'`.
- `host`: IP/Domain (ví dụ `10.200.186.210`).
- `port`: Integer (mặc định 5432).
- `default_database`: Tên database đích (ví dụ `analytics_db`).
- `default_schema`: Schema đích (mặc định `public`).
- `secret_ref`: Chuỗi định danh `'vault:' + connection_code` hoặc plain pointer.
- `options_json`: Lưu DSN và thông số:
  ```json
  {
    "dsn": "postgres://user:password@host:port/database?sslmode=disable",
    "url": "postgres://user:password@host:port/database?sslmode=disable",
    "sslmode": "disable"
  }
  ```
- `status`: `'active'` (hoặc `'retired'` khi bị xoá).

## 2. Chi tiết Implementation CMS Backend (`cdc-cms-service`)

### 2.1 Test Connection API (`POST /api/v1/master-connections/test`)
- Nhận thông tin: `host`, `port`, `database`, `username`, `password`, `sslmode`.
- Build DSN tạm thời:
  ```go
  dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=5",
      url.QueryEscape(req.Username),
      url.QueryEscape(req.Password),
      req.Host, req.Port, req.Database, req.SSLMode,
  )
  ```
- Thử kết nối dùng `sql.Open("pgx", dsn)` hoặc `gorm.Open(postgres.Open(dsn))`:
  + Đo thời gian latency: `start := time.Now(); err := sqlDB.PingContext(ctx); elapsed := time.Since(start)`
  + Query kiểm tra phiên bản: `SELECT version()`
  + Đóng connection ngay sau khi test (`sqlDB.Close()`).
  + Nếu lỗi: Trả về HTTP 400 kèm chi tiết lỗi thân thiện (ví dụ `connection refused`, `password authentication failed`).
  + Nếu thành công: Trả về HTTP 200: `{ "success": true, "latency_ms": elapsed.Milliseconds(), "version": versionStr }`.

### 2.2 Create Master Connection API (`POST /api/v1/master-connections`)
- Validate: `connection_code` không được trùng với các connection active khác.
- Kiểm tra tính hợp lệ của port, host, database.
- Build DSN và chèn vào bảng `cdc_system.connection_registry`.
- Trả về đối tượng connection vừa tạo.

### 2.3 Delete / Deactivate API (`DELETE /api/v1/master-connections/:id`)
- Guard an toàn: Kiểm tra bảng `cdc_system.master_binding WHERE master_connection_id = ? AND is_active = true`.
- Nếu có bảng Master đang sử dụng connection này: Từ chối xoá và trả về lỗi: `không thể xoá connection đang được sử dụng bởi N bảng master (ví dụ: orders_master, payments_master)`.
- Nếu không có binding active: Đổi `status = 'retired'`.

## 3. Chi tiết Implementation CMS Web (`cdc-cms-web`)

### 3.1 Modal Tạo / Sửa Target Connection
- Tên component: `TargetConnectionModal` hoặc tích hợp trong `MasterRegistry.tsx`.
- Các trường:
  + Mã kết nối (`connection_code`): validate regex.
  + Tên gợi nhớ (`display_name`).
  + Host / IP và Cổng (Port).
  + Tên Database và Schema mặc định.
  + Username và Password (Input.Password).
  + SSL Mode (Select: `disable`, `require`, `prefer`).
- Nút `[Test Connection]`:
  + Khi bấm: gọi `POST /api/v1/master-connections/test`.
  + Hiển thị Spin `Đang kiểm tra kết nối...`.
  + Nếu thành công: hiển thị Alert xanh lá *"Kết nối thành công (PostgreSQL 16.2 - Latency: 8ms)"*.
  + Nếu thất bại: hiển thị Alert đỏ kèm thông báo lỗi từ server.

### 3.2 Tích hợp Drawer Quản Lý Kết Nối & Quick Add
- Trên thanh công cụ Master Registry: Thêm nút `<DatabaseOutlined /> Quản Lý Kết Nối Đích` mở Drawer hiển thị bảng danh sách các Master Connections đang có, nút Test lại, nút Thêm mới, nút Sửa/Xoá.
- Trong Modal "Tạo Master Table": Thêm nút `<PlusOutlined />` cạnh Select `Target Database Connection`. Bấm vào sẽ mở nhanh form tạo connection, sau khi submit thành công thì dropdown tự động chọn connection mới tạo.
