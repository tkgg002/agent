# Implementation Design: TriggeredBy Refactor

## Proposed Shape
- Add `internal/activity` as the core package for worker audit taxonomy.
- Define constants:
  - TriggeredBy: `scheduler`, `nats-command`, `kafka-consumer`, `recon-healer`
  - Operations: `kafka-consume-batch` and any touched operation literals.
- Add a Kafka post-consume action interface/function option in `internal/handler/kafka_consumer.go`.
- Default action is no-op so existing production startup remains compatible.
- Tests should instantiate the consumer with a fake action and verify it is called with batch metadata.

## Non-goals
- No DB migration.
- No config mutation to force a passing run.
- No broad CommandBus rewrite in this phase.
- No sinkworker audit refactor unless required by compile/test fallout.

