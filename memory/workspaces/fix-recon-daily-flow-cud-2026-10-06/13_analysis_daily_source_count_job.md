# Phân Tích & Thiết Kế Chuẩn: Tích Hợp Tác Vụ Count Source & Confluence Report Vào Hệ Thống Activity Manager (`cdc_worker_schedule`)

- **Ngày khởi tạo:** 2026-10-12 (Cập nhật 16:50)
- **Tác giả:** Brain (Architect)
- **Tiếp thu bài học:** Hệ thống đã có sẵn hạ tầng điều phối job `/activity-manager` và bảng `cdc_worker_schedule`. TUYỆT ĐỐI KHÔNG set tĩnh giờ giấc trong code. Tận dụng 100% cơ chế động này.

---

## 1. Hiện Trạng Hạ Tầng Scheduler Của Hệ Thống (`/activity-manager`)

Hệ thống Data Hub đã có sẵn một hệ thống điều phối job nền cực kỳ mạnh mẽ:
- **Giao diện quản trị:** Trang **`/activity-manager`** (`cdc-cms-web/src/pages/ActivityManager.tsx`).
- **Bảng dữ liệu điều phối:** **`cdc_system.cdc_worker_schedule`** (Migration `007_worker_schedule.sql`).
- **Bộ máy thực thi (Runner Engine):** **`server_scheduler.go`** trong `centralized-data-service`.
- **Cơ chế phân tán:** Tích hợp **Redis Distributed Lock** (`SetNX` `cdc:lock:scheduler_poll`) đảm bảo chỉ có 1 pod thực thi tại một thời điểm, tránh race condition.
- **Tính năng trên UI:**
  - Người dùng tự do bật/tắt (`is_enabled`).
  - Người dùng tự do chỉnh chu kỳ chạy **`interval_minutes`** (ví dụ: 60 phút, 720 phút, 1440 phút = 1 ngày...).
  - Chỉ định phạm vi bảng **`target_table`** hoặc chạy cho toàn bộ bảng.
  - Theo dõi `last_run_at`, `next_run_at`, `run_count`, `last_error`.

---

## 2. Giải Pháp Tích Hợp: Đăng Ký 2 Operations Mới Vào Hệ Thống Activity Manager

Thay vì viết thêm cron scheduler riêng lẻ hay set tĩnh giờ chạy trong code, chúng ta chỉ cần **đăng ký thêm 2 Operation tiêu chuẩn** vào hệ thống job runner sẵn có:

### 2.1. Operation 1: `source-count` (Đếm số bản ghi theo ngày tạo)
- **Mã Operation:** `source-count`
- **Mô tả trên UI:** "Kiểm đếm bản ghi nguồn theo ngày tạo"
- **Hành vi khi runner gọi:**
  1. Quét các bảng được cấu hình (hoặc `target_table`).
  2. Lấy tên cột `date_column` riêng của bảng đó từ cấu hình pipeline (CamelCase `"createdAt"`, snake_case `"created_at"`...).
  3. Chạy câu lệnh `SELECT COUNT(*) GROUP BY DATE(...)` cho khoảng thời gian gần nhất.
  4. Lưu kết quả vào bảng tổng hợp `cdc_system.daily_source_counts`.
- **Cấu hình trên `/activity-manager`:**
  - `interval_minutes`: Mặc định `1440` (1 ngày) hoặc `360` (6 tiếng) do Admin tự cấu hình trên web.
  - `is_enabled`: Bật/Tắt linh hoạt.

### 2.2. Operation 2: `confluence-report` (Xuất báo cáo đối soát Confluence)
- **Mã Operation:** `confluence-report`
- **Mô tả trên UI:** "Xuất báo cáo đối soát lên Confluence"
- **Hành vi khi runner gọi:**
  1. Đọc các cấu hình Confluence đang bật trong `confluence_report_configs`.
  2. Lấy dữ liệu `DailyFlow` + `daily_source_counts` + `ReconciliationReport`.
  3. Render XHTML và gọi Confluence REST API bắn vào đúng `parent_page_id` tương ứng.
- **Cấu hình trên `/activity-manager`:**
  - `interval_minutes`: Mặc định `1440` (1 ngày) hoặc theo nhu cầu của team.
  - Có thể bấm trigger chạy ngay bất cứ lúc nào.

---

## 3. Cập Nhật UI Danh Sách Operations Trong `ActivityManager.tsx`

```typescript
const ALL_OPERATIONS = [
  { value: 'transform', label: 'Chuyển đổi field (Transform)' },
  { value: 'field-scan', label: 'Quét field mới (Field Scan)' },
  { value: 'partition-check', label: 'Kiểm tra partition' },
  { value: 'drop-gin-index', label: 'Xoá GIN Index' },
  { value: 'create-default-columns', label: 'Tạo field mặc định' },
  // MỚI: Tích hợp 2 operation mới vào Activity Manager
  { value: 'source-count', label: 'Đếm bản ghi theo ngày tạo (Source Count)' },
  { value: 'confluence-report', label: 'Xuất báo cáo Confluence (Daily Report)' },
];
```

---

## 4. Ưu Điểm Tuyệt Đối Của Phương Án Này
1. **Không có bất kỳ mốc thời gian tĩnh nào:** Mọi thời gian và chu kỳ chạy do Quản trị viên tự quyết định trên web.
2. **Kế thừa 100% hạ tầng vững chắc có sẵn:** Tận dụng Redis lock, retry, audit log, logging của `server_scheduler.go`.
3. **Trải nghiệm quản trị đồng nhất:** Cả team chỉ cần vào một nơi duy nhất là `/activity-manager` để bật/tắt, theo dõi trạng thái và thời gian chạy tiếp theo (`next_run_at`) của tất cả các loại job trong hệ thống.
