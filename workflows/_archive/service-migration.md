---
description: Chuẩn hóa quy trình refactor 1 service sang CQRS/DDD/Event-Driven
---

# Service Migration Workflow

> Dùng khi refactor 1 service. Workflow này đảm bảo migration nhất quán cho ~60 services.

## Khi nào dùng
Trigger: `/service-migration`
- GĐ3: Refactor services sang CQRS/DDD
- Khi cần migrate 1 service từ pattern cũ sang target architecture

## Input Requirements
- **Service name**: Tên service cần migrate
- **Service type**: Node.js (Moleculer) / Go (Fiber/Echo)
- **Phase target**: CQRS / Event-Driven / Both
- **Priority group**: Financial Core / Banking / Gateway / Business / Utilities

## Workflow Steps

### 1. Audit Current State

// turbo
```bash
# Xem cấu trúc hiện tại
find /Users/trainguyen/Documents/work/<service>/src -type f -name "*.ts" | head -30

# Xem dependencies
cat /Users/trainguyen/Documents/work/<service>/package.json | grep -A 20 "dependencies"

# Tìm RPC calls đi ra
grep -rn "ctx.call\|broker.call" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts" | head -20

# Tìm RPC calls đi vào (services khác gọi service này)
grep -rn "<service-name>\." /Users/trainguyen/Documents/work/*/src/ --include="*.ts" -l | head -20
```

Output: Document dependencies (in/out), DB models, current patterns.

### 2. Plan Migration

Dựa trên target architecture:

```
service-name/
├── domain/
│   ├── entities/           # Aggregate roots
│   ├── events/             # Domain events
│   ├── repositories/       # Interfaces
│   └── services/           # Domain services
├── application/
│   ├── commands/           # Write operations
│   ├── queries/            # Read operations
│   ├── handlers/           # Command/query handlers
│   └── events/             # Event handlers
├── infrastructure/
│   ├── repositories/       # Implementations
│   ├── adapters/           # External services
│   └── persistence/        # Database
└── interface/
    ├── dto/                # Data transfer objects
    └── providers/          # Service providers (Moleculer actions)
```

Checklist:
- [ ] List entities cần extract từ models
- [ ] List commands (write operations)
- [ ] List queries (read operations)
- [ ] Domain events cần define
- [ ] Compensating actions cho saga

### 3. Implement (Incremental)

**KHÔNG refactor big-bang**. Incremental steps:

1. **Tạo folder structure** → domain/, application/, infrastructure/
2. **Extract entities** → domain/entities/ (không thay đổi behavior)
3. **Extract repository interfaces** → domain/repositories/
4. **Create command handlers** → application/commands/ 
5. **Create query handlers** → application/queries/
6. **Wire up** → interface/providers/ (Moleculer actions gọi handlers)
7. **Add domain events** → emit events qua NATS

### 4. Integration Test

// turbo
```bash
# Chạy tests service vừa migrate
cd /Users/trainguyen/Documents/work/<service>
npm test

# Chạy tests services phụ thuộc
cd /Users/trainguyen/Documents/work/<dependent-service>
npm test
```

Verify:
- [ ] API contract không thay đổi (backward compatible)
- [ ] Tất cả Moleculer actions vẫn hoạt động
- [ ] DB queries vẫn đúng
- [ ] Events emit đúng format

### 5. K8s Deploy Check

// turbo
```bash
cat /Users/trainguyen/Documents/work/workload-sre-*/deployments/<service>.yaml
```

Verify deployment config vẫn compatible.

### 6. Canary Deploy & Monitor

- Deploy lên staging trước
- Monitor 24h
- Nếu OK → merge & deploy production

## Migration Scorecard

Review mỗi service đã migrate:

| Criteria | Before | After |
|----------|--------|-------|
| Has domain layer? | ❌ | ✅ |
| CQRS separation? | ❌ | ✅ |
| Domain events? | ❌ | ✅ |
| Saga compensation? | ❌ | ✅ |
| Repository pattern? | ❌ | ✅ |
| Test coverage % | XX% | XX% |

## Definition of Done
- [ ] Migration Scorecard đạt 5/5 criteria ✅
- [ ] Test coverage không giảm so với trước migration
- [ ] `/qa-agent` và `/security-agent` đã pass cho service vừa migrate
- [ ] `05_progress.md` đã được cập nhật với service name và kết quả migration
