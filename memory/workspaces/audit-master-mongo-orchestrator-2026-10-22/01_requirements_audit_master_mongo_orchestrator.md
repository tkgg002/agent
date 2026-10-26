---
tier: T2
project: ai-orchestrator
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: audit_master_mongo_orchestrator
created_at: 2026-10-22T10:28:35+07:00
---

# 01 Requirements: Audit Task "Tích hợp Master MongoDB Destination" thực thi qua `ai-orchestrator`

## 1. Yêu cầu của User
- User yêu cầu: `"audit lại cái task 'tích hợp Master MongoDB Destination' mà tao đang làm bằng ai-orchstrator"`.
- Mục tiêu:
  1. Kiểm tra toàn diện trạng thái thực tế của task **"Tích hợp Master MongoDB Destination"** đang được quản lý và thực thi bởi `ai-orchestrator` (`.orchestrator/tasks/`, SQLite DB `state.db`, worktrees `.orchestrator/worktrees/`, logs/artifacts, co-architect chat, sub-tasks `TASK-M01..M10`).
  2. Kiểm tra đối chiếu giữa những gì `ai-orchestrator` đã chạy/báo cáo với mã nguồn thực tế trên các repository đích của `data-hub` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) và hồ sơ gốc tại `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/`.
  3. Phản biện (Adversarial Audit) từng khâu:
     - `ai-orchestrator` đang chạy task này ở bước nào (`S0 -> S6`, `TASK-1`, `TASK-8`, `TASK-11`, `TASK-12`, `TASK-13`, `t-ch-h-p-Master-MongoDB-Destination`)?
     - Các Sub-Tasks (`TASK-M01 .. TASK-M10`) và các bước thực thi của `ai-orchestrator` có thực sự sửa code thật trong worktree/repo đích hay chỉ sinh báo cáo/template giả tạo (hollow execution / smoke-test)?
     - Nội dung `09_tasks_solution_master_mongo.md` và code đã thực thi trong `data-hub` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) có lỗ hổng kiến trúc, bug logic, thiếu sót hay sai lệch nào không?
  4. Thiết kế phương án khắc phục tối ưu nhất (kèm Code Demo) trong `09_tasks_solution_audit_master_mongo_orchestrator.md`, kiểm định qua `/laya_router` (`DG1/DG2` và `DG3`), xuất `audit_report_audit_master_mongo_orchestrator.md`, và tuân thủ tuyệt đối kỷ luật **Propose-Only** (chờ anh `APPROVE` trước khi sửa bất kỳ dòng source code nào).
  5. **Bổ sung Yêu cầu từ Phản hồi Thực tế của User (`AF-ORCH-06` & Ranh Giới Bất Khả Xâm Phạm Repo `data-hub`):**
     - User phản ánh: *"khi tao cho chạy 1 task 11, thì đã có làm nhưng khi tao dùng 1 model ai khác audit lại kết quả thì ... => tao quăng lên ai-orchestrator thì ai-orchestrator ko làm bất cứ gì"* và chấn chỉnh nghiêm khắc: *"sao mày đi edit code repo của tao"*.
     - **Ranh giới Bất khả xâm phạm (`Strict Repo Boundary`):** Agent **TUYỆT ĐỐI CẤM** tự ý chạy `git stash apply/pop` hoặc sửa trực tiếp (`replace_file_content` / `write_to_file`) lên working tree chính của các repo `/Users/trainguyen/Documents/work/data-hub/{cdc-cms-service,cdc-cms-web,centralized-data-service}` của anh! Working tree chính của `data-hub` phải giữ sạch 100%.
     - Mọi thay đổi code của Agent chỉ được phép thực hiện trên **`/Users/trainguyen/Documents/work/ai-orchestrator/`** để nâng cấp bộ máy `ai-orchestrator` (`AF-ORCH-01..06`: tạo Git Worktree cô lập thật tại `.orchestrator/worktrees/TASK-11/<repo>`, tự động nhận diện và bóc tách Báo cáo Audit khi anh dán vào `TASK-11`, và để chính `ai-orchestrator` thực thi bên trong `.orchestrator/worktrees/TASK-11/<repo>`).


