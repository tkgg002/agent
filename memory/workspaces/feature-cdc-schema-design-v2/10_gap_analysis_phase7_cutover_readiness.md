# Gap Analysis — Phase 7 Cutover Readiness

## Residual Technical Debt

1. `cdc_table_registry` vẫn còn được dùng như compatibility projection ở vài nơi để giữ shape `model.TableRegistry`.
2. `full_count` và `detect_timestamp_field` vẫn có side-effect update legacy registry khi bảng cũ còn tồn tại.
3. `sinkworker` hiện vẫn ghi shadow legacy vào `cdc_internal`; mới chỉ chuẩn hóa trigger identity xuống downstream.
4. CMS/API layer chưa được refactor toàn bộ sang CRUD V2 metadata.

## Why Cutover Is Still Reasonable

- Runtime lookup chính đã lấy metadata từ V2 trước.
- Master path và scheduler path đã được neo vào `cdc_system`.
- Với đợt wipe/bootstrap, bạn có thể seed V2 metadata làm nguồn sự thật chính và để compatibility layer chỉ đóng vai trò phụ.
