# Kế Hoạch Triển Khai Tinh Gọn 2 Giai Đoạn: Daily Source Count (Trước) & Confluence Daily (Sau)

- **Ngày cập nhật:** 2026-10-12
- **Tác giả:** Brain (Chairman & Architect)
- **Tối ưu hóa:** BÃI BỎ bảng `confluence_report_configs`, đưa 100% cấu hình Confluence vào `params JSONB` của `cdc_worker_schedule`. Toàn bộ thao tác chỉ diễn ra trong 1 Modal duy nhất trên `/activity-manager`.

---

## TỔNG QUAN HAI GIAI ĐOẠN

```
┌────────────────────────────────────────────────────────────────────────┐
│ GIAI ĐOẠN 1: DAILY SOURCE COUNT (TRIỂN KHAI TRƯỚC)                    │
│ 1. DDL: Migration 081 (thêm run_at_time, params vào worker schedule).  │
│ 2. DDL: Migration 082 (bảng đa hình cdc_system.daily_source_counts).   │
│ 3. FE: Modal Cascading 5 tầng trên /activity-manager theo engine.      │
│ 4. BE: Handler /api/worker-schedule nhận params JSONB.                 │
│ 5. Worker: Scheduler căn 00:05:00 AM & Runner đếm đa driver.           │
│ 6. Verify: Chạy thử đếm PostgreSQL, MongoDB, SFTP file.                │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ (Kế thừa dữ liệu daily_source_counts)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ GIAI ĐOẠN 2: CONFLUENCE DAILY REPORT (TRIỂN KHAI SAU)                  │
│ 1. DDL: KHÔNG CẦN BẢNG MỚI (Lưu 100% vào params JSONB có sẵn).        │
│ 2. FE: Modal Confluence All-in-One trên /activity-manager:             │
│    - Điền Confluence Base URL, Space, Parent Page ID, Auth Token.      │
│    - Multi-Select chọn NHIỀU TABLE cùng một lúc.                      │
│    - Giờ chạy cố định 06:00 AM.                                        │
│ 3. Worker: Scheduler căn 06:00:00 AM.                                  │
│ 4. Worker: Đọc daily_source_counts + CDC flow, render bảng XHTML.      │
│ 5. Confluence Client: Gọi API bắn Child Page dưới Parent 463833042.   │
│ 6. Verify: Kiểm tra trang sinh ra trên Atlassian Confluence thật.      │
└────────────────────────────────────────────────────────────────────────┘
```

---

## GIAI ĐOẠN 1: DAILY SOURCE COUNT (THỰC HIỆN TRƯỚC)

### Mục tiêu:
Xây dựng cơ chế đếm số lượng bản ghi / số dòng dữ liệu nghiệp vụ sinh ra trong ngày ($D-1$) từ mọi nguồn (SFTP, MongoDB, PostgreSQL, MySQL, ClickHouse...), chạy tự động vào lúc `00:05` sáng hàng ngày và ghi kết quả vào bảng `cdc_system.daily_source_counts`.

### Danh sách Tasks:
- [ ] **Task 1.1 (DDL):** Migration `081_enhance_worker_schedule.sql` (`run_at_time VARCHAR(8)`, `params JSONB`).
- [ ] **Task 1.2 (DDL):** Migration `082_create_polymorphic_daily_source_counts.sql`.
- [ ] **Task 1.3 (BE Control Plane):** Cập nhật DTO và Handler `/api/worker-schedule` trong `cdc-cms-service`.
- [ ] **Task 1.4 (Worker Core):** Nâng cấp `centralized-data-service`:
  - `mastermodel.WorkerSchedule`: Bổ sung `RunAtTime *string`, `Params datatypes.JSON`.
  - `server_scheduler.go`: Hàm `calculateNextRun` căn chuẩn xác `00:05:00` sáng giờ VN (`Asia/Ho_Chi_Minh`).
  - Viết `SourceCountRunner` đa driver: SQL Driver, MongoDB Driver, SFTP Driver.
  - Đăng ký `RegisterScheduleJob("source-count", ...)`.
- [ ] **Task 1.5 (FE UI):** Nâng cấp `ActivityManager.tsx` trong `cdc-cms-web`:
  - Thêm `source-count` vào `ALL_OPERATIONS`.
  - Cascading Selector 5 tầng tự động biến đổi theo `engine_type` (SFTP, Mongo, Postgres/MySQL).
- [ ] **Task 1.6 (Integration & Verification):** Chạy thử nghiệm lệnh đếm trên DB và file thật, kiểm tra bản ghi ghi vào `cdc_system.daily_source_counts`.

---

## GIAI ĐOẠN 2: CONFLUENCE DAILY REPORT (THỰC HIỆN SAU)

### Mục tiêu:
Tự động tổng hợp dữ liệu từ `cdc_system.daily_source_counts` (kết quả của Giai đoạn 1) kết hợp với lưu lượng CDC trong ngày, format thành bảng báo cáo kiểm tra dữ liệu chuyên nghiệp và đẩy lên Atlassian Confluence làm **trang con (child page)** của `Core Trans Proxy History Service` (Page ID: `463833042`).

### Danh sách Tasks:
- [ ] **Task 2.1 (FE UI All-in-One):** Cập nhật `ActivityManager.tsx`:
  - Thêm `confluence` vào `ALL_OPERATIONS`.
  - Modal Form Confluence All-in-One: Nhập thông tin Confluence (Base URL, Space, Parent Page ID `463833042`, Auth Token) + Multi-Select chọn nhiều table cùng một lúc + Mốc giờ `06:00` AM.
  - Toàn bộ lưu vào `params JSONB` của `cdc_worker_schedule`.
- [ ] **Task 2.2 (Worker Core):** Nâng cấp `centralized-data-service`:
  - Viết `ConfluenceReportRunner`:
    - Đọc thông tin Confluence và `selected_tables` từ `params JSONB`.
    - Lặp qua danh sách `selected_tables`: lấy `creation_count` từ `daily_source_counts` (nếu có) + lưu lượng CDC từ `recon_hourly_pipeline_stats`.
    - Render bảng XHTML Confluence Storage Format.
    - Gọi Confluence REST API (`POST /wiki/rest/api/content`) với `ancestors: [{"id": "463833042"}]`.
  - Đăng ký `RegisterScheduleJob("confluence", ...)`.
- [ ] **Task 2.3 (Verification):** Kích hoạt chạy thử báo cáo, kiểm tra trang con hiển thị trực tiếp trên URL: `https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea463f119dd38b08c/pages/463833042/Core+Trans+Proxy+History+Service`.
