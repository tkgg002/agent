# Báo cáo thay đổi (11_report)

## Các file đã thay đổi
- File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`
- Số lượng dòng thay đổi: ~35 dòng code được thêm/chỉnh sửa.

## Chi tiết thay đổi
1. Thêm `isSourceObjectsLoading` và `isMastersLoading` từ `useQuery`.
2. Tạo hàm `isShadowOff(p)` để kiểm tra nếu shadow bị tắt (`isOnstream === false`).
3. Tạo hàm `isMasterSyncOff(p)` để kiểm tra nếu master sync bị tắt (`!mstObj` hoặc `!mstObj.is_active`).
4. Lọc `activePipelines` từ `pipelines` bằng `useMemo`.
5. Đưa `activePipelines` vào `flatData` để render danh sách pipeline và tính `tableCount` chính xác.
6. Cập nhật `loading` prop của Table thành `loading || isSourceObjectsLoading || isMastersLoading`.
