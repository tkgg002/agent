# 08_tasks_recon_history_manual_task.md — Danh sách Task thực thi

- [ ] Task 1: Cập nhật nhãn Tab "Smoke" thành "Đối soát tự động" và "Recon" thành "Đối soát thủ công" trong `ReconPipelineGrid.tsx`.
- [ ] Task 2: Kiểm tra & bổ sung API backend `GET /api/reconciliation/jobs/active` để truy vấn danh sách `ReconJob` đang ở trạng thái `PENDING` / `RUNNING`.
- [ ] Task 3: Thêm hook `useActiveReconJobs` trong `useReconStatus.ts` để tự động poll dữ liệu tiến trình jobworker đang chạy (3s-5s/lần).
- [ ] Task 4: Tích hợp Tab thứ 3 "Tiến trình đối soát thủ công" hiển thị thanh tiến độ %, trạng thái, checkpoint_ts, và total_diff_count trong `ReconPipelineGrid.tsx`.
- [ ] Task 5: Chạy test verification & linter quy trình `verify_governance.py`.
