# Kế Hoạch Triển Khai - Fix PostgreSQL Scan Fields ID

1. Sửa `discovery_utils.go` (loại bỏ `continue` khi gặp `pkColumn` để quét cả cột `id`).
2. Sửa `discover_handler_utils.go` (bóc tách key `after` Debezium CDC payload).
3. Biên dịch test `go build ./...` trong `centralized-data-service`.
