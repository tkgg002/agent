# Walkthrough - Kết quả Khắc phục 3 Rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này mô tả chi tiết các thay đổi code đã thực hiện và kết quả kiểm thử.

---

## 🛠 Chi tiết các thay đổi code

### 1. File: `internal/handler/shadow/batch_buffer.go`
*   **Vấn đề (SINK-H5)**: Khi fallback sang ghi sequential từng record, nếu gặp lỗi db transient ở giữa chừng, hàm thoát sớm và không bắn trigger transmute cho các record đã ghi Shadow thành công trước đó.
*   **Giải pháp**: 
    - Khởi tạo mảng `successfulRecords []*shadow.UpsertRecord`.
    - Trong vòng lặp sequential, nếu ghi thành công (cho cả soft delete và upsert), append record vào `successfulRecords`.
    - Khi gặp lỗi transient db error (`isRetryableDBError`), bắn trigger transmute bằng `publishTransmuteTrigger` cho các record đã ghi thành công trước khi return error.
    - Bắn trigger transmute cho các record thành công sau khi kết thúc vòng lặp fallback sequential bình thường.

### 2. File: `internal/service/master/transmuter.go`
*   **Vấn đề (TX-H3)**: Điều kiện OCC `EXCLUDED._source_ts >= master_table._source_ts` quá khắt khe, dễ gây drop update nếu clock ở các node nguồn CDC bị lệch nhỏ.
*   **Giải pháp**:
    - Khai báo hằng số dung sai clock skew `clockSkewToleranceMs = 2000` (2 giây).
    - Cập nhật mệnh đề `WHERE` trong câu lệnh OCC bulk upsert SQL thành:
      `WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d`

### 3. File: `internal/service/master/transmuter_utils.go`
*   **Vấn đề (TX-H6)**: FNV-1a 64-bit dễ xảy ra va chạm ID mảng khi flatten số lượng bản ghi lớn trong cùng một master transaction.
*   **Giải pháp**:
    - Sử dụng thuật toán SHA-256 để băm dữ liệu kết hợp của shadow `_gpay_id` và `keySuffix`.
    - Lấy 8 byte đầu của mã SHA-256 băm được, chuyển đổi sang số nguyên không dấu 64-bit bằng `binary.BigEndian.Uint64` và đảm bảo kết quả luôn dương (int63) để tương thích khóa chính master DB.

---

## 🧪 Kết quả Kiểm thử (Testing Verification)

Sau khi áp dụng thay đổi, chúng tôi đã tiến hành chạy unit test cụ thể cho 2 packages bị ảnh hưởng:
- `centralized-data-service/internal/handler/shadow`
- `centralized-data-service/internal/service/master`

### Kết quả chạy lệnh test:
Lệnh chạy: `go test -v ./internal/handler/shadow/... ./internal/service/master/...`

**Kết quả: PASS 100%**
```
ok  	centralized-data-service/internal/handler/shadow	3.195s
ok  	centralized-data-service/internal/service/master	0.526s
ok  	centralized-data-service/internal/service/master/transmute	(cached)
```

Tất cả các unit tests hiện tại bao gồm kiểm thử logic ghi batch buffer, cơ chế transmuter (Soft Delete, Chunking) và phân phối khóa deterministic (`deterministicGpayID`) đều vượt qua thành công mà không gây ra bất kỳ regression nào.
