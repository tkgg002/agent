# Validation — Phase 8 Worker Schedule Contract

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

## Kiểm tra logic

1. `worker-schedule` API đã có swagger/comment cho list/create/update.
2. `GET /api/worker-schedule` trả enriched scope từ metadata V2 khi resolve được.
3. `POST /api/worker-schedule` vẫn tương thích `target_table`, đồng thời nhận thêm metadata source/shadow để resolve chính xác hơn.
4. `ActivityManager` dùng scope từ API cho list view thay vì tự suy diễn hoàn toàn từ `/api/registry`.
