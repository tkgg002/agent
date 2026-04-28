# Solution — Phase 12 Wipe Script

## Output

- [wipe_cdc_runtime_v2.sql](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql)
- [wipe_bootstrap_v2.md](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md)

## Cách dùng

1. stop service
2. backup nếu cần
3. chạy wipe script
4. chạy migrate
5. chạy seed local
6. start lại service
7. verify theo runbook
