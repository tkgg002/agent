---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T10:53:00+07:00
---

# 08. Tasks — Audit phản biện nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

- [x] **TASK-1 (S2 Investigate — Git Diff & Commit History Critique):**
  - **Context:** `/Users/trainguyen/Documents/work/centralized-export-service` nhánh `feat/trans-his-postgres-export` so với `develop`.
  - **Action:** Thu thập `git log develop..HEAD`, `git diff develop...HEAD --stat`, `wc -l` trên toàn bộ file thay đổi/thêm mới, và đọc từng dòng diff (`line-by-line`) đối chiếu với Design Pattern gốc trên `develop`.
  - **Logs/Test:** Kiểm tra cấu trúc layer (`defined/`, `dbHandler/`, `data-transfers/entities/`, `data-transfers/params/`, `models/`, `domain/queries/`, `domain/handlers/`, `logics/export/`, `logics/processors/`), trách nhiệm đơn (SRP), và tương thích ngược.
  - **DoD:** Nắm trọn 100% thay đổi từng dòng code và từng commit trên `feat/trans-his-postgres-export`.

- [x] **TASK-2 (S2 Investigate — Physical Build, Lint & Test Verification via `evidence_run.py`):**
  - **Context:** `package.json`, `tsconfig.json`, `vitest.config.ts`, `test/` trong `/Users/trainguyen/Documents/work/centralized-export-service`.
  - **Action:** Chạy `tsc --noEmit`, `npm run lint`, và `vitest run` qua `evidence_run.py` để lấy biên lai vật lý `EV-...` (không can thiệp source code).
  - **Logs/Test:** Biên lai `EV-20261103T105759-fb0c`, `EV-20261103T105700-86a1`, `EV-20261103T105705-3a2b`, `EV-20261103T105725-23d3`, `EV-20261103T105729-22f1`, `EV-20261103T105710-0538`, `EV-20261103T105803-0a6d`, `EV-20261103T110325-0d83` tại `06_evidence/`.
  - **DoD:** Có đầy đủ biên lai thực tế chứng minh trạng thái biên dịch, lint, unit test và regression test hiện tại của nhánh `feat/trans-his-postgres-export`.

- [x] **TASK-3 (S2.5 Pre-Plan Adversarial Critique — DG0):**
  - **Context:** Các phát hiện từ S2 và các giả định kiến trúc trong nhánh `feat/trans-his-postgres-export`.
  - **Action:** Lập `claims_dg0.json` và chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg0`.
  - **Logs/Test:** `06_laya_summary_dg0_20261103T110447.json` đạt `PASS`.
  - **DoD:** Kiểm định xong tính khả thi và tính xác thực của các phát hiện trước khi lập giải pháp S3.

- [x] **TASK-4 (S3 Single-Best Solution & DG1/DG2 Verification):**
  - **Context:** Toàn bộ phát hiện `F1..F6` trên nhánh `feat/trans-his-postgres-export`.
  - **Action:** Thiết kế ĐÚNG 1 phương án tối ưu nhất ánh xạ 1-1 theo từng phát hiện `F1..F6` kèm **Code Demo** nguyên văn từ HEAD (`L004`) vào `09_tasks_solution_trans_his_postgres_export.md`; chạy `laya_gate.py run --stage dg1`.
  - **Logs/Test:** `06_laya_summary_dg1_20261103T110837.json` & `06_laya_summary_dg2_20261103T110837.json` đạt `PASS`.
  - **DoD:** Hồ sơ giải pháp hoàn chỉnh, chính xác từng dòng code và qua cổng DG1/DG2.

- [x] **TASK-5 (S5/DG3 Audit Report & Propose-Only Presentation):**
  - **Context:** `audit_report_trans_his_postgres_export.md` và `05_progress_trans_his_postgres_export.md`.
  - **Action:** Chạy `laya_gate.py run --stage dg3`, xuất `audit_report_trans_his_postgres_export.md` đủ 6 mục chuẩn + Bảng Tổng Hợp Issues (`L006`), trình bày trực tiếp kết quả Audit + Code Demo trên khung chat và dừng chờ User `APPROVE`.
  - **Logs/Test:** `06_laya_summary_dg3_20261103T110905.json` đạt `PASS`.
  - **DoD:** Báo cáo đầy đủ, sâu sắc, trung thực 100%, giữ nguyên chế độ Read-Only trên repo đích chờ `APPROVE`.

- [x] **TASK-6 (S4 Execute — Implement `S-FIX-F1`..`S-FIX-F6` & Unit Tests):**
  - **Context:** `models/trans-his-pg/*`, `utils/trans-his-export-source.ts`, `dbHandler/trans-his-pg.ts`, `start.ts`, `logics/health-check.logic.ts`, `domain/handlers/trans-his/GetListTransHisByTransIdPgHandler.ts`, `logics/export/merchant/payout-request-export.pure.ts`, `logics/processors/BaseExportProcessor.ts`, `test/integration/trans-his-pg.integration.test.ts`, `test/unit/trans-his-pg/*`.
  - **Action:** Tách module SRP (`trans-his-pg.regex.ts`, `trans-his-pg.sort.ts`, `trans-his-pg.numeric.ts`), chuẩn hóa layer cấu hình Postgres về `utils/trans-his-export-source.ts`, gỡ kết nối thừa Mongo `trans-his` khi chạy `postgres` mà không cấu hình `TRANS_HIS_MONGO_URI`, lọc `listTransId` rỗng + rethrow lỗi sub-query, cache `hasIdempotencyKeyPromise`, sửa ESLint `start.ts` và thêm `console.warn` trong `trans-his-pg.integration.test.ts`.
  - **Logs/Test:** Bổ sung unit test cho `F3`, `F4`, `F5` và chạy `tsc` (`EV-20261103T113403-b34e`), `eslint` (`EV-20261103T113358-97a0`), `vitest` (`EV-20261103T113336-1859` — 114/114 PASS), `vitest.integration` (`EV-20261103T113239-b095`) qua `evidence_run.py`.
  - **DoD:** 100% file `< 500` dòng (`max 340` dòng ở `trans-his-pg.query.ts`), `0` lỗi ESLint trên toàn bộ 23 file chạm tới, `114/114` unit tests PASS.

- [x] **TASK-7 (S5 Final Adversarial Audit & S6 Close):**
  - **Context:** `git diff`, `wc -l`, `claims_dg3.json`, `audit_report_trans_his_postgres_export.md`, `05_progress_trans_his_postgres_export.md`.
  - **Action:** Kiểm tra lại từng dòng diff sau lần sửa cuối, đo `wc -l`, chạy `laya_gate.py run --stage dg3`, cập nhật `audit_report_trans_his_postgres_export.md`, append `DONE` vào `05_progress_trans_his_postgres_export.md`, và chạy `verify_governance.py`.
  - **Logs/Test:** `06_laya_summary_dg3_20261103T113449.json` (`PASS`) và `verify_governance.py` (`PASSED` 🟢).
  - **DoD:** Đạt 8 cổng chất lượng `G1–G8` và `verify_governance.py` `PASSED`.
