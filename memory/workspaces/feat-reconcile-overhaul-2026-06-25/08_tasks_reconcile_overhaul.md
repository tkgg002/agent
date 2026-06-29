# Tasks: Reconcile Component Overhaul

## Phase 1: Research & Audit [DONE]
- [x] Rà soát cấu trúc bảng `cdc_reconciliation_report` và các migration.
- [x] Phân tích logic ghi report `stampA` và `stampB` trong `internal/service/recon/recon_engine_segment_b.go`.
- [x] Phân tích cách truy vấn của `cdc-cms-service` lên bảng này.

## Phase 2: Technical Design & Planning [DONE]
- [x] Xây dựng yêu cầu chi tiết (`01_requirements_reconcile_overhaul.md`).
- [x] Thiết lập roadmap thực thi (`02_plan_reconcile_overhaul.md`).
- [x] Thiết kế kỹ thuật chi tiết (`03_implementation_reconcile_overhaul.md`).
- [x] Soạn thảo giải pháp kỹ thuật chi tiết kèm code demo (`09_tasks_solution_reconcile_overhaul.md`).

## Phase 3: User Approval [PENDING]
- [ ] Trình bày giải pháp cho User và chờ duyệt.

## Phase 4: Implementation [PENDING]
- [ ] Sửa đổi `internal/service/recon/recon_engine_segment_b.go` để tích hợp hàm `stamp` chung và logic deduplicate.
- [ ] Sửa đổi `internal/service/recon/recon_engine_run.go` bổ sung hàm `pruneSuccessReports` và tích hợp vào các chu kỳ quét.
- [ ] Sửa đổi `internal/service/recon/recon_tier_b.go` để gọi hàm `pruneSuccessReports` khi chạy Segment B.

## Phase 5: Verification [PENDING]
- [ ] Chạy bộ tests trong `internal/service/recon/...` để đảm bảo không lỗi biên dịch/logic.
- [ ] Chạy bộ integration tests `test/internal/handler/...`.
- [ ] Thực hiện Security Check bằng `/security-agent`.
- [ ] Xuất báo cáo bàn giao `report_reconcile_overhaul_2026_06_25.md` ghi nhận sự thay đổi dòng code.
