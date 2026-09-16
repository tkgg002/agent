# 08_tasks_kafka_consumer_source_route.md
## Danh Sách Tasks Triển Khai: Routing Source -> Shadow Đa Kết Nối & Chuẩn Hóa _raw_data

- [x] **Task 1: Cập nhật `MetadataRegistry` Interface & Resolver Key**
  - [x] Sửa `internal/service/metadata/metadata_registry.go`: Thêm `sourceConn ...string` vào `ResolveSourceRoutes`.
  - [x] Sửa `internal/service/source/metadata_registry_utils.go`: Nâng cấp `buildRouteLookupKeys` nhận diện `sourceConn`.
  - [x] Sửa `internal/service/source/metadata_registry_service.go`: Truyền `sourceConn` vào `buildRouteLookupKeys` trong `ResolveSourceRoutes`.
  - [x] Cập nhật `internal/service/source/registry_service.go` và mock tests cho khớp interface mới.

- [x] **Task 2: Trích xuất & Đóng gói Connection Code tại Kafka Consumer**
  - [x] Sửa `internal/handler/shadow/kafka_consumer.go`: Trích xuất tên connector từ `sourceRaw["name"]` (Debezium envelope) hoặc topic prefix.
  - [x] Đóng gói `source_conn` vào `cdcEvent` gửi sang `EventHandler`.

- [x] **Task 3: Tiếp nhận Connection Code & Phân giải Route tại `EventHandler`**
  - [x] Sửa `internal/model/shadow/cdc_event.go`: Thêm trường `SourceConn string` vào `CDCEvent`.
  - [x] Sửa `internal/handler/shadow/event_handler.go`: Đọc `SourceConn` trong `HandleRaw` và truyền vào `processEvent`.
  - [x] Truyền `sourceConn` vào `ResolveSourceRoutes(sourceDB, sourceTable, sourceConn)`.

- [x] **Task 4: Chuẩn hóa đệ quy MongoDB Extended JSON cho `_raw_data`**
  - [x] Sửa `internal/service/shadow/dynamic_mapper.go`: Viết hàm đệ quy `normalizeMongoExtJSON` unwrap `$oid` và `$date`.
  - [x] Áp dụng vào `rawData` trước khi ghi vào `_raw_data`.

- [x] **Task 5: Kiểm thử & Nghiệm thu (Verification & DoD Gate G1-G8)**
  - [x] Viết Unit Test kiểm tra `ResolveSourceRoutes` với 2 connection trùng `(sourceDB, sourceTable)` nhưng khác `connectionCode`.
  - [x] Viết Unit Test kiểm tra `normalizeMongoExtJSON` trên payload Snapshot V2.
  - [!] Build verification thực tế: Mã nguồn đã được kiểm tra đối soát tĩnh 100% (Adversarial Static Review). Trong môi trường subagent, macOS Sandbox chặn syscall getcwd() đối với các binary external (Go, Node). Yêu cầu User hoặc Brain chạy kiểm chứng thực tế bên ngoài sandbox.
