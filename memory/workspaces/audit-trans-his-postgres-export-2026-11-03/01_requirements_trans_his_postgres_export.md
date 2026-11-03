---
tier: T2
suffix: trans_his_postgres_export
workspace: audit-trans-his-postgres-export-2026-11-03
created_at: 2026-11-03T10:53:00+07:00
---

# 01. Requirements — Audit phản biện nhánh `feat/trans-his-postgres-export` so với `develop` (`centralized-export-service`)

## 1. Yêu cầu của User
- Audit lại toàn bộ "quá trình" vừa thực hiện ở diff trên nhánh `feat/trans-his-postgres-export` (thêm kết nối PostgreSQL / export `trans-his` từ Postgres) so với nhánh `develop` trên repository `/Users/trainguyen/Documents/work/centralized-export-service`.
- Sử dụng tư duy phản biện cho:
  1. Các task/bước đã làm qua chuỗi commit trên nhánh `feat/trans-his-postgres-export` so với `develop`.
  2. Từng file đã update/thêm mới trên `develop...HEAD`.
  3. Từng dòng update (`line-by-line diff critique`).
- Kiểm tra xem code mới còn lỗi/sơ hở nào không so với `develop`, đảm bảo:
  - Mọi thay đổi đặt đúng layer theo Design Pattern gốc của `centralized-export-service` (`defined/`, `dbHandler/`, `data-transfers/entities/`, `data-transfers/params/`, `models/`, `domain/queries/`, `domain/handlers/`, `logics/export/`, `logics/processors/`).
  - Tách file/module theo đúng trách nhiệm đơn (Single-Responsibility), tuyệt đối không nhồi nhét thêm vào các file lớn (`God-files`).
  - Các chức năng export cũ (MongoDB, MariaDB, v.v.) không bị lỗi và tương thích ngược 100%.
- Giữ chế độ **Propose-Only** (Rule #1, #9, #13, `L013`): Lập báo cáo Audit tại `audit_report_trans_his_postgres_export.md` + thiết kế phương án duy nhất tại `09_tasks_solution_trans_his_postgres_export.md` kèm **Code Demo** nguyên văn từ HEAD (`L004`) trực tiếp trên khung chat và **DỪNG CHỜ `APPROVE`**.

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ Lessons `INDEX.md`)
- **`L001` (Branch-Aware Repository Resolution & Branch Mismatch Critique Gate):** Khóa chính xác repository `/Users/trainguyen/Documents/work/centralized-export-service` tại nhánh `feat/trans-his-postgres-export`, nạp trực tiếp tập thay đổi thực tế `git diff develop...HEAD` và lịch sử commit `git log develop..HEAD` vào bước điều tra `S2` và phản biện `DG0`.
- **`L004` (Verbatim Current-File Code Demo & Cross-Layer Default Parameter Check):** Mọi khối trích dẫn và Code Demo bắt buộc tạo trực tiếp từ các dòng văn bản nguyên văn của tệp tại HEAD hiện hành; kiểm tra giá trị mặc định của tham số xuyên suốt từ tầng gọi xuống tầng thực thi.
- **`L006` & `L030` (Direct-Answer Summary First, Executive Issues Summary Table & 1-to-1 Finding Mapping):** Mở đầu câu trả lời bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi chính; lập **Bảng Tổng Hợp Issues (Executive Summary Table)** tách rõ nhóm *Còn tồn tại (Open/Pending)* lên trước, nhóm *Đã xử lý / Đạt chuẩn (Resolved/Verified)* xuống sau; ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **`L007` & `L026` (No Silent Live-DB Test Skip & Zero-Mockup Evidence):** Kiểm tra kỹ các bài unit test / integration test mới có che giấu lỗi, giả lập sai lệch với PostgreSQL runtime thật, hoặc bị skip ngầm không; chạy compiler (`tsc`), linter (`npm run lint`), và test (`vitest`) qua `evidence_run.py` để lấy biên lai vật lý `EV-...`.
- **`L012` & `L028` (Explicit Runtime Call-Site & End-to-End Callchain Tracing):** Truy vết đầy đủ callchain từ `start.ts` -> `DbConnectorFactory` -> `postgres.ts` / driver -> `MainProcess` -> `ExportJobProcessor` / `BaseExportProcessor` -> `logics/export/...` -> `CQRS Query & Handler` -> `Models` -> PostgreSQL query execution, đối chiếu tương thích ngược với các luồng export hiện hữu trên `develop`.
- **`L013` (Propose-Only Discipline & Explicit Approval Gate — `LS-GLOBAL-001-propose-only-gate`):** Giữ nguyên chế độ 100% Read-Only đối với source code của `centralized-export-service`, xuất đầy đủ `09_tasks_solution_trans_his_postgres_export.md` + `audit_report_trans_his_postgres_export.md`, hiển thị Code Demo trực tiếp trên chat và dừng chờ `APPROVE`.
- **`L016` (Single-Responsibility Module Split & Line-Budget Guard):** Đo `wc -l` thực tế của toàn bộ các file chạm tới so với `develop` để kiểm chứng có nhồi nhét vào God-file hay đặt sai layer không.
- **`L019` & `L020` (Pre-Plan Adversarial Critique DG0 & Cross-Validation DG1/DG3):** Chạy cổng `/laya_router` (`dg0`, `dg1`, `dg3`) có đầy đủ `REF: <path>:<start>-<end>` và đối chứng âm/dương (`control_false` / `control_true`).
