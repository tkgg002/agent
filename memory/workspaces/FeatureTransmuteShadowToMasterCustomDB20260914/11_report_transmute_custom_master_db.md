# 11_report_transmute_custom_master_db.md - Báo Cáo Triển Khai Kỹ Thuật

## 1. Tổng quan công việc
Role **Muscle (Chief Engineer)** đã hoàn thành toàn diện (full-loop) triển khai mã nguồn, kiểm thử và verify cho tính năng:
> **"Transmute chỉ từ Shadow sang Master, cho phép Master chọn Database Connection đích độc lập (ví dụ một PostgreSQL khác) kèm DDL Safety Gate"**.

---

## 2. Danh sách các file đã thay đổi & Chi tiết

### A. Centralized Data Service (CDS Engine)
1. **`pkgs/database/multi.go`**
   - **Số dòng thay đổi:** +5 dòng
   - **Chi tiết thay đổi:**
     + Bổ sung alias hằng số `RoleSystem = RoleControlPlane`.
     + Export hàm `OpenGorm(dsn, role string) (*gorm.DB, error)` trên struct `Registry` để cho phép mở kết nối GORM động với pool config chuẩn (MaxOpenConns, MaxIdleConns, ConnMaxLifetime, Opentelemetry, Metrics callback).

2. **`internal/service/source/connection_manager.go`**
   - **Số dòng thay đổi:** +84 dòng
   - **Chi tiết thay đổi:**
     + Bổ sung `masterPoolMu sync.RWMutex` và `masterDBPool map[string]*gorm.DB` vào struct `ConnectionManager`.
     + Khởi tạo cache `masterDBPool` trong constructor `NewConnectionManagerWithRegistry`.
     + Cài đặt cơ chế dynamic resolve trong `GetMasterDB(ctx, key)`:
       * Fallback nhanh về destination pool nếu `key == "" || key == "default"`.
       * Thread-safe cache lookup với `RLock` và `Lock` (double-check).
       * Tra cứu override URI từ `cfg.ConnectionOverrides[key]`.
       * Truy vấn DB hệ thống (`cdc_system.connection_registry WHERE (connection_code = ? OR connection_code = ?) AND status = 'active'`).
       * Trích xuất DSN từ `options_json`, `secret_ref` hoặc tổng hợp qua `buildDSNFromFieldsPatched`.
       * Fallback an toàn về destination pool nếu không tìm thấy cấu hình riêng.
       * Mở kết nối GORM với pool chuẩn và lưu vào cache `masterDBPool`.
     + Cập nhật `MasterKeys()` trả về danh sách các connection keys đang được cache.

3. **`internal/service/master/transmuter.go`**
   - **Số dòng thay đổi:** +37 dòng
   - **Chi tiết thay đổi:**
     + Tại method `Run()`: Triển khai **DDL Safety Gate** ngay trước vòng lặp xử lý batch.
     + Lấy `masterDB` thông qua `connMgr.GetMasterDB(ctx, masterRow.MasterConnectionKey)`.
     + Kiểm tra sự tồn tại của bảng master trên target database qua câu lệnh:
       `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?)` (hỗ trợ cả `sqlite_master` cho SQLite unit test).
     + Trả về error rõ ràng: `master table %s.%s does not exist on target database (DDL not created or pending approval)` nếu bảng chưa có DDL vật lý trên DB đích.
     + Đánh dấu runtime failure và cập nhật trạng thái transmute job là `FAILED`.

4. **`test/internal/service/connection_manager_test.go`**
   - **Số dòng thay đổi:** +41 dòng
   - **Chi tiết thay đổi:**
     + Bổ sung unit test `TestConnectionManager_GetMasterDB_OverrideAndCache` xác nhận:
       * Khởi tạo kết nối custom master DB thành công.
       * Lần gọi thứ 2 hit đúng cache instance trong `masterDBPool`.
       * `MasterKeys()` hiển thị đúng key custom master đã đăng ký.

---

### B. CDC CMS Service (Backend)
1. **`internal/domain/master/binding.go`**
   - **Số dòng thay đổi:** +14 dòng
   - **Chi tiết thay đổi:**
     + Bổ sung struct `MasterConnectionItem` làm DTO biểu diễn thông tin connection registry cho tầng Master.

2. **`internal/app/ports/repository.go`**
   - **Số dòng thay đổi:** +1 dòng
   - **Chi tiết thay đổi:**
     + Khai báo phương thức `ListMasterConnections(ctx context.Context) ([]master.MasterConnectionItem, error)` trong interface `MasterRepo`.

3. **`internal/infra/persistence/master/master_repo_gorm.go`**
   - **Số dòng thay đổi:** +16 dòng
   - **Chi tiết thay đổi:**
     + Cài đặt `ListMasterConnections` truy vấn từ `cdc_system.connection_registry` các connection đang `status = 'active'` và là postgres/master connection.

4. **`internal/api/master/master_registry_handler_connections.go`** (NEW FILE)
   - **Số dòng thay đổi:** +24 dòng
   - **Chi tiết thay đổi:**
     + Cài đặt handler `ListConnections` cho endpoint `GET /api/v1/master-connections`, trả về danh sách active master connections kèm count.

5. **`internal/router/router.go`**
   - **Số dòng thay đổi:** +2 dòng
   - **Chi tiết thay đổi:**
     + Đăng ký route `shared.Get("/v1/master-connections", h.Master.Registry.ListConnections)` và `dual("GET", shared, "/master-connections", h.Master.Registry.ListConnections)`.

6. **`internal/app/commands/master/create_master.go`**
   - **Số dòng thay đổi:** +5 dòng
   - **Chi tiết thay đổi:**
     + Ưu tiên sử dụng `cmd.MasterConnectionCode` truyền từ request frontend.
     + Nếu rỗng, fallback về `h.defaultMasterConnectionCode`.
     + Resolve `connID` từ `cdc_system.connection_registry` và liên kết `master_connection_id` vào `master_binding`.

---

### C. CDC CMS Web (Frontend)
1. **`src/pages/MasterRegistry.tsx`**
   - **Số dòng thay đổi:** +56 dòng
   - **Chi tiết thay đổi:**
     + Bổ sung `useQuery` load danh sách `masterConnections` từ `/api/v1/master-connections` khi mở Create Modal.
     + Cập nhật state `form` với `master_connection_code: 'default'`.
     + Trong Create Master Modal: Bổ sung dropdown "Target Database Connection" cho phép operator chỉ định instance PostgreSQL đích độc lập.
     + Truyền `master_connection_code` vào payload API `POST /api/v1/masters`.
     + Hiển thị tag Target Connection `<DatabaseOutlined /> {r.master_connection_code}` tại cột `DB Master` trên bảng danh sách.
     + Kiểm soát nút `Sync` và nút `Chạy ngay`: bắt buộc `r.schema_status === 'approved'` mới cho phép kích hoạt. Cập nhật tooltip hướng dẫn rõ ràng về DDL readiness.

2. **`src/pages/TableRegistry.tsx`**
   - **Kiểm tra audit:**
     + Xác nhận 100% không có bất kỳ nút Transmute nào được đặt sai chỗ trên hàng Shadow table chính.
     + Giữ nguyên luồng chuẩn: Shadow table chỉ có nút "Create" để điều hướng sang trang Master Registry với metadata prefill, không vi phạm DDL lifecycle.

---

## 3. Đối soát Lesson Learnt (Tripwires Check)
1. **#master-ddl-prerequisite-fallacy:**
   - Transmute chỉ được kích hoạt khi Master Table đã hoàn tất vòng đời DDL (`schema_status = 'approved'`).
   - Đã cài đặt DDL Safety Gate trực tiếp trên CDS Engine: `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?)`.
   - UI disable triệt để các action Transmute/Sync khi Master chưa Approved.
   - Không đặt nút Transmute trên hàng Shadow table ở `TableRegistry.tsx`.
2. **#accidental-variable-obliteration:**
   - Áp dụng phạm vi diff tối thiểu (minimal target chunks) trên `MasterRegistry.tsx`, không làm mất các biến cục bộ lân cận.
3. **#simplicity-first:**
   - Giữ trọn vẹn kiến trúc và pattern hiện có của từng repository, tái sử dụng connection pool settings và GORM registry chuẩn.
