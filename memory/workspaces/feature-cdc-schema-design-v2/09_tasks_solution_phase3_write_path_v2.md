# Solution Phase 3 — Write Path V2 Foundation

## What changed architecturally

Trước phase này:
- route biết source -> shadow
- nhưng write path vẫn chỉ thực sự hiểu `tableName`

Sau phase này:
- record ingest mang:
  - connection role
  - connection key
  - schema
  - table
- batch buffer group theo identity destination thật
- schema adapter generate SQL theo schema-qualified table
- event delete path cũng bắt đầu đi theo route thật

## Why this matters

Đây là bước biến kiến trúc từ "V2 trên giấy" thành "V2 đã chạm runtime".

Nó chưa phải full end-state, nhưng đã giải quyết điểm nghẽn lớn nhất:
- write path không còn bị trói cứng vào một `tableName` flat nữa

## Remaining work

1. `DynamicMapper` vẫn là compatibility-driven cho shadow path.
2. `TransmuterModule` và master DDL/runtime vẫn còn assumption cũ.
3. Nhiều flow recon/dlq vẫn dựa trên legacy `TableRegistry`.

## Recommended next slice

1. Chuyển master/transmuter sang:
   - `master_binding`
   - `ConnectionManager`
   - `schema-qualified` writes
2. Sau đó mới dọn compatibility legacy còn lại.
