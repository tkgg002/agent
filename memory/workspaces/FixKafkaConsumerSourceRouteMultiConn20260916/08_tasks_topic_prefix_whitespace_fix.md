# DANH SÁCH TASKS: KHẮC PHỤC LỖI KAFKA CONNECT TOPIC.PREFIX INVALID FORMAT (WHITESPACE SANITIZATION)

---

## Task 1: Frontend CMS Web (`cdc-cms-web/src/pages/SourceConnectors.tsx`)
- [ ] 1. Khai báo helper function `sanitizeTopicPrefix(prefix: string, name?: string): string`:
  - Loại bỏ toàn bộ whitespace `\s+`.
  - Cắt bỏ dấu chấm thừa ở đầu và cuối.
  - Gom các dấu chấm lặp lại liên tiếp `\.+` thành 1 dấu chấm đơn `.`.
  - Nối `prefix` và `name` (nếu có).
- [ ] 2. Cập nhật `useEffect` tự động set `topicPrefix` (dòng 480-494):
  - Áp dụng `sanitizeTopicPrefix` cho cả 4 loại database (sftp, mongodb, mysql, postgresql).
- [ ] 3. Cập nhật `buildConnectorConfig` (dòng 212):
  - Dùng `sanitizeTopicPrefix` thay vì chỉ gọi `values.topicPrefix.trim()`.
- [ ] 4. Cập nhật `parseConnectionSeed` (dòng 394, 422, 435):
  - Bọc `sanitizeTopicPrefix` quanh giá trị topicPrefix cũ để tự động dọn sạch khoảng trắng khi mở modal Edit.
- [ ] 5. Chạy kiểm tra biên dịch TypeScript:
  - `npm run build` trong `cdc-cms-web`.

---

## Task 2: Backend CMS Service (`cdc-cms-service`)
- [ ] 1. Sửa `internal/api/source/system_connectors_handler.go`:
  - Thêm helper `sanitizeKafkaTopicPrefix(val string) string`.
  - Áp dụng trong `Create` (dòng 150) và `UpdateConfig` (dòng 212) cho `req.Config["topic.prefix"]` và `req.Config["topic"]`.
- [ ] 2. Sửa `internal/app/commands/source/debezium_connector.go`:
  - Thêm defensive sanitize cho `merged["topic.prefix"]` trong `UpdateSystemConnectorConfigHandler` và `cmd.Config["topic.prefix"]` trong `CreateSystemConnectorHandler`.
- [ ] 3. Bổ sung Unit Tests kiểm chứng:
  - Viết test case xác thực chuỗi dính khoảng trắng `"cdc.goopaylocal. traitestmongodevct"` được tự động làm sạch thành `"cdc.goopaylocal.traitestmongodevct"`.
- [ ] 4. Chạy kiểm tra biên dịch Go:
  - `go test ./...` và `go build ./cmd/server` trong `cdc-cms-service`.

---

## Task 3: Nghiệm thu & Báo cáo
- [ ] 1. Kiểm định chéo bằng Adversarial QC.
- [ ] 2. Cập nhật tiến độ `05_progress.md`.
- [ ] 3. Xuất bản báo cáo hoàn thành.
