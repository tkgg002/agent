# Implementation — Phase 16 Sources Connectors V2 Screen

## Audit kết luận

API hiện tại đủ để dựng một màn V2-native mức đầu tiên:

- `GET /api/v1/system/connectors`
  - cho runtime connector state + tasks
- `GET /api/v1/sources`
  - cho source fingerprint đã persist

Chưa có API riêng cho `connection_registry`/`source_object_registry`/`shadow_binding`, nên phase này tập trung vào lớp source-runtime/fingerprint trước.

## Thay đổi đã áp dụng

- Refactor `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/SourceConnectors.tsx`
  - fetch thêm `/api/v1/sources`
  - thêm summary cards:
    - connectors
    - fingerprints
    - linked
    - orphans
  - thêm cảnh báo mismatch:
    - fingerprint-only
    - runtime-only connectors
  - tách page thành 2 tab:
    - `Connectors`
    - `Source Fingerprints`
  - giữ nguyên các runtime actions destructive đã có

## Kết luận

Đây là màn V2-native đầu tiên ở FE theo nghĩa:
- không còn chỉ nhìn registry/table
- bắt đầu nhìn runtime connector và metadata fingerprint như hai lớp riêng
- giúp operator phát hiện drift metadata sớm hơn

## Debt còn lại

- `GET /api/v1/sources` hiện vẫn dựa trên `cdc_internal.sources`
- chưa có màn riêng cho:
  - `connection_registry`
  - `source_object_registry`
  - `shadow_binding`
  - `master_binding`
