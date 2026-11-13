---
tier: T2
suffix: payment_v2
workspace: audit-payment-v2-2026-11-10
---
# 04 Decisions — `Payment-v2` Readiness & Architecture Decisions

- **ADR-PV2-01 (DG0 Resolution `C-CONTRA-1`):** Xác nhận trực tiếp tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_bson_restorer.go:111-131` rằng `ApplySourceFieldTypesToBSONDoc` hiện đang chỉ lặp qua 4 khóa nested `[]string{"info", "sender", "receiver", "error"}` và chỉ xóa `transaction_history_id` / `transactionHistoryId`. Do đó cần mở rộng danh sách này (hoặc tự động hóa theo schema) khi áp dụng luồng khôi phục BSON (`multi_shadow`) cho `payment_bill_payload` (11 cột JSONB, khóa ngoại `payment_bill_id`) và `payment_payload` (7 cột JSONB, khóa ngoại `payment_id`).
- **ADR-PV2-02 (Confluence Inline Comments):** Theo 5 bình luận đã chốt trên Confluence (`512557312`, `513179664`, `512590004`):
  1. Chấp nhận có downtime khi chuyển ghi $\rightarrow$ bỏ hoàn toàn bảng `payment_bill_control` và `payment_control` (`write_mode`).
  2. Gộp `payment_bill*` và `payment*` nằm chung trong 1 PostgreSQL database.
- [2026-11-10T11:09:29+07:00] [Agent:laya_gate] DG1 D-BEHAV-1 FLAG resolved: Code Demo 1 in 09_tasks_solution_payment_v2.md returns "{" + strings.Join(parts, ",") + "}", true for PostgreSQL array columns before composite JSON marshaling. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:361-375
- [2026-11-10T11:09:29+07:00] [Agent:laya_gate] DG2 D-BEHAV-1 FLAG resolved: Code Demo 1 in 09_tasks_solution_payment_v2.md returns "{" + strings.Join(parts, ",") + "}", true for PostgreSQL array columns before composite JSON marshaling. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:361-375
- [2026-11-10T11:12:43+07:00] [Agent:laya_gate] DG3 C-CONTRA-1 FLAG resolved: Verified directly in mongo_bson_restorer.go lines 111-131. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/mongo_bson_restorer.go:111-131
- [2026-11-10T11:40:17+07:00] [Agent:laya_gate] DG1 D-BEHAV-1 FLAG resolved: Code Demo 1 in 09_tasks_solution_payment_v2.md returns "{" + strings.Join(parts, ",") + "}", true for PostgreSQL array columns before composite JSON marshaling. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:361-375
- [2026-11-10T11:40:17+07:00] [Agent:laya_gate] DG2 D-BEHAV-1 FLAG resolved: Code Demo 1 in 09_tasks_solution_payment_v2.md returns "{" + strings.Join(parts, ",") + "}", true for PostgreSQL array columns before composite JSON marshaling. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go:361-375
- [2026-11-10T12:03:00+07:00] [Agent:laya_gate] DG1 D-BEHAV-1 FLAG resolved: REF: /Users/trainguyen/Documents/work/agent/memory/workspaces/audit-payment-v2-2026-11-10/09_tasks_solution_payment_v2.md:61-106 formatPostgresArrayLiteral returns brace-enclosed comma-separated string
- [2026-11-10T12:03:03+07:00] [Agent:laya_gate] DG2 D-BEHAV-1 FLAG resolved: REF: /Users/trainguyen/Documents/work/agent/memory/workspaces/audit-payment-v2-2026-11-10/09_tasks_solution_payment_v2.md:61-106 formatPostgresArrayLiteral returns brace-enclosed comma-separated string
- [2026-11-12T10:28:54+07:00] [Agent:laya_gate] DG3 A-NORMALIZER-ARRAY FLAG resolved: REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go:45-54: normalizePostgresType maps dataType ARRAY to ANSI SQL array types like TEXT[] and BIGINT[] based on udtName
