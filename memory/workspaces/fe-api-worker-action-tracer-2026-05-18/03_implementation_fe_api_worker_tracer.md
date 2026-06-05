# Implementation Design: FE API Worker Action Tracer

## Initial Design
- Prefer a lightweight `trace_id`/`action`/`origin` metadata envelope in payloads that already cross boundaries.
- FE should send or generate a trace id per action.
- API should log the trace id and include it in any worker command payload.
- Worker should log the same trace id when a command handler starts and completes.

## Expected Flow
- `Sync Fields to Shadow`: FE action -> API endpoint -> NATS `cdc.cmd.shadow.bind` or equivalent worker command -> worker creates/updates fields in shadow binding path.
- `Snapshot Now`: FE action -> API endpoint -> NATS Debezium signal or batch/snapshot command -> worker triggers snapshot/sync into shadow.

## Non-goals
- No broad UI rewrite.
- No DB migration unless source inspection proves it is required.
- No direct DB writes from FE/API to simulate worker completion.

