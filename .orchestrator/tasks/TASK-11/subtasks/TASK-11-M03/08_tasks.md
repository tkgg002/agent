# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-11/task-11-m03`
- [ ] **TASK-11-M03 (cdc-cms-web):**
  - **Context (`file:line`):** `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)`
  - **Input Contract (`UpstreamReady=false`):** Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `src/constants/postgresDataTypes.ts` (`export const POSTGRES_DATA_TYPES = [`) trong worktree cô lập của `cdc-cms-web`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-11-M05, TASK-11-M06` (Biên lai `EV-SUB-TASK-11-M03-PENDING`)
  - **Definition of Done:** npm run build && npm test
