# Solution Proposal — Schema Design V2 For This Project

## Executive Summary

Giải pháp đúng cho dự án này không phải là "thêm vài cột connection_id vào registry cũ", mà là đổi hẳn mental model từ:

- `source_table -> target_table`

sang:

- `physical connection`
- `logical source object`
- `shadow binding`
- `master binding`
- `runtime state`

Nếu không tách 5 lớp này, code hiện tại sẽ tiếp tục:
- ingest bằng `source_table`
- shadow bằng `target_table`
- master bằng `public.<master>`
- ops bằng `cdc_internal`

và hệ thống sẽ không bao giờ thực sự hỗ trợ multi-destination theo từng bảng.

## Concrete Solution For `centralized-data-service`

### 1. Metadata plane

Thêm schema `cdc_system` và đưa toàn bộ control plane vào đây:

- `connection_registry`
- `source_object_registry`
- `shadow_binding`
- `master_binding`
- `mapping_rule_v2`
- `sync_runtime_state`
- `schema_drift_log_v2`

### 2. Data plane

Payload không còn gắn semantic vào `cdc_internal/public` nữa.

Thay vào đó:
- shadow được route bằng `shadow_binding`
- master được route bằng `master_binding`

Ví dụ:
- `postgres_shadow_a.shadow_billing_public.invoices`
- `postgres_shadow_b.shadow_wallet.transactions`
- `postgres_dw_finance.dw_finance.payment_reports`

### 3. Runtime architecture

#### 3.1. Config layer

`config/config.go`

- giữ `DBConfig` hiện tại cho system DB trong phase đầu
- thêm cấu hình secret resolver / connection manager defaults

#### 3.2. Database layer

`pkgs/database/postgres.go`

- giữ factory mở pool Postgres
- thêm abstraction để được gọi bởi `ConnectionManager`

#### 3.3. New service

`internal/service/connection_manager.go`

API gợi ý:

```go
type ConnectionManager interface {
    GetSystemDB(ctx context.Context) (*gorm.DB, error)
    GetPostgresByConnectionID(ctx context.Context, id int64) (*gorm.DB, error)
    GetShadowDB(ctx context.Context, bindingID int64) (*gorm.DB, error)
    GetMasterDB(ctx context.Context, bindingID int64) (*gorm.DB, error)
}
```

#### 3.4. Registry layer

Thay `RegistryService` bằng `MetadataRegistryService` mới:

- `GetSourceObjectByNormalizedKey`
- `GetActiveShadowBinding`
- `GetActiveMasterBindings`
- `GetMappingRulesByMasterBinding`

### 4. Code modules to change first

#### Wave 1

- `internal/model/*`
- `internal/repository/*`
- `internal/service/registry_service.go`

#### Wave 2

- `internal/handler/event_handler.go`
- `internal/service/schema_adapter.go`
- `internal/sinkworker/*`

#### Wave 3

- `internal/service/master_ddl_generator.go`
- `internal/service/transmuter.go`
- `internal/service/transmute_scheduler.go`

#### Wave 4

- `internal/handler/recon_handler.go`
- `internal/service/recon_heal.go`
- `internal/service/backfill_source_ts.go`
- `internal/service/schema_inspector.go`
- `internal/handler/command_handler.go`

## Why This Solves The User's Actual Complaint

### Complaint 1: `cdc_internal` không thể vừa là system vừa là shadow

Đã xử lý bằng:
- `cdc_system` = control plane
- shadow/master là physical destinations do binding quyết định

### Complaint 2: phải giữ được schema cha / namespace cha

Đã xử lý bằng:
- `source_database`
- `source_schema`
- `source_namespace`
- `shadow_schema`
- `master_schema`

### Complaint 3: mỗi table phải chọn riêng nơi chứa shadow/master

Đã xử lý bằng:
- `shadow_binding`
- `master_binding`

### Complaint 4: hiện tại dự án chỉ có một DB pool

Đã xử lý bằng:
- `connection_registry`
- `connection_manager`

## Recommended Next Implementation Phase

Phase kế tiếp nên là:

1. Tạo migrations `029` -> `035`.
2. Thêm model/repo V2.
3. Viết `ConnectionManager`.
4. Viết compatibility backfill từ V1 sang V2.
5. Chỉ sau đó mới refactor ingest/transmute.
