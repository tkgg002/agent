# Phân tích kỹ thuật sửa lỗi đối soát Full Search (Full Diff)

## 1. Bản chất sự cố
Khi thực hiện đối soát ở chế độ `full_diff` (Tier 2):
1. Hệ thống chạy `TimeBoundedDiffMissingFromShadow` để so sánh trực tiếp danh sách ID giữa Source và Shadow.
2. Để lọc dữ liệu trong khoảng thời gian `[startTime, endTime)`, hệ thống truy vấn Shadow DB (Postgres) và Source DB.
3. Trường timestamp đích `dstTS` được giải quyết bằng `resolveSourceAndDestTSFields`. Đối với nguồn là MongoDB, `dstTS` thường được fallback về `_source_ts` (đại diện cho thời gian bản ghi được Debezium bắt sự kiện và đẩy về Kafka/Shadow DB, kiểu dữ liệu `BIGINT`).
4. Truy vấn SQL vào Shadow DB:
   ```sql
   SELECT "_source_id"::text FROM shadow_table WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND _source_ts >= ? AND _source_ts < ?
   ```
   Do tham số truyền vào là `time.Time` (`startTime` và `endTime`), GORM/driver Postgres định dạng các tham số này thành timestamp có múi giờ (e.g. `'2026-07-06 10:20:00+00'`).
5. Postgres ném lỗi hoặc không thể so sánh kiểu dữ liệu `BIGINT` (`_source_ts`) với `TIMESTAMP WITH TIME ZONE` mà không có ép kiểu rõ ràng, dẫn đến truy vấn trả về rỗng hoặc báo lỗi:
   ```
   operator does not exist: bigint >= timestamp with time zone
   ```
6. Điều này làm cho danh sách `shadowIDs` rỗng, dẫn đến kết quả đối soát lệch/không chính xác hoặc báo lỗi rỗng.

## 2. Giải pháp khắc phục đề xuất
Ta cần tạo ra cơ chế tự động phân giải kiểu dữ liệu (data type) của cột timestamp trong Postgres. Nếu cột có kiểu dữ liệu là số (`BIGINT`, `INTEGER`, `NUMERIC`, v.v.), tham số thời gian truyền vào phải được chuyển thành dạng epoch milliseconds hoặc epoch seconds (dựa trên giá trị thực tế của cột trong bảng).

### 2.1. Helper phân giải tham số timestamp cho Postgres
Chúng ta sẽ thiết kế một hàm helper hoặc khối code để phân giải kiểu dữ liệu và sinh ra tham số thời gian phù hợp:
```go
func resolvePostgresTimeParams(ctx context.Context, db *gorm.DB, tableName, columnName string, tLo, tHi time.Time) (interface{}, interface{}, error) {
	var schemaName, bareTable string
	if i := strings.IndexByte(tableName, '.'); i > 0 {
		schemaName = tableName[:i]
		bareTable = tableName[i+1:]
	} else {
		schemaName = "public"
		bareTable = tableName
	}

	var dataType string
	err := db.WithContext(ctx).Raw(`
		SELECT data_type FROM information_schema.columns 
		WHERE table_schema = ? AND table_name = ? AND column_name = ?
	`, schemaName, bareTable, columnName).Scan(&dataType).Error
	if err != nil {
		return tLo, tHi, err
	}
	dataType = strings.ToLower(dataType)

	if strings.Contains(dataType, "int") || strings.Contains(dataType, "num") || columnName == "_source_ts" {
		isEpochMillis := true
		// Heuristic: kiểm tra xem giá trị max có nằm trong khoảng epoch seconds (< 1e11) không
		var maxVal int64
		sqlMax := fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM %s`, quoteIdent(columnName), quoteRelation(tableName))
		if err := db.WithContext(ctx).Raw(sqlMax).Scan(&maxVal).Error; err == nil {
			if maxVal > 0 && maxVal < 1e11 {
				isEpochMillis = false
			}
		}
		if isEpochMillis {
			return tLo.UnixMilli(), tHi.UnixMilli(), nil
		}
		return tLo.Unix(), tHi.Unix(), nil
	}

	return tLo, tHi, nil
}
```

### 2.2. Áp dụng giải pháp vào các file nguồn
1. **`internal/service/recon/recon_tier_a.go`**:
   Trong `TimeBoundedDiffMissingFromShadow`, sử dụng helper để tính toán `startVal` và `endVal` dựa trên kiểu dữ liệu của `dstTS`.
2. **`internal/service/recon/recon_stream.go`**:
   - Trong `listIDsInWindowPostgres`, phân giải `tLo` và `tHi` dựa trên kiểu dữ liệu của `tsField`.
   - Trong `streamIDsPostgresInTimeRange`, phân giải `startTime` và `endTime` tương tự dựa trên kiểu dữ liệu của `timestampField`.

## 3. So sánh các cơ chế đối soát: full_diff, hash_window và Deep Check

| Tiêu chí | `hash_window` (Auto/Periodic) | `full_diff` (Manual/Full Search) | `Deep Check` (Segment B Detail) |
| :--- | :--- | :--- | :--- |
| **Mục tiêu** | Phát hiện lệch nhanh chặng Source ↔ Shadow. | Đối soát triệt để một chiều, tìm bản ghi thiếu ở Shadow. | So sánh chi tiết từng trường dữ liệu chặng Shadow ↔ Master. |
| **Cơ chế quét** | So sánh XOR Hash theo từng bucket nhỏ (1h). Chỉ drill-down bucket bị lệch. | Quét trực tiếp, tải toàn bộ IDs Shadow trong window để so khớp với Source. | Lấy các ID bị lệch, so sánh giá trị dự kiến (từ raw_data) với master. |
| **Cửa sổ (Window)** | Mặc định 7 ngày gần nhất (tự động chạy định kỳ). | Tối đa 30 ngày (do người dùng chủ động chọn từ UI). | Theo lookback window của Segment B (mặc định 24h hoặc custom). |
| **Tính trọn vẹn** | Chỉ quét và liệt kê chi tiết các bản ghi trong bucket bị lệch XOR Hash. | Quét **đầy đủ và triệt để 100%** dữ liệu của cửa sổ được yêu cầu. | Quét chi tiết tất cả các bản ghi bị lệch được phát hiện trong Segment B. |

### Đảm bảo đủ window đối với `full_diff`:
- Trước khi sửa lỗi: Do lỗi kiểu dữ liệu timestamp (`BIGINT` vs `TIMESTAMP WITH TIME ZONE`), câu truy vấn Shadow DB trả về rỗng, khiến `full_diff` không thể so khớp dữ liệu (hiểu lầm là Shadow trống). Do đó không đảm bảo độ bao phủ của window.
- Sau khi sửa lỗi: Nhờ cơ chế chuyển đổi tham số động (`resolvePostgresTimeParams`), các truy vấn thời gian sẽ khớp chính xác kiểu dữ liệu thực tế của cột. Điều này đảm bảo `full_diff` sẽ quét đầy đủ và chính xác dữ liệu trong toàn bộ cửa sổ thời gian được chọn từ giao diện.
