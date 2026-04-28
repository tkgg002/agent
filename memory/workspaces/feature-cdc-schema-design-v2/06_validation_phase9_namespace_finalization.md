# Validation — Phase 9 Namespace Finalization

## Audit

- Search runtime cho `cdc_internal.` trong `internal/` và `cmd/` trả về rỗng ở lớp code thực thi.
- Những chỗ còn `cdc_internal` chủ yếu nằm ở:
  - migration history
  - comment lịch sử

## Formatting

- Đã chạy `gofmt` cho toàn bộ file Go vừa chỉnh.

## Test results

Đã chạy pass:

```bash
go test ./internal/service ./internal/handler ./internal/server
go test ./internal/sinkworker
go test ./internal/service ./internal/server ./internal/repository ./internal/model
```

Kết quả quan sát:

- `internal/service`: pass
- `internal/handler`: pass
- `internal/server`: compile path pass
- `internal/sinkworker`: pass
- `internal/repository`: compile path pass
- `internal/model`: compile path pass

## Compliance note

- `public` schema mặc định của PostgreSQL không được xem là object application cần drop cưỡng bức.
- End-state được đảm bảo ở mức application:
  - không còn system tables của app nằm ở `public`
  - không còn runtime dependency vào `cdc_internal`
