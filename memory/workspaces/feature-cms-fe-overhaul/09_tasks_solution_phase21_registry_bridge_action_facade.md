# Solution — Phase 21 Registry Bridge Action Facade

## Vấn đề

Sau Phase 20, nhiều read path của FE đã sạch hơn, nhưng một số operator actions vẫn còn gọi trực tiếp `/api/registry/...`. Điều này làm FE vẫn lộ compatibility shell quá rõ, dù các action đó thực tế chỉ đang dùng `registry_id` như bridge.

## Giải pháp

- Không đổi semantics backend giả tạo
- Thêm facade endpoints dưới:
  - `/api/v1/source-objects/registry/:id/...`
- FE chuyển sang facade mới
- backend vẫn delegate về `RegistryHandler`

## Kết quả

- FE-facing API gọn và đúng vocabulary hơn
- operator-flow không bị gãy
- hệ thống vẫn trung thực về việc bridge legacy còn tồn tại ở backend
