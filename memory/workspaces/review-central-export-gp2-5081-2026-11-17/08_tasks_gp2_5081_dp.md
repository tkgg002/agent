# 08. Tasks — Design Pattern & Architecture Audit `feat/GP2-5081 -> develop` (`7f4e9a7`)

## TDP-01: Mid-Session Self-Improvement Loop (Rule #5 & #6)
- **Context:** Bổ sung bài học Global về bắt buộc kiểm toán Design Pattern, chiều phụ thuộc tầng và Template Method hooks khi review code vào `memory/global/lessons.md`.
- **Logs/Test:** Kiểm tra `wc -l` trước/sau (`767 -> 774`) và chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py`.
- **Definition of Done:** `lessons.md` và `INDEX.md` đồng bộ `100/100/100` đạt chuẩn Global Hygiene.

## TDP-02: Quét Toàn Bộ Codebase Để Đối Chiếu 5 Trục Design Pattern (`DP1`–`DP5`) & Chạy `DG0`
- **Context:** Đối chiếu `feat/GP2-5081` với `README.md:11,104-118`, `BaseExportProcessor.ts:24-36,158,219-239`, `payout-request.export.ts:14-18`, `payout-request-export.pure.ts:22-70`, `payment-history-export.pure.ts:134-160`, `trans-his-export.pure.ts:211-233`, `trans-his-collection-export.pure.ts:190`, `internal-transfer-export.pure.ts:168`, và toàn bộ `domain/handlers/**`.
- **Logs/Test:** `grep -rn "logics/" domain/`, `grep -rn "subQueryClass" logics/export/`, `grep -rn "mainProcess.models." domain/handlers/`, chạy `laya_gate.py run --stage dg0`.
- **Definition of Done:** Xác định đầy đủ `DP1`–`DP5` kèm bằng chứng `file:line` xác thực 100%.

## TDP-03: Thiết Kế Kiến Trúc Chuẩn Pattern (`09_tasks_solution_gp2_5081_dp.md`), Chạy `DG1`/`DG3` & Xuất `audit_report_gp2_5081_dp.md`
- **Context:** Lập phương án tái cấu trúc chuẩn Clean Architecture + CQRS + Template Method (kèm Code Demo đầy đủ), kiểm định qua `laya_gate.py` (`dg1`, `dg3`) và `verify_governance.py --workspace review-central-export-gp2-5081-2026-11-17 --suffix gp2_5081_dp`.
- **Logs/Test:** `laya_gate.py`, `evidence_run.py` (xác thực trạng thái repo sạch), `verify_governance.py`.
- **Definition of Done:** Toàn bộ cổng `PASS` / `PASSED 🟢`; báo cáo trung thực, cầu thị và trình Code Demo chờ lệnh `APPROVE` của anh.
