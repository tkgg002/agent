# Solution — Phase 11 Wipe Bootstrap Runbook

## Output

- [Makefile](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/Makefile)
- [wipe_bootstrap_v2.md](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md)

## Cách dùng nhanh

1. chuẩn bị file seed local từ template
2. wipe dữ liệu theo môi trường
3. chạy `make migrate`
4. chạy seed local hoặc `make migrate-bootstrap` cho demo
5. restart service
6. verify theo checklist SQL trong runbook
