# 08 Tasks — mr_recon_heal_develop

- [x] **TASK-01**: Kiểm tra trạng thái nhánh (`recon-heal`, `develop`, `origin/develop`, `origin/recon-heal`), commit history, merge-base, xung đột tiềm ẩn (`git merge-tree` / divergence) và working tree (`git status`) tại 2 repo `/Users/trainguyen/Documents/work/data-hub/centralized-data-service` và `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`.
  - Context: `centralized-data-service` & `cdc-cms-service` git metadata.
  - Test/Logs: `git log`, `git diff --stat`, `git status --short`.
  - DoD: Nắm rõ 100% danh sách commit, danh sách file thay đổi trong MR (`develop...recon-heal`) và các file đang dirty ngoài working tree.
- [x] **TASK-02**: Rà soát chi tiết từng file thay đổi (`git diff develop...recon-heal` và working tree diff) trên `centralized-data-service` và `cdc-cms-service`, kiểm tra tương thích ngược, hợp đồng liên service (CMS <-> Worker), SQL/DB safety, concurrency, và build/vet/test tĩnh.
  - Context: Toàn bộ các file trong `git diff develop...recon-heal` ở cả 2 repo.
  - Test/Logs: `git diff`, `go build`, `go vet` (`EV-20261109T112016-0802`, `EV-20261109T112017-9c7b`), `go test` (`EV-20261109T112037-78e5`, `EV-20261109T112038-bf44`).
  - DoD: Liệt kê đầy đủ các rủi ro `F1..F5` kèm mức độ (`P1/P2`), phân loại `VERIFIED`, và bằng chứng `file:line`.
- [x] **TASK-03**: Chạy cổng phản biện `DG0`, `DG1` (cho giải pháp/Code Demo đề xuất) và `DG3` (`laya_gate.py`) kiểm chứng toàn bộ phát hiện và xuất `09_tasks_solution_mr_recon_heal_develop.md` + `audit_report_mr_recon_heal_develop.md`.
  - Context: Workspace `audit-mr-recon-heal-develop-2026-11-09`.
  - Test/Logs: `laya_gate.py run --stage dg0/dg1/dg3`, `verify_governance.py`.
  - DoD: Tất cả các cổng Laya đạt `PASS`, `verify_governance.py` đạt `PASSED`.
