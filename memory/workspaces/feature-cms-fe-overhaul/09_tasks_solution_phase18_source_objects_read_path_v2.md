# Solution — Phase 18 Source Objects Read Path V2

## Vấn đề

`TableRegistry` đã đổi ngôn ngữ sang `Source Objects`, nhưng read path vẫn còn lấy từ `/api/registry`. Điều này khiến page tiếp tục coi legacy registry là source-of-truth, trong khi metadata V2 thật đã nằm ở `cdc_system`.

## Giải pháp

- Dựng `GET /api/v1/source-objects` làm read-model V2 cho page này.
- Giữ `/api/registry` cho write path/operator actions còn sống.
- Trả thêm `registry_id` bridge để FE biết action nào còn chạy được qua compatibility layer.
- Với row không có bridge:
  - vẫn hiển thị để monitoring
  - nhưng các action legacy bị disable rõ ràng

## Kết quả

- `TableRegistry` giờ đọc từ V2 metadata thật.
- Operator vẫn dùng được action cũ ở những row còn bridge hợp lệ.
- UI trung thực hơn với trạng thái chuyển đổi hiện tại, không “giả capability”.
