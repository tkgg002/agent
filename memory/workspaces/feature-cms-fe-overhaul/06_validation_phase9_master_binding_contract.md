# Validation — Phase 9 Master Binding Contract

## Backend

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
```

Kết quả:
- pass

Lưu ý:
- lượt chạy đầu bị sandbox chặn Go build cache
- đã rerun với escalation hợp lệ và pass

## Frontend

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

Kết quả:
- pass

## Contract checks

1. `GET /api/v1/masters` trả master namespace + source/shadow context từ `cdc_system`.
2. `POST /api/v1/masters` nhận `master_schema`, `shadow_schema`, `shadow_table` và resolve bằng metadata V2.
3. `source_shadow` vẫn tồn tại như compatibility fallback.
4. Swagger/comment đã được cập nhật cùng phase.
