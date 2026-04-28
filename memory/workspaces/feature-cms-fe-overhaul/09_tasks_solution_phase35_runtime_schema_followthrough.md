# Solution — Phase 35 Runtime Schema Follow-through

## Vấn đề gốc

Sau Phase 34, phần lớn command/operator path chính đã hiểu `shadow_schema`, nhưng vẫn còn một “đuôi” nguy hiểm:

- `discover` nhìn column list ở `public`
- `backfill` update vào table không qualify schema
- `SchemaInspector` vẫn dùng schema cache theo `tableName` đơn lẻ

## Cách giải

- Tách helper `GetTableColumnsInSchema()`.
- Cho `SchemaInspector` resolve schema từ metadata V2 và cache theo `schema.table`.
- Cho `discover/backfill` bám chung `resolveTargetSchema()`.

## Kết quả

- Worker maintenance paths bớt lệch giữa metadata V2 và physical shadow schema.
- Drift detection, discover và backfill nhất quán hơn với shadow convention hiện tại.

## Chưa làm trong phase này

- Chưa đụng `cms-service/internal/repository/registry_repo.go` vì đó vẫn là compatibility shell phía CMS.
- Chưa bóc toàn bộ helper legacy còn assumption `public` ở mọi nhánh ít dùng hơn.
