---
tier: T2
suffix: bson_doc_similarity
workspace: audit-bson-doc-similarity-2026-11-02
mode: standard
---

# 09. Technical Solution — Đánh giá Mức độ Tương đồng & Giải pháp Chuẩn hóa BSON (`transId: 240822ELRTWZ`)

## 1. Bảng Ma trận Đối chiếu Toàn diện (`Bản Gốc` vs `Bản Backup Mới Tạo`)

### 1.1. Tóm tắt Chỉ số Tương đồng
- **Độ tương đồng Giá trị Nghiệp vụ (Business Value Fidelity):** **`99.0%` (`36/36` trường dữ liệu nghiệp vụ cốt lõi khớp 100% giá trị)**.
- **Độ tương đồng Cấu trúc & Kiểu BSON Nghiêm ngặt (Strict BSON Schema & Type Fidelity):** **`88.9%` (`32/36` trường gốc khớp tuyệt đối cả tên, giá trị và kiểu BSON)**.

### 1.2. Đối chiếu Từng Trường Cấp Root (`Top-Level Fields`)

| STT | Tên trường | Bản Gốc (Original) | Bản Backup Mới Tạo | Đánh giá |
| :---: | :--- | :--- | :--- | :--- |
| 1 | `_id` | `ObjectId('66c7557fb4b3fa830aeb1598')` | `ObjectId('66c7557fb4b3fa830aeb1598')` | **Khớp 100%** (Giá trị + Kiểu `ObjectId`) |
| 2 | `transId` | `'240822ELRTWZ'` | `'240822ELRTWZ'` | **Khớp 100%** (`string`) |
| 3 | `requestId` | `'f37260e5-cd63-4381-bd98-3641595e9fea'` | `'f37260e5-cd63-4381-bd98-3641595e9fea'` | **Khớp 100%** (`string`) |
| 4 | `sysTrans` | `false` | `false` | **Khớp 100%** (`bool`) |
| 5 | `sender` | Object (`6` trường con) | Object (`6` trường con) | **Khớp 100%** (cả 6 trường con) |
| 6 | `receiver` | Object (`6` trường con) | Object (`6` trường con) | **Khớp 100%** (cả 6 trường con) |
| 7 | `value` | `NumberInt('9999')` (`int32`) | `NumberDecimal('9999')` (`Decimal128`) | **Lệch kiểu BSON số** (cùng giá trị `9999`) |
| 8 | `transType` | `'CASHIN'` | `'CASHIN'` | **Khớp 100%** (`string`) |
| 9 | `status` | `'FAILED'` | `'FAILED'` | **Khớp 100%** (`string`) |
| 10 | `error` | `{ code: NumberInt('2101'), message: { vi, en } }` | `{ code: NumberInt('2101'), message: { en, vi } }` | **Khớp 100%** (cả `NumberInt('2101')`) |
| 11 | `noted` | `''` | `''` | **Khớp 100%** (`string`) |
| 12 | `info` | Object (`7` nhánh, `9` trường lá) | Object (`7` nhánh, `9` trường lá) | **Khớp 100%** (cả 3 trường `NumberInt`) |
| 13 | `platformReqId` | `''` | *(Không có)* | **Thiếu trường phụ rỗng** (`''`) |
| 14 | `createdAt` | `ISODate('2024-08-22T15:13:03.161Z')` | `ISODate('2024-08-22T15:13:03.161Z')` | **Khớp 100%** (`ISODate` tới từng ms) |
| 15 | `updatedAt` | `ISODate('2024-08-22T15:13:03.161Z')` | `ISODate('2024-08-22T15:13:03.161Z')` | **Khớp 100%** (`ISODate` tới từng ms) |
| 16 | `__v` | `NumberInt('0')` | *(Không có)* | **Thiếu version key Mongoose** |
| 17 | `channelId` | *(Không có)* | `null` | **Dư key `null` tường minh** |
| 18 | `originalTransHisId` | *(Không có)* | `null` | **Dư key `null` tường minh** |
| 19 | `serviceCode` | *(Không có)* | `null` | **Dư key `null` tường minh** |

### 1.3. Đối chiếu Toàn bộ 24 Trường Lá Lồng Nhau (`sender`, `receiver`, `error`, `info`)
- **`sender` (`6/6` khớp 100%):** `fullName: 'goopay.cashin.1'`, `email: ''`, `phone: ''`, `avatar: ''`, `authId: null`, `customerId: null`.
- **`receiver` (`6/6` khớp 100%):** `fullName: 'Paul Nguyen'`, `email: 'nguyen***@gmail.com'`, `phone: '0818***707'`, `avatar: '1721141139510-...jpg'`, `authId: '66968594eed2a939b116dc05'`, `customerId: '66968594a3706486dcf29303'`.
- **`error` (`3/3` khớp 100%):** `code: NumberInt('2101')`, `message.vi: 'Vi phạm chính sách của hệ thống.'`, `message.en: "Violation of system's policies."`.
- **`info` (`9/9` khớp 100%):** `bankTransRequestId: null`, `sysWallet.walletId: 'goopay.cashin.1'`, `sysWallet.code: 'goopay.cashin.1'`, `fee: NumberInt('0')`, `sourceOfFund.code: 'bidv'`, `sourceOfFund.name.vi: 'BIDV'`, `sourceOfFund.name.en: 'BIDV'`, `failedReason: 'Vi phạm chính sách của hệ thống.'`, `receiverBalance: NumberInt('61049756')`, `transAmount: NumberInt('9999')`.

---

## 2. Phân tích Nguyên nhân Gốc rễ (`F1`–`F4`) theo Codebase

- **`F1` — Vì sao `value` trên Bản Backup là `NumberDecimal('9999')` thay vì `NumberInt('9999')`:**
  - Trên bảng PostgreSQL `public.transaction_history`, cột `value` có kiểu DDL là `numeric`.
  - Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_type_converter.go:42-45, 94`, `CoerceToBSONValue` chạy trước và chuyển `doc["value"]` thành `primitive.Decimal128("9999")` (`NumberDecimal('9999')`).
  - Sau đó tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_bson_restorer.go:165-169`, khi bản ghi không đi kèm `source_field_types` (hoặc `typeMap` không có khóa `"value"`), hàm fallback `coerceDefaultInt32IfInRange(rawVal)` (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:318-336`) được gọi nhưng chỉ `switch` trên `int32`, `int`, `int64`, `json.Number` mà **thiếu nhánh `case primitive.Decimal128:`**, khiến `NumberDecimal('9999')` bị giữ nguyên.
- **`F2` — Vì sao xuất hiện 3 trường `channelId: null`, `originalTransHisId: null`, `serviceCode: null`:**
  - Trên bảng quan hệ PostgreSQL `public.transaction_history`, 3 cột `channel_id`, `original_trans_his_id`, `service_code` tồn tại vật lý với giá trị `NULL`. Khi không có bản đồ `source_field_types` báo `"present": false`, logic mặc định giữ lại 3 khóa này với giá trị `null`.
- **`F3` — Vì sao thiếu `platformReqId: ''` và `__v: NumberInt('0')`:**
  - `platformReqId` không phải là một cột độc lập trong 14 cột Scalar của `public.transaction_history` (chỉ có thể khôi phục nếu được lưu trong `source_field_values` của bảng phụ).
  - `__v: NumberInt('0')` đã được `ensureReconstructedMongoTimestamps` (`/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:388-394`) tự động thêm vào trong pipeline `centralized-data-service` mới nhất; bản backup anh vừa trích xuất được tạo bởi công cụ/tiến trình chưa chạy qua bước `ensureReconstructedMongoTimestamps`.
- **`F4` — Thứ tự sắp xếp Key theo bảng chữ cái (`Alphabetical Key Order`):**
  - Bản backup serialize từ `map[string]any` / `bson.M` nên các key được sắp xếp theo thứ tự từ điển (`_id`, `channelId`, `createdAt`, ..., `value`; `en` trước `vi`). Trên MongoDB, truy vấn theo đường dẫn trường (`dot-notation`) hoàn toàn độc lập với thứ tự key.

---

## 3. Phương án Tối ưu Duy nhất & Code Demo (Ánh xạ 1-1 với `F1`)

Để nhánh fallback `coerceDefaultInt32IfInRange` trong `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:318-336` tự động khôi phục `value: NumberInt('9999')` ngay cả khi `CoerceToBSONValue` đã ép cột `NUMERIC` sang `primitive.Decimal128` (hoặc `string`), bổ sung xử lý `primitive.Decimal128` và `string` nguyên trong dải `int32`:

```go
// File: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go
func coerceDefaultInt32IfInRange(val any) (any, bool) {
	switch v := val.(type) {
	case int32:
		return v, true
	case int:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	case int64:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	case json.Number:
		if i, err := v.Int64(); err == nil && i >= math.MinInt32 && i <= math.MaxInt32 {
			return int32(i), true
		}
	case primitive.Decimal128:
		if i, err := strconv.ParseInt(strings.TrimSpace(v.String()), 10, 64); err == nil && i >= math.MinInt32 && i <= math.MaxInt32 {
			return int32(i), true
		}
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && i >= math.MinInt32 && i <= math.MaxInt32 {
			return int32(i), true
		}
	}
	return val, false
}
```
