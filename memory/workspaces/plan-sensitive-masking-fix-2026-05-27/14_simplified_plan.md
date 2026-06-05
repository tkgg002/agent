# 14_simplified_plan — Replace `"***"` bằng hash

> Created: 2026-06-01 (sau feedback user).
> **SUPERSEDES**: Plan phức tạp ở `02_plan.md` + `03_implementation_phase_p0/p1/p2.md`. Các file đó move sang **Phase 2 backlog** (không xoá vì §11).
> Target user: "Func chỉ quét field name → có trong list rủi ro → replace `"***"` → giờ đổi thành chạy hash."

## Scope (3 tasks)

| # | Task | Effort | File thay đổi |
|---|---|---|---|
| **S-1** | Thêm env `MASKING_HMAC_KEY` + helper `hashValue()` | 1h | `data-hub/centralized-data-service/internal/service/masking_service.go` (+ ~15 dòng) |
| **S-2** | Thay 5 chỗ `"***"` literal bằng `ms.hashValue(value)` | 1.5h | Cùng file — lines 71, 77, 91, 133, 153 |
| **S-3** | Unit test deterministic + non-leak + length | 1h | `internal/service/masking_service_test.go` (NEW hoặc append) |

**Tổng: 3.5h Muscle**.

## Implementation chi tiết

### S-1 — Helper `hashValue()`

```go
// File: internal/service/masking_service.go (append section)

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "os"
)

var maskingHMACKey = []byte(os.Getenv("MASKING_HMAC_KEY"))

// hashValue trả về HMAC-SHA256 hex của value bất kỳ.
// Dùng thay cho literal "***" để compliance Luật 91/2025 (giữ tính đối soát + audit accuracy).
// nil/empty → trả "" (không leak structure).
func (ms *MaskingService) hashValue(v interface{}) string {
    if v == nil {
        return ""
    }
    s := fmt.Sprintf("%v", v)
    if s == "" {
        return ""
    }
    if len(maskingHMACKey) == 0 {
        // Fallback: nếu chưa config key, giữ behavior cũ "***" để không crash production.
        // Ops phải set MASKING_HMAC_KEY trước deploy — log warn 1 lần.
        ms.logger.Warn("MASKING_HMAC_KEY not set, fallback to ***")
        return "***"
    }
    mac := hmac.New(sha256.New, maskingHMACKey)
    mac.Write([]byte(s))
    return hex.EncodeToString(mac.Sum(nil))
}
```

### S-2 — Thay 5 chỗ `"***"` literal

```diff
// Line 71 (MaskJSONPayload — invalid JSON path)
-       wrapped, _ := json.Marshal(map[string]string{"raw": "***"})
+       wrapped, _ := json.Marshal(map[string]string{"raw": ms.hashValue(string(data))})

// Line 77 (MaskJSONPayload — fallback path)
-       wrapped, _ := json.Marshal(map[string]string{"raw": "***"})
+       wrapped, _ := json.Marshal(map[string]string{"raw": ms.hashValue(string(data))})

// Line 91 (MaskFieldSample)
-       return "***"
+       return ms.hashValue(value)

// Line 133 (maskMapRecursive)
-               out[key] = "***"
+               out[key] = ms.hashValue(value)

// Line 153 (maskAnyRecursive default branch)
-               return "***"
+               return ms.hashValue(value)
```

**Lưu ý**:
- Signature `MaskTableData`, `MaskJSONPayload`, `MaskFieldSample` **KHÔNG ĐỔI** → 22 caller không break.
- `maskMapRecursive` & `maskAnyRecursive` giữ nguyên cấu trúc recursive (vẫn xử lý nested object đúng).
- `shouldMaskField` (line 159-179) không cần đổi → vẫn match field name theo list nhạy cảm.

### S-3 — Unit test

```go
// File: internal/service/masking_service_test.go (append hoặc tạo mới)

func TestHashValue_Deterministic(t *testing.T) {
    os.Setenv("MASKING_HMAC_KEY", "test_key_at_least_32_bytes_long!!!")
    ms := &MaskingService{logger: zap.NewNop()}

    h1 := ms.hashValue("4111111111111111")
    h2 := ms.hashValue("4111111111111111")
    require.Equal(t, h1, h2, "cùng input phải ra cùng hash")
    require.Len(t, h1, 64, "SHA256 hex = 64 chars")
    require.NotEqual(t, "4111111111111111", h1, "không leak plaintext")
    require.NotEqual(t, "***", h1, "không còn literal *** path")
}

func TestHashValue_DifferentInput(t *testing.T) {
    os.Setenv("MASKING_HMAC_KEY", "test_key_at_least_32_bytes_long!!!")
    ms := &MaskingService{logger: zap.NewNop()}

    h1 := ms.hashValue("4111111111111111")
    h2 := ms.hashValue("5222222222222222")
    require.NotEqual(t, h1, h2, "input khác phải hash khác")
}

func TestHashValue_NilOrEmpty(t *testing.T) {
    os.Setenv("MASKING_HMAC_KEY", "test_key_at_least_32_bytes_long!!!")
    ms := &MaskingService{logger: zap.NewNop()}

    require.Equal(t, "", ms.hashValue(nil))
    require.Equal(t, "", ms.hashValue(""))
}

func TestMaskTableData_UsesHash(t *testing.T) {
    os.Setenv("MASKING_HMAC_KEY", "test_key_at_least_32_bytes_long!!!")
    // ... setup MaskingService với sensitive field "card_number" ...
    ms := newTestMaskingService(t, []string{"card_number"})

    out := ms.MaskTableData("transactions", map[string]any{
        "card_number": "4111111111111111",
        "amount":      1000,
    })
    require.Len(t, out["card_number"], 64, "card_number phải là hash 64 chars")
    require.NotEqual(t, "***", out["card_number"])
    require.Equal(t, 1000, out["amount"], "non-sensitive giữ nguyên")
}
```

### Verify

```bash
# 1. Build PASS
cd data-hub/centralized-data-service && go build ./internal/service/...

# 2. Test PASS
MASKING_HMAC_KEY="$(openssl rand -hex 32)" go test ./internal/service -run TestHash -v
MASKING_HMAC_KEY="$(openssl rand -hex 32)" go test ./internal/service -run TestMaskTableData -v

# 3. Không còn "***" trong DB path (chỉ còn warning fallback path)
grep -n '"\*\*\*"' internal/service/masking_service.go
# Expected: 1 match (fallback warn message line) — acceptable.
```

## Out of scope (KHÔNG làm Phase 1)

❌ Strategy enum 4 mode
❌ Per-field config qua `mapping_rule` column mới
❌ Migration SQL schema thay đổi
❌ API CRUD mask-config
❌ UI tab Sensitive Masking
❌ Audit log table + writer
❌ Backfill script
❌ Re-snapshot policy
❌ Erasure rights (Điều 16)
❌ Multi-strategy per field
❌ Rule cache + pub-sub invalidation
❌ Dual-method signature
❌ Caller refactor 22 site

→ Tất cả gom lại Phase 2 backlog (`02_plan.md` cũ + 11/12/13 file giữ làm reference, KHÔNG execute trừ khi user explicit yêu cầu).

## Deployment

### Pre-deploy
1. Generate key: `openssl rand -hex 32`
2. Set K8s Secret: `MASKING_HMAC_KEY=<above>` cho cluster đích.

### Deploy
1. Restart `centralized-data-service` pod (rolling).
2. Verify: 1 event sync → check shadow PG `_raw_data` của field nhạy cảm là string hex 64 chars, không phải `"***"`.

### Rollback
- Unset env `MASKING_HMAC_KEY` → fallback path tự revert về `"***"` (giữ behavior cũ, không crash).
- Hoặc rollback image.

## Compliance Phase 1

Phase này đạt:
- ✓ Bỏ literal `"***"` khỏi DB path (5 vị trí) → tuân thủ Luật 91/2025 Điều 13 (Accuracy: hash giữ tính đối soát thay vì destroy data).
- ✓ HMAC-SHA256 với secret key → NĐ 356 "biện pháp kỹ thuật" cơ bản.
- ⚠️ Chưa có audit log trail — Phase 2 nếu thanh tra yêu cầu.
- ⚠️ Chưa có per-field strategy — chấp nhận: 1 hàm hash cho mọi field nhạy cảm.

## File workspace status sau pivot

| File | Trạng thái |
|---|---|
| `00_context.md` | Giữ — context vẫn đúng |
| `01_requirements.md` | **MARK Phase 2** — FR/NFR đa số là Phase 2 |
| `02_plan.md` | **MARK Phase 2** — plan 3-phase ban đầu |
| `03_implementation_phase_p0/p1/p2.md` | **MARK Phase 2** |
| `04_decisions.md` (ADR-001..015) | Giữ làm reference Phase 2; Phase 1 chỉ apply ADR-001 (HMAC vs SHA) |
| `05_progress.md` | APPEND entry pivot |
| `06_validation.md` | **MARK Phase 2** |
| `07_status_report.md` | Update: Phase 1 = S-1..S-3 (file này) |
| `08_tasks_phase_*.md`, `09_tasks_solution_*.md` | **MARK Phase 2** |
| `10_gap_analysis.md` | Giữ làm reference |
| `11_risk_register.md` | **MARK Phase 2** — risk register dành cho plan phức tạp |
| `12_rollout_runbook.md` | **MARK Phase 2** |
| `13_caller_inventory.md` | **MARK Phase 2** (Phase 1 không refactor caller) |
| `14_simplified_plan.md` | **PHASE 1 ACTIVE** — file này |
| `report_*.md` | Update sau khi Muscle execute Phase 1 |

## Verb chờ User

- `execute s1` — Muscle thực thi S-1..S-3 (3.5h, không touch các Phase 2 task).
- `revise` — Sửa simplified plan.
