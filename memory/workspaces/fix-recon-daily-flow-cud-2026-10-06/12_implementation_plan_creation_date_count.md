# Kế Hoạch Triển Khai Toàn Diện: Cascading Data Selector & Daily Count Đa Hình (SFTP, MongoDB, PostgreSQL, MySQL, ClickHouse)

- **Ngày cập nhật:** 2026-10-12
- **Tác giả:** Brain (Chairman & Architect)
- **Tình trạng:** Đã hoàn thiện thiết kế kỹ thuật đa hình, chờ lệnh `APPROVE` từ User để triển khai.

---

## I. MỤC TIÊU & PHẠM VI (SCOPE & OBJECTIVES)

1. **Chuẩn hóa Bộ chọn Đối tượng Dữ liệu Đa Hình (Polymorphic Cascading Selector):**
   - Không hạn chế ở 2 loại table/collection. Hỗ trợ toàn diện các engine trong hệ sinh thái Data Hub: **PostgreSQL**, **MongoDB**, **MySQL**, **MariaDB**, **SFTP**, **ClickHouse**, **Kafka**, **File**.
   - Phân cấp 5 tầng linh hoạt:
     `Layer Type (source/shadow/master)` $\rightarrow$ `Connection (chứa engine_type)` $\rightarrow$ `Container / Namespace / Path` $\rightarrow$ `Object Name / Pattern` $\rightarrow$ `Date Strategy & Field`.

2. **Cơ chế Job Đếm Số Bản Ghi Theo Khoảng Thời Gian / Ngày Tạo (`source-count`):**
   - Tích hợp vào `/activity-manager` (`cdc_worker_schedule` và `server_scheduler.go`).
   - Hỗ trợ mốc giờ chạy cố định `00:05` sáng hàng ngày (`run_at_time = '00:05'`) để chốt sổ ngày hôm qua ($D-1$).
   - Nhận diện đúng chiến lược ngày theo từng engine:
     - PostgreSQL / MySQL / MongoDB: Cột ngày tạo (`createdAt`, `created_at`, `create_time`...).
     - SFTP / File: Trích xuất ngày từ tên file (`filename_regex`, vd `YYYYMMDD`) hoặc `file_mtime`.

3. **Lưu trữ Kết Quả Đếm Tập Trung:**
   - Kết quả được ghi vào bảng đa hình **`cdc_system.daily_source_counts`** trong database `cdc_dw`.
   - Phục vụ đồng thời 2 đích đến:
     - Lưới CMS Web Reconciliation Daily Flow (trả kết quả siêu tốc 0.001s).
     - Báo cáo kiểm tra đối soát dữ liệu trên Atlassian Confluence (child page của `Core Trans Proxy History Service`, ID `463833042`).

---

## II. LỘ TRÌNH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PHASES)

### Phase 1: DDL Database & Migration Scripts (`cdc-cms-service`)
- [ ] **Task 1.1:** Migration `081_enhance_worker_schedule_for_dynamic_jobs.sql`:
  - Thêm `run_at_time VARCHAR(8) NULL`, `params JSONB NOT NULL DEFAULT '{}'::jsonb`.
- [ ] **Task 1.2:** Migration `082_create_polymorphic_daily_source_counts.sql`:
  - Tạo bảng `cdc_system.daily_source_counts` hỗ trợ `container_namespace`, `object_name`, `object_type`, `date_strategy`, `engine_type`.
- [ ] **Task 1.3:** Migration `083_create_confluence_report_configs.sql`:
  - Tạo bảng `cdc_system.confluence_report_configs` hỗ trợ Multi-Target Confluence.

### Phase 2: Nâng Cấp Backend Control Plane (`cdc-cms-service`)
- [ ] **Task 2.1:** Cập nhật DTO `WorkerScheduleCreateRequest` & Model hỗ trợ params đa hình.
- [ ] **Task 2.2:** Metadata Cascading APIs theo `engine_type`:
  - Trả danh sách connection theo role type (`source`, `shadow`, `master`).
  - Trả danh sách namespace/database/schema/directory theo engine type.
  - Trả danh sách table/collection/file patterns.
- [ ] **Task 2.3:** Cập nhật API `/api/reconciliation/daily-flow`:
  - Left join `cdc_system.daily_source_counts` để lấy số đếm nghiệp vụ sinh ra trong ngày.

### Phase 3: Nâng Cấp Worker Runner & Scheduler (`centralized-data-service`)
- [ ] **Task 3.1:** Cập nhật model `WorkerSchedule` (`RunAtTime`, `Params`).
- [ ] **Task 3.2:** Thuật toán căn chuẩn `00:05:00` sáng giờ VN (`Asia/Ho_Chi_Minh`) trong `server_scheduler.go`.
- [ ] **Task 3.3:** Viết Handler cho Job `source-count` đa hình:
  - Driver SFTP: Quét remote directory, lọc file theo ngày $D-1$, đếm line count.
  - Driver PostgreSQL / MySQL / ClickHouse: Query SQL đếm có khoảng thời gian $D-1$.
  - Driver MongoDB: Query `countDocuments` có filter ngày $D-1$.
  - Driver Kafka: Đếm message count theo timestamp offsets.
  - Upsert kết quả vào `cdc_system.daily_source_counts`.
- [ ] **Task 3.4:** Viết Handler cho Job `confluence-report`:
  - Đọc `confluence_report_configs`.
  - Tổng hợp số liệu từ `daily_source_counts` và Kafka metrics.
  - Gọi Confluence REST API để tạo/cập nhật trang con của `Core Trans Proxy History Service` (ID: `463833042`).

### Phase 4: Giao Diện CMS Web (`cdc-cms-web`)
- [ ] **Task 4.1:** Cập nhật `ActivityManager.tsx`:
  - Thêm `source-count` và `confluence-report` vào `ALL_OPERATIONS`.
  - Cascading Selector đa hình tự động chuyển đổi giao diện theo `engine_type` (Postgres, Mongo, MySQL, SFTP, ClickHouse).
- [ ] **Task 4.2:** Cập nhật `ReconPipelineGrid.tsx` và `DailyFlowView.tsx` hiển thị cột "Bản Ghi Ngày Tạo".
