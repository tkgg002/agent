# Validation — Phase 10 Mapping Rules V2

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

1. `GET /api/mapping-rules` đọc từ `cdc_system.mapping_rule_v2`.
2. `POST /api/mapping-rules` nhận source/shadow context V2, vẫn giữ fallback `source_table`.
3. `POST /api/mapping-rules/reload` resolve target reload theo `shadow_table`.
4. `POST /api/mapping-rules/{id}/backfill` dispatch theo shadow target của rule.
5. Swagger/comment đã được cập nhật cùng phase.
