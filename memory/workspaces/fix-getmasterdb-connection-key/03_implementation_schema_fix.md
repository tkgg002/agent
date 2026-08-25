# 03_implementation_schema_fix.md — Detailed Technical Design

## 1. Kiến trúc định danh Schema-Qualified (FQN)

```
HTTP API (CMS) ───────────► Command Bus ─────────────► NATS Bus (cdc.cmd.transmute)
  ScheduleCreateRequest        CreateTransmuteCmd          {"master_table": "schema.table"}
  (MasterSchema,               (MasterSchema,                                 │
   MasterTable)                 MasterTable)                                  ▼
                                                                     TransmuteHandler (CDS)
                                                                              │
                                                                              ▼
                                                                     TransmuterModule.Run()
                                                                              │
                                                                              ▼
                                                                     loadMaster("schema.table")
                                                                     WHERE master_table = ?
                                                                       AND master_schema = ?
```

## 2. Chi tiết sửa đổi kỹ thuật

### A. `cdc-cms-service/internal/api/scheduler/transmute_schedule_handler.go`
- **DTO**:
  ```go
  type ScheduleCreateRequest struct {
      MasterSchema string `json:"master_schema"`
      MasterTable  string `json:"master_table"`
      Mode         string `json:"mode"`
      CronExpr     string `json:"cron_expr"`
      IsEnabled    bool   `json:"is_enabled"`
      Reason       string `json:"reason"`
  }
  ```
- **Validation & Dispatch**:
  ```go
  if req.MasterSchema != "" && !schedNameRe.MatchString(req.MasterSchema) {
      return c.Status(400).JSON(fiber.Map{"error": "invalid_master_schema"})
  }
  cmd := schedulerCmd.CreateTransmuteScheduleCommand{
      MasterSchema: req.MasterSchema,
      MasterTable:  req.MasterTable,
      Mode:         req.Mode,
      CronExpr:     req.CronExpr,
      NextRunAt:    nextRunAt,
      IsEnabled:    req.IsEnabled,
      CreatedBy:    actor,
  }
  ```

### B. `centralized-data-service/internal/service/master/transmute_scheduler.go`
- **SQL Concat**:
  ```sql
  SELECT ts.id,
         COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn,
         ts.cron_expr
    FROM cdc_system.transmute_schedule ts
    JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
   WHERE ts.is_enabled = true
     AND ts.mode = 'cron'
     AND mb.is_active = true
     AND mb.schema_status = 'approved'
     AND (ts.next_run_at IS NULL OR ts.next_run_at <= NOW())
   FOR UPDATE SKIP LOCKED
   LIMIT 10
  ```

### C. `centralized-data-service/internal/repository/master/master_binding_repo.go`
- **SQL Concat trong ListMasterTablesByShadowTable & ListMasterTablesByShadowIdentity**:
  ```sql
  SELECT COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn
  ```

### D. `cdc-cms-service/internal/infra/persistence/scheduler/transmute_schedule_repository_gorm.go`
- **Save() Query**:
  ```sql
  SELECT id FROM cdc_system.master_binding
   WHERE master_table = ?
     AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')
   LIMIT 1
  ```
