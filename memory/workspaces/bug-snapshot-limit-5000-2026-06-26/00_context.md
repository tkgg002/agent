# Context: Bug Snapshot Limit 5000 Records
## Task Description
User reported a bug where snapshot.v2 only processed 5000 records.
Log message:
`[2026-06-26T08:27:01.822Z] INFO TraiNguyens-MacBook-Pro.local/CDC-WORKER: snapshot.v2 completed {"batches_total":2,"component":"snapshot_runner","duration_sec":3.16122125,"op":"run_snapshot","phase":"completed","progress_id":40,"rows_total":5000,"source_object_id":76,"target_table":"payment_bills","trace_id":"fe-snapshot-a4ca9c64-a932-4043-b39c-bd40df06fb93"}`
Objective: Investigate why the snapshot only ran 5000 records and fix the issue.

## Environment & Architecture
- Codebases: `centralized-data-service`, `cdc-cms-service`, etc.
- Current active document: `internal/server/server_jobs.go`
- Target table: `payment_bills`
- Source Object ID: 76
- Component: `snapshot_runner`
