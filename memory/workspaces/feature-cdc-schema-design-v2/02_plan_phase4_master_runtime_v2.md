# Plan Phase 4 — Master Runtime V2

## English

1. Move transmuter runtime off legacy `cdc_internal.master_table_registry` and `public.<master>`.
2. Resolve `master_table` requests into V2 `master_binding`.
3. Read shadow rows through `shadow_binding + connection key + schema`.
4. Write master rows through `master_binding + connection key + schema`.
5. Move master DDL generation to V2 metadata and V2 mapping rules.
6. Keep NATS subjects and payload names stable for compatibility.

## Tiếng Việt

1. Kéo transmuter runtime ra khỏi assumption cũ `cdc_internal.master_table_registry` và `public.<master>`.
2. Resolve `master_table` sang `master_binding` của V2.
3. Đọc shadow rows theo `shadow_binding + connection key + schema`.
4. Ghi master rows theo `master_binding + connection key + schema`.
5. Chuyển master DDL generator sang metadata V2 và mapping rules V2.
6. Giữ nguyên subject/payload NATS để không làm gãy caller hiện tại.
