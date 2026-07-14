# Yêu cầu cập nhật logic hiển thị tab đối soát (Healed/Unhealed Reports)

## 1. Bối cảnh
Hiện tại, logic lọc hiển thị tab 'Phiên đã xử lý' (healed reports) và 'Phiên chưa xử lý' (unhealed reports) trong `ExecuteHealModal.tsx` đang chưa chính xác theo logic chữa lành thực tế. Cần cập nhật logic này để phản ánh đúng trạng thái thực tế của các phiên.

## 2. Yêu cầu chi tiết
- Định nghĩa hàm kiểm tra `isReportFullyHealed(r: any): boolean`:
  - Nếu `r.status === 'ok'` -> Trả về `true` (phiên không có lỗi).
  - Nếu `r.healed_at != null || r.status === 'healed'` -> Trả về `true`.
  - Xác định các cờ lỗi ban đầu:
    - `hasMissing = (r.missing_count || 0) > 0`
    - `hasStale = (r.stale_count || 0) > 0`
    - `hasOrphan = (r.orphan_count || 0) > 0`
  - Nếu ban đầu không có lỗi nào (`!hasMissing && !hasStale && !hasOrphan`) -> Trả về `true`.
  - Tính trạng thái đã chữa lành cho từng loại lỗi:
    - `missingOk = !hasMissing || ((r.healed_missing_dest_count || 0) >= r.missing_count)`
    - `staleOk = !hasStale || ((r.healed_mismatched_count || 0) >= r.stale_count)`
    - `orphanOk = !hasOrphan || ((r.pruned_missing_src_count || 0) >= r.orphan_count)`
  - Trả về kết quả: `missingOk && staleOk && orphanOk`

- Lọc dữ liệu hiển thị:
  - `reports` (Phiên chưa xử lý): `(data?.data || []).filter((r: any) => !isReportFullyHealed(r))`
  - `totalUnhealed`: `reports.length` (đồng bộ số liệu tự động)
  - `healedReports` (Phiên đã xử lý): `(historyData?.data || []).filter((r: any) => r.status !== 'ok' && r.status !== 'error' && isReportFullyHealed(r))`

## 3. Xác minh (DoD)
- Không có lỗi biên dịch TypeScript khi chạy `npx tsc --noEmit` tại thư mục `cdc-cms-web`.
- Không sử dụng các lệnh Git làm ô nhiễm lịch sử repo.
