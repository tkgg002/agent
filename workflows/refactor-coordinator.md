---
description: Điều phối refactor toàn dự án — Track phases, dependencies, checkpoints
---

# Refactor Coordinator Workflow

> Workflow cấp cao nhất. Điều phối 5 giai đoạn refactor cho ~60 services.
> Chỉ Brain sử dụng. Muscle không gọi trực tiếp.

## Khi nào dùng
Trigger: `/refactor-coordinator`
- Khi bắt đầu session làm việc → check overall progress
- Khi hoàn thành 1 task → update progress & check next steps
- Khi cần quyết định chuyển giai đoạn

## Workflow Steps

### 1. Load Current State

// turbo
```bash
cat agent/memory/active_plans.md
```

Xác định:
- Đang ở Phase nào?
- Phase hiện tại hoàn thành bao nhiêu %?
- Có blockers không?

### 2. Phase Transition Checklist

Trước khi chuyển sang giai đoạn mới, PHẢI verify:


### 2.1 Phase Transition Checklist (EXAMPLE)

#### GĐ0 → GĐ1 Gate
- [ ] Database audit hoàn thành (UNIQUE INDEX tất cả tables)
- [ ] Redis key patterns chuẩn hóa (có TTL)
- [ ] Environment variables inventory done
- [ ] Shared library version mới published

#### GĐ1 → GĐ2 Gate
- [ ] 60/60 services có graceful shutdown
- [ ] K8s configs updated (terminationGracePeriodSeconds, preStop, readinessProbe)
- [ ] Zero 502 errors khi restart bất kỳ service nào
- [ ] Pilot group verified (notification, promotion)

#### GĐ2 → GĐ3 Gate
- [ ] Retry policies enabled cho tất cả Moleculer services
- [ ] Transaction sweeper chạy và detect stuck transactions
- [ ] Admin portal có module quản lý stuck transactions
- [ ] Go HTTP clients có retry (go-retryablehttp)

#### GĐ3 → GĐ4 Gate
- [ ] NATS JetStream streams created
- [ ] Critical paths migrated: payment→disbursement (async)
- [ ] Saga Coordinator operational
- [ ] 6+ services refactored sang CQRS/DDD

### 3. Service Group Execution Order

Trong mỗi Phase, thực hiện theo thứ tự risk thấp → cao:

```
1. Utilities (⚪ Lowest)     — Pilot, test quy trình
2. Business (🔵 Low)         — Feature, ít critical
3. Gateways (🟢 Medium)      — Entry points
4. Banking Connectors (🟡 High) — External deps
5. Financial Core (🔴 Critical) — Cuối cùng, sau khi tự tin
```

### 4. Dependency Tracking

Trước khi refactor service X, kiểm tra:
```
Xong chưa?  [Service X phụ thuộc vào]
    ↳ Nếu chưa → Refactor dependency trước
    ↳ Nếu rồi → Proceed

Ai phụ thuộc vào Service X?
    ↳ Nếu thay đổi contract → Update tất cả dependents
    ↳ Nếu backward compatible → OK proceed
```

### 5. Update Progress

Sau mỗi task, update `agent/memory/active_plans.md`:
- Mark task hoàn thành
- Update phase progress %
- Ghi nhận blockers/risks mới

## Decision Matrix

| Tình huống | Quyết định |
|-----------|-----------|
| Task xong, phase chưa xong | Tiếp tục task tiếp trong phase |
| Phase hoàn thành, gate pass | Chuyển sang phase mới |
| Phase hoàn thành, gate FAIL | Fix blockers trước khi chuyển |
| Phát hiện risk mới | Ghi vào memory, đánh giá priority |
| ADR cần quyết định | Pause, present options cho User |

## Definition of Done
- [ ] Phase hiện tại đã pass tất cả Quality Gates
- [ ] `05_progress.md` đã được cập nhật với % hoàn thành của phase
- [ ] Mọi ADR trong phase đã được ghi vào `04_decisions.md`
- [ ] Brain thông báo rõ ràng cho User trước khi chuyển sang phase tiếp theo
