# Operator Map — Antigravity Core Skills (`agent/skills/`)

Bản đồ điều phối 10 Core Skills chuẩn hóa của hệ thống `/agent` trên **Google Antigravity** (đăng ký qua `.agents/skills.json`). Brain và Muscle sử dụng bảng này theo **Master Workflow `S0 → S6`**.

## 1. Bảng Điều phối 10 Core Skills (`agent/skills/<name>/SKILL.md`)

| Skill | Role | Giai đoạn (`S0–S6`) | Chức năng cốt lõi |
| :--- | :--- | :---: | :--- |
| `brain-delegate` | Brain | `S0–S3` | Phân loại Tier (`T0–T3`), tạo workspace, viết `09_tasks_solution_*.md`, chạy `DG1`/`DG2`, xin `APPROVE` của User và lập lệnh Delegate chuẩn. |
| `muscle-execute` | Muscle | `S4–S5` | Thực thi code theo DNA dự án, chạy `checks` và `runtime_check` từ `.agent-project.json`, vượt qua 8 cổng `DoD G1–G8`. |
| `laya-router` | Brain / Muscle / QA | `S2, S3, S5` | Vận hành `tooling/laya_gate.py` (Mode A: Calibrated Claim Gate & Mode B: Agentic Debate Review) tại `DG1`, `DG2`, `DG3`. |
| `audit-propose-only` | Brain / QA | `S2, S5` | Rà soát kiến trúc & code theo nguyên tắc `Propose-Only` (Rule #1, #13): chỉ đọc và xuất báo cáo kèm `REF: path:line`, cấm sửa code khi chưa có `APPROVE`. |
| `debug-agent` | Debugger | `S2, S4` | Truy tìm Root Cause bằng thực chứng (`Reproduce Red -> Green`), kiểm chứng qua `laya_gate.py`. |
| `qa-agent` | QA | `S5` | Kiểm định đối kháng (Adversarial Review), đối chiếu `DoD G1–G8`, kiểm tra `06_validation_*.md` và `DG3`. |
| `security-agent` | Security | `S3, S5` | Quét lộ lọt Secret/PII, kiểm tra bảo mật API/DB/Auth trước khi báo hoàn thành. |
| `workspace-docs` | Brain / Muscle | `S1–S6` | Chuẩn hóa Mandatory Doc Registry (`00`–`14`) theo Tier `T0–T3`, bảo vệ các file append-only (`05_progress`, `04_decisions`, `lessons.md`). |
| `context-manager` | Context Manager | `S0, S6` | Ghi bài học chuẩn 5 phần vào `memory/global/lessons.md` và tái tạo chỉ mục 2 tầng (`tooling/build_lessons_index.py`). |
| `session-wrapup` | Brain / Muscle | `S6` | Đóng phiên: hoàn tất `11_report_*.md`, chạy `tooling/verify_governance.py --workspace <ws>`, đối soát bài học. |

## 2. Hạ tầng Kiểm soát Tự động (Hooks & Tooling)

| Thành phần | Đường dẫn | Vai trò |
| :--- | :--- | :--- |
| **Lifecycle Hooks** | `hooks/agy/agy_guard.py` (`.agents/hooks.json`) | Thực thi 7 chốt chặn tự động `H1–H7` (`PreInvocation`, `PreToolUse`, `PostInvocation`, `Stop`). |
| **Decision Gate CLI** | `tooling/laya_gate.py` | Chạy kiểm định `/laya_router` có đối chứng (`control_true` / `control_false` / `corrupted-diff`). |
| **Lessons Indexer** | `tooling/build_lessons_index.py` | Sinh `memory/global/lessons/INDEX.md` (`<15KB`) và `topics/*.md` từ `lessons.md` (Read-Only). |
| **Process Linter** | `tooling/verify_governance.py` | Kiểm định cấu trúc tài liệu Workspace theo Tier (`T0–T3`) và định dạng Audit Log `05_progress_*.md`. |
| **Project Manifest** | `<project-root>/.agent-project.json` | Khai báo `checks` và `runtime_check` đặc thù của từng dự án để giữ `/agent` 100% project-agnostic. |

> [!NOTE]
> Toàn bộ 96 file workflow phẳng thế hệ cũ đã được lưu trữ an toàn tại `agent/workflows/_archive/` (xem `agent/workflows/README.md`).
