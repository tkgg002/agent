# Solution — Phase 34 Shadow Schema Runtime Hardening

## Vấn đề gốc

Sau khi direct-V2 hóa dần operator actions ở CMS, runtime worker vẫn còn nhiều helper assume `public`. Kết quả là:

- API/UX có thể đã trỏ đúng `source_object_id`
- nhưng command thật vẫn có nguy cơ đọc/ghi nhầm `public.<target_table>`

## Cách giải

- Mở lookup `target_table -> ResolvedSourceRoute` ở metadata registry.
- Dùng route này để resolve `shadow_schema` cho worker helper thay vì bám `TableRegistry` tổng hợp.
- Qualify toàn bộ SQL ở các operator path đang chạy thật bằng `schema.table`.

## Kết quả

- `batch-transform`, `scan raw`, `periodic scan`, `drop index`, `scan-fields`, `schema validation` đã bớt phụ thuộc vào `public`.
- Operator-flow V2 bây giờ nhất quán hơn giữa:
  - FE namespace
  - CMS API
  - worker runtime

## Chưa làm trong phase này

- Chưa refactor toàn bộ helper legacy còn lại trên mọi nhánh.
- Chưa đụng sâu vào các path compatibility-only không còn là runtime chính.
