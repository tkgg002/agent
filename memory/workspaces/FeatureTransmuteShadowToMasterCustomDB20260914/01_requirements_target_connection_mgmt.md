# 01_requirements_target_connection_mgmt.md - Target Database Connection Management Suite

## 1. Bối cảnh & Động lực (Context & Motivation)
- **Vấn đề thực tế**:
  + Người vận hành không thể SSH vào database production để gõ lệnh `INSERT INTO cdc_system.connection_registry ...` bằng tay (vi phạm quy trình an ninh, rủi ro sai sót cú pháp, thiếu audit log).
  + Việc cấu hình tĩnh `connectionOverrides` qua file `config.yaml` / biến môi trường đòi hỏi phải sửa config và restart cụm Worker Engine mỗi khi có thêm/sửa database đích, không thể scale động khi có nhiều DB (Analytics, Reporting, Tenant DB).
- **Mục tiêu**:
  Xây dựng tính năng **Quản trị Kết Nối Đích Toàn Trình (Target Database Connection Management Suite)** trực tiếp trên giao diện CMS Web và Backend API, cho phép:
  1. Thêm mới, chỉnh sửa, vô hiệu hoá các Target Database Connection (PostgreSQL).
  2. Nút **Test Connection** (Pre-flight Ping Check) thử nghiệm kết nối mạng và kiểm tra user/pass trước khi lưu.
  3. Quản lý danh sách kết nối đích tập trung, hot-reload ngay lập tức vào dropdown của Modal Tạo Master Table mà không cần restart server hay sửa config.

## 2. Phạm vi Triển khai (Scope & Boundaries)
- **In-Scope**:
  - **CMS Backend (`cdc-cms-service`)**:
    + `POST /api/v1/master-connections/test`: Nhận thông số kết nối (`host`, `port`, `database`, `username`, `password`, `sslmode`), thử kết nối PostgreSQL bằng pgx/gorm với timeout 5s, ping `SELECT 1` hoặc `SELECT version()`, trả về kết quả latency hoặc chi tiết lỗi.
    + `POST /api/v1/master-connections`: Tạo mới target connection, validate `connection_code` unique, build DSN/options_json an toàn, lưu vào `cdc_system.connection_registry` (`role_type = 'master'`, `engine_type = 'postgresql'`, `status = 'active'`).
    + `PUT /api/v1/master-connections/:id`: Cập nhật thông số kết nối.
    + `DELETE /api/v1/master-connections/:id`: Soft-delete/deactivate connection (chuyển `status = 'retired'`). Guard an toàn: Kiểm tra nếu connection đang được liên kết bởi bảng `master_binding` active thì chặn xoá kèm thông báo chi tiết.
  - **CMS Web Frontend (`cdc-cms-web`)**:
    + Tại trang **Master Registry** (`/masters` - `MasterRegistry.tsx`):
      * Thêm nút `[Quản Lý Kết Nối Đích]` ở header mở Drawer/Modal danh sách toàn bộ các Target Database Connection.
      * Trong Drawer có nút `[+ Thêm Kết Nối Đích]` mở Modal nhập form: Mã kết nối, Tên hiển thị, Host, Port, Database, Schema, Username, Password, SSL Mode.
      * Có nút `[Test Connection]` kiểm tra kết nối realtime kèm icon trạng thái (Success / Error).
      * Ngay tại Modal "Tạo Master Table", cạnh dropdown `Target Database Connection`, có nút quick-action `[+ Thêm mới]` mở nhanh modal tạo connection và tự động chọn ngay sau khi tạo xong.

- **Out-of-Scope**:
  - Không thay đổi Worker Engine (`centralized-data-service`): Do Worker Engine đã có sẵn cơ chế dynamic connection pool qua `ConnectionManager.GetMasterDB(ctx, key)` tự động query từ `cdc_system.connection_registry`, nên khi CMS lưu connection mới, Worker tự động kết nối được ngay lập tức.
