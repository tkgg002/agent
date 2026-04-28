# Plan Phase 2 — Metadata Registry

## English

1. Introduce a V2-aware metadata registry service that reads source/shadow routing from `cdc_system.*`.
2. Keep the current ingest runtime stable by preserving compatibility for legacy mapping rules.
3. Move `EventHandler` and `DynamicMapper` to depend on an interface rather than the concrete legacy registry.
4. Wire the worker server to the new metadata registry service.
5. Validate service/handler/server package compilation and unit tests.

## Tiếng Việt

1. Tạo `MetadataRegistryService` đọc routing source/shadow từ `cdc_system.*`.
2. Giữ luồng ingest hiện tại ổn định bằng cách bảo toàn compatibility cho legacy mapping rules.
3. Chuyển `EventHandler` và `DynamicMapper` sang phụ thuộc vào interface thay vì `RegistryService` cũ.
4. Nối `worker_server` sang service mới.
5. Verify bằng test ở các package `service`, `handler`, `server`.
