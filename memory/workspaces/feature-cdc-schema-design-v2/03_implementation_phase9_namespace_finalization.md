# Implementation — Phase 9 Namespace Finalization

## Code changes

### 1. Runtime chuyển từ `cdc_internal` sang `cdc_system`

- `cmd/sinkworker/main.go`
  - `claim_machine_id` -> `cdc_system.claim_machine_id`
  - `heartbeat_machine_id` -> `cdc_system.heartbeat_machine_id`
- `internal/sinkworker/schema_manager.go`
  - fencing trigger -> `cdc_system.tg_fencing_guard()`
- `internal/service/master_ddl_generator.go`
  - RLS helper -> `cdc_system.enable_master_rls(?)`

### 2. Integration tests đổi sang schema mới

- `internal/handler/kafka_consumer_integration_test.go`
- `internal/handler/recon_handler_integration_test.go`
- `internal/handler/command_handler_activity_integration_test.go`
- `internal/handler/dlq_handler_integration_test.go`
- `internal/service/recon_heal_audit_integration_test.go`

Các test này giờ đọc/ghi:
- `cdc_system.failed_sync_logs`
- `cdc_system.cdc_activity_log`
- `cdc_system.cdc_table_registry`

### 3. Chuẩn hóa shadow naming trong comment/code doc

- comment mô tả shadow path đổi từ `cdc_internal.<table>` sang `shadow_<source_db>.<table>`
- enum lookup comment đổi sang `cdc_system.enum_types`

### 4. Migration finalization

Thêm `038_finalize_cdc_system_namespace.sql`:

- move:
  - `cdc_internal.machine_id_seq` -> `cdc_system.machine_id_seq`
  - `cdc_internal.fencing_token_seq` -> `cdc_system.fencing_token_seq`
- recreate tại `cdc_system`:
  - `claim_machine_id`
  - `heartbeat_machine_id`
  - `tg_fencing_guard`
  - `enable_master_rls`
  - `gen_sonyflake_id`
  - `tg_sonyflake_fallback`
- drop legacy functions ở `cdc_internal`
- `DROP SCHEMA IF EXISTS cdc_internal`

### 5. Grant update

`005_pg_users.sql` được bổ sung:

- `USAGE` trên schema `cdc_system`
- quyền table/sequence/function trong `cdc_system`
- grant trực tiếp cho:
  - `cdc_system.cdc_table_registry`
  - `cdc_system.cdc_mapping_rules`
  - `cdc_system.pending_fields`
  - `cdc_system.schema_changes_log`

## Kết quả implementation

- runtime chính không còn hard dependency vào `cdc_internal`
- system table path trong code đã quy về `cdc_system`
- shadow naming đã chốt theo `shadow_<source_db>`
