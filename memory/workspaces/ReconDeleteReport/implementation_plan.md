# Kế hoạch Thêm Chức năng Xoá Phiên Đối Soát (Cập nhật: Fix hiển thị Phiên đã xử lý)

Tài liệu này mô tả phương án sửa đổi logic để danh sách các phiên đối soát đã xử lý (healed reports) hiển thị đúng tiến độ sau khi người dùng thực hiện chữa lành.

## User Review Required

> [!IMPORTANT]
> **Nguyên nhân lỗi và giải pháp:**
> 1. **Thiếu Tự động Làm mới (Cache Invalidation):** Mutation `useExecuteHealMutation` sau khi chạy thành công không gọi cơ chế invalidate cache của React Query. Kết quả là UI không tự động refetch lại danh sách lịch sử mới từ backend. Ta sẽ thêm hàm `onSuccess` để invalidate các query key `unhealed-reports`, `recon-history`, `recon-report`.
> 2. **Bộ lọc quá ngặt nghèo trên UI:** Bảng "Phiên đã xử lý" lọc bản ghi dựa trên điều kiện `healed_at != null`. Tuy nhiên, với các phiên chữa lành một phần (partially healed), backend thiết lập `healed_at = NULL` (chỉ set `status = "partially_healed"`). Ta sẽ cập nhật bộ lọc hiển thị tất cả các phiên có trạng thái `healed`, `partially_healed` hoặc có số lượng bản ghi đã chữa lành (`healed_count` hay `pruned_missing_src_count`) > 0.

## Proposed Changes

### Frontend (cdc-cms-web)

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Bổ sung callback `onSuccess` cho `useExecuteHealMutation` để làm mới cache của React Query.

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Cập nhật hằng số `healedReports` để lọc theo status (`healed`, `partially_healed`) hoặc số lượng bản ghi đã xử lý > 0.

---

## Verification Plan

### Automated Tests
1. **Kiểm tra frontend:**
   ```bash
   npx tsc --noEmit
   ```

### Manual Verification
1. Mở modal "Chữa lành đối soát" cho một bảng có lỗi drift.
2. Click "Thực hiện chữa lành".
3. Chờ tiến trình kết thúc, mở lại modal hoặc chuyển sang tab "Phiên đã xử lý".
4. Xác nhận phiên đối soát đó đã chuyển sang tab "Phiên đã xử lý" và hiển thị đầy đủ thông tin xử lý (số lượng đã heal, thời gian).
