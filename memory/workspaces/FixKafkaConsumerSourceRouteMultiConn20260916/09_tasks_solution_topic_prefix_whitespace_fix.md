# HỒ SƠ GIẢI PHÁP KỸ THUẬT: KHẮC PHỤC TRIỆT ĐỂ LỖI KAFKA CONNECT HTTP 400 TOPIC.PREFIX INVALID FORMAT

---

## 1. Bối cảnh & Hiện tượng Lỗi Runtime
Khi User thực hiện thao tác cập nhật (Update Config) hoặc tạo mới Connector `traitestmongodevct` trên CMS UI, hệ thống trả về lỗi:
```json
{
    "detail": "kafka connect HTTP 400: {\"error_code\":400,\"message\":\"Connector configuration is invalid and contains the following 1 error(s):\\nThe 'topic.prefix' value is invalid: cdc.goopaylocal. traitestmongodevct has invalid format (only the underscore, hyphen, dot and alphanumeric characters are allowed)\\nYou can also find the above list of errors at the endpoint `/connector-plugins/{connectorType}/config/validate`\"}",
    "error": "connector_update_failed"
}
```

---

## 2. Truy vết Mã Nguồn & Phân tích Gốc rễ (Root Cause Analysis)

### 2.1. Tại sao Kafka Connect quăng lỗi HTTP 400?
Kafka Connect REST API (đặc biệt là Debezium MongoDB/MySQL/PostgreSQL connector) thực hiện validate cấu hình trước khi lưu/chạy:
- Quy tắc định dạng của `topic.prefix`: `^[a-zA-Z0-9_.-]+$` (chỉ cho phép chữ cái, chữ số, dấu gạch dưới, gạch ngang và dấu chấm).
- Giá trị gửi sang: `"cdc.goopaylocal. traitestmongodevct"` chứa ký tự khoảng trắng `' '` ngay sau dấu chấm `.`.
- Khoảng trắng vi phạm trực diện format rule của Kafka Connect -> Kafka Connect từ chối với HTTP 400.

### 2.2. Khoảng trắng sinh ra từ đâu?
Có 3 điểm yếu chí mạng trong chuỗi xử lý mã nguồn hiện tại:

1. **Frontend `SourceConnectors.tsx` (dòng 480-494):**
   - Khi tạo connector, `useEffect` sinh `topicPrefix` bằng template string:
     ```tsx
     const name = slugifyForShadow(String(connectorNameValue || 'connector'));
     form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
     ```
   - Hằng số `TOPIC_PREFIX_MONGODB` lấy từ `import.meta.env.VITE_TOPIC_PREFIX_MONGODB`. Nếu biến môi trường hoặc cấu hình có trailing space hoặc dấu chấm dính khoảng trắng (`"cdc.goopaylocal. "`), chuỗi sinh ra bị nhồi khoảng trắng vào giữa.
2. **Frontend `SourceConnectors.tsx` (dòng 212 & 394):**
   - Trong `buildConnectorConfig`:
     ```tsx
     const topicPrefix = values.topicPrefix.trim() || `cdc.${values.database}`;
     ```
     Hàm `.trim()` của JavaScript **chỉ xóa khoảng trắng ở hai đầu chuỗi**, hoàn toàn không xóa khoảng trắng ở giữa (`"a. b".trim()` vẫn là `"a. b"`).
   - Trong `parseConnectionSeed` (dòng 394): Khi mở modal Edit cho connector cũ, `seed.topicPrefix` lấy trực tiếp từ `source.topic_prefix || cfg['topic.prefix']`. Nếu bản ghi cũ đã dính khoảng trắng, nó được nạp nguyên xi vào Form.
   - Form field `topicPrefix` (dòng 1596) là `<Input disabled />`: Người dùng bị khóa cứng, không thể tự sửa khoảng trắng trên UI!
3. **Backend `cdc-cms-service` (`system_connectors_handler.go` & `debezium_connector.go`):**
   - Tầng API Handler nhận `req.Config` và chuyển tiếp thẳng sang Kafka Connect REST API mà không hề có lớp phòng vệ Defensive Sanitization hoặc Regex Validation cho `topic.prefix`.

---

## 3. Thiết Kế Kỹ Thuật Giải Pháp Toàn Diện

### 3.1. Frontend (`cdc-cms-web/src/pages/SourceConnectors.tsx`)

#### A. Viết hàm chuẩn hóa chuyên dụng `sanitizeTopicPrefix`:
```ts
export function sanitizeTopicPrefix(prefix: string, name?: string): string {
  // 1. Loại bỏ 100% khoảng trắng
  const cleanPrefix = (prefix || '').replace(/\s+/g, '').replace(/\.+$/, '');
  if (!name) {
    return cleanPrefix.replace(/\.+/g, '.').replace(/^\.+|\.+$/g, '');
  }
  const cleanName = (name || '').replace(/\s+/g, '').replace(/^\.+/, '');
  // 2. Ghép chuỗi và chuẩn hóa dấu chấm
  return `${cleanPrefix}.${cleanName}`.replace(/\.+/g, '.').replace(/^\.+|\.+$/g, '');
}
```

#### B. Cập nhật `useEffect` tự động gán `topicPrefix` (dòng 480-494):
```ts
  useEffect(() => {
    if (!editorOpen || editorMode !== 'create') return;
    const name = slugifyForShadow(String(connectorNameValue || 'connector'));
    if (dbKind === 'sftp') {
      form.setFieldValue('topicPrefix', sanitizeTopicPrefix(TOPIC_PREFIX_SFTP, name));
    } else if (dbKind === 'mongodb') {
      form.setFieldValue('topicPrefix', sanitizeTopicPrefix(TOPIC_PREFIX_MONGODB, name));
    } else if (dbKind === 'mysql') {
      form.setFieldValue('topicPrefix', sanitizeTopicPrefix(TOPIC_PREFIX_MYSQL, name));
    } else if (dbKind === 'postgresql') {
      form.setFieldValue('topicPrefix', sanitizeTopicPrefix(TOPIC_PREFIX_POSTGRESQL, name));
    }
  }, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
```

#### C. Cập nhật `buildConnectorConfig` (dòng 212):
```ts
function buildConnectorConfig(values: ConnectionFormValues, mode: EditorMode) {
  const rawPrefix = values.topicPrefix || `cdc.${values.database}`;
  const topicPrefix = sanitizeTopicPrefix(rawPrefix);
  let config: Record<string, string> = {};
  // ... các config còn lại giữ nguyên
```

#### D. Cập nhật `parseConnectionSeed` (dòng 394, 422, 435):
Tự động làm sạch `topicPrefix` khi load dữ liệu cũ lên Modal Edit:
```ts
topicPrefix: sanitizeTopicPrefix(source.topic_prefix || cfg['topic.prefix'] || TOPIC_PREFIX_MONGODB),
```

---

### 3.2. Backend (`cdc-cms-service`)

#### A. Bổ sung helper `SanitizeTopicPrefix` trong `system_connectors_handler.go`:
```go
var kafkaTopicPrefixRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func sanitizeKafkaTopicPrefix(val string) string {
	// 1. Xóa toàn bộ ký tự khoảng trắng
	cleaned := strings.Join(strings.Fields(val), "")
	// 2. Gom nhiều dấu chấm liên tiếp thành 1 dấu chấm
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", ".")
	}
	// 3. Cắt dấu chấm ở đầu và cuối
	return strings.Trim(cleaned, ".")
}
```

#### B. Áp dụng Defensive Sanitization trong `Create` và `UpdateConfig`:
Tại `Create` (dòng 150) và `UpdateConfig` (dòng 212):
```go
if tp, ok := req.Config["topic.prefix"]; ok && tp != "" {
    req.Config["topic.prefix"] = sanitizeKafkaTopicPrefix(tp)
}
if top, ok := req.Config["topic"]; ok && top != "" {
    req.Config["topic"] = sanitizeKafkaTopicPrefix(top)
}
```

#### C. Lớp bảo vệ bổ sung tại Command Handler (`debezium_connector.go`):
Trước khi gọi `h.writer.Create` hoặc `h.writer.UpdateConfig`:
Tự động sanitize `merged["topic.prefix"]` để bảo đảm 100% payload sang Kafka Connect không bao giờ chứa khoảng trắng.
