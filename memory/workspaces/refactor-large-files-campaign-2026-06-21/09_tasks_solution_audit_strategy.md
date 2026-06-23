# Hồ sơ Giải pháp Kỹ thuật: Audit và Hợp nhất Luồng xử lý (Flow-based Consolidation)

## 1. Hợp nhất `recon_tier_a`
- Tệp tin chính `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go` sẽ nhận lại:
  - Hàm `Run` chính từ `recon_tier_a_run.go`.
  - Logic Lock/Unlock từ `recon_tier_a_lock.go`.
  - Logic Pruning từ `recon_tier_a_prune.go`.
  - Các helper nội bộ từ `recon_tier_a_helpers.go`.
- Các file `recon_tier_a_helpers.go`, `recon_tier_a_lock.go`, `recon_tier_a_prune.go`, `recon_tier_a_run.go` sẽ bị xóa hoàn toàn.
- Commit hoặc backup sẽ được tạo trước khi xóa.

## 2. Hợp nhất `recon_heal`
- Tệp tin chính `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go` sẽ nhận lại:
  - Logic audit từ `recon_heal_audit.go`.
  - Logic thực thi sửa lỗi từ `recon_heal_action.go`.
  - Logic legacy từ `recon_heal_legacy.go`.
- File `recon_heal_utils.go` sẽ chứa các struct models và các helper độc lập.
- Các file `recon_heal_action.go`, `recon_heal_audit.go`, `recon_heal_legacy.go`, `recon_heal_models.go` sẽ bị xóa hoàn toàn.

## 3. Hợp nhất `provisioning_orchestrator`
- Tệp tin chính `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/provisioning_orchestrator.go` sẽ nhận lại:
  - Logic state machine, action runner từ `provisioning_orchestrator_actions.go`.
  - Logic recovery từ `provisioning_orchestrator_recovery.go`.
  - Logic seed dữ liệu từ `provisioning_orchestrator_seed.go`.
- File `provisioning_orchestrator_helpers.go` sẽ chứa các helper cấu hình và struct models.
- Các file `provisioning_orchestrator_actions.go`, `provisioning_orchestrator_recovery.go`, `provisioning_orchestrator_seed.go`, `provisioning_orchestrator_models.go` sẽ bị xóa hoàn toàn.

## 4. Phân rã `transmuter.go` (Giai đoạn 2)
- Tệp tin chính `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` sẽ giữ nguyên luồng chạy chính và struct `TransmuterModule` để bất kỳ ai mở file cũng đọc được toàn bộ quy trình: shadow -> run -> fetch -> mapping -> upsert.
- Các helper ghi nhận trạng thái (`markRuntimeSuccess`, `markRuntimeFailure`, `markRuntimeSkipped`, `persistRuntimeState`) sẽ nằm trong `transmuter_state.go`.
- Các pure helpers và SQL helpers sẽ nằm trong `transmuter_utils.go`.
