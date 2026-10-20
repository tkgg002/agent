# 08 Tasks: Danh mục Task Tích hợp Master MongoDB Destination vào Hệ thống

> **Workspace**: `feat-master-mongo-destination-2026-10-20`
> **Tier**: T1 (Task Breakdown & Codebase Traceability)

---

## Nhóm 1: Quản lý Kết nối Master Mongo (`cdc-cms-service` & `cdc-cms-web`)

- [ ] **TASK-M01 — Mở rộng API Create/Update Master Connection hỗ trợ MongoDB**
  - **Context (`file:line`)**: `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:172-249` (hiện đang gán cứng `EngineType: "postgresql"`, `port = 5432` và build DSN `postgres://`).
  - **Mô tả**: Cho phép nhận `engine_type = "mongodb"`, default `port = 27017`, hỗ trợ build/lưu `mongodb://` URI (kèm `authSource`, `replicaSet`, `tls`).
  - **Test / DoD**: Gọi `POST /api/v1/master-connections` với `engine_type: "mongodb"` lưu thành công vào `cdc_system.connection_registry`.

- [ ] **TASK-M02 — Mở rộng API Test Connection hỗ trợ ping MongoDB**
  - **Context (`file:line`)**: `cdc-cms-service/internal/api/master/master_registry_handler_connections.go:64-158` (hiện chỉ gọi `gorm.Open(postgres.Open(dsn))` tại dòng `122`).
  - **Mô tả**: Phân nhánh theo `engine_type == "mongodb"` hoặc prefix `mongodb://` để dùng `mongo.Connect` + `client.Ping(ctx, readpref.Primary())` và lấy `buildInfo` version.
  - **Test / DoD**: Gọi `POST /api/v1/master-connections/test` tới MongoDB trả về `{"success": true, "latency_ms": ..., "version": "MongoDB ..."}`.

- [ ] **TASK-M03 — Cập nhật Giao diện Quản lý Kết nối Đích trên CMS Web**
  - **Context (`file:line`)**: `cdc-cms-web/src/pages/MasterRegistry.tsx`.
  - **Mô tả**: Thêm lựa chọn `Engine Type` (`PostgreSQL` | `MongoDB`) trong Modal Quản lý Kết nối Đích và hiển thị tag Engine trên bảng Master Registry.
  - **Test / DoD**: `npm run build` sạch lỗi; tạo và test kết nối Mongo trực tiếp trên UI thành công.

---

## Nhóm 2: Connection Pool & Provisioning Collection/Index (`centralized-data-service`)

- [ ] **TASK-M04 — Bổ sung Master Mongo Client Pool vào `ConnectionManager`**
  - **Context (`file:line`)**: `centralized-data-service/internal/service/source/connection_manager.go:18-194` (hiện chỉ có `masterDBPool map[string]*gorm.DB` tại dòng `25` và `GetMasterDB` tại dòng `58`).
  - **Mô tả**: Thêm `masterMongoPool map[string]*mongo.Client`, hàm `GetMasterMongoClient(ctx, key)` đọc URI từ `cdc_system.connection_registry`, và đóng kết nối trong `InvalidatePool` / `Close`.
  - **Test / DoD**: Unit test & integration test khởi tạo, cache và invalidate `*mongo.Client` theo `connection_code`.

- [ ] **TASK-M05 — Hỗ trợ Provisioning Collection & Index trong `MasterDDLGenerator`**
  - **Context (`file:line`)**: `centralized-data-service/internal/service/master/master_ddl_generator.go:370-486`.
  - **Mô tả**: Khi `engine_type == "mongodb"`, thay vì thực thi SQL `CREATE TABLE / ALTER TABLE`, gọi `CreateCollection` và `Indexes().CreateMany()` cho `_gpay_id` (unique), `_source_id`, `_source_ts` và các business key khi Approve Schema.
  - **Test / DoD**: Approve Master Binding trỏ tới kết nối Mongo tự động tạo Collection & Index trên MongoDB đích.

---

## Nhóm 3: Transmute Engine & BSON Type Coercion (`centralized-data-service`)

- [ ] **TASK-M06 — Xây dựng bộ chuyển đổi kiểu dữ liệu BSON (`mongo_type_converter.go`)**
  - **Context (`file:line`)**: `centralized-data-service/internal/service/master/` & tham chiếu `memory/workspaces/feat-cdc-postgres-to-mongo-3tables/08_tasks_cdc_pg_to_mongo_3tables.md:18`.
  - **Mô tả**: Chuyển đổi chuẩn xác từ `_raw_data` / `mapping_rule` sang BSON (`UUID`/`BinData:04` ➔ string `8-4-4-4-12`, `ObjectId` ➔ `primitive.ObjectID`, `NUMERIC` ➔ `primitive.Decimal128`, `TIMESTAMPTZ` ➔ `primitive.DateTime`, JSONB/Nested ➔ `bson.M`/`bson.A`).
  - **Test / DoD**: Pass 100% unit test ép kiểu cho mọi loại dữ liệu nguồn.

- [ ] **TASK-M07 — Thực thi Batch `BulkWrite` & OCC Validation trong `TransmuterModule`**
  - **Context (`file:line`)**: `centralized-data-service/internal/service/master/transmuter.go:270-465`.
  - **Mô tả**: Phân nhánh khi `binding.EngineType == "mongodb"`:
    - Thực thi `collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))`.
    - Đảm bảo OCC (`_source_ts`): chặn event cũ out-of-order ghi đè bản ghi mới trên Master Mongo.
    - Xử lý Soft-Delete (`_deleted: true`) và xoá phần tử mảng mồ côi (`OrphanKeySuffixes` / Pull-then-Push).
  - **Test / DoD**: Chạy Transmute Realtime & Schedule từ Shadow Table sang Master Mongo thành công, bảo toàn thứ tự OCC.

---

## Nhóm 4: Reconciliation (Đối soát) & Self-Healing (`centralized-data-service`)

- [ ] **TASK-M08 — Mở rộng `GetMasterAgent` trong Recon Engine hỗ trợ MongoDB**
  - **Context (`file:line`)**: `centralized-data-service/internal/service/recon/recon_engine.go:200-215`, `recon_smoke.go:485-495`, `recon_tier_b.go:35-45`.
  - **Mô tả**: Khi `MasterConnectionKey` có `engine_type == "mongodb"`, trả về MongoDB Recon Agent để đếm count và băm XOR/hash đối soát giữa Source/Shadow và Master Mongo Collection.
  - **Test / DoD**: Smoke Recon (Tier A) và Full Recon (Tier B) chạy chính xác trên Master Mongo.

- [ ] **TASK-M09 — Mở rộng Self-Healing (`recon_execute_heal_handler.go`) cho Master Mongo**
  - **Context (`file:line`)**: `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go:466-475`.
  - **Mô tả**: Hỗ trợ heal (re-sync / upsert / soft-delete) đích danh các `_id` bị lệch từ Source/Shadow sang Master MongoDB.
  - **Test / DoD**: Trigger Heal trên CMS đồng bộ lại đúng các document bị drift trên MongoDB đích (`drift == 0` sau khi heal).

---

## Nhóm 5: Kiểm thử E2E & Nghiệm thu (DoD Gates G1–G8)

- [ ] **TASK-M10 — Kiểm thử tích hợp toàn trình (E2E) & Multi-worker Concurrency**
  - **Context (`file:line`)**: `centralized-data-service/test/` & giao diện CMS.
  - **Mô tả**: Kiểm thử luồng khép kín: Tạo kết nối Master Mongo ➔ Tạo Master Binding ➔ Approve Schema ➔ Ingest & Transmute đa worker ➔ Recon & Heal.
  - **Test / DoD**: Đạt đủ 8 cổng chất lượng G1–G8, `count(source) == count(master_mongo)`.
