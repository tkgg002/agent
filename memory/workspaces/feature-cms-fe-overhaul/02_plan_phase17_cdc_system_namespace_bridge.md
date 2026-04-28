# Plan — Phase 17 cdc_system Namespace Bridge

## Kế hoạch thực hiện

1. Audit model/repo nào trong CMS vẫn còn trỏ `cdc_internal`.
2. Đối chiếu với migration end-state của hệ thống.
3. Sửa namespace cho các model/repo runtime đang dùng thật:
   - `Source`
   - `WizardSession`
4. Verify bằng grep + `go test ./...`.
