# Tech Stack & Guidelines

> **Last Updated**: [DATE]
> **Maintained by**: /context-manager

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Language** | [e.g. TypeScript, Go, Python] | |
| **Framework** | [e.g. Express, Fiber, FastAPI] | |
| **Database** | [e.g. PostgreSQL, MongoDB, MySQL] | |
| **Messaging** | [e.g. NATS, Kafka, RabbitMQ] | |
| **Cache** | [e.g. Redis, Memcached] | |
| **Infra** | [e.g. Kubernetes, Docker, ECS] | |
| **Frontend** | [e.g. React, Vue, None] | |
| **Monitoring** | [e.g. Prometheus/Grafana, Datadog, SignOZ] | |
| **AI (Brain)** | `gemini-3-pro-high` | Planning & Reasoning |
| **AI (Muscle)**| `gemini-3-flash`    | Execution & CLI Tools |

---

## AI Models & Roles Mapping

| Role | Strategy | Primary Model | Fallback / Alternative |
|------|----------|---------------|------------------------|
| **Brain** (Chairman) | Prioritize reasoning & quality | `gemini-3-pro-high` | `claude-sonnet-4-5-thinking` |
| **Muscle** (Engineer) | Prioritize speed & cost efficiency | `gemini-3-flash` | `gemini-2.5-flash` |

> [!TIP]
> Cấu hình model có thể được ghi đè linh hoạt thông qua `agent/models.env` cho từng phiên làm việc hoặc task đặc thù.

---

## Coding Guidelines

### General
- [Project-specific code standards]
- [Naming conventions]
- [Error handling patterns]

### Service / Module Structure
```
[module-name]/
├── [layer-1]/     # e.g. domain, core, models
├── [layer-2]/     # e.g. application, services
├── [layer-3]/     # e.g. infrastructure, repositories
└── [layer-4]/     # e.g. interface, controllers, routes
```

### Patterns đang sử dụng / sẽ áp dụng
<!-- Đánh dấu trạng thái: ✅ Đã có | 🚧 Đang triển khai | 📋 Planned -->
- **[Pattern A]** (e.g. CQRS): [Scope — services nào áp dụng]
- **[Pattern B]** (e.g. Saga): [Scope]
- **[Pattern C]** (e.g. DDD): [Scope]

---

## Architecture

### Communication Patterns
```
[Mô tả luồng giao tiếp giữa các thành phần]
e.g. Gateway → Service A → Service B → External API
```

### Critical Paths
<!-- Các luồng quan trọng nhất — không được fail -->
1. **[Flow 1]**: [ServiceA → ServiceB → ServiceC]
2. **[Flow 2]**: [...]
