# Context

- Task: Audit lại 3 repo `cdc-cms-service` (api), `cdc-cms-web` (fe), `centralized-data-service` (cdc-worker) và kiểm tra Flow 1 trên CMS.
- Date: 2026-05-08 ICT.
- Scope:
  - Thu thập chứng cứ build/test/log/git-state của 3 repo trong `cdc-system/`.
  - Đối chiếu Flow 1 với trạng thái runtime, API contract, và artifacts đã có trong workspace cũ `feature-cdc-system-refactor`.
- Constraints:
  - Read-first, minimal-impact, không sửa source code nếu chưa cần.
  - Kết luận phải dựa trên verification thực tế; không chỉ đọc code.
  - Memory files append-only sau khi khởi tạo.
