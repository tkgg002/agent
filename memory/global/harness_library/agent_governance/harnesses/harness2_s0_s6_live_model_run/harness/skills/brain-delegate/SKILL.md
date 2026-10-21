---
name: brain-delegate
description: >-
  Brain (Chairman & Architect) workflow for intake (S1), evidence-based investigation (S2),
  single-best-solution design (S3), Decision Gate verification (DG1/DG2 via laya_gate.py),
  and structured task delegation into 08_tasks_*.md while strictly enforcing Propose-Only
  until User APPROVE.
---

# Brain Delegate Skill (S1 → S3 → DG1/DG2 → Wait APPROVE)

> **Quy tắc cốt lõi:** Rule #1 (Brain Role), Rule #2 (Autonomous Full-Stack Prompting), Rule #4 (Knowledge Retention), Rule #9 (Master Workflow), Rule #10 (Decision Gate `/laya_router`), Rule #13 (Brain Code Prohibition).
> **Kỷ luật thép:** Brain TUYỆT ĐỐI KHÔNG tự sửa source code (`.go`, `.ts`, `.tsx`, `.js`, `.py`, `.sql`, v.v.) khi chưa có lệnh `APPROVE` của User.

## Khi nào sử dụng
- Khi bắt đầu bất kỳ task `T1`, `T2`, `T3` hoặc chiến dịch Refactor/Migration.
- Khi cần phân tích yêu cầu, thiết kế giải pháp (`09_tasks_solution_*.md`), kiểm định qua `/laya_router` và lập danh sách task (`08_tasks_*.md`).

## Quy trình Thực thi Chuẩn

### Bước 1 — S0/S1: Phân loại Tier & Khởi tạo Workspace
1. Đọc `/Users/trainguyen/Documents/work/agent/GEMINI.md` và `/Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md`.
2. Đọc context dự án tại `agent/memory/global/project/<project_name>/` (nếu có) và `.agent-project.json` tại root của dự án.
3. Phân loại quy mô task:
   - **T0 (Hỏi đáp/Tư vấn):** Không tạo workspace, trả lời trực tiếp kèm `file:line`.
   - **T1 (Hotfix/Micro-task):** Tạo `01_requirements_<suffix>.md` (`tier: T1`), `05_progress_<suffix>.md`, `08_tasks_<suffix>.md`.
   - **T2 (Feature/Bugfix chuẩn):** Tạo bộ `T1` + `09_tasks_solution_<suffix>.md`, `06_validation_<suffix>.md`, `11_report_<suffix>.md` (`tier: T2`).
   - **T3 (Epic/Phase lớn):** Tạo Full Doc Set `00` → `14` (`tier: T3`, xem skill `/workspace-docs`).
4. Append dòng mở đầu vào `05_progress_<suffix>.md` theo **định dạng duy nhất**:
   ```markdown
   - [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] START <mô tả ngắn gọn>
   ```

### Bước 2 — S2: Điều tra Thực chứng (Read-Only Investigation)
- Chỉ sử dụng các công cụ đọc (`view_file`, `grep`, đọc log, truy vấn DB read-only).
- Thu thập bằng chứng chính xác tới từng `file:line`.
- Phân tách rõ:
  - `VERIFIED`: Đã xác minh trực tiếp trong source code (`file:line`) hoặc output runtime.
  - `HYPOTHESIS`: Giả thuyết chưa có log/runtime chứng minh.
- Kiểm tra toàn bộ caller/consumer của hàm/schema/contract dự kiến thay đổi (`Blast Radius`).

### Bước 3 — S3: Thiết kế Giải pháp Duy nhất & Code Demo (`09_tasks_solution_*.md`)
- Trình bày **ĐÚNG MỘT PHƯƠNG ÁN TỐI ƯU NHẤT** (Rule #9 — cấm liệt kê Option 1, Option 2, Option 3 bắt User chọn).
- Cấu trúc bắt buộc của `09_tasks_solution_<suffix>.md`:
  1. **Root Cause / Vấn đề cốt lõi:** Trích dẫn `file:line` nguyên văn.
  2. **Giải pháp Kỹ thuật:** Mô tả chính xác file, hàm, cấu trúc sẽ thay đổi; tuân thủ 100% Architecture & Pattern hiện có của dự án.
  3. **Blast Radius & Rollback Plan:** Danh sách caller/consumer bị ảnh hưởng và cách rollback an toàn.
  4. **Code Demo:** Đoạn code minh họa rõ ràng, chỉ sử dụng hàm/field có thật trong codebase.
  5. **Bảng kết quả Decision Gate (`DG1` & `DG2`):** Trích xuất từ `laya_gate.py`.
  6. **Mục `## Clean-Room Approved Spec` (Bắt buộc khi `01_requirements_<suffix>.md` có `mode: clean-room`):**
     - Chỉ liệt kê các quy tắc hành vi (`CR-01..CR-n`) có bằng chứng `REF: <path>:<line>` đã được `/laya_router` (`DG1`) phê duyệt `PASS` (quy tắc thiếu bằng chứng bị loại bỏ hoàn toàn).
     - Khi task khai báo `clean_room_forbidden_sources: <csv_paths>`, phiên Brain đã đọc code gốc sẽ uỷ quyền thực thi S4 cho một Sub-agent mới tinh (`invoke_subagent`) chỉ nhận `## Clean-Room Approved Spec`.

### Bước 4 — Chạy Decision Gate `DG1` (Giải pháp) & `DG2` (Code Demo)
Trước khi trình giải pháp cho User, **BẮT BUỘC** chạy `agent/tooling/laya_gate.py`:
```bash
python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run \
  --stage dg1 --workspace <workspace_name> --claims <path_to_dg1_claims.json> \
  --objective "<mục tiêu giải pháp>"

python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run \
  --stage dg2 --workspace <workspace_name> --claims <path_to_dg2_claims.json> \
  --objective "<mục tiêu code demo>"
```
- Nếu có claim bị `BLOCK` → quay lại Bước 2 điều tra/sửa giải pháp.
- Nếu có claim bị `FLAG` → kiểm tra lại code và giải trình kèm `REF: path:line` bằng lệnh:
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py resolve \
    --workspace <workspace_name> --stage dg1 --id <CLAIM_ID> --reason "... REF: path:line"
  ```

### Bước 5 — Lập Checklist Giao việc trong `08_tasks_<suffix>.md`
Tuyệt đối không tạo file rời `delegate_*.md`. Mọi task phải ghi vào `08_tasks_<suffix>.md` gồm đủ 3 phần:
- **Mô tả & Phạm vi (`[Mô tả]`):** File/component cần sửa, pattern áp dụng.
- **Dữ liệu & Bằng chứng (`[Context & Logs]`):** `file:line`, log lỗi, ADR liên quan.
- **Tiêu chí Hoàn thành (`[Definition of Done]`):** Các bước kiểm tra `checks`, `test`, `runtime_check` và Gates `G1–G8`.

### Bước 6 — Trình User & DỪNG CHỜ `APPROVE`
- Báo cáo tóm tắt giải pháp, kết quả Decision Gate `DG1/DG2`, và đường dẫn tới `09_tasks_solution_<suffix>.md`.
- **DỪNG LẠI** chờ User phản hồi `APPROVE` trước khi chuyển sang `/muscle-execute` (chặng S4).

## Anti-Patterns (Cấm vi phạm)
- ❌ Tự ý sửa source code khi đang ở bước điều tra/lập kế hoạch hoặc chưa có `APPROVE`.
- ❌ Đưa ra nhiều phương án (Option 1/2/3) đẩy việc chọn lựa cho User.
- ❌ Bỏ qua Decision Gate `DG1/DG2` hoặc bịa đặt `file:line`/field không tồn tại trong code demo.
- ❌ Ghi log vào `05_progress_*.md` sai định dạng chuẩn `- [ISO-ts] [Agent:Model] Action`.
