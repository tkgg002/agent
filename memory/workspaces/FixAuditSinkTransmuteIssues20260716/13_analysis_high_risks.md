# Phân tích kỹ thuật 3 rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này ghi nhận phân tích chi tiết root cause và giải pháp đề xuất cho 3 rủi ro High còn lại.

---

## 1. Phân tích SINK-H5: Batch rollback + sequential fallback

### Root Cause
Khi flush batch buffer gặp lỗi ghi xuống DB, transaction cho chunk 500 records bị rollback toàn bộ. Hệ thống chuyển sang cơ chế fallback tuần tự (`sequential fallback`) chạy từng dòng đơn lẻ bên ngoài transaction để cô lập dòng lỗi (dòng lỗi đẩy vào DLQ, dòng đúng ghi thành công).
Tuy nhiên, nếu quá trình ghi tuần tự gặp lỗi transient (mất kết nối DB tạm thời), hàm `batchUpsert` trả về lỗi và thoát sớm:
```go
if isRetryableDBError(res.Error) {
    return written + int(chunkWritten), res.Error
}
```
Tại tầng caller (`Flush`), khi nhận `gerr != nil`, khối bắn tin trigger transmute sang Master DB bị bỏ qua hoàn toàn:
```go
// Flush()
groupWritten, gerr := bb.batchUpsert(ctx, recs)
if gerr == nil {
    ...
    if bb.natsConn != nil {
        bb.publishTransmuteTrigger(ctx, records[0].SchemaName, records[0].TableName, records)
    }
}
```
*   **Hậu quả**: Một phần dữ liệu (đã ghi Shadow thành công trước khi gặp transient error) nằm ở Shadow DB, nhưng Master DB **không bao giờ nhận được trigger để xử lý** -> Mất đồng bộ Shadow ↔ Master.

### Giải pháp đề xuất
1.  Khai báo mảng `successfulRecords []*shadow.UpsertRecord` để tích lũy các bản ghi ghi shadow thành công trong loop fallback.
2.  Khi gặp transient error và phải thoát sớm, thực hiện bắn trigger cho các bản ghi đã thành công:
    ```go
    if len(successfulRecords) > 0 && bb.natsConn != nil {
        bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
    }
    ```
3.  Khi hoàn thành fallback thành công (hoặc chỉ lỗi permanent), chỉ truyền `successfulRecords` vào `publishTransmuteTrigger` thay vì truyền toàn bộ records.

---

## 2. Phân tích TX-H3: OCC timestamp comparison (Clock skew)

### Root Cause
Tại Master DB, transmuter chạy câu lệnh SQL upsert dạng:
```sql
INSERT INTO master_table (...) VALUES (...)
ON CONFLICT (conflict_target) DO UPDATE SET ...
WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(master_table._source_ts, 0)
```
*   **Hậu quả**: Nếu các node nguồn CDC bị lệch múi giờ hoặc clock skew nhẹ (e.g. Node 1 chạy nhanh hơn Node 2), một cập nhật mới hơn được Node 2 phát ra (nhưng có timestamp nhỏ hơn Node 1 do clock skew) sẽ bị Postgres từ chối cập nhật im lặng do vi phạm mệnh đề `WHERE` so sánh timestamp. Dữ liệu trên Master DB bị kẹt ở bản cũ của Node 1.

### Giải pháp đề xuất
Bổ sung một dung sai thời gian (Tolerance Window) cho so sánh OCC. Thay vì so sánh cứng `>= master_table._source_ts`, ta cho phép cập nhật nếu timestamp mới nằm trong khoảng dung sai lệch giờ chấp nhận được (mặc định là 2 giây / 2000ms):
```sql
WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(master_table._source_ts, 0) - 2000
```
Để an toàn hơn, transmuter sẽ hỗ trợ config `ClockSkewToleranceMs` (tải từ AppConfig) để điều chỉnh động tùy thuộc độ ổn định NTP của hạ tầng.

---

## 3. Phân tích TX-H6: FNV hash collision trong flatten

### Root Cause
Để tạo khóa chính `int64` duy nhất và ổn định cho các record con được bóc tách từ mảng (flatten), hàm `deterministicGpayID` băm chuỗi `shadowGpayID + suffix` sử dụng FNV-1a 64-bit và mask bit dấu để giữ số dương:
```go
h := fnv.New64a()
h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
h.Write([]byte(keySuffix))
return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
```
*   **Hậu quả**: FNV-1a là một thuật toán băm chất lượng trung bình, dễ bị va chạm khi quy mô dữ liệu lớn hoặc với các chuỗi tuần tự ngắn (e.g. index mảng `#0`, `#1`, `#2`...). Sự va chạm khóa chính làm bản ghi con của tài liệu này ghi đè phá hủy dữ liệu của bản ghi con của tài liệu khác (Silent Data Overwrite).

### Giải pháp đề xuất
Thay thế FNV-1a bằng thuật toán băm chất lượng cao hơn: **SHA-256**.
```go
import "crypto/sha256"

func deterministicGpayID(shadowGpayID int64, keySuffix string) int64 {
	if keySuffix == "" {
		return shadowGpayID
	}
	h := sha256.New()
	_, _ = h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
	_, _ = h.Write([]byte(keySuffix))
	sum := h.Sum(nil)
	val := binary.BigEndian.Uint64(sum[:8])
	return int64(val & 0x7FFFFFFFFFFFFFFF)
}
```
SHA-256 đảm bảo sự phân bố băm cực kỳ đồng đều, triệt tiêu hoàn toàn va chạm do cấu trúc tuần tự và đạt tỉ lệ va chạm ngẫu nhiên thấp nhất có thể trong giới hạn toán học của không gian 63-bit.
