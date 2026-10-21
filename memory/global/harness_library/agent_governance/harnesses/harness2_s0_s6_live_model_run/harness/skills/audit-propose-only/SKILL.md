---
name: audit-propose-only
description: >-
  Read-Only Architectural & Code Audit workflow — investigates codebase/runtime with
  file:line evidence (S2), validates findings via Decision Gate DG3 (laya_gate.py),
  and produces gap analysis or audit reports while strictly prohibiting any code edits
  until explicit User APPROVE.
---

# Audit Propose-Only Skill (Read-Only Investigation & DG3 Audit)

> **Quy tắc cốt lõi:** Rule #1A (Propose-Only & Architectural Refinement), Rule #9 (Audit Flow & S5 Boundary), Rule #10 (Decision Gate `DG3`), Rule #13 (Brain Code Prohibition).
> **Bài học gốc:** `#audit-propose-only` & `#no-unauthorized-fix` (2026-10-14) — Khi User yêu cầu kiểm tra, rà soát, audit hoặc đánh giá: **CHỈ ĐỌC VÀ BÁO CÁO**, tuyệt đối không tự ý sửa bất kỳ dòng code nào.

## Khi nào sử dụng
- Khi User yêu cầu: *"kiểm tra"*, *"audit"*, *"review"*, *"rà soát"*, *"tối ưu quy trình/kiến trúc cần làm gì"*, *"tìm lỗ hổng"*.
- Ở chặng **S5 SELF-AUDIT** khi phát hiện vấn đề nằm ngoài phạm vi thiết kế đã được `APPROVE` ở S3.

## Quy trình 4 Bước Bắt Buộc

### Bước 1 — Khởi tạo Hồ sơ Audit (Không chạm vào Source Code)
- Chọn hoặc tạo workspace trong `agent/memory/workspaces/<audit_ws>/`.
- Append `START audit (Propose-Only)` vào `05_progress_<suffix>.md`.
- Chỉ sử dụng các công cụ đọc (`view_file`, `grep`, lệnh kiểm tra trạng thái read-only).

### Bước 2 — Thu thập Bằng chứng `file:line` & Phân loại
Mỗi phát hiện (Finding `F1..Fn`) phải có:
- **Bằng chứng cụ thể:** Đường dẫn file, số dòng (`file:line`), đoạn code trích nguyên văn hoặc kết quả đo đạc thực tế.
- **Mức độ nghiêm trọng:** `P0 (Critical)` | `P1 (High)` | `P2 (Medium/Low)`.
- **Phân loại xác thực:** `VERIFIED` (có code/log chứng minh) hoặc `HYPOTHESIS` (cần đo thêm trên runtime).

### Bước 3 — Kiểm định Qua Decision Gate `DG3` (`laya_gate.py`)
Trước khi xuất báo cáo cho User, lập file `claims_dg3.json` (gồm các claim `A-F` cho từng finding + `≥2 control_false` + `≥1 control_true`) và chạy:
```bash
python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run \
  --stage dg3 --workspace <audit_ws> --claims <claims_dg3.json> \
  --objective "Kiểm chứng các phát hiện audit"
```
- Bất kỳ finding nào bị `BLOCK` (sai lệch trích dẫn `REF:` hoặc bị bác bỏ với độ tự tin cao) phải được kiểm tra lại và loại bỏ/sửa trước khi báo cáo.

### Bước 4 — Xuất Báo cáo Vật lý & Chờ Lệnh `APPROVE`
- Ghi báo cáo vào `13_analysis_<suffix>.md`, `10_gap_analysis_<suffix>.md`, hoặc `09_tasks_solution_<suffix>.md` kèm bảng kết quả `DG3`.
- Trình bày **1 phương án khắc phục tối ưu nhất** và kết thúc bằng câu hỏi xin lệnh `APPROVE` từ User.
- **DỪNG LẠI** — không thực hiện bất kỳ chỉnh sửa code nào cho đến khi User xác nhận `APPROVE`.
