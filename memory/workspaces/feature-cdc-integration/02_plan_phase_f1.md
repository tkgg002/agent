# 02 — Plan Phase F1 — Admin-API Security Hardening

**Date**: 2026-05-04
**Phase**: F1
**Strategy**: Fix-the-perimeter-first — auth path trước (1+2), throttle (3), data leak (4), resource limit (5).

---

## Strategic ordering rationale

Theo `agent/workflows/security-agent.md` "perimeter inward" + L-three-layer-trust:

1. **Issue 1 (boot fail-fast)** — Highest blast radius, **chặn deploy production miss-config**. Fix đầu để tránh false-confidence trong khi test.
2. **Issue 2 (constant-time compare)** — Auth path. Fix trước rate-limit vì nếu auth bypass đc, rate-limit vô nghĩa.
3. **Issue 3 (rate limit)** — Throttle layer phải nằm SAU auth (chỉ rate limit authenticated request, không leak quota cho attacker chưa auth).
4. **Issue 4 (sanitize error)** — Data leakage. Code-local change ở 3 call site `c.JSON(...err.Error()...)`.
5. **Issue 5 (body size limit)** — Resource exhaustion. Middleware-level, áp dụng SAU auth nhưng TRƯỚC handler.

**Ordering effect** request lifecycle:
```
HTTP Request
  ↓
[/healthz bypass]
  ↓
[Body size limit  — Issue 5]   ← Fail fast cho oversized body, save downstream cost
  ↓
[Auth — Issue 2 constant-time] ← Reject 401 cho bad token
  ↓
[Rate limit per-token — Issue 3]  ← Chỉ rate limit authenticated
  ↓
[Handler — Issue 4 sanitize on error path]
```

**Boot lifecycle**:
```
main()
  ↓
[Issue 1: token + DEV check] → exit non-zero nếu fail
  ↓
NewServer + Run
```

---

## Per-issue plan

### Fix 1 — Boot fail-fast (FR-F1-1)

**File**: `cmd/admin-api/main.go`

**Approach**:
- Sau line 65 (`token := os.Getenv("ADMIN_API_TOKEN")`):
  - Read `ADMIN_API_DEV` env.
  - If token empty AND dev != "true" → log fatal + return non-zero.
  - If token empty AND dev = "true" → log WARN bold (giữ behavior cũ).
  - If token set → log INFO normal.

**Verify**:
- Unit test: spawn subprocess `go run ./cmd/admin-api/` với env empty → expect exit ≠ 0 (skip nếu phức tạp).
- Manual smoke: `unset ADMIN_API_TOKEN ADMIN_API_DEV; ./cdc-admin-api` → fatal.

**Diff size**: ~10 line.

---

### Fix 2 — Constant-time token compare (FR-F1-2)

**File**: `internal/admin/server.go`

**Approach**:
- Import `crypto/subtle`.
- Trong `authMiddleware()` line 64:
  - So sánh length trước (constant-time vẫn cần length match để work).
  - `subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1` → 401.

**Verify**:
- Unit test mới: `TestAuthMiddleware_ConstantTimeCompare` — happy path 200, wrong token 401, missing token 401.
- Smoke: existing tests PASS.

**Diff size**: ~5 line + import.

---

### Fix 3 — Rate limit per-token (FR-F1-3)

**File**: `internal/admin/server.go` (mới middleware) + có thể thêm helper.

**Approach**:
- Import `golang.org/x/time/rate` + `sync`.
- Tạo `rateLimiterStore` struct: `map[string]*rate.Limiter` + `sync.Mutex`.
  - Key = token (hash hoặc raw — vì single-instance, raw OK; sau này abstract).
  - Get-or-create limiter với `rate.NewLimiter(rate.Every(6*time.Second), 3)` → 10 req/min với burst 3.
- Middleware `rateLimitMiddleware`:
  - Skip `/healthz`.
  - Skip nếu auth disabled (dev mode).
  - Lookup limiter cho token.
  - `limiter.Allow()` false → set `Retry-After: 6` header + 429.
- Wire vào `buildEngine` SAU `authMiddleware`.

**Edge case**:
- Token rotation → store giữ stale limiter forever. Mitigation: TTL eviction trong scope F2 (acceptable cho single-instance + low-token-count environment).
- Memory bound: max ~100 token = ~100×rate.Limiter × ~64B = ~6 KiB. Acceptable.

**Verify**:
- Unit test mới: `TestRateLimit_Allows3ThenBlocks` — gọi 4 lần liên tiếp với cùng token, lần 4 trả 429.
- Smoke live: 11 POST trong 10s → ≥1 × 429.

**Diff size**: ~50 line.

---

### Fix 4 — Sanitize error response (FR-F1-4)

**File**: `internal/admin/source_register.go`

**Approach**:
- Import `centralized-data-service/internal/service` (cho `SanitizeFreeformText`).
- 3 call site (line 40, 56, 76):
  - Replace `err.Error()` → `service.SanitizeFreeformText(err.Error(), 200)` cho response body.
  - Add `s.deps.Logger.Error("step1/2/3 failure", zap.Error(err))` để giữ full detail server-side.
- 2 site `LastStepError: err.Error()` → cũng sanitize.

**Verify**:
- Unit test mới: `TestRegister_StepFailure_SanitizedError` — inject sqlmock error chứa "table.column" syntax → response body KHÔNG chứa "table.column".
- Logger called với raw err (verify qua mock logger — observer từ zap/zaptest).

**Diff size**: ~15 line + import.

---

### Fix 5 — Body size limit + header size (FR-F1-5)

**File**: `internal/admin/server.go`

**Approach**:
- Hằng số `maxRequestBodyBytes = 64 * 1024`.
- Trong `Run()`:
  - `httpSrv.MaxHeaderBytes = 64 * 1024`.
- Middleware `bodyLimitMiddleware`:
  - `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)`.
- Wire vào `buildEngine` TRƯỚC `authMiddleware` (skip `/healthz` không cần body).

**Verify**:
- Unit test mới: `TestBodyLimit_TooLarge` — POST với body 70 KiB → 413 (gin default behavior khi MaxBytesReader trip).
- Smoke: existing 1KB POST PASS.

**Diff size**: ~10 line.

---

## File modification matrix

| File | Issue | Δ |
|---|---|---|
| `cmd/admin-api/main.go` | 1 | +~10 line |
| `internal/admin/server.go` | 2,3,5 | +~70 line, +2 import |
| `internal/admin/source_register.go` | 4 | +~15 line, +1 import |
| `internal/admin/server_test.go` | 1-5 | +~80 line, +5 test |

Total diff: ~175 line code + tests.

---

## Verification ordering

1. `go build ./...` after each fix landed.
2. `go test ./internal/admin/ -count=1 -run TestExtendDatabaseList|TestExtendDebeziumInclude|TestAuth|TestRateLimit|TestBodyLimit|TestRegister` PASS.
3. Boot smoke: `unset ADMIN_API_TOKEN ADMIN_API_DEV; go run ./cmd/admin-api/...` → fatal.
4. Boot smoke OK path: `ADMIN_API_TOKEN=test123 ./cdc-admin-api` → start.
5. Live HTTP smoke after restart:
   - `curl /healthz` → 200.
   - `curl -H "Authorization: Bearer test123" POST /v2/sources/register {valid}` → 200.
   - `curl -H "Authorization: Bearer wrong" POST ...` → 401.
   - 11× burst → ≥1 × 429.
   - `curl --data @70kb.json POST ...` → 413.

---

## Rollback

- `git revert <commit>` trong `centralized-data-service/` + restart admin-api binary.
- Memory file: APPEND-only nên không rollback; chỉ record outcome.
