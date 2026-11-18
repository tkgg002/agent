# 01. Requirements — Final Pass Code Review `feat/GP2-5081 -> develop` (`92ebb72`, Pass 3)

tier: T2
workspace: review-central-export-gp2-5081-2026-11-17
suffix: gp2_5081_v3
mode: audit-propose-only
project: centralized-export-service
repo_path: /Users/trainguyen/Documents/work/centralized-export-service
source_branch: origin/feat/GP2-5081 (92ebb728c008808618b56e93c316c803416fcc02)
target_branch: origin/develop (3eef80790b319e4d0f188e3d12cef9f0ac90b302)

## 1. Yêu cầu của User (Requirements)
- **R1 (Verify Design Pattern & Residual Fixes on `92ebb72`):** Rà soát toàn diện bản cập nhật mới nhất của dev tại commit `92ebb72` (`19 files changed, +1262 / -5` so với `origin/develop`) qua đủ 4 lăng kính: **Design Pattern / Clean Architecture (`DP1`–`DP5`)**, **Logic / SQL / Partial Index (`N1`–`N3`)**, **Contract & Regression trên các export cũ (`GetAllTransHisExportHandler`, `trans-his-pg.query.ts`)**, và **Security**.
- **R2 (Real Test, Build & ESLint Execution):** Thực thi toàn bộ unit tests, integration tests, `tsc --noEmit` và `eslint` trên nhánh `92ebb72` qua `evidence_run.py` lấy biên lai `EV-...` ngày `2026-11-18`.
- **R3 (Propose-Only & Decision Gate):** Giữ chế độ *Audit Propose-Only* (không tự sửa source code), kiểm định qua `laya_gate.py` (`DG0`, `DG1`/`DG2`, `DG3`), lưu báo cáo vật lý và kết luận rõ ràng cho anh xem MR đã đạt điều kiện `APPROVE` để merge vào `develop` chưa.

## 2. Ràng buộc Thực thi Trích xuất từ Global Lessons (Technical Constraints)
- **TC1 (`L002` / Design Pattern Audit):** Kiểm tra chặt `grep -rn "logics/" domain/` phải bằng `0`, `IQuery` bất biến 100%, `BaseExportProcessor` hooks (`async getConfig(..., queryExecutor)` + `subQueryClass`/`buildSubQueryParams`/`mergeData`) đấu nối đúng contract, không làm ảnh hưởng 3 export `trans-his` cũ khi sửa `GetAllTransHisExportHandler.ts`.
- **TC2 (`L078` — Partial Index & Bounded Queries):** Kiểm tra `isDelete: { $ne: true }` -> `core.is_delete IS DISTINCT FROM TRUE` đã khớp Partial Index `idx_th_cd_flow_created_at_id` chưa, và kiểm tra `chunkSize` / `limit` của `GetPaymentBillExportAuxiliaryHandler`.
- **TC3 (`L080` & `L082` — Schema & Type Coercion):** Kiểm tra mapping dữ liệu giữa `GetAllTransHisExportHandler` / `GetAllTransHisExportPgHandler` -> `mergeData` -> `transformRow`, cũng như `SELECT_FIELDS` trong `normalizeSelect` của `TransHisPgModel`.
- **TC4 (`G1`–`G4`, `L044`):** Không overwrite memory, không sửa source code trước `APPROVE`, tách lệnh CLI sạch, kèm biên lai `EV-...` thật.
