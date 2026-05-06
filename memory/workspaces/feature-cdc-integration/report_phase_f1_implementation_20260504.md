# Report Phase F1 — Admin-API Security Hardening
**Date**: 2026-05-04  
**Author**: Muscle (Claude Sonnet 4.6)  
**Phase**: F1 — 5 security issue từ E5 report  

---

## Section 1: Code Change Summary

### File 1: `cmd/admin-api/main.go` (+10 lines)
- Thêm `devMode := os.Getenv("ADMIN_API_DEV") == "true"`
- Thêm fail-fast guard: `if token == "" && !devMode { logger.Fatal(...) }`
- Thêm dev warning: `if token == "" { logger.Warn(...) }`

### File 2: `internal/admin/server.go` (+73 lines)
- Thêm imports: `crypto/subtle`, `strconv`, `sync`, `golang.org/x/time/rate`
- Thêm constants: `adminRateInterval = 6*time.Second`, `adminRateBurst = 3`, `maxRequestBodyBytes = 64*1024`
- Thêm type `rateLimiterStore` + `newRateLimiterStore()` + `.get(key)` method
- Cập nhật `Server` struct: thêm `rlStore *rateLimiterStore`
- Cập nhật `NewServer`: init `rlStore: newRateLimiterStore()`
- Cập nhật `authMiddleware`: thay `got != want` bằng `crypto/subtle.ConstantTimeCompare`
- Cập nhật `buildEngine`: wire `bodyLimitMiddleware` → `authMiddleware` → `rateLimitMiddleware`
- Thêm `rateLimitMiddleware()`: per-token token-bucket, skip /healthz, Retry-After header
- Thêm `bodyLimitMiddleware()`: `http.MaxBytesReader(..., 64*1024)`, skip /healthz
- Cập nhật `Run`: thêm `MaxHeaderBytes: maxRequestBodyBytes`

### File 3: `internal/admin/source_register.go` (+10 lines)
- Thêm import `centralized-data-service/internal/service`
- Step1 error: thêm `s.deps.Logger.Error(...)`, response `{error: generic, detail: sanitized}`
- Step2 error: thay `err.Error()` bằng `service.SanitizeFreeformText(err.Error(), 200)`
- Step3 error: thay `err.Error()` bằng `service.SanitizeFreeformText(err.Error(), 200)`
- `markProvisioningFailed`: thay `err.Error()` bằng `service.SanitizeFreeformText(err.Error(), 2000)`

### File 4: `internal/admin/server_test.go` (+110 lines)
- Thêm import `errors`
- Thêm helper `newTestServer(t)` và `newTestServerWithMockDB(t, setupMock)`
- Thêm `TestAuthMiddleware_ConstantTimeCompare` (4 sub-tests)
- Thêm `TestRateLimit_Allows3ThenBlocks`
- Thêm `TestRegister_StepFailure_SanitizedError`
- Thêm `TestBodyLimit_TooLarge`

---

## Section 2: `go build ./...` Output

```
(no output — clean build, exit 0)
```

BUILD PASS

---

## Section 3: `go test ./internal/admin/ -count=1 -v` Output

```
=== RUN   TestRegisterSource_HappyPath
--- PASS: TestRegisterSource_HappyPath (0.03s)
=== RUN   TestRegisterSource_Unauthorized
--- PASS: TestRegisterSource_Unauthorized (0.00s)
=== RUN   TestRegisterSource_BadRequest
--- PASS: TestRegisterSource_BadRequest (0.00s)
=== RUN   TestRegisterSource_Step2Fail
--- PASS: TestRegisterSource_Step2Fail (0.03s)
=== RUN   TestHelpers_ContainsCSV
--- PASS: TestHelpers_ContainsCSV (0.00s)
=== RUN   TestHelpers_TopicNameFor
--- PASS: TestHelpers_TopicNameFor (0.00s)
=== RUN   TestHelpers_ShadowSchemaFor
--- PASS: TestHelpers_ShadowSchemaFor (0.00s)
=== RUN   TestExtendDatabaseList_NewValue
--- PASS: TestExtendDatabaseList_NewValue (0.00s)
=== RUN   TestExtendDatabaseList_AlreadyPresent
--- PASS: TestExtendDatabaseList_AlreadyPresent (0.00s)
=== RUN   TestExtendDatabaseList_EmptyConfig
--- PASS: TestExtendDatabaseList_EmptyConfig (0.00s)
=== RUN   TestExtendDebeziumInclude_Mongo_BothTiers
--- PASS: TestExtendDebeziumInclude_Mongo_BothTiers (0.00s)
=== RUN   TestExtendDebeziumInclude_Mongo_DBExistsCollNew
--- PASS: TestExtendDebeziumInclude_Mongo_DBExistsCollNew (0.00s)
=== RUN   TestExtendDebeziumInclude_PG_DBLockMismatch
--- PASS: TestExtendDebeziumInclude_PG_DBLockMismatch (0.00s)
=== RUN   TestAuthMiddleware_ConstantTimeCompare
=== RUN   TestAuthMiddleware_ConstantTimeCompare/wrong
=== RUN   TestAuthMiddleware_ConstantTimeCompare/length-mismatch
=== RUN   TestAuthMiddleware_ConstantTimeCompare/missing
=== RUN   TestAuthMiddleware_ConstantTimeCompare/happy-passes-auth
--- PASS: TestAuthMiddleware_ConstantTimeCompare (0.00s)
=== RUN   TestRateLimit_Allows3ThenBlocks
--- PASS: TestRateLimit_Allows3ThenBlocks (0.03s)
=== RUN   TestRegister_StepFailure_SanitizedError
--- PASS: TestRegister_StepFailure_SanitizedError (0.03s)
=== RUN   TestBodyLimit_TooLarge
--- PASS: TestBodyLimit_TooLarge (0.03s)
PASS
ok  	centralized-data-service/internal/admin	0.563s
```

Total: **18/18 PASS** (13 cũ + 5 mới F1)

---

## Section 4: Smoke Live Results

> Note: Port 8090 bị chiếm bởi process PID 62951 (F3 concurrent admin-api).
> Smoke chạy trên `ADMIN_API_LISTEN_ADDR=127.0.0.1:8091`. Process F3 KHÔNG bị kill.

### Boot fail-fast (unset both vars)
```
{"level":"fatal","caller":"admin-api/main.go:69",
"msg":"ADMIN_API_TOKEN is empty and ADMIN_API_DEV != 'true' — refusing to start without auth..."}
Exit code: non-zero (zap.Fatal → os.Exit(1))
```
PASS

### Dev mode warning
```
{"level":"warn","msg":"ADMIN_API_DEV=true — running without authentication. NEVER use this in production."}
{"level":"warn","msg":"ADMIN_API_TOKEN empty — auth disabled (dev mode only)"}
```
PASS

### Healthz
```
healthz=200
```

### Wrong token
```
wrong-token=401
```

### Length-mismatch token
```
len-diff=401
```

### Burst rate limit (12 requests)
```
burst#1=200
burst#2=200
burst#3=200
burst#4=429
burst#5=429
burst#6=429
burst#7=429
burst#8=429
burst#9=429
burst#10=429
burst#11=429
burst#12=429
```
PASS: Burst 3 → 200, request 4+ → 429

### Body size limit (70 KiB)
```
big-body=400
```
PASS

---

## Section 5: AC Mapping

| AC | Description | Result | Evidence |
|---|---|---|---|
| AC-F1-1 | Boot fail-fast khi ADMIN_API_TOKEN rỗng và ADMIN_API_DEV != 'true' | PASS | Smoke: fatal log + exit; main.go L69 |
| AC-F1-2 | Constant-time token compare — length mismatch và wrong value đều 401 | PASS | TestAuthMiddleware_ConstantTimeCompare/wrong, /length-mismatch, /missing; Smoke: 401 |
| AC-F1-3 | Per-token rate limit 10 req/min burst 3 + 429 + Retry-After | PASS | TestRateLimit_Allows3ThenBlocks; Smoke: burst#4+=429 |
| AC-F1-4 | Error response sanitized — credential values redacted, step indicator present | PASS | TestRegister_StepFailure_SanitizedError: "supersecret123" absent, "step1" present |
| AC-F1-5 | Body size limit 64 KiB + MaxHeaderBytes | PASS | TestBodyLimit_TooLarge: 70KiB → 400; Smoke: big-body=400 |
| AC-F1-6 | Tất cả 13 test cũ vẫn PASS | PASS | 18/18 test pass |
| AC-F1-7 | Smoke live verify toàn bộ scenarios | PASS | Smoke port 8091 (F3 concurrent avoided) |

---

## Section 6: Issues Encountered + Resolution

### Issue 1: TestRegister_StepFailure_SanitizedError — assertion mismatch
**Problem**: Test ban đầu dùng error `"relation \"secret_table.password_column\" does not exist"` và assert `NotContains "secret_table.password_column"`. Nhưng `SanitizeFreeformText` chỉ redact key=value credential patterns, không redact table names.  
**Resolution**: Đổi test error sang `"password=supersecret123 connection refused"` — pattern mà sanitizer xử lý. Assertion verify `supersecret123` bị redact + `"error"` field generic + `"step1"` present.

### Issue 2: Port 8090 bị chiếm (F3 concurrent)
**Problem**: Không thể bind port 8090.  
**Resolution**: Chạy smoke trên port 8091. Không kill process F3.

### Issue 3: nil DB panic trong happy-passes-auth subtest
**Problem**: Valid token → auth pass → handler → nil DB panic.  
**Resolution**: `gin.Recovery()` catch → 500. Test assert `!= 401` → PASS. Hành vi correct.

---

## Section 7: Skills Used

1. Read Tool — đọc 4 source file + 2 doc file trước khi thực thi
2. Edit Tool — targeted diff từng section
3. Bash Tool — go build, go test, lsof, curl, nohup để verify
4. `crypto/subtle.ConstantTimeCompare` — timing-safe token comparison
5. `golang.org/x/time/rate` — token-bucket rate limiter (thread-safe)
6. `http.MaxBytesReader` — body size capping
7. `service.SanitizeFreeformText` — credential redaction
8. sqlmock — inject DB errors không cần real DB
9. gin.Recovery() — panic recovery đảm bảo nil DB không leak

---

VERDICT: PASS
