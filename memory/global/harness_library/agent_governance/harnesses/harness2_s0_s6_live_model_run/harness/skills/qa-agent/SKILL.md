---
name: qa-agent
description: >-
  Quality Assurance & Adversarial Review workflow (S5 Self-Audit) — enforces all 8 Definition
  of Done Quality Gates (G1-G8), Anti-Smoke-Test Speculation Gate, runs Decision Gate DG3 via
  laya_gate.py, verifies runtime and edge cases, and writes physical validation evidence into
  audit_report_*.md and 06_validation_*.md.
---

# QA Agent Skill (Definition of Done Gates G1–G8, Anti-Smoke-Speculation & DG3)

> **Quy tắc cốt lõi:** Rule #9 (S5 Self-Audit — Propose-Only), Rule #10 (Decision Gate `DG3`), Rule #12 & #14 (Quality Gates `G1–G8` & Chống Suy Diễn Smoke Test), Rule #18 (Final Adversarial Audit), Rule #22 (Process Linter).

## Khi nào sử dụng
- Ở chặng **S5 SELF-AUDIT** sau khi Muscle đã hoàn thành thay đổi code/config ở S4 (Hook `H6` sẽ tự động nhắc thực thi ngay sau mỗi lần sửa code).
- Hoặc khi User yêu cầu kiểm thử, nghiệm thu chất lượng feature/fix.

## 8 Cổng Chất Lượng Bắt Buộc (Gates G1–G8 — Rule #14)

Thiếu bất kỳ cổng nào dưới đây = **CHƯA DONE**:

### (G1) Truy vết Yêu cầu (Requirement Traceability)
- Mở `01_requirements_<suffix>.md`, lập bảng đối chiếu **từng yêu cầu (`R1..Rn`)** với file/dòng code thực tế và kết quả test tương ứng.
- Nếu có bất kỳ thay đổi phạm vi nào → phải có mục ghi nhận trong `04_decisions_<suffix>.md`, tuyệt đối không âm thầm bỏ sót.

### (G2) Reproduce trước khi Fix (Red → Green)
- Với mọi task sửa lỗi (bugfix): xác nhận có test hoặc kịch bản chứng minh lỗi xảy ra trước khi sửa (Red) và cùng kịch bản đó đạt PASS sau khi sửa (Green).

### (G3) Test thật & Runtime thật — CẤM Suy Diễn Smoke Test (Anti-Smoke-Speculation)
- Chạy toàn bộ `checks` và `test` của component theo `<project-root>/.agent-project.json`.
- Chạy `runtime_check` (nếu component có khai báo): xác nhận tiến trình mới thực sự đang chạy (PID mới, port đang listen) và thực hiện request/luồng dữ liệu thật qua tiến trình đó.
- **Ma trận Chống Suy Diễn Smoke Test (L001, L044, L056, L061 — BẮT BUỘC KIỂM TRA):**
  1. **Cấm Mock/Closure giả trong Production Code (L001/L056):** Nghiêm cấm tiêm hàm giả (`*Fn func(`, `lookupMasterTargetsFn`, `runEventsFn`, `sqlmock`, `fakemodel`) vào production struct hoặc chỉ khởi tạo struct literal trong RAM để ép pass unit test rồi suy diễn luồng thật chạy đúng. (Hook `H4` chặn cứng).
  2. **Cấm Smoke Script rời / Build-OK = Done:** Không dùng script rời bên ngoài (`scratch/test_...`, `/tmp/test_...`) hoặc `"Compile OK / Build OK / /health = 200"` để báo Done.
  3. **Cấm Suy diễn từ Happy-Path đơn lẻ (`1 msg < 30s`, `Worker = 1`):** Với hệ thống bất đồng bộ / batching / consumer / worker, bắt buộc kiểm tra 3 trục thực tế:
     - **Concurrency / Multi-worker:** Gửi đồng thời nhiều message/worker (`WorkerCount > 1`) kiểm tra Race Condition, Head-of-Line (HoL) blocking, và `TableDebouncer` gộp lô.
     - **Timeout / Heartbeat:** Kiểm tra khi tác vụ chạy lâu hơn `AckWait` (`msg.InProgress()` heartbeat vs `context.WithTimeout` hủy sớm — L044).
     - **Data Continuity / Outage Recovery:** Kiểm tra tính liên tục của `Watermark` / `Cursor` qua khoảng đứt gãy (Outage gap) và idempotency khi replay (L061).
  4. **Cấm Hallucination kết quả / Phối hợp Trace Runtime thật:** Nếu thiếu môi trường hoặc dữ liệu thật (Production DB, JetStream stream thật, config/credential runtime), **BẮT BUỘC nói rõ và đề nghị phối hợp với User để trace runtime value thật**, tuyệt đối không tự bịa output hoặc dùng container rỗng thay thế.
  5. **Cổng Kiểm thử Đột biến 3 Bản Hỏng (`tooling/mutation_gate.py` — Chống "Test Phế" L039):** Với mọi task `mode: clean-room` (hoặc khi tạo bộ test mới), bắt buộc chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py --workspace <ws> --suffix <suffix> --mutations <mutations.json>` với tối thiểu 3 biến thể cố tình làm hỏng logic (`M1`, `M2`, `M3`). Chỉ đạt khi code chuẩn `PASS` test và cả $\ge 3$ bản hỏng đều bị test đánh trượt (`KILLED`, `survived_count == 0`, `timeout_count == 0`, `all_restored_sha256 == True`).

### (G4) Edge-Case & Negative-Path
Kiểm tra và có bằng chứng test cho các trường hợp biên:
- Input `null`, rỗng (`""`, `[]`, `{}`), sai kiểu dữ liệu, giá trị biên (`0`, âm, vượt ngưỡng).
- Bản ghi trùng lặp (idempotency / duplicate event).
- Lỗi kết nối DB / mạng / timeout / thiếu quyền.
- Đảm bảo mọi nhánh lỗi đều trả về hoặc log rõ ràng — **CẤM nuốt lỗi (silent failure / silent drop)**.

### (G5) Chống Regression (Minimal-Impact)
- Chạy lại toàn bộ test suite hiện có của component bị sửa.
- Nếu sửa hàm/API/schema/event dùng chung → grep toàn bộ caller/consumer và xác nhận không phá vỡ luồng hiện có.

### (G6) Output Correctness trên Dữ liệu Thật/Sample
- Kiểm tra **giá trị nội dung** của output (response JSON, bản ghi DB, file xuất ra) khớp kỳ vọng.
- Với các tác vụ xử lý tập dữ liệu/batch/migration: đối soát số lượng tuyệt đối (`count(nguồn) == count(đích)`).

### (G7) Adversarial Self-Review, Decision Gate `DG3` & `audit_report_<suffix>.md`
1. Tự review toàn bộ `git diff` từng dòng với tâm thế "tìm cách phá mã" (*Một Staff Engineer có duyệt PR này không?*).
2. Chạy Decision Gate `DG3` với đầy đủ context:
   - **Mode A (Kiểm chứng Requirement & Finding):**
     ```bash
     python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run \
       --stage dg3 --workspace <workspace_name> --claims <dg3_claims.json>
     ```
   - **Mode B (Agentic Debate Review trên `git diff -U30`):**
     ```bash
     python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py review \
       --workspace <workspace_name> --repo <repo_path>
     ```
3. Lập báo cáo **`audit_report_<suffix>.md`** theo đúng cấu trúc 6 phần bắt buộc bên dưới (Hook `H7` và `verify_governance.py` sẽ chặn đóng task nếu thiếu file này hoặc thiếu bất kỳ phần nào trong 6 phần).
4. **Kỷ luật Tự Khắc Phục Khép Kín `S5 → S3 → S4 → S5` (Enforced bởi Hook `H4b` & `H7`):** Khi đã bước vào chặng `S5` (đã ghi `audit_report_<suffix>.md`), nếu phát hiện bất kỳ bug, lỗ hổng bảo mật hay thiếu sót so với Plan ban đầu, **CẤM sửa mò thẳng source code khi chưa cập nhật Plan** và **CẤM dừng lại bắt User `APPROVE` lần 2**. Bắt buộc thực hiện khép kín ngay trong cùng lượt: (1) ghi `lessons.md` + chạy `build_lessons_index.py`, (2) quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` và chạy lại `DG1`/`DG2` (`PASS`), (3) chuyển sang `S4` sửa dứt điểm source code + chạy lại test, và (4) quay lại `S5` chạy `git diff` + `DG3` + cập nhật `audit_report_<suffix>.md` cho tới khi sạch bóng bug rồi mới báo OK.

### (G8) Bằng chứng Vật lý trong Workspace (`audit_report_<suffix>.md` & `06_validation_<suffix>.md`)

#### A. Cấu trúc Bắt buộc của `audit_report_<suffix>.md` (Kiểm tra bởi Hook `H7` & `verify_governance.py`)
Ghi đầy đủ vào `agent/memory/workspaces/<workspace_name>/audit_report_<suffix>.md`:
```markdown
# Final Adversarial Audit Report: <Tên Task>

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng code & từng task)
| Task / File:Line | Thay đổi thực tế | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
|------------------|------------------|--------------------------------------------------------|-------------------------------|----------|

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống Mockup, Smoke Test & Hallucination)
- **Kiểm tra Production Mock Pollution:** Xác nhận 0 hàm mock/closure giả (`*Fn`, `sqlmock`, `fakemodel`) trong code production.
- **Bằng chứng Test & Runtime thật:** Lệnh đã chạy, PID/port thật, request/payload thật, log/trace runtime thật.
- **Điểm cần phối hợp User trace runtime value (nếu có):** Nêu rõ biến/luồng cần phối hợp với User để trace trên môi trường thật (tuyệt đối không tự suy diễn).

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Timeout & Liên tục dữ liệu)
- **Concurrency / Multi-worker / Batching:** Kết quả kiểm tra đồng thời, Race Condition, HoL blocking, Debouncer.
- **Timeout / Heartbeat / AckWait:** Kết quả kiểm tra tác vụ dài hạn vs timeout hủy sớm.
- **Data Continuity / Watermark / Idempotency:** Kết quả kiểm tra tính liên tục dữ liệu và chống trùng lặp.

## 4. Anti-Speculation & Honesty Check (Kiểm tra Suy diễn & Báo cáo láo)
- Rà soát từng khẳng định trong báo cáo: đối chiếu 1-1 với `file:line` và output lệnh thật (không có khẳng định nào dựa trên suy đoán).

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit)
- Đối chiếu với `lessons/INDEX.md` (đặc biệt L001, L044, L056, L061).
- Ghi nhận bài học mới vào `lessons.md` + chạy `build_lessons_index.py` (nếu phát hiện lỗi/anti-pattern mới).

## 6. Decision Gate `/laya_router` DG3 Final Verdict
- Summary file: `06_laya_summary_dg3_<ts>.json` | Gate: `PASS`
- Chi tiết các claim `DG3` (bao gồm `control_false` và `control_true`) và nhận xét phản biện cuối cùng.
```

#### B. Cấu trúc `06_validation_<suffix>.md`
Ghi đầy đủ vào `agent/memory/workspaces/<workspace_name>/06_validation_<suffix>.md`:
```markdown
# 06 — Validation Report: <Tên Task>

## 1. Bảng Truy vết Yêu cầu (G1)
| Req ID | Mô tả | File:Line đáp ứng | Bằng chứng Test/Runtime | Trạng thái |
|--------|-------|-------------------|-------------------------|------------|

## 2. Kết quả Test & Runtime (G2, G3, G4, G5, G6)
- Lệnh đã chạy: `<exact command>`
- Output thực tế: `<trích xuất kết quả PASS / đối soát số lượng>`

## 3. Kết quả Decision Gate DG3, Security & Final Audit (G7)
- Summary file: `06_laya_summary_dg3_<ts>.json` | Gate: `PASS`
- Harness² Surface & Clean-Room Check: `python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py validate-surface` (`VALIDATED`)
- Final Adversarial Audit Report: `audit_report_<suffix>.md`
```

