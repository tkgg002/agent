# 08 Tasks Checklist (Sub-Task Execution Unit): TASK-11-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)
- **Parent Task:** `TASK-11` | **Wave:** `Wave 1` | **Isolated Branch:** `task/TASK-11/task-11-m04`
- [ ] **TASK-11-M04 (cdc-cms-web):**
  - **Context (`file:line`):** `cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)`
  - **Input Contract (`UpstreamReady=false`):** Kết nối dây dẫn nghiệp vụ tại `src/hooks/useAuditStatus.ts` (`export function useAuditStatus() {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-web`.
  - **Output Contract (`HandoffReady=false`):** Bàn giao cho `TASK-11-M05, TASK-11-M06` (Biên lai `EV-SUB-TASK-11-M04-PENDING`)
  - **Definition of Done:** npm run build && npm test
