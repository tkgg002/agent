# 13_analysis_custom_master_db.md

## Phân tích kỹ thuật & Đánh giá rủi ro (Muscle: Chief Engineer)

### 1. Phân tích DDL Lifecycle & Transmute Safety Gate
- **Vấn đề cốt lõi:**
  Trước đây, Transmuter giả định Master Table luôn nằm trên destination database mặc định (`m.reg.GetDB(database.RoleDestination)`). Khi mở rộng cho phép Master trỏ sang instance database khác (qua `MasterConnectionKey`), nếu Master Table chưa chạy DDL tạo bảng (`cdc.cmd.master-create` chưa hoàn tất hoặc bảng đang ở `pending_review`), câu lệnh bulk upsert sẽ lập tức crash runtime với lỗi `relation does not exist`.
- **Giải pháp:**
  - Bắt buộc kiểm tra `schema_status === 'approved'` ở tầng API/CMS Web.
  - Cài đặt chốt chặn an toàn vật lý (Physical DDL Safety Gate) ngay tại CDS Engine:
    `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?)`
    Nếu bảng chưa tồn tại vật lý trên instance DB đích, tiến trình Transmute dừng ngay lập tức, trả về lỗi chi tiết `master table %s.%s does not exist on target database (DDL not created or pending approval)` và đánh dấu trạng thái transmute job là `FAILED`.

### 2. Phân tích Connection Pooling & Dynamic Routing
- **Vấn đề cốt lõi:**
  Mỗi instance PostgreSQL đích có thể có DSN, host, credentials và thông số kết nối riêng. Cần đảm bảo:
  1. Không tạo kết nối mới liên tục gây cạn kiệt connection (exhaustion).
  2. Áp dụng chuẩn cấu hình connection pool (MaxOpenConns, MaxIdleConns, ConnMaxLifetime).
  3. Hỗ trợ cả `connectionOverrides` (dev/hotfix) và dynamic database registry (`cdc_system.connection_registry`).
- **Giải pháp:**
  - Sử dụng `masterDBPool map[string]*gorm.DB` với `sync.RWMutex` (double-check lock pattern).
  - Tận dụng `Registry.OpenGorm` để đồng nhất 100% telemetry, tracing plugin và thông số connection pool.
  - Fallback an toàn về destination pool mặc định nếu key là `""`, `"default"` hoặc không tìm thấy cấu hình riêng.

### 3. Phân tích UI & UX Alignment
- **Vấn đề cốt lõi:**
  Người vận hành cần biết bảng Master đang ghi vào database connection nào và có thể tùy chọn database đích khi khởi tạo Master.
- **Giải pháp:**
  - Cung cấp API `GET /api/v1/master-connections` trả về các active connections.
  - Hiển thị dropdown Target Database Connection trong Modal tạo Master.
  - Thêm Tag hiển thị Target Connection trên bảng danh sách Master.
  - Đảm bảo các nút Sync/Transmute bị disable khi Master chưa hoàn tất vòng đời DDL.
