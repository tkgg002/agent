# Danh Sách Task Chi Tiết - Fix PostgreSQL Scan Fields ID

- [ ] **Task 1 (Discovery Utils)**: Loại bỏ logic `if strings.EqualFold(name, pkColumn) { continue }` trong `inferPGCols`, `inferMySQLCols`, `inferMongoCols` (`centralized-data-service/internal/handler/source/discovery_utils.go`).
- [ ] **Task 2 (Discover Handler Utils)**: Hỗ trợ bóc tách Debezium `after` payload trong `processDiscoveryRows` (`centralized-data-service/internal/handler/source/discover_handler_utils.go`).
- [ ] **Task 3 (Verification)**: Chạy test biên dịch `go build ./...` trong `centralized-data-service`.
