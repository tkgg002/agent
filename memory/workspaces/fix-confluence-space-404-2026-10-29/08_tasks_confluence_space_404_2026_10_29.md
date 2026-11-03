# 08 Tasks: Audit & Xử lý Lỗi Confluence 404 (`confluence_space_404_2026_10_29`)

- **Workspace**: `fix-confluence-space-404-2026-10-29`
- **Suffix**: `confluence_space_404_2026_10_29`
- **Tier**: `T2`

---

- [x] **TASK-01 — Điều tra nguyên nhân gốc rễ lỗi `404 No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c` (`S2 Investigate`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go:210-264`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go:629-705`
    - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx:1212-1224, 1302-1311, 1374-1386, 1743-1748`
  - **Logs/Test**: Kiểm tra thực tế Atlassian Confluence Cloud REST API (`/wiki/rest/api/content/500432919?expand=space`, `/wiki/rest/api/space/~712020a3dcebcbe93646fea464f119dd38b08c?expand=permissions`, `/wiki/rest/api/user/memberof?accountId=712020:fd809a67-b541-4346-9aa7-7a482cf14324`).
  - **Definition of Done (DoD)**: Xác định 100% `VERIFIED` vì sao `sre@goopay.vn` xác thực thành công (`authorized: true`) nhưng nhận `404 No space with key : ~712020a3dcebcbe93646fea464f119dd38b08c` (do thiếu group `confluence-users-goopay` và thiếu quyền `read:space` / `create:page` trên Space `GooPay Technical Team`).

- [x] **TASK-02 — Chạy Pre-Plan Critique (`DG0`) và Thiết kế Giải pháp + Code Demo (`S3 Solution + DG1/DG2`)**
  - **Context (`file:line`)**: `09_tasks_solution_confluence_space_404_2026_10_29.md`.
  - **Logs/Test**: Chạy `laya_gate.py run --stage dg0` và `laya_gate.py run --stage dg1`.
  - **Definition of Done (DoD)**: Hoàn tất `09_tasks_solution_confluence_space_404_2026_10_29.md` ánh xạ 1-1 (`F1`, `F2`) kèm hướng dẫn cấp quyền trên Confluence và Code Demo cải tiến `pkgs/confluence/client.go`; đạt `DG0 PASS` + `DG1+DG2 PASS`; dừng chờ User `APPROVE`.

- [ ] **TASK-03 — Tự động đồng bộ `SpaceKey` từ `ParentPageID` và bóc tách thông báo lỗi phân quyền Space `404 (authorized: true)` trong `pkgs/confluence/client.go` (`Fix F2`, sau `APPROVE`)**
  - **Context (`file:line`)**:
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go:209-264`
    - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client_test.go:71-230`
  - **Logs/Test**: `go test -v ./pkgs/confluence/... ./internal/service/confluence/...`
  - **Definition of Done (DoD)**:
    1. Khi `ParentPageID != ""`, nếu `resolveParentPageSpaceKey` lấy được `space.key` thực tế của trang cha khác với `req.SpaceKey`, tự động đồng bộ `req.SpaceKey = parentSpaceKey`.
    2. Khi Confluence trả `404` kèm `"authorized":true` và `"No space with key"`, trả về thông báo lỗi tiếng Việt/kỹ thuật rõ ràng chỉ đích danh tài khoản `req.UserEmail` thiếu quyền `View / Add Page` trên Space `req.SpaceKey` (hoặc chưa thuộc group `confluence-users-goopay`).
