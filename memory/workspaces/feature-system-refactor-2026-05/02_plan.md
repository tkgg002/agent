# 02 — Plan (Phase B1 + B2)

**Date**: 2026-05-04 17:10+07
**Phase**: B1 (Quick Wins) + B2 (Tooling & Hygiene)
**Strategy**: Inside-out — sửa hygiene + memory đầu, sau đó start dịch vụ + smoke. Không sửa source code Go/TS trừ khi chứng minh được lỗi cụ thể qua smoke.

---

## Sequence

1. **B1.1** Commit pending — Brain (git, không phải code edit).
2. **B1.2** Update `architecture.md` — Brain (`.md` không phải code).
3. **B1.3** Fill `project_context.md` + `tech_stack.md` — Brain.
4. **B1.4** Update `active_plans.md` — Brain.
5. **B1.5** APPEND lesson `L-input-fallback-pattern` — Brain.
6. **B2.1** Inventory + start `cdc-auth-service` — Brain (chạy lệnh build/run, không sửa source). Nếu fail → Muscle.
7. **B2.2** Inventory + start `cdc-cms-web` — Brain (npm install + npm run dev). Nếu fail → Muscle.
8. **B2.3** E2E operator smoke — Brain (curl/exercise). Nếu thấy bug → log vào `10_gap_analysis.md` (KHÔNG fix vội ở B2 — defer B3+).
9. **B2.4** Script `scripts/dev-up.sh` — Brain (shell script không phải Go/TS/SQL).
10. **B-final** Report + APPEND progress + verify all-services-work — Brain.

> Quy tắc: Brain edit `.md`, `.sh`, `Makefile`, file config dạng `.yml/.json` config (không phải `.go/.ts/.js/.py/.sql`). Edit code → Muscle delegate.

## Strategy

### B1 — Quick Wins

- B1.1 commit message tiếng Việt rõ:
  > `chore(admin-api): hardening Phase F1 + helpers.go fallback Phase F3`
- B1.2 sed/grep occurrences "Airbyte" trong architecture.md → thay bằng note "đã gỡ commit 8ef7d71". Giữ section Mongo→Debezium→Kafka như cũ.
- B1.3 `project_context.md` fill:
  - Project Name: cdc-system
  - Scale: 4 service (3 Go + 1 TS), monorepo, ~250 .go file + 22 tsx
  - Stage: Development local + smoke
  - Domain terminologies: shadow, master, transmute, recon, schema_proposal, source_object_registry, master_binding, fencing
  - Business rules: schema_status='approved' gate, OCC theo `_source_ts`, masking PII trước log, write-before-publish DLQ
- B1.3 `tech_stack.md` fill: Go 1.26.1 (3 service), Vite/React/TS (FE), PG/Mongo/MariaDB/Redis/Kafka/NATS/SchemaRegistry/Debezium/OTel.
- B1.4 active_plans entry mới + close Phase F.
- B1.5 lesson abstract:
  > Global Pattern [A reads optional key K from request payload B → uses raw value as table-name part X] → Result Y: empty propagation, dirty entries, stuck pipeline. Đúng: A PHẢI fallback to canonical field B.canonicalName khi K missing/empty.

### B2 — Tooling & Hygiene

- B2.1: `cd cdc-auth-service && cat config/* && cat Makefile` → tìm port + DB + JWT config. Build `go build ./cmd/server`. Run với env phù hợp. Smoke `/healthz` + `/api/v1/auth/login` (nếu có) + ghi test user.
- B2.2: `cd cdc-cms-web && cat package.json && cat .env* 2>/dev/null` → tìm API base URL. `npm run dev` background. Smoke `curl localhost:5173/` → 200. Nếu cần xem page UI → user check browser, Brain ghi rõ "browser smoke yêu cầu user xác nhận".
- B2.3: chuẩn bị `b2_smoke_*` prefix. Script `b2_smoke.sh` chạy curl tuần tự. Verify shadow + master sau cron tick (1 phút).
- B2.4: `scripts/dev-up.sh` cấu trúc:
  - `set -euo pipefail`
  - check 13 docker container healthy (lệnh `docker ps`).
  - start auth (background, log /tmp/cdc-auth.log).
  - start cms-server (đã chạy, skip nếu PID alive).
  - start cdc-worker (đã chạy docker, skip).
  - start admin-api (đã chạy, skip).
  - start FE dev (background).
  - check 4 health endpoints, exit 0 nếu OK.

## Verification per step

| Step | Verify command | Pass criterion |
|---|---|---|
| B1.1 | `git log -1 --stat` | Hiện 5 file mới commit |
| B1.2 | `grep -ic airbyte cdc-system/architecture.md` | ≤ 1 (chỉ note đã gỡ) |
| B1.3 | `grep -c "\[DATE\]\|\[Project Name\]" agent/memory/global/{project_context,tech_stack}.md` | 0 |
| B1.4 | `grep "feature-system-refactor-2026-05" agent/memory/global/active_plans.md` | 1+ |
| B1.5 | `grep "L-input-fallback-pattern" agent/memory/global/lessons.md` | 1+ |
| B2.1 | `curl -sS http://127.0.0.1:<port>/healthz` | 200 |
| B2.2 | `curl -sS http://localhost:5173/` | 200 |
| B2.3 | psql query shadow + master | rows landed |
| B2.4 | `bash scripts/dev-up.sh; echo $?` | 0 |
| B-final | `report_phase_b_*.md` | exists + có evidence |

## Out-of-scope của phase này

- B3 architectural debts (D1 auto-create shadow, D2 prune V1, D5 master cascade từ admin-api) → Phase B3 sau.
- B4 testing reinforcement → Phase B4 sau.
- B5 big refactor → User chốt riêng nếu cần.
- Issue 6/7/8 LOW từ E5 report.

## Lessons re-applied

- "Brain plan dựa state tưởng tượng" → mọi step Brain tự verify disk/runtime.
- "Service listening ≠ healthy" → smoke business endpoint, không chỉ /healthz.
- "Báo Done mà không restart" → mọi commit + restart phải có log evidence.
- "Fix bug 1 service quên cross-service" → B2.3 cần test xuyên 4 service.
- "Build pass ≠ test pass" → phải chạy test go test, npm test (nếu có).
