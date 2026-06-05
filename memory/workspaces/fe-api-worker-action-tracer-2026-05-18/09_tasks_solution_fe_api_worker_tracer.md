# Technical Solution: FE API Worker Action Tracer

## Candidate Fix Shape
- Add `trace_id` to FE action request body or headers.
- API normalizes trace metadata and includes it in NATS worker command payload.
- Worker command payload structs accept `trace_id`/`action`/`origin` without breaking older clients.
- Worker logs start/end with trace fields.

## Definition of Done
- Both user actions have confirmed FE -> API -> worker call chains.
- Missing trigger links are fixed in code.
- Trace id is visible at each boundary in source/log path.
- Tests/builds pass or failures are documented as unrelated/pre-existing with evidence.

