# Yêu cầu: Ẩn các Pipeline có Shadow Off hoặc Master Sync Tắt trên tab Pipelines của Data Integrity

## Mô tả yêu cầu
Trong trang Data Integrity (`/data-integrity`), tab **Pipelines**:
- Kiểm tra trực tiếp ở Frontend (FE).
- Những pipeline nào có:
  - `Shadow: Off` (shadow active / is_active = false)
  HOẶC
  - `Master Sync: Tắt` (Master Sync bị tắt / disabled)
- **Hành vi kỳ vọng**: Không hiển thị (filter out / ẩn) các pipeline này lên danh sách tab Pipelines.

## Phạm vi tác động
- Frontend repository: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`
- File chính: `src/components/ReconPipelineGrid.tsx` và/hoặc `src/pages/DataIntegrity.tsx`

## DoD (Definition of Done)
- Pass toàn bộ Gates G1-G8.
- Chạy `npm run build` hoặc `npx tsc --noEmit` thành công.
- Ẩn đúng các row pipeline có Shadow Off hoặc Master Sync Tắt.
