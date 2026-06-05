# report_2026-06-01_simplified_execution — Phase 1 (S-1..S-3) DONE

> Workspace: `plan-sensitive-masking-fix-2026-05-27`
> Plan nguồn: `14_simplified_plan.md`
> Người thực thi: Muscle (Claude Code) theo verb `execute s1`
> Date: 2026-06-01

## 1. Tóm tắt thay đổi

Thay literal `"***"` ở 3 vị trí **field-value masking** bằng HMAC-SHA256 hash để giữ tính đối soát (Luật 91/2025 Điều 13 — Accuracy) thay vì phá huỷ data. Giữ nguyên 2 vị trí `"***"` ở `MaskJSONPayload` invalid-JSON fallback (không phải PII, chỉ là indicator) theo nguyên tắc Simplicity First (§6 CLAUDE.md).

## 2. Files thay đổi

| # | File | Loại | LOC delta | Ghi chú |
|---|------|------|-----------|---------|
| 1 | `data-hub/centralized-data-service/internal/service/masking_service.go` | MODIFY | **+28 net** (215 → 243 dòng) | Thêm imports, field `hmacKey` + `hmacKeyWarnOnce`, method `hashValue()`, replace 3 `"***"` literal |
| 2 | `data-hub/centralized-data-service/internal/service/masking_service_test.go` | NEW | **+157** | 10 test case (deterministic, non-leak, length, nil, non-string, fallback, MaskTableData, MaskFieldSample, nested recursive, JSON payload) |

**Tổng**: 2 files, +185 LOC ròng (28 logic + 157 test).

### Chi tiết delta `masking_service.go`

| Block | Trước | Sau | Lý do |
|---|---|---|---|
| Imports (line 3-15) | 6 imports | 10 imports (+4) | Cần `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `os` |
| Struct `MaskingService` (line 32-39) | 4 fields | 6 fields (+2) | Thêm `hmacKey []byte` + `hmacKeyWarnOnce sync.Once` |
| Constructor `NewMaskingService` (line 41-52) | 4 field init | 5 field init (+1) | Load `os.Getenv("MASKING_HMAC_KEY")` |
| `MaskFieldSample` (line 96-101) | `return "***"` | `return ms.hashValue(value)` | Field-value masking |
| `maskMapRecursive` (line 136-146) | `out[key] = "***"` | `out[key] = ms.hashValue(value)` | Field-value masking |
| `maskAnyRecursive` default branch (line 148-164) | `return "***"` | `return ms.hashValue(value)` | Field-value masking |
| **NEW** method `hashValue` (line 188-208) | — | 21 dòng (HMAC-SHA256 + nil/empty + fallback) | Helper hash chính |

### Vị trí `"***"` còn lại (CHỦ ĐÍCH giữ)

| Line | Context | Lý do giữ |
|---|---|---|
| 78 | `MaskJSONPayload` — JSON invalid → `{"raw": "***"}` | Indicator cho invalid JSON, không phải PII. Không leak gì. |
| 84 | `MaskJSONPayload` — JSON unmarshal fail → `{"raw": "***"}` | Tương tự line 78. |
| 188-190 | Comment giải thích | Doc string. |
| 203 | `hashValue` fallback khi `MASKING_HMAC_KEY` chưa set | An toàn rollback — ops chưa kịp set env không làm crash prod, chỉ log warn 1 lần (sync.Once). |

## 3. Kết quả verify

### 3.1 Build
```bash
$ cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service
$ go build ./internal/service/...
# PASS (no output)

$ go build ./cmd/... ./internal/... ./pkgs/...
# PASS (no output) — toàn bộ production code build sạch
```

Lưu ý: `go build ./...` có 2 errors ở `scratch/` (multiple `main` declarations) — **PRE-EXISTING**, không liên quan tới masking, không nằm trên production code path.

### 3.2 Unit test
```bash
$ go test ./internal/service/ -run "TestHashValue|TestMaskTableData|TestMaskFieldSample|TestMaskAnyRecursive|TestMaskJSONPayload" -v
=== RUN   TestHashValue_Deterministic                        --- PASS (0.00s)
=== RUN   TestHashValue_DifferentInputs                      --- PASS (0.00s)
=== RUN   TestHashValue_NilOrEmpty                           --- PASS (0.00s)
=== RUN   TestHashValue_NonString                            --- PASS (0.00s)
=== RUN   TestHashValue_FallbackWhenKeyMissing               --- PASS (0.00s)
=== RUN   TestMaskTableData_UsesHashForSensitiveField        --- PASS (0.00s)
=== RUN   TestMaskFieldSample_UsesHash                       --- PASS (0.00s)
=== RUN   TestMaskAnyRecursive_NestedSensitive               --- PASS (0.00s)
=== RUN   TestMaskJSONPayload_NoStarLiteralForValidData      --- PASS (0.00s)
=== RUN   TestMaskJSONPayload_InvalidJSONKeepsStarFallback   --- PASS (0.00s)
PASS — 10/10 (0.666s)
```

### 3.3 Regression test
```bash
$ go test ./internal/... -count=1 -short
ok  centralized-data-service/internal/handler  0.761s
ok  centralized-data-service/internal/service  0.512s
# tất cả package internal PASS, không regression caller MaskTableData/MaskJSONPayload/MaskFieldSample
```

### 3.4 Vet
```bash
$ go vet ./internal/service/
# Không có warning mới từ masking_service.go.
# (2 warning ở pkgs/idgen/sonyflake.go là pre-existing, không liên quan.)
```

### 3.5 Final grep `"***"` trong masking path
```bash
$ grep -n '"\*\*\*"' internal/service/masking_service.go
78:  invalid JSON fallback (giữ chủ đích)
84:  unmarshal fallback (giữ chủ đích)
188: comment
190: comment
203: env-not-set fallback (giữ chủ đích, có log warn)
```
→ KHÔNG còn `"***"` ở field-value masking path. ✅

## 4. Acceptance criteria

| # | Tiêu chí | Trạng thái |
|---|---|---|
| AC-1 | Bỏ literal `"***"` ở 3 vị trí field-value masking (line 91, 133, 153 cũ) | ✅ |
| AC-2 | Helper `hashValue()` dùng HMAC-SHA256 với secret key từ env | ✅ |
| AC-3 | Cùng input → cùng output (deterministic, đối soát được) | ✅ test PASS |
| AC-4 | Khác input → khác output | ✅ test PASS |
| AC-5 | Nil/empty → "" (không leak structure) | ✅ test PASS |
| AC-6 | Fallback `"***"` khi env chưa set (không crash prod), log warn 1 lần | ✅ test PASS |
| AC-7 | Signature `MaskTableData/MaskJSONPayload/MaskFieldSample` KHÔNG đổi | ✅ — 22 caller hot-path không break |
| AC-8 | Nested map + slice masking vẫn đúng | ✅ test PASS |
| AC-9 | `go build` + `go test` PASS toàn `internal/` | ✅ |
| AC-10 | Không regression test cũ | ✅ — internal/handler + internal/service đều PASS |

## 5. Deploy guide

### 5.1 Pre-deploy
1. Sinh key:
   ```bash
   openssl rand -hex 32
   ```
2. Set K8s Secret `MASKING_HMAC_KEY` cho cluster đích (staging trước, prod sau):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: centralized-data-service-secrets
   type: Opaque
   data:
     MASKING_HMAC_KEY: <base64 của openssl rand -hex 32 output>
   ```
3. Mount vào pod env qua `envFrom: - secretRef: name: centralized-data-service-secrets`.

### 5.2 Deploy
- Rolling restart `centralized-data-service` (không cần thay schema, không cần migration).
- Smoke test: trigger 1 event sync MongoDB → check shadow PG `_raw_data` của field `phone/email/card`: phải là chuỗi hex 64 char, KHÔNG phải `"***"`.

### 5.3 Rollback
- Option A (giữ image mới): `kubectl unset env deploy/centralized-data-service MASKING_HMAC_KEY` → fallback path tự revert về `"***"` (log warn 1 lần), behavior giống bản cũ, không crash.
- Option B (revert image): `kubectl rollout undo deploy/centralized-data-service`.

## 6. Out of scope (KHÔNG làm Phase 1 — đã document Phase 2 backlog)

❌ Migration thêm column `mask_strategy`/`mask_options`
❌ API CRUD mask-config
❌ UI tab Sensitive Masking
❌ Audit log table + writer
❌ Backfill script PII đã lưu `"***"`
❌ Strategy enum 4 mode (NONE/DROP/HMAC/PARTIAL)
❌ Per-field config qua DB
❌ Key rotation versioning
❌ Erasure rights (Điều 16)
❌ Rule cache + pub-sub invalidation
❌ Refactor signature 22 caller

→ Reference: `02_plan.md` + `03_implementation_phase_p0/p1/p2.md` + `11_risk_register.md` + `12_rollout_runbook.md` + `13_caller_inventory.md` đã mark Phase 2 backlog.

## 7. Compliance Phase 1 (achievement)

- ✅ Luật 91/2025 Điều 13 (Accuracy): hash giữ tính đối soát (cùng plaintext → cùng hash) thay vì destroy data thành `"***"`.
- ✅ NĐ 356/2025 "biện pháp kỹ thuật": HMAC-SHA256 với secret key (256-bit) đạt mức tối thiểu cho hash field nhạy cảm.
- ⚠️ Chưa có audit trail riêng → Phase 2 nếu thanh tra yêu cầu.
- ⚠️ Chưa có per-field strategy → chấp nhận: 1 hàm hash đồng nhất cho mọi field nhạy cảm.
- ⚠️ Chưa có erasure rights (Luật 91/2025 Điều 16) → Phase 2.

## 8. Notes cho ops

1. **Key management**: `MASKING_HMAC_KEY` phải ≥ 32 bytes. Lưu trong K8s Secret hoặc Vault. KHÔNG commit vào repo. KHÔNG share giữa môi trường dev/staging/prod.
2. **Determinism**: cùng plaintext + cùng key → cùng hash. Đây là FEATURE (cho phép join/dedup theo hashed value), KHÔNG phải bug.
3. **Key rotation**: nếu rotate key → hash cũ trong DB không match hash mới. Phase 1 chưa support key version → defer Phase 2.
4. **Log warn fallback**: nếu thấy `"MASKING_HMAC_KEY not set, fallback to *** literal"` trong log → ops phải set env ngay. Hệ thống vẫn chạy được nhưng đang giữ behavior cũ.

## 9. Skill đã dùng

- Read/Edit/Write (Claude Code)
- Go test runner (`go test`, `go vet`, `go build`)
- Pattern: HMAC-SHA256 (crypto/hmac + crypto/sha256)
- §6 Simplicity First (CLAUDE.md): minimal impact — 1 file logic + 1 file test, không touch schema/API/UI/caller signature
- §11 Memory Protection: APPEND-only `05_progress.md`
- §12 Brain Code Prohibition: Muscle thực thi đúng phạm vi plan đã approve
- §14 Pre-flight check: verify build + test + grep trước khi report done
