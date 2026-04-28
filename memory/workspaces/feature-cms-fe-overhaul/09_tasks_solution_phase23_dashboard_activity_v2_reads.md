# Phase 23 Solution - Dashboard + ActivityManager V2 Reads

## Quyết định chính

- `GET /api/registry/stats` là read-only compatibility surface nên có thể thay ngay bằng V2 read-model.
- `ActivityManager` chỉ dùng `/api/registry` để enrich scope hiển thị; phần này nên chuyển sang `/api/v1/source-objects` ngay.
- `PATCH /api/registry/:id` và `POST /api/registry/batch` chưa bọc V2 ở phase này vì đó vẫn là mutation legacy thật.

## Kết quả mong muốn

- Dashboard và activity scheduling nhìn metadata theo V2 nhiều hơn.
- FE surface giảm phụ thuộc vào `/api/registry` mà không làm CMS operator-flow bị “vỏ”.

## Kết quả thực tế

- Dashboard đã dùng `GET /api/v1/source-objects/stats`.
- ActivityManager đã dùng `GET /api/v1/source-objects`.
- `PATCH /api/registry/:id`, `POST /api/registry`, `POST /api/registry/batch` được giữ nguyên như compatibility shell.
- Backend tests và frontend build đều pass.
- Swagger annotations đã cập nhật nhưng generated docs chưa regen được do thiếu `swag`.
