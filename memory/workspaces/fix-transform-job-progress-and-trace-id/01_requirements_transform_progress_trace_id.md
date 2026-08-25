# 01_REQUIREMENTS: PROGRESS % VÀ TRACE ID CHO CẢ TRANSFORM VÀ TRANSMUTE

## 1. Yêu cầu Giao diện & Trải nghiệm (UI/UX)
1. **Live Progress Calculation (Cả Shadow & Masters):**
   - Đếm trước tổng số bản ghi cần xử lý (`total_rows`).
   - Cập nhật % tiến độ thực tế theo thời gian thực: `Hoàn thành / Tổng số rows (X%)`.
2. **Compact Trace ID Copy Icon (Siêu gọn):**
   - KHÔNG hiển thị chuỗi dài gây vỡ layout/chiếm diện tích.
   - CHỈ hiển thị 1 icon copy nhỏ gọn (Tooltip: "SigNoz Trace ID: ... (Click để copy)"). Click vào sẽ copy vào clipboard và hiển thị thông báo toast.
3. **Persistence across F5 (Chống mất trạng thái):**
   - Khi F5 tại `/shadow` hoặc `/masters`, trạng thái hoàn thành kèm số lượng rows và icon Trace ID vẫn luôn luôn được hiển thị tức thì từ Read Model.

## 2. Phạm vi Kỹ thuật
- **Database:** Bổ sung `total_rows` vào `cdc_system.transform_jobs` và `cdc_system.transmute_jobs`.
- **Worker Service:** `batch_transform_handler.go` và `transmuter.go`.
- **CMS Service:** Models, Repos, Handlers, và SQL LATERAL joins.
- **CMS Web:** `TableRegistry.tsx` (`TransformJobStatus`) và `MasterRegistry.tsx` (`TransmuteJobStatus`).
