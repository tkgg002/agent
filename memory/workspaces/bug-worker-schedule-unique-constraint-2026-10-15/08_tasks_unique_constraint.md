# 08 Tasks — bug-worker-schedule-unique-constraint-2026-10-15

- [x] **Task 1:** Đọc `GEMINI.md` và `lessons.md`, khởi tạo Workspace đạt chuẩn Rule #4 & Rule #16.
- [x] **Task 2:** Ghi bài học Mid-Session Fix (Rule #5 & Rule #6) vào `agent/memory/global/lessons.md` (kèm kiểm tra số dòng trước/sau: 1840 -> 1847).
- [x] **Task 3:** Truy vết mã nguồn (`cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`) và CSDL (`cdc_worker_schedule` schema/constraints) để xác định chính xác vị trí ném lỗi `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"`.
- [x] **Task 4:** Thiết kế giải pháp kỹ thuật tối ưu vào `09_tasks_solution_unique_constraint.md` và `12_implementation_plan_unique_constraint.md` + `13_analysis_unique_constraint.md`, tuân thủ kỷ luật Propose-Only (Rule #1 & Rule #13).
- [x] **Task 5:** Kiểm tra tuân thủ quy trình (Governance Pre-flight Check) trước khi phản hồi User.
- [x] **Task 6:** Tạo migration `026_fix_worker_schedule_unique_scope.sql` & cập nhật `migrations/seed/worker_schedules.sql` trong `cdc-cms-service`.
- [x] **Task 7:** Cập nhật Backend (`create_worker_schedule.go`, `update_schedule.go`, `schedule_handler.go`) và viết unit test `canonical_target_table_test.go` trong `cdc-cms-service`.
- [x] **Task 8:** Cập nhật Worker (`source_count_runner.go`) trong `centralized-data-service`.
- [x] **Task 9:** Cập nhật Frontend (`ActivityManager.tsx`, `apiError.ts`) trong `cdc-cms-web`.
- [x] **Task 10:** Thực thi migration, chạy unit tests, build, restart services và kiểm thử E2E HTTP thật trên `:8083` (Gate G1–G8), ghi báo cáo vào `06_validation_unique_constraint.md` & `11_report_unique_constraint.md`.

