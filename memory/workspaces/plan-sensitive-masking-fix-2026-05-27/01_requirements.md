# 01_requirements — Sensitive Masking Compliance

## Functional Requirements (FR)

### FR-1 — Mask Strategy Engine (multi-strategy)
Hệ thống PHẢI hỗ trợ ≥ 4 chiến lược masking, configurable per-field qua `mapping_rule`:

| Strategy | Mô tả | Use case ngân hàng |
|---|---|---|
| `NONE` | Không mask, giữ nguyên giá trị | Field non-sensitive (trans_id, created_at) |
| `DROP` | Loại bỏ hoàn toàn → `NULL` ở shadow | password, OTP, PIN, CVV |
| `HASH_HMAC` | `HMAC-SHA256(value, secret_salt)` | CCCD, card_number, account_number (giữ tính đối soát/đếm distinct) |
| `PARTIAL` | Format-preserving mask (e.g., `**** **** **** 1234`) | Card display, phone display |
| `TOKENIZE` | (Future) FPE token via vault | Long-term roadmap |

### FR-2 — Per-field strategy config
`cdc_mapping_rules` PHẢI có column `mask_strategy` (enum) + `mask_options` (JSONB) override `cdc_table_registry.sensitive_fields` mặc định.

### FR-3 — Salt/Key Management
- HMAC secret key PHẢI đọc từ **Vault** (hoặc K8s Secret + env var), KHÔNG hardcode.
- Key rotation supported (versioned: `salt_v1`, `salt_v2`).
- Audit log mỗi lần rotate key.

### FR-4 — API CRUD masking config
- `GET /api/v1/admin/mapping-rules/{id}/mask-config`
- `PUT /api/v1/admin/mapping-rules/{id}/mask-config`
- Endpoint require role `admin` (đã có `RequireRole("admin")` pattern).
- Audit log mọi thay đổi vào `cdc_system.mask_config_audit`.

### FR-5 — Admin UI (cdc-cms-web)
- Trang `/mapping-rules/{id}` thêm tab `Sensitive Masking`.
- Mỗi column show dropdown chọn strategy + JSON editor cho options.
- Preview trước khi save: hiển thị sample input → output.

### FR-6 — Migration kế thừa
- Migration không break dữ liệu cũ. Tất cả mapping_rule hiện tại default `NONE` cho non-sensitive, `DROP` cho field trong `sensitive_fields` list.
- Backfill option (manual run): re-mask shadow data theo strategy mới.

### FR-7 — DDM cho audit role (optional, ADR)
- Tạo Postgres `VIEW` cho role `auditor` show plaintext, role `analyst` show masked.
- (ADR sẽ quyết định scope.)

## Non-Functional Requirements (NFR)

### NFR-1 — Performance
- Masking overhead < 5ms p99 per event (HMAC-SHA256 ~1μs).
- Throughput không drop > 5% so với baseline.

### NFR-2 — Compliance audit trail
- Mọi field bị mask PHẢI log một dòng: `[event_id, table, field, strategy, key_version]` vào table `cdc_system.mask_audit_log` (sample rate configurable, default 1%).

### NFR-3 — Security
- Salt key chỉ accessible bởi worker process, không leak qua log/metric.
- HMAC output 32 bytes hex (64 chars) — không leak length pattern của plaintext.

### NFR-4 — Backward compat
- Field hiện đang có `"***"` literal trong shadow PG: cung cấp script backfill chuyển sang `NULL` (DROP) hoặc HMAC tùy strategy chốt.

## Definition of Done

### DoD-1 — Schema & Migration
- [ ] Migration thêm `mask_strategy`, `mask_options` vào `cdc_mapping_rules`.
- [ ] Migration tạo `cdc_system.mask_audit_log` + `cdc_system.mask_config_audit`.
- [ ] Migration seed default strategy cho field nhạy cảm hiện có.

### DoD-2 — Masking Engine
- [ ] `masking_service.go` refactor: thêm `Strategy` interface + 4 implementation (NoneStrategy, DropStrategy, HmacStrategy, PartialStrategy).
- [ ] Loại bỏ hardcode `"***"` khỏi path DB persistence (chỉ giữ trong `text_sanitizer.go` cho log path — đó là acceptable vì log không phải DB).
- [ ] HMAC key đọc từ env `MASKING_HMAC_KEY` + version `MASKING_KEY_VERSION`.
- [ ] Unit test `masking_service_test.go` coverage ≥ 90%.

### DoD-3 — Worker integration
- [ ] `dynamic_mapper.go` apply strategy theo `mapping_rule.mask_strategy`.
- [ ] `batch_buffer.go`, `recon_heal.go`, `kafka_consumer.go` integration consistent.
- [ ] DROP strategy → field value `nil` (json null) thay vì `"***"`.

### DoD-4 — API + UI
- [ ] `cdc-cms-service` endpoint GET/PUT mask-config với audit log.
- [ ] `cdc-cms-web` trang config trong tab Sensitive Masking.

### DoD-5 — Backfill
- [ ] Script `scripts/backfill_mask.go` re-mask shadow data theo strategy mới (idempotent, batch 1000 rows).

### DoD-6 — Compliance evidence
- [ ] Document mapping criteria → law article trong `04_decisions.md`.
- [ ] Audit log table sample được generate qua test (proof of write).

### DoD-7 — Governance
- [ ] §11 APPEND-only memory.
- [ ] §12 Brain Code Prohibition (Plan này không touch .go/.ts/.sql).
- [ ] §14 Pre-flight verify file count.
- [ ] /security-agent scan sau Muscle execute.
- [ ] Report `report_sensitive_masking_fix_2026-05-27.md`.
