# 04_decisions — ADRs

## ADR-001: HMAC-SHA256 vs SHA256 thuần
- **Context**: Khử định danh field như CCCD/card_number cần kháng được rainbow table attack.
- **Decision**: Chọn **HMAC-SHA256** với salt key bí mật.
- **Rationale**:
  - SHA256 thuần dễ bị brute-force rainbow table (CCCD 12 digit = 10^12 entry, khả thi).
  - HMAC với secret key 32+ bytes — attacker phải biết key mới tính lại được hash.
  - Standard library Go có sẵn `crypto/hmac`, không thêm dependency.
- **Alternative rejected**: bcrypt/scrypt — quá chậm cho hot path, không cần slowness (đây là de-identification, không phải password hashing).
- **Status**: Accepted.

## ADR-002: HMAC key storage — Vault vs K8s Secret + env
- **Context**: Cần single source of truth cho HMAC key + rotation support.
- **Decision**: **K8s Secret + env var** ở Phase 1; Vault ở roadmap tương lai.
- **Rationale**:
  - K8s Secret + env đủ dùng cho VN compliance hiện tại (encrypted at rest via etcd).
  - Vault tăng độ phức tạp + dependency thêm 1 service.
  - `KeyProvider` interface cho phép swap implementation sau (đã design abstraction).
- **Trade-off**: Operator có quyền `kubectl get secret` xem được key. Mitigation: RBAC chặt + Sealed Secrets nếu cần.
- **Status**: Accepted (revisit khi go global hoặc audit yêu cầu).

## ADR-003: Default strategy cho field hiện có
- **Context**: Migration cần set strategy mặc định cho field trong `sensitive_fields` của 100+ table.
- **Decision**: Mapping rule theo tên field (case-insensitive):
  - `password`, `pin`, `otp`, `cvv`, `secret`, `token` → **DROP**.
  - `card_number`, `cccd`, `cmnd`, `account_number` → **HASH_HMAC**.
  - `phone`, `email` → **PARTIAL** (prefix=0, suffix=3 cho phone, suffix=4 cho email local).
  - Field khác trong list → **DROP** (an toàn nhất).
- **Rationale**: Auto-classify giảm effort operator. Có thể override sau qua UI Admin.
- **Status**: Accepted.

## ADR-004: DDM (Postgres VIEW) — in scope hay defer?
- **Context**: User mention DDM (Dynamic Data Masking) qua RBAC tại DB đích.
- **Decision**: **Defer**.
- **Rationale**:
  - Hệ thống đã có HMAC + PARTIAL — đáp ứng compliance.
  - DDM bằng VIEW yêu cầu thay tất cả query path từ analyst tool — invasive.
  - Có thể bổ sung Phase 3 nếu auditor yêu cầu plaintext access.
- **Status**: Deferred (ghi vào backlog roadmap).

## ADR-005: Backfill "***" cũ — NULL hay HMAC?
- **Context**: Shadow PG có rows còn `"***"` literal.
- **Decision**: Re-mask theo strategy mới của mapping_rule (NULL nếu DROP, HMAC nếu HASH_HMAC, ...).
- **Rationale**:
  - Plaintext gốc đã mất → không thể HMAC chính xác. Trong trường hợp này, set `_raw_data` field về `null` (vì giá trị `"***"` không có ý nghĩa).
  - Backfill chỉ là cleanup; data quality về sau dựa trên event mới sync.
- **Status**: Accepted với caveat ghi rõ trong runbook.

## ADR-006: Audit log sample rate — 100% hay 1%?
- **Context**: Mọi field bị mask sinh 1 audit record → cardinality lớn (millions/day).
- **Decision**: **1% default, configurable**.
- **Rationale**:
  - Compliance không yêu cầu 100% — chỉ cần evidence control hoạt động.
  - Khi điều tra sự cố, có thể bump 100% trong cửa sổ hẹp.
  - 1% × 10M event/day = 100k record/day → manageable.
- **Status**: Accepted.

## ADR-007: Strategy enum — DB type vs check constraint
- **Context**: Cần đảm bảo `mask_strategy` value hợp lệ ở cả DB + Go.
- **Decision**: **DB ENUM TYPE** + Go constant.
- **Rationale**:
  - ENUM ép check ở DB layer → bảo vệ kể cả khi bypass app.
  - Migration thêm enum value dễ (`ALTER TYPE ... ADD VALUE`).
- **Alternative rejected**: VARCHAR + CHECK constraint — kém type-safe khi mở rộng.
- **Status**: Accepted.

## ADR-008: Loại bỏ `"***"` khỏi log path?
- **Context**: `text_sanitizer.go` dùng `"***"` cho error/log messages.
- **Decision**: **Giữ nguyên `text_sanitizer.go`**.
- **Rationale**:
  - Log không phải DB persistence → không vi phạm Điều 13 Accuracy.
  - Log đã rotate + retention ngắn, không nhằm mục đích audit dữ liệu KH.
  - Sửa toàn bộ sanitizer chỉ vì compliance là over-engineering.
- **Status**: Accepted (giữ `text_sanitizer.go`, chỉ refactor `masking_service.go`).

---

> Sau đây là ADR bổ sung từ Review round 1 (2026-06-01). Không sửa ADR cũ.

## ADR-009: Path-based vs flat strategy lookup
- **Context**: MongoDB document có nested object (e.g. `user.profile.cccd`). Plan P0/M-4 code demo dùng `for k, v := range data` flat one-level → field trong sub-document sẽ KHÔNG được mask → regression compliance.
- **Decision**: **Recursive walker** — đệ quy vào `map[string]any` và `[]any`, dispatch strategy ở leaf node theo `field_name` (không dùng full path).
- **Rationale**:
  - Bảo toàn semantic của `maskAnyRecursive` hiện tại.
  - `mapping_rule.target_column` đang là tên field, không phải JSON path → match theo leaf key đơn giản nhất.
  - Nếu cần path-based (e.g. `user.profile.cccd` khác `transaction.cccd`) → defer ADR riêng (out of scope round 1).
- **Alternative rejected**: Flat lookup — bỏ sót nested.
- **Trade-off**: Cùng tên field ở 2 nested level dùng cùng strategy. Chấp nhận.
- **Status**: Accepted.

## ADR-010: Deploy ordering (migration vs code)
- **Context**: Nếu apply migration UPDATE seed `mask_strategy='DROP'` TRƯỚC khi deploy code Worker mới → Worker cũ vẫn ghi `"***"` cho row mới + audit log empty → vi phạm compliance trong cửa sổ gap.
- **Decision**: **Migration DDL** → **Worker code** → **Migration seed UPDATE** → **CMS API + UI** → **Backfill**.
- **Rationale**:
  - DDL với DEFAULT NONE backward compat: Worker cũ bỏ qua column mới an toàn.
  - Worker mới đọc column NONE → no-op, hành vi như cũ.
  - Seed UPDATE chỉ chạy khi Worker đã sẵn sàng đọc strategy.
- **Implementation**: chi tiết `12_rollout_runbook.md` Stage 1–6.
- **Status**: Accepted (HARD constraint).

## ADR-011: HMAC với empty string input
- **Context**: `HmacStrategy.Apply` hiện tại chỉ check `nil`. `HMAC(key, "")` deterministic → attacker phát hiện "card_number rỗng" cluster.
- **Decision**: Empty string → return `nil` (treat as nullable, không hash).
- **Rationale**:
  - Field rỗng không có ý nghĩa đối soát → không cần hash.
  - Giảm leak structure (không biết bao nhiêu record có card_number rỗng qua hash collision count).
- **Alternative rejected**: Pepper per-row random — mất tính deterministic, không đối soát được.
- **Status**: Accepted.

## ADR-012: Right-to-erasure (Luật 91/2025 Điều 16)
- **Context**: HMAC một chiều → khi user yêu cầu xoá data, không thể locate record của họ ở shadow nếu chỉ có hash.
- **Decision**: **Erasure propagation qua CDC tombstone** từ source service.
- **Rationale**:
  - Source MongoDB là single source of truth — xoá tại source → CDC bắn delete event → Worker propagate xoá row ở shadow theo primary key, KHÔNG cần biết HMAC value.
  - Shadow chỉ là replica, không có erasure logic độc lập.
  - Document rõ trong `docs/compliance/sensitive-masking-vn-law.md` Phase 2.
- **Alternative rejected**: Giữ map `user_id → hash_set` — phình storage + lộ join risk.
- **Caveat**: Source service phải có erasure endpoint (out of scope plan này, có ticket riêng cho team Source).
- **Status**: Accepted (Phase 1: document only; Phase 2 implementation tách scope).

## ADR-013: Backfill policy — re-snapshot vs set null
- **Context**: ADR-005 cũ chốt "set `_raw_data` field về null vì plaintext đã mất". Nhưng đây là **data loss** ảnh hưởng analytics + audit history.
- **Decision (revised)**:
  - **Bước 1**: Trigger re-snapshot từ source MongoDB cho table còn legacy `"***"`. Source vẫn có plaintext → CDC mới sẽ apply strategy đúng.
  - **Bước 2**: Sau khi re-snapshot xong, drop legacy row có `"***"` (đã được thay bằng row mới).
  - **Bước 3**: Nếu source không còn (data đã expire) → fallback set null + ghi vào `mask_backfill_loss_log` table (audit evidence).
- **Rationale**:
  - Giữ data analytics integrity.
  - Compliance audit có evidence rõ ràng record nào lost (do source expire).
- **Trade-off**: Re-snapshot có cost I/O + thời gian. Chấp nhận trong off-peak window.
- **Status**: Accepted (SUPERSEDES ADR-005 cho phần policy).

## ADR-014: Multi-strategy per logical field
- **Context**: Field `email` có thể cần CẢ: HMAC (cho dedup analytics) + PARTIAL (cho support display). Schema hiện chỉ 1 strategy/column.
- **Decision**: **Out of scope Phase 1**. Phase 1 = 1 strategy/field. Phase 2 hỗ trợ multi-output qua shadow column derivative.
- **Rationale Phase 1**:
  - 90% use case fintech VN chỉ cần 1 strategy/field.
  - Đa output yêu cầu schema shadow thay đổi (`email_hash`, `email_display` columns) — invasive.
- **Phase 2 design (preview)**:
  - `cdc_mapping_rules.derived_columns JSONB` — array `[{column: "email_hash", strategy: "HASH_HMAC"}, {column: "email_display", strategy: "PARTIAL"}]`.
  - Worker apply tất cả strategy, write multiple column.
- **Status**: Deferred (Phase 2 roadmap).

## ADR-015: Migration must have DOWN file
- **Context**: Migration `068_add_mask_strategy.sql` không có file rollback.
- **Decision**: Convention dự án — mọi migration PHẢI có `*.down.sql` đi kèm.
- **Rationale**: Rollback procedure trong `12_rollout_runbook.md` Trường hợp 3 cần DOWN script ready.
- **Status**: Accepted.
