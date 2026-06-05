# 08_tasks_audit — Task Breakdown

## Audit phase (this session — DONE)
| ID | Task | Status |
|---|---|---|
| A-1 | Read GEMINI.md + lessons.md + project_context.md | DONE |
| A-2 | Init workspace + 00_context, 01_requirements | DONE |
| A-3 | Trace FE `/shadow` → registry_handler_register → RegisterRegistryCommand → NATS dispatch | DONE |
| A-4 | Identify Bug 1 root cause | DONE — `mapping_rule_v2_repo.go:54-61` + `command_handler.go:649` |
| A-5 | Identify Bug 2 root cause | DONE — `command_handler.go:586-602` + `command_handler.go:163-172` |
| A-6 | Cross-check `_source_ts` ở các path khác | DONE — sinkworker/upsert+schema_manager + service/master_ddl+transmuter đều dùng đúng |
| A-7 | Write 02_plan, 03_implementation, 04_decisions, 09_tasks_solution | DONE |
| A-8 | Build verify 2 service (compile sạch hiện tại) | DONE (xem `06_validation.md`) |
| A-9 | Write `report_audit_shadow_create_bugs_2026-05-27.md` | DONE |

## Fix phase (PENDING — chờ user approve)
| ID | Task | Owner | Pre-cond |
|---|---|---|---|
| F-1 | Apply SOL-1 patch (Bug 1 swap caller) | Muscle | User OK |
| F-2 | Apply SOL-2 patch A/B/C (Bug 2 add cols + index + UNIQUE) | Muscle | User OK |
| F-3 | `go build ./...` + `go vet ./...` ở centralized-data-service | Muscle | F-1, F-2 |
| F-4 | Unit test scope: nếu có test cho `HandleCreateDefaultColumns`, mở rộng assert 3 cột mới | Muscle | F-1, F-2 |
| F-5 | E2E: tạo shadow mới qua FE → psql verify 11 cột system + 0 business cross-leak | Muscle | F-3 |
| F-6 | /security-agent gate (§8) | Muscle | F-5 |

## Migration phase (FUTURE — out-of-scope hôm nay)
| ID | Task |
|---|---|
| MIGR-1 | Audit toàn bộ shadow tables hiện hữu trong PG 5436 (shadow_*): SELECT từ information_schema.columns, list shadows thiếu `_source_ts` / `_gpay_source_id` / `_gpay_deleted` |
| MIGR-2 | Dry-run plan ALTER TABLE ADD COLUMN cho từng shadow — order: critical (đang nhận traffic) trước, dormant sau |
| MIGR-3 | Backup snapshot trước migration |
| MIGR-4 | Apply ALTER + verify OCC guard không broken |

## Gap follow-up (xem `10_gap_analysis.md`)
| ID | Gap |
|---|---|
| GAP-1 | 2 path DDL builder song song (handler + sinkworker) dễ drift |
| GAP-2 | `HandleScanFields` line 1389 vẫn dùng `GetActiveRulesBySourceTable` — same cross-bleed risk |
| GAP-3 | Không có integration test cho `HandleCreateDefaultColumns` |
| GAP-4 | Không có lint rule chặn ai đó thêm DDL Shadow mà thiếu `_source_ts` |
