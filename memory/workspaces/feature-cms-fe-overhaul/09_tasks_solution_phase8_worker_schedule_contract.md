# Solution — Phase 8 Worker Schedule Contract

## Hướng xử lý

- Không tạo endpoint mới.
- Giữ `worker-schedule` là API của operator-flow, nhưng làm nó đủ dữ liệu để FE không phải dựng thêm nghĩa cho danh sách schedule.
- Dùng metadata V2 (`cdc_system.shadow_binding`, `cdc_system.source_object_registry`) để:
  - enrich response
  - resolve scope create request
- Giữ fallback `target_table` để không làm gãy dữ liệu hiện có.

## Definition of Done

1. `GET /api/worker-schedule` trả đủ source/shadow context.
2. `POST /api/worker-schedule` chấp nhận scope giàu hơn và resolve được từ metadata V2.
3. `ActivityManager` dùng API mới cho list view thay vì tự đoán hoàn toàn.
4. Swagger/comment của endpoint được cập nhật đồng bộ.
5. `go test ./...` và `npm run build` pass.
