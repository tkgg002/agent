# 08 Tasks: Kiểm tra & Phân loại `git diff` 3 Repo CDC

- **TASK-01**: Thu thập `git status`, `git diff`, và untracked files trên cả 3 repo (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`).
  - **Context**: `/Users/trainguyen/Documents/work/data-hub/{centralized-data-service,cdc-cms-service,cdc-cms-web}`
  - **Logs/Test**: `git status -s`, `git diff --stat`, `git diff`
  - **DoD**: Xác định chính xác 100% các file modified (`M`) và untracked (`??`) kèm số dòng thêm/xóa và vị trí `file:line`.
- **TASK-02**: Phân tích và gom nhóm các thay đổi theo từng Feature / Nhóm chức năng xuyên suốt 3 repo.
  - **Context**: Đối chiếu nội dung hunk `git diff` trên 3 repo với lịch sử các workspace gần nhất (`feat-master-mongo-destination-2026-10-20`, `nats-monitor-missing-core-stream-2026-10-23`, `audit-recon-heal-delete-id-impact-2026-10-23`).
  - **Logs/Test**: Lưu báo cáo phân tích vào `13_analysis_cdc_3repos_diff.md` và chạy `verify_governance.py`.
  - **DoD**: Báo cáo rõ ràng từng Feature đang nằm trong working tree diff của 3 repo kèm bảng ánh xạ `Repo -> File:Line -> Nội dung thay đổi`.
