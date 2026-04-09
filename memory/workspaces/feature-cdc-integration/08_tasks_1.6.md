# Phase 1.5 & 1.6 - Mapping & Airbyte Orchestration Tasks

> **Project**: CDC Integration
> **Phase**: 1.5 (Mapping) & 1.6 (Airbyte Orchestration)
> **Status**: Completed (2026-03-31)

---

## Epic: CDC-PHASE-1.5 - Mapping & Introspection
### CDC-M10: Mapping Visualization & Introspection ✅
- **Status**: Done (2026-03-31)
- **Description**: UI xem mapping, tự động quét field chưa map từ JSONB, nút Backfill.
- **Key Features**: 
  - Dynamic schema discovery from JSONB data.
  - Mapping rule visualization in CMS.
  - Manual backfill trigger for specific tables.

## Epic: CDC-PHASE-1.6 - Airbyte Orchestration
### CDC-M11: Airbyte Orchestration ✅
- **Status**: Done (2026-03-31)
- **Description**: Auto-register table, auto-update catalog, sync status badge.
- **Key Features**:
  - `BulkRegister` for automated table onboarding.
  - `AutoDiscovery` of new source tables.
  - Sync status monitoring via Airbyte Jobs API.
  - Integrated health badges in the CMS UI.

## Epic: CDC-PHASE-1.6 - Legacy Support & Field Discovery
### CDC-M12: Legacy Table Discovery & Standardization ✅
- **Status**: In Progress (2026-03-31)
- **Description**: Giải quyết vấn đề các bảng có sẵn thiếu metadata (`_raw_data`) và chưa hiện Field Mappings trong CMS.
- **Sub-tasks**:
  - [ ] **Backend**: Viết API `/api/registry/:id/standardize` để tự động chạy `ALTER TABLE` thêm các cột metadata thiếu (`_raw_data`, `_source`, v.v.).
  - [ ] **Backend**: Viết API `/api/registry/:id/discover` để quét `information_schema.columns` và tự động insert vào `cdc_mapping_rules`.
  - [ ] **PostgreSQL**: Cập nhật function `create_cdc_table` để có thể chạy an toàn trên bảng đã có sẵn (Idempotent).
  - [ ] **Frontend**: Thêm nút "Standardize & Discover" trong Table Registry UI.
  - [ ] **Frontend**: Hiển thị danh sách Field Mapping hiện tại của bảng (đã map vs chưa map) dựa trên database thực tế.
