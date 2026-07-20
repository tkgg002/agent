# Hồ sơ Giải pháp Kỹ thuật Khắc phục 3 Rủi ro High (SINK-H5, TX-H3, TX-H6)

Tài liệu này chứa đặc tả code thay đổi chi tiết để Muscle Agent thực thi trực tiếp vào codebase.

---

## 🛠 SỬA ĐỔI 1: batch_buffer.go
*   **File**: [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
*   **Mục tiêu**: Tích lũy `successfulRecords` trong loop fallback tuần tự và bắn trigger NATS cho các record này khi gặp lỗi transient (thoát sớm) hoặc khi hoàn thành.
*   **Code Diff đề xuất**:

```diff
<<<<
		if txErr != nil {
			// Fallback to sequential execution for this chunk to isolate failures.
			// Counter recounted from per-row RowsAffected — TX rolled back so
			// nothing landed yet.
			chunkWritten = 0
			for _, r := range chunk {
				if r.IsDelete {
					rows, err := bb.executeSoftDelete(db, schemaAdapter, schema, bb.recordSchema(r), r.TableName, effectivePK, r)
					if err != nil {
						bb.logger.Error("soft delete failed",
							zap.String("schema", bb.recordSchema(r)),
							zap.String("table", tableName),
							zap.String("pk", r.PrimaryKeyValue),
							zap.Error(err),
						)
						if isRetryableDBError(err) {
							return written + int(chunkWritten), err
						}
						bb.writeFailedSyncLog(tableName, r, err)
					} else {
						chunkWritten += rows
					}
				} else {
					query, values := schemaAdapter.BuildUpsertSQLInSchema(
						schema, bb.recordSchema(r), r.TableName, effectivePK,
						r.PrimaryKeyValue, r.MappedData,
						r.RawData, r.Source, r.Hash, r.SourceTsMs,
					)
					res := db.Exec(query, values...)
					if res.Error != nil {
						bb.logger.Error("upsert failed",
							zap.String("schema", bb.recordSchema(r)),
							zap.String("table", tableName),
							zap.String("pk", r.PrimaryKeyValue),
							zap.Any("mapped_data", r.MappedData),
							zap.Error(res.Error),
						)
						if isRetryableDBError(res.Error) {
							return written + int(chunkWritten), res.Error
						}
						bb.writeFailedSyncLog(tableName, r, res.Error)
						metrics.SyncFailed.WithLabelValues(tableName, "upsert", r.Source).Inc()
					} else {
						chunkWritten += res.RowsAffected
						metrics.SyncSuccess.WithLabelValues(tableName, "upsert", r.Source).Inc()
					}
				}
			}
====
		if txErr != nil {
			// Fallback to sequential execution for this chunk to isolate failures.
			chunkWritten = 0
			var successfulRecords []*shadow.UpsertRecord

			for _, r := range chunk {
				if r.IsDelete {
					rows, err := bb.executeSoftDelete(db, schemaAdapter, schema, bb.recordSchema(r), r.TableName, effectivePK, r)
					if err != nil {
						bb.logger.Error("soft delete failed",
							zap.String("schema", bb.recordSchema(r)),
							zap.String("table", tableName),
							zap.String("pk", r.PrimaryKeyValue),
							zap.Error(err),
						)
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
					query, values := schemaAdapter.BuildUpsertSQLInSchema(
						schema, bb.recordSchema(r), r.TableName, effectivePK,
						r.PrimaryKeyValue, r.MappedData,
						r.RawData, r.Source, r.Hash, r.SourceTsMs,
					)
					res := db.Exec(query, values...)
					if res.Error != nil {
						bb.logger.Error("upsert failed",
							zap.String("schema", bb.recordSchema(r)),
							zap.String("table", tableName),
							zap.String("pk", r.PrimaryKeyValue),
							zap.Any("mapped_data", r.MappedData),
							zap.Error(res.Error),
						)
						if isRetryableDBError(res.Error) {
							if len(successfulRecords) > 0 && bb.natsConn != nil {
								bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
							}
							return written + int(chunkWritten), res.Error
						}
						bb.writeFailedSyncLog(tableName, r, res.Error)
						metrics.SyncFailed.WithLabelValues(tableName, "upsert", r.Source).Inc()
					} else {
						chunkWritten += res.RowsAffected
						metrics.SyncSuccess.WithLabelValues(tableName, "upsert", r.Source).Inc()
						successfulRecords = append(successfulRecords, r)
					}
				}
			}

			// Bắn trigger transmute cho các bản ghi đã ghi Shadow thành công sau khi kết thúc fallback
			if len(successfulRecords) > 0 && bb.natsConn != nil {
				bb.publishTransmuteTrigger(ctx, schemaName, tableName, successfulRecords)
			}
>>>>
```

---

## 🛠 SỬA ĐỔI 2: transmuter.go
*   **File**: [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
*   **Mục tiêu**: Bổ sung dung sai clock skew cho câu lệnh OCC upsert.
*   **Code Diff đề xuất**:

```diff
<<<<
	sets = append(sets, `"_updated_at" = NOW()`)

	qt := quoteTransmuteQualified(binding.MasterSchema, binding.MasterTable)
	sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
		ON CONFLICT (%s) DO UPDATE SET %s
		WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0)
		RETURNING (xmax = 0) AS inserted`,
		qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "), quoteTransmuteIdent(conflictTarget), strings.Join(sets, ", "), qt)
====
	sets = append(sets, `"_updated_at" = NOW()`)

	// Dung sai clock skew mặc định là 2 giây (2000ms) để khắc phục lệch giờ ở các node nguồn CDC
	const clockSkewToleranceMs = 2000

	qt := quoteTransmuteQualified(binding.MasterSchema, binding.MasterTable)
	sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
		ON CONFLICT (%s) DO UPDATE SET %s
		WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
		RETURNING (xmax = 0) AS inserted`,
		qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "), 
		quoteTransmuteIdent(conflictTarget), strings.Join(sets, ", "), qt, clockSkewToleranceMs)
>>>>
```

---

## 🛠 SỬA ĐỔI 3: transmuter_utils.go
*   **File**: [transmuter_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go)
*   **Mục tiêu**: Thay thế FNV-1a bằng SHA-256 để triệt tiêu va chạm ID mảng.
*   **Code Diff đề xuất**:

```diff
<<<<
import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)
====
import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)
>>>>
```

```diff
<<<<
func deterministicGpayID(shadowGpayID int64, keySuffix string) int64 {
	if keySuffix == "" {
		return shadowGpayID
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(strconv.FormatInt(shadowGpayID, 10)))
	_, _ = h.Write([]byte(keySuffix))
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}
====
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
>>>>
```
