# Báo cáo Thay đổi (Change Report) - SortTransmuteSchedulesLastRun20260826

## File Thay đổi
- `cdc-cms-web/src/pages/TransmuteSchedules.tsx` (+17 lines, -4 lines)

## Chi tiết Thay đổi
- Import `useMemo` và `ColumnsType` từ `antd/es/table`.
- Thêm `sortedData` tự động sắp xếp danh sách schedules theo `last_run_at` mới nhất lên trên cùng.
- Bổ sung `sorter` và `defaultSortOrder: 'descend' as const` cho cột **Last run** trong bảng Antd.
- Cập nhật `dataSource` của `<Table>` sang `sortedData`.

## Verification
- `npm run build` thành công 100% (0 errors).
