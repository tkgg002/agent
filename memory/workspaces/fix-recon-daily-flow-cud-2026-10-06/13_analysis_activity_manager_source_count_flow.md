# Phân Tích & Thiết Kế Toàn Diện: Cơ Chế Cấu Hình Source-Count Trên Activity Manager & Bảng Lưu Trữ Kết Quả

- **Ngày khởi tạo:** 2026-10-12 (Cập nhật 17:00)
- **Tác giả:** Brain (Architect)
- **Giải quyết trọn vẹn 2 câu hỏi cốt lõi:**
  1. Khi tạo Operation trên UI Activity Manager: Bổ sung các trường để chọn field ngày tạo (`date_field`), giờ chạy cố định (`run_at_time`), và khoảng thời gian đếm.
  2. Count xong ghi vào đâu: Thiết kế bảng lưu trữ kết quả tập trung `cdc_system.daily_source_counts` để phục vụ CMS Web và Confluence Report.

---

## 1. THIẾT KẾ CẤU HÌNH TRÊN UI ACTIVITY MANAGER (FORM MODAL)

### Hiện trạng:
Modal "Tạo lịch trình mới" trên `ActivityManager.tsx` chỉ có:
- Tác vụ (`operation`)
- Bảng / Collection (`scope_key`)
- Chu kỳ phút (`interval_minutes`)
- Ghi chú (`notes`)
$\rightarrow$ **Hoàn toàn thiếu:** field ngày tạo của bảng, mốc giờ chạy cố định `00:05`, và khoảng thời gian đếm.

### Giải pháp thiết kế:
Khi người dùng chọn `operation = "source-count"`, Form Modal trên giao diện sẽ **động hiển thị thêm 3 trường cấu hình đặc thù**:

1. **Trường ngày tạo (`date_field`):**
   - Ô nhập text hoặc Select gợi ý (ví dụ: `createdAt`, `created_at`, `create_time`, `order_date`...).
   - Giải thích: *"Tên cột ngày tạo riêng biệt của bảng/collection này (không cố định)"*.
2. **Giờ chạy cố định trong ngày (`run_at_time`):**
   - Ô nhập format `HH:mm` (mặc định: `00:05`).
   - Nếu có giá trị này: Job sẽ tự động căn chuẩn xác vào lúc `00:05:00` sáng mỗi ngày để chốt sổ ngày hôm qua.
3. **Khoảng thời gian đếm (`count_window`):**
   - Lựa chọn:
     - `yesterday` (Mặc định - Chốt sổ 1 ngày hôm qua $D-1$).
     - `last_7_days` (Quét lại 7 ngày gần nhất).
     - `last_14_days` (Quét lại 14 ngày gần nhất).
4. **Múi giờ (`timezone`):**
   - Mặc định: `Asia/Ho_Chi_Minh` (GMT+7).

### Cấu trúc lưu trữ linh hoạt trong Database (`cdc_system.cdc_worker_schedule`):
Thay vì tạo thêm nhiều cột lẻ tẻ, áp dụng pattern chuẩn của các hệ thống Scheduler hiện đại: Thêm cột **`params JSONB NOT NULL DEFAULT '{}'::jsonb`** vào bảng `cdc_worker_schedule`:
```sql
ALTER TABLE cdc_system.cdc_worker_schedule
  ADD COLUMN IF NOT EXISTS run_at_time VARCHAR(8) NULL,
  ADD COLUMN IF NOT EXISTS params JSONB NOT NULL DEFAULT '{}'::jsonb;
```
Dữ liệu lưu trong `params` khi tạo operation `source-count`:
```json
{
  "date_field": "createdAt",
  "timezone": "Asia/Ho_Chi_Minh",
  "window_mode": "yesterday"
}
```

---

## 2. COUNT XONG NÓ GHI VÀO ĐÂU? (KẾT QUẢ ĐƯỢC LƯU TRỮ RA SAO)

Khi Job `source-count` chạy lúc `00:05`, kết quả đếm được sẽ được ghi vào bảng:
👉 **`cdc_system.daily_source_counts`** trong database `cdc_dw`.

### Schema DDL bảng lưu trữ:
```sql
CREATE TABLE IF NOT EXISTS cdc_system.daily_source_counts (
    id             BIGSERIAL PRIMARY KEY,
    schedule_id    INT REFERENCES cdc_system.cdc_worker_schedule(id) ON DELETE SET NULL,
    target_table   VARCHAR(128) NOT NULL,          -- Tên bảng (ví dụ: 'trans_his')
    source_db      VARCHAR(64)  NULL,              -- Database nguồn (ví dụ: 'goopay_prod')
    report_date    DATE         NOT NULL,          -- Ngày được đếm (ví dụ: '2026-10-11')
    creation_count BIGINT       NOT NULL DEFAULT 0,-- Số bản ghi sinh ra trong ngày
    date_field     VARCHAR(64)  NOT NULL,          -- Tên cột đã dùng để đếm (createdAt)
    query_duration_ms INT       NOT NULL DEFAULT 0,-- Thời gian thực thi query đếm
    calculated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    calculated_by  VARCHAR(64)  NOT NULL DEFAULT 'worker_schedule',
    UNIQUE (target_table, report_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_lookup
    ON cdc_system.daily_source_counts(target_table, report_date DESC);
```

---

## 3. Dữ Liệu Sau Khi Ghi Vào Bảng Này Được Dùng Ở Đâu?

Sau khi bảng `cdc_system.daily_source_counts` có số liệu, dữ liệu này sẽ phục vụ ngay cho 2 nơi:

1. **Hiển thị trên CMS Web (`ReconPipelineGrid.tsx`):**
   - API `/api/reconciliation/daily-flow` chỉ cần `LEFT JOIN cdc_system.daily_source_counts` theo `target_table` và `date`.
   - Cột **"Ngày Tạo (Creation Count)"** trên bảng CMS hiển thị ngay lập tức (tốc độ 0.001 giây, không quét DB nguồn).
2. **Xuất báo cáo lên Confluence (`Core Trans Proxy History Service`):**
   - Khi bot Confluence chạy, nó đọc từ bảng `daily_source_counts` để điền số lượng bản ghi nghiệp vụ sinh ra trong ngày vào cột của trang Confluence.
3. **Tra cứu lịch sử:**
   - Trên trang `/activity-manager`, khi click vào dòng schedule `source-count`, có thể mở modal xem lịch sử số lượng bản ghi của 30 ngày gần nhất.
