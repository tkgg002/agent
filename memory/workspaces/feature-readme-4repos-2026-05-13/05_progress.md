# 05 Progress — feature-readme-4repos-2026-05-13

> APPEND-ONLY. CLAUDE.md §11.

## 2026-05-13 — Session start

- Boss directive: "cập nhật README.md cho 4 repo chính."
- Muscle (CC CLI) đảm nhiệm. Brain không can dự.
- Đọc context: `project_context.md`, `tech_stack.md`, `active_plans.md` (global).
- Workspace `feature-readme-4repos-2026-05-13` khởi tạo: 00_context, 01_requirements, 02_plan, 08_tasks, 05_progress.
- Tiếp theo: scan repo + write 4 README.

## 2026-05-13 — Scan repos

- `cdc-auth-service`: Go 1.26.1, Fiber, JWT v5, GORM/PG. Binary `cmd/server`. Port `:8081`. Routes confirmed: `/health`, `/api/auth/{login,register,refresh}`, `/swagger/*`. DB `gpay_auth` @ 5432.
- `cdc-cms-service`: Go 1.26.1, Fiber + NATS + Redis + Prom + OTel + cron. 2 binary (`cmd/server`, `cmd/sync_v2`). 52 migrations. Hexagonal layout `app/{commands,queries,ports}` + `domain/{source,mapping,master,job,reconciliation}` + `infra/{cache,http,messaging,observability,persistence}` xác nhận. DB `cdc_dw` @ 5433. NATS subjects `cdc.cmd.*` (26 subjects unique) confirmed bằng grep.
- `cdc-cms-web`: React 19 + Vite 8 + AntD 6 + TanStack Query 5 + react-router 7. 14 pages confirmed via `ls src/pages`. Scripts `dev/build/build:prod/lint/preview`. README cũ là Vite template default → overwrite.
- `centralized-data-service`: 4 binary `worker/admin-api/sinkworker/profile_table` confirmed. Multi-DB layout `cdc_dw(5433)`/`goopay_dest(5434)`/`goopay_source(5435)`/`gpay_auth(5432)`. Kafka topic prefix `[cdc.gpay, cdc.goopay, cdc.mariadb]`. Makefile có `infra/migrate/reset-local/wipe` (destructive marker noted).

## 2026-05-13 — README written

- `cdc-system/cdc-auth-service/README.md` — 126 dòng. ✅
- `cdc-system/cdc-cms-service/README.md` — 152 dòng. ✅
- `cdc-system/cdc-cms-web/README.md` — 125 dòng (overwrite Vite template default). ✅
- `cdc-system/centralized-data-service/README.md` — 200 dòng (FS case-insensitive: overwrite `readme.md` cũ). ✅

DoD R3 đạt: tất cả ≥ 80 và < 400 dòng. File vật lý xác nhận qua `wc -l` + `head -1`.

## 2026-05-13 — Lesson learned

- **Pattern macOS FS case-insensitive**: Khi repo có file `readme.md` (lowercase) và Write `README.md` (uppercase), tool yêu cầu Read trước vì FS coi 2 path là CÙNG inode. Cần Read uppercase path để pass guard, sau đó Write overwrite cùng inode.
- Global Pattern [A writes file B with case variant X to repo Y với case-insensitive FS Z] → Result: tool guard reject vì coi là existing file. Đúng: Read uppercase path Y/B before Write, không cần rename.

## 2026-05-13 — Pre-flight check (CLAUDE.md §14)

- §0: trả lời tiếng Việt + plan trước + liệt kê skill ở cuối ✓
- §1: Muscle thi công, Brain không can dự ✓
- §3: plan đầy đủ trước khi gõ; verification = Read + wc ✓
- §6: minimal impact — chỉ touch README per-repo, không sửa source ✓
- §7: workspace `feature-readme-4repos-2026-05-13` có đủ 00→05+08 ✓
- §11: APPEND-only progress, không overwrite memory cũ ✓
- §12: README.md là docs, không phải source code → không vi phạm Brain code prohibition ✓
- §14: file vật lý xác nhận tồn tại + line count đúng ✓

## 2026-05-13 — Commit

- Boss verb: "làm đi. dư gì thì tính sau. mấy file readme thôi mà" → cho phép commit.
- Commit `ef37ad1` on `main`: 4 files changed, 552 insertions(+), 244 deletions(-).
  - A `cdc-auth-service/README.md`
  - A `cdc-cms-service/README.md`
  - M `cdc-cms-web/README.md` (overwrite Vite template)
  - M `centralized-data-service/readme.md` (case-insensitive FS: README.md ⇄ readme.md cùng inode)
- KHÔNG push remote (CLAUDE.md §8 — chờ verb riêng).
