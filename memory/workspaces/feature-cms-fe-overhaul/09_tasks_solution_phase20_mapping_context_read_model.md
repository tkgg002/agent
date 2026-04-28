# Solution — Phase 20 Mapping Context Read Model

## Vấn đề

`MappingFieldsPage` trước đó đang gọi `GET /api/registry`, tải cả danh sách rồi tự tìm row theo `id`. Đây là dependency compatibility tệ, vừa nặng vừa giữ sai source-of-truth.

## Giải pháp

- Thêm `GET /api/v1/source-objects/registry/{registry_id}`
- Endpoint này lấy `registry_id` làm bridge identity, nhưng enrich toàn bộ context từ metadata V2
- Refactor page mappings dùng endpoint mới
- Giữ action legacy `create-default-columns` qua `registry_id`

## Kết quả

- Page mappings không còn phụ thuộc vào full registry list
- FE gọn hơn và đúng kiến trúc hơn
- Operator không mất action thực chiến hiện tại
