# Danh sách Task: Chuẩn hóa phân loại đối soát (Refactor type_recon)

## Phase 1: Triển khai Centralized Data Service (CDS)
- [ ] 1.1. Sửa `ReconPayload` struct trong [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go): thay `tier` bằng `type_recon`.
- [ ] 1.2. Cấu trúc lại toàn bộ `switch` case của `recon_check_handler.go` theo `TypeRecon` (`smoke`, `hash_window`, `full_diff`, `deep_check`, `prune`).
- [ ] 1.3. Cập nhật `TimeBoundedDiffMissingFromShadow` trong [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go) để trả về thêm `destCount`.
- [ ] 1.4. Cập nhật các vị trí gọi `TimeBoundedDiffMissingFromShadow` ở [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go) và [recon_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_handler.go).
- [ ] 1.5. Chạy unit tests cho CDS recon để verify.

## Phase 2: Triển khai CMS Service (cdc-cms-service)
- [ ] 2.1. Sửa struct `ReconCheckCommand` trong [recon_check.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go): thay `Tier` bằng `TypeRecon`.
- [ ] 2.2. Sửa handler [reconciliation_handler_commands.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go) để lấy `type_recon` thay cho `tier` và dispatch đúng command.
- [ ] 2.3. Sửa câu truy vấn SQL UNION ALL trong [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go).
- [ ] 2.4. Sửa logic format status & drift trong [reconciliation_handler_reports.go](file:///Users/trainguyen/Documents/work/cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go).
- [ ] 2.5. Chạy test suite của CMS.

## Phase 3: Triển khai CMS Web (cdc-cms-web)
- [ ] 3.1. Sửa API mutation trong [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts).
- [ ] 3.2. Sửa modal state & prop binding trong [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx).
- [ ] 3.3. Sửa map value & callback param trong [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx).
- [ ] 3.4. Cập nhật `buildPipelines` và UI rendering trong [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx).

## Phase 4: Xác minh tích hợp
- [ ] 4.1. Khởi động các service và kiểm tra tích hợp trên giao diện.
