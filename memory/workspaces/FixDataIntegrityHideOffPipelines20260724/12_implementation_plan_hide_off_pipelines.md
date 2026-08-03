# Implementation Plan: Ẩn các Pipeline có Shadow Off hoặc Master Sync Tắt trên tab Pipelines FE

## User Review Required
> [!NOTE]
> Thay đổi này thuần túy thực hiện ở tầng Frontend (`ReconPipelineGrid.tsx`). Tất cả các pipeline có trạng thái `Shadow: Off` hoặc `Master Sync: Tắt` sẽ bị loại khỏi bảng hiển thị tab Pipelines.

## Proposed Changes

### Component: `ReconPipelineGrid`
File: [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)

#### Details:
1. Thêm hàm kiểm tra `isShadowOff` và `isMasterSyncOff` dựa trên data từ `sourceObjects` và `masters`.
2. Lọc mảng `pipelines` trong `useMemo` trước khi dựng `flatData`.
3. Bổ sung cờ `isSourceObjectsLoading` & `isMastersLoading` vào `loading` prop của Table để đảm bảo không bị flash hay ẩn nhầm khi đang fetch.

## Verification Plan

### Automated Tests / Build Check
- Run `npm run build` trong `cdc-cms-web` để đảm bảo không có lỗi TypeScript hay JSX syntax error.
