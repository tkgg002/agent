# 01 — Requirements Phase F1 — Admin-API Security Hardening

**Date**: 2026-05-04
**Phase**: F1 (security hardening tiếp theo Phase E close-loop)
**Trigger**: Phase E5 security gate trả 2 HIGH + 3 MEDIUM. User mandate: *"Phase F1: Triển khai 5 issue bảo mật đã nêu trong báo cáo."*
**Source report**: `report_security_agent_admin_api_20260504.md`.
**Author**: Brain (Antigravity).

---

## 1. Functional Requirements

### FR-F1-1 — Boot-time fail-fast cho `ADMIN_API_TOKEN`
- Service `cdc-admin-api` PHẢI exit non-zero ngay khi boot nếu env `ADMIN_API_TOKEN` empty AND env `ADMIN_API_DEV != "true"`.
- Khi `ADMIN_API_DEV=true` được set rõ ràng, service chạy với auth disabled + log WARN ngay từ boot.
- Khi `ADMIN_API_TOKEN` set: service chạy bình thường với enforced auth.

### FR-F1-2 — Constant-time token compare
- Bearer token compare PHẢI dùng `crypto/subtle.ConstantTimeCompare`.
- Length-equality check trước khi gọi ConstantTimeCompare để tránh leak length info qua early return.

### FR-F1-3 — Rate limit per-token cho POST orchestration endpoint
- POST `/v2/sources/register` PHẢI bị rate limit để chống DoS spam Debezium config rewrite.
- Limit policy ban đầu: **10 req/min per token** (token-bucket, burst=3).
- Vượt quota: trả 429 `{"error": "rate limited"}` + header `Retry-After`.
- `/healthz` không bị rate limit.

### FR-F1-4 — Sanitize error response body
- Response 5xx/207 PHẢI dùng `service.SanitizeFreeformText(err.Error(), 200)` thay vì raw `err.Error()`.
- Internal full error PHẢI vẫn log qua `s.deps.Logger.Error(...)` cho operator.
- Field `LastStepError` trong `RegisterSourceResponse` cũng phải sanitize.

### FR-F1-5 — Request body size limit + header size cap
- HTTP body PHẢI bị giới hạn 64 KiB qua `http.MaxBytesReader` middleware.
- `httpSrv.MaxHeaderBytes = 64 * 1024`.
- Vượt limit: trả 413 `{"error": "request body too large"}`.

---

## 2. Non-Functional Requirements

| Code | NFR |
|---|---|
| NFR-F1-A | Backward compat — không break smoke test cũ E1 (test với token = "secret-test-token"). |
| NFR-F1-B | Zero new external dep — sử dụng `crypto/subtle`, `golang.org/x/time/rate` (đã có v0.15.0), `service.SanitizeFreeformText` (đã có). |
| NFR-F1-C | Test coverage — mỗi fix ≥ 1 unit test mới (5 fix × ≥1 test = ≥5 test added). |
| NFR-F1-D | Smoke verify live — 1 token-valid request PASS, 1 token-empty request 401, 1 spam 11 req → ít nhất 1 × 429. |
| NFR-F1-E | Memory append-only (CLAUDE.md §11). |
| NFR-F1-F | Brain prohibition (CLAUDE.md §12) — toàn bộ `.go` edit qua Muscle Agent. |

---

## 3. Acceptance Criteria

- [x-target] **AC-F1-1** — `ADMIN_API_TOKEN=""` + `ADMIN_API_DEV=""` → service exit code ≠ 0; stderr message rõ.
- [x-target] **AC-F1-2** — `crypto/subtle.ConstantTimeCompare` xuất hiện trong code path auth; old `got != want` xóa.
- [x-target] **AC-F1-3** — 11 POST nhanh trong 60s với cùng token → ≥1 response 429 + header `Retry-After`.
- [x-target] **AC-F1-4** — Inject error path trả response body chứa `[redacted]` hoặc generic message, KHÔNG có raw SQL fragment / table name leak.
- [x-target] **AC-F1-5** — POST body 1 MiB → 413; POST body 1 KiB happy path 200.
- [x-target] **AC-F1-6** — `go build ./...` PASS, `go test ./internal/admin/ -count=1` PASS (existing 13 + new ≥5).
- [x-target] **AC-F1-7** — Smoke live admin-api restart PASS; `/healthz` 200; valid POST 200.

---

## 4. Out of Scope (defer)

- LOW issue 6 (SourceLocator typed struct) → Phase F2.
- LOW issue 7 (CORS) → Phase F2 khi có frontend.
- LOW issue 8 (audit trail actor/IP) → Phase F2.
- F2 V2 anchor port cho `addtest_pg_orders` → riêng phase.
- F3 E2 smoke → song song F1 (validate Phase E artifact).

---

## 5. Risks

| Risk | Mitigation |
|---|---|
| Boot fail-fast break docker-compose dev (token chưa set) | Yêu cầu set `ADMIN_API_DEV=true` trong compose dev override; document rõ |
| Rate limit storage in-memory → reset khi restart | Acceptable cho admin endpoint (low-frequency, single-instance), document trong code comment |
| Sanitize đụng existing `LastStepError` consumer | Brain check `RegisterSourceResponse` consumer hiện tại — chưa có client production, smoke test chỉ assert `provisioning_state` |
| Body size limit chặn legitimate big payload | 64 KiB đủ rộng cho `SourceLocator` + `Notes`; document trong API spec |

---

## 6. Dependencies + Inputs

- `centralized-data-service/cmd/admin-api/main.go` (line 64-74)
- `centralized-data-service/internal/admin/server.go` (line 32-86)
- `centralized-data-service/internal/admin/source_register.go` (line 38-77)
- `centralized-data-service/internal/admin/types.go` (line 24)
- `centralized-data-service/internal/admin/server_test.go` (line 1-265)
- `centralized-data-service/internal/service/text_sanitizer.go` (helper reused)
- `centralized-data-service/go.mod` — `golang.org/x/time v0.15.0` (already)

---

## 7. Definition of Done

1. ✅ 5 fix landed trên 4 file: `cmd/admin-api/main.go`, `internal/admin/server.go`, `internal/admin/source_register.go`, `internal/admin/server_test.go`.
2. ✅ `go build ./...` PASS.
3. ✅ `go test ./internal/admin/ -count=1` PASS toàn bộ (existing + new).
4. ✅ Live restart admin-api → `/healthz` 200, valid POST 200, spam → 429.
5. ✅ Boot tests: `ADMIN_API_TOKEN=""` không kèm DEV → exit non-zero.
6. ✅ Brain APPEND `05_progress.md` + ghi `report_phase_f1_security_hardening_<ts>.md`.
7. ✅ Memory append-only, Brain 0 source code edit.
