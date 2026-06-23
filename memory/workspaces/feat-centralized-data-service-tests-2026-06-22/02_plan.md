# Plan: Bổ sung unit tests toàn diện cho centralized-data-service

## 1. Các bước thực hiện (Phases)

### Phase 1: Bổ sung unit tests cho toàn bộ tầng Pkgs (Hạ tầng & Tiện ích)
- **Task 1.1**: Viết unit test cho `pkgs/crypto/aes.go` (`aes_test.go`).
- **Task 1.2**: Viết unit test cho `pkgs/natsconn/action_trace.go` và `nats_client.go` (`action_trace_test.go` và `nats_client_test.go`).
- **Task 1.3**: Viết unit test cho `pkgs/kafka/avro.go` (`avro_test.go`).
- **Task 1.4**: Viết unit test cho `pkgs/mongodb/client.go` (`client_test.go`).
- **Task 1.5**: Viết unit test cho `pkgs/rediscache/redis_client.go` (`redis_client_test.go`).
- **Task 1.6**: Viết unit test cho `pkgs/metrics/` và `pkgs/observability/`.

### Phase 2: Bổ sung unit tests cho Metadata & Shadow Services
- **Task 2.1**: Viết unit test cho `internal/service/metadata/` (`metadata_helpers_test.go` và `metadata_mapping_test.go`).
- **Task 2.2**: Viết unit test cho `internal/service/shadow/` (`shadow_services_test.go` test child_explode, dynamic_mapper, enrichment_service, schema_adapter, type_resolver).

### Phase 3: Bổ sung unit tests cho Orchestration & Database Repositories
- **Task 3.1**: Thiết lập SQLMock và viết unit test cho các repositories trong `internal/repository/...` (`repository_mock_test.go`).
- **Task 3.2**: Viết unit test cho `internal/service/orchestration/` (`provisioning_orchestrator_test.go`).

### Phase 4: Bổ sung unit tests cho Base Handlers & Tối ưu hóa
- **Task 4.1**: Viết unit test cho `internal/handler/base/` (`base_handler_test.go`).
- **Task 4.2**: Chạy kiểm thử đo coverage và tối ưu hóa thời gian chạy.

## 2. Tiêu chí Đạt (Definition of Done)
- 100% test cases mới pass và không làm ảnh hưởng các test cũ.
- Tăng độ phủ test coverage thực tế của các package mục tiêu.
- Không phát sinh goroutine leak (goleak check pass).
