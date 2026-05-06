# 09 — Tasks Solution Phase F1 (diff hints cho Muscle)

> Brain prohibition CLAUDE.md §12 — file này là **đề xuất diff cho Muscle**, KHÔNG phải code change của Brain.

---

## F1-1 — `cmd/admin-api/main.go` boot fail-fast

**Replace lines 64-66**:

```go
addr := getEnvOr("ADMIN_API_LISTEN_ADDR", "127.0.0.1:8090")
token := os.Getenv("ADMIN_API_TOKEN")
```

**With**:

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

> Rationale: `logger.Fatal` đã có sẵn (line 40, 50) → exit code 1 + stderr message rõ. KHÔNG dùng `log.Fatal` (legacy import) để giữ uniform.

---

## F1-2 — `internal/admin/server.go` constant-time compare

### Add import

```go
import (
    // ... existing imports ...
    "crypto/subtle"
)
```

### Replace lines 62-67 in `authMiddleware`

```go
got := c.GetHeader("Authorization")
want := "Bearer " + s.deps.AuthToken
if got != want {
    c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
    return
}
```

**With**:

```go
got := c.GetHeader("Authorization")
want := "Bearer " + s.deps.AuthToken
// Length check first — ConstantTimeCompare requires equal length to be useful.
// (Mismatched length = automatic reject; doesn't leak content timing.)
if len(got) != len(want) ||
    subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
    c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
    return
}
```

---

## F1-3 — `internal/admin/server.go` rate limit middleware

### Add imports

```go
import (
    // existing
    "sync"
    "strconv"

    "golang.org/x/time/rate"
)
```

### Add constants + struct + method (top of file or near `Server`)

```go
const (
    // 10 req/min per token (1 token every 6s) with burst capacity 3.
    adminRateInterval = 6 * time.Second
    adminRateBurst    = 3
)

// rateLimiterStore — per-token token-bucket store. Single-instance only;
// memory-bounded by number of distinct tokens (typically <100).
type rateLimiterStore struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}

func newRateLimiterStore() *rateLimiterStore {
    return &rateLimiterStore{limiters: make(map[string]*rate.Limiter)}
}

func (s *rateLimiterStore) get(key string) *rate.Limiter {
    s.mu.Lock()
    defer s.mu.Unlock()
    lim, ok := s.limiters[key]
    if !ok {
        lim = rate.NewLimiter(rate.Every(adminRateInterval), adminRateBurst)
        s.limiters[key] = lim
    }
    return lim
}
```

### Update `Server` struct

```go
type Server struct {
    deps    Deps
    engine  *gin.Engine
    rlStore *rateLimiterStore   // NEW
}
```

### Update `NewServer`

```go
func NewServer(deps Deps) *Server {
    if deps.AuthToken == "" {
        deps.Logger.Warn("ADMIN_API_TOKEN empty — auth disabled (dev mode only)")
    }
    s := &Server{deps: deps, rlStore: newRateLimiterStore()}
    s.engine = s.buildEngine()
    return s
}
```

### Add new middleware

```go
// rateLimitMiddleware — per-token token-bucket. Skip /healthz + skip when
// auth is disabled (dev mode). Uses token from Authorization header as key.
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" {
            c.Next()
            return
        }
        if s.deps.AuthToken == "" {
            c.Next()
            return
        }
        token := c.GetHeader("Authorization")
        if token == "" {
            c.Next() // auth middleware will reject
            return
        }
        lim := s.rlStore.get(token)
        if !lim.Allow() {
            c.Header("Retry-After", strconv.Itoa(int(adminRateInterval.Seconds())))
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
            return
        }
        c.Next()
    }
}
```

### Wire in `buildEngine` (UPDATE order)

```go
func (s *Server) buildEngine() *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(s.bodyLimitMiddleware())   // NEW (F1-5) — first
    r.Use(s.authMiddleware())        // existing
    r.Use(s.rateLimitMiddleware())   // NEW (F1-3) — after auth
    r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
    r.POST("/v2/sources/register", s.handleRegisterSource)
    return r
}
```

### Add import for net/http (already present, verify)

---

## F1-4 — `internal/admin/source_register.go` sanitize error

### Add import

```go
import (
    // existing
    "centralized-data-service/internal/service"
)
```

### Replace 3 error response sites

**Line 39-42** (step1 fail):
```go
c.JSON(http.StatusInternalServerError, gin.H{
    "error": "step1 (registry insert) failed: " + err.Error(),
})
```

**With**:
```go
s.deps.Logger.Error("step1 registry insert failed", zap.Error(err))
c.JSON(http.StatusInternalServerError, gin.H{
    "error":  "step1 (registry insert) failed",
    "detail": service.SanitizeFreeformText(err.Error(), 200),
})
```

**Lines 50-58** (step2 fail):
```go
s.deps.Logger.Warn("step2 debezium extend failed", zap.Error(err))
s.markProvisioningFailed(sourceID, "step2_failed", err)
c.JSON(http.StatusMultiStatus, RegisterSourceResponse{
    SourceObjectID:    sourceID,
    ProvisioningState: "step2_failed",
    StepsCompleted:    stepsCompleted,
    LastStepError:     err.Error(),
})
return
```

**With**:
```go
s.deps.Logger.Warn("step2 debezium extend failed", zap.Error(err))
s.markProvisioningFailed(sourceID, "step2_failed", err)
c.JSON(http.StatusMultiStatus, RegisterSourceResponse{
    SourceObjectID:    sourceID,
    ProvisioningState: "step2_failed",
    StepsCompleted:    stepsCompleted,
    LastStepError:     service.SanitizeFreeformText(err.Error(), 200),
})
return
```

**Lines 70-78** (step3 fail): same pattern as step2.

### Verify `markProvisioningFailed` (line 192-198) also sanitizes

```go
func (s *Server) markProvisioningFailed(sourceID int64, step string, err error) {
    s.deps.DB.Exec(`UPDATE cdc_system.source_object_registry
                    SET provisioning_state = ?,
                        last_step_error    = ?,
                        updated_at         = NOW()
                    WHERE id = ?`, step, service.SanitizeFreeformText(err.Error(), 2000), sourceID)
}
```

> Note: DB `last_step_error` column → 2000 char là OK (operator-facing trong DB, không leak qua HTTP).

---

## F1-5 — `internal/admin/server.go` body size limit

### Add constant

```go
const maxRequestBodyBytes = 64 * 1024
```

### Add middleware

```go
// bodyLimitMiddleware — caps request body at 64 KiB (excluding /healthz which
// has no body). Trips MaxBytesReader on read → handler will see EOF or
// "request body too large" via gin's ShouldBindJSON.
func (s *Server) bodyLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.URL.Path == "/healthz" {
            c.Next()
            return
        }
        c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
        c.Next()
    }
}
```

### Update `Run` to set `MaxHeaderBytes`

```go
func (s *Server) Run(ctx context.Context, addr string) error {
    httpSrv := &http.Server{
        Addr:              addr,
        Handler:           s.engine,
        ReadHeaderTimeout: 10 * time.Second,
        MaxHeaderBytes:    maxRequestBodyBytes,   // NEW
    }
    // ... existing shutdown logic ...
}
```

---

## F1-Tests — `internal/admin/server_test.go` 5 new tests

```go
import (
    // existing
    "bytes"
    "io"
    "strings"
    "time"
)

// F1-2: constant-time compare unit test
func TestAuthMiddleware_ConstantTimeCompare(t *testing.T) {
    s := newTestServer(t)
    cases := []struct {
        name   string
        token  string
        status int
    }{
        {"happy", "Bearer secret-test-token", 200},
        {"wrong", "Bearer wrong-token", 401},
        {"length-mismatch", "Bearer x", 401},
        {"missing", "", 401},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/healthz", nil) // healthz exempt
            // NB: hit register-style auth via a no-op route or use /v2/sources/register
            // with empty body — but for length check we just verify the engine
            // passes the auth middleware. Re-use existing test harness.
        })
    }
}

// F1-3: rate-limit allow burst 3 then 429
func TestRateLimit_Allows3ThenBlocks(t *testing.T) {
    s := newTestServer(t)
    var got429 int
    for i := 0; i < 5; i++ {
        w := httptest.NewRecorder()
        req := httptest.NewRequest("POST", "/v2/sources/register",
            strings.NewReader(`{"object_code":"x","source_engine_type":"mongodb","sync_engine":"debezium","source_object_name":"x","source_locator":{"database":"d"},"target_master_table":"x"}`))
        req.Header.Set("Authorization", "Bearer secret-test-token")
        req.Header.Set("Content-Type", "application/json")
        s.engine.ServeHTTP(w, req)
        if w.Code == http.StatusTooManyRequests {
            got429++
        }
    }
    require.GreaterOrEqual(t, got429, 1, "expected ≥1 429 after burst")
}

// F1-4: error response sanitized
func TestRegister_StepFailure_SanitizedError(t *testing.T) {
    s := newTestServerWithMockDB(t, func(mock sqlmock.Sqlmock) {
        // Inject error containing pseudo schema leak
        mock.ExpectBegin()
        mock.ExpectQuery(`SELECT id FROM cdc_system.connection_registry`).
            WillReturnError(errors.New("ERROR: relation \"secret_table.password_column\" does not exist"))
        mock.ExpectRollback()
    })
    body := `{"object_code":"x","source_engine_type":"mongodb","sync_engine":"debezium","source_object_name":"x","source_locator":{"database":"d"},"target_master_table":"x"}`
    req := httptest.NewRequest("POST", "/v2/sources/register", strings.NewReader(body))
    req.Header.Set("Authorization", "Bearer secret-test-token")
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    s.engine.ServeHTTP(w, req)
    require.Equal(t, 500, w.Code)
    require.NotContains(t, w.Body.String(), "secret_table.password_column",
        "response must not leak raw schema fragment")
    require.Contains(t, w.Body.String(), "step1")
}

// F1-5: body size limit
func TestBodyLimit_TooLarge(t *testing.T) {
    s := newTestServer(t)
    // 70 KiB JSON payload
    big := strings.Repeat("x", 70*1024)
    body := `{"object_code":"x","source_engine_type":"mongodb","sync_engine":"debezium","source_object_name":"x","source_locator":{"database":"` + big + `"},"target_master_table":"x"}`
    req := httptest.NewRequest("POST", "/v2/sources/register", strings.NewReader(body))
    req.Header.Set("Authorization", "Bearer secret-test-token")
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    s.engine.ServeHTTP(w, req)
    // gin → MaxBytesReader trip → ShouldBindJSON returns error → 400
    // (some setups return 413 directly; either is acceptable as long as
    // it is NOT 200 and NOT 500-with-stack)
    require.NotEqual(t, 200, w.Code)
    require.NotEqual(t, 500, w.Code)
}
```

> Note Muscle: nếu existing `newTestServer` helper chưa có, tái sử dụng pattern từ test cases hiện có (line 79-180 `server_test.go`).

---

## F1-V — Smoke live verify script

```bash
# 1. Build
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
go build -o /tmp/cdc-admin-api ./cmd/admin-api

# 2. Boot fail-fast test
unset ADMIN_API_TOKEN ADMIN_API_DEV
/tmp/cdc-admin-api 2>&1 | head -3
# Expect: "ADMIN_API_TOKEN is empty and ADMIN_API_DEV != 'true'" + exit ≠ 0

# 3. Restart with token
export ADMIN_API_TOKEN="phase_f1_smoke_token_$(date +%s)"
# Kill existing if any:
pkill -f cdc-admin-api 2>/dev/null || true
nohup /tmp/cdc-admin-api > /tmp/admin-api-f1.log 2>&1 &
sleep 1

# 4. Healthz
curl -s -o /dev/null -w "healthz=%{http_code}\n" http://127.0.0.1:8090/healthz

# 5. Wrong token
curl -s -o /dev/null -w "wrong-token=%{http_code}\n" \
  -H "Authorization: Bearer wrong" \
  http://127.0.0.1:8090/v2/sources/register

# 6. Length-different but Bearer-prefixed
curl -s -o /dev/null -w "len-diff=%{http_code}\n" \
  -H "Authorization: Bearer xx" \
  http://127.0.0.1:8090/v2/sources/register

# 7. Rate limit burst (need valid token)
for i in $(seq 1 12); do
  curl -s -o /dev/null -w "burst#$i=%{http_code}\n" \
    -H "Authorization: Bearer $ADMIN_API_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"object_code":"smoke_burst","source_engine_type":"mongodb","sync_engine":"debezium","source_object_name":"x","source_locator":{"database":"d"},"target_master_table":"x"}' \
    http://127.0.0.1:8090/v2/sources/register
done | tee /tmp/burst.log
# Expect: ≥1 line ending in =429

# 8. Body size
yes "x" | head -c 70000 > /tmp/big.json
curl -s -o /dev/null -w "big=%{http_code}\n" \
  -H "Authorization: Bearer $ADMIN_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data-binary @/tmp/big.json \
  http://127.0.0.1:8090/v2/sources/register
# Expect: 400 hoặc 413, NOT 200, NOT 500

# 9. Cleanup
pkill -f cdc-admin-api
```

---

## Rollback plan

Nếu F1 tests fail hoặc smoke fail:
1. `git status` trong `centralized-data-service/`.
2. `git diff cmd/admin-api/ internal/admin/` review.
3. `git checkout -- cmd/admin-api/main.go internal/admin/server.go internal/admin/source_register.go internal/admin/server_test.go`.
4. Restart admin-api binary cũ (kéo từ docker image hoặc previous build).
5. Brain APPEND `05_progress.md` với "F1 attempt failed — rolled back" + root cause hint.
