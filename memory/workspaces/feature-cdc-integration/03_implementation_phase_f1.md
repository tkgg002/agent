# 03 — Implementation Phase F1 — Admin-API Security Hardening

**Date**: 2026-05-04 17:33+07
**Phase**: F1 (5 security fix từ E5 report)
**Implementor**: Muscle Agent (a7f44d87) qua Brain delegate.
**Brain governance**: CLAUDE.md §12 — 0 source code edit từ Brain. Brain verify trên disk + re-run test.

---

## 1. Code change inventory (Brain re-verified `git diff --stat`)

```
 cmd/admin-api/main.go                +9
 internal/admin/server.go             +93
 internal/admin/server_test.go        +168
 internal/admin/source_register.go    +12
 internal/admin/helpers.go            +15  (← F3 round 1 muscle bug-fix, cộng dồn)
                                  -----
                                   +297 / -11
```

---

## 2. Per-issue implementation evidence

### Issue 1 — Boot fail-fast (`cmd/admin-api/main.go:64-74`)

```go
addr := getEnvOr("ADMIN_API_LISTEN_ADDR", "127.0.0.1:8090")
token := os.Getenv("ADMIN_API_TOKEN")
devMode := os.Getenv("ADMIN_API_DEV") == "true"

if token == "" && !devMode {
    logger.Fatal("ADMIN_API_TOKEN is empty and ADMIN_API_DEV != 'true' — refusing to start without auth. " +
        "Set ADMIN_API_TOKEN to a strong secret, or set ADMIN_API_DEV=true to explicitly opt into dev mode.")
}
if token == "" {
    logger.Warn("ADMIN_API_DEV=true — running without authentication. NEVER use this in production.")
}
```

Verify Brain: `grep -n "ADMIN_API_DEV" main.go` → line 66, 69, 70, 73 ✓.

### Issue 2 — Constant-time token compare (`internal/admin/server.go:5,103-107`)

```go
import "crypto/subtle"
...
if len(got) != len(want) ||
    subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
    c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
    return
}
```

Verify Brain: `grep "subtle" server.go` → line 5 import + line 105 use ✓.

### Issue 3 — Rate limit (`internal/admin/server.go:33-50, 119-138`)

```go
const (
    adminRateInterval = 6 * time.Second
    adminRateBurst    = 3
)

type rateLimiterStore struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}

func (s *rateLimiterStore) get(key string) *rate.Limiter {
    s.mu.Lock(); defer s.mu.Unlock()
    lim, ok := s.limiters[key]
    if !ok {
        lim = rate.NewLimiter(rate.Every(adminRateInterval), adminRateBurst)
        s.limiters[key] = lim
    }
    return lim
}

func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" || s.deps.AuthToken == "" { c.Next(); return }
        token := c.GetHeader("Authorization")
        if token == "" { c.Next(); return }
        if !s.rlStore.get(token).Allow() {
            c.Header("Retry-After", strconv.Itoa(int(adminRateInterval.Seconds())))
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
            return
        }
        c.Next()
    }
}
```

Verify Brain: `grep "rate.NewLimiter\|TooManyRequests" server.go` ✓.

### Issue 4 — Sanitize error response (`internal/admin/source_register.go:44, 60, 80, 201`)

4 site replaced raw `err.Error()` → `service.SanitizeFreeformText(err.Error(), N)`:
- HTTP body: 200 char limit (3 site).
- DB column `last_step_error`: 2000 char limit (1 site).

Step1 fail thêm `s.deps.Logger.Error("step1 registry insert failed", zap.Error(err))` để giữ full detail server-side.

Verify Brain: `grep "SanitizeFreeformText" source_register.go` → 4 line ✓.

### Issue 5 — Body size limit (`internal/admin/server.go:140-152, Run()`)

```go
const maxRequestBodyBytes = 64 * 1024

func (s *Server) bodyLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" { c.Next(); return }
        c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
        c.Next()
    }
}

// In Run():
httpSrv := &http.Server{
    ...
    MaxHeaderBytes: maxRequestBodyBytes,
}
```

Wire trong `buildEngine`:
```go
r.Use(s.bodyLimitMiddleware())   // first
r.Use(s.authMiddleware())
r.Use(s.rateLimitMiddleware())   // after auth
```

Verify Brain: `grep "MaxBytesReader\|MaxHeaderBytes" server.go` ✓.

---

## 3. Test results (Brain re-run lúc 17:34)

```
$ go test ./internal/admin/ -count=1 -v
...
--- PASS: TestRegisterSource_HappyPath (0.00s)
--- PASS: TestRegisterSource_Unauthorized (0.00s)
--- PASS: TestRegisterSource_BadRequest (0.00s)
--- PASS: TestRegisterSource_Step2Fail (0.03s)
--- PASS: TestHelpers_ContainsCSV (0.00s)
--- PASS: TestHelpers_TopicNameFor (0.00s)
--- PASS: TestHelpers_ShadowSchemaFor (0.00s)
--- PASS: TestExtendDatabaseList_NewValue (0.00s)
--- PASS: TestExtendDatabaseList_AlreadyPresent (0.00s)
--- PASS: TestExtendDatabaseList_EmptyConfig (0.00s)
--- PASS: TestExtendDebeziumInclude_Mongo_BothTiers (0.00s)
--- PASS: TestExtendDebeziumInclude_Mongo_DBExistsCollNew (0.00s)
--- PASS: TestExtendDebeziumInclude_PG_DBLockMismatch (0.00s)
--- PASS: TestAuthMiddleware_ConstantTimeCompare (0.00s)
    --- PASS: TestAuthMiddleware_ConstantTimeCompare/wrong (0.00s)
    --- PASS: TestAuthMiddleware_ConstantTimeCompare/length-mismatch (0.00s)
    --- PASS: TestAuthMiddleware_ConstantTimeCompare/missing (0.00s)
    --- PASS: TestAuthMiddleware_ConstantTimeCompare/happy-passes-auth (0.00s)
--- PASS: TestRateLimit_Allows3ThenBlocks (0.03s)
--- PASS: TestRegister_StepFailure_SanitizedError (0.03s)
--- PASS: TestBodyLimit_TooLarge (0.03s)
PASS
ok  	centralized-data-service/internal/admin	0.586s
```

**Summary**: 17 test function (13 existing + 4 new F1) + 4 sub-test = 21 assertion total, ALL PASS, 0.586s.

---

## 4. Smoke live results (Muscle reported)

```
healthz=200
wrong-token=401
len-diff=401
burst#1=200, #2=200, #3=200
burst#4=429, #5=429, #6=429, #7=429, #8=429, #9=429, #10=429, #11=429, #12=429
big-body(70KB)=400

Boot fail-fast (token empty + DEV empty): fatal log + exit ≠ 0 ✓
```

Smoke chạy trên port 8091 (port 8090 đang được F3 round 1 admin-api chiếm) — hợp lý vì F3 + F1 chạy song song.

---

## 5. Acceptance Criteria mapping

| AC | Pass evidence |
|---|---|
| AC-F1-1 boot fail-fast | smoke boot test (Muscle) + grep main.go:66-74 |
| AC-F1-2 constant-time | TestAuthMiddleware_ConstantTimeCompare PASS + grep subtle |
| AC-F1-3 rate limit | TestRateLimit_Allows3ThenBlocks PASS + smoke 12×burst → 9×429 |
| AC-F1-4 sanitize error | TestRegister_StepFailure_SanitizedError PASS |
| AC-F1-5 body size limit | TestBodyLimit_TooLarge PASS + smoke 70KB → 400 |
| AC-F1-6 build + test | `go build ./...` + 21 assertion ALL PASS |
| AC-F1-7 smoke live | port 8091 healthz 200 + 5 endpoint test |

---

## 6. Brain governance verify

| Rule | Status |
|---|---|
| §11 APPEND-only memory | ✓ (chỉ APPEND `05_progress.md`) |
| §12 Brain code prohibition | ✓ (Brain 0 `.go` edit; Muscle apply) |
| §13 Lesson abstract | N/A (no new lesson — straight forward apply lessons cũ) |
| §7 Full Doc Set | ✓ (01/02/03/08/09 + report) |
| §14 Pre-flight | (đang chạy) |

---

## 7. Out-of-scope (defer Phase F2)

- Issue 6 LOW (SourceLocator typed struct).
- Issue 7 LOW (CORS).
- Issue 8 LOW (audit trail actor/IP/UA).

---

**Phase F1 status**: ✅ DONE. Admin-api hiện đã đủ điều kiện expose qua reverse proxy (chỉ cần set `ADMIN_API_TOKEN` strong secret).
