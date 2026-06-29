# Workspace Context: GoTests

## Goal
Bao phủ toàn bộ Go unit test cho hệ thống `centralized-data-service`.

## Scope
- Viết unit test cho các file trong `internal/service/metadata/` và `internal/service/shadow/`.
- Mock DB thông qua `sqlmock` và `miniredis` / `nats-server` nếu có.
- Tránh kết nối infra ngoài.
