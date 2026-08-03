# 13_analysis_recon_history_manual_task.md — Phân tích chi tiết hiện trạng & giải pháp

## I. PHÂN TÍCH HIỆN TRẠNG
1. **Giao diện `ReconPipelineGrid.tsx`**:
   - Card `"Nhật ký đối soát (30 phiên gần nhất)"` hiện đang dùng 2 Tabs:
     * Key `'smoke'`, label `'Smoke'`: Hiển thị lịch sử các phiên smoke test tự động.
     * Key `'recon'`, label `'Recon'`: Hiển thị lịch sử các phiên đối soát thủ công (hash_window, full_diff, deep_check).
   - Tên tiếng Anh gây chưa rõ ràng cho người dùng vận hành Việt Nam.

2. **Tiến trình JobWorker đối soát thủ công**:
   - Khi trigger đối soát thủ công bất đồng bộ (`POST /api/reconciliation/check-async`), hệ thống tạo một record trong `cdc_system.recon_jobs` với trạng thái `PENDING` / `RUNNING`.
   - `ReconJobWorker` liên tục cập nhật `progress_percent`, `checkpoint_ts`, và `total_diff_count` vào DB.
   - Chưa có giao diện tập trung để xem tiến trình JobWorker đang chạy trên card Nhật ký đối soát.

## II. ĐỀ XUẤT NÂNG CẤP
1. Đổi tên nhãn 2 Tab cũ sang Tiếng Việt chuẩn: "Đối soát tự động" & "Đối soát thủ công".
2. Bổ sung Tab 3 "Tiến trình đối soát thủ công": Quét các job đang `RUNNING`/`PENDING`, hiển thị Progress bar và thông tin JobWorker thực thi realtime.
