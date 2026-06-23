# Architectural Decisions

## ADR 1: Decoupling Database Access from Handlers in centralized-data-service

### Status
Accepted

### Context
The handler layer had leakages where GORM queries and raw database operations were executed directly in `ReconHandler`, `BatchBuffer`, `ReconHealer`, and `DLQHandler`. This violated our layer architecture boundaries.

### Decision
* All logs, reports, and sync failures must be routed via their corresponding domain repositories and services (`FailedSyncLogRepo`, `ActivityLogger`).
* Handlers will receive repository and service dependencies through builders or constructors, eliminating direct database interactions (`h.DB`).

### Consequences
* Better maintainability, clear encapsulation of database operations.
* Simpler unit testing using mock repositories and services.

## ADR 2: Dynamic DB Resolution & Repository abstraction (Phase 5)

### Status
Accepted

### Context
In addition to internal tables, the shadow registry and master engines require dynamically resolving and operating on isolated DB configurations without hardcoding GORM connections or executing manual transaction blocks inside registration endpoints.

### Decision
* Introduce `ConnectorResolver` service to dynamically lookup and build connections from registration entries.
* Introduce dedicated repositories (`TransmuteScheduleRepo`, `MasterBindingRepo`, `MappingRuleV2Repo`, `ReconciliationReportRepo`, `TableRegistryRepo`) to cleanly encapsulate SQL queries.
* Encapsulate registration transactional flows into `SourceRegistrationService`.

## ADR 3: Pure DDL Handler & Recon Healer Decoupling (Phase 6)

### Status
Accepted

### Context
`ReconHealer` and `SchemaDDLHandler` still leaked direct DB access via `shadowDB` and raw SQL/DDL operations (`CREATE TABLE`, `ALTER TABLE`, etc.) causing tight coupling and architectural bypasses.

### Decision
* Replace all direct `shadowDB` Exec operations in `SchemaDDLHandler` with high-level adapter commands inside `SchemaAdapter` (e.g. `CreateEmptyTable`, `AddPrimaryKeyColumn`, `AddPrimaryKeyConstraint`, `EnsureCDCColumnsInSchema`).
* Replace raw mapping v2 sync triggers with `DiscoverService.BridgeMappingRulesToV2`.
* Inject `ConnectorResolver` inside `ReconHealer` instead of raw `*gorm.DB` connection.
* Delete the dead code file `schema_ddl_handler_schema.go`.
