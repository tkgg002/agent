# 00_context — Normalize Global Lessons

- **Ngày khởi tạo**: 2026-06-03
- **Vai trò thực thi**: Muscle (CC CLI) — Chief Engineer
- **Yêu cầu gốc của User**: "lessons.md — thống kê, tổng hợp, sắp xếp, chuẩn hoá lại theo hướng global".

## Bối cảnh
- File `agent/memory/global/lessons.md` đã phình to: **530 KB / 5.061 dòng / ~194–210 lesson**.
- Format drift nặng qua thời gian: 134 lesson theo chuẩn `## [DATE]`, ~76 lesson theo format cũ (`## Lesson 10:`, `## L-xxx`, `## 2026-04-28 — Lesson:`...).
- Tuân thủ field markers lệch: Trigger 153 · Root Cause 122 · Fix 15 · Lesson-marker 5 · Tags 173.
- Tag sprawl: **750 tag riêng biệt**.

## Ràng buộc Governance (BẮT BUỘC)
- **Rule 11 / Rule 7**: `lessons.md` là Memory File / Immutable Audit Log → **CẤM overwrite**, chỉ APPEND.
- **Rule 13**: Mọi lesson phải abstract thành Global Pattern dùng biến A/B/X/Y, format `Global Pattern [A does B to X] → Result Y. Đúng: [correct flow]`, kiểm tra "áp dụng ≥3 dự án".
- **Rule 9**: Workspace-First (đã khởi tạo workspace này).

## Quyết định của User (qua AskUserQuestion)
1. **Đích xuất**: File mới `lessons_global_normalized.md` + APPEND index ngắn vào cuối `lessons.md`.
2. **Độ sâu**: Chuẩn hoá TOÀN BỘ ~210 lesson (viết lại từng cái sang canonical, KHÔNG merge/dedup).
