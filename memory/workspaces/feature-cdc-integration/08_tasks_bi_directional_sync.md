# Task Checklist: Bi-directional Sync Implementation

## Phase 1: Khám phá & Đối soát (Discovery & Reconciliation)
- [x] Bổ sung API `GetConnection` và `ListConnections` (nếu chưa có hoặc chưa đầy đủ) vào `AirbyteClient`.
- [x] Xây dựng function `CompareCatalog` để phát hiện sai lệch giữa Registry và Airbyte Connection.
- [x] Tạo API endpoint `/api/airbyte/sync-audit` để hiển thị các bảng bị "mismatched" trạng thái.

## Phase 2: Cơ chế Reconciliation Loop (Worker)
- [x] Khởi tạo `BackgroundWorker` trong CDC CMS Service.
- [x] Implement logic Polling định kỳ (mỗi 5-10 phút) để đồng bộ trạng thái từ Airbyte về DB Registry.
- [x] Xử lý kịch bản: Hợp nhất (Merge) thay đổi khi cả hai bên cùng thay đổi đồng thời.

## Phase 3: Tính năng Smart Import
- [x] Xây dựng UI/API cho phép liệt kê các Connection hiện có trên Airbyte.
- [x] Implement logic "Import" tự động tạo registry entries từ Airbyte Connection Catalog.

## Phase 4: Kiểm thử & Onboarding
- [ ] Unit test cho logic so khớp Catalog.
- [ ] Manual test: Tắt stream trên Airbyte UI và kiểm tra DB sau loop cycle.
