# Architectural Decision Records — GooPay

> **Last Updated**: 2026-02-10
> **Maintained by**: /context-manager

## [ADR-001] Hybrid Node.js + Go Stack
- **Status**: Accepted (hiện tại)
- **Context**: Cần IO-heavy (API, orchestration) + compute-heavy (banking, reconcile)
- **Decision**: Node.js/Moleculer cho đa số services, Go cho banking/disbursement/reconcile
- **Consequences**: Cần maintain 2 tech stacks, nhưng tận dụng strengths của mỗi ngôn ngữ

## [ADR-002] NATS làm Message Broker
- **Status**: Accepted
- **Context**: Cần RPC + pub/sub cho 60 services
- **Decision**: NATS (hiện tại basic), migrate sang JetStream (GĐ3)
- **Consequences**: JetStream cho durable consumers, at-least-once delivery, stream replay

## [ADR-003] Temporal.io vs Custom Workflow Engine
- **Status**: Proposed (chưa quyết định)
- **Context**: Cần workflow orchestration cho 10 transaction flows
- **Option A**: Custom Config-Driven Engine (YAML-based TDL) — Linh hoạt, tốn 18 tháng
- **Option B**: Temporal.io — Production-ready 1-2 tháng, workflow-as-code
- **Recommendation**: POC Temporal với booking-ticket-flow trước khi quyết định
- **References**: `config-driven-transaction-plan.md`, `temporal-migration-plan.md`

## [ADR-004] Saga Pattern cho Distributed Transactions
- **Status**: Accepted
- **Context**: Giao dịch tài chính cần rollback khi fail (ví dụ: trừ tiền xong nhưng bank transfer fail)
- **Decision**: Saga Coordinator trung tâm orchestrate compensating actions
- **Consequences**: Mọi step phải có compensating action. State được persist trước khi execute.
- **References**: `saga-coordinator-design.md`

## [ADR-005] CQRS/DDD Standard cho Services
- **Status**: Accepted
- **Context**: Services hiện tại không có pattern rõ ràng, khó maintain
- **Decision**: Chuẩn hóa theo CQRS/DDD template (học từ wallet-trans-service)
- **Target services**: payment, merchant, customer, profile, promotion, rule
- **Consequences**: Tốn effort refactor nhưng maintainability tăng đáng kể
