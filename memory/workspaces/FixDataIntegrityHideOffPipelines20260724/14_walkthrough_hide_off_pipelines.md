# Walkthrough: Ẩn các Pipeline Shadow Off hoặc Master Sync Tắt (FE)

## Thay đổi đã thực hiện
Đã chỉnh sửa file [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx):

- **Helper Functions**:
  - `isShadowOff`: So sánh `shadowName` với danh sách `sourceObjects` thu được từ API. Nếu `isOnstream === false`, trả về `true` (Shadow off).
  - `isMasterSyncOff`: Nếu pipeline có `masterName`, so sánh với danh sách `masters` thu được từ API. Nếu không tìm thấy master config hoặc master có `is_active === false`, trả về `true` (Master sync tắt/chưa duyệt).

- **Filtering & Data Lineage**:
  - Lọc ra mảng `activePipelines` chỉ chứa các pipeline có Shadow `on` và Master Sync khác `Tắt`.
  - Gom nhóm và tính số lượng pipeline (`tableCount`) dựa trên `activePipelines`.

- **UX/Loading**:
  - Thêm `isSourceObjectsLoading` và `isMastersLoading` vào trạng thái `loading` của Table để không bị ẩn nhầm pipeline khi dữ liệu đang tải.

## Kết quả kiểm thử (Verification Results)
- `npx tsc --noEmit`: PASS (Không có lỗi TypeScript).
- `npm run build`: PASS (Build Vite thành công bundle `DataIntegrity-DKeBBZBq.js`).
