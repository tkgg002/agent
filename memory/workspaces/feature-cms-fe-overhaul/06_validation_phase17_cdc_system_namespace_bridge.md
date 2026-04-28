# Validation — Phase 17 cdc_system Namespace Bridge

## Lệnh kiểm chứng

```bash
rg -n "cdc_internal\\.(sources|cdc_wizard_sessions)" \
  /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal

cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service && go test ./...
```

## Kết quả

- grep: no match
- `go test ./...`: pass
