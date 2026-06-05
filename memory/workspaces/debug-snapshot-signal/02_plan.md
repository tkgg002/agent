# Debug Snapshot Signal Plan

1. **Verify topic data**: Confirm `cdc.goopay.centralized-export-service.export-jobs` actually has messages (resolve anomaly first=0/last=0).
2. **Verify 2-worker hypothesis**: Re-check consumer group members + committed offsets to see if another worker is consuming.
3. **Verify local shadow DB**: Confirm the shadow table is empty.
4. **Verify local worker process**: Inspect worker actually running + check if it processed any message.
5. **Root-cause identify**: Find out who consumed the 152 messages.
6. **Apply fix**: Fix the root cause without cheating the config.
7. **End-to-end verify**: Ensure shadow table has rows after the fix.
8. **Document**: `report_*.md`, append `05_progress.md`, append `lessons.md`.
