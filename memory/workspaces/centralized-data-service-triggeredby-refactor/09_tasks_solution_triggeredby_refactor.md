# Technical Solution: TriggeredBy Refactor

## Solution Details
- Replace direct string literals at touched call sites with typed constants from `internal/activity`.
- Use a small `PostConsumeAction` callback in Kafka consumer internals rather than coupling Kafka consumption to a concrete downstream service.
- Pass batch context to the post-consume action: topic/table identifiers, row count, and TriggeredBy.
- Ensure action failures are logged and surfaced through existing consumer error handling without acknowledging success silently.

## Expected Debug Flow
1. Kafka batch is decoded.
2. Batch write/process completes.
3. Activity log records `TriggeredBy=kafka-consumer`, `Operation=kafka-consume-batch`.
4. Post-consume action executes immediately.
5. Log includes action name, batch size, and success/failure.

