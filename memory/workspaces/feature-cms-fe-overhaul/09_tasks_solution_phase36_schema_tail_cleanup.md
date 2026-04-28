# Solution — Phase 36 Schema Tail Cleanup

## Vấn đề gốc

Sau các phase 34-35, phần nóng của worker đã schema-aware hơn rõ rệt, nhưng vẫn còn một “đuôi” không sạch:

- `event_bridge` poll query còn assume `public`
- `transform_service` chưa có đường resolve schema nếu được dùng lại
- `cms registry_repo` chưa có API helper schema-aware, khiến compatibility shell dễ quay lại `public`

## Cách giải

- Hardening nhẹ nhưng đúng chỗ:
  - resolve schema từ `shadow_binding` trong `event_bridge`
  - chuẩn bị metadata-aware transform helper
  - thêm wrappers `...InSchema()` cho CMS repo

## Kết quả

- Những tail helpers này giờ không còn khóa cứng vào `public`.
- Repo có nền tốt hơn cho các phase kế tiếp nếu cần dọn nốt compatibility shell.

## Chủ đích không làm

- Không đổi external API contract.
- Không ép refactor sâu những path hiện chưa có caller runtime thật.
