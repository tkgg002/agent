# Phase 31 Requirements — Direct Scan Fields & Transform Status

## Mục tiêu
- Giảm thêm dependency của operator-flow vào `registry_id`.
- Ưu tiên các action/read-path có thể direct V2 hóa mà không làm gãy worker runtime.

## Audit conclusions
- `transform-status`: direct V2 hóa được ngay vì chỉ cần resolve `target_table`.
- `scan-fields`: direct V2 hóa được vì worker path hiện chỉ cần:
  - `target_table`
  - `source_table`
  - `sync_engine`
  - `source_type`
- `create-default-columns`: chưa direct an toàn vì worker path vẫn cần legacy metadata sâu hơn.

## Ràng buộc
- Phải giữ đúng 2 luồng:
  - auto-flow Debezium
  - cms-fe operator-flow
- Swagger annotations phải update cùng phase.
