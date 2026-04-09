# Service Boundary Analysis — CDC Integration Phase 1.6

> **Ngày phân tích**: 2026-03-31  
> **Mục tiêu**: Làm rõ ranh giới trách nhiệm giữa `cdc-cms-service` (API) và `centralized-data-service` (CDC Worker).

---

## Nguyên tắc cốt lõi

> **API chỉ đọc/ghi bảng config** (`cdc_table_registry`, `cdc_mapping_rules`, `cdc_schema_changes`).  
> Bất kỳ thao tác nào chạm vào **DW tables** hoặc `information_schema` của DW đều thuộc về **CDC Worker**.

---

## Hiện trạng vi phạm ranh giới

### `cdc-cms-service` (API) — Chức năng hiện tại

| Handler / Service | Endpoint | Tính chất | Đúng chỗ? |
|---|---|---|---|
| `RegistryHandler` List/Register/Update | `GET/POST/PATCH /registry` | Config CRUD | ✅ |
| `RegistryHandler.GetStatus` | `GET /registry/:id/status` | Đọc Airbyte status | ✅ |
| `RegistryHandler.GetStats` | `GET /registry/stats` | Thống kê config | ✅ |
| `MappingRuleHandler` List/Create | `GET/POST /mapping-rules` | Config CRUD | ✅ |
| `SchemaChangeHandler` | `/schema-changes/*` | Approval workflow | ✅ |
| `AirbyteHandler.ListSources` | `GET /airbyte/sources` | Đọc Airbyte metadata | ✅ |
| `MappingRuleHandler.Reload` | `POST /mapping-rules/reload` | Publish NATS trigger | ✅ |
| **`MappingRuleHandler.Backfill`** | `POST /mapping-rules/:id/backfill` | **UPDATE trực tiếp DW table** | ❌ |
| **`IntrospectionHandler.Scan`** | `GET /introspection/scan/:table` | **SELECT từ `_raw_data` DW** | ❌ |
| **`LegacyService.Standardize`** | `POST /registry/:id/standardize` | **Gọi `standardize_cdc_table()` trên DW** | ❌ |
| **`LegacyService.DiscoverMappings`** | `POST /registry/:id/discover` | **Quét `information_schema` DW** | ❌ |

### `centralized-data-service` (CDC Worker) — Chức năng hiện tại

| Component | Chức năng | Đúng chỗ? |
|---|---|---|
| `EventHandler` | Consume NATS, parse event, upsert DW | ✅ |
| `EventHandler.handleDelete` | Soft-delete record DW | ✅ |
| `SchemaInspector.InspectEvent` | Detect schema drift, lưu `cdc_pending_fields`, publish NATS | ✅ |
| `RegistryService` (cache) | Load config từ DB vào RAM | ✅ |
| `DynamicMapper` | Stub Phase 2 | ✅ |
| ❌ Thiếu: Table Standardization | — | ❌ Thiếu |
| ❌ Thiếu: Schema Discovery | — | ❌ Thiếu |
| ❌ Thiếu: Backfill logic | — | ❌ Thiếu |

---

## Nguyên nhân lỗi `standardize_cdc_table does not exist`

API `cdc-cms-service` kết nối tới **DB của chính nó** (bảng config), không phải **DW DB** của `centralized-data-service`.  
Function `standardize_cdc_table` chỉ tồn tại trong **DW DB** → nên gọi từ API sẽ luôn thất bại.

---

## Giải pháp đề xuất: NATS Command Pattern

Thay vì API trực tiếp thực thi trên DW, API publish lệnh vào NATS. CDC Worker lắng nghe và thực thi trên đúng DB của nó.

| NATS Subject | Payload | Worker Action |
|---|---|---|
| `cdc.cmd.backfill` | `{table, field, column, type}` | UPDATE DW table |
| `cdc.cmd.standardize` | `{registry_id, target_table}` | Gọi `standardize_cdc_table()` |
| `cdc.cmd.discover` | `{registry_id, target_table, source_table}` | Scan `information_schema`, insert mapping rules |

**API flow**: publish NATS → trả về `202 Accepted` (async).  
**Worker flow**: lắng nghe → thực thi → cập nhật status vào DB config.

---

## Mức độ ưu tiên refactor

| Ưu tiên | Chức năng | Lý do |
|---------|-----------|-------|
| 🔴 Critical | Standardize & Discover | Gây lỗi 500 vì sai DB |
| 🟠 High | Backfill | Mutate DW data trực tiếp từ API |
| 🟡 Medium | Introspection Scan | Read-only nhưng vi phạm boundary |
| 🟢 Low | Airbyte orchestration trong Register | Coupling nhưng chưa ảnh hưởng ngay |

---

## Ranh giới đúng (Target State)

```
cdc-cms-service (API)                    centralized-data-service (Worker)
─────────────────────                    ─────────────────────────────────
✅ CRUD: registry, mapping rules         ✅ NATS consumer + event upsert
✅ Schema approval workflow              ✅ Schema drift detection → NATS alert
✅ Airbyte metadata read                 ✅ NATS listener: schema.config.reload
✅ Publish NATS commands                 ➕ NATS listener: cdc.cmd.standardize
                                         ➕ NATS listener: cdc.cmd.discover
                                         ➕ NATS listener: cdc.cmd.backfill
```
