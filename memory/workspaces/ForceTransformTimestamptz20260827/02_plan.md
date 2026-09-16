# 02 Plan — Force Transform + Timestamp Bug Fix

## Scope
- **Task 1 (Bug):** FE hiển thị transform in-progress sai trên child binding table
- **Task 2 (Feature):** Force transform mode — re-extract field từ `_raw_data` mà không cần NULL field trước
- **Task 2a (Fix):** `BuildCastExpr` ELSE branch sai cho `timestamptz`

## Skills khai báo (Pre-flight)
1. `golang-patterns` — BE batch_transform_handler, mapping_utils
2. `react-patterns` — FE TableRegistry
3. `database-design` — SQL chunked CTE UPDATE, _raw_data cast expr

## Architecture Decision
- Force transform KHÔNG chạy cùng normal rules → `force=true` chỉ SET field trong `force_fields`, bỏ `IS NULL` WHERE
- `force=false`: behavior cũ hoàn toàn không đổi
- Validate: `force=true` bắt buộc `force_fields` không rỗng → trả error ngay

## High-level Roadmap

### Phase 1 — BE Core (batch_transform_handler.go)
1. Thêm `Force bool`, `ForceFields []string` vào `BatchTransformPayload`
2. Sửa `runTransformJob`: tách nhánh force/normal khi build SET + WHERE
3. Validate input

### Phase 2 — BE Cast Fix (mapping_utils.go)
1. Tách case `timestamptz`/`timestamp with time zone` riêng
2. Fix ELSE branch: `::TIMESTAMP` → `::TIMESTAMPTZ`

### Phase 3 — BE HTTP handler
1. Đọc `force` + `force_fields` từ request body
2. Forward vào `BatchTransformPayload` → NATS

### Phase 4 — FE Bug Fix (TableRegistry.tsx)
1. Sửa key `activeTransformJobs` cho child binding: `b:${shadow_binding_id}`
2. Sửa render child row: `activeJobId={activeTransformJobs[\`b:${r.shadow_binding_id}\`]}`
