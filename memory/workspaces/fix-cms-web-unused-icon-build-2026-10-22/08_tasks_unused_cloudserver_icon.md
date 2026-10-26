# 08 Tasks: fix-cms-web-unused-icon-build-2026-10-22

## TASK-UI-01: Xóa import dư thừa `CloudServerOutlined` tại `cdc-cms-web/src/App.tsx` và kiểm chứng `npm run build:prod`
- **Mô tả & Context `file:line`:**
  - File mục tiêu: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/App.tsx:5-15` và `140-144`.
  - Hiện trạng: Dòng `14` của `src/App.tsx` import `CloudServerOutlined` từ `@ant-design/icons`, nhưng nơi duy nhất dùng icon này là mục menu `/nats-monitor` tại `src/App.tsx:140-144` đang bị comment (`//`). Khi chạy `npm run build:prod` (`package.json:9`: `tsc -b && vite build --mode production`, được gọi bởi `Dockerfile:11`), trình biên dịch TypeScript với `"noUnusedLocals": true` (`tsconfig.app.json:21`) báo lỗi `error TS6133: 'CloudServerOutlined' is declared but its value is never read.` (`exit code: 2`).
  - Hành động (sau khi có `APPROVE` của User): Xóa dòng `14` (`CloudServerOutlined,`) khỏi khối import `@ant-design/icons` trong `cdc-cms-web/src/App.tsx` (đồng nhất với pattern của mục `/audit` tại `src/App.tsx:157-161` đang comment `// icon: <SafetyCertificateOutlined />,` và không import `SafetyCertificateOutlined` ở đầu file).
- **Logs / Test Commands:**
  - Chạy qua `evidence_run.py`:
    - `npx tsc -b` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`
    - `npm run build:prod` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`
- **Definition of Done (DoD):**
  - `npx tsc -b` và `npm run build:prod` hoàn tất với `exit code: 0`, không còn bất kỳ lỗi `TS6133` nào.
  - Hoàn tất kiểm định S5 (`git diff`, `DG3` qua `laya_gate.py`, `audit_report_unused_cloudserver_icon.md`) và `verify_governance.py` đạt `PASSED`.
