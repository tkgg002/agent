# Context & Scope: DDL Status in Edit Modal & Snapshot V2 Sync Fix

## 1. Problem Statement
- **Issue**: Giao diện Snapshot V2 cần cờ `overwrite` động dạng true/false từ người dùng để reset tiến trình đồng bộ dữ liệu thay vì hardcode hoặc phớt lờ.

## 2. Scope of Work
- **Frontend (cdc-cms-web)**:
  - Bổ sung ô chọn chế độ Snapshot (Tiếp tục / Chạy lại từ đầu) bằng Radio.Group vào content của Modal.confirm trong `handleSnapshot` ở file `src/pages/TableRegistry.tsx`.
  - Truyền động giá trị `overwrite` đã chọn vào API.
