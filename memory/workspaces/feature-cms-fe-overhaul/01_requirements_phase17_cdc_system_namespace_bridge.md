# Requirements — Phase 17 cdc_system Namespace Bridge

## Mục tiêu

- Loại bỏ namespace drift còn sót trong CMS backend cho các bảng system đã được move sang `cdc_system`.
- Ưu tiên các model/repo đang phục vụ trực tiếp cho các màn mới:
  - `Source`
  - `WizardSession`

## Yêu cầu

1. Nếu migration end-state đã move bảng sang `cdc_system`, CMS model/repo không được tiếp tục trỏ `cdc_internal`.
2. Chỉ sửa những chỗ đã audit là đang được dùng thật trong runtime.
3. Verify lại backend tests sau khi đổi namespace.
