# Context: Debug Snapshot Signal

- **Feature**: CDC MongoDB Snapshot Signal integration.
- **Problem**: Previously `snapshot now` signal worked and data was synced. Now it does not work.
- **Current State**: 
  - `centralized-data-service` is running locally (PID 87730).
  - Kafka is running (likely on Kubernetes or reachable network).
  - Suspected missing messages or another worker consuming the messages (152 messages).
  - Shadow table is empty.
- **Goal**: Find root cause and fix without cheating configurations.


# Fix CDC Worker Materialization Failures

This plan addresses the root causes of the silent snapshot signal failure. The worker is successfully picking up the messages from Kafka, but multiple severe pipeline bugs cause the worker to crash silently and hang indefinitely, leading to empty shadow tables and stalled message consumption.

## User Review Required

> [!WARNING]
> This plan modifies core `KafkaConsumer` logic.
> - **Go Select Behavior:** I'll rewrite the `KafkaConsumer.Start()` loop so that `FetchMessage` no longer blocks the `default` branch of the `select` statement. This is a critical Go concurrency bug that is starving the internal batch flush tickers.
> - **DLQ Constraints:** The plan involves sanitizing Avro binary payloads to strip out null bytes (`\x00`) before they hit the Postgres `jsonb` column in `failed_sync_logs`. Otherwise, DLQ writes will always crash on Avro data.

## Open Questions

> [!IMPORTANT]
> The worker is trying to reach the schema registry at `http://schema-registry-v1.data-hub:8081` (from `config-local.yml`). Since this DNS doesn't resolve on your local Mac environment, the HTTP request hangs forever because `http.DefaultClient` has no timeout.
> I will add a 5-second HTTP timeout in the code to prevent the worker from hanging.
> However, to successfully test this end-to-end, the worker **must** be able to reach the schema registry. How do you want to handle the local configuration?
> 1. Pass an environment variable: `KAFKA_SCHEMAREGISTRYURL=http://10.200.186.203:8081 make run`
> 2. Let the Kubernetes worker handle the actual insert, and we only fix the Go codebase?

## Proposed Changes

---

### Centralized Data Service

#### [MODIFY] [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/kafka_consumer.go)
1. **Fix Concurrency Block:** Move `FetchMessage(ctx)` out of the `select { default: ... }` block. We will wrap the `FetchMessage(ctx)` with a local context timeout (`100ms`), or run it asynchronously. Using a context timeout of 100ms ensures the `select` loop evaluates `flushTicker.C` without starving, and avoids context cancellation overhead since it's just polling.
2. **Fix HTTP Hang:** In `getAvroCodec`, replace the usage of `http.Get(url)` (which has NO timeout) with a custom `http.Client` that has a `5 * time.Second` timeout.
3. **Fix Postgres Null Byte Constraint:** In `extractDLQMetadata` and `sanitizeDLQRawJSON`, clean `\x00` bytes from Avro payloads by replacing them using `bytes.ReplaceAll(raw, []byte{0}, []byte{})`. Postgres text/jsonb types strictly forbid the `\u0000` sequence which occurs if we directly cast Avro magic bytes to a string before `json.Marshal`.

## Verification Plan

### Automated Tests
- Run `make test` inside the centralized-data-service to ensure no regressions.
- Verify `KafkaConsumer` can correctly flush batches every 5 seconds.

### Manual Verification
- Start the worker using `make run`. 
- Monitor the terminal output; we should see `fetch schema: context deadline exceeded` instead of the worker hanging indefinitely if the DNS is unreachable.
- When `KAFKA_SCHEMAREGISTRYURL` is correctly provided, confirm the worker inserts the 1 pending message (Offset 152) into the shadow table `sd_export_jobs_dev`.
