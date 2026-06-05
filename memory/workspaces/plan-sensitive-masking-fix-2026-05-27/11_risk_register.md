# 11_risk_register — Sensitive Masking Compliance Fix

> Created: 2026-06-01 (review round 1 — append-style, không sửa file cũ).
> Owner: Brain (planning) + Muscle (execute) — phối hợp.
> Format: P (probability 1-5) × I (impact 1-5) = Score (1-25). Score ≥ 12 = blocker.

## Risk matrix tổng

| ID | Risk | P | I | Score | Owner | Mitigation tóm tắt |
|---|---|---|---|---|---|---|
| **R-01** | Migration `015` conflict với `067` hiện có | 5 | 5 | **25** | Muscle | Rename → `068_add_mask_strategy.sql` |
| **R-02** | Service path sai (thiếu prefix `data-hub/`) | 5 | 3 | **15** | Muscle | Sửa toàn bộ path trong implementation doc |
| **R-03** | 3 caller bị bỏ sót (dlq_handler, dlq_worker, schema_inspector) → build break | 4 | 5 | **20** | Muscle | Đối chiếu `13_caller_inventory.md` checklist |
| **R-04** | Signature `MaskTableData` thay đổi đột ngột → 22 call-site break | 4 | 5 | **20** | Muscle | Dual-method approach: giữ legacy + thêm `*Ctx` variant |
| **R-05** | Regression nested object (mongo sub-document không được mask) | 4 | 5 | **20** | Muscle | ADR-009 path-based strategy resolution |
| **R-06** | Deploy ordering sai → migration apply trước code → audit log empty | 3 | 5 | **15** | Muscle + Ops | ADR-010 + `12_rollout_runbook.md` |
| **R-07** | Rule lookup hot-path không cache → query DB 10M lần/ngày → perf drop | 4 | 4 | **16** | Muscle | sync.Map cache + pub-sub invalidation |
| **R-08** | HMAC `fmt.Sprintf("%v", any)` không deterministic cho float/BSON | 4 | 4 | **16** | Muscle | `normalizeValue()` per-type |
| **R-09** | Empty string HMAC leak (HMAC("") cluster detect) | 3 | 3 | 9 | Muscle | ADR-011: empty → nil |
| **R-10** | Audit log phình table (15M/tháng, không partition) | 4 | 3 | **12** | Muscle + DBA | PARTITION BY RANGE (masked_at) + retention 13 tháng |
| **R-11** | Right-to-erasure (Điều 16) không có path un-hash | 3 | 5 | **15** | Brain + Legal | ADR-012: tombstone CDC từ source |
| **R-12** | Migration không có DOWN/rollback | 3 | 4 | 12 | Muscle | Convention dự án bắt buộc `*.down.sql` |
| **R-13** | Backfill set null = mất data | 3 | 5 | **15** | Muscle + Ops | ADR-013: re-snapshot từ source MongoDB |
| **R-14** | Multi-strategy field (email cần hash + partial) | 2 | 3 | 6 | Brain | ADR-014: array strategies |
| **R-15** | Mask config version per-row missing → backfill bỏ sót | 3 | 3 | 9 | Muscle | `_mask_version` shadow column + compare |
| **R-16** | NFR-3 (key không leak qua log) không có verify command | 3 | 4 | 12 | Muscle | Add grep check post-test |
| **R-17** | Coverage 90% không enforce hard gate | 3 | 2 | 6 | Muscle | `-coverpkg` + threshold script CI |
| **R-18** | M-11 UI 6h dưới estimate | 3 | 2 | 6 | Muscle FE | Bump → 10h |
| **R-19** | Performance baseline trước/sau missing | 3 | 3 | 9 | Muscle | M-5b benchmark before/after |
| **R-20** | Sealed Secrets vs etcd encryption không specific | 2 | 3 | 6 | Ops | ADR-002 mở rộng: chọn bitnami sealed-secrets |
| **R-21** | Unicode NFC normalization cho tên/địa chỉ | 2 | 3 | 6 | Muscle | `golang.org/x/text/unicode/norm` |
| **R-22** | `MaskFieldSample` (schema inspector preview) vẫn còn `"***"` | 4 | 2 | 8 | Muscle | M-4b refactor preview path |
| **R-23** | Log path có `zap.Any(rawPayload)` trước sanitize? Chưa audit | 3 | 4 | 12 | Muscle | `grep -rn "zap.Any\|zap.Reflect" internal/` audit |
| **R-24** | Apply migration dev→staging→prod thứ tự không nêu | 3 | 3 | 9 | Ops | `12_rollout_runbook.md` |
| **R-25** | Lesson global cụ thể chưa list trong progress | 1 | 1 | 1 | Brain | Update Entry 06 reference L-IDs |
| **R-26** | ADR count inconsistency (7 vs 8 trong 02_plan vs 04_decisions) | 5 | 1 | 5 | Brain | Fix 02_plan list 8 ADR |
| **R-27** | OpenAPI contract cho M-10 missing → FE/BE drift | 3 | 3 | 9 | Brain | Add `api/openapi/mask-config.yaml` |

## Top 10 blocker (Score ≥ 12)

| Rank | ID | Score | Hành động đầu tiên |
|---|---|---|---|
| 1 | R-01 | 25 | Edit `03_implementation_phase_p0.md` đổi `015` → `068` |
| 2 | R-03 | 20 | Tạo `13_caller_inventory.md`, đánh dấu 8 file |
| 3 | R-04 | 20 | Edit P1 implementation: dual-method strategy |
| 4 | R-05 | 20 | Append ADR-009 vào `04_decisions.md` |
| 5 | R-08 | 16 | Edit P0/M-2 thêm `normalizeValue()` helper |
| 6 | R-07 | 16 | Edit P0/M-2b thêm rule cache + invalidation |
| 7 | R-02 | 15 | Sed path `centralized-data-service/` → `data-hub/centralized-data-service/` |
| 8 | R-06 | 15 | Append ADR-010 + `12_rollout_runbook.md` |
| 9 | R-11 | 15 | Append ADR-012 (erasure rights) |
| 10 | R-13 | 15 | Append ADR-013 (re-snapshot policy) |

## Risk lifecycle

- **OPEN**: chưa có mitigation file vật lý.
- **MITIGATING**: mitigation đang plan (ADR/file mới chưa merged).
- **MITIGATED**: mitigation merged vào workspace doc set.
- **EXECUTED**: Muscle đã apply trên code/infra.

Sau round 1 (2026-06-01): tất cả R-01..R-27 status = **MITIGATING** (qua workspace doc set update). Chuyển **MITIGATED** sau khi user approve doc + Brain sync xong tất cả file.

## Review schedule

- Lần 1: 2026-06-01 (file này tạo).
- Lần 2: ngay sau Muscle hoàn thành P0 — re-score R-01..R-08.
- Lần 3: ngay sau Muscle hoàn thành P1 — re-score R-03..R-11.
- Lần 4: ngay sau Muscle hoàn thành P2 — close-out.
