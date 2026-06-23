# Tasks: Bổ sung unit tests toàn diện cho centralized-data-service

## Task: Phase 1 - Pkgs Package Tests
- **Mô tả**: Viết unit test cho các package hạ tầng & tiện ích (`pkgs/crypto`, `pkgs/natsconn`, `pkgs/kafka`, `pkgs/mongodb`, `pkgs/rediscache`, `pkgs/metrics`, `pkgs/observability`)
- **Trạng thái**: [x] DONE
- **Definition of Done**:
  - [x] Tạo file `pkgs/crypto/aes_test.go` và viết test cho `EncryptAES` / `DecryptAES`.
  - [x] Tạo file `pkgs/natsconn/action_trace_test.go` và viết test cho `ActionTrace` (sử dụng nats mock).
  - [x] Tạo file `pkgs/natsconn/nats_client_test.go` và test publish/subscribe.
  - [x] Tạo file `pkgs/kafka/avro_test.go` và test avro payload decoder.
  - [x] Tạo file `pkgs/mongodb/client_test.go` và test client connection.
  - [x] Tạo file `pkgs/rediscache/redis_client_test.go` và test redis caching.
  - [x] Tạo test cho `pkgs/metrics` và `pkgs/observability`.
  - [x] Chạy và verify tất cả test trong `pkgs/` pass 100%.


## Task: Phase 2 - Metadata & Shadow Services Tests
- **Mô tả**: Viết unit test cho `internal/service/metadata` và `internal/service/shadow`
- **Trạng thái**: [ ] TODO
- **Definition of Done**:
  - [ ] Tạo file `test/internal/service/metadata_helpers_test.go` test các logic validate config trong `internal/service/metadata/helpers.go`.
  - [ ] Tạo file `test/internal/service/metadata_mapping_test.go` test logic mapping trong `internal/service/metadata/mapping_utils.go`.
  - [ ] Tạo file `test/internal/service/shadow_services_test.go` test logic explode, dynamic_mapper, enrichment_service, schema_adapter, type_resolver.
  - [ ] Chạy và verify tất cả service tests pass 100%.

## Task: Phase 3 - Orchestration & DB Repository Tests
- **Mô tả**: Thiết lập GORM DB Mocking và test orchestration service
- **Trạng thái**: [ ] TODO
- **Definition of Done**:
  - [ ] Thiết lập SQLMock và tạo file `test/internal/repository/repository_mock_test.go` test cho các repositories trong `internal/repository/...`.
  - [ ] Tạo file `test/internal/service/provisioning_orchestrator_test.go` test logic orchestration.
  - [ ] Chạy và verify repo & orchestration tests pass 100%.

## Task: Phase 4 - Base Handlers Tests & Optimization
- **Mô tả**: Viết test cho base handler và tối ưu hóa toàn bộ suite
- **Trạng thái**: [ ] TODO
- **Definition of Done**:
  - [ ] Tạo file `internal/handler/base/base_handler_test.go` test JSON response, error wrapper, và step completed.
  - [ ] Chạy `go test -cover ./...` pass 100% dưới 5 giây.
