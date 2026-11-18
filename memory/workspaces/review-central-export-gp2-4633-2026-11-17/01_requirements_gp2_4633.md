---
tier: T2
mode: audit
suffix: gp2_4633
repository: /Users/trainguyen/Documents/work/centralized-export-service
source_branch: origin/feat/GP2-4633
target_branch: origin/develop
created_at: 2026-11-17T11:17:15+07:00
---

# 01. Requirements — Code Review `feat/GP2-4633 -> dev` (`centralized-export-service`)

## 1. Yêu cầu gốc của User
- Review code tại repository `centralized-export-service` cho Merge Request / nhánh `feat/GP2-4633 -> dev` (`origin/feat/GP2-4633` vào `origin/develop`).
- Chế độ thực thi: **Read-Only Code Audit (`Propose-Only`)** — tuyệt đối không tự ý sửa source code hay chuyển nhánh làm ảnh hưởng working tree hiện tại (`feat/GP2-5081`) của User cho đến khi có lệnh `APPROVE`.

## 2. Mục tiêu Kiểm định (`R1` – `R6`)
- **R1 (Architecture & Contract Parity):** Kiểm tra tên `exportType` (`PartnerUserReserveExport`), wiring tại `QueryHandlerFactory`, `domain/queries`, `data-transfers/params`, `logics/export` và đối chiếu contract với CMS / service nguồn (nếu có repo liên quan trên máy).
- **R2 (Query & Filter Correctness):** Kiểm tra tính chính xác của điều kiện lọc (ngày giờ, timezone, khoảng tiền/số dư, regex/exact match, SQL/Mongo query injection, xử lý filter rỗng/unknown keys, thứ tự sort và pagination/chunking).
- **R3 (Data Mapping, Formatting & Edge Cases):** Kiểm tra hàm pure transform (`partner-user-reserve-export.pure.ts`), định dạng tiền tệ/số âm/null/undefined/BigInt/Decimal, timezone khi xuất Excel/CSV, thứ tự cột và tiêu đề cột.
- **R4 (Performance, Memory & Concurrency Safety):** Kiểm tra cơ chế đọc dữ liệu lô lớn (cursor/stream/keyset vs offset pagination, chunk size, nguy cơ OOM hoặc full-table scan trên bảng `trans-his` / `reserve` lớn).
- **R5 (Test Coverage & Anti-Smoke-Speculation):** Kiểm tra chất lượng unit test & integration test trong PR (`partner-user-reserve-export.integration.test.ts`, handler test, pure test) — phát hiện mock hổng, test pass giả hoặc thiếu edge cases.
- **R6 (Security & PII/Secret Hygiene):** Kiểm tra lộ thông tin nhạy cảm, lỗi phân quyền/tenant filter (`partnerId` / `userId`), hoặc lỗ hổng injection.

## 3. Ràng buộc Kỹ thuật trích xuất từ Global Lessons (S0 Boot)
- **C1 (`L001` — Naming & Centralized Endpoint Parity):** Đối chiếu tên `exportType` và contract gọi từ CMS/Gateway xem có tuân thủ quy ước đặt tên chuẩn (số nhiều/số ít theo mã màn hình) và endpoint tập trung hay không.
- **C2 (`L034` & `L048` — Branch-Aware & Worktree Isolation):** Working tree chính của `centralized-export-service` đang đứng ở nhánh `feat/GP2-5081`; mọi thao tác đọc/chạy test của `origin/feat/GP2-4633` phải thực hiện trên Git Worktree cô lập và kiểm chứng lệch tên nhánh đích (`dev` vs `origin/develop`) ở cổng `DG0`.
- **C3 (`L037` — Verbatim Current-File Code Demo):** Mọi trích dẫn `file:line` và Code Demo đề xuất phải lấy nguyên văn từ commit HEAD của `origin/feat/GP2-4633` (`7d9bcf1`), kiểm tra giá trị mặc định xuyên tầng.
- **C4 (`L039` & `L063` — Direct Summary & Executive Issues Table):** Mở đầu phản hồi bằng khối `Direct Summary` 2–4 câu, trình bày `Executive Issues Summary Table` rõ ràng và ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **C5 (`L041` — Clean CLI Args):** Khi gọi `laya_gate.py` / `verify_governance.py`, chạy lệnh đơn lẻ, không nối chuỗi lệnh và không đưa từ khóa nhạy cảm vào `--objective` / `--reason`.
- **C6 (`L074` & `L078` — Large Table Scan & Timezone/Type Coercion):** Kiểm tra index coverage, tránh full-table scan trên bảng lớn và kiểm tra ép kiểu số/thời gian chuẩn xác.
