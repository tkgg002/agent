# Project Brain Governance Standard (Rule #4)

Tài liệu hướng dẫn thực thi (Operational Guide) cho Quy tắc số 4 (Knowledge Retention), đảm bảo tính nhất quán, trực quan và khả năng truy vết tuyệt đối của hệ thống Agentic Core (`/agent`).

## 1. Nguyên tắc Cốt lõi (Core Principles)

- **Workspace-First (Tier `T1–T3`)**: Mọi task từ Tier `T1` trở lên đều phải có thư mục vật lý trong `agent/memory/workspaces/<workspace_name>/`. Tier `T0` (hỏi đáp/tư vấn, không sửa file) không yêu cầu tạo workspace.
- **Immutable Log (`05_progress_*.md`)**: Nhật ký tiến độ là lịch sử Audit tối cao. Chỉ cho phép **APPEND**. Quy tắc: *"Sai thì ghi nối dòng Log mới là Phản hồi/Revert, tuyệt đối không xóa/sửa dòng cũ"*.
- **Metadata Integrity**: Mọi dòng trạng thái PHẢI tuân thủ định dạng duy nhất: `- [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] <Action>`.
- **Knowledge Loop**: Mọi bài học mới đều phải được chuẩn hóa 5 phần vào `memory/global/lessons.md` và cập nhật chỉ mục 2 tầng qua `python3 tooling/build_lessons_index.py`.

## 2. Hệ thống Định danh Prefix (Mandatory Doc Registry `00–14`)

| Prefix | Loại tài liệu | Tier yêu cầu | Nội dung chi tiết |
| :--- | :--- | :---: | :--- |
| `00` | `00_context_*.md` | `T3` | Phạm vi, kiến trúc tổng quát, các thành phần liên quan. |
| `01` | `01_requirements_*.md` | `T1, T2, T3` | Yêu cầu chi tiết, tiêu chí nghiệm thu, khai báo `tier: T1\|T2\|T3`. |
| `02` | `02_plan_*.md` | `T3` | Roadmap cao tầng & khai báo 3–5 Skills sử dụng (Rule #7). |
| `03` | `03_implementation_*.md` | Tuỳ chọn | Thiết kế kỹ thuật chi tiết. |
| `04` | `04_decisions_*.md` | `T3` / Khi có ADR | Nhật ký quyết định kiến trúc (ADRs & Laya FLAG resolutions — **Append ONLY**). |
| `05` | `05_progress_*.md` | `T1, T2, T3` | **Audit Log tối cao (Append ONLY)**. |
| `06` | `06_validation_*.md` | `T2, T3` | Bằng chứng kiểm thử (`DoD G1–G8`, test output, runtime check, `06_laya_summary_*.json`). |
| `07` | `07_status_report_*.md` | Tuỳ chọn | Báo cáo hiện trạng phiên/phase. |
| `08` | `08_tasks_*.md` | `T1, T2, T3` | Checklist Task chi tiết. |
| `09` | `09_tasks_solution_*.md` | `T2, T3` | Hồ sơ giải pháp kỹ thuật (1 phương án tối ưu duy nhất + code demo, chờ User `APPROVE`). |
| `10` | `10_gap_analysis_*.md` | Khi Audit | Phân tích lỗ hổng kiến trúc (`Propose-Only`). |
| `11` | `11_report_*.md` | `T2, T3` | Báo cáo bàn giao: danh sách file thay đổi + số dòng code + tổng quan. |
| `12` | `12_implementation_plan_*.md` | `T3` | Kế hoạch triển khai chi tiết đã thực thi. |
| `13` | `13_analysis_*.md` | `T3` / Audit | Báo cáo phân tích sâu. |
| `14` | `14_walkthrough_*.md` | Tuỳ chọn | Hướng dẫn kiểm tra/nghiệm thu sau triển khai. |

## 3. Định dạng Chuẩn cho `05_progress_*.md` (Hợp lệ với `verify_governance.py` & Hook `H2`)

```markdown
# 05 — Progress (Append ONLY)

- [2026-10-16T10:00:00+07:00] [Agent:Claude-Opus-5.5] START khởi tạo workspace và phân tích yêu cầu.
- [2026-10-16T10:30:00+07:00] [Agent:Claude-Opus-5.5] Hoàn tất 09_tasks_solution và chạy DG1/DG2 PASS. Chờ User APPROVE.
- [2026-10-16T11:00:00+07:00] [Agent:Claude-Opus-5.5] Thực thi S4, pass DoD G1-G8 và verify_governance.py. DONE.
```

> [!IMPORTANT]
> **No Overwrite Policy (Rule #17 — Hook H2)**: TUYỆT ĐỐI CẤM dùng `Overwrite: true` trên bất kỳ Memory file nào đã tồn tại, và phải giữ nguyên `TargetContent` khi nối thêm vào các file append-only (`05_progress`, `04_decisions`, `lessons.md`).
