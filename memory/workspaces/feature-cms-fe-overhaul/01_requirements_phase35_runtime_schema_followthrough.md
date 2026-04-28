# Requirements — Phase 35 Runtime Schema Follow-through

## Mục tiêu

- Tiếp tục bóc các assumption `public` còn caller thật trong worker runtime.
- Tập trung vào các path operator/maintenance còn sống:
  - `discover`
  - `backfill`
  - `schema inspector`
  - `pending field schema lookup`

## Ràng buộc

- Không lan sang compatibility shell phía CMS nếu chưa là runtime nóng.
- Không đổi contract external API ở phase này.
- Giữ `auto-flow` Debezium và `cms-fe operator-flow` ổn định.

## Definition of Done

- `discover` và `backfill` resolve đúng `shadow_schema`.
- `SchemaInspector` không còn assume `public` khi đọc column list.
- `go test ./internal/service ./internal/handler ./internal/server ./internal/repository` pass.
