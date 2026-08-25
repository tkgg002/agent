# 09_tasks_solution_schema_fix.md — Technical Solutions & Code Snippets

## Giải pháp kỹ thuật chi tiết cho Round 2

### 1. Fix Task 5: HTTP Controller `transmute_schedule_handler.go`
```go
// 1. Thêm field vào DTO
type ScheduleCreateRequest struct {
    MasterSchema string `json:"master_schema"` // Schema của master table (tùy chọn, mặc định rỗng/public)
    MasterTable  string `json:"master_table"`  // Tên table thuần
    Mode         string `json:"mode"`
    CronExpr     string `json:"cron_expr"`
    IsEnabled    bool   `json:"is_enabled"`
    Reason       string `json:"reason"`
}

// 2. Thêm validation và truyền vào Command trong Create()
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

### 2. Fix Task 6: `transmute_scheduler.go`
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

### 3. Fix Task 7: `master_binding_repo.go`
```sql
-- Trong ListMasterTablesByShadowTable & ListMasterTablesByShadowIdentity
SELECT COALESCE(NULLIF(mb.master_schema, ''), 'public') || '.' || mb.master_table AS master_fqn
  FROM cdc_system.master_binding mb
  ...
```

### 4. Fix Task 8: `transmute_schedule_repository_gorm.go`
```sql
-- Trong Save()
SELECT id FROM cdc_system.master_binding
 WHERE master_table = ?
   AND COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')
 LIMIT 1
```
