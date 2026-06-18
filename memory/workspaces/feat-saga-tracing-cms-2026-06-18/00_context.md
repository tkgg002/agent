# 00 — Context & Scope: Saga Pattern & OTel Tracing

**Workspace**: `feat-saga-tracing-cms-2026-06-18`  
**Service**: `cdc-cms-service`  
**Tạo bởi**: Brain (Claude Sonnet)  
**Ngày**: 2026-06-18  

---

## Mục tiêu

Implement **Saga Pattern** (distributed transaction integrity) và **OTel Distributed Tracing** (request lifecycle visibility) cho toàn bộ service `cdc-cms-service`.

## Phạm vi

- **Service**: `cdc-cms-service` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`)
- **Không ảnh hưởng**: `centralized-data-service`, `cdc-auth-service`, `cdc-cms-web`

## Kiến trúc hiện tại

- Hexagonal Architecture (Port & Adapter)
- CQRS với CommandBus (Sync/Async)
- NATS messaging cho async dispatch
- OTel SDK đã có sẵn trong `pkgs/observability/otel.go` nhưng chưa dùng trong command layer

## Số lượng commands cần audit

| Nhóm | Files | Sync | Async |
|------|-------|------|-------|
| source | 11 | 7 | 4 |
| shadow | 1 | 1 | 0 |
| master | 8 | 7 | 1 |
| governance | 5 | 4 | 1 |
| recon | 4 | 1 | 7 |
| scheduler | 9 | 7 | 2 |
| system | 2 | 2 | 0 |
| **Total** | **50** | **29** | **15** |

## Tech Stack liên quan

- Go 1.26.1
- `go.opentelemetry.io/otel` v1.43.0 (đã có)
- `go.opentelemetry.io/otel/trace` v1.43.0 (đã có)
- `go.uber.org/zap` v1.27.1
- `github.com/gofiber/fiber/v2` v2.52.12
- `github.com/nats-io/nats.go` v1.50.0

## Nguyên tắc thiết kế

- **Simplicity First**: Saga chỉ áp dụng cho luồng có multi-system side-effects thực sự
- **Minimal Impact**: Không thay đổi API contracts, không break existing tests
- **Pattern consistency**: Follow pattern hiện tại (SyncHandler interface, ports abstraction)
