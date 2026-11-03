---
tier: T2
suffix: auto_recon_export
workspace: audit-auto-recon-export-2026-11-02
created_at: 2026-11-02T13:42:00+07:00
---

# 01. Requirements — Audit phản biện nhánh `feat/auto-recon-export` so với `develop` (`centralized-export-service`)

## 1. Yêu cầu của User
- Audit lại toàn bộ "quá trình" vừa thực hiện ở diff trên nhánh `feat/auto-recon-export` (sau commit `699a833`) so với nhánh `develop` trên repository `/Users/trainguyen/Documents/work/centralized-export-service`.
- Sử dụng tư duy phản biện cho:
  1. Các task đã làm theo Plan sửa 8 bước (`F1`–`F6`, `F9`–`F10`, `Test`, cùng 2 điểm chờ xác nhận `F7`, `F8`).
  2. Từng file đã update/thêm mới trên `develop..HEAD` và riêng commit `699a833`.
  3. Từng dòng update (`line-by-line diff critique`).
- Kiểm tra xem code mới còn lỗi/sơ hở nào không so với `develop`, đảm bảo:
  - Mọi thay đổi đặt đúng layer theo Design Pattern gốc của `centralized-export-service`.
  - Tách file/module theo đúng trách nhiệm đơn (Single-Responsibility), tuyệt đối không nhồi nhét thêm vào các file lớn (`God-files`).
  - Các chức năng export cũ không bị lỗi và tương thích ngược 100%.
- Giữ chế độ **Propose-Only** (Rule #1, #9, #13, `L009`): Lập báo cáo Audit + nếu còn phát hiện sơ hở (`F_NEW_*` hoặc phần làm thiếu/sai so với Plan/Design Pattern) thì thiết kế phương án duy nhất tại `09_tasks_solution_auto_recon_export.md` kèm **Code Demo** trực tiếp trên khung chat và **DỪNG CHỜ `APPROVE`**.

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ Lessons `INDEX.md`)
- **`L002` & `L026` (Direct-Answer Summary First, Executive Issues Summary Table & 1-to-1 Finding Mapping):** Mở đầu câu trả lời bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi chính; lập **Bảng Tổng Hợp Issues (Executive Summary Table)** tách rõ nhóm *Còn tồn tại (Open/Pending)* lên trước, nhóm *Đã xử lý (Resolved)* xuống sau; ánh xạ giải pháp 1-1 theo từng phát hiện.
- **`L003` & `L022` (No Silent Live-DB Test Skip & Zero-Mockup Evidence):** Kiểm tra kỹ test tích hợp ClickHouse (`RUN_CLICKHOUSE_EXPORT_TESTS`) có bị skip ngầm hoặc che giấu lỗi thực thi không; phân biệt rõ kết quả Unit Test (mock) với Runtime/Live ClickHouse thật.
- **`L008` & `L024` (Explicit Runtime Call-Site & End-to-End Callchain Tracing):** Truy vết đầy đủ callchain từ `start.ts` -> `DbConnectorFactory` -> `MainProcess` -> `ExportJobProcessor` / `BaseExportProcessor` -> `pures/auto-reconcile` -> `CQRS Query & Handler` (`GetAllMerchantReconcileTransactionsExportHandler`, `GetAllMerchantFeeTotalsByCodeExportHandler`) -> `Models` (`MerchantReconcileResultModel`, `MerchantFeeOrderModel`) -> ClickHouse Driver (`clickhouse.ts`).
- **`L009` (Propose-Only Discipline & Explicit Approval Gate — `LS-GLOBAL-001-propose-only-gate`):** Giữ nguyên chế độ Read-Only đối với source code của `centralized-export-service`, xuất đầy đủ `09_tasks_solution_auto_recon_export.md` + `audit_report_auto_recon_export.md`, hiển thị Code Demo trực tiếp trên chat và dừng chờ `APPROVE`.
- **`L012` (Single-Responsibility Module Split & Line-Budget Guard):** Đo `wc -l` thực tế của toàn bộ các file chạm tới so với `develop` (`BaseExportProcessor.ts`, `DbConnectorFactory.ts`, `start.ts`, barrel files, model/handler/pure) để kiểm chứng có nhồi nhét vào God-file hay đặt sai layer không.
- **`L015` & `L016` (Pre-Plan Adversarial Critique DG0 & Cross-Validation DG1/DG3):** Chạy cổng `/laya_router` (`dg0`, `dg1`, `dg3`) có đầy đủ `REF: <path>:<start>-<end>` và đối chứng âm/dương (`control_false` / `control_true`).
