# 08. Tasks — fix-goopay-npm-403-2026-11-04

## Danh sách Công việc (Autonomous Full-Stack Tasks)

- [ ] **TASK-01 (S2 Investigate & S2.5 DG0 Critique):**
  - **Mô tả + Context `file:line`:** Kiểm tra hồ sơ `memory/workspaces/task-goopay-dependency-analysis/findings.md`, cấu hình `.npmrc` / `.yarnrc` / `package.json` / `yarn.lock` tại `/Users/trainguyen/Documents/work/rule-service` và các service liên quan (`customer-service`, `auth-service`, `mobile-service`), kiểm tra HTTP response từ `https://npm.goopay.vn/@goopay%2fgoopay-library`, kiểm tra `yarn cache dir` và các bản `@goopay/goopay-library` đang có sẵn trong `node_modules` hoặc local cache trên máy.
  - **Logs/Test:** Chạy `curl -I` / kiểm tra `.npmrc` (mask token), kiểm tra `yarn cache list` và chạy cổng phản biện `laya_gate.py run --stage dg0`.
  - **Definition of Done (DoD):** Xác định rõ `VERIFIED` root cause vì sao `https://npm.goopay.vn/@goopay%2fgoopay-library` trả về `403` và xác định nguồn gói `@goopay/goopay-library` hợp lệ sẵn có trên máy; `DG0` đạt `PASS`.

- [ ] **TASK-02 (S3 Solution & DG1/DG2 Verification):**
  - **Mô tả + Context `file:line`:** Thiết kế 1 phương án tối ưu nhất ánh xạ 1-1 với các phát hiện `F1..F5` trong `09_tasks_solution_goopay_npm_403_20261104.md` kèm Code Demo trực tiếp trên khung chat (bao gồm cập nhật `MONGO_URI` trong `/Users/trainguyen/Documents/work/config-service/.run.local.env:7` từ `localhost:27020` đang bị `ECONNREFUSED` sang cụm `10.200.186.11..13` giống `/Users/trainguyen/Documents/work/centralized-export-service/.run.local.env:59` để tránh `this.stopService(this.broker, "STOP_SERVICE_ERR_CONNECT_DB")` -> `broker.destroyService` tại `/Users/trainguyen/Documents/work/config-service/start.ts:450,503`).
  - **Logs/Test:** Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1`.
  - **Definition of Done (DoD):** `09_tasks_solution_goopay_npm_403_20261104.md` hoàn chỉnh, `DG1` đạt `PASS`, ghi log vào `05_progress_goopay_npm_403_20261104.md`, trình bày cho User và chờ `APPROVE`.
