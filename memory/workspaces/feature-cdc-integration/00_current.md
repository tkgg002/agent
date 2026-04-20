# CDC Integration — Current Status (2026-04-13)

## Tổng quan
- **Project**: Hybrid CDC (Debezium real-time + Airbyte batch) cho GooPay (~60 microservices)
- **Tiến độ**: ~75% Phase 1
- **Kiến trúc**: 4 services, PostgreSQL Single Source of Truth, JSONB Landing Zone

---

## 4 Services

| # | Service | Project | Port | Status | Build |
|---|---------|---------|------|--------|-------|
| 1 | Auth Service | `cdc-auth-service` | :8081 | ✅ Done | OK |
| 2 | CDC Worker | `centralized-data-service` | :8082 | ✅ Done | OK |
| 3 | CMS API | `cdc-cms-service` | :8080 | ✅ Done | OK |
| 4 | CMS FE | `cdc-cms-web` | :5173 | ⏳ In Progress | OK |

---

## Đã hoàn thành

- **Database Migration**: Schema cho `pending_fields`, `mapping_rules`, `schema_change_logs`, `cdc_table_registry`
- **CDC Worker**: NATS consumer, schema inspector, batch buffer, Prometheus metrics (6 metrics), dynamic_mapper stub
- **CMS API**: 13 REST endpoints, JWT auth, Airbyte OAuth2 client, approval workflow (Approve/Reject → ALTER TABLE + Mapping Rule + NATS Publish)
- **Auth Service**: Login, Register, JWT access+refresh, RBAC (admin/operator)
- **Airbyte Client**: Discover Schema, Trigger Sync, Update Connection
- **Schema Inspector**: HandleIntrospect logic, type inference
- **Context Propagation**: NATS reload payload includes user_id + metadata
- **Docker**: Dockerfiles cho cả 4 services, docker-compose

---

## Backend Status (Verified 2026-04-13)

### Track E: Bridge + Transform ✅ ALL DONE
- ~~P0 Reload Subscriber~~: ✅ `worker_server.go:94-115` + `registry_service.go`
- ~~E0 Bridge~~: ✅ Verified 1M rows `payment-bills` (2026-04-08)
- ~~E1 Batch Transform~~: ✅ `HandleBatchTransform` (SQL-based)
- ~~E2 Periodic Scheduler~~: ✅ Ticker goroutine (bridge+transform mỗi 5m, field scan mỗi 1h)
- ~~E3 Transform Status~~: ✅ `GET /registry/:id/transform-status`

### Track A: Airbyte APIs ✅ ALL DONE
- ~~A1~~: ✅ `GET /airbyte/destinations`
- ~~A2~~: ✅ `GET /airbyte/connections` (enriched)
- ~~A3~~: ✅ `GET /airbyte/connections/:id/streams`

### Track B: Stream Sync ✅ ALL DONE
- ~~B1~~: ✅ `POST /registry/sync-from-airbyte`
- ~~B2~~: ✅ Migration `004_bridge_columns.sql` + model columns (sync_mode, cursor_field, namespace)
- ~~B3~~: ✅ `reconciliation_service.go` auto-heal + `UpdateConnection` toggle

### Track C: Field Mapping ✅ ALL DONE
- ~~C1~~: ✅ Auto-detect fields (ExecuteImport)
- ~~C2~~: ✅ Periodic field scan (HandlePeriodicScan + scanTicker)
- ~~C3~~: ✅ Batch approve/reject (`PATCH /mapping-rules/batch`)

### Track D: Monitoring ✅ ALL DONE
- ~~D1~~: ✅ `GET /sync/health`
- ~~D2~~: ✅ `GET /sync/reconciliation` — per-table report (row count + field coverage + transform %)

### v1.12 Additions (2026-04-13)
- ✅ Sonyflake package (`pkgs/idgen/sonyflake.go`) + init trong main.go
- ✅ Migration `003_sonyflake_schema.sql` — Dual-PK (BIGINT seq + source_id)
- ✅ Bridge auto-detect v1.12 schema (backward compatible)
- ✅ Dependencies: sonyflake + gjson added
- ✅ Gjson integrated: `bridge_batch.go` dùng gjson parse _raw_data
- ✅ pgx.Batch bridge: `bridge_batch.go` + `pgx_pool.go` — high-throughput pipeline
- ✅ Sonyflake benchmark: 25K IDs/sec, 1M unique IDs verified
- ✅ Partitioning: `004_partitioning.sql` + `ensure_cdc_partition()` + Worker daily maintenance
- ✅ CMS API: `?mode=batch` on Bridge endpoint
- ✅ FE: Bridge (SQL) + Batch (pgx) buttons

## Còn thiếu

### Frontend (CMS Web) — CHÍNH LÀ BOTTLENECK
- **Transform progress bar** trong TableRegistry.tsx
- **Batch approve/reject UI** trong MappingFieldsPage.tsx
- **Sync health widget** trong Dashboard.tsx
- **Destinations display** trong SourceConnectors.tsx

### Infrastructure
- NATS Permissions/ACL chưa configure
- PostgreSQL user separation
- Sonyflake Lab benchmark (1M rows)

### Testing
- Unit tests: CDC-M2.7, M3.7
- Integration tests: E2E flow

---

## Plan v1.11 (2026-04-08) — Bridge + Worker Transform

### Phát hiện quan trọng: 2 hệ thống table tách biệt

| System | Table | Data |
|--------|-------|------|
| Airbyte | `_airbyte_raw_merchants` | ✅ Có data |
| CDC Worker | `cdc_merchants` | ❌ Trống |

**Solution**: Bridge Pattern — Worker đọc `_airbyte_raw_*` → populate CDC `_raw_data` → transform sang typed columns

### 5 Tracks (ước tính 3 tuần)
- **Track A**: Airbyte Read APIs (2 ngày)
- **Track B**: Stream Sync — auto-detect PK, registry, bidirectional toggle (3 ngày)
- **Track C**: Field Mapping Sync — auto-detect, periodic scan, batch approve (3 ngày)
- **Track D**: Monitoring & Reconciliation (2 ngày)
- **Track E (CRITICAL)**: Airbyte Bridge + Worker Transform + Scheduler (6 ngày)

---

## Decisions chính (ADRs)

| ADR | Quyết định | Status |
|-----|-----------|--------|
| ADR-001 | Hybrid Event Bridge (Triggers cho critical, Polling cho non-critical) | Proposed |
| ADR-005 | Worker Pool 10 goroutines, batch 500, target 5K events/sec/pod | Proposed |
| ADR-008 | JSONB Landing Zone cho zero data loss | Approved |
| ADR-009 | Dynamic Mapping Engine — rule-based, Redis cache, hot reload via NATS | Approved |
| ADR-010 | CMS Approval Workflow (detect → pending → approve → ALTER TABLE → reload) | Approved |
| ADR-011 | Schema Drift Detection <1 phút | Approved |
| ADR-012 | Target-Table Based Indexing cho mapping cache | Approved |

---

## Timeline

| Ngày | Milestone |
|------|-----------|
| 2026-03-16 | Workspace init, v2.0 features |
| 2026-03-19 | Split Phase 1 (static) + Phase 2 (dynamic) |
| 2026-03-25 | Chuyển sang config-driven (~200 tables) |
| 2026-03-30 | Implementation start — migrations, worker, CMS, Auth |
| 2026-03-31 | 4 services build OK, governance fix |
| 2026-04-06 | Status report: 75% Phase 1, phát hiện reload gap |
| 2026-04-08 | Plan v1.11: Bridge + Transform roadmap |
| 2026-04-13 | **Hôm nay** — Tổng hợp current status |

---

## Next Steps (ưu tiên)

1. Fix reload subscriber trong Worker (`schema.config.reload`)
2. Implement Airbyte → CDC Bridge (Track E0)
3. Hoàn thiện CMS Frontend
4. Worker Transform + Periodic Scheduler (E1-E2)
5. Unit + Integration tests


-----
Hiểu ý bạn rồi, mình sẽ loại bỏ hoàn toàn phần liên quan đến **Snowflake (Data Warehouse)** để tránh phát sinh chi phí "đốt" Credit không cần thiết. Chúng ta sẽ tập trung vào việc tối ưu hóa nội tại trên **PostgreSQL** và **Go Worker** để gánh 500GB dữ liệu đó một cách "kinh tế" nhất.

Dưới đây là bản tóm tắt kế hoạch đã được tinh chỉnh, tập trung vào hiệu năng thuần túy và giải pháp mã nguồn mở:

---

## 🛠️ CHIẾN LƯỢC TỐI ƯU HÓA CDC (QUY MÔ 500GB)

### 1. Vấn đề ID: Tại sao chọn Sonyflake?
Thay vì dùng các ID dạng chuỗi (NanoID) gây nát Index, bạn dùng **Sonyflake (64-bit BigInt)**:
* **Machine ID (16 bits):** Rất quan trọng cho 60 microservices. Bạn có thể dùng 16 bit cuối của IP để làm Machine ID mà không cần server quản lý tập trung.
* **Tăng dần (K-ordered):** Giúp Postgres chèn dữ liệu vào cuối B-Tree. Ở quy mô 100M dòng, điều này giúp giảm tình trạng **Index Fragmentation** (phân mảnh chỉ mục) và giữ tốc độ Insert ổn định.

### 2. Xử lý dung lượng (Hóa giải 500GB)
Vì chúng ta bỏ qua Snowflake, Postgres phải tự gánh vác. Bạn cần thực hiện **"Phẳng hóa dữ liệu" (Data Flattening)**:
* **Bỏ JSONB ở tầng cuối:** Go Worker sẽ bóc các field từ `_raw_data` và ghi vào các cột định danh (`int`, `timestamp`, `varchar`). 
* **Tỉ lệ nén:** Cột định danh trong Postgres được nén tốt hơn JSONB rất nhiều. Việc chuyển từ JSONB sang Typed Columns có thể giúp bạn tiết kiệm **~150GB - 200GB** trên tổng 500GB dự kiến.

### 3. Giải pháp "Partitioning" (Thay thế OLAP đắt đỏ)
Để tránh việc một câu truy vấn báo cáo làm treo cả DB giao dịch:
* **Table Partitioning:** Chia bảng `cdc_merchants` theo tháng (ví dụ: `merchants_2026_04`). 
* **Lợi ích:** Khi cần xóa dữ liệu cũ hoặc bảo trì, bạn chỉ cần tác động lên Partition đó thay vì quét toàn bộ 500GB.

---

## 📝 FILE PLAN CẬP NHẬT (PHASE 1.12)

| Hạng mục | Giải pháp | Trạng thái |
| :--- | :--- | :--- |
| **ID Generator** | **Sonyflake** (bản Go) | 🚀 Sẵn sàng tích hợp |
| **JSON Parser** | **Gjson** (truy xuất cực nhanh) | 🚀 Thay thế encoding/json |
| **Storage Strategy** | **Postgres Partitioning** | 🛠️ Cần triển khai |
| **Data Flow** | **Airbyte Raw -> Go Worker -> Typed Table** | 🛠️ Track E0-E2 |

### 🚀 Hành động tiếp theo (Next Steps):
1.  **Fix P0:** Implement NATS Subscriber để Worker tự động reload mapping rule khi bạn nhấn "Approve" trên CMS mà không cần restart Pod.
2.  **Track E0 (Bridge):** Viết logic Worker đọc theo lô (Batch) từ bảng raw của Airbyte, gán ID Sonyflake và đẩy sang bảng CDC chính thức.
3.  **Tối ưu ghi:** Dùng `pgx.Batch` hoặc `CopyFrom` trong Go để đảm bảo nuốt trôi 5.000 events/giây.

**Kết luận:** Bằng cách dùng **Sonyflake** và **Go**, bạn có thể xử lý 500GB ngay trên Postgres mà không tốn một xu Credit nào cho các nền tảng bên ngoài. 

Bạn có cần mình hỗ trợ viết đoạn code Go để tích hợp **Sonyflake** vào Worker pool hiện tại không?
--- 
update từ user vào curent step

Chào bạn, dựa trên những gì chúng ta vừa phân tích về bài toán **x100 data (500GB)**, **Sonyflake ID**, và việc loại bỏ Snowflake để tối ưu chi phí, mình đã cập nhật lại bản kế hoạch (Action Plan) này.

Bản kế hoạch này bám sát cấu trúc 4 services và các Track (A-E) bạn đã đề ra, nhưng "bơm" thêm các yếu tố kỹ thuật để chịu tải thực tế cho GooPay.

---

# 🚀 CDC STRATEGIC PLAN — PHASE 1.12 (REVISED)
**Target**: Handle 100M+ Records | **Tech**: Go, NATS, Postgres | **ID**: Sonyflake

## 1. Ưu tiên cao nhất (Fix P0 & Gap)
* **Reload Subscriber (Hot-reload):**
    * Triển khai ngay NATS listener cho topic `schema.config.reload`.
    * Worker khi nhận signal phải update **In-memory Mapping Cache** (sử dụng `sync.Map` hoặc RWMutex) để tránh restart Pod gây gián đoạn luồng stream 5K events/sec.
* **Sonyflake Integration:**
    * Thay thế các ID tự tăng (Serial) bằng **Sonyflake ID** cho các bảng CDC chính.
    * Cấu hình `MachineID` dựa trên 16-bit cuối của Pod IP trong K8s để đảm bảo tính duy nhất trên 60 microservices.

## 2. Track E: Bridge & High-Performance Transform
Ở quy mô 500GB, Track E là "xương sống" để dọn dẹp dung lượng:
* **E0 (Airbyte Bridge):**
    * Không dùng `SELECT *`. Dùng cơ chế **Keyset Pagination** (dựa trên `_airbyte_emitted_at`) để đẩy data từ bảng raw sang bảng CDC.
    * Worker dùng **Gjson** để bóc tách thần tốc dữ liệu từ `_raw_data`.
* **E1-E2 (Worker Transform):**
    * **Flattening:** Chuyển dữ liệu từ JSONB sang **Typed Columns** (Native Postgres types).
    * **Batching:** Bắt buộc dùng `pgx.Batch` hoặc `CopyFrom`. Gom 500-1000 records/batch trước khi flush xuống DB.

## 3. Quản trị dung lượng & Database (No-Snowflake)
Vì bỏ qua Cloud Data Warehouse, Postgres phải "tự lực cánh sinh":
* **PostgreSQL Partitioning (ADR-013 - Proposed):**
    * Áp dụng **Declarative Partitioning** theo tháng cho các bảng `cdc_merchants`, `cdc_transactions`.
    * Giúp Index gọn nhẹ hơn, truy vấn báo cáo không cần scan toàn bộ 500GB.
* **Index Optimization:** * Chỉ Index những field thực sự cần truy vấn (bóc ra từ JSONB).
    * Xóa bỏ các GIN Index cồng kềnh trên cột `_raw_data` sau khi đã transform thành công.

## 4. Timeline cập nhật (Short-term)

| Ngày | Task | Mục tiêu |
| :--- | :--- | :--- |
| **14/04** | **P0 Fix** | Worker nhận được reload signal từ CMS. |
| **15-16/04**| **Sonyflake Lab** | Thử nghiệm gán ID Sonyflake cho 1M dòng demo, đo đạc Index size. |
| **17-20/04**| **Track E0 Logic** | Hoàn thành Bridge bóc dữ liệu từ Airbyte Raw sang CDC Landing Zone. |
| **21/04+** | **Batch Upsert** | Test performance đạt ngưỡng 5K events/sec. |

---

### 💡 Lưu ý quan trọng cho "Senior Data Architect":
Việc bạn phát hiện ra "2 hệ thống table tách biệt" (Airbyte vs CDC) là một điểm chạm cực kỳ tốt. Thay vì coi đó là lỗi, hãy coi bảng của Airbyte là **Staging Area**. Worker của bạn đóng vai trò là **ELT Processor** thực hiện bước cuối cùng (Transform) để tối ưu hóa storage.

**Bạn thấy bản Plan cập nhật này đã sát với thực tế "x100 data" của bạn chưa?** Nếu rồi, mình có thể giúp bạn draft nhanh đoạn code Go cho `Reload Subscriber` hoặc `Sonyflake ID generator`.