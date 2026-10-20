# 11 — Change Report: Tối ưu hoá toàn diện hệ thống `/agent`

## 1. Tổng quan Thay đổi (Architectural Overview)
1. **Rút gọn Hiến pháp Lõi (`GEMINI.md`):** Giảm từ `30,949 bytes` xuống `11,928 bytes` (`-61.5%`, dưới ngưỡng cắt `24,000 bytes` của Antigravity và đạt mục tiêu `<= 12,000 bytes`), giữ nguyên 100% số hiệu `Rule #0` đến `Rule #22`, bổ sung **Master Workflow `S0–S6`**, **Decision Gate `/laya_router` (`DG1–DG3`)**, **Task Sizing Tiers `T0–T3`**, và **Project Manifest `.agent-project.json`**.
2. **Chuẩn hóa 10 Core Skills (`agent/skills/<name>/SKILL.md`):** Thay thế 96 file workflow phẳng cũ (đã lưu trữ vào `workflows/_archive/`) bằng 10 skill chuẩn Antigravity, đăng ký qua `.agents/skills.json`.
3. **Tích hợp Cổng Quyết định `/laya_router` (`tooling/laya_gate.py`):** Hỗ trợ Mode A (Calibrated Claim Gate với `>=2 control_false`, `>=1 control_true`, kiểm tra trích dẫn `REF: path:start-end` trên đĩa, tự động retry khi HTTP 503) và Mode B (Agentic Debate Recursive Fencing với control `corrupted-diff` chống rubber-stamping).
4. **Cơ chế Cưỡng chế Tự động bằng Hooks (`hooks/agy/agy_guard.py`):** Triển khai 7 chốt chặn `H1–H7` đăng ký qua `.agents/hooks.json`, đọc cấu hình đặc thù dự án từ `.agent-project.json` để giữ `/agent` 100% project-agnostic.
5. **Chỉ mục Bài học 2 Tầng (`tooling/build_lessons_index.py`):** Tách `memory/global/lessons.md` (`487 KB`, `241` bài học — giữ nguyên Read-Only) thành `memory/global/lessons/INDEX.md` (`14,576 bytes < 15 KB`) và 5 file `topics/01..05_*.md`.
6. **Nâng cấp Process Linter (`tooling/verify_governance.py` & `tooling/governance_metrics.sh`):** Bỏ đường dẫn tĩnh, bắt buộc truyền `--workspace <name>`, kiểm định tài liệu theo Tier `T0–T3`.

## 2. Danh sách File Thay đổi & Số dòng Code (LOC)

| Đường dẫn File | Thao tác | Số dòng (LOC) | Mô tả thay đổi |
| :--- | :---: | :---: | :--- |
| `GEMINI.md` | Modified | 216 | Rút gọn còn `11,928B`, tích hợp `S0–S6`, `DG1–DG3`, `T0–T3`, `.agent-project.json`. |
| `rules/GEMINI_FULL_ARCHIVE_2026-10.md` | Created | 476 | Bản lưu trữ nguyên vẹn 100% `GEMINI.md` trước khi rút gọn (`30,949B`). |
| `OPERATOR_MAP.md` | Modified | 35 | Bản đồ điều phối 10 Core Skills và bộ công cụ Antigravity. |
| `.agent-project.json` | Created | 15 | Manifest khai báo `checks` và `runtime_check` của chính repo `/agent`. |
| `.agents/hooks.json` | Created | 45 | Cấu hình 5 lifecycle hook events cho Antigravity. |
| `.agents/skills.json` | Created | 6 | Đăng ký thư mục `agent/skills` và `~/.agent/skills`. |
| `hooks/agy/agy_guard.py` | Created | 417 | Dispatcher thực thi 7 chốt chặn `H1–H7`. |
| `hooks/agy/tests/test_agy_guard.py` | Created | 396 | Bộ 16 fixture tests kiểm thử toàn diện `H1–H7`. |
| `hooks/README.md` | Created | 49 | Tài liệu kiến trúc và cách vận hành `agy_guard.py`. |
| `hooks/_archive_claude/*` | Archived | 7 files | Lưu trữ 6 hook `.sh` cũ của Claude Code + `organize_skills.sh` (`chmod -x`). |
| `tooling/laya_gate.py` | Created | 681 | CLI cổng quyết định `/laya_router` (Mode A + Mode B + Backtest). |
| `tooling/tests/test_laya_gate.py` | Created | 360 | Bộ unit test + backtest cho `laya_gate.py`. |
| `tooling/build_lessons_index.py` | Created | 272 | Script sinh chỉ mục 2 tầng `INDEX.md` + `topics/*.md` từ `lessons.md`. |
| `tooling/tests/test_build_lessons_index.py` | Created | 57 | Unit test kiểm tra bảo toàn số lượng entry và giới hạn `< 15 KB`. |
| `tooling/verify_governance.py` | Modified | 183 | Process Linter theo Tier `T0–T3`, bắt buộc `--workspace`. |
| `tooling/tests/test_verify_governance.py` | Created | 85 | Unit test cho `verify_governance.py` (`T0`, `T1`, `T2`, lỗi thiếu flag/log). |
| `tooling/governance_metrics.sh` | Modified | 95 | Đo KPI hệ thống theo kiến trúc Antigravity mới. |
| `skills/*/SKILL.md` | Created | 10 files (875 dòng) | 10 Core Skills (`brain-delegate`, `muscle-execute`, `laya-router`, `audit-propose-only`, `debug-agent`, `qa-agent`, `security-agent`, `workspace-docs`, `context-manager`, `session-wrapup`). |
| `workflows/README.md` & `workflows/_archive/*` | Archived | 96 files | Lưu trữ gọn gàng 96 file workflow phẳng cũ vào `_archive/`. |
| `memory/global/lessons/INDEX.md` + `topics/*.md` | Created | 6 files | Chỉ mục `INDEX.md` (`14,731B`) và 5 file chủ đề (`244` bài học). |

## 3. Cập nhật Phase 6 — Khắc phục Triệt để 5 Lỗ hổng Bỏ qua Workflow & Hook (`2026-10-16`)
- **`~/.gemini/config/skills/` & `data-hub/.agents/{skills,hooks}.json`:** Lưu trữ `438` skill ngoại lai sang `~/.gemini/config/skills_archive/`, symlink 10 Core Skills trực tiếp vào `~/.gemini/config/skills/` (Priority 3) và đồng bộ `.agents/{skills,hooks}.json` tại `data-hub` (Priority 2) để không bao giờ bị cắt khỏi System Prompt do tràn Context Budget.
- **`GEMINI.md` (Rule #9):** Định nghĩa rõ ranh giới `T0` vs `T1–T3/Audit` và inline checklist bắt buộc của từng bước `S0 → S6` kèm đường dẫn `agent/skills/<skill>/SKILL.md`.
- **`hooks/agy/agy_guard.py` (`545` dòng) & `hooks/agy/tests/test_agy_guard.py` (`20/20 PASS`):**
  - Thêm `_unquote()` bóc dấu `"` bao ngoài các trường `args` trong `transcript.jsonl` thật của Antigravity.
  - Nâng cấp `H1` chạy theo từng lượt `USER_INPUT` mới (`last_user_idx`).
  - Nâng cấp `H4` bổ sung cổng **S1 Intake Gate trước S3 Solution** (`deny` nếu tạo `09_tasks_solution_<suffix>.md` mà thiếu `01`, `08`, hoặc chưa append `05_progress`).
  - Nâng cấp `H7` reset bộ đếm `.stop` theo từng lượt `USER_INPUT` và chặn **Shadow Analysis** (đọc `>= 4` file dự án mà không ghi hồ sơ workspace).
- **`tooling/verify_governance.py` (`256` dòng) & `tooling/tests/test_verify_governance.py` (`7/7 PASS`):** Hỗ trợ `--suffix <suffix>` và `--stage S3|S6`, chặn việc PASS "ăn theo" các file cũ khi tái sử dụng workspace.
