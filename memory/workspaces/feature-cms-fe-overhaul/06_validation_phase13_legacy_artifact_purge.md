# Validation — Phase 13 Legacy Artifact Purge

## Lệnh kiểm chứng

```bash
rg -n "CDCInternalRegistry|QueueMonitoring|CDCInternalRegistryHandler|NewCDCInternalRegistryHandler" \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service

cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-web && npm run build
```

## Kết quả

- `rg`: không còn reference runtime sống; command trả no-match
- `go test ./...`: pass
- `npm run build`: pass
