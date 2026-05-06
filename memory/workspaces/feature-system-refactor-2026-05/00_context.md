# 00 — Context: System Refactor 2026-05

**Created**: 2026-05-04 17:05+07
**Owner**: Brain (Antigravity), CC = Muscle (CLI)
**Status**: 🟡 Active — Discovery + Plan
**Trigger**: User mandate "hệ thống thành nồi cám heo, refactor toàn hệ thống, giúp service chạy được các chức năng"

---

## Repo Layout (sự thật trên disk lúc 17:05)

`/Users/trainguyen/Documents/work/cdc-system` là **monorepo single git** — 6 commit, branch `main`. 5 file uncommitted (Phase F1+F3 fix admin-api, 286 line).

| Service | Loại | go.mod / package | Lines / Files | Test | Live state |
|---|---|---|---|---|---|
| `cdc-auth-service` | Go | `cdc-auth-service` (fiber+JWT) | 9 .go | 0 test | KHÔNG thấy chạy |
| `cdc-cms-service` | Go | `cdc-cms-service` (fiber+nats+redis) | 76 .go | 10 test | Live local `/tmp/cms-server` PID 13653, uptime 5d |
| `centralized-data-service` | Go | 4 binary: `worker`, `admin-api`, `sinkworker`, `profile_table` | 144 .go | 39 test | Worker docker `gpay-cdc-worker` (2h), admin-api `/tmp/cdc-admin-api-f3v2` |
| `cdc-cms-web` | TS/Vite/React | `vite + tsc -b` | 22 .tsx, 7634 LOC | tsc clean | KHÔNG thấy chạy |

Build từng service hiện tại đều `go build ./... → exit 0`, FE `tsc --noEmit → exit 0`. Compile OK toàn cục.

---

## Live Infra (docker ps + lsof)

| Container | Port | Tuổi | Health |
|---|---|---|---|
| gpay-postgres | 5432 | 6d | healthy |
| gpay-postgres-cdc | 5433 | 6d | healthy |
| gpay-postgres-dest | 5434 | 6d | healthy |
| gpay-postgres-source | 5435 | 6d | healthy |
| gpay-mongo | 17017 | 6d | healthy |
| gpay-mariadb | 13307 | 5d | healthy |
| gpay-redis | 16379 | 6d | up |
| gpay-kafka | 19092/19093 | 6d | up |
| gpay-kafka-connect | 18083 | 6h | healthy |
| gpay-schema-registry | 18081 | 6d | up |
| gpay-nats | 14222 | 6d | up |
| gpay-otel-collector | 14317/14318 | 6h | up |
| gpay-cdc-worker | 8080 | 2h | up |

---

## Active vs Drift Findings

1. Memory `active_plans.md` ghi `feature-cdc-integration` đang Active "Hybrid Debezium + Airbyte" — nhưng commit `8ef7d71 remove airbyte` cho thấy Airbyte đã bị gỡ. Doc drift.
2. Memory `feature-cdc-system-refactor/07_status.md` ghi "Implemented and validated locally" nhưng `10_gap_analysis.md` flag DLQ wiring still pending. Out-of-date.
3. Workspace cũ liên quan refactor (cần khảo cứu trước khi plan):
   - `feature-cdc-system-refactor` — Sprint cũ
   - `feature-refactor-2026` — GooPay core refactor 2026
   - `feature-cdc-integration` — đang Active, tích lũy 12+ phase, có 1700+ dòng lessons.md
   - `feature-multi-pg-isolation-e2e` — vừa Active, plan curried-waddling-spindle (P2/P3/P4) chưa execute

---

## Constraints (CLAUDE.md)

- §1 Brain Chairman, Muscle Engineer — Brain TUYỆT ĐỐI không edit `.go/.ts/.sql`.
- §11 APPEND-only memory.
- §7 Full Doc Set per phase.
- §13 Lesson abstract Global Pattern A/B/X/Y.
- §14 Pre-flight check trước khi kết thúc.
- Lessons cảnh báo:
  - "Plan dựa state tưởng tượng" → đã có scan thật.
  - "ngầu từ ngữ thiếu OPS reality" → plan phải có Scale Budget + verify path.
  - "scope-cut hèn nhát" → đề xuất Systematic Reconstruction, không band-aid.
  - "Báo Done mà không restart + verify service chạy" → mọi PASS cần exercise-driven.
  - "Service listening ≠ healthy" → smoke business endpoint.

---

## Mandate Open Questions (cần User clarify TRƯỚC khi plan chi tiết)

1. **Scope**: refactor 4 service đồng thời, hay chỉ 1 service ưu tiên trước?
2. **"Chức năng work"** = chức năng nào cụ thể? (CDC pipeline đang work theo report Phase F. Có chức năng nào *operator-facing* đang đứt mà user thấy gần nhất?)
3. **Goal 1 dòng**: clean up code drift / merge fragmented branches / rebuild from scratch / fix specific UX flow / consolidate config?
4. **Risk tolerance**: được phép kill service đang chạy để rebuild, hay phải zero-downtime?
5. **Time budget**: 1 ngày spike, 1 tuần phase, hay open-ended?

→ Q1 + Q3 sẽ quyết định kích thước phase đầu tiên. Brain đề xuất phase nhỏ trước (xem `02_plan.md`), nhưng giữ đường mở để mở rộng.
