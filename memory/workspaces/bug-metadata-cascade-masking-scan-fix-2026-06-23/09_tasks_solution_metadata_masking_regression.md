# Solution Design: Metadata Mapping Cache & Non-String Masking (Elegant Fix)

## 1. Vấn đề hiện tại (Root Cause)
Khi một trường nhạy cảm có kiểu dữ liệu cứng trong database (như `TIMESTAMP`, `DATE`, `INTEGER`, `NUMERIC`, `BOOLEAN`) được cấu hình mã hoá (`IsSensitiveField = true`, ví dụ: `createdAt` dùng `hmac`), việc áp dụng các thuật toán mã hoá thông thường sẽ tạo ra chuỗi bản mã dạng text (như hex string của hmac hoặc `aesv1:...` của aes_gcm).

Nếu ta cố gắng lưu chuỗi bản mã text này vào cột có kiểu dữ liệu cứng (`TIMESTAMP` hoặc `INTEGER`), database sẽ quăng lỗi cast kiểu dữ liệu (SQLSTATE 22007/22P02) và skip toàn bộ dòng dữ liệu, dẫn đến degraded luồng đồng bộ.

Việc viết logic parse chuỗi bản mã và gán fallback ở lớp `Transmuter` (downstream) là một giải pháp **"fix bẩn" (workaround)**, vì:
- Nó che giấu lỗi cấu hình sai kiểu dữ liệu ở lớp trên.
- Nó làm phình to code transmuter bằng các logic kiểm tra định dạng bản mã phức tạp và không đúng nhiệm vụ của transmuter (transmuter chỉ nên dịch chuyển dữ liệu 1-1).
- Bản chất dữ liệu nhạy cảm đã bị ghi sai kiểu vào shadow table trước đó.

## 2. Giải pháp thiết kế thanh lịch (Elegant Design)
Chúng ta sẽ giải quyết triệt để vấn đề tương thích kiểu dữ liệu ngay tại lớp **`DynamicMapper`** (lúc chuẩn bị ghi dữ liệu vào shadow table) thay vì vá víu ở transmuter:

- Nếu một trường được bật `IsSensitiveField = true`:
- Ta kiểm tra kiểu dữ liệu đích (`rule.DataType`) của trường đó.
- Nếu `rule.DataType` thuộc nhóm non-string:
  - Nhóm Datetime (`TIMESTAMP`, `DATE`, `TIMESTAMPTZ`, `DATETIME`)
  - Nhóm Numeric (`INT`, `INTEGER`, `BIGINT`, `SMALLINT`, `NUMERIC`, `DECIMAL`, `FLOAT`, `DOUBLE`, `REAL`)
  - Nhóm Boolean (`BOOL`, `BOOLEAN`)
- Thay vì gọi các chiến lược mã hoá tạo ra chuỗi (hmac, aes_gcm), ta sẽ che giấu thông tin nhạy cảm này bằng cách gán về giá trị an toàn tương thích kiểu:
  - Nếu trường cho phép NULL (`rule.IsNullable == true`): Trả về `nil` (NULL).
  - Nếu không cho phép NULL:
    - Đối với Datetime: Trả về mốc thời gian mặc định an toàn: `1970-01-01 00:00:00 UTC` (`time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)`).
    - Đối với Numeric: Trả về `0`.
    - Đối với Boolean: Trả về `false`.
- Các trường có kiểu dữ liệu string (`VARCHAR`, `TEXT`, v.v.) vẫn được mã hoá bình thường qua `MaskByStrategy`.

### Lợi ích của thiết kế này:
1. **Đúng kiểu dữ liệu ở Shadow & Master**: Dữ liệu được ghi vào shadow và master đều đúng chuẩn kiểu dữ liệu (SQL-friendly), loại bỏ hoàn toàn nguy cơ lỗi cast ở mọi stage.
2. **Transmuter sạch sẽ**: Transmuter chỉ dịch chuyển dữ liệu thô 1-1 từ shadow sang master mà không cần thêm bất kỳ logic parse bản mã bẩn thỉu nào.
3. **Bảo mật tối đa**: Dữ liệu nhạy cảm của các kiểu datetime/numeric vẫn được che giấu hoàn toàn (dưới dạng NULL hoặc mặc định).

---

## 3. Các thay đổi chi tiết

### A. Hoàn tác (Revert) code transmuter về nguyên bản
Chúng ta sẽ loại bỏ hoàn toàn logic fallback và helper tự chế trong `transmuter.go` và `transmuter_utils.go` để giữ code sạch sẽ.

### B. Sửa đổi `dynamic_mapper.go`
Đường dẫn: `centralized-data-service/internal/service/shadow/dynamic_mapper.go`

Cập nhật hàm `maybeMaskColumn`:
```go
func (dm *DynamicMapper) maybeMaskColumn(bindingID int64, rule mastermodel.MappingRule, value interface{}) interface{} {
	if dm.masking == nil {
		return value
	}
	if !rule.IsSensitiveField {
		return value
	}

	// Tránh lỗi cast kiểu dữ liệu khi lưu trữ bản mã hash/encryption vào cột non-string
	dt := strings.ToUpper(strings.TrimSpace(rule.DataType))
	isDateTime := strings.Contains(dt, "TIMESTAMP") || strings.Contains(dt, "DATE") || strings.Contains(dt, "TIME")
	isNumeric := strings.Contains(dt, "INT") || strings.Contains(dt, "NUMERIC") || strings.Contains(dt, "DECIMAL") || strings.Contains(dt, "FLOAT") || strings.Contains(dt, "DOUBLE") || strings.Contains(dt, "REAL")
	isBool := strings.Contains(dt, "BOOL")

	if isDateTime || isNumeric || isBool {
		if rule.IsNullable {
			return nil
		}
		if isDateTime {
			return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		if isNumeric {
			return 0
		}
		if isBool {
			return false
		}
	}

	strategy := rule.MaskStrategy
	if strategy == "" {
		strategy = metadata.MaskStrategyHMAC
	}
	if strategy == metadata.MaskStrategyNone {
		return value
	}
	if strategy == metadata.MaskStrategyJSONMask {
		return dm.masking.MaskJSONFields(bindingID, value)
	}
	return dm.masking.MaskByStrategy(value, strategy)
}
```

## 4. Kế hoạch Verification
1. Viết unit test cho `maybeMaskColumn` trong `dynamic_mapper_test.go` để verify việc gán fallback đúng kiểu dữ liệu cho timestamp/numeric/bool khi bật `IsSensitiveField`.
2. Chạy toàn bộ test của service để đảm bảo pass 100%.
