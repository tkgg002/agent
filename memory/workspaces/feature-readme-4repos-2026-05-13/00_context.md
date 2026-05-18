# 00 Context — feature-readme-4repos-2026-05-13

> Created: 2026-05-13 (ICT)
> Owner: Muscle (CC CLI) — Chief Engineer
> Lane: docs (không sửa source code, chỉ touch README.md)

## Trigger
Boss directive: "cập nhật README.md cho 4 repo chính."

## Repo scope
1. `cdc-auth-service` (Go 1.26.1, Fiber, JWT). README.md hiện CHƯA tồn tại.
2. `cdc-cms-service` (Go 1.26.1, Fiber + NATS + Redis, hexagonal). README.md hiện CHƯA tồn tại.
3. `cdc-cms-web` (TypeScript + React + Vite). README.md hiện đang là default Vite template.
4. `centralized-data-service` (Go 1.26.1, Gin + GORM, 4 binary: worker/admin-api/sinkworker/profile_table). Hiện có `readme.md` (lowercase) — cần dựng `README.md` chuẩn.

## Constraints
- CLAUDE.md §12: README.md là docs → không vi phạm "Brain code prohibition" (đây là Muscle thi công).
- CLAUDE.md §11: KHÔNG được overwrite memory files — chỉ APPEND `05_progress.md`.
- README per-repo phải self-contained: ai clone repo lẻ vẫn build chạy được.
- Cross-link về `architecture.md` (root) cho big-picture.

## Source data
- `agent/memory/global/project_context.md` (snapshot 2026-05-04).
- `agent/memory/global/tech_stack.md`.
- `architecture.md` (root cdc-system).
- Makefile / cmd / config / docs / package.json của mỗi repo.
