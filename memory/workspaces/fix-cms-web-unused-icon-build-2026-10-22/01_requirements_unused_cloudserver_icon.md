---
tier: T2
suffix: unused_cloudserver_icon
workspace: fix-cms-web-unused-icon-build-2026-10-22
---

# 01 Requirements: Sửa lỗi `TS6133: 'CloudServerOutlined' is declared but its value is never read` tại `cdc-cms-web/src/App.tsx`

## 1. Bối cảnh & Yêu cầu của User
User chạy Docker build cho `cdc-cms-web` (`RUN npm run build:prod`) và gặp lỗi:
```text
7.588 src/App.tsx(14,3): error TS6133: 'CloudServerOutlined' is declared but its value is never read.
------
ERROR: failed to build: failed to solve: process "/bin/sh -c npm run build:prod" did not complete successfully: exit code: 2
```
Câu hỏi & yêu cầu của User: `"mày thêm cái CloudServerOutlined vào chi vậy"` — làm rõ nguyên nhân vì sao `CloudServerOutlined` xuất hiện tại `cdc-cms-web/src/App.tsx:14` và trình phương án xử lý dứt điểm để `npm run build:prod` (`tsc -b && vite build --mode production`) chạy sạch `exit code: 0`.

## 2. Danh sách Yêu cầu (Requirements Traceability)
- **R1 (Truy vết nguyên nhân gốc `file:line` & `git log`):** Giải thích chính xác commit nào đã thêm `CloudServerOutlined` vào `cdc-cms-web/src/App.tsx:14`, mục đích ban đầu của nó tại `src/App.tsx:140-144` (mục menu `/nats-monitor` đang bị comment out), và lý do vì sao lỗi lọt qua bước kiểm tra trước đó.
- **R2 (Xử lý triệt để lỗi `TS6133` tại `cdc-cms-web/src/App.tsx`):** Loại bỏ import dư thừa `CloudServerOutlined` tại `cdc-cms-web/src/App.tsx:14` (do mục menu `/nats-monitor` tại `src/App.tsx:140-144` đang chủ đích đóng comment `//` giống `/source-to-master`, `/schema-proposals`, `/schema-changes`, `/audit` — nơi icon `SafetyCertificateOutlined` tại `L159` cũng không import ở đầu file), đảm bảo `npm run build:prod` (`tsc -b && vite build --mode production`) đạt `exit 0`.
