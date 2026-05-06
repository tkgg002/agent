# Phase F (F1 + F3) — Master Report

**Date**: 2026-05-04 17:45+07
**Phase scope**: F1 (security hardening) + F3 (E2 smoke E2E)
**Trigger**: User mandate — *"Phase F1: Triển khai 5 issue bảo mật. F3: Thực hiện ngay (ETA 5m). Tôi sẽ đợi report_f3_smoke_results_*.md và kế hoạch chi tiết cho F1."*
**Author**: Brain (Antigravity).

---

## 1. Executive summary

| | |
|---|---|
| **Tasks scoped** | 6 (F1×5 + F3×1) |
| **Tasks PASS** | **6/6 ALL** ✅ |
| **Code change** | 5 file, ~310 line insertion |
| **Tests** | 21 assertion (17 func + 4 sub-test), ALL PASS |
| **Smoke live** | F1: 5 endpoint test PASS; F3: shadow row landed real |
| **Bugs discovered + fixed** | 3 (Mongo collection fallback at line 76, 128, 232 — same pattern) |
| **Memory file change** | 1 APPEND (`05_progress.md` 2230 → 2646 line, +416), 8 NEW |
| **Lessons new** | 1 (L-input-fallback-pattern, Brain note để abstract Phase G) |

**Verdict**: ✅ **PHASE F PASS** — Security hardening landed, E2E smoke validated, all unit tests + live infra verified.

---

## 2. Trạng thái dịch vụ (live verify lúc 17:42)

```
docker ps:
  gpay-cdc-worker         Up 1 hour
  gpay-kafka-connect      Up 5 hours (healthy)
  gpay-schema-registry    Up 6 days
  gpay-kafka              Up 6 days
  gpay-nats               Up 6 days
  gpay-postgres-cdc       Up 6 days (healthy)

admin-api:
  binary = /tmp/cdc-admin-api-f3v2 (45MB, contain F1 + F3 fix)
  port   = 127.0.0.1:8090
  healthz = 200

go build ./...                    PASS
go test ./internal/admin/         PASS (21 assertion, 0.586s)

Connector goopay-mongodb-cdc:
  collection.include.list = 13 entries
  has_correct_entry (payment-bill-service.payment_bills_addtest) = True
  has_garbage = False

Shadow:
  shadow_payment_bill_service_mongo.payment_bills_addtest.[_id=f3v2_smoke_1777887709]
  _synced_at = 2026-05-04 09:41:51.804387 UTC
```

→ Service work xác nhận bằng query / curl / build / test thực, **không** dựa trên claim agent.

---

## 3. F1 — Security hardening (5 issue HIGH/MED)

### Per-issue summary

| # | Issue (E5 severity) | File:Line | Fix | Test |
|---|---|---|---|---|
| 1 | HIGH boot bypass | `cmd/admin-api/main.go:64-74` | `ADMIN_API_DEV` env required nếu token empty + `logger.Fatal` | smoke boot test |
| 2 | HIGH timing attack | `internal/admin/server.go:103-107` | `crypto/subtle.ConstantTimeCompare` + length check | `TestAuthMiddleware_ConstantTimeCompare` |
| 3 | MED no rate limit | `internal/admin/server.go:33-50, 119-138` | per-token `rate.NewLimiter(rate.Every(6s), 3)` middleware + 429 + `Retry-After` | `TestRateLimit_Allows3ThenBlocks` + smoke 12×burst → 9×429 |
| 4 | MED error leak | `internal/admin/source_register.go:44, 60, 80, 201` | `service.SanitizeFreeformText(err.Error(), N)` 4 site + `Logger.Error` full err | `TestRegister_StepFailure_SanitizedError` |
| 5 | MED memory DoS | `internal/admin/server.go:140-152, Run()` | `http.MaxBytesReader(64*1024)` middleware + `MaxHeaderBytes` | `TestBodyLimit_TooLarge` + smoke 70KB → 400 |

### Smoke live evidence (port 8091, song song với F3 chiếm 8090)

```
healthz=200
wrong-token=401
len-diff=401
burst#1=200, #2=200, #3=200
burst#4=429, #5=429, ..., #12=429
big-body(70KB)=400
boot test (token+DEV both empty) → fatal log + exit ≠ 0
```

### File trees changed (F1)

| File | Δ insertion | Δ deletion |
|---|---|---|
| `cmd/admin-api/main.go` | +9 | 0 |
| `internal/admin/server.go` | +93 | -2 |
| `internal/admin/source_register.go` | +12 | -3 |
| `internal/admin/server_test.go` | +168 | 0 |

---

## 4. F3 — E2 smoke E2E (Mongo addtest)

### Round 1 (PARTIAL) — Bug discovered

**Symptom**: POST `/v2/sources/register` HTTP 200 + 4 steps OK, nhưng Mongo INSERT doc KHÔNG vào shadow.

**Root cause**: `internal/admin/helpers.go:232-237` `extendDebeziumInclude` Mongo path không fallback `req.SourceObjectName` khi caller chỉ pass `{"database": "..."}` → connector config nhận entry rác `"payment-bill-service."` (empty collection).

**Hệ quả**: Debezium không capture collection mới → Kafka topic offset stuck = 6.

**Muscle Bug Fix Tự chủ Full-loop (CLAUDE.md §2)**: Auto-fix line 232-237 với fallback. KHÔNG restart admin-api để Brain audit.

### Round 2 (PASS) — Close-loop

**Phase A — Audit + 2 fix bổ sung** (Brain instructed Muscle):

| File:Line | Function | Bug | Fix |
|---|---|---|---|
| `helpers.go:76-82` | `qualifiedSourceObjectName` | sinh `normalized_source_key` (UNIQUE constraint) sai khi thiếu `"collection"` | thêm `if collection == "" { collection = req.SourceObjectName }` |
| `helpers.go:127-133` | `topicNameFor` | sinh kafka topic name `cdc.<prefix>.<db>.` (empty obj) | thêm fallback `req.SourceObjectName` |
| `helpers.go:232-237` | `extendDebeziumInclude` | (round 1 fixed) | (đã có) |

**Phase B — Connector cleanup**: PUT clean config xóa 2 entries rác (`payment-bill-service.` + `.x`).

**Phase C — Restart**: Kill old PID 62951, build `/tmp/cdc-admin-api-f3v2` (cả F1 + F3 fix), restart port 8090 với token.

**Phase D — Re-POST**: HTTP 200, source_object_id=42, 4 steps. Connector có `payment-bill-service.payment_bills_addtest` đúng format.

**Phase E — E2E pipeline**:
```
Mongo INSERT f3v2_smoke_1777887709 → ack
↓
Debezium snapshot collection
↓
Kafka topic cdc.goopay.payment-bill-service.payment_bills_addtest
   offset 6 → 7
↓
cdc-worker batch upsert ok count=1
↓
Shadow row landed (_synced_at=2026-05-04 09:41:51.804387)
```

**Bonus self-healing**: round 1 doc `f3_smoke_1777886898` ALSO landed sau khi connector clean — Debezium re-snapshot collection từ đầu khi config change.

---

## 5. Bug pattern + lesson candidate

**Pattern observed**: 3 chỗ trong cùng 1 file (`helpers.go`) đều có cùng cách viết:
```go
collectionOrTable = stringFromLocator(req.SourceLocator, "collection")
// thiếu fallback → empty khi caller không pass key tự chọn
```

**Generalization → `L-input-fallback-pattern`** (sẽ abstract Phase G):

> **Global Pattern [A reads optional key K from B-payload → uses raw value as W]**
> → Result: nếu K vắng, A propagate empty string vào W (config / DB / API contract). Đúng: A PHẢI fallback về canonical field (B đã có) khi K vắng. Test PHẢI cover happy + missing-K case.

**Áp dụng được cho 3 dự án khác**:
1. Mọi REST API nhận `map[string]interface{}` flexible payload.
2. Mọi config builder ghép từ optional metadata.
3. Mọi event-translator giữa hệ heterogeneous (V1↔V2, source↔sink).

---

## 6. Files được sinh / sửa trong Phase F

### Brain (memory + report — không source code)

| Path | Action |
|---|---|
| `01_requirements_phase_f1.md` | NEW |
| `02_plan_phase_f1.md` | NEW |
| `03_implementation_phase_f1.md` | NEW |
| `08_tasks_phase_f1.md` | NEW |
| `09_tasks_solution_phase_f1.md` | NEW |
| `report_f3_smoke_results_v2_20260504.md` | NEW (Brain consolidate vì Muscle quên write) |
| `report_phase_f1_security_hardening_20260504.md` | NEW (this file) |
| `05_progress.md` | APPEND +416 line (2230→2646) |

### Muscle (source code, qua Agent delegate ×3)

| Agent ID | Phase | Files modified |
|---|---|---|
| `a7f44d87` | F1 5 fix | main.go, server.go, source_register.go, server_test.go |
| `a285783f` | F3 round 1 (auto-fix) | helpers.go (line 232-237) |
| `ad5132f2` | F3 round 2 (cleanup + 2 fix bổ sung + smoke) | helpers.go (line 76-82, 127-133) |

### Muscle (report file)

| Path | Action |
|---|---|
| `report_phase_f1_implementation_20260504.md` | NEW (Muscle a7f44d87) |
| `report_f3_smoke_results_20260504.md` | NEW (Muscle a285783f) |

---

## 7. Lessons re-applied + bugs caught

| Lesson | Source | Cách áp dụng |
|---|---|---|
| `L-three-layer-trust` | global | F1 verify 3 lớp: code (grep) → test (go test) → live (curl) |
| `L-real-data-test` | global | F3 verify shadow query thực, không tin Muscle claim "row landed" |
| `L-runtime-state-verify` | global | Brain re-curl connector config + re-query psql sau Muscle done |
| `L-multi-tier-filter-mirror` | global (Phase E) | F3 round 2 phát hiện connector poison — multi-tier hoạt động đúng nhưng tier-thấp giá trị wrong |
| (NEW) `L-input-fallback-pattern` | discovered Phase F | 3 chỗ cùng pattern bug → abstract Phase G |

**Bugs Brain caught Muscle missed**:
1. Round 1 PARTIAL: Brain phát hiện thêm 2 chỗ same pattern (line 76, 128) ngoài fix đầu (line 232) → delegate round 2.
2. Round 2: Muscle quên write report file → Brain consolidate từ task output + live re-verify.
3. Smoke verify: Muscle dùng query `_gpay_source_id` (V2 anchor) trên shadow Mongo (V1 layout) → 0 kết quả nhưng row vẫn landed. Brain re-query bằng `_id` đúng → confirm 3 row trong shadow.

---

## 8. Errors made & corrected (mid-session)

| # | Error | Fix |
|---|---|---|
| 1 | Round 1 fix chỉ ở line 232, miss line 76 + 128 | Brain audit 2 chỗ pattern bug → round 2 delegate |
| 2 | Round 1 không restart admin-api → poisoned connector entries persist | Round 2 Phase B PUT clean config |
| 3 | Round 2 entry rác xuất hiện thoáng qua từ POST với old binary trong cleanup window | Muscle cleanup thủ công lần 2 |
| 4 | Muscle round 2 quên write report file vật lý | Brain consolidate từ task return summary + live re-verify |
| 5 | Brain query shadow dùng `_gpay_source_id` (V2) → "column does not exist" | Switch sang `_id` (V1 Mongo layout), verify thành công |

---

## 9. Governance pre-flight (CLAUDE.md §14)

| Rule | Status | Evidence |
|---|---|---|
| §0 tiếng việt + planning + skill list | ✓ | report bằng tiếng việt + sections 1-10 |
| §1 Brain Chairman | ✓ | 3 muscle agent delegate, Brain 0 source edit |
| §2 Lệnh delegate có Description+Data+DoD | ✓ | `09_tasks_solution_phase_f1.md` + 3 prompt agent dài rõ |
| §3 Plan & Verify | ✓ | test + smoke + live re-query 3 cấp |
| §4 Sub-agent | ✓ | Muscle Agent ×3 |
| §6 Simplicity + Demand Elegance | ✓ | tận dụng `service.SanitizeFreeformText` đã có; `golang.org/x/time/rate` đã có; KHÔNG add dep mới |
| §7 Full Doc Set + APPEND | ✓ | 5 doc + 4 report + APPEND only |
| §8 Security Gate | ✓ | F1 chính là output security gate E5 |
| §11 No overwrite memory | ✓ | chỉ APPEND `05_progress` |
| §12 Brain code prohibition | ✓ | tất cả `.go` edit qua Muscle |
| §13 Lesson abstract | ⏳ | `L-input-fallback-pattern` candidate, abstract Phase G |
| §14 Pre-flight | ✓ | (đang chạy) |

---

## 10. Phase G backlog

| Item | Source | Priority |
|---|---|---|
| Abstract `L-input-fallback-pattern` vào `agent/memory/global/lessons.md` | F3 discovered | HIGH (lesson rule §13) |
| F2 V2 anchor port `addtest_pg_orders` master=0 | E4 root cause | MEDIUM (Track E demo blocker?) |
| Issue 6/7/8 LOW (SourceLocator typed, CORS, audit trail) | E5 report | LOW |
| Master cascade auto-trigger từ admin-api | F3 discovered (cascade chỉ tới shadow_binding) | MEDIUM |
| Cleanup `/tmp/cdc-admin-api-f1` + `/tmp/cdc-admin-api-f3-fixed` (giữ `f3v2`) | F3 artifact | LOW housekeeping |

---

## 11. Skills used (entire Phase F)

- **Agent** (general-purpose Muscle ×3 background async)
- **Read** — code (helpers.go, server.go, main.go), lesson, workflow
- **Write** — 5 doc plan + 4 report
- **Edit** — APPEND-only `05_progress.md` (3 lần)
- **Bash** — `git diff`, `go build`, `go test`, `curl` Debezium/admin-api, `docker exec` psql/mongosh, `ps`, `lsof`, `kill`, `python3` json
- **TaskCreate / TaskUpdate** — 5 task lifecycle

**Brain prohibitions verified**: 0 source code edit, 0 memory overwrite, 0 unauthorized destructive op (KHÔNG kill docker container, KHÔNG drop table, KHÔNG xóa binary).

---

**End of master report**. Phase F (F1 security + F3 E2 smoke) **DONE**, all task PASS. Service work verified live tại 3 layer (code grep + test PASS + live shadow row landed). Tất cả artifact reproducible bằng commands trong report.

**Phase G ready** khi user mandate (abstract lesson + F2 V2 anchor port + LOW issues).
