# Thiết kế Kỹ thuật: Audit và Hợp nhất Luồng xử lý (Flow-based Consolidation)

## 1. Thiết kế Hợp nhất Recon Module (Phase 1)
Sau khi đánh giá, một số file trong Giai đoạn 1 đã bị băm quá nhỏ. Chúng tôi đề xuất hợp nhất như sau:

### 1.1 `recon_tier_a` (Từ 5 file về 2 file)
- **Tệp tin hiện tại**: `recon_tier_a.go`, `recon_tier_a_helpers.go`, `recon_tier_a_lock.go`, `recon_tier_a_prune.go`, `recon_tier_a_run.go`.
- **Giải pháp gộp**:
  - Gộp tất cả các logic chạy, lock, prune và helper chính vào lại tệp tin duy nhất: `recon_tier_a.go` (khoảng 350-400 dòng).
  - Di chuyển các model dùng chung sang `recon_models.go`.
  - Xóa các file: `recon_tier_a_helpers.go`, `recon_tier_a_lock.go`, `recon_tier_a_prune.go`, `recon_tier_a_run.go`.

### 1.2 `recon_heal` (Từ 6 file về 2 file)
- **Tệp tin hiện tại**: `recon_heal.go`, `recon_heal_action.go`, `recon_heal_audit.go`, `recon_heal_legacy.go`, `recon_heal_models.go`, `recon_heal_utils.go`.
- **Giải pháp gộp**:
  - `recon_heal.go`: Chứa toàn bộ luồng đối soát & xử lý sai lệch (luồng chính, actions, audit và các hàm logic chạy chính).
  - `recon_heal_utils.go`: Chứa các helper, legacy logic, models phụ trợ.
  - Xóa các file: `recon_heal_action.go`, `recon_heal_audit.go`, `recon_heal_legacy.go`, `recon_heal_models.go`.

### 1.3 `provisioning_orchestrator` (Từ 6 file về 2 file)
- **Tệp tin hiện tại**: `provisioning_orchestrator.go`, `provisioning_orchestrator_actions.go`, `provisioning_orchestrator_helpers.go`, `provisioning_orchestrator_models.go`, `provisioning_orchestrator_recovery.go`, `provisioning_orchestrator_seed.go`.
- **Giải pháp gộp**:
  - `provisioning_orchestrator.go`: Chứa toàn bộ luồng điều phối chính, bao gồm cả actions,recovery và logic seed dữ liệu (Single Flow).
  - `provisioning_orchestrator_helpers.go`: Chứa các helper, struct models phụ trợ.
  - Xóa các file: `provisioning_orchestrator_actions.go`, `provisioning_orchestrator_recovery.go`, `provisioning_orchestrator_seed.go`, `provisioning_orchestrator_models.go`.

---

## 2. Thiết kế Refactor `transmuter.go` (Phase 2)
Thay vì chia tách thành `transmuter_run.go` và `transmuter_extract.go` như kế hoạch ban đầu, chúng tôi thiết kế phân rã như sau:

### 2.1 `transmuter.go` (Core Flow - Single Responsibility of Piping Data)
Giữ lại toàn bộ luồng chạy chính từ đầu đến cuối:
- Định nghĩa struct `TransmuterModule`, cache structures, mapping structs.
- Constructor `NewTransmuterModule`.
- `Run`: Điều phối toàn bộ luồng transmute.
- `loadMaster`: Tải cấu hình master binding.
- `shadowActive`: Kiểm tra trạng thái shadow.
- `loadRules` & `InvalidateRuleCache`: Tải và cache mapping rules.
- `fetchShadowBatch`: Đọc batch dữ liệu từ Shadow DB.
- `processBatch`: Lặp qua các dòng dữ liệu, chạy transform và mapping.
- `extractColumnsFn` & `extractColumns`: Trích xuất dữ liệu từ JSON shadow sang columns.
- `upsertMaster`: Thực hiện câu lệnh Postgres Upsert ON CONFLICT.

### 2.2 `transmuter_state.go` (Nhiệm vụ phụ trợ: Persistence)
- Chứa logic lưu trữ trạng thái chạy vào database điều khiển:
  - `markRuntimeSuccess`
  - `markRuntimeFailure`
  - `markRuntimeSkipped`
  - `persistRuntimeState`

### 2.3 `transmuter_utils.go` (Nhiệm vụ phụ trợ: Pure Data & SQL Helpers)
- Chứa các hàm chuyển đổi kiểu dữ liệu thuần túy (Pure Functions) không phụ thuộc luồng:
  - `gjsonValueToGo`, `unwrapMongoExtJSON`, `mongoNumberToGo`, `coerceForColumn`, `epochToTime`, `deterministicGpayID`, `sqlBindValueTransmute`.
  - Các hàm quote SQL: `quoteTransmuteIdent`, `quoteTransmuteQualified`, `sortedKeysAny`, `isJSONColumnType`, `isTimestampColumnType`.
