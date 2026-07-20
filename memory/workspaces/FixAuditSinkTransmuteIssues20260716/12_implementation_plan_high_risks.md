# Kế hoạch Triển khai Khắc phục 3 Rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này trình bày thiết kế chi tiết và kế hoạch triển khai để khắc phục các rủi ro High còn lại.

---

## 1. Thay đổi chi tiết trong Codebase

### ⚙️ Component: Centralized Data Service (Shadow Buffer & Transmuter)

---

#### 🛠 [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
*   **Mục tiêu**: Đảm bảo đồng bộ trigger transmute khi chạy fallback tuần tự.
*   **Thiết kế chi tiết**:
    1.  Khai báo `successfulRecords := make([]*shadow.UpsertRecord, 0, len(chunk))` bên trong scope fallback.
    2.  Khi một bản ghi ghi shadow thành công (soft delete hoặc upsert): append vào `successfulRecords`.
    3.  Khi gặp transient error và phải dừng sớm: gọi `bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)` trước khi `return`.
    4.  Khi kết thúc fallback thành công: chỉ gọi trigger cho các bản ghi trong `successfulRecords`.

```go
// Minh họa thay đổi logic trong loop fallback tuần tự:
var successfulRecords []*shadow.UpsertRecord

for _, r := range chunk {
    if r.IsDelete {
        rows, err := bb.executeSoftDelete(db, schemaAdapter, schema, bb.recordSchema(r), r.TableName, effectivePK, r)
        if err != nil {
            if isRetryableDBError(err) {
                if len(successfulRecords) > 0 && bb.natsConn != nil {
                    bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
                }
                return written + int(chunkWritten), err
            }
            bb.writeFailedSyncLog(tableName, r, err)
        } else {
            chunkWritten += rows
            successfulRecords = append(successfulRecords, r)
        }
    } else {
        query, values := schemaAdapter.BuildUpsertSQLInSchema(...)
        res := db.Exec(query, values...)
        if res.Error != nil {
            if isRetryableDBError(res.Error) {
                if len(successfulRecords) > 0 && bb.natsConn != nil {
                    bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
                }
                return written + int(chunkWritten), res.Error
            }
            bb.writeFailedSyncLog(tableName, r, res.Error)
        } else {
            chunkWritten += res.RowsAffected
            successfulRecords = append(successfulRecords, r)
        }
    }
}

// Ở cuối hàm batchUpsert (khi không gặp lỗi transient):
if len(successfulRecords) > 0 && bb.natsConn != nil {
    bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
}
```

---

#### 🛠 [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
*   **Mục tiêu**: Bổ sung dung sai clock skew cho câu lệnh OCC upsert.
*   **Thiết kế chi tiết**:
    1.  Khai báo hằng số dung sai `const defaultClockSkewToleranceMs = 2000` (hoặc cấu hình động nếu cần mở rộng sau).
    2.  Cập nhật chuỗi query SQL upsert master:
    ```go
    sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
        ON CONFLICT (%s) DO UPDATE SET %s
        WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
        RETURNING (xmax = 0) AS inserted`,
        qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "), 
        quoteTransmuteIdent(conflictTarget), strings.Join(sets, ", "), 
        qt, defaultClockSkewToleranceMs)
    ```

---

#### 🛠 [MODIFY] [transmuter_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go)
*   **Mục tiêu**: Thay thế FNV-1a bằng SHA-256 để tránh va chạm ID mảng.
*   **Thiết kế chi tiết**:
    ```diff
    - import "hash/fnv"
    + import (
    +     "crypto/sha256"
    +     "encoding/binary"
    + )

    func deterministicGpayID(shadowGpayID int64, keySuffix string) int64 {
        if keySuffix == "" {
            return shadowGpayID
        }
    -   h := fnv.New64a()
    -   _, _ = h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
    -   _, _ = h.Write([]byte(keySuffix))
    -   return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
    +   h := sha256.New()
    +   _, _ = h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
    +   _, _ = h.Write([]byte(keySuffix))
    +   sum := h.Sum(nil)
    +   val := binary.BigEndian.Uint64(sum[:8])
    +   return int64(val & 0x7FFFFFFFFFFFFFFF)
    }
    ```

---

## 2. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

### Kiểm thử tự động (Automated Tests)
1.  **Unit Test cho SINK-H5**:
    *   Giả lập quá trình ghi Shadow DB: bulk write lỗi, chuyển sang fallback tuần tự.
    *   Mock DB để trả lỗi transient tại bản ghi thứ 3 trong chunk.
    *   Xác minh (Assert) NATS trigger transmute được bắn chính xác 2 bản ghi đầu đã ghi shadow thành công.
2.  **Unit Test cho TX-H3**:
    *   Tạo bản ghi Master có `_source_ts = 1000`.
    *   Truyền bản ghi mới có `_source_ts = 900` (nhỏ hơn 100ms, nằm trong dung sai 2000ms).
    *   Xác minh bản ghi mới được cập nhật đè thành công lên DB.
    *   Truyền tiếp bản ghi mới có `_source_ts = -1500` (nhỏ hơn 2500ms, ngoài dải dung sai).
    *   Xác minh bản ghi này bị skip không cập nhật.
3.  **Unit Test cho TX-H6**:
    *   Chạy thử nghiệm băm SHA-256 cho 10 triệu records con (index mảng `#0` đến `#10000000`).
    *   Assert tính ổn định (luôn cho cùng kết quả khi băm lại) và không phát sinh bất kỳ va chạm nào.
