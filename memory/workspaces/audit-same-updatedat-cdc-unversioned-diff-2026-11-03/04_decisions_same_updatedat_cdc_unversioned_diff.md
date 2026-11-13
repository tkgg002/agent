# 04. Architecture Decision Records (ADRs) — `audit-same-updatedat-cdc-unversioned-diff-2026-11-03`

## ADR-01: Sub-Second MongoDB Oplog Tie-Breaker (`source.ord`) & OCC Same-Timestamp Allowance (`Code Demo 1A + 1B`)
- **Date:** `2026-11-03`
- **Status:** `APPROVED & IMPLEMENTED`
- **Scope:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` only (Zero changes to `core-trans-proxy-service` per User direction).
- **Laya DG3 Resolution (`06_laya_summary_dg3_20261103T132747.json`):**
  - `A-FIX-1`: Resolved `FLAG` via `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/utils.go:72-97` and unit test receipt `EV-20261103T132348-cf1d` (`PASS`).
  - `A-FIX-2`: Resolved `FLAG` via `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go:768-789` and unit test receipts `EV-20261103T132409-6241` & `EV-20261103T132536-a432` (`PASS`).
- [2026-11-03T13:52:00+07:00] [Agent:laya_gate] DG3 A-FIX-1 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/utils.go:72-117 verified by EV-20261103T132348-cf1d.
- [2026-11-03T13:52:04+07:00] [Agent:laya_gate] DG3 A-FIX-2 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go:768-789 verified by EV-20261103T132409-6241.
- [2026-11-03T13:59:06+07:00] [Agent:laya_gate] DG3 A-FIX-2 FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go:768-789 verified by EV-20261103T135657-b281.
