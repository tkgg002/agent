---
name: muscle-execute
description: >-
  Muscle (Chief Engineer) workflow for executing an APPROVED technical solution (S4) —
  logs START in 05_progress, creates local git restore-points, implements code matching
  project DNA, runs checks/tests/runtime_check from .agent-project.json, and triggers
  QA/Security gates. Never runs git push.
---

# Muscle Execute Skill (S4 Full-Loop Execution)

> **Quy tắc cốt lõi:** Rule #1B (Muscle Role), Rule #3 (Agnostic Core & Project Manifest), Rule #4 (Immutable Progress Log), Rule #9 (S4 Execute), Rule #12 (Simplicity First, No Workarounds, No Cheat DB), Rule #15 (NO `git push`), Rule #19 (VCS Restore-point).

## Khi nào sử dụng
- Chỉ sử dụng ở chặng **S4 EXECUTE** sau khi giải pháp trong `09_tasks_solution_*.md` đã qua cổng `DG1/DG2` và **đã được User `APPROVE`**.

## Quy trình Thực thi Chuẩn

### Bước 1 — Kiểm tra Điều kiện Tiền thực thi (Pre-flight Gate)
1. Đọc `09_tasks_solution_<suffix>.md` và `08_tasks_<suffix>.md` trong workspace hiện tại.
2. Xác nhận đã có lệnh `APPROVE` từ User và kết quả `DG1` đạt `PASS`. Nếu thiếu → **DỪNG NGAY**, không sửa code.
3. Tìm và đọc file `<project-root>/.agent-project.json` của dự án đang sửa để biết danh sách `components`, `checks`, `test`, và `runtime_check`.

### Bước 2 — Ghi Log `START` & Tạo Restore-Point (Trước khi chạm vào code)
1. `view_file` phần cuối `05_progress_<suffix>.md` và dùng `replace_file_content` để append dòng `START`:
   ```markdown
   - [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] START thực thi <tên task> theo 09_tasks_solution
   ```
2. Xác định đúng cấp git repo của component đang sửa (`Rule #19`):
   ```bash
   git -C <component_dir> rev-parse --show-toplevel
   ```
3. Tạo local restore-point trước khi sửa (hoặc copy `.bak` nếu thư mục không có git):
   ```bash
   git -C <repo_root> status -s
   # Nếu có thay đổi quan trọng cần chốt mốc trước khi sửa:
   git -C <repo_root> commit -am "chore(restore-point): pre-<task_name>"
   ```
   > **CẢNH BÁO TUYỆT ĐỐI (Rule #15):** CẤM chạy `git push` dưới mọi hình thức!

### Bước 3 — Kiểm tra Phụ thuộc & Tác động Chéo (Dependency Impact Check)
- Tìm toàn bộ nơi gọi/sử dụng hàm, struct, interface, bảng DB hoặc event sắp sửa.
- Đảm bảo thay đổi giữ tương thích ngược hoặc cập nhật đồng bộ tất cả caller trong phạm vi đã duyệt.

### Bước 4 — Triển khai Code (Style Adaptation, Minimal Impact & Anti-Mock-Pollution)
- Mô phỏng 100% phong cách code, cấu trúc thư mục, naming convention và cách xử lý lỗi của dự án hiện tại.
- Giải quyết tận gốc rễ vấn đề theo đúng thiết kế trong `09_tasks_solution_*.md`.
- **Cấm tuyệt đối:**
  - Không cheat DB (không chạy `UPDATE/DELETE/INSERT` trực tiếp để làm đẹp kết quả).
  - **Không làm bẩn code production bằng hàm mock/closure giả** (`*Fn func(`, `lookupMasterTargetsFn`, `runEventsFn`, `sqlmock`, `fakemodel`) để lách tích hợp thật (L001, L056 — Hook `H4` chặn cứng).
  - **Cách ly Phòng Sạch (`mode: clean-room`):** Nếu `01_requirements_<suffix>.md` khai báo `clean_room_forbidden_sources: <csv_paths>` và phiên hiện tại đã đọc bất kỳ file nào trong danh sách cấm, Hook `H4b` sẽ chặn sửa code trực tiếp — bắt buộc uỷ quyền qua `invoke_subagent` (Clean-Room Builder) chỉ nhận `## Clean-Room Approved Spec` mà không mở code gốc cũ.
  - Không nuốt lỗi (silent failure), không sửa vượt quá phạm vi đã được User `APPROVE`.

### Bước 5 — Kiểm thử Thực tế, 3-Mutant Gate, Chống Suy Diễn Smoke Test & Xác minh Runtime (Gate G3, G5, G6)
1. **Fast Checks, Unit/Integration Tests & 3-Mutant Verification Gate:**
   - Chạy toàn bộ lệnh `checks` và `test` được khai báo cho component trong `<project-root>/.agent-project.json`.
   - Viết/cập nhật test cho tính năng hoặc bug vừa sửa (đảm bảo test đỏ trước khi sửa đã chuyển sang xanh — Gate G2).
   - Khi task có `mode: clean-room` (hoặc cần kiểm chứng độ mạnh của bộ test), tạo file `<mutations.json>` chứa $\ge 3$ biến thể cố tình làm hỏng logic (`M1`, `M2`, `M3`) và chạy:
     ```bash
     python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py \
       --workspace <ws> --suffix <suffix> --mutations <path_to_mutations.json>
     ```
     Đảm bảo `gate == "PASS"` (Baseline `PASS` + cả $\ge 3$ bản hỏng đều bị test đánh trượt `KILLED` + `SHA-256` khôi phục 100%).
2. **Xác minh Runtime thật & Chống Suy Diễn Smoke Test (`runtime_check` — L001, L044, L056, L061):**
   - "Build OK / Compile OK / `/health` = 200" hoặc chạy smoke script rời bên ngoài **KHÔNG PHẢI** là Done.
   - Không suy diễn từ test 1 message happy-path (`< 30s`, `1 worker`) rằng luồng thật sẽ đúng: bắt buộc kiểm tra **Concurrency/Multi-worker** (Race condition, HoL blocking, Debouncer), **Timeout/Heartbeat** (`AckWait` vs `InProgress`), và **Data Continuity** (`Watermark`/`Cursor` qua khoảng đứt gãy).
   - Nếu component có tiến trình chạy nền/server (khai báo `runtime_check` trong `.agent-project.json`), phải khởi động lại tiến trình mới, kiểm tra PID/port đang listen, bắn request/luồng thật và kiểm tra log khởi động sạch lỗi.
   - Nếu cần trace runtime value trên môi trường thật mà Agent không tự truy cập được, **phải nói rõ và phối hợp với User để trace runtime value thật**, tuyệt đối không dùng mockup hay hallucinate kết quả.

### Bước 6 — Chuyển giao Kiểm định S5 (`/security-agent` → `/qa-agent` → `DG3` → `audit_report_<suffix>.md`)
1. Chạy `/security-agent` để rà soát lỗ hổng bảo mật và đảm bảo không lộ secret/PII.
2. Chạy `/qa-agent` và Decision Gate `DG3` (`laya_gate.py run --stage dg3` và/hoặc `laya_gate.py review`) với đầy đủ context.
   - Lưu ý ranh giới **S4 / S5 (`Autonomous Self-Healing Loop S5 → S3 → S4 → S5`)**: Ngay khi đã bước sang S5 (`audit_report_<suffix>.md`), nếu phát hiện bug/lỗ hổng mới, Hook `H4b` chặn sửa mò trực tiếp khi chưa cập nhật Plan — bắt buộc quay về S3 cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md`, chạy lại `DG1/DG2` (`PASS`), rồi **tự sửa dứt điểm source code (`S4`) + chạy lại test + Audit lại (`S5` + `DG3`) ngay trong cùng lượt mà KHÔNG dừng lại bắt User `APPROVE` lần 2**.
3. Tạo local commit restore-point sau khi hoàn tất kiểm định (`KHÔNG git push`).
4. Lập báo cáo **`audit_report_<suffix>.md`** (đủ 6 phần phản biện từng dòng diff, chống suy diễn smoke test, concurrency/timeout/continuity, chống báo cáo láo, Self-Improvement Loop, và `/laya_router` `DG3`), ghi kết quả vào `06_validation_<suffix>.md`, `11_report_<suffix>.md` và append `DONE` vào `05_progress_<suffix>.md`.
