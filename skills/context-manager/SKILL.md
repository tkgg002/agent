---
name: context-manager
description: >-
  Knowledge Retention & Lesson Management Standard (Rules #4, #5, #6, #17, #21) —
  manages project memory, enforces Canonical 5-part Lesson Format with A/B/X/Y abstraction,
  anti-overwrite wc -l validation, and synchronizes lessons/INDEX.md + topics/*.md via
  build_lessons_index.py.
---

# Context Manager Skill (Memory & Lesson Governance)

> **Quy tắc cốt lõi:** Rule #4 (Knowledge Retention), Rule #5 (Self-Improvement Loop), Rule #6 (Lesson Management Standard), Rule #17 (Memory File Protection), Rule #21 (Context Hygiene).

## Khi nào sử dụng
- Đầu phiên (`S0 BOOT`): Đọc `lessons/INDEX.md` và tra cứu `lessons/topics/<topic>.md`.
- Giữa phiên (`Mid-Session Fix` — Rule #5): Ngay khi bị User sửa lưng hoặc nhắc nhở.
- Cuối phiên (`S6 CLOSE`): Rà soát bài học, ghi lesson mới (nếu có), đồng bộ `INDEX.md`, cập nhật `governance_metrics.md`.

## 1. Cấu trúc Bộ nhớ Chuẩn (`agent/memory/`)
- `agent/memory/global/lessons.md`: File gốc (Single Source of Truth) chứa toàn bộ bài học — **CẤM đọc nguyên file vào context chính (Rule #21)** và **CẤM Overwrite (Rule #17)**.
- `agent/memory/global/lessons/INDEX.md`: Danh mục rút gọn (1 dòng/bài học, ≤ 20KB) dùng để đọc ở đầu mỗi phiên.
- `agent/memory/global/lessons/topics/*.md`: Các file bài học tách theo từng nhóm chủ đề để tra cứu sâu mà không tốn context.
- `agent/memory/global/project/<project_name>/`: Kiến trúc, tech stack, active plans của từng dự án.
- `agent/memory/global/harness_library/<domain>/`: Thư viện Harness² liên nhiệm vụ (`INDEX.md`, `harnesses/<task_id>/EXPERIENCE.md`, `harnesses/<task_id>/harness.json`) được cập nhật bởi `python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py run-recursion --use-live-model --sync-lessons`.
- `agent/memory/workspaces/<workspace_name>/`: Hồ sơ tài liệu của từng task/feature (`01` → `14`).

## 2. Quy trình Ghi Bài học Mới vào `lessons.md` (Bắt buộc 5 bước)

### Bước 1 — Chụp Snapshot Số Dòng Trước Khi Sửa
```bash
wc -l /Users/trainguyen/Documents/work/agent/memory/global/lessons.md
```

### Bước 2 — Soạn Bài học Theo Đúng Canonical Format 5 Phần (Rule #6)
- **Trừu tượng hóa (Abstraction):** Quy trình phải dùng biến `[A]`, `[B]`, `[X]`, `[Y]` để áp dụng được cho ≥ 3 dự án khác nhau.
- **Format bắt buộc (viết liền khối hoặc theo đúng cấu trúc nhóm trong `lessons.md`):**
  ```markdown
  ### [YYYY-MM-DD] <Tiêu đề tập trung vào hậu quả>
  - **Global Pattern:** [A] làm [B] lên [X] -> [Y]. **Đúng:** <flow chuẩn>
  - **Bối cảnh (Trigger):** <ngữ cảnh phát sinh, đã mask mọi secret/PII>
  - **Root Cause:** <nguyên nhân gốc rễ về tư duy/quy trình/kỹ thuật>
  - **Fix/Correct Flow:** <quy trình/giải pháp chuẩn>
  - **Tags:** #kebab-case-tag1 #kebab-case-tag2
  ```

### Bước 3 — Chèn Cục Bộ Bằng `replace_file_content` (CẤM `write_to_file`)
1. Dùng `view_file` đọc đúng đoạn tiêu đề nhóm Taxonomy cần chèn trong `lessons.md` (Ví dụ: `## NHÓM 1` .. `## NHÓM 5`).
2. Dùng `replace_file_content` giữ nguyên tiêu đề nhóm và chèn bài học mới ngay dưới tiêu đề nhóm đó (bài mới nhất nằm trên đầu nhóm).

### Bước 4 — Kiểm tra Snapshot `wc -l` & Đồng bộ `INDEX.md`
```bash
wc -l /Users/trainguyen/Documents/work/agent/memory/global/lessons.md
python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py
```
- Số dòng sau khi chèn **phải lớn hơn** số dòng trước khi chèn.
- Script `build_lessons_index.py` phải báo cáo `count(INDEX) == count(lessons.md)`.

### Bước 5 — Đo lường KPI Quản trị (Định kỳ)
```bash
bash /Users/trainguyen/Documents/work/agent/tooling/governance_metrics.sh
```
