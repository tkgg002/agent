# Requirements — Schema Design V2

## 1. Business Requirements

1. Hệ thống phải nhận dữ liệu từ nhiều nguồn khác nhau:
   - MongoDB
   - MariaDB / MySQL
   - PostgreSQL
2. Mỗi source có thể chứa nhiều namespace cha:
   - Mongo: `database -> collection`
   - MariaDB / PostgreSQL: `database -> schema -> table`
3. Mỗi source object phải được chọn destination riêng cho từng layer:
   - `shadow`
   - `master`
   - `system`
4. `shadow` phải giữ được lineage với namespace cha của source; không được flatten toàn bộ về một cụm `target_table`.
5. `master` có thể được route sang connection/schema khác với `shadow`.
6. `system` là metadata/control plane riêng, không được lẫn với shadow/master payload tables.
7. Một source object có thể phát sinh:
   - 0..1 shadow binding active
   - 0..N master bindings active
8. Kiến trúc mới phải hỗ trợ migrate dần từ hệ thống cũ đang chạy, không yêu cầu "big bang rewrite".

## 2. Technical Requirements

1. Không dùng `target_table` làm identity chính nữa.
2. Identity phải phân biệt được:
   - connection vật lý
   - engine
   - database/catalog
   - schema/namespace
   - table/collection
3. Mapping rules phải bind vào logical object/binding, không phụ thuộc duy nhất vào chuỗi `source_table`.
4. Worker phải chọn connection động theo binding, không dùng một `*gorm.DB` duy nhất cho mọi read/write.
5. DDL generator phải tạo đúng namespace cha trước khi tạo table.
6. Runtime phải support per-layer routing:
   - ingest -> shadow binding
   - transmute -> master binding
   - admin/CMS -> system connection
7. Tất cả flow phải trace được lineage:
   - source object -> shadow binding -> shadow physical object
   - source/shadow -> master binding -> master physical object
8. Thiết kế phải hỗ trợ mở rộng engine đích sau này, ít nhất không khóa cứng ở Postgres-only abstraction.

## 3. Non-Functional Requirements

1. Backward compatibility cho flow đang chạy.
2. Cho phép rollout theo phase.
3. Tối thiểu hóa blast radius trong code:
   - Phase đầu ưu tiên thêm registry/binding mới song song với bảng cũ.
4. Tối ưu vận hành:
   - connection pooling theo destination
   - cache registry/binding
   - secret tách khỏi code/config cứng
5. Bảo mật:
   - connection string không lưu plain text trong code
   - system DB có boundary rõ

## 4. Constraints From Current Codebase

1. `RegistryService` hiện build cache dựa trên `target_table` và `source_table`.
2. `EventHandler`, `ReconHandler`, `CommandHandler`, `BackfillSourceTS` đang query trực tiếp `cdc_table_registry`.
3. `MasterDDLGenerator` và `TransmuterModule` đang assume:
   - shadow ở `cdc_internal`
   - master ở `public`
4. `config` và `pkgs/database` hiện chưa có abstraction cho multiple sinks.
5. Nhiều migration cũ vẫn song song tồn tại:
   - `cdc_table_registry`
   - `cdc_internal.table_registry`
   - `cdc_internal.master_table_registry`
   - `cdc_internal.sources`

## 5. Definition of Done

1. Có bản thiết kế V2 chi tiết cho schema metadata/control plane.
2. Có DDL đề xuất cụ thể cho các bảng mới.
3. Có mapping rõ từ bảng cũ sang bảng mới.
4. Có đề xuất thay đổi runtime theo từng module/file của dự án.
5. Có lộ trình rollout/migration để không làm gãy flow hiện hữu.
