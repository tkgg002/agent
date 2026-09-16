# Kế hoạch Triển khai: Sửa lỗi hiển thị Pipelines cũ tại /data-integrity

## 1. Mục tiêu
Sửa lỗi trang `http://localhost:5173/data-integrity` hiển thị danh sách tất cả các pipelines cũ/inactive, đưa trang về trạng thái chỉ hiển thị và thống kê đúng các Pipelines đang HOẠT ĐỘNG (Active).

## 2. Các file cần sửa đổi (Proposed Changes)

### Component: `cdc-cms-web`

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Khôi phục và cập nhật logic `isShadowOff` và `isMasterSyncOff`.
- Tạo `activePipelines` bằng cách lọc `pipelines` qua bộ lọc `isShadowOff` và `isMasterSyncOff`.
- Cập nhật `flatData` để render từ `activePipelines`.
- Cập nhật `loading` state của Table bao gồm `isSourceObjectsLoading` và `isMastersLoading`.

#### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Nạp danh sách `sourceObjects` và `masters` từ API.
- Lọc `activePipelines` để tính tổng số bảng (`Tổng bảng`), số khớp (`Khớp`), và số lệch (`Lệch`) cho phần Statistic Cards ở Header.

## 3. Kế hoạch Kiểm thử (Verification Plan)
- Chạy `npx tsc --noEmit` hoặc `npm run build` trong `cdc-cms-web` để đảm bảo không có lỗi TypeScript/Build.
- Kiểm tra trang `http://localhost:5173/data-integrity` hiển thị đúng 1 Pipeline đang hoạt động (`shadow_traitestctphs.trans_his` -> `master_core_trans_proxy_history_service.trans_his`).
- Kiểm tra Thống kê Header `Tổng bảng` hiển thị đúng số lượng Pipelines active (1).
