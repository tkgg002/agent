# Phân tích kỹ thuật: Ẩn Pipeline Shadow Off hoặc Master Sync Tắt trên FE

## 1. Hiện trạng
- Component `ReconPipelineGrid.tsx` trong `cdc-cms-web/src/components/ReconPipelineGrid.tsx` đảm nhận hiển thị mảng pipelines.
- Dữ liệu `sourceObjects` (`/api/v1/source-objects`) quyết định trạng thái Shadow: `on` (isOnstream = true) hoặc `off` (isOnstream = false).
- Dữ liệu `masters` (`/api/v1/masters`) và `schedules` (`/api/v1/schedules`) quyết định trạng thái Master Sync: `Sync: Realtime`, `Sync: Hẹn giờ`, `Sync: Manual` (nếu active) hoặc `Sync: Tắt` / `Sync: Tắt (Chưa duyệt)` (nếu master ngưng active hoặc không tìm thấy master config).

## 2. Tiêu chí lọc trên FE (Filtering Criteria)
Một `PipelineRow` `p` sẽ bị **ẨN (FILTER OUT)** nếu thỏa mãn 1 trong 2 điều kiện:
1. **Shadow: Off**: `isShadowOff(p)` = `true` (khi `isOnstream === false`).
2. **Master Sync: Tắt**: `isMasterSyncOff(p)` = `true` (khi `p.masterName` có giá trị và `mstObj` không tồn tại hoặc `!mstObj.is_active`).

## 3. Xử lý Edge-case Loading
- Khi component mới mount, `sourceObjects` và `masters` có thể đang ở trạng thái `isLoading` / `undefined`.
- Để tránh việc filter nhầm khi chưa có dữ liệu `sourceObjects` / `masters`, kết hợp cờ `isLoading` của 2 query này với `loading` chung của Table:
  `const isGridLoading = loading || isSourceObjectsLoading || isMastersLoading;`
- Nhờ đó Table hiển thị loading state cho đến khi có đủ data `sourceObjects` & `masters` rồi mới tiến hành lọc an toàn.
