# Validation — Phase 14 Registry Surface Pruning

## Lệnh kiểm chứng

```bash
rg -n "/sync/reconciliation|/registry/:id/status|/registry/scan-source|/registry/:id/sync|/registry/:id/jobs|/registry/:id/discover|/registry/:id/drop-gin-index" \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/registry_handler.go

cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Kết quả

- grep route dead: no match
- `go test ./...`: pass
- `npm run build`: pass
