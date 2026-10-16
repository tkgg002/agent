# 12 — Implementation Plan (AI execution log): Audit V4 P1

1. Đọc plan V4 (`09_tasks_solution_audit_v4_clear_and_grid.md`) và thực thi T1–T9 theo thứ tự: writer CDS → reader/clear CMS → router → FE.
2. T10: build và test từng repo (`go build ./cmd/... ./internal/...`, vì `./...` vướng `scratch/`, `docs/server_bk*` có sẵn).
3. T11: trace data thật trên `cdc_dw` (bindings, phân bố `target_table` theo operation, partitions) → viết harness tạm gọi chính `ClearActivity` trong transaction có ROLLBACK, seed binding thứ hai trùng tên bảng → 6 kịch bản. Kịch bản S5 lộ ra BUG-P1-FE (FE gửi master FQN) → sửa 1 dòng FE → build lại → xoá harness.
4. EXPLAIN DELETE để xác nhận partition pruning.
5. T12: rà soát bảo mật (chain middleware, tham số bind, kênh truyền `reason`, guard scope).
6. Ghi bằng chứng vào 05/06/08/11.

Còn lại: T13 (User commit tách theo feature), T14/P2 (chờ quyết định).
