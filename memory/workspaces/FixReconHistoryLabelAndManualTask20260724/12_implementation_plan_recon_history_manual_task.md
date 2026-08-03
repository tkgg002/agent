# 12_implementation_plan_recon_history_manual_task.md — Kế hoạch triển khai chi tiết AI

## I. MỤC TIÊU
1. Cập nhật nhãn Tabs trong card "Nhật ký đối soát (30 phiên gần nhất)":
   - Tab 1: "Đối soát tự động" (thay cho "Smoke")
   - Tab 2: "Đối soát thủ công" (thay cho "Recon")
2. Bổ sung task hiển thị tiến trình đối soát thủ công (Lấy thông tin JobWorker đang chạy đối soát ngầm).

## II. DANH SÁCH FILE THAY ĐỔI
- [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- [MODIFY] [recon_job_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon_job_repo.go)
- [MODIFY] [recon_job_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_job_handler.go)

## III. BƯỚC THỰC THI CHÍNH
1. Cập nhật UI Label trong `ReconPipelineGrid.tsx`.
2. Khai báo endpoint API `GET /api/reconciliation/jobs/active` ở Backend.
3. Khai báo Hook `useActiveReconJobs` ở Frontend.
4. Tích hợp Tab "Tiến trình đối soát thủ công" hiển thị thanh tiến độ %, trạng thái và metric của JobWorker.
