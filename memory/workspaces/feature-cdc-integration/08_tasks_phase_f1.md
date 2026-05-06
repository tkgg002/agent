# 08 — Tasks Phase F1 (atomic)

**Phase**: F1 — Admin-API Security Hardening (5 issue từ E5 report).
**Total atomic tasks**: 18.

---

## Sequencing rule

Code task chạy theo thứ tự **F1-1 → F1-2 → F1-3 → F1-4 → F1-5** (perimeter inward). Tests có thể viết song song trong cùng commit, miễn là implementation cho issue đó đã đặt.

| ID | Task | Owner | Blocks |
|---|---|---|---|
| F1-1A | Edit `cmd/admin-api/main.go` thêm DEV check + fatal | Muscle | F1-1B |
| F1-1B | `go build ./cmd/admin-api/` PASS | Muscle | F1-2A |
| F1-2A | Edit `internal/admin/server.go` import `crypto/subtle` + thay token compare | Muscle | F1-2B |
| F1-2B | Add unit test `TestAuthMiddleware_ConstantTimeCompare` | Muscle | F1-2C |
| F1-2C | `go test ./internal/admin/ -run TestAuthMiddleware_` PASS | Muscle | F1-3A |
| F1-3A | Edit `internal/admin/server.go` thêm `rateLimitMiddleware` + store | Muscle | F1-3B |
| F1-3B | Wire middleware vào `buildEngine` sau auth | Muscle | F1-3C |
| F1-3C | Add unit test `TestRateLimit_*` (allow 3, block 4th) | Muscle | F1-3D |
| F1-3D | `go test ./internal/admin/ -run TestRateLimit_` PASS | Muscle | F1-4A |
| F1-4A | Edit `internal/admin/source_register.go` import service + sanitize 3 call site + 2 LastStepError | Muscle | F1-4B |
| F1-4B | Logger.Error full err mỗi step fail | Muscle | F1-4C |
| F1-4C | Add unit test `TestRegister_StepFailure_SanitizedError` | Muscle | F1-4D |
| F1-4D | `go test ./internal/admin/ -run TestRegister_StepFailure_` PASS | Muscle | F1-5A |
| F1-5A | Edit `internal/admin/server.go` thêm `bodyLimitMiddleware` + `MaxHeaderBytes` | Muscle | F1-5B |
| F1-5B | Wire middleware TRƯỚC auth | Muscle | F1-5C |
| F1-5C | Add unit test `TestBodyLimit_TooLarge` | Muscle | F1-5D |
| F1-5D | `go test ./internal/admin/ -run TestBodyLimit_` PASS | Muscle | F1-V |
| F1-V  | Smoke live: build → restart binary → 4 curl test (healthz, valid, wrong-token, 11×burst, 413) | Muscle | F1-R |
| F1-R  | Brain APPEND `05_progress.md` + write `report_phase_f1_security_hardening_<ts>.md` | Brain | (end) |

---

## Estimated time

- Muscle implementation: 2-3h (5 fix × ~30min + test).
- Smoke live verify: 15min.
- Brain plan + report: 30min (đã chạy plan).

Total: ~3-4h.

---

## DoD ⇨ Acceptance Criteria mapping

| AC | Verified by |
|---|---|
| AC-F1-1 | F1-1B + smoke boot test |
| AC-F1-2 | F1-2A + F1-2C |
| AC-F1-3 | F1-3D + F1-V curl 11×burst |
| AC-F1-4 | F1-4D |
| AC-F1-5 | F1-5D + F1-V curl 70KB |
| AC-F1-6 | F1-1B + F1-2C + F1-3D + F1-4D + F1-5D |
| AC-F1-7 | F1-V |
