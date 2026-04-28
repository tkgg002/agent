# Tasks Phase 2 — Metadata Registry

- [x] Tạo `MetadataRegistry` interface.
- [x] Tạo `MetadataRegistryService` đọc source/shadow routing từ V2 tables.
- [x] Giữ compatibility cho mapping rules legacy.
- [x] Chuyển `DynamicMapper` sang interface.
- [x] Chuyển `EventHandler` sang `ResolveSourceRoute`.
- [x] Chuyển `WorkerServer` sang tạo service mới.
- [x] Thêm test cho route resolution.
- [x] Verify package `internal/service`, `internal/handler`, `internal/server`.
- [ ] Phase tiếp theo: thay `BatchBuffer`/`UpsertRecord` để mang `schema + connection key`.
- [ ] Phase tiếp theo: cắm `ConnectionManager` vào ingest write path.
