# 00_context — Audit config-local.yml (centralized-data-service)

> **Workspace**: `audit-config-local-2026-05-15`
> **Owner**: Brain (Antigravity) — read-only audit (CLAUDE.md §12 Brain Code Prohibition: KHÔNG sửa source).
> **Created**: 2026-05-15
> **Source request**: User yêu cầu kiểm tra `data-hub/centralized-data-service/config/config-local.yml` xem theo flow hiện tại key nào còn dùng / key nào không.

## Scope

- **Target file**: `data-hub/centralized-data-service/config/config-local.yml` (129 dòng).
- **Code scope**: toàn bộ `centralized-data-service/` (cmd/{worker,admin-api,sinkworker} + internal/* + pkgs/*).
- **Out of scope**: cdc-cms-service, cdc-auth-service, cdc-cms-web (chỉ animated worker plane).
- **Mục tiêu**: cho mỗi key trong YAML xác định:
  1. Có được parse vào `AppConfig` struct không?
  2. Có caller thực sự đọc giá trị không?
  3. Còn phù hợp với flow hiện tại (Debezium-only sau commit 8ef7d71 — remove Airbyte) không?

## Reference

- Cấu trúc `AppConfig`: `centralized-data-service/config/config.go:19–48`.
- Flow hiện tại: Mongo/PG → Debezium → Kafka → worker → shadow → master (xem `tech_stack.md` §Architecture).
- Lesson liên quan: 2026-05-05 "Validation BEFORE fallback merging" (config.go:243–249), 2026-04-06 "Airbyte retire" (commit 8ef7d71).
