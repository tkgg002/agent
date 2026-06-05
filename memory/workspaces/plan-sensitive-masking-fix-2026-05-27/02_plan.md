# 02_plan — Roadmap Sensitive Masking Compliance Fix

## Sequencing (dependency order)

```
Phase P0 — Schema & Engine core (16h)
  ├── M-1 Migration: mask_strategy + mask_options + audit tables
  ├── M-2 Strategy interface + 4 impl (None/Drop/Hmac/Partial)
  ├── M-3 HMAC key vault integration
  ├── M-4 MaskingService refactor (loại bỏ "***" literal khỏi DB path)
  └── M-5 Unit test masking_service_test.go ≥ 90% coverage
         ↓
Phase P1 — Worker integration (10h)
  ├── M-6 dynamic_mapper.go apply strategy
  ├── M-7 batch_buffer.go + recon_heal.go + kafka_consumer.go align
  ├── M-8 Audit log writer (sample rate)
  └── M-9 Integration test E2E
         ↓
Phase P2 — Admin API + UI + Backfill (14h)
  ├── M-10 cdc-cms-service API CRUD mask-config + audit endpoint
  ├── M-11 cdc-cms-web tab Sensitive Masking + preview
  ├── M-12 Backfill script (re-mask shadow data)
  └── M-13 Compliance evidence doc
```

**Tổng effort**: ~~40h~~ **50h** Muscle work (revised round 1: +10h cho gap missing C1–C7, H1–H9).

## Phase P0 — 16h

| Task | Effort | Files thay đổi | Detail |
|---|---|---|---|
| M-1 Schema migration | 3h | `data-hub/cdc-cms-service/migrations/schema/core/068_add_mask_strategy.sql` (NEW); kèm `068_add_mask_strategy.down.sql` (NEW, ADR-015); cập nhật `data-hub/centralized-data-service/internal/model/mapping_rule.go` (thêm 3 field GORM) | §M-1 |
| M-1b Audit log partition | 1h | Phần trong `068_*.sql` — PARTITION BY RANGE (masked_at) + retention 13 tháng (H4) | §M-1 |
| M-2 Strategy interface + 4 impl | 4h | `data-hub/centralized-data-service/internal/service/masking/strategy.go` (NEW); `none.go`, `drop.go`, `hmac.go`, `partial.go` (NEW); kèm `normalizeValue()` helper (H2) | §M-2 |
| M-2b Rule cache + invalidation | 2h | `internal/service/masking/rule_cache.go` (NEW) — sync.Map + pub-sub invalidation hook (H1) | §M-2b |
| M-3 HMAC key vault | 2h | `data-hub/centralized-data-service/internal/config/config.go` + `pkgs/vault/key_loader.go` (NEW) | §M-3 |
| M-4 MaskingService refactor (recursive walker) | 5h | `data-hub/centralized-data-service/internal/service/masking_service.go` (REFACTOR: bỏ `***` literal khỏi DB path; **DUAL method** giữ legacy + `*Ctx`; recursive walker ADR-009 cho nested object) | §M-4 |
| M-4b Schema inspector preview refactor | 1h | `internal/service/schema_inspector.go` + signature `MaskFieldSample()` — return `FieldSample{IsMasked,Strategy,Type,Length}` thay literal `"***"` (R-22) | §M-4b |
| M-5 Unit test | 2h | `internal/service/masking_service_test.go` + `masking/*_test.go` (NEW) — bao gồm test nested object + empty string + type normalization | §M-5 |
| M-5b Benchmark baseline | 1h | `BenchmarkMaskTableData_Before_After` (M-r5) — measure p99 trước/sau | §M-5b |

## Phase P1 — 13h (revised từ 10h: +3h cho 3 caller missing & dual-method migration)

| Task | Effort | Files thay đổi |
|---|---|---|
| M-6 Mapper integration | 3h | `data-hub/centralized-data-service/internal/service/dynamic_mapper.go` (3 call-site: line 67, 114, 127; xoá helper `maskRawData` line 123) |
| M-7 Buffer/Recon/Consumer align | 3h | `internal/handler/batch_buffer.go:412`, `internal/handler/recon_handler.go:701`, `internal/handler/kafka_consumer.go:1390`, `internal/service/recon_heal.go:807` |
| **M-7b Missing caller refactor (C3)** | 3h | `internal/handler/dlq_handler.go:335`, `internal/service/dlq_worker.go:359` — chuyển sang `*Ctx` variant. Tham chiếu `13_caller_inventory.md` |
| M-8 Audit log writer | 2h | `internal/service/masking/audit_writer.go` (NEW) — batch insert + sample rate; insert vào partition theo `masked_at` (H4) |
| M-9 E2E integration test | 2h | `internal/service/masking_e2e_test.go` (NEW) — testcontainers postgres; bao gồm test nested + multi-field + audit verify |

## Phase P2 — 18h (revised từ 14h: +4h UI bump + OpenAPI contract)

| Task | Effort | Files thay đổi |
|---|---|---|
| M-10 API CRUD | 4h | `data-hub/cdc-cms-service/internal/api/mask_config_handler.go` (NEW); query + DTO; router wire admin group |
| **M-10b OpenAPI contract (M-r3)** | 1h | `data-hub/cdc-cms-service/api/openapi/mask-config.yaml` (NEW) — FE/BE đồng bộ contract |
| M-11 UI tab | 10h (bumped, M-r4) | `data-hub/cdc-cms-web/src/pages/MappingRuleEditPage.tsx`; `src/components/masking/StrategySelector.tsx` + `MaskPreview.tsx` + `MaskAuditList.tsx` (NEW); hook `useMaskConfig.ts` |
| **M-12 Backfill script (revised, ADR-013)** | 3h | `data-hub/centralized-data-service/scripts/backfill_mask.go` (NEW) — trigger re-snapshot từ source MongoDB; fallback set null cho row source-expired + ghi `mask_backfill_loss_log` |
| M-13 Compliance doc | 2h | `data-hub/centralized-data-service/docs/compliance/sensitive-masking-vn-law.md` (NEW); kèm section erasure rights (ADR-012) |

## Phase Dependency map

```
P0 (engine) ─► P1 (integration) ─► P2 (UI + backfill)
                                      └── Backfill chỉ chạy sau khi P0+P1 deploy.
```

UI có thể parallel với P1 nếu BE API contract chốt sớm.

## ADRs (xem `04_decisions.md`) — Round 1: 15 ADR
- ADR-001: HMAC-SHA256 vs SHA256 thuần — chọn HMAC.
- ADR-002: Key storage — Vault vs K8s Secret + env.
- ADR-003: Default strategy cho field hiện có (sensitive_fields list).
- ADR-004: DDM (Postgres VIEW) — in scope hay defer?
- ADR-005: Backfill `"***"` cũ → `NULL` hay HMAC? (SUPERSEDED bởi ADR-013)
- ADR-006: Audit log sample rate — 100% hay 1%?
- ADR-007: Strategy enum location — DB enum type vs Go enum + check constraint?
- ADR-008: Loại bỏ `"***"` khỏi log path? — giữ `text_sanitizer.go`.
- **ADR-009 (NEW)**: Path-based vs flat strategy lookup — chọn recursive walker (R-05).
- **ADR-010 (NEW)**: Deploy ordering — DDL → Worker → Seed → CMS/UI → Backfill (R-06).
- **ADR-011 (NEW)**: HMAC empty string → return nil (R-09).
- **ADR-012 (NEW)**: Right-to-erasure qua CDC tombstone từ source (R-11).
- **ADR-013 (NEW)**: Backfill policy — re-snapshot từ source MongoDB (R-13).
- **ADR-014 (NEW)**: Multi-strategy per field — defer Phase 2 (R-14).
- **ADR-015 (NEW)**: Migration phải có DOWN file (R-12).

## Verification per phase
- Mỗi gap có acceptance check (unit test / integration test / smoke).
- Composite không track score (đây là compliance fix, không phải QA gap fix).
- Service work verify (build + vet + test) trước khi báo done.

## Brain workflow
1. Brain (file này) tạo doc set đầy đủ.
2. User review + approve specific phase (verb: `execute p0|p1|p2`, `revise`, `defer`).
3. Muscle execute phase được approved → APPEND `05_progress.md`.
4. /security-agent scan sau mỗi phase.
