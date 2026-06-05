# Report — Plan vá Sensitive Masking theo Luật BVDLCN VN

**Date**: 2026-05-27
**Workspace**: `agent/memory/workspaces/plan-sensitive-masking-fix-2026-05-27/`
**Status**: PLAN READY — chờ User verb

---

## TL;DR

- **Vấn đề**: Hệ thống CDC hiện tại hardcode `"***"` literal ghi vào DB shadow → vi phạm:
  - **Luật 91/2025/QH15** Điều 13 (tính chính xác + quyền chỉnh sửa).
  - **Nghị định 356/2025/NĐ-CP** (biện pháp kỹ thuật phù hợp).
  - **VBHN 25/VBHN-NHNN** (audit + thanh tra).
- **Chế tài rủi ro**: 3 tỷ VND hoặc 5% doanh thu năm liền kề.
- **Giải pháp**: Strategy engine 4 mode (NONE/DROP/HASH_HMAC/PARTIAL) per-field, có audit log + admin UI.
- **Effort**: 40h (P0 16h + P1 10h + P2 14h).
- **18 file workspace** đã tạo theo §7 Full Doc Set, tuân thủ §11 APPEND-only + §12 Brain Code Prohibition.

---

## Evidence vấn đề (từ Explore subagent thorough scan)

| # | Vấn đề | File:line |
|---|---|---|
| M1 | `"***"` hardcode trong DB write-path | `centralized-data-service/internal/service/masking_service.go:91,133,152-153` |
| M2 | `cdc_mapping_rules` thiếu `mask_strategy/mask_options` | `cdc-cms-service/migrations/schema/core/001_init_schema.sql:112-149` |
| M3 | Masking apply ở write-path (phá huỷ gốc) | `dynamic_mapper.go:67,114` |
| M4 | Không có API CRUD masking config | (không tìm thấy trong `cdc-cms-service/internal/`) |
| M5 | Test rời rạc assert `"***"` | `batch_buffer_test.go:14-29`, `recon_handler_test.go:12-27`, `text_sanitizer_test.go:9-44` |
| M6 | Không có design doc masking | (không tìm thấy trong `docs/`) |

---

## Plan 3-phase

| Phase | Effort | Task | Deliverable |
|---|---|---|---|
| **P0 — Engine** | 16h | M-1 migration ENUM + audit tables + seed; M-2 Strategy interface + 4 impl; M-3 HMAC KeyProvider env-based; M-4 MaskingService refactor (bỏ `"***"` khỏi DB path); M-5 unit test ≥ 90% | Engine sẵn sàng, không còn `"***"` literal |
| **P1 — Integration** | 10h | M-6 DynamicMapper; M-7 BatchBuffer/Recon/Consumer align; M-8 AuditWriter sample 1%; M-9 E2E testcontainers | Worker E2E dùng strategy engine |
| **P2 — Admin + UI + Backfill** | 14h | M-10 API CRUD `/mapping-rules/:id/mask-config`; M-11 cdc-cms-web tab Sensitive Masking + StrategySelector + MaskPreview; M-12 backfill script; M-13 `docs/compliance/sensitive-masking-vn-law.md` | Self-service admin + legacy cleanup |

## Strategy → Use case mapping (ADR-003)

| Strategy | Field | Lý do |
|---|---|---|
| **NONE** | trans_id, created_at | Không nhạy cảm |
| **DROP** | password, OTP, PIN, CVV | Không có giá trị đối soát, theo NĐ 356 "loại bỏ" |
| **HASH_HMAC** | CCCD, card_number, account_number | Luật 91/2025 "De-identification", giữ tính đối soát |
| **PARTIAL** | phone, email | VBHN 25 "hiển thị một phần cho audit" |

---

## ADRs

| ADR | Topic | Decision |
|---|---|---|
| 001 | HMAC vs SHA256 thuần | **HMAC-SHA256** (chống rainbow table) |
| 002 | Key storage | **K8s Secret + env**, Vault defer |
| 003 | Default strategy seed | Classify theo tên field |
| 004 | DDM (Postgres VIEW) | **Defer** (Phase 3 nếu auditor yêu cầu) |
| 005 | Backfill plaintext gốc đã mất | Re-mask theo strategy mới với `null` nếu DROP/HMAC |
| 006 | Audit sample rate | **1%** default, configurable |
| 007 | Strategy enum | **DB ENUM TYPE** + Go constant |
| 008 | `text_sanitizer.go` log path | **Giữ nguyên** (log không phải DB persistence) |

---

## Files in workspace (18 file vật lý)

```
plan-sensitive-masking-fix-2026-05-27/
├── 00_context.md                        — Bối cảnh pháp lý + evidence
├── 01_requirements.md                   — FR-1..7 + NFR-1..4 + DoD-1..7
├── 02_plan.md                           — 3-phase sequencing
├── 03_implementation_phase_p0.md        — Engine chi tiết (M-1..M-5)
├── 03_implementation_phase_p1.md        — Worker integration (M-6..M-9)
├── 03_implementation_phase_p2.md        — Admin + UI + Backfill (M-10..M-13)
├── 04_decisions.md                      — 8 ADR
├── 05_progress.md                       — APPEND-only audit log
├── 06_validation.md                     — Acceptance + verify command
├── 07_status_report.md                  — Status tổng + Verb chờ
├── 08_tasks_phase_p0.md                 — Checklist Muscle P0
├── 08_tasks_phase_p1.md                 — Checklist Muscle P1
├── 08_tasks_phase_p2.md                 — Checklist Muscle P2
├── 09_tasks_solution_phase_p0.md        — Rationale + alternative rejected
├── 09_tasks_solution_phase_p1.md
├── 09_tasks_solution_phase_p2.md
├── 10_gap_analysis.md                   — Gap → Fix → Verify map
└── report_sensitive_masking_fix_2026-05-27.md  ← file này
```

---

## File sẽ thay đổi khi Muscle execute (tổng hợp từ implementation_phase_*)

### centralized-data-service (Phase P0 + P1)
- NEW: `internal/service/masking/strategy.go`, `none.go`, `drop.go`, `hmac.go`, `partial.go`, `audit_writer.go`
- NEW: `pkgs/vault/key_loader.go`
- NEW: `internal/service/masking_service_test.go`, `masking_e2e_test.go` (build tag `integration`)
- NEW: `scripts/backfill_mask.go` (build tag `backfill`)
- NEW: `deployments/k8s/cdc-masking-keys-secret.yaml`
- SỬA: `internal/service/masking_service.go` (refactor — bỏ `"***"` khỏi DB path)
- SỬA: `internal/service/dynamic_mapper.go` (lines 67, 114; xóa helper cũ 123-127)
- SỬA: `internal/handler/batch_buffer.go` (lines 374-383)
- SỬA: `internal/service/recon_heal.go` (lines 771-776 — cập nhật assert)
- SỬA: `internal/handler/kafka_consumer.go` (lines 1124-1125 — cập nhật assert)
- SỬA: `internal/model/mapping_rule.go` (thêm 3 field GORM)
- SỬA: `internal/config/config.go` (thêm MaskingConfig)
- **KHÔNG SỬA**: `internal/service/text_sanitizer.go` (ADR-008)

### cdc-cms-service (Phase P2)
- NEW: `migrations/schema/core/015_mask_strategy.sql`
- NEW: `internal/api/mask_config_handler.go`, `internal/api/dto/mask_config_dto.go`
- NEW: `internal/app/commands/update_mask_config.go`
- NEW: `internal/app/queries/get_mask_config.go`, `list_mask_audit.go`
- NEW: `internal/api/mask_config_handler_test.go`
- SỬA: `internal/router/router.go` (thêm 3 route admin)
- SỬA: `internal/server/server.go` (DI wire)

### cdc-cms-web (Phase P2)
- NEW: `src/types/masking.ts`
- NEW: `src/hooks/useMaskConfig.ts`
- NEW: `src/components/masking/StrategySelector.tsx`, `MaskPreview.tsx`, `MaskingTab.tsx`
- SỬA: `src/pages/MappingRuleEditPage.tsx` (thêm Tab Sensitive Masking)

### docs (Phase P2)
- NEW: `docs/compliance/sensitive-masking-vn-law.md`

---

## Governance compliance

| Quy tắc | Status |
|---|---|
| §1 Brain plan-only | ✓ Không touch code source |
| §7 Full Doc Set 18 file | ✓ |
| §11 Memory APPEND-only | ✓ `05_progress.md` chỉ APPEND |
| §12 Brain Code Prohibition | ✓ Code demo trong MD, không sửa .go/.ts/.sql |
| §14 Pre-flight 18 file vật lý | ✓ |

---

## Verb chờ User

| Verb | Hành động |
|---|---|
| `execute p0` | Muscle bắt đầu Engine (M-1..M-5, 16h) |
| `execute p1` | Muscle Worker Integration (M-6..M-9, 10h, yêu cầu P0 done) |
| `execute p2` | Muscle Admin + UI + Backfill (M-10..M-13, 14h) |
| `revise` | User chỉ định item cụ thể cần plan lại |
| `defer` | Tạm hoãn, lưu trạng thái |

---

## Skill đã sử dụng

- Memory governance (workspace init, APPEND progress, Full Doc Set §7).
- Brain plan-only delegation (§1, §12 Brain Code Prohibition).
- Explore subagent thorough scan (evidence file:line cho 3 service).
- ADR documentation pattern (8 ADR).
- Compliance mapping (Article ↔ Control).
- Strategy pattern + Registry design (Go interface).
- Per-phase task checklist + solution rationale + gap analysis (separation theo §7).
- Verification mapping (verify command định lượng cho mỗi task).
- Brain pre-flight Governance check §14.
