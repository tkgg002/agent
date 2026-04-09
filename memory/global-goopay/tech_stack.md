# Tech Stack & Guidelines — GooPay

> **Last Updated**: 2026-02-10
> **Maintained by**: /context-manager

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Language** | Node.js/TypeScript (55 svc), Go 1.22-1.24 (4 svc) | Hybrid stack |
| **Framework** | Moleculer (Node), Fiber/Echo (Go) | NATS transporter |
| **Database** | MongoDB (Mongoose), MySQL (một số svc), GORM (Go) | |
| **Messaging** | NATS (RPC + pub/sub), Socket.io, Redis pub/sub | Future: NATS JetStream |
| **Cache** | Redis | Distributed locks, session, cache |
| **Infra** | Kubernetes, Docker | workload-sre-*/deployments/ |
| **Frontend** | React (6 apps) | admin-portal-web, merchant-portal, payment-web |
| **DI** | Inversify (wallet-trans, bank-transfer), Wire (Go) | |
| **Monitoring** | SignOZ (hiện tại) | Future: Jaeger + ELK + Prometheus/Grafana |

## Coding Guidelines

### Node.js/Moleculer Services
- Service structure: `src/` → `services/`, `models/`, `mixins/`, `utils/`
- Config: Environment variables, moleculer.config.ts
- Patterns: Một số có CQRS/DDD (wallet-trans), nhiều svc chưa có pattern rõ ràng

### Go Services
- Structure: `cmd/`, `internal/`, `pkg/`
- DB: GORM (MySQL), go-mongo-driver (MongoDB)
- HTTP: Fiber hoặc Echo framework

### Target Architecture (Post-Refactor)
```
service-name/
├── domain/         # Aggregate roots, events, repository interfaces
├── application/    # Commands, queries, handlers, event handlers
├── infrastructure/ # Repository implementations, adapters, persistence
└── interface/      # DTOs, service providers
```

### Patterns đang sử dụng / sẽ áp dụng
- **CQRS**: wallet-trans-service (đã có) → mở rộng ra 6+ services
- **Event Sourcing**: Một phần (wallet-trans)
- **Saga Pattern**: wallet-trans (đã có) → Saga Coordinator trung tâm
- **DDD**: disbursement-service (Go, đã có) → chuẩn hóa

## Architecture

### Communication Patterns
```
Gateway → Business Service → Financial Service → Banking Connector
         (3-4 hops qua NATS/RPC)
```

### Critical Paths
1. **Payment Flow**: gateway → payment-bill → payment → wallet → bank-transfer → connector
2. **Disbursement**: gateway → disbursement(Go) → wallet → bank-handler(Go) → connector
3. **Wallet Transfer**: gateway → wallet-trans → wallet (internal)
