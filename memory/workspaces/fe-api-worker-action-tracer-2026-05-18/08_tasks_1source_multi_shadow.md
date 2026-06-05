# Tasks — 1 source → multi shadow target

**Phase**: 1source_multi_shadow
**Date**: 2026-05-19

| # | Task | Owner | Status |
|---|---|---|---|
| 1 | Identify constraint blocker (V1 cdc_table_registry UNIQUE) | Muscle | done |
| 2 | Confirm V2 model đã 1→N (source_object_registry + shadow_binding) | Muscle | done |
| 3 | Create migration 053 (DROP + ADD 3-col UNIQUE) | Muscle | done |
| 4 | Audit V1 INSERT path (register_registry.go) | Muscle | done |
| 5 | Audit V2 sync path (source_object_v2_sync.go) | Muscle | done |
| 6 | Audit bootstrap mirror (registry_mirror.go) | Muscle | done |
| 7 | Audit worker reads (metadata_registry_service.go sourceCache) | Muscle | done |
| 8 | Workspace Full Doc Set (01/02/08/09) | Muscle | done |
| 9 | APPEND 05_progress.md | Muscle | done |
| 10 | APPEND global lesson | Muscle | done |
| 11 | report_1source_multi_shadow.md | Muscle | done |
| 12 | User: apply migration 053 | User | pending |
| 13 | User: verify retry register success | User | pending |
