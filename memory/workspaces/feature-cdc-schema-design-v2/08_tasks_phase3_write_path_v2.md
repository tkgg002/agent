# Tasks Phase 3 — Write Path V2 Foundation

- [x] Mở rộng `UpsertRecord` với schema/connection fields.
- [x] Bổ sung `ShadowConnectionKey` vào metadata route.
- [x] Nối `ConnectionManager` vào worker wiring.
- [x] Nâng `EventHandler` để emit V2-aware write metadata.
- [x] Nâng `BatchBuffer` để group theo `connection + schema + table`.
- [x] Nâng `SchemaAdapter` để support schema-qualified SQL.
- [x] Giữ backward compatibility cho legacy cache/lookups.
- [x] Verify package `internal/service`, `internal/handler`, `internal/server`.
- [ ] Phase tiếp theo: chuyển transmuter/master write path sang cùng model connection/schema-aware.
- [ ] Phase tiếp theo: bỏ dần compatibility phụ thuộc `TableRegistry` giả lập.
