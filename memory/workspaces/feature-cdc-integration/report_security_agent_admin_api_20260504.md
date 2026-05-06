# Security Agent Report — `cdc-admin-api` (Phase E task E5)

**Date**: 2026-05-04 (17:05+07)
**Scope**:
- `cmd/admin-api/main.go`
- `internal/admin/server.go`
- `internal/admin/source_register.go`
- `internal/admin/helpers.go`
- `internal/admin/types.go`
**Trigger**: CLAUDE.md §8 mandatory security gate sau task hoàn thành (P0.2 + Phase E1).

---

## Scan Summary

| Category | Issues Found | Highest Severity |
|---|---|---|
| Input Validation | 1 | LOW |
| Secrets | 0 | - |
| Dependencies | (defer — Go module audit) | - |
| API Security | 5 | HIGH (×2) |
| Error/Output Leak | 1 | MEDIUM |
| Resilience/DoS | 2 | MEDIUM |

---

## Vulnerabilities Found

### 1. **[HIGH]** Dev-mode silent bypass khi token rỗng

**File**: `internal/admin/server.go:58-61`

```go
if s.deps.AuthToken == "" {
    // dev mode: không enforce auth
    c.Next()
    return
}
```

**Risk**: Nếu deploy production mà quên set ENV `ADMIN_API_TOKEN` → service silently trả no-auth → mọi request 200 OK, infra side-effects (Debezium config rewrite, NATS publish, registry insert) ai cũng làm được.

**Remediation**:
- Boot-time fail-fast: trong `cmd/admin-api/main.go`, nếu env `ADMIN_API_TOKEN` empty AND không có flag `--dev` → exit non-zero với error message rõ.
- Hoặc thay dev-mode bypass bằng explicit env `ADMIN_API_DEV=true` (require both).

```go
// cmd/admin-api/main.go
token := os.Getenv("ADMIN_API_TOKEN")
devMode := os.Getenv("ADMIN_API_DEV") == "true"
if token == "" && !devMode {
    return fmt.Errorf("ADMIN_API_TOKEN is required (or set ADMIN_API_DEV=true)")
}
```

---

### 2. **[HIGH]** Token compare không constant-time → timing attack

**File**: `internal/admin/server.go:64-67`

```go
got := c.GetHeader("Authorization")
want := "Bearer " + s.deps.AuthToken
if got != want {
    c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
}
```

**Risk**: Go string compare short-circuit ở byte đầu mismatch — attacker đo response time → infer token byte-by-byte. Local network attacker có thể brute-force token trong giờ ngắn.

**Remediation**: Dùng `crypto/subtle.ConstantTimeCompare`:

```go
import "crypto/subtle"
...
got := c.GetHeader("Authorization")
want := "Bearer " + s.deps.AuthToken
if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
    c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
    return
}
```

---

### 3. **[MEDIUM]** No rate limit → DoS via spam register

**File**: `internal/admin/server.go` middleware chain

**Risk**: POST `/v2/sources/register` trigger 5 step orchestration: registry INSERT (atomic), Debezium PUT (heavy — rewrite full config), Schema Registry PUT, NATS publish, registry UPDATE. Spammer có valid token (or local network access) gửi 1000 req/s → Debezium connector restart loop, schema registry pollution, NATS subject flood.

**Remediation**: Add per-token token-bucket rate limiter — `golang.org/x/time/rate` hoặc gin middleware như `github.com/ulule/limiter/v3`.

Suggested: 10 req/min per token cho `/v2/sources/register`, 100 req/min per token cho `/healthz`.

---

### 4. **[MEDIUM]** Error response leak raw internal detail

**File**: `internal/admin/source_register.go:40,56,76`

```go
c.JSON(..., gin.H{"error": "step1 (registry insert) failed: " + err.Error()})
...
LastStepError: err.Error()
```

**Risk**: `err.Error()` từ pgx/gorm có thể chứa table name / column name / SQL fragment / type system error → operator-friendly nhưng leak schema details cho attacker đã có valid token để tiếp cận lateral.

**Remediation**: Sanitize trước khi return — kiểm tra error type, return generic message + log internal:

```go
import "centralized-data-service/pkgs/utils"   // assumed có SanitizeFreeformText

c.JSON(http.StatusInternalServerError, gin.H{
    "error": "step1 failed",
    "detail": utils.SanitizeFreeformText(err.Error(), 200), // truncate + strip newlines
})
s.deps.Logger.Error("step1 failure", zap.Error(err))   // full detail in logs only
```

Phase E1 đã thấy `SanitizeFreeformText` được dùng ở Phase P0.2 — apply luôn ở đây.

---

### 5. **[MEDIUM]** No request body size limit → memory DoS

**File**: `internal/admin/server.go:74-80` http.Server config

```go
httpSrv := &http.Server{
    Addr:              addr,
    Handler:           s.engine,
    ReadHeaderTimeout: 10 * time.Second, // assume có
    // KHÔNG set MaxHeaderBytes, KHÔNG dùng http.MaxBytesReader
}
```

**Risk**: `SourceLocator map[string]interface{}` cho phép body lớn tùy ý. Attacker valid-token gửi 1GB JSON → memory bloat → OOM kill.

**Remediation**:
1. Set `httpSrv.MaxHeaderBytes = 64 * 1024`.
2. Thêm middleware giới hạn body:
```go
r.Use(func(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
    c.Next()
})
```

---

### 6. **[LOW]** `SourceLocator` schemaless → uncontrolled write to JSONB

**File**: `internal/admin/types.go:13`, `source_register.go:132`

```go
SourceLocator map[string]interface{} `json:"source_locator" binding:"required"`
...
locatorJSON, _ := json.Marshal(req.SourceLocator)
... source_locator_json = ?::jsonb
```

**Risk**: Attacker stuff arbitrary keys/large structures vào `source_locator_json` JSONB column. Không exec, không SQL injection (parameterized) — nhưng có thể abuse storage/index.

**Remediation**: Define explicit struct cho mỗi engine type:

```go
type MongoLocator struct {
    Database string `json:"database" binding:"required"`
}
type PGLocator struct {
    Schema string `json:"schema" binding:"required"`
}
// dispatch theo source_engine_type, validate per-engine
```

---

### 7. **[LOW]** Missing CORS policy

**File**: `internal/admin/server.go` (no CORS config)

**Risk**: Nếu sau này có cdc-portal frontend gọi từ browser → preflight fail. Nếu deploy đặt sau reverse proxy có CORS allow-all → CSRF-like attack từ browser victim qua subdomain.

**Remediation**: Add explicit CORS middleware với `AllowedOrigins` whitelist. Nếu chưa có frontend → defer; nhưng trong report ghi "to be added before frontend integration".

---

### 8. **[LOW]** Audit trail không capture actor/IP/user-agent

**File**: `internal/admin/source_register.go` không log per-request đến `cdc_system.cdc_activity_log`

**Risk**: Khi có incident, không trace được ai gọi endpoint từ đâu. `provisioning_step_log` chỉ ghi step name, không actor.

**Remediation**: Middleware audit-log mỗi POST request — INSERT vào `cdc_activity_log` với:
- `actor` (token name nếu có alias map)
- `client_ip` từ `c.ClientIP()`
- `user_agent` từ header
- `payload_hash` (SHA256 của body, KHÔNG full body — bảo vệ secrets nếu có)
- `status`, `duration_ms`

---

## API Security Checklist

- [x] Endpoints có auth middleware — Bearer token enforced (trừ /healthz)
- [ ] **Authorization checks** — chỉ có authentication, không phân role/scope. Nếu future cần multi-tenant → cần per-token scope policy.
- [ ] **Rate limiting** — KHÔNG có (issue #3)
- [ ] **CORS** — KHÔNG có (issue #7)
- [x] Response không leak password/secret cụ thể (chỉ leak schema fragment qua err — issue #4)
- [ ] **Error messages** không expose internal — fail (issue #4)

## Input Validation Checklist

- [x] Required fields enforced (gin binding)
- [x] Engine/sync_engine enum hạn chế (`oneof=postgresql mongodb mariadb mysql` / `oneof=debezium`)
- [x] Object type enum (`oneof=table collection view`)
- [ ] `source_locator` schemaless — không validate per-engine (issue #6)
- [x] SQL parameterized (`?` placeholder)

## Secrets Checklist

- [x] Token từ ENV `ADMIN_API_TOKEN` (không hardcode)
- [x] Connection credentials từ DB `cdc_system.connection_registry` (không hardcode trong source)
- [ ] Token validate strength tại boot — không có (8+ char, entropy check) — defer LOW

---

## Verdict

⚠️ **PASS WITH WARNINGS** — không có Critical, nhưng có **2 HIGH** cần fix trước khi expose admin-api ra ngoài `127.0.0.1`.

### Block-push reasoning
- Nếu admin-api **chỉ dùng nội bộ trên `127.0.0.1`** (như hiện tại) → 2 HIGH severity giảm xuống MEDIUM (timing attack chỉ có giá trị qua network).
- Nếu chuẩn bị expose qua reverse proxy / mở 0.0.0.0 → BLOCK PUSH cho đến khi fix #1 + #2.

### Recommended Phase F (security hardening)

| Issue | Severity | Phase |
|---|---|---|
| #1 dev-mode silent bypass | HIGH | F1 (mandatory before production) |
| #2 timing attack | HIGH | F1 |
| #3 rate limit | MEDIUM | F1 |
| #4 error sanitize | MEDIUM | F1 |
| #5 body size limit | MEDIUM | F1 |
| #6 SourceLocator typed | LOW | F2 |
| #7 CORS | LOW | F2 (chỉ khi có frontend) |
| #8 audit trail | LOW | F2 |

Phase F1 ETA ~2-3h Muscle work + 30min Brain plan + verify.

---

## Skills + Lessons applied

- Workflow `agent/workflows/security-agent.md` (CLAUDE.md §8)
- Lessons: L-three-layer-trust (3-layer scan: code → infra → contract), L-real-data-test (live curl auth bypass test, không tin code-only review)
- Tools: Bash (grep), Read (source files)

**File này được sinh tự động bởi Brain trong Phase E task E5 — KHÔNG phải code change. Remediation defer Phase F1.**
