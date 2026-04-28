# Plan — Schema Design V2

## English

1. Verify the current CDC architecture directly from the repository.
   - Inspect models, migrations, repositories, services, handlers, and config.
   - Identify where the system is coupled to `target_table`, `cdc_internal`, and a single Postgres sink.
2. Define the V2 domain model.
   - Separate control-plane metadata from shadow/master payload storage.
   - Model physical connections, logical source objects, and per-layer bindings independently.
3. Design the V2 metadata schema.
   - Propose normalized tables for connections, source objects, shadow bindings, master bindings, schemas, and runtime state.
   - Include indexes, constraints, and uniqueness rules.
4. Map V2 to the existing project.
   - Specify which current tables can be deprecated, mirrored, or transformed.
   - Specify which code modules must be adapted first and which can remain temporarily compatible.
5. Define rollout phases.
   - Add V2 metadata alongside V1.
   - Introduce dynamic connection manager.
   - Move ingest to shadow bindings.
   - Move transmute/master DDL to master bindings.
   - Retire legacy assumptions after verification.

## Tiếng Việt

1. Xác minh kiến trúc CDC hiện tại trực tiếp từ repository.
   - Đọc model, migration, repository, service, handler, config.
   - Xác định chính xác chỗ nào đang khóa cứng vào `target_table`, `cdc_internal`, và một Postgres sink duy nhất.
2. Định nghĩa domain model V2.
   - Tách control plane metadata khỏi payload storage của shadow/master.
   - Mô hình hóa độc lập: physical connection, logical source object, và binding theo từng layer.
3. Thiết kế metadata schema V2.
   - Đề xuất bộ bảng chuẩn hóa cho connections, source objects, shadow bindings, master bindings, namespace, runtime state.
   - Kèm index, constraint, uniqueness rule.
4. Ánh xạ V2 vào dự án hiện tại.
   - Chỉ rõ bảng cũ nào sẽ deprecate, mirror, hay transform.
   - Chỉ rõ module code nào phải đổi trước và module nào có thể giữ backward compatibility tạm thời.
5. Chốt rollout phases.
   - Thêm metadata V2 song song V1.
   - Bổ sung dynamic connection manager.
   - Chuyển ingest sang shadow bindings.
   - Chuyển transmute/master DDL sang master bindings.
   - Gỡ assumption cũ sau khi verify.
