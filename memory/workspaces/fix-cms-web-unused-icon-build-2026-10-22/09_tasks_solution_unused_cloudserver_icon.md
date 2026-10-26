---
tier: T2
suffix: unused_cloudserver_icon
workspace: fix-cms-web-unused-icon-build-2026-10-22
---

# 09 Tasks Solution: Giải quyết lỗi `TS6133: 'CloudServerOutlined' is declared but its value is never read` tại `cdc-cms-web/src/App.tsx`

## 1. Kết quả Điều tra Nguyên nhân Gốc (`S2 Investigate` — `VERIFIED` 100% bằng `git log` & `file:line`)

### 1.1. Tại sao `CloudServerOutlined` lại xuất hiện trong `cdc-cms-web/src/App.tsx:14`?
Qua truy vết lịch sử Git của `cdc-cms-web/src/App.tsx` (`git log -p -n 3 -- src/App.tsx`):
1. **Commit `ac8a401` (`2026-10-12 15:55:19 +0700`, `"update nat monitor, fix number count update master"`):**
   - Lần đầu thêm trang `NatsMonitor` (`src/App.tsx:35`, route `/nats-monitor` tại `src/App.tsx:228`) và thêm `CloudServerOutlined` vào dòng `14` để dùng làm icon cho mục menu Sidebar `/nats-monitor` (`src/App.tsx:140-144`), nhưng mục menu này được đóng comment (`//`).
2. **Commit `fc189e6` (`2026-10-15 16:14:34 +0700`, `"update count, report daily"`):**
   - Đã xóa dòng `CloudServerOutlined` khỏi `src/App.tsx:14` để không bị lỗi unused import khi mục menu `/nats-monitor` đang đóng comment.
3. **Commit `67be11a` (`2026-10-20 14:30:51 +0700`, `"update nats monitor"` — thuộc đợt làm `feature-nats-monitor-v2-2026-10-19`):**
   - Khi cập nhật `src/pages/NatsMonitor.tsx`, commit này đã **thêm lại `CloudServerOutlined` vào `src/App.tsx:14`** và sửa nhãn ở dòng `143` từ `// label: <Link to="/nats-monitor">NATS Monitor</Link>,` thành `// label: <Link to="/nats-monitor">Giám sát NATS</Link>,`, **nhưng toàn bộ khối menu tại `src/App.tsx:140-144` vẫn đang bị comment (`//`)**:
     ```tsx
     // cdc-cms-web/src/App.tsx:5-15
     import {
       DashboardOutlined,
       DatabaseOutlined,
       BranchesOutlined,
       SettingOutlined,
       LogoutOutlined,
       CompassOutlined,
       ThunderboltOutlined,
       LockOutlined,
       CloudServerOutlined, // <-- Dòng 14: được import nhưng không nơi nào đọc!
     } from '@ant-design/icons';
     ```
     ```tsx
     // cdc-cms-web/src/App.tsx:140-144
             // {
             //   key: '/nats-monitor',
             //   icon: <CloudServerOutlined />,
             //   label: <Link to="/nats-monitor">Giám sát NATS</Link>,
             // },
     ```

### 1.2. Tại sao lỗi này gây gãy `npm run build:prod` trong Docker (`Dockerfile:11`) và bị bỏ lọt ở phiên trước?
- Tại `cdc-cms-web/Dockerfile:11`, bước build image chạy `RUN npm run build:prod`.
- Tại `cdc-cms-web/package.json:8-9`, lệnh `"build:prod"` là `"tsc -b && vite build --mode production"`.
- Khi `tsc -b` chạy, nó biên dịch theo `cdc-cms-web/tsconfig.app.json:20-22` có bật `"strict": true` và `"noUnusedLocals": true`, nên báo lỗi cứng:
  `src/App.tsx(14,3): error TS6133: 'CloudServerOutlined' is declared but its value is never read.` (`exit code: 2`).
- Ngược lại, nếu chỉ chạy `npx tsc --noEmit` (do `tsconfig.json:1-6` để `"files": []` với project references) hoặc chỉ chạy `npx vite build` (như phiên `fix-mongo-jsonb-extjson-compat-2026-10-22` đã lách sau khi gặp lỗi tại biên lai `EV-20261022T112136-5b73.json`), bước `tsc -b` bị bỏ qua nên không chặn lại kịp thời.

---

## 2. Phương án Tối ưu Duy nhất (`S3 Solution`) & Code Demo

### 2.1. Thiết kế Giải pháp
Theo đúng quy ước hiện hữu trong `cdc-cms-web/src/App.tsx`:
- Các mục menu đang được ẩn bằng comment trong `menuItems` (như `/source-to-master` tại `L88-92`, `/schema-proposals` tại `L115-119`, `/nats-monitor` tại `L140-144`, `/schema-changes` tại `L152-156`, và `/audit` tại `L157-161` dùng `// icon: <SafetyCertificateOutlined />,`) **không import icon chỉ dành riêng cho mục đang bị comment** ở đầu file (ví dụ `SafetyCertificateOutlined` hoàn toàn không nằm trong khối `import { ... } from '@ant-design/icons'` tại `L5-15`).
- Do đó, xóa dòng `14` (`CloudServerOutlined,`) khỏi khối import `@ant-design/icons` trong `cdc-cms-web/src/App.tsx:5-15`. Toàn bộ `npx tsc -b` và `npm run build:prod` trên `cdc-cms-web` hiện chỉ có duy nhất 1 lỗi này (`Found 1 error.`).

### 2.2. Code Demo (`cdc-cms-web/src/App.tsx:5-15`)
```tsx
import {
  DashboardOutlined,
  DatabaseOutlined,
  BranchesOutlined,
  SettingOutlined,
  LogoutOutlined,
  CompassOutlined,
  ThunderboltOutlined,
  LockOutlined,
} from '@ant-design/icons';
```

## 3. Kế hoạch Kiểm chứng Sau Khi Có `APPROVE` (`S4 -> S5`)
1. Tạo git restore-point, xóa dòng `CloudServerOutlined,` tại `cdc-cms-web/src/App.tsx:14`.
2. Chạy `npx tsc -b` và `npm run build:prod` thông qua `evidence_run.py` để lấy biên lai `EV-...` xác nhận `exit=0`.
3. Thực hiện kiểm định `S5` (`git diff`, `DG3` qua `laya_gate.py`, `audit_report_unused_cloudserver_icon.md`) và `verify_governance.py`.
