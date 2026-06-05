# Context: Fixing CDC Activity Log Metrics

## Background
- **Topic**: Data pipeline metrics accuracy and materialization.
- **Problem**: The CDC pipeline's activity log currently reports incorrect values for `RowsAffected` because it is based on the volume of messages consumed from Kafka rather than the actual rows materialized in the destination database.
- **Impact**: Inaccurate operational monitoring and false representation of synchronization success counts.

## Objectives
1. Implement synchronous database writes for CDC events via `BatchBuffer` or `EventHandler` to capture exact rows successfully materialized.
2. Synchronize offsets to ensure At-Least-Once delivery and reliable error propagation.
3. Update `ActivityLog` tracking to log exact DB rows affected.
4. Keep per-message logs at `DEBUG` level and batch logs at `INFO` level.
5. Avoid duplicate logging in `failed_sync_logs` by streamlining DLQ propagation.
