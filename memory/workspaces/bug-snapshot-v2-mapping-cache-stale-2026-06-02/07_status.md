# 07_status.md

| Field | Value |
|---|---|
| Phase | READY_FOR_MUSCLE (REVISED — Simplicity First) |
| Patch size | 2 file worker, ~20 LOC |
| Service touched | 1 (`centralized-data-service` only) |
| CMS/NATS touched | NO |
| Risk | LOW (snapshot path isolated, realtime CDC không động) |
| Rollback | git revert, an toàn |
| Blocker | None |

## Approach
Snapshot bypass cache, query `mapping_rule_v2` trực tiếp → always fresh, không phụ thuộc cache invalidation.

## Next
Muscle apply patch theo `03_implementation.md §1-3` + verify `06_validation.md`.
