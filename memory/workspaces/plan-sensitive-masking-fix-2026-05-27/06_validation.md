# 06_validation — Acceptance & Verify Commands

## Verify pattern chung
1. **Build**: `go build ./...` (centralized-data-service + cdc-cms-service)
2. **Vet**: `go vet ./...`
3. **Unit test**: `go test ./...`
4. **Lint FE**: `pnpm lint && pnpm typecheck`
5. **E2E**: `go test -tags=integration ./...`
6. **Security**: /security-agent scan no high-severity.

---

## Phase P0 — Engine

| Task | Acceptance | Verify command |
|---|---|---|
| M-1 Migration | 3 column mới trong `cdc_mapping_rules`; 2 audit table tồn tại; seed default chia 3 cluster strategy đúng | `psql -c "\d cdc_system.cdc_mapping_rules"` + `psql -c "SELECT mask_strategy, COUNT(*) FROM cdc_system.cdc_mapping_rules GROUP BY 1"` |
| M-2 Strategy interface | 4 strategy register + resolve hoạt động | `go test ./internal/service/masking -v` |
| M-3 Key vault | Env-based load + cache + version + reject key < 32 chars | `go test ./pkgs/vault -v` |
| M-4 MaskingService refactor | Không còn `"***"` literal trong `masking_service.go` (DB path) | `grep -n '"\*\*\*"' internal/service/masking_service.go` → 0 match |
| M-5 Unit test | ≥ 90% coverage; assert `NotEqual("***",...)` lock anti-pattern | `go test ./internal/service -run TestMaskingService -cover` |

---

## Phase P1 — Worker Integration

| Task | Acceptance | Verify command |
|---|---|---|
| M-6 Mapper | `_raw_data` của shadow không chứa `"***"` cho field DROP/HMAC/PARTIAL | `psql -c "SELECT COUNT(*) FROM shadow.users WHERE _raw_data::text LIKE '%\"***\"%'"` = 0 |
| M-7 Buffer/Recon/Consumer | Failed_sync_log/recon_drift dùng strategy mới | `go test ./internal/handler -v` + `go test ./internal/service -run TestReconHeal -v` |
| M-8 Audit writer | Record insert vào `mask_audit_log` với sample rate đúng | Set rate=1.0 → run 100 event → `SELECT COUNT(*) FROM cdc_system.mask_audit_log` ≈ N field × 100 |
| M-9 E2E | testcontainers full pipeline PASS | `go test -tags=integration ./internal/service -run TestMaskingE2E -v` |

---

## Phase P2 — Admin + UI + Backfill

| Task | Acceptance | Verify command |
|---|---|---|
| M-10 API GET/PUT | Endpoint trả MaskConfig; PUT audit log row mới | `curl :8080/api/v1/admin/mapping-rules/42/mask-config` 200 + `SELECT COUNT(*) FROM cdc_system.mask_config_audit` ≥ 1 |
| M-11 UI tab | Tab Sensitive Masking render đủ 4 strategy option + preview hoạt động client-side | Browser navigate `/mapping-rules/42` → tab Masking → chọn PARTIAL → preview thay đổi |
| M-12 Backfill | Sau dry-run + thực thi, không còn `"***"` trong shadow | `psql -c "SELECT COUNT(*) FROM shadow.<table> WHERE _raw_data::text LIKE '%\"***\"%'"` = 0 |
| M-13 Compliance doc | File `docs/compliance/sensitive-masking-vn-law.md` tồn tại + mapping article ↔ control | `cat docs/compliance/sensitive-masking-vn-law.md \| grep -c "Điều"` ≥ 3 |

---

## Compliance evidence checklist (cho audit)

- [ ] Mapping rule field nhạy cảm có `mask_strategy` ≠ `NONE` cho 100% PII column.
- [ ] `mask_audit_log` có record trong 24h gần nhất (chứng minh control hoạt động).
- [ ] `mask_config_audit` có record cho mỗi UPDATE config (truy vết actor).
- [ ] Shadow PG `_raw_data` không có `"***"` literal cho field DROP/HMAC.
- [ ] Key version đang dùng ≥ 1, env var hiện diện (không bị missing).
- [ ] Backup K8s Secret encrypt at rest.
- [ ] /security-agent scan PASS (không leak key qua log/metric).

---

## Definition of Done global
- [ ] Migration apply PASS trên dev + staging.
- [ ] `centralized-data-service` build + vet + test PASS (kèm coverage ≥ 90% cho masking package).
- [ ] `cdc-cms-service` build + vet + test PASS.
- [ ] `cdc-cms-web` lint + typecheck + build PASS.
- [ ] E2E testcontainers PASS.
- [ ] Backfill chạy thành công trên ≥ 1 table staging.
- [ ] /security-agent scan PASS.
- [ ] Report `report_sensitive_masking_fix_2026-05-27.md` cập nhật final.
