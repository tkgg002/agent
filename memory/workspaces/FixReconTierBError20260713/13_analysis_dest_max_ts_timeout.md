# Phân tích lỗi `dest max ts: timeout: context deadline exceeded`

## 1. Hiện tượng & Lịch sử lỗi
Trong quá trình chạy đối soát (Reconciliation) cho bảng `schedule_histories` (Segment: `source_shadow`), hệ thống liên tục ghi nhận lỗi:
* **Error Message**: `dest max ts: timeout: context deadline exceeded`
* **Error Code**: `SRC_TIMEOUT` (được phân loại bởi hàm `classifyMongoError` khi phát hiện chuỗi `deadline exceeded`)
* **Checked At**: `2026-07-14 02:14:18.167729` (phiên đối soát gần nhất)

## 2. Nguyên nhân gốc rễ (Root Cause Analysis)

### A. Cơ chế Watermarking & Timestamp Field Resolution
Hàm `resolveSourceAndDestTSFields` phân tích cấu hình cột timestamp của bảng:
1. Trường timestamp được chỉ định trong registry (`source_object_registry.timestamp_field`) cho `schedule_histories` là `lastUpdatedAt`.
2. Do cột `"lastUpdatedAt"` tồn tại trong bảng shadow ở Postgres, hệ thống chọn `"lastUpdatedAt"` làm trường timestamp đối soát cho cả nguồn và đích (`srcTS = lastUpdatedAt`, `dstTS = lastUpdatedAt`).

### B. Truy vấn Tìm Max Timestamp Đích (`MaxWindowTs`)
Hàm `pickScanRangeWithLag` thực hiện truy vấn để lấy giá trị max timestamp hiện tại của Shadow DB nhằm xác định biên thời gian đối soát:
```sql
SELECT MAX("lastUpdatedAt") FROM shadow_testss.schedule_histories;
```
Bảng `shadow_testss.schedule_histories` có quy mô **2,713,345 records**. 

### C. Phân tích Kịch bản Thực thi SQL (Query Plan & Latency)
Trước khi tối ưu, chạy thử nghiệm `EXPLAIN ANALYZE` cho câu truy vấn trên:
```sql
EXPLAIN ANALYZE SELECT MAX("lastUpdatedAt") FROM shadow_testss.schedule_histories;
```
Kết quả trả về:
* **Query Plan**: `Parallel Seq Scan on schedule_histories`
* **Execution Time**: **46,538.704 ms (~46.5 giây)**

Do thời gian thực thi lên tới **46.5 giây**, truy vấn này vượt quá cấu hình `QueryTimeout` mặc định của agent (**30 giây**). Điều này dẫn tới lỗi ngắt kết nối `context deadline exceeded`.

### D. Điểm Khác biệt giữa Master DB và Shadow DB
* **Master DB**: Bảng `master_scheduler_service.schedule_histories` đã được tạo sẵn Index `"ix_schedule_histories_lastUpdatedAt" btree ("lastUpdatedAt")`, giúp các truy vấn max timestamp diễn ra tức thì.
* **Shadow DB**: Bảng `shadow_testss.schedule_histories` **thiếu hoàn toàn** index trên cột `"lastUpdatedAt"`. Chỉ có các index bổ trợ khác như `_source_ts`, `_synced_at`, và `_source_id`.

---

## 3. Giải pháp Khắc phục & Thực nghiệm Kiểm chứng

### A. Khắc phục Bằng Indexing
Để giải quyết triệt để Full Table Scan, ta cần bổ sung Index trên cột timestamp đối soát `"lastUpdatedAt"` trong Shadow DB:
```sql
CREATE INDEX idx_schedule_histories_last_updated_at 
ON shadow_testss.schedule_histories("lastUpdatedAt");
```

### B. Kết quả Thực nghiệm Sau khi Đánh Index
Chạy lại `EXPLAIN ANALYZE` sau khi tạo Index:
* **Query Plan**: `Index Only Scan Backward using idx_schedule_histories_last_updated_at on schedule_histories`
* **Execution Time**: **0.438 ms (chưa tới 1 mili-giây)**

**Hiệu quả**: Tốc độ truy vấn tăng **~106,250 lần**, triệt tiêu hoàn toàn khả năng timeout.

---

## 4. Khuyến nghị Phòng ngừa (Prevention Recommendations)
1. **Kiểm tra/Tự động hóa Tạo Index**: Cần bổ sung vào `IndexManager` hoặc pipeline tạo bảng Shadow/Master tự động phát hiện cột cấu hình `timestamp_field` và tạo index tương ứng.
2. **Override Timeout cho Tác vụ Nền**: Cân nhắc thiết lập cấu hình timeout riêng cho các truy vấn kiểm tra watermark (`MaxWindowTs`) lớn hơn `QueryTimeout` thông thường (ví dụ: 60s) hoặc bóc tách timeout của chặng đối soát từ parent context.
3. **Chuẩn hóa Error Code**: Điều chỉnh logic phân loại lỗi `classifyMongoError` để phân tách rõ ràng lỗi timeout ở Shadow/Master DB (`DST_TIMEOUT`) và Source MongoDB (`SRC_TIMEOUT`).
