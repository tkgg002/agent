# Phase 23 Requirements - Dashboard + ActivityManager V2 Reads

## Mục tiêu

- Giảm tiếp dependency read-only của CMS FE vào `/api/registry`.
- Giữ operator-flow thực chiến cho monitoring / scheduling, không cắt nhầm luồng backup-retry-reconcile.
- Chỉ thay phần read/enrichment đã có semantics V2 đủ rõ; không bọc giả mutation shell legacy.

## Yêu cầu chức năng

1. `Dashboard` không còn đọc `GET /api/registry/stats`.
2. `ActivityManager` không còn đọc list scope từ `GET /api/registry`.
3. Phải có read endpoint V2-native cho source-object stats đủ thay thế dashboard summary cũ.
4. `ActivityManager` phải lấy source/shadow scope từ `GET /api/v1/source-objects`.
5. `PATCH /api/registry/:id` và `POST /api/registry/batch` chưa đổi ở phase này.

## Yêu cầu API

- Thêm `GET /api/v1/source-objects/stats`.
- Cập nhật swagger annotations cùng phase.
- Không thêm API mutation mới nếu semantics backend vẫn còn legacy thật.

## Definition of Done

- `Dashboard.tsx` dùng endpoint V2 mới.
- `ActivityManager.tsx` dùng `GET /api/v1/source-objects`.
- `go test ./...` ở `cdc-cms-service` pass.
- `npm run build` ở `cdc-cms-web` pass.
- Có file vật lý đầy đủ cho phase này trong workspace.
