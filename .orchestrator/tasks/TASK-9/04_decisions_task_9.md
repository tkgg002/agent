# 04 Architectural Decisions (ADR) — TASK-9: Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5

- **Suffix:** `task_9`
- **Created At:** `2026-10-27T17:10:06+07:00`
- **ADR-01 (Per-Task Isolation & Clean-Room):** Mọi thay đổi thực thi trong Git Worktree cô lập, kiểm định qua Laya Gate DG1/DG2/DG3 và biên lai vật lý `EV-...`.
