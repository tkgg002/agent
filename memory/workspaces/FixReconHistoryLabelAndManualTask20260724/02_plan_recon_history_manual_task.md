# 02_plan_recon_history_manual_task.md — Kế hoạch đối soát thủ công & đổi nhãn Nhật ký đối soát

## I. MỤC TIÊU PHIÊN LÀM VIỆC
1. Đổi nhãn tab trong `ReconPipelineGrid.tsx`:
   - `Smoke` -> `Đối soát tự động`
   - `Recon` -> `Đối soát thủ công`
2. Bổ sung Task "Tiến trình đối soát thủ công" (Get JobWorker đang chạy đối soát):
   - Xây dựng API query các job đang chạy trong `cdc_system.recon_jobs` (`status IN ('PENDING', 'RUNNING')`).
   - Xây dựng hook FE `useActiveReconJobs` poll dữ liệu tiến trình jobworker.
   - Thêm tab "Tiến trình đối soát thủ công" hiển thị thanh tiến độ %, trạng thái, khoảng quét và số lượng lệch tạm tính.

## II. LỘ TRÌNH THỰC THI (ROADMAP)
- **Phase 1 (FE Label Update):** Đổi nhãn tab Smoke / Recon trong `ReconPipelineGrid.tsx`.
- **Phase 2 (Backend JobWorker API):** Bổ sung endpoint `GET /api/reconciliation/jobs/active` trong `centralized-data-service` / `cdc-cms-service` (nếu chưa có).
- **Phase 3 (FE JobWorker Widget):** Bổ sung UI tab "Tiến trình đối soát thủ công" với thanh Progress và thông tin chi tiết công việc ngầm đang thực thi.
- **Phase 4 (Verification & Governance):** Verify build FE/BE, chạy `verify_governance.py`, cập nhật report.
