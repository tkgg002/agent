# 08. Tasks — Audit & Khắc phục Nhánh `feat/auto-recon-export` (`centralized-export-service`)

## Task 1: S2 Line-by-Line Diff & Process Audit (`develop..HEAD` & `e37fe8b..699a833`)
- **Mô tả & Context:**
  - Đối chiếu toàn bộ `git diff develop...HEAD` và `git show 699a833` trong `/Users/trainguyen/Documents/work/centralized-export-service` với 8 bước trong Plan đã đề ra (`F1`..`F10`) và với các export mẫu hiện có trên `develop`.
  - Đo `wc -l` các file thay đổi so với `develop` để kiểm tra nguyên tắc Single-Responsibility (`L012`) và phân tầng kiến trúc (`entities`, `params`, `pures`, `cqrs`, `models`, `processors`, `connectors`).
- **Logs/Test:**
  - Chạy `tsc --noEmit`, `npm run lint`, `npx vitest run` qua `evidence_run.py` để lấy biên lai vật lý `EV-...`.
- **Definition of Done (DoD):**
  - Bảng đối chiếu chi tiết 8 bước của Plan (Đã làm đúng / Làm thiếu / Phát sinh lỗi mới) và phân tích từng file, từng dòng diff kèm `file:line`.

## Task 2: S2.5 & S3 Thiết kế Giải pháp Khắc phục Triệt để các Sơ hở Còn Tồn tại (Propose-Only)
- **Mô tả & Context:**
  - Chạy `laya_gate.py run --stage dg0` kiểm định các phát hiện thực tế từ diff.
  - Thiết kế phương án duy nhất ánh xạ 1-1 cho mọi vấn đề còn tồn tại trong `09_tasks_solution_auto_recon_export.md` + hiển thị Code Demo trực tiếp trên chat.
  - Chạy `laya_gate.py run --stage dg1` và `dg3` lưu biên bản vào `audit_report_auto_recon_export.md`.
- **Logs/Test:**
  - Biên lai `06_laya_summary_dg0_*.json`, `06_laya_summary_dg1_*.json`, `06_laya_summary_dg3_*.json` đạt `PASS`.
- **Definition of Done (DoD):**
  - Hồ sơ `09_tasks_solution_auto_recon_export.md` và `audit_report_auto_recon_export.md` đầy đủ, trình bày Bảng Tổng Hợp Issues + Code Demo trên khung chat, giữ nguyên trạng thái Read-Only trên `centralized-export-service` và chờ lệnh `APPROVE` của User.
