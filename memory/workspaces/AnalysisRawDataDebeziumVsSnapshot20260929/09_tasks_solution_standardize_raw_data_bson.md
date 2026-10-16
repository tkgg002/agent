# 09 Technical Solution: Demo Code Chi Tiết Đồng Bộ Debezium và Bảo Toàn BSON Wrapper cho JSONB

Tài liệu này ghi nhận hồ sơ giải pháp kỹ thuật chi tiết theo quy tắc Rule #4 (No Shadow Files) và Rule #13 (Brain Code Prohibition).

---

## 1. TỔNG HỢP CÁC FILE CẦN EDIT

| STT | File cần chỉnh sửa | Mục đích thay đổi |
| :--- | :--- | :--- |
| **1** | `centralized-data-service/internal/handler/shadow/kafka_consumer.go` | Ngăn `unwrapAvroUnionMap` bóc nhầm payload JSON string từ MongoDB Debezium. |
| **2** | `centralized-data-service/internal/handler/shadow/avro_helper.go` | Thêm defensive guard trong `unwrapAvroUnion` để không bao giờ bóc map có key bắt đầu bằng `$`. |
| **3** | `centralized-data-service/internal/service/shadow/schema_adapter_coerce.go` | Loại bỏ việc ép phẳng `normalizeMongoExtendedJSON` khi ghi dữ liệu vào cột `JSONB` ở Shadow DB. |
| **4** | `centralized-data-service/internal/service/master/transmuter.go` | Bỏ qua `unwrapMongoExtJSON` đối với cột đích kiểu `JSONB` để bảo toàn wrapper BSON sang Master DB. |
| **5** | `centralized-data-service/test/internal/service/schema_adapter_coerce_test.go` | Cập nhật unit test kiểm chứng bảo toàn wrapper BSON trong cột `JSONB`. |

---

## 2. CHI TIẾT CODE THAY ĐỔI TỪNG FILE (BEFORE VS AFTER)

### File 1: `centralized-data-service/internal/handler/shadow/kafka_consumer.go`
- **Vị trí:** Dòng 598–617
- **Hiện tại (Before):**
  ```go
  		switch v := afterRaw.(type) {
  		case map[string]interface{}:
  			afterData = v
  		case string:
  			json.Unmarshal([]byte(v), &afterData)
  		}

  		switch v := beforeRaw.(type) {
  		case map[string]interface{}:
  			beforeData = v
  		case string:
  			json.Unmarshal([]byte(v), &beforeData)
  		}

  		if afterData != nil {
  			afterData = unwrapAvroUnionMap(afterData)
  		}
  		if beforeData != nil {
  			beforeData = unwrapAvroUnionMap(beforeData)
  		}
  ```
- **Thay đổi thành (After):**
  ```go
  		switch v := afterRaw.(type) {
  		case map[string]interface{}:
  			afterData = unwrapAvroUnionMap(v)
  		case string:
  			json.Unmarshal([]byte(v), &afterData)
  		}

  		switch v := beforeRaw.(type) {
  		case map[string]interface{}:
  			beforeData = unwrapAvroUnionMap(v)
  		case string:
  			json.Unmarshal([]byte(v), &beforeData)
  		}
  ```
- **Giải thích:** Với nguồn quan hệ qua Avro (`afterRaw` là map), ta unwrap Avro union bình thường. Với nguồn MongoDB (`afterRaw` là JSON string), sau khi `json.Unmarshal`, dữ liệu đã ở dạng map chuẩn và chứa các wrapper BSON (`$oid`, `$date`), ta giữ nguyên vẹn mà không gọi `unwrapAvroUnionMap`.

---

### File 2: `centralized-data-service/internal/handler/shadow/avro_helper.go`
- **Vị trí:** Dòng 67–77
- **Hiện tại (Before):**
  ```go
  // unwrapAvroUnion extracts value from goavro union type.
  func unwrapAvroUnion(v interface{}) interface{} {
  	if v == nil {
  		return nil
  	}
  	if m, ok := v.(map[string]interface{}); ok && len(m) == 1 {
  		for _, val := range m {
  			return val
  		}
  	}
  	return v
  }
  ```
- **Thay đổi thành (After):**
  ```go
  // unwrapAvroUnion extracts value from goavro union type.
  func unwrapAvroUnion(v interface{}) interface{} {
  	if v == nil {
  		return nil
  	}
  	if m, ok := v.(map[string]interface{}); ok && len(m) == 1 {
  		for k, val := range m {
  			// Guard: MongoDB BSON Extended JSON wrappers ($oid, $date, $numberLong, etc.)
  			// are never Avro union types (Avro type names cannot start with '$').
  			if strings.HasPrefix(k, "$") {
  				return v
  			}
  			return val
  		}
  	}
  	return v
  }
  ```
- **Giải thích:** Phòng vệ đa tầng (Defense-in-depth). Ký tự `$` không bao giờ là tên kiểu dữ liệu hợp lệ trong Avro Union Spec. Nếu map có 1 key và key đó bắt đầu bằng `$`, hàm sẽ giữ nguyên vẹn BSON ExtJSON.

---

### File 3: `centralized-data-service/internal/service/shadow/schema_adapter_coerce.go`
- **Vị trí:** Dòng 54–82
- **Hiện tại (Before):**
  ```go
  	if !sa.IsJSONB(schema, colName) {
  		return val
  	}

  	switch v := val.(type) {
  	case string:
  		if decoded := decodeBase64JSON(v); len(decoded) > 0 {
  			return string(decoded)
  		}
  		if json.Valid([]byte(v)) {
  			var parsed interface{}
  			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
  				normalized := normalizeMongoExtendedJSON(parsed)
  				if marshaled, err := json.Marshal(normalized); err == nil {
  					return string(marshaled)
  				}
  			}
  			return v
  		}
  		jsonVal, _ := json.Marshal(v)
  		return string(jsonVal)
  	case map[string]interface{}, []interface{}:
  		jsonVal, _ := json.Marshal(normalizeMongoExtendedJSON(v))
  		return string(jsonVal)
  	default:
  		jsonVal, _ := json.Marshal(normalizeMongoExtendedJSON(v))
  		return string(jsonVal)
  	}
  ```
- **Thay đổi thành (After):**
  ```go
  	if !sa.IsJSONB(schema, colName) {
  		return val
  	}

  	switch v := val.(type) {
  	case string:
  		if decoded := decodeBase64JSON(v); len(decoded) > 0 {
  			return string(decoded)
  		}
  		if json.Valid([]byte(v)) {
  			return v
  		}
  		jsonVal, _ := json.Marshal(v)
  		return string(jsonVal)
  	case map[string]interface{}, []interface{}:
  		jsonVal, _ := json.Marshal(v)
  		return string(jsonVal)
  	default:
  		jsonVal, _ := json.Marshal(v)
  		return string(jsonVal)
  	}
  ```
- **Giải thích:** Loại bỏ hoàn toàn `normalizeMongoExtendedJSON(v)` khi lưu vào cột `JSONB`. Mọi wrapper BSON (`$oid`, `$date`, `$numberLong`...) bên trong sub-documents/arrays của cột JSONB đều được giữ nguyên 100%.

---

### File 4: `centralized-data-service/internal/service/master/transmuter.go`
- **Vị trí:** Dòng 1272–1274
- **Hiện tại (Before):**
  ```go
  		val := gjsonValueToGo(gres)
  		val = unwrapMongoExtJSON(val)
  ```
- **Thay đổi thành (After):**
  ```go
  		val := gjsonValueToGo(gres)
  		if !isJSONColumnType(r.DataType) {
  			val = unwrapMongoExtJSON(val)
  		}
  ```
- **Giải thích:** Nếu cột đích ở Master DB là `JSONB` hoặc `JSON`, ta không bóc trần scalar wrapper mà giữ nguyên cấu trúc BSON wrapper từ Shadow chuyển sang. Đối với các cột kiểu scalar (TEXT, TIMESTAMPTZ, BIGINT...), hàm vẫn gọi `unwrapMongoExtJSON` để map đúng kiểu dữ liệu.

---

### File 5: `centralized-data-service/test/internal/service/schema_adapter_coerce_test.go`
- **Vị trí:** Dòng 195–209
- **Cập nhật Unit Test:**
  Assert rằng `parsed["_id"]` là map `{"$oid": "507f1f77bcf86cd799439011"}` và `parsed["ts"]` là map `{"$date": "2026-04-14T10:00:00Z"}` thay vì chuỗi bị bóc trần.
