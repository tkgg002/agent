---
tier: Audit
mode: diff-review
suffix: gp2_5072
repository: /Users/trainguyen/Documents/work/centralized-export-service
base: origin/develop (3eef807)
head: origin/feature/GP2-5072-partner-disbursement-export (c1f36d3)
merge_base: cb27f46
created_at: 2026-11-17T16:47:01+07:00
agent: Claude Fable 5.1 (OpenCode)
---

# 01. Requirements — `/diff-review` GP2-5072 Partner Disbursement Export

## 1. Yêu cầu gốc của User
- Lệnh: `/diff-review --repo centralized-export-service --base origin/develop --head feature/GP2-5072-partner-disbursement-export --workspace review-export-GP2-5072`.
- Lượt 1 User gõ `GP2-5072-partner-disbursement-export` (không có `feature/`), nhánh không tồn tại local/remote; User đã sửa lại thành `feature/GP2-5072-partner-disbursement-export` (VERIFIED `git ls-remote` → `c1f36d3`).
- Chế độ: **Propose-Only / Read-Only Audit**. Working tree đang ở `feat/GP2-5081` (sạch) — KHÔNG checkout, chỉ `git diff/show` theo ref.

## 2. Sự thật đã xác minh (S0/S2 sơ bộ)
- `origin/develop..head` = 7 commit: `c1f36d3` (GP2-5072) + 6 commit kế thừa từ `origin/testing-1.10.0` (GP2-5068 `672d87c`, Node 24.22.1 bump `6d36760`, các merge develop→testing).
- Diff 3-dot `origin/develop...head`: 36 file, +2645/−6. Riêng commit `c1f36d3`: 17 file, +984/−5.
- Repo KHÔNG có `.agent-project.json` → chỉ nạp L1 base; `L2 fintech` nạp thủ công vì service xử lý giao dịch giải ngân (money path). Không có `review_profile` → Design lens finding layer = `HYPOTHESIS`.

## 3. Phạm vi review
- **Primary (4 lens đầy đủ):** 17 file của commit `c1f36d3` — phần việc GP2-5072.
- **Secondary (ghi nhận, không review sâu):** phần GP2-5068 + Node bump đã nằm trên `testing-1.10.0` — nêu ở mục "Scope note" vì MR target `develop` sẽ kéo kèm chúng.

## 4. Ràng buộc kỹ thuật từ Global Lessons
- **C1 (`L001`):** kiểm `test -e` cấp file trước khi tạo 01/05/08 — đã làm, workspace mới rỗng.
- **C2 (`L037`):** nhánh theo tên user đưa không tồn tại → tra `git branch -a --list '*5072*'` thay vì đoán; không checkout working tree của User.
- **C3 (`L040`):** mọi Quote trích nguyên văn từ `git show c1f36d3:<path>`, không trích từ working tree `feat/GP2-5081`.
- **C4 (`L042`/`L066`):** Direct Summary + Executive Issues Table, finding ánh xạ 1-1 `F1..Fn`.
- **C5 (`L049`):** Propose-Only, không sửa source, không `git push`.
- **C6 (`L062`):** không suy diễn từ unit test mock; test chạy phải qua `evidence_run.py`.
- **C7 (`L004`):** đối chiếu reuse endpoint export tập trung / pattern export hiện có (`logics/export/trans-his/*`).

## 5. Definition of Done
- `scope_gp2_5072.json` (R0), `review_report_gp2_5072.md` (R6) với verdict `BLOCK|REQUEST_CHANGES|APPROVE`.
- Mỗi finding có `REF: <abs>:<s>-<e>` + quote; R5: `laya_gate.py review` + `run --stage dg3` có `06_laya_summary_*`.
- `05_progress` có `START review` và `DONE review verdict=...`; `verify_governance.py` PASSED.
