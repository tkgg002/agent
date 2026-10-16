# 08_tasks_legacy_topic_routing.md

## Checklist Triển khai (Muscle Implementation Tasks)

- [x] **Task 1: Bổ sung phương thức đọc `cdc_system.sources` trong Repository**
  - File: `internal/repository/source/connection_registry_repo.go`
  - Viết hàm `GetTopicPrefixToConnectionMap(ctx context.Context) (map[string]string, error)`
  - Query: `SELECT connector_name, topic_prefix FROM cdc_system.sources WHERE status != 'deleted' AND topic_prefix IS NOT NULL AND topic_prefix != ''`
  - Đã hoàn tất: Đọc động map `topic_prefix -> connector_name`, lọc bỏ status deleted và prefix rỗng.

- [x] **Task 2: Nạp và tra cứu Longest Prefix Match trong Metadata Registry**
  - File: `internal/service/source/metadata_registry_service.go`
  - Thêm trường `topicPrefixToConnCode map[string]string` vào struct
  - Nạp dữ liệu trong `ReloadAll(ctx)`
  - Thêm method `ResolveConnectionByTopic(topic string) string` (thuật toán Longest Prefix Match)
  - Đã hoàn tất: Nạp từ repo trong `ReloadAll`, tra cứu Longest Prefix Match an toàn dưới RLock.

- [x] **Task 3: Nâng cấp Kafka Consumer nhận diện Connection theo Topic**
  - File: `internal/handler/shadow/kafka_consumer.go`
  - Mở rộng interface `registrySvc` với `ResolveConnectionByTopic(topic string) string`
  - Trong `processMessage`: Gán `sourceConnCode = kc.registrySvc.ResolveConnectionByTopic(msg.Topic)` trước khi fallback
  - Xóa bỏ struct field `TopicPrefixConnectionMap` tạm bợ
  - Đã hoàn tất: Gán `sourceConnCode` từ `ResolveConnectionByTopic`, fallback phân tích segment và `IsGenericConnectionCode`.

- [x] **Task 4: Dọn dẹp cấu hình thừa & server setup**
  - File: `config/config-local.yml` (đã dọn)
  - File: `internal/server/server_setup.go` (xóa truyền `TopicPrefixConnectionMap`)
  - Đã hoàn tất: Xóa bỏ `TopicPrefixConnectionMap` khỏi `KafkaConsumerConfig` trong `server_setup.go`.

- [x] **Task 5: Kiểm thử và Biên dịch (Verification)**
  - Chạy `go build ./cmd/worker` biên dịch thành công 100% (exit code = 0)
  - Viết Unit Test kiểm tra logic `ResolveConnectionByTopic` và `GetTopicPrefixToConnectionMap` trong `internal/service/source/metadata_registry_service_test.go`
  - Verify routing phân lập đúng giữa `traitestctphs` và `traitestmongodevct`

