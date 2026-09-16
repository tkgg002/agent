# Báo cáo Thay đổi (Overview Change Report) - FixDataIntegrityFilterActivePipelines20260826

## Danh sách Tệp tin Thay đổi (Changed Files)
1. `cdc-cms-web/src/components/ReconPipelineGrid.tsx` (+46 lines, -28 lines)
2. `cdc-cms-web/src/pages/DataIntegrity.tsx` (+30 lines, -82 lines)

## Tổng quan Thay đổi
- Khôi phục và nâng cấp logic matching của 2 hàm helper `isShadowPipelineOff` và `isMasterPipelineSyncOff` tại `ReconPipelineGrid.tsx`.
- Lọc danh sách `activePipelines` dựa trên `sourceObjects` và `masters`.
- Cập nhật `flatData` hiển thị và gom nhóm dựa trên `activePipelines`.
- Đồng bộ tính toán `activePipelines` tại `DataIntegrity.tsx` cho các thẻ Thống kê Header (`Tổng bảng`, `Khớp`, `Lệch`).
- **Loại bỏ 2 Tab dư thừa**: Bỏ Tab **Tổng quan** (Overview) và Tab **Backfill _source_ts**.
- **Tối ưu Tab & Card "Lỗi đồng bộ"**: Lọc `unresolvedFailedLogs` chỉ đếm và hiển thị các bản ghi lỗi chưa được xử lý (`status !== 'resolved'`), giúp con số hiển thị chính xác thực tế.

## Kiểm thử & Verification
- `npx tsc --noEmit` thành công 100% không văng lỗi kiểu dữ liệu.
