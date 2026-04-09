# 02_plan.md - Implementation Strategy (Master Plan)

## Phase 0: Preparation (Current)

### 0.1 Audit Database Schema (Idempotency)
- **Target**: Mongoose Models (`models/*.model.ts`) in Payment, Wallet, and Disbursement files.
- **Risk**: `payment.model.ts` has `index({ merchantTransId: 1 })` but **MISSING** `unique: true`.
- **Action**:
    - [ ] Create script to scan all 60 services for `Schema` definitions.
    - [ ] Identify fields `requestId`, `merchantTransId`, `referenceCode`.
    - [ ] Report models missing `{ unique: true }` in their index definition.
    - [ ] Generate migration script (MongoDB/Mongoose `.createIndex`).

### 0.2 Audit Redis Key TTL (Deadlock Prevention)
- **Target**: Service logic (`logics/**/*.ts`) and `moleculer.config.ts`.
- **Action**:
    - [ ] Grep `broker.cacher.lock` and `redlock` usage.
    - [ ] Verify if `ttl` argument is passed and < 30s.
    - [ ] Output list of "Forever Locks" (missing TTL).

### 0.3 Upgrade Shared Library (`package` repo)
- **Target**: `@goopay/goopay-library`.
- **Action**:
    - [ ] Create `GracefulShutdownPlugin` in `helpers/moleculer.helper.js`.
    - [ ] Logic: Enforce `tracking: { enabled: true, shutdownTimeout: 30000 }` when mixin is used.
    - [ ] Implement `preStop` hook support for K8s signals.


## Phase 1: Infrastructure Stabilization
- [ ] 1.1 Config Graceful Shutdown (Node.js/Moleculer).
- [ ] 1.2 Config Graceful Shutdown (Go Services).
- [ ] 1.3 Update K8s Deployment YAML (`preStop`, `readinessProbe`).

## Phase 2: Resilience
- [ ] 2.1 Config Retry Policies (Service-to-Service).
- [ ] 2.2 Implement Transaction Sweeper (Cronjob).
- [ ] 2.3 Build Admin Tooling (CMS Module).

## Phase 3: Core Re-architecture
- [ ] 3.1 NATS JetStream Migration.
- [ ] 3.2 Saga Pattern Implementation.

## Phase 4: UX & Ops
- [ ] 4.1 Frontend Timeout Handling.
- [ ] 4.2 Async Response API (HTTP 202).
- [ ] 4.3 Distributed Tracing (Jaeger).
