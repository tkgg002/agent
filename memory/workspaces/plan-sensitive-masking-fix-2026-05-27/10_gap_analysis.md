# 10_gap_analysis — Sensitive Masking Compliance Gap Analysis

## Gap → Fix → Verify map

| # | Gap hiện tại | Điều luật vi phạm | Fix (task M-x) | Verify |
|---|---|---|---|---|
| **M1** | Hardcode `"***"` literal tại `masking_service.go:91,133,152-153` | Luật 91/2025 Điều 13 (Accuracy) | M-4 refactor + dispatch Strategy | `grep -n '"\*\*\*"' masking_service.go` = 0 |
| **M2** | Không có `mask_strategy` per-field | NĐ 356 "biện pháp kỹ thuật phù hợp" | M-1 migration ALTER TABLE + ENUM | `\d cdc_mapping_rules \| grep mask_strategy` |
| **M3** | Masking apply ở write-path (phá hủy gốc) | Luật 91/2025 Điều 13 + Điều 14 quyền yêu cầu | M-4 + M-7: DROP → null; HMAC giữ tính đối soát | E2E test PASS |
| **M4** | Không có API CRUD masking config | VBHN 25 "kiểm toán cấu hình" | M-10 API + audit log | curl PUT → 204 + audit row |
| **M5** | Không có audit log mỗi field bị mask | VBHN 25 thanh tra evidence | M-1 table + M-8 AuditWriter | `SELECT COUNT(*) FROM mask_audit_log` > 0 |
| **M6** | Không có doc compliance | Audit thanh tra | M-13 `docs/compliance/sensitive-masking-vn-law.md` | File tồn tại + grep "Điều" ≥ 3 |
| **M7** | Test assert `"***"` (lock anti-pattern) | (test smell) | M-5 + M-6 + M-7 + M-9 cập nhật assert `NotEqual("***",...)` | Test coverage ≥ 90% |
| **M8** | Không có per-field strategy → 1 cách duy nhất cho mọi loại field | Pháp lý yêu cầu tỷ lệ thuận (proportionality) — không over-mask + không under-mask | M-2 Strategy engine 4 mode + M-1 seed default | Distribution `GROUP BY mask_strategy` |
| **M9** | HMAC key không có (chưa có hash) → nếu dùng SHA256 thuần sẽ rainbow table | NĐ 356 "biện pháp kỹ thuật" | M-3 KeyProvider + env loader + version | Test `TestHmacStrategy_Deterministic` PASS |
| **M10** | Shadow PG có rows `"***"` legacy | Dữ liệu rác + risk audit thấy | M-12 backfill script | `SELECT COUNT WHERE LIKE '%"***"%'` = 0 |

## Strategy distribution (kỳ vọng sau seed M-1)

Dựa vào field classification trong ADR-003 + `cdc_table_registry.sensitive_fields`:

| Strategy | Field ví dụ | Tỷ lệ ước tính |
|---|---|---|
| NONE | trans_id, created_at, source_code... | ~70% (non-sensitive) |
| DROP | password, OTP, PIN, CVV, secret, token | ~10% |
| HASH_HMAC | CCCD, card_number, account_number | ~10% |
| PARTIAL | phone, email | ~10% |

> Số liệu chính xác chỉ có sau khi Muscle apply migration trên dev — Brain không thể tính láo.

## Compliance evidence post-fix

Sau P0+P1+P2, hệ thống PHẢI có:
- ✓ Field nhạy cảm 100% có `mask_strategy` ≠ NONE.
- ✓ `mask_audit_log` có record trong 24h gần nhất (sample 1%).
- ✓ `mask_config_audit` có record cho mỗi UPDATE config.
- ✓ Shadow PG không còn `"***"` literal trong `_raw_data`.
- ✓ K8s Secret `cdc-masking-keys` tồn tại + encrypted at rest.
- ✓ `docs/compliance/sensitive-masking-vn-law.md` mapping article ↔ control.

## Out-of-scope (defer roadmap)

- **DDM (Postgres VIEW)** — ADR-004 defer. Có thể bổ sung Phase 3 nếu auditor yêu cầu plaintext access cho role `auditor`.
- **Tokenization (FPE)** — cần vault + token registry, scope lớn.
- **Vault integration** (thay env) — ADR-002 defer khi go global.
- **GDPR compliance** — không trong scope hiện tại (luật VN trước, EU sau).
- **Multi-strategy per field** — ADR-014 defer Phase 2.

---

## Review Round 1 — 2026-06-01: Gap bổ sung

> 27 gap mới phát hiện qua cross-check code thật vs plan. Chia 3 mức: CRITICAL (blocker execute) / HIGH (nên fix trước execute) / MEDIUM (nice-to-have).

### CRITICAL (7 gap)

| # | Gap | Evidence | Fix | Verify |
|---|---|---|---|---|
| **C1** | Migration số `015` conflict với `067` hiện tại | `ls data-hub/cdc-cms-service/migrations/schema/core/` → max=067 | Rename → `068_add_mask_strategy.sql` (P0/M-1 sửa) | `ls 068_*.sql` exists |
| **C2** | Path thiếu prefix `data-hub/` | Plan dùng `centralized-data-service/...`, actual root `data-hub/centralized-data-service/` | Sửa toàn bộ doc P0/P1/P2 | `grep -rn "data-hub/centralized" 03_implementation_*` ≥ 1 |
| **C3** | 3 caller bị bỏ sót: `dlq_handler.go`, `dlq_worker.go`, `schema_inspector.go` | `grep -rn "MaskTableData\|MaskJSONPayload\|MaskFieldSample" internal/` = 22 call-site, plan chỉ liệt kê 19 | Bổ sung P1/M-7b + `13_caller_inventory.md` | Checklist 22/22 trong inventory |
| **C4** | Signature `MaskTableData` thay đổi đột ngột → 22 call-site break | Hiện tại `func (table, data) map`, plan đề xuất `func (ctx, eventID, sourceCode, table, data) (map, error)` | Dual-method: giữ legacy + thêm `*Ctx` variant | `grep "MaskTableData(" internal/` còn caller cũ + caller mới song song |
| **C5** | Regression nested object — code demo flatten one-level | Plan `for k, v := range data` không recursive; hiện tại `maskAnyRecursive` đệ quy map+array | ADR-009 + P0/M-4 dùng recursive walker | Test `TestMaskingService_NestedObject` PASS |
| **C6** | Deploy ordering race condition (migration seed trước Worker code) | Chưa có ADR sequencing | ADR-010 + `12_rollout_runbook.md` | Runbook Stage 1–6 ready |
| **C7** | ADR count inconsistency (02_plan=7, 04_decisions=8) | `02_plan.md:65-72` list 7 ADR; `04_decisions.md` có ADR-001..008 | Sync 02_plan lên 8 ADR | `grep -c "^## ADR-" 04_decisions.md` = 8 (cũ) + 7 (mới) = 15 |

### HIGH (9 gap)

| # | Gap | Fix | Verify |
|---|---|---|---|
| **H1** | Rule lookup hot-path không cache → DB query 10M lần/ngày | M-2b: sync.Map cache + pub-sub invalidation từ CMS | Benchmark `BenchmarkMaskTableData_Cached` < 1μs |
| **H2** | HMAC `fmt.Sprintf("%v", any)` không deterministic cho float/BSON | `normalizeValue(v any) string` per-type (int→%d, float→strconv, BSON.ObjectID→Hex) | Test `TestHmacStrategy_TypeNormalization` PASS với 5 type |
| **H3** | Empty string HMAC leak | ADR-011: `""` → return nil | Test `TestHmacStrategy_EmptyString_ReturnsNil` PASS |
| **H4** | Audit log phình table (15M/tháng) | M-1b: PARTITION BY RANGE (masked_at) + retention 13 tháng | `\d+ mask_audit_log` show partition |
| **H5** | Right-to-erasure (Điều 16) missing | ADR-012: tombstone CDC từ source | Document trong `docs/compliance/.../erasure.md` |
| **H6** | Migration không có DOWN | ADR-015: convention bắt buộc `068_add_mask_strategy.down.sql` | File `*.down.sql` exists |
| **H7** | Backfill set null = mất data | ADR-013 supersedes ADR-005: re-snapshot từ source MongoDB | `mask_backfill_loss_log` table tồn tại |
| **H8** | Multi-strategy field (email cần hash + partial) | ADR-014 defer Phase 2 | Backlog entry trong `00_context.md` (đã có Phase 2 note) |
| **H9** | Mask config version per-row missing | Thêm `_mask_version` shadow column + compare backfill | Schema mới có column này |

### MEDIUM (11 gap)

| # | Gap | Fix | Owner |
|---|---|---|---|
| **M-r1** | NFR-3 (key không leak qua log) không có verify command | Add `grep -rEn "MASKING_HMAC_KEY\|salt"` post-test | Muscle |
| **M-r2** | Coverage 90% không enforce hard gate | `-coverpkg=./internal/service/masking/...` + threshold script CI | Muscle |
| **M-r3** | M-10 thiếu OpenAPI contract | `api/openapi/mask-config.yaml` | Brain |
| **M-r4** | M-11 UI 6h optimistic (4 component + audit list + preview) | Bump → 10h | Brain |
| **M-r5** | Performance baseline trước/sau missing | M-5b: `BenchmarkMaskTableData_Before_After` | Muscle |
| **M-r6** | Sealed Secrets vs etcd encryption không specific | ADR-002 mở rộng: chọn bitnami sealed-secrets | Ops |
| **M-r7** | Unicode NFC normalization thiếu | `golang.org/x/text/unicode/norm` cho HMAC input | Muscle |
| **M-r8** | `MaskFieldSample` (schema_inspector) chưa được refactor | M-4b: return metadata thay vì `"***"` | Muscle |
| **M-r9** | Log path có `zap.Any(rawPayload)` chưa? Chưa audit | `grep -rn "zap.Any\|zap.Reflect" internal/` rồi review | Muscle |
| **M-r10** | Apply migration thứ tự dev→staging→prod không nêu | `12_rollout_runbook.md` Pre-flight | Ops |
| **M-r11** | Lesson global cụ thể (ID) chưa list trong progress | Update Entry 06 reference L-2026-05-26-..., L-63 | Brain |

## Tổng kết review

- **Gap baseline (M1–M10)**: ✓ đã có trong plan gốc.
- **Gap bổ sung Round 1**: 27 (C1–C7, H1–H9, M-r1..M-r11).
- **File workspace doc set**: 18 → 21 (thêm 11_risk_register, 12_rollout_runbook, 13_caller_inventory).
- **ADR**: 8 → 15 (thêm ADR-009..015).
- **Effort bổ sung Brain**: ~6h (file mới + ADR + edit).
- **Effort bổ sung Muscle**: ~10h (recursive walker, cache, normalize, backfill re-snapshot, audit partition).
- **Tổng effort plan**: 40h + 10h = **50h Muscle**.
