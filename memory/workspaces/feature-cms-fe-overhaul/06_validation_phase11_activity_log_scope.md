# Validation — Phase 11 Activity Log Scope

## Backend

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
```

Kết quả:
- pass

## Frontend

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

Kết quả:
- pass

## Contract checks

1. `GET /api/activity-log` trả source/shadow context V2 khi resolve được.
2. `GET /api/activity-log` support filter:
   - `source_database`
   - `source_table`
   - `shadow_schema`
   - `shadow_table`
   - `target_table` fallback
3. `GET /api/activity-log/stats` enrich `recent_errors` với scope V2.
4. `useAsyncDispatch` support `statusParams` nhưng vẫn giữ `targetTable` fallback.
