# KẾ HOẠCH TRIỂN KHAI CHI TIẾT: KHẮC PHỤC TRIỆT ĐỂ LỖI KAFKA CONNECT TOPIC.PREFIX INVALID FORMAT

---

## 1. Mục tiêu
Giải quyết dứt điểm lỗi Kafka Connect HTTP 400 `The 'topic.prefix' value is invalid: cdc.goopaylocal. traitestmongodevct has invalid format (only the underscore, hyphen, dot and alphanumeric characters are allowed)` khi tạo hoặc cập nhật connector trên CMS UI.

## 2. Các bước triển khai chi tiết

### Bước 1: Frontend CMS Web (`cdc-cms-web/src/pages/SourceConnectors.tsx`)
1. Thêm hàm chuẩn hóa format:
   ```ts
   export function sanitizeTopicPrefix(prefix: string, name?: string): string {
     const cleanPrefix = (prefix || '').replace(/\s+/g, '').replace(/\.+$/, '');
     if (!name) {
       return cleanPrefix.replace(/\.+/g, '.').replace(/^\.+|\.+$/g, '');
     }
     const cleanName = (name || '').replace(/\s+/g, '').replace(/^\.+/, '');
     return `${cleanPrefix}.${cleanName}`.replace(/\.+/g, '.').replace(/^\.+|\.+$/g, '');
   }
   ```
2. Thay thế logic gán `topicPrefix` trong `useEffect` (dòng 480-494) bằng `sanitizeTopicPrefix`.
3. Thay thế logic trong `buildConnectorConfig` (dòng 212) bằng `sanitizeTopicPrefix`.
4. Bọc `sanitizeTopicPrefix` trong `parseConnectionSeed` (dòng 394, 422, 435) để tự động làm sạch cả dữ liệu cũ khi load vào form.

### Bước 2: Backend CMS Service (`cdc-cms-service`)
1. Trong `internal/api/source/system_connectors_handler.go`:
   Thêm helper `sanitizeKafkaTopicPrefix`:
   ```go
   func sanitizeKafkaTopicPrefix(val string) string {
       cleaned := strings.Join(strings.Fields(val), "")
       for strings.Contains(cleaned, "..") {
           cleaned = strings.ReplaceAll(cleaned, "..", ".")
       }
       return strings.Trim(cleaned, ".")
   }
   ```
   Gọi `sanitizeKafkaTopicPrefix` trên `req.Config["topic.prefix"]` và `req.Config["topic"]` trước khi chuyển sang Command Bus.
2. Trong `internal/app/commands/source/debezium_connector.go`:
   Thêm defensive sanitize trong `UpdateSystemConnectorConfigHandler` và `CreateSystemConnectorHandler`.

### Bước 3: Kiểm thử
1. `cdc-cms-web`: Chạy `npm run build` verify 100% pass TypeScript.
2. `cdc-cms-service`: Chạy `go test ./...` và `go build ./cmd/server`.
