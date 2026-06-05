# 08_tasks_phase_p0 — Checklist Muscle Phase P0 (Engine)

> Reference: `03_implementation_phase_p0.md`. Effort 16h.

## M-1 — Migration mask_strategy + audit tables (3h)
- [ ] Tạo NEW `cdc-cms-service/migrations/schema/core/015_mask_strategy.sql` với:
  - `CREATE TYPE cdc_system.mask_strategy AS ENUM ('NONE','DROP','HASH_HMAC','PARTIAL')`.
  - `ALTER TABLE cdc_mapping_rules ADD COLUMN mask_strategy/mask_options/mask_key_version`.
  - `CREATE TABLE cdc_system.mask_audit_log` + index.
  - `CREATE TABLE cdc_system.mask_config_audit` + FK + check.
  - Seed UPDATE phân loại theo tên field (DROP cho password/OTP/PIN; HASH_HMAC cho CCCD/card_number; PARTIAL cho phone/email).
- [ ] Sửa `centralized-data-service/internal/model/mapping_rule.go` thêm 3 field GORM.
- [ ] Apply migration trên dev: `make migrate-up`.
- [ ] Verify: `psql -c "\d cdc_system.cdc_mapping_rules" | grep mask_strategy`.

## M-2 — Strategy interface + 4 implementation (4h)
- [ ] Tạo NEW package `centralized-data-service/internal/service/masking/`:
  - `strategy.go` — `Strategy interface`, `Input`, `Output`, `Registry`.
  - `none.go` — NoneStrategy.
  - `drop.go` — DropStrategy (return nil + ShouldDrop=true).
  - `hmac.go` — HmacStrategy với `KeyProvider` injection.
  - `partial.go` — PartialStrategy với options (prefix/suffix/placeholder).
- [ ] Mỗi strategy có doc comment + example.
- [ ] Test: `go test ./internal/service/masking/ -v` PASS.

## M-3 — HMAC key vault (2h)
- [ ] Tạo NEW `centralized-data-service/pkgs/vault/key_loader.go`:
  - Struct `KeyLoader` với mutex + cache `map[int16][]byte`.
  - Method `Get(ctx, version int16)` đọc env `MASKING_HMAC_KEY_V{n}`.
  - Reject key < 32 chars.
- [ ] Sửa `internal/config/config.go` thêm `MaskingConfig` struct.
- [ ] Tạo K8s Secret template `deployments/k8s/cdc-masking-keys-secret.yaml`.
- [ ] Test: `go test ./pkgs/vault/ -v` PASS với env injection.

## M-4 — MaskingService refactor (5h)
- [ ] Sửa `internal/service/masking_service.go`:
  - Constructor mới: `NewMaskingService(registry, ruleRepo, auditCh, logger)`.
  - `MaskTableData` lookup `mapping_rule.mask_strategy` per field + dispatch via registry.
  - DROP → `out[k] = nil` (NOT `"***"`).
  - Unknown strategy → fallback DROP + warn log.
  - Audit emit non-blocking qua channel.
- [ ] **GIỮ NGUYÊN** `internal/service/text_sanitizer.go` (ADR-008).
- [ ] Verify: `grep -n '"\*\*\*"' internal/service/masking_service.go` → 0 match.
- [ ] Build: `go build ./internal/service/...` PASS.

## M-5 — Unit test (2h)
- [ ] Tạo NEW `internal/service/masking_service_test.go` với 5 test:
  - `TestMaskingService_NoneStrategy` — non-sensitive giữ nguyên.
  - `TestMaskingService_DropStrategy` — assert nil + NotEqual `"***"`.
  - `TestMaskingService_HmacStrategy_Deterministic` — same input → same output, 64 chars.
  - `TestMaskingService_PartialStrategy` — `"0901234567"` → `"*******567"`.
  - `TestMaskingService_AuditEmitted` — channel receive record.
- [ ] Mock `MappingRuleRepo` interface.
- [ ] Coverage ≥ 90%: `go test -cover ./internal/service -run TestMaskingService`.

## Post-phase
- [ ] `go build ./... && go vet ./... && go test ./...` PASS.
- [ ] /security-agent scan no high.
- [ ] APPEND `05_progress.md` entry "P0 executed by Muscle".
