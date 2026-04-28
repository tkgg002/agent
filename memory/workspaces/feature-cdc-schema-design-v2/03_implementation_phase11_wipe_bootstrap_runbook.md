# Implementation — Phase 11 Wipe Bootstrap Runbook

## Thay đổi trong repo

### 1. Cập nhật Makefile

- [Makefile](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/Makefile)

Đã đổi:

- `migrate`
  - từ chạy mỗi `migrations/001_init_schema.sql`
  - sang chạy toàn bộ `migrations/*.sql` theo lexical order
- thêm `migrate-bootstrap`
  - chạy full migrations
  - sau đó seed template V2

### 2. Thêm runbook

- [wipe_bootstrap_v2.md](/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/deployments/runbooks/wipe_bootstrap_v2.md)

Runbook gồm:

- tiền đề trước cutover
- cách copy/sửa file seed
- thứ tự dừng service / backup / wipe / migrate / seed / restart
- checklist SQL verify sau bootstrap

## Giá trị đạt được

- repo không còn chỉ dẫn migrate lỗi thời
- team có playbook rõ để reset lớn theo V2
