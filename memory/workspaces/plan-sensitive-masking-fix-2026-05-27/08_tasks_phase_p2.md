# 08_tasks_phase_p2 — Checklist Muscle Phase P2 (Admin + UI + Backfill)

> Reference: `03_implementation_phase_p2.md`. Effort 14h.

## M-10 — API CRUD mask-config (4h)

### Backend cdc-cms-service
- [ ] Tạo NEW `internal/api/dto/mask_config_dto.go` với struct `MaskConfigRequest`, `MaskConfigResponse`, `MaskAuditItem`.
- [ ] Tạo NEW `internal/api/mask_config_handler.go` với 3 method `Get`, `Update`, `Audit`.
- [ ] Tạo NEW `internal/app/queries/get_mask_config.go` + `list_mask_audit.go`.
- [ ] Tạo NEW `internal/app/commands/update_mask_config.go`:
  - Transaction: UPDATE `cdc_mapping_rules` + INSERT `mask_config_audit`.
  - Validate strategy ∈ enum + key_version ≥ 1.
- [ ] Sửa `internal/router/router.go` thêm 3 route dưới `adminGroup`:
  - `GET /api/v1/admin/mapping-rules/:id/mask-config`
  - `PUT /api/v1/admin/mapping-rules/:id/mask-config`
  - `GET /api/v1/admin/mapping-rules/:id/mask-config/audit`
- [ ] Sửa `internal/server/server.go` wire DI cho MaskConfigHandler.
- [ ] Test NEW `internal/api/mask_config_handler_test.go` (fiber httptest).
- [ ] Verify: `curl -X PUT :8080/api/v1/admin/mapping-rules/42/mask-config -H "Authorization: Bearer $ADMIN" -d '{...}'` → 204; audit row insert.

## M-11 — Admin UI tab Sensitive Masking (6h)

### Frontend cdc-cms-web
- [ ] Tạo NEW `src/types/masking.ts`: types `MaskStrategy`, `MaskConfig`, `MaskAuditItem`.
- [ ] Tạo NEW `src/hooks/useMaskConfig.ts`:
  - `useMaskConfig(id)` — React Query GET.
  - `useUpdateMaskConfig(id)` — mutation PUT + invalidate.
  - `useMaskAudit(id, page)` — React Query GET audit log.
- [ ] Tạo NEW `src/components/masking/StrategySelector.tsx`:
  - Select dropdown 4 strategy.
  - Alert thông tin legal + use case theo strategy.
- [ ] Tạo NEW `src/components/masking/MaskPreview.tsx`:
  - Input sample + output preview client-side.
  - HMAC chỉ show placeholder vì cần secret server-side.
- [ ] Tạo NEW `src/components/masking/MaskingTab.tsx` aggregate Selector + Preview + Audit history Table.
- [ ] Sửa `src/pages/MappingRuleEditPage.tsx` thêm Tab "Sensitive Masking".
- [ ] Verify FE:
  - `pnpm lint && pnpm typecheck && pnpm build` PASS.
  - Browser navigate `/mapping-rules/42` → tab Masking → chọn strategy → save → audit history hiện row mới.

## M-12 — Backfill script (2h)
- [ ] Tạo NEW `centralized-data-service/scripts/backfill_mask.go` với build tag `//go:build backfill`:
  - Flag: `-dsn`, `-table`, `-batch`, `-dry-run`.
  - Query `WHERE _raw_data::text LIKE '%"***"%'` LIMIT batch.
  - Re-mask qua MaskingService.
  - Dry-run print sample; thực tế UPDATE.
- [ ] Test trên staging:
  - Dry-run trước, check sample output.
  - Thực thi với `-dry-run=false`.
  - Verify `SELECT COUNT(*) WHERE _raw_data::text LIKE '%"***"%'` = 0.

## M-13 — Compliance evidence doc (2h)
- [ ] Tạo NEW `docs/compliance/sensitive-masking-vn-law.md`:
  - Mapping table: Văn bản pháp lý ↔ Control.
  - Strategy decision matrix.
  - Audit trail evidence.
  - Key rotation procedure.
- [ ] Verify: `cat docs/compliance/sensitive-masking-vn-law.md | grep -c "Điều"` ≥ 3.

## Post-phase
- [ ] Build/vet/test 3 service PASS.
- [ ] Lint/typecheck FE PASS.
- [ ] /security-agent scan PASS.
- [ ] APPEND `05_progress.md`.
- [ ] Update `report_sensitive_masking_fix_2026-05-27.md` với kết quả backfill (số row migrated, ms thời gian, sample evidence).
