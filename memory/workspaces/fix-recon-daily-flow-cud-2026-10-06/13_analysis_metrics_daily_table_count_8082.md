# 13 — Phân Tích Thực Chứng: Độ Chính Xác Metrics Đếm Bảng Theo Ngày Trên `:8082/metrics` (2026-10-16)

- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Task Suffix:** `metrics_daily_table_count_8082`
- **Chế độ:** `Audit / Propose-Only` (Rule #1, Rule #13 — Không tự ý chỉnh sửa source code trước khi có lệnh `APPROVE`).
- **Nguyên tắc kiến trúc:** Tuyệt đối **KHÔNG** thực hiện các tác vụ đếm tổng toàn bảng / toàn hệ thống (`source_total`, `full_source_count`, `window_source_count`, `shadow_total`, `master_total`). Chỉ kiểm định **Tín hiệu Luồng Thời gian thực** và **Số đếm Bản ghi theo Lát cắt Ngày (`Daily Sliced Count`)**.

---

## 1. Các Phát Hiện Thực Chứng (`VERIFIED` Kèm `file:line`)

### 1.1. `[VERIFIED - Finding F1]` Bảng Source (`source_object_registry`) CHƯA HỀ CÓ chỗ cấu hình Cột Ngày Tạo (`Creation Date Field`)
- **Bằng chứng mã nguồn:**
  - Tại [`source_object_registry.go:8-24`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L8-L24) và [`table_registry.go:9-33`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/table_registry.go#L9-L33), struct `SourceObjectRegistry` chỉ khai báo `PrimaryKeyField` (`primary_key_field`) và `TimestampField` (`timestamp_field` — mặc định `lastUpdatedAt` dùng để lọc cửa sổ cập nhật cho Recon).
  - Trên UI [`TableRegistry.tsx:1647-1801`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx#L1647-L1801), Form đăng ký và chỉnh sửa Source Object hoàn toàn không có trường nhập cột ngày tạo của bảng Source.
- **Hệ quả:** Khi lập lịch đếm bản ghi theo ngày (`source_count`) tại [`ActivityManager.tsx:221-234`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L221-L234), hệ thống phải đoán mò bằng `CREATION_COL_REGEX` hoặc điền mặc định `'createdAt'` / `'created_at'`.

### 1.2. `[VERIFIED - Finding F2]` UI `ActivityManager.tsx` gán nhầm fallback `_created_at` cho Bảng Master gây lỗi `SQLSTATE 42703`
- **Bằng chứng mã nguồn & Runtime:**
  - Tại [`ActivityManager.tsx:444-446`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L444-L446): khi chọn tầng `master` mà chưa tìm thấy rule khớp regex, code thực thi `form.setFieldsValue({ date_field: '_created_at' })`.
  - Bảng vật lý Master chỉ có 4 cột hệ thống (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`), **không có cột `_created_at`**, khiến lịch `#24` (`master_payment_bill_service.payment_bills_1`) lúc `00:05:01` sáng `16/10/2026` bị lỗi `ERROR: column "_created_at" does not exist (SQLSTATE 42703)`.
  - Tương tự, lịch `#23` (`shadow_traitestpbs.payment_bills`) cấu hình `date_field = "createdAt"` cũng lỗi `SQLSTATE 42703` vì cột `createdAt` chưa được map ra khỏi `_raw_data` trên bảng Shadow.

### 1.3. `[VERIFIED - Finding F3]` Câu đếm SQL theo ngày (`countSQL`) trên Shadow và Master chưa lọc bản ghi xóa mềm (`_deleted = false`)
- **Bằng chứng mã nguồn:**
  - Tại [`source_count_runner.go:288-295`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L288-L295), hàm `countSQL` sinh câu lệnh:
    `SELECT COUNT(*) FROM "<schema>"."<table>" WHERE "<dateField>" >= ? AND "<dateField>" < ?`
  - Câu lệnh này thiếu điều kiện `AND "_deleted" = false` đối với các bảng thuộc tầng `shadow` và `master` (vốn dùng cơ chế xóa mềm `_deleted = true`), dẫn tới nguy cơ đếm thừa các dòng đã xóa mềm trong ngày so với bảng Source.

### 1.4. `[VERIFIED - Finding F4]` Kết quả đếm bản ghi theo ngày (`cdc_system.daily_source_counts`) chưa được đẩy ra `:8082/metrics`
- **Bằng chứng mã nguồn & Runtime:**
  - Tại [`source_count_runner.go:210-270`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L210-L270), sau khi `SourceCountRunner.Run` lưu `countResult` vào bảng `cdc_system.daily_source_counts`, hàm chỉ ghi `activityLogger.Quick` và log info rồi kết thúc, hoàn toàn chưa gọi `pkgs/metrics` để xuất Gauge đếm theo ngày ra `:8082/metrics`.
  - Đồng thời các Counter sự kiện theo thời gian thực (`cdc_sink_events_total`, `cdc_sync_success_total`, `cdc_transmute_ops_total`) chỉ nằm trên RAM của tiến trình `:8082` (bị reset về `0` khi restart tiến trình lúc `15:52:07` ngày `15/10/2026`).

