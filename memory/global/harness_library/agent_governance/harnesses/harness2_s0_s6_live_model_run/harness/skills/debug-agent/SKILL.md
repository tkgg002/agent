---
name: debug-agent
description: >-
  Debugger specialist workflow (S2 Investigate & Bug Handling SOP) — systematic root-cause
  analysis from logs, stack traces, and runtime state; mandatory Red-to-Green reproduction
  (Gate G2); cross-component pattern search; zero blind-fixing.
---

# Debug Agent Skill (Root Cause Analysis & Bug Handling SOP)

> **Quy tắc cốt lõi:** Rule #0 (Không đoán mò, không suy diễn), Rule #5 (Root Cause & Self-Improvement), Rule #9 (S2 Investigate), Rule #10 (Decision Gate `/laya_router`), Rule #14-G2 (Reproduce Red → Green), Rule #15 (Escalation >3 lần lặp).

## Khi nào sử dụng
- Khi điều tra lỗi (bug, crash, sai lệch dữ liệu, timeout, race condition, test fail).
- Được gọi ở chặng **S2 INVESTIGATE** trước khi viết `09_tasks_solution_*.md`.

## Quy trình 6 Bước Săn Lỗi (Evidence-First)

### Bước 1 — Khởi tạo Log & Thu thập Triệu chứng (Symptom Capture)
1. Đảm bảo workspace đã có `05_progress_<suffix>.md` và append dòng `START điều tra root cause`.
2. Thu thập nguyên văn:
   - Error message / stack trace / HTTP status / response body.
   - Dòng log thực tế trước và sau thời điểm xảy ra lỗi.
   - Trạng thái tiến trình và dữ liệu thực tế (chỉ đọc — `SELECT` / read-only).

### Bước 2 — Tái hiện Lỗi (Reproduce: Red Phase — Gate G2)
- **BẮT BUỘC** tái hiện được lỗi trước khi đề xuất sửa:
  - Viết unit/integration test tái hiện đúng điều kiện gây lỗi (test phải FAIL — Red), **HOẶC**
  - Chạy kịch bản request/input thực tế tái hiện chính xác dòng log lỗi.
- Nếu chưa tái hiện được bằng dữ liệu thực tế, mọi nhận định chỉ được gắn nhãn `HYPOTHESIS`, tuyệt đối không báo là `VERIFIED` và không sửa mò (blind-fixing).

### Bước 3 — Khoanh vùng & Truy vết Luồng Dữ liệu (Isolate & Trace)
1. Truy vết từng bước từ điểm vào (Entrypoint / Handler / Consumer / Job) → UseCase / Service → Repository / Adapter → DB / External API.
2. Ghi nhận chính xác `file:line` tại mỗi mắt xích.
3. Kiểm tra lịch sử thay đổi gần nhất nếu là lỗi hồi quy:
   ```bash
   git -C <repo> log --oneline -n 15 -- <path/to/file>
   ```

### Bước 4 — Xác định Nguyên nhân Gốc rễ & Quét Mẫu hình Ngang (Cross-Pattern Scan)
1. Xác định đúng phân loại lỗi gốc rễ:
   - `Logic Error` | `Data/Schema Mismatch` | `Race Condition / Teardown Order` | `Config / Env Drift` | `Contract / Integration Break` | `Resource / Memory Leak`.
2. **Quét toàn dự án (Cross-Component Search):**
   - Dùng `grep` tìm xem cùng pattern lỗi đó còn xuất hiện ở file/component nào khác không để tránh lỗi "sửa 1 chỗ, sót 3 chỗ".

### Bước 5 — Đóng gói Hồ sơ Giải pháp & Kiểm định Decision Gate (`DG1`)
- Ghi toàn bộ bằng chứng (`Symptom`, `Timeline/Hypotheses`, `Root Cause file:line`, `Suggested Fix`, `Blast Radius`, `Code Demo`) vào `09_tasks_solution_<suffix>.md` (và `03_implementation_<suffix>_fix.md` nếu là T3).
- Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` (và `dg2` cho code demo).
- Trình User duyệt (`APPROVE`) trước khi sửa code.

### Bước 6 — Xác minh Sau Sửa (Green Phase & Lesson Check)
Sau khi được `APPROVE` và sửa ở chặng S4:
1. Test tái hiện lỗi ở Bước 2 phải chuyển từ **FAIL (Red) → PASS (Green)**.
2. Kiểm tra log khởi động và log lúc chạy thực tế sạch lỗi (`error`, `panic`, `fatal`, `sqlstate`).
3. Đối chiếu số liệu `Before` vs `After` và ghi vào `06_validation_<suffix>.md`.
4. Nếu phát hiện mẫu hình lỗi mới có giá trị phòng ngừa cho các dự án sau → gọi `/context-manager` để ghi lesson tổng quát (A/B/X/Y) vào `lessons.md`.

## Escalation Rule (Rule #15)
- Nếu đã thử quá **3 giả thuyết / 3 vòng lặp** mà chưa tái hiện hoặc chưa giải quyết được gốc rễ → **DỪNG NGAY**, báo cáo toàn bộ dữ kiện đã loại trừ cho User để re-plan. Tuyệt đối không đoán mò thử tiếp.
