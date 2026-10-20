---
description: Kiểm tra infrastructure — K8s YAML, NATS, Redis, Database indexes cho GooPay
---

# Infrastructure Validator Workflow

> Kiểm tra infrastructure layer. Đặc biệt quan trọng cho GĐ0-1 (Rà soát & Hạ tầng).

## Khi nào dùng
Trigger: `/infra-validator`
- GĐ0: Database audit, Redis key review
- GĐ1: K8s config validation
- GĐ3: NATS JetStream setup verification
- Khi deploy service mới

## Workflow Steps

### 1. K8s Deployment Validation

// turbo
```bash
# Kiểm tra deployment config của 1 service
SERVICE_NAME="<service>"
for ENV in testing staging live prelive; do
  echo "=== $ENV ==="
  cat /Users/trainguyen/Documents/work/workload-sre-$ENV/deployments/$SERVICE_NAME.yaml 2>/dev/null | grep -E "terminationGracePeriod|preStop|readinessProbe|livenessProbe|resources" -A 3
done
```

Checklist:
- [ ] `terminationGracePeriodSeconds` ≥ shutdown timeout + 10s
  - Standard services: ≥ 45s
  - Banking connectors: ≥ 75s
- [ ] `preStop` hook: `sleep 10` (để LB cập nhật IP)
- [ ] `readinessProbe` trỏ vào `/health/ready`
- [ ] Resource limits hợp lý (CPU/Memory)

### 2. Graceful Shutdown Check

// turbo
```bash
# Node.js service: kiểm tra tracking config
grep -rn "tracking\|shutdownTimeout\|MOLECULER_SHUTDOWN" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts" --include="*.js"

# Go service: kiểm tra signal handling
grep -rn "signal.Notify\|Shutdown\|SIGTERM" /Users/trainguyen/Documents/work/<service>/ --include="*.go"
```

### 3. Database Index Audit

// turbo
```bash
# Tìm unique index cho idempotency
grep -rn "unique\|UNIQUE\|createIndex" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts" --include="*.go"

# Tìm request_id / reference_code fields
grep -rn "request_id\|reference_code\|requestId\|referenceCode" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts"
```

Checklist:
- [ ] Transaction tables có UNIQUE INDEX trên `request_id`
- [ ] Idempotency keys được validate
- [ ] Migration scripts sẵn sàng cho missing indexes

### 4. Redis Configuration Check

// turbo
```bash
# Tìm Redis usage
grep -rn "redis\|ioredis\|RedisClient" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts"

# Tìm distributed locks
grep -rn "lock\|setNX\|SET.*NX\|redlock" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts"

# Kiểm tra TTL cho locks
grep -rn "expire\|TTL\|ttl\|EX " /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts"
```

Checklist:
- [ ] Tất cả distributed locks có TTL
- [ ] Key naming pattern chuẩn: `<service>:<purpose>:<id>`
- [ ] Không có lock không bao giờ expire (deadlock risk)

### 5. NATS Configuration Check (GĐ3+)

// turbo
```bash
# Tìm NATS config
grep -rn "JetStream\|Stream\|Consumer\|nats" /Users/trainguyen/Documents/work/<service>/src/ --include="*.ts" --include="*.go"
```

Checklist (khi đã migrate):
- [ ] JetStream streams configured (FileStore, Replicas 3)
- [ ] Durable consumers set up
- [ ] Manual ACK enabled (không auto-ack)
- [ ] Dead Letter Queue cho failed messages

### 6. Report

```markdown
## Infrastructure Validation Report

### Service: <name>
### Environment: <testing/staging/live>

| Check | Status | Notes |
|-------|--------|-------|
| K8s Deployment | ✅/❌ | |
| Graceful Shutdown | ✅/❌ | |
| DB Indexes | ✅/❌ | |
| Redis Config | ✅/❌ | |
| NATS Config | ✅/❌/N/A | |

### Issues Found
1. [CRITICAL/HIGH/MEDIUM] <description>
   - File: `<path>`
   - Fix: <recommendation>

### Overall: ✅ PASS / ❌ FAIL
```

## Definition of Done
- [ ] Tất cả 5 categories đã được kiểm tra (K8s / Graceful Shutdown / DB Index / Redis / NATS)
- [ ] Infrastructure Validation Report đã output với Overall verdict
- [ ] Không có CRITICAL issues tồn tại chưa được resolve
- [ ] Kết quả đã được báo cáo về Brain để quyết định có tiếp tục phase tiếp không
