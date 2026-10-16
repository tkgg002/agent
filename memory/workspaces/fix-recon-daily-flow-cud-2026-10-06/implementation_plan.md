# IMPLEMENTATION PLAN — AUDIT & FIX SOURCE COUNT, MULTI-DB ROUTING & RUN NOW
> **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`  
> **Cập nhật lần cuối:** 2026-10-14T15:57:00+07:00

## 1. Mục tiêu
1. Khắc phục 3 lỗi `source_count` trên Activity Manager:
   - Lỗi 1: Chọn Layer Type `source` / `shadow` không hiện danh sách kết nối trong `connection_code`.
   - Lỗi 2: Chọn Layer Type `master` và `connection_code` (`master_2` / `default_master`) không hiện đúng PostgreSQL Schema thực tế (`container_namespace`).
   - Lỗi 3: Hiển thị chi tiết lỗi thực thi (`last_error`) trong Tooltip kèm nút sao chép.
2. Khắc phục lỗi bấm **Chạy ngay** (`run_now = true`) chỉ thấy `schedule-update` mà không chạy `source_count`, và lỗi `ERROR: relation "master_centrallized_export_service.export_jobs_2" does not exist (SQLSTATE 42P01)`.
3. Thực hiện Adversarial Audit từng dòng code (Line-by-Line), kiểm thử Runtime thật (No Mockup), thẩm định qua `/laya_router` (`http://127.0.0.1:8200/v1/predict`), và khắc phục 6 điểm mù phát sinh.

## 2. Các hạng mục triển khai
- **`cdc-cms-service`**:
  - Mở route `GET /api/v1/connections` trả về danh sách kết nối đa tầng (`source`, `shadow`, `master`).
  - Bổ sung `sanitizeConnectionHost` che mật khẩu `user:pass@` thành `***:***@` trong cột `host`.
- **`cdc-cms-web` (`ActivityManager.tsx`)**:
  - Kết nối `/api/v1/connections`, lọc `connectionOptions` theo `layer_type`.
  - Chuẩn hóa `namespaceOptions` chỉ lấy PostgreSQL Schema thực tế (`master_schema`, `shadow_schema`), chuẩn hóa `effectiveConnCode = m.master_connection_code || 'default_master'`, tách `bareTable` và kế thừa `timestamp_field` từ `sourceObjects`.
  - Bọc `last_error` trong Tooltip monospace kèm nút copy.
- **`centralized-data-service`**:
  - Tích hợp `ConnectionManager` (`GetMasterDB`, `GetShadowDB`) và `ActivityLogger` vào `SourceCountRunner` và `ConfluenceReportRunner`.
  - Đồng bộ `ticker = 2s` với TTL khóa phân tán Redis `SetNX` (`1500ms`).
  - Nhận diện chính xác lệnh Run Now thủ công bằng `sched.LastRunAt == nil && sched.NextRunAt != nil && !sched.NextRunAt.After(now)`.
