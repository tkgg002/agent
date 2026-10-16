# Phân Tích Kỹ Thuật & Kế Hoạch: Chạy Cố Định 00:05 & Chuẩn Hóa Cột CreatedAt Cho Source Count

- **Ngày khởi tạo:** 2026-10-12 (Cập nhật 16:55)
- **Tác giả:** Brain (Architect)
- **Hai vấn đề cốt lõi do User chỉ rõ:**
  1. `interval_minutes` chỉ tính khoảng cách lặp lại từ lúc khởi động server, CHƯA có cơ chế chạy vào mốc giờ cố định trong ngày (`00:05`).
  2. `source-count` hiện tại đang bị lấy sai cột `created_at` (do hệ thống đang nhầm lẫn hoặc fallback sang `updated_at`/`lastUpdatedAt` của luồng watermark).

---

## 1. VẤN ĐỀ 1: NÂNG CẤP HẠ TẦNG SCHEDULER HỖ TRỢ GIỜ CHẠY CỐ ĐỊNH (`00:05`)

### Hiện trạng hạn chế trong `server_scheduler.go`:
```go
// Hiện tại chỉ so sánh interval đơn thuần:
if sched.LastRunAt != nil && now.Sub(*sched.LastRunAt) < intervalDur-tickSlack {
    continue
}
```
Nếu server khởi động lúc 16:30 chiều và đặt `interval_minutes = 1440` (24h), lần chạy tiếp theo sẽ rơi vào 16:30 chiều hôm sau, **hoàn toàn không thể rơi vào 00:05 sáng**!

### Giải pháp thiết kế: Bổ sung trường `run_at_time` vào `cdc_worker_schedule`
1. **Migration schema:**
   ```sql
   ALTER TABLE cdc_system.cdc_worker_schedule
     ADD COLUMN IF NOT EXISTS run_at_time VARCHAR(8) NULL; -- Format 'HH:mm', ví dụ '00:05'
   ```
2. **Logic tính toán `next_run_at` trong `server_scheduler.go`:**
   - Nếu `sched.RunAtTime != ""` (ví dụ `'00:05'`):
     - Hàm tính toán `CalculateNextDailyRun(now, "00:05")`:
       - Cắt chuỗi lấy `hour = 0`, `minute = 5`.
       - Lấy mốc 00:05 của ngày hôm nay. Nếu thời điểm hiện tại `now` đã vượt qua 00:05 hôm nay $\rightarrow$ `next_run_at` = 00:05:00 sáng ngày mai!
     - Runner mỗi phút kiểm tra: Nếu `now >= next_run_at` $\rightarrow$ Kích hoạt job chạy ngay lúc 00:05!
   - Nếu `sched.RunAtTime == ""`: Tiếp tục dùng `interval_minutes` như cũ (tương thích ngược 100%).
3. **Giao diện UI `/activity-manager` (`ActivityManager.tsx`):**
   - Thêm cột & ô nhập: **"Giờ chạy cố định (Run At Time)"** (TimePicker hoặc Input `HH:mm`, ví dụ `00:05`).

---

## 2. VẤN ĐỀ 2: NGUYÊN NHÂN SOURCE-COUNT KHÔNG LẤY ĐÚNG CREATED_AT & CÁCH SỬA

### Nguyên nhân gốc rễ (Root Cause):
1. Trong mã nguồn Recon hiện tại (`recon_tier_a.go:204`):
   ```go
   primary := tsField(entry)
   if primary == "" {
       primary = "updated_at" // <-- NGUYÊN NHÂN LỆCH SỐ LIỆU!
   }
   ```
   Hệ thống Recon trước đây được thiết kế cho Watermark Delta Replication nên ưu tiên `updated_at` / `lastUpdatedAt`.
2. Hậu quả thực tế:
   - Các bản ghi cũ (tạo từ năm ngoái) nếu hôm qua có phát sinh cập nhật trạng thái $\rightarrow$ `updated_at` nhảy sang ngày hôm qua.
   - Khi source-count đếm theo `updated_at`, các bản ghi này bị **đếm nhầm là bản ghi mới sinh ra trong ngày**, làm sai lệch hoàn toàn bản chất "Số lượng bản ghi tạo mới (Creation Count)"!

### Giải pháp chuẩn hóa:
1. **Tách biệt hoàn toàn `created_at_field`:**
   - Trong bảng cấu hình metadata (`source_object_registry` hoặc `confluence_report_configs.pipelines`):
     - Cấu hình tường minh: `"created_at_field": "createdAt"` (cho CamelCase như `trans_his`) hoặc `"created_at"` (cho snake_case như `payment_bills`).
2. **Câu lệnh Query đếm chuẩn xác 100%:**
   - BẮT BUỘC chỉ truy vấn đúng trên trường `created_at_field` này:
     ```sql
     SELECT COUNT(*) 
     FROM "%[schema]s"."%[table]s"
     WHERE "%[created_at_field]" >= :startOfDay 
       AND "%[created_at_field]" < :endOfDay
       AND (_deleted IS NOT TRUE OR _deleted IS NULL);
     ```
   - **Tuyệt đối CẤM fallback sang `updated_at` hay `lastUpdatedAt`**. Nếu bảng nào không có trường `created_at_field`, hệ thống báo lỗi rõ ràng hoặc chỉ đếm các event CDC có `op = 'c'` (create/insert).

---

## 3. Lộ Trình Triển Khai

1. **Step 1:** Thêm cột `run_at_time` vào bảng `cdc_system.cdc_worker_schedule` và cập nhật thuật toán tính `next_run_at` trong `server_scheduler.go` để chạy chính xác lúc `00:05`.
2. **Step 2:** Thêm trường `created_at_field` vào cấu hình pipeline của từng bảng.
3. **Step 3:** Viết hàm `CountDailyCreatedRecords` trong worker chỉ filter đúng `created_at_field` (không chạm `updated_at`).
4. **Step 4:** Cập nhật giao diện `/activity-manager` hiển thị ô cấu hình giờ chạy cố định `00:05`.
