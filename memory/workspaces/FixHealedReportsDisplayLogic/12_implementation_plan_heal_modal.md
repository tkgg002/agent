# Kế hoạch Triển khai Chi tiết (Implementation Plan)

## 1. Mục tiêu
Cập nhật logic phân loại tab của các phiên đối soát trong `ExecuteHealModal.tsx` để ẩn các phiên đã chữa lành khỏi danh sách "Phiên chưa xử lý" và hiển thị chúng ở danh sách "Phiên đã xử lý".

## 2. Chi tiết các bước thực hiện

### Bước 1: Chuẩn bị tiến trình
Ghi nhận tiến trình bắt đầu vào file `05_progress_heal_modal.md`.

### Bước 2: Sửa đổi source code
Tại file `cdc-cms-web/src/components/ExecuteHealModal.tsx`, thay thế logic từ dòng 42-46:
```typescript
  const reports = data?.data || EMPTY_ARRAY;
  const totalUnhealed = data?.total || 0;
  const healedReports = (historyData?.data || []).filter(
    (r: any) => r.healed_at != null || r.status === 'healed'
  );
```
bằng logic mới:
```typescript
  const isReportFullyHealed = (r: any): boolean => {
    if (r.status === 'ok') return true;
    if (r.healed_at != null || r.status === 'healed') return true;
    
    const hasMissing = (r.missing_count || 0) > 0;
    const hasStale = (r.stale_count || 0) > 0;
    const hasOrphan = (r.orphan_count || 0) > 0;
    
    if (!hasMissing && !hasStale && !hasOrphan) return true;
    
    const missingOk = !hasMissing || ((r.healed_missing_dest_count || 0) >= r.missing_count);
    const staleOk = !hasStale || ((r.healed_mismatched_count || 0) >= r.stale_count);
    const orphanOk = !hasOrphan || ((r.pruned_missing_src_count || 0) >= r.orphan_count);
    
    return missingOk && staleOk && orphanOk;
  };

  const reports = (data?.data || []).filter((r: any) => !isReportFullyHealed(r));
  const totalUnhealed = reports.length;
  const healedReports = (historyData?.data || []).filter(
    (r: any) => r.status !== 'ok' && r.status !== 'error' && isReportFullyHealed(r)
  );
```

### Bước 3: Xác minh (Verify)
- Chạy lệnh `npx tsc --noEmit` ở thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` để đảm bảo code compile thành công.
- Lưu lại kết quả verify vào `06_validation_heal_modal.md`.

### Bước 4: Hoàn tất
Cập nhật trạng thái hoàn thành vào `05_progress_heal_modal.md`.

## 3. Cập nhật Phase 2 (Smoke Check logic update - 2026-07-13)
- **Mục tiêu**: Loại bỏ loại kiểm tra Smoke Check khỏi tab Phiên đã xử lý và đảm bảo chúng không được coi là fully healed (do không hỗ trợ chữa lành bằng ID và không có số lượng lỗi chi tiết).
- **Chi tiết sửa đổi**:
  - Thêm kiểm tra `if (r.check_type === 'smoke') return false;` vào đầu hàm `isReportFullyHealed` trong `ExecuteHealModal.tsx`.
  - Cập nhật filter của `healedReports` để chỉ giữ các báo cáo có `r.check_type !== 'smoke'`.
- **Xác minh**: Chạy `npx tsc --noEmit` ở thư mục frontend và lưu vết vào `06_validation_heal_modal.md`.
