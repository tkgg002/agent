# Phase 23 Validation - Dashboard + ActivityManager V2 Reads

## Kiểm thử dự kiến

1. `go test ./...` trong `cdc-cms-service`
2. `npm run build` trong `cdc-cms-web`
3. grep call-site FE để xác nhận:
   - `Dashboard` không còn dùng `/api/registry/stats`
   - `ActivityManager` không còn dùng `/api/registry`

## Swagger

- annotations trong source code phải được cập nhật cùng phase
- generated docs có thể chưa regen được nếu môi trường thiếu `swag`

## Kết quả thực tế

- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...`
  - pass
  - lần đầu fail vì sandbox chặn Go build cache
  - đã rerun ngoài sandbox với escalation hợp lệ và pass
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build`
  - pass
- grep call-site:
  - `Dashboard.tsx` không còn `/api/registry/stats`
  - `ActivityManager.tsx` không còn `cmsApi.get('/api/registry'...)`
- `cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && make swagger`
  - fail: `swag: No such file or directory`
  - kết luận: annotations đã update, generated docs chưa thể regen trên máy hiện tại
