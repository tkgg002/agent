# Validation — Phase 36 Schema Tail Cleanup

## Commands

```bash
gofmt -w \
  /Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/event_bridge.go \
  /Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/service/transform_service.go \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/repository/registry_repo.go

cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go test ./internal/handler ./internal/service ./internal/server ./internal/repository

cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service
go test ./internal/repository ./internal/api ./internal/router ./internal/server
```

## Kết quả

- `gofmt`: pass
- Worker verify:
  - `go test ./internal/handler ./internal/service ./internal/server ./internal/repository`: pass
- CMS verify:
  - `go test ./internal/repository ./internal/api ./internal/router ./internal/server`: pass

## Ghi chú

- Lần chạy đầu trong sandbox bị chặn bởi Go build cache.
- Đã rerun ngoài sandbox hợp lệ và test pass.
