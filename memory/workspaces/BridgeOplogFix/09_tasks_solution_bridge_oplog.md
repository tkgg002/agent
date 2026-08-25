# 09 Technical Solution: Bridge Oplog Alignment

## Technical Solution Overview

1. **System DB Binding**:
   - Worker server init (`server_setup.go`) passes system `db` to `NewBridgeHandler`.
   - `governance.NewActivityLogger(h.db, h.logger)` operates directly on `cdc_system.cdc_activity_log` table.

2. **Config Primary Key Enforcement**:
   - `resolveCollection` reads `tc.PrimaryKeyField` from `registrySvc.ResolveSourceRoutes`.
   - Maps to `TargetColumn` if dynamic rule exists, otherwise preserves exact `tc.PrimaryKeyField`.

3. **Metrics Tracking**:
   - Tracks `oplog_fetched` (Mongo change stream / collection events read).
   - Tracks `shadow_written` (upserted rows in PostgreSQL shadow table).
   - Records both metrics in `details` JSON payload of `cdc_system.cdc_activity_log`.
