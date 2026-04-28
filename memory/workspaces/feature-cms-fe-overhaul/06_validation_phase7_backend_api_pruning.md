# Validation — Phase 7 Backend API Pruning

## Commands

```bash
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Result

- `go test ./...`: pass
- `npm run build`: pass

## Extra Checks

Search xác nhận runtime surface đã thay đổi:

- router không còn mount `v1/tables`
- router không còn mount `registry/:id/bridge`
- swagger/comment không còn `@Router /api/registry/{id}/refresh-catalog [post]`

## Sandbox Note

- `go test ./...` ban đầu bị sandbox chặn Go build cache.
- Đã rerun hợp lệ với escalation và pass.
