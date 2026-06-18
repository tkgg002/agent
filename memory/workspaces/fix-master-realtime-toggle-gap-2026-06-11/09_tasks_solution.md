# 09_tasks_solution — Code solution

## File: cdc-cms-service/internal/api/transmute_schedule_handler.go — method `Toggle`

### Trước (chỉ flip cờ, không catch-up)
```go
user := middleware.GetUsername(c)
cmd := commands.ToggleTransmuteScheduleCommand{ID: id, IsEnabled: req.IsEnabled, UpdatedBy: user}
ctx := messaging.WithMetadata(...)
if _, err := h.bus.Execute(ctx, cmd); err != nil { ... }
return c.JSON(fiber.Map{"status": "toggled", "id": id, "is_enabled": req.IsEnabled})
```

### Sau (thêm prev-state lookup + catch-up off→on)
```go
user := middleware.GetUsername(c)

// GAP-FIX realtime toggle: đọc trạng thái + master_table TRƯỚC khi flip để phát
// hiện transition off→on. Tắt realtime → record vào shadow trong "cửa sổ off"
// không transmute; bật lại chỉ forward ⇒ record đó thiếu VĨNH VIỄN. Catch-up
// bên dưới đóng gap ngay tại re-enable (tự động hoá "đồng bộ thủ công").
var prev ScheduleRow
_ = h.db.WithContext(c.Context()).Raw(`
    SELECT ts.id, mb.master_table AS master_table, ts.is_enabled
      FROM cdc_system.transmute_schedule ts
      LEFT JOIN cdc_system.master_binding mb ON mb.id = ts.master_binding_id
     WHERE ts.id = ? LIMIT 1`, id).Scan(&prev)

cmd := commands.ToggleTransmuteScheduleCommand{ID: id, IsEnabled: req.IsEnabled, UpdatedBy: user}
ctx := messaging.WithMetadata(c.UserContext(), user, c.Get("X-Correlation-Id"), c.Get("Idempotency-Key"))
if _, err := h.bus.Execute(ctx, cmd); err != nil {
    switch {
    case errors.Is(err, commands.ErrTransmuteScheduleNotFound):
        return c.Status(404).JSON(fiber.Map{"error": "not_found"})
    default:
        return c.Status(500).JSON(fiber.Map{"error": "internal_error"})
    }
}

// Catch-up khi BẬT LẠI realtime (off→on): dispatch full Shadow→Master transmute
// (idempotent OCC upsert — chỉ insert record thiếu, không ghi đè data mới hơn).
// Reuse y hệt RunNow, KHÔNG cơ chế mới. Worker tự gate (active+approved+rule).
catchupDispatched := false
if req.IsEnabled && !prev.IsEnabled && strings.TrimSpace(prev.MasterTable) != "" {
    schedID, _ := c.ParamsInt("id")
    if schedID > 0 {
        _ = h.db.WithContext(c.Context()).Exec(
            `UPDATE cdc_system.transmute_schedule SET last_status='running', last_run_at=NOW(), updated_at=NOW() WHERE id=?`, schedID)
    }
    runCmd := commands.TransmuteRunCommand{
        MasterTable:   prev.MasterTable,
        TriggeredBy:   user,
        CorrelationID: "realtime-reenable-" + id + "-" + time.Now().UTC().Format(time.RFC3339Nano),
        ScheduleID:    int64(schedID),
    }
    if _, derr := h.bus.Dispatch(ctx, runCmd); derr != nil {
        h.logger.Warn("realtime re-enable catch-up dispatch failed",
            zap.String("schedule_id", id), zap.String("master_table", prev.MasterTable), zap.Error(derr))
        if schedID > 0 { // revert: tránh kẹt last_status='running' khi dispatch fail (G4)
            _ = h.db.WithContext(c.Context()).Exec(
                `UPDATE cdc_system.transmute_schedule SET last_status='failed', last_error=?, updated_at=NOW() WHERE id=?`,
                "catch-up dispatch failed: "+derr.Error(), schedID)
        }
    } else {
        catchupDispatched = true
        h.logger.Info("realtime re-enabled → catch-up transmute dispatched",
            zap.String("schedule_id", id), zap.String("master_table", prev.MasterTable))
    }
}

return c.JSON(fiber.Map{"status": "toggled", "id": id, "is_enabled": req.IsEnabled, "catchup_dispatched": catchupDispatched})
```

### Ghi chú
- Imports đã có: `time` (RunNow dùng), `zap` (handler.logger), `strings`, `commands`, `messaging`. Không thêm import mới.
- Reuse: `ScheduleRow` (=queries.TransmuteScheduleRow, có ID/MasterTable/IsEnabled), `commands.TransmuteRunCommand` (Async), `h.bus.Dispatch` — đúng pattern RunNow.
- KHÔNG đụng worker/transmuter/DB schema. 1 file, ~25 dòng.
