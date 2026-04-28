# Implementation — Phase 5 Operational Pages Practical

## `src/pages/ActivityManager.tsx`

- Thay `tables: string[]` bằng `registryRows` đọc từ `/api/registry`.
- Tạo:
  - `registryByTarget`
  - `tableOptions`
  - `normalizeShadowSchema()`
- Đổi cột `Bảng đích` thành `Scope`:
  - dòng 1: `source_db.source_table`
  - dòng 2: `shadow_<source_db>.<target_table>`
- Đổi form tạo schedule:
  - field chọn scope giờ hiển thị `source object -> shadow target`
  - backend submit vẫn là `target_table`
- Thêm `Alert` giải thích đây là UI enriched trên contract legacy.

## `src/pages/DataIntegrity.tsx`

- Thêm helper:
  - `normalizeShadowSchema()`
  - `getShadowFqn()`
- Tạo `reportByTarget` để enrich failed logs từ chính recon report data.
- Đổi cột đầu bảng ở tab overview:
  - `Bảng` -> `Source / Shadow`
- Đổi cột đầu bảng ở tab failed logs:
  - show source/shadow context nếu có metadata
  - fallback về `target_table` nếu không map được
- Thêm `Alert` giải thích mọi action hiện vẫn gọi API theo `target_table` legacy nhưng operator đã có context chuẩn hơn.

## Kết quả

- Hai page vận hành chính đã bớt là "vỏ".
- Operator có thể chọn và đọc scope hành động chính xác hơn mà không phải đoán từ `target_table`.
