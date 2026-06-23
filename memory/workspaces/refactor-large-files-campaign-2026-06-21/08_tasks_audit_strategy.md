# Danh sách Task: Audit và Hợp nhất Luồng xử lý (Flow-based Consolidation)

## Task: Hợp nhất các file đã phân tách quá mức tại Recon Module (Phase 1)
- **Phase**: GĐ3 (Architecture)
- **Service Group**: Financial Core / Business
- **Service(s)**: centralized-data-service (recon package)
- **Mô tả**: Gộp các file nhỏ bị phân tách tại Phase 1 (`_helpers.go`, `_actions.go`, `_models.go`, `_recovery.go`, `_seed.go`, v.v.) về lại các file core tương ứng (`recon_tier_a.go`, `recon_heal.go`, `provisioning_orchestrator.go`) để giữ luồng code liên tục, không bị băm nhỏ.
- **Trạng thái**: [ ] TODO (chưa thực hiện)

### [Context]
- Current state: Giai đoạn 1 đã hoàn thành việc tách 6 file ban đầu thành tổng cộng 25 file phụ trợ. Điều này dẫn đến sự phân mảnh và băm nhỏ logic luồng chạy.
- Dependencies: `internal/service/recon/` package.
- ADR liên quan: ADR 2: Flow-based Consolidation.

### [Definition of Done]
- [ ] Gộp `recon_tier_a_helpers.go`, `recon_tier_a_lock.go`, `recon_tier_a_prune.go`, `recon_tier_a_run.go` vào lại `recon_tier_a.go`. Xóa các file phụ trợ thừa.
- [ ] Gộp `recon_heal_action.go`, `recon_heal_audit.go`, `recon_heal_legacy.go`, `recon_heal_models.go` vào lại `recon_heal.go`, giữ `recon_heal_utils.go` cho các pure helpers. Xóa các file phụ trợ thừa.
- [ ] Gộp `provisioning_orchestrator_actions.go`, `provisioning_orchestrator_recovery.go`, `provisioning_orchestrator_seed.go`, `provisioning_orchestrator_models.go` vào lại `provisioning_orchestrator.go`, giữ `provisioning_orchestrator_helpers.go` cho config và models. Xóa các file phụ trợ thừa.
- [ ] **[QA Gate]**: Đảm bảo toàn bộ dự án biên dịch thành công (`go build ./...`) và toàn bộ unit tests package `recon` chạy đạt 100% (`go test -v ./internal/service/recon/...`).
- [ ] **[Security Gate]**: Chạy rà soát bảo mật qua workflow `/security-agent` để đảm bảo không rò rỉ hoặc vi phạm an toàn thông tin khi chuyển đổi code.
- [ ] Blast radius verified: Không có thay đổi về logic nghiệp vụ, các method signatures và logic core được bảo toàn nguyên vẹn.
- [ ] Model Tracking: Ghi nhận đầy đủ các bước thực thi vào `05_progress.md` kèm theo timestamp và model tag.

---

## Task: Phân rã transmuter.go theo thiết kế mới (Phase 2)
- **Phase**: GĐ3 (Architecture)
- **Service Group**: Financial Core / Business
- **Service(s)**: centralized-data-service (master package)
- **Mô tả**: Thực hiện refactor file lớn `transmuter.go` (903 LoC). Giữ nguyên toàn bộ luồng pipeline dữ liệu shadow -> master (`Run`, `processBatch`, `extractColumns`, `upsertMaster`) trong file chính. Chỉ tách các nhiệm vụ phụ trợ độc lập: trạng thái chạy và helpers/quote SQL.
- **Trạng thái**: [ ] TODO (chưa thực hiện)

### [Context]
- Current state: File `transmuter.go` có kích thước 903 dòng, chứa cả logic piping dữ liệu và các helper logic conversion, quote SQL, DB state logging.
- Dependencies: `internal/service/master/` package.
- ADR liên quan: ADR 2: Flow-based Consolidation.

### [Definition of Done]
- [ ] Di chuyển các hàm ghi nhận trạng thái runtime (`markRuntimeSuccess`, `markRuntimeFailure`, `markRuntimeSkipped`, `persistRuntimeState`) sang file mới `transmuter_state.go`.
- [ ] Di chuyển các pure helpers và SQL helpers (`gjsonValueToGo`, `unwrapMongoExtJSON`, `mongoNumberToGo`, `coerceForColumn`, `epochToTime`, `deterministicGpayID`, `sqlBindValueTransmute`, `quoteTransmuteIdent`, `quoteTransmuteQualified`, `sortedKeysAny`, `isJSONColumnType`, `isTimestampColumnType`) sang file mới `transmuter_utils.go`.
- [ ] Rút gọn file chính `transmuter.go` nhưng giữ nguyên struct `TransmuterModule`, constructor và toàn bộ luồng chạy chính.
- [ ] **[QA Gate]**: Đảm bảo dự án biên dịch thành công và unit tests package `master` đạt 100% (`go test -v ./internal/service/master/...`).
- [ ] **[Security Gate]**: Chạy rà soát bảo mật qua workflow `/security-agent`.
- [ ] Blast radius verified: logic piping được bảo toàn, các methods giữ nguyên chức năng.
- [ ] Model Tracking: Ghi nhận đầy đủ các bước thực thi vào `05_progress.md` kèm theo timestamp và model tag.
