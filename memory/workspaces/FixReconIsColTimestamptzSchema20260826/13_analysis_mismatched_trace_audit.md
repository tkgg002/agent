# 13_analysis_mismatched_trace_audit.md: Phân tích Chuyên sâu Trace & Nguyên nhân Gốc rễ

## 1. Phân tích Dữ liệu Đầu ra & Jaeger Trace `feadd5cb8c4c819569ba1e7587e69e0d`

Dữ liệu kết quả Recon thực tế trên bảng `trans_his`:
- **Thiếu ở Shadow (`MissingFromDest`):** (2 IDs) `6a8e5009d1d2b120fd1e17da`, `6a8e66197c0b8081237dabd5`
- **Thừa ở Shadow (`MissingFromSrc`):** (2 IDs) `6a8e5009d1d2b120fd1e17da`, `6a8e66197c0b8081237dabd5`
- **Lệch dữ liệu (`Mismatched`):** (47 IDs)

---

## 2. Nguyên nhân Gốc rễ (Root Cause)

### Vấn đề 1: Trùng ID ở 2 danh mục "Thiếu ở Shadow" và "Thừa ở Shadow" (Absurd Duplication)
- **Bản chất:** 2 ID này tồn tại ở CẢ HAI DB (Mongo và Postgres).
- **Cơ chế gây lỗi:**
  - Ở Sub-window 15-phút A, Mongo quét thấy ID này nhưng Postgres không thấy $\rightarrow$ Thêm vào `MissingFromDest`.
  - Ở Sub-window 15-phút B, Postgres quét thấy ID này nhưng Mongo không thấy $\rightarrow$ Thêm vào `MissingFromSrc`.
- **Gốc rễ:** `ChunkStreamBucketEngine` thiếu bước **Khử trùng lặp (Deduplication & Reclassification)** ở cấp Engine sau khi quét xong tất cả sub-windows. Đúng ra, ID nào xuất hiện ở cả 2 mảng missing PHẢI được loại bỏ khỏi 2 mảng missing và chuyển sang mảng `Mismatched`.

### Vấn đề 2: 47 IDs bị `Mismatched` do Lệch 7 tiếng (+7h Timezone Shift)
- **Công thức Hash per Record:**
  `xorAcc ^= hashIDPlusTsMs(id, timestampMs)`
- **Tại sao `timestampMs` giữa Mongo và Postgres không khớp nhau?**
  - **Mongo (Source):** BSON `updatedAt` được đọc chuẩn UTC Epoch Milliseconds (ví dụ: `15:00:00.000Z` $\rightarrow$ `1787737200000`).
  - **Shadow Postgres (Dest):** Cột `updatedAt` có kiểu `timestamp without time zone`. CDC Worker ghi giá trị UTC (`15:00:00`) vào Postgres.
  - **Lỗi ở Parser:** Khi `parsePostgresTimestampWithLocationAndType` (`recon_query.go:680-686`) đọc giá trị `15:00:00` từ Postgres với `isTZ = false`:
    Do `dbLoc` là `Asia/Ho_Chi_Minh` (+7h), parser lầm tưởng `15:00:00` là giờ ICT local time, nên tự động **trừ đi 7 tiếng** để quy đổi về UTC (`08:00:00 UTC` $\rightarrow$ `1787712000000`).
  - **Hậu quả:** `timestampMs` phía Postgres bị trôi lệch đúng 7 tiếng (25,200,000 ms) so với Mongo $\rightarrow$ Hash per record bị sai lệch 100% $\rightarrow$ Toàn bộ 47 bản ghi bị đẩy vào danh sách `Mismatched`.

---

## 3. Đề xuất Kế hoạch Giải quyết (Proposed Action Plan)

1. **Khắc phục Lỗi Lệch 7 tiếng trong Parser (`recon_query.go`):**
   - Trong `parsePostgresTimestampWithLocationAndType`: Giá trị `time.Time` do driver PostgreSQL `database/sql` trả về đối với cột `TIMESTAMP` vốn đã mang mốc thời gian UTC tường minh.
   - Bỏ đoạn logic gượng ép tự chuyển đổi `dbLoc` trừ 7 tiếng khi `isTZ = false`. Giữ nguyên `v.UTC()` cho mọi đối tượng `time.Time`.

2. **Bổ sung Khử trùng lặp tại Engine (`recon_stream_bucket_engine.go`):**
   - Bổ sung hàm `deduplicateStaleIDsPayload(stale *StaleIDsPayload)` ở cuối hàm `Execute`.
   - Nếu ID xuất hiện ở cả `MissingFromDest` và `MissingFromSrc`, tự động loại khỏi 2 mảng missing và chuyển sang `Mismatched`.
