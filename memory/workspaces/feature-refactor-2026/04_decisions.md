# 04_decisions.md - Decision Log

## ADR-001: NATS JetStream for Queue
- **Decision**: Use NATS JetStream instead of RabbitMQ/Kafka.
- **Reasoning**: Already in tech stack, supports lightweight streams, sufficient for current scale.

## ADR-002: Saga Pattern over 2PC
- **Decision**: Use Orchestration-based Saga.
- **Reasoning**: Distributed transactions (2PC) are too brittle. Saga allows eventual consistency via compensating transactions.

## ADR-003: Timeout Groups
- **Decision**: Group A (User) = 30s. Group B (Connectors) = 60s.
- **Reasoning**: Banking partners (BIDV, Napas) typically have long response times.
