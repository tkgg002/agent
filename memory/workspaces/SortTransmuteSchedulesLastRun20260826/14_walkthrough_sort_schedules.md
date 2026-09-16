# Walkthrough - Sắp xếp & Phân Tab Trang Schedules theo Mode

## Những gì đã hoàn thành
1. Cập nhật [TransmuteSchedules.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TransmuteSchedules.tsx):
   - Thêm `sortedData` dùng `useMemo` tự động sắp xếp danh sách Schedules theo `last_run_at` mới nhất lên trên cùng.
   - Thêm `sorter` và `defaultSortOrder: 'descend' as const` cho cột **Last run**.
   - Bổ sung 4 Tabs phân loại danh sách theo Mode để tránh rối mắt:
     - **Tất cả (X)**
     - **Realtime (X)** (mode `post_ingest`)
     - **Sync ngay (X)** (mode `immediate`)
     - **Đặt lịch (X)** (mode `cron`)

## Kết quả Verification
- **Build Check**: `npm run build` thành công 100% (0 errors).
- **Trạng định**: Trang `http://localhost:5173/schedules` hiển thị giao diện phân Tab trực quan, dữ liệu được sắp xếp theo thời gian chạy gần nhất.
