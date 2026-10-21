---
name: workspace-docs
description: >-
  Mandatory Workspace Document Registry (00-14), Task Sizing Tiers (T0-T3), Immutable
  Log Format (05_progress), and Project Manifest (.agent-project.json) specification
  (Rule #3, Rule #4, ADR-03, ADR-07).
---

# Workspace Docs & Project Manifest Skill

> **Quy tắc cốt lõi:** Rule #3 (Agnostic Core & `.agent-project.json`), Rule #4 (Mandatory Doc Registry, Task Sizing Tiers `T0–T3`, Immutable Progress Log), Rule #17 (Memory File Protection).

## 1. Phân tầng Quy mô Task (Task Sizing Tiers — ADR-03)

Mỗi file `01_requirements_<suffix>.md` **BẮT BUỘC** khai báo dòng `tier: T1` (hoặc `T2`, `T3`) ở phần đầu file để linter `verify_governance.py` và hook `H7` kiểm tra đúng tập tài liệu:

| Tier | Khi nào áp dụng | Các file bắt buộc trong `agent/memory/workspaces/<ws>/` | Linter (`verify_governance.py`) |
|------|-----------------|--------------------------------------------------------|---------------------------------|
| **T0** | Hỏi đáp, giải thích code, tư vấn nhanh — **không sửa file dự án** | Không tạo workspace | Không chạy |
| **T1** | Hotfix / Micro-task nhỏ | `01_requirements_*.md`, `05_progress_*.md`, `08_tasks_*.md` (và `audit_report_*.md` nếu có sửa code) | Kiểm tra 3 file + format `05` |
| **T2** | Feature / Bugfix chuẩn | Bộ `T1` + `09_tasks_solution_*.md`, `06_validation_*.md`, `11_report_*.md`, `audit_report_*.md` | Kiểm tra 7 file + 6 phần `audit_report` + format `05` |
| **T3** | Epic / Phase lớn / Kiến trúc | Full Doc Set (`00` → `14` + `audit_report_*.md` với suffix riêng cho từng phase) | Kiểm tra Full Set + 6 phần `audit_report` + format `05` |

## 2. Hệ thống Đánh số Tài liệu Chuẩn (Mandatory Doc Registry)

- `00_context_<suffix>.md`: Phạm vi & Thành phần (Scope & Context)
- `01_requirements_<suffix>.md`: Yêu cầu chi tiết + dòng khai báo `tier: T1|T2|T3` (kèm tuỳ chọn `mode: clean-room` và `clean_room_forbidden_sources: <csv_paths>` khi rebuild/refactor phòng sạch)
- `02_plan_<suffix>.md`: Roadmap cao tầng + danh sách 3–5 skills sử dụng
- `03_implementation_<suffix>.md`: Thiết kế kỹ thuật chi tiết / hồ sơ debug fix
- `04_decisions_<suffix>.md`: Nhật ký quyết định kiến trúc (ADRs & giải trình `laya_gate.py resolve` — Append ONLY)
- `05_progress_<suffix>.md`: Nhật ký tiến độ (Audit Log — Append ONLY)
- `06_validation_<suffix>.md` (hoặc `06_test_cases_<suffix>.md`): Kế hoạch & Bằng chứng kiểm thử/runtime thật + raw/summary Decision Gate (`06_laya_summary_*.json`) & Mutation Gate (`06_mutation_summary_<suffix>_<ts>.json`)
- `07_status_report_<suffix>.md`: Báo cáo hiện trạng
- `08_tasks_<suffix>.md`: Danh sách Task & Checklist thực thi chi tiết
- `09_tasks_solution_<suffix>.md`: Hồ sơ giải pháp kỹ thuật + Code Demo + Bảng Decision Gate `DG1/DG2` (bắt buộc có mục `## Clean-Room Approved Spec` khi `mode: clean-room`)
- `10_gap_analysis_<suffix>.md`: Phân tích lỗ hổng kiến trúc / code smell
- `11_report_<suffix>.md`: Báo cáo tổng kết thay đổi (danh sách file thay đổi + số dòng code + mô tả thay đổi)
- `12_implementation_plan_<suffix>.md`: Kế hoạch triển khai chi tiết của AI
- `13_analysis_<suffix>.md`: Báo cáo điều tra / phân tích chuyên sâu của AI
- `14_walkthrough_<suffix>.md`: Hướng dẫn nghiệm thu / walkthrough (nếu có)
- `audit_report_<suffix>.md`: Báo cáo Audit phản biện cuối phiên (6 phần bắt buộc: Phản biện từng dòng diff & từng task, Chống suy diễn Smoke Test & Trace Runtime thật, Kiểm tra Concurrency/Timeout/Data Continuity, Kiểm tra Suy diễn/Báo cáo láo, Self-Improvement Loop, và kết quả `/laya_router` `DG3`).

## 3. Định dạng Duy nhất cho `05_progress_*.md` (Immutable Log)

- **Append-ONLY:** Chỉ nối thêm vào cuối file bằng `replace_file_content` (sau khi đã `view_file` dòng cuối).
- **Format bắt buộc (khớp 100% với `verify_governance.py`):**
  ```markdown
  - [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] <Mô tả hành động>
  ```
  *(Tuyệt đối không dùng bảng Markdown hay đảo `[Agent]` lên trước `[Timestamp]`).*

## 4. Chuẩn Project Manifest (`<project-root>/.agent-project.json` — ADR-07)

Core `agent/` hoàn toàn độc lập với dự án. Mỗi dự án khai báo file `.agent-project.json` tại thư mục gốc của dự án để hook `H6` (fast-feedback), `H7` (done-gate) và `pentagi_bridge.py` (S5 DAST Gate) tự động thực thi:

```json
{
  "name": "<project-name>",
  "allowed_pentest_targets": ["http://localhost:8080", "http://127.0.0.1:8080"],
  "components": [
    {
      "name": "<component-name>",
      "path": "<relative-path-from-project-root>",
      "checks": ["<fast check command returning non-zero on failure>"],
      "test": "<full test suite command>",
      "runtime_check": "<command verifying live process/port after restart>",
      "pentest_check": "<optional command triggering pentagi_bridge.py for local/staging DAST>"
    }
  ]
}
```
- `allowed_pentest_targets` *(tuỳ chọn, mặc định chỉ cho phép `localhost` / `127.0.0.1` / `[::1]`)*: Danh sách prefix URL được phép quét DAST qua `tooling/pentagi_bridge.py trigger`. Mọi domain nằm ngoài danh sách này bị chặn cứng để bảo vệ môi trường production.
- `pentest_check` *(tuỳ chọn)*: Lệnh kiểm thử xâm nhập tự động có phạm vi (scoped DAST) chạy ở chặng `S5` đối với các service có HTTP/gRPC/GraphQL API.
