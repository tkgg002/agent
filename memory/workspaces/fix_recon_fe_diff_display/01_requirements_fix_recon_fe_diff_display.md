# Yêu cầu: Sửa lỗi hiển thị chênh lệch đối soát trên CMS FE

## 1. Bối cảnh
Khi đối soát chặng Shadow → Master (Segment B) cho bảng `export_jobs`, DB trả về bản ghi có chênh lệch dữ liệu (cụ thể là `missing_from_shadow` có 1 phần tử), nhưng trên giao diện CMS FE cột **ID lệch** hiển thị `—`, và cột **Thiếu** hiển thị `0`, **Lệch** hiển thị `1`, **Thừa** hiển thị `1`.

## 2. Phân tích nguyên nhân
* **Lý do Thiếu = 0, Lệch = 1, Thừa = 1:**
  * Backend Segment B (`recon_tier_b.go`) định nghĩa:
    * `MissingCount: len(missingFromMaster)` (Thiếu ở Master)
    * `StaleCount: len(mismatchedIDs) + len(missingFromShadow)` (Lệch dữ liệu + Thiếu ở Shadow)
    * `OrphanCount: len(missingFromShadow)` (Thiếu ở Shadow - tức Master thừa)
  * Do đó, khi `missing_from_shadow` có 1 phần tử, `missing_from_master` rỗng và `mismatched` rỗng:
    * `missing_count` = 0 (Thiếu = 0)
    * `stale_count` = 1 (Lệch = 1)
    * `orphan_count` = 1 (Thừa = 1)
    * Đây là logic của BE gộp đếm để tính toán healing, FE chỉ hiển thị lại đúng các số liệu này.
* **Lý do cột ID lệch hiển thị `—`:**
  * Hàm `getDiffIDs` ở file `ExecuteHealModal.tsx` khi parse `stale_ids` của Segment `shadow_master` sử dụng sai key:
    * Đọc `parsed.stale_ids` và `parsed.orphan_in_master` thay vì `parsed.missing_from_master`, `parsed.missing_from_shadow`, và `parsed.mismatched`.
    * Do đó, danh sách ID trả về rỗng `[]` và cột **ID lệch** hiển thị `—`.

## 3. Mục tiêu & Giải pháp
1. Sửa hàm `getDiffIDs` trong `ExecuteHealModal.tsx` để parse đúng cấu trúc JSON `stale_ids` cho cả Segment B (`shadow_master`) và Segment A (`source_shadow`).
2. Nâng cấp hiển thị Popover của cột **ID lệch** trong `ExecuteHealModal.tsx`:
   * Khi click vào icon list / nút xem chi tiết, hiển thị Popover phân tách rõ ràng 3 danh sách ID theo 3 loại:
     * **Thiếu ở Shadow** (hoặc Thiếu ở Source)
     * **Thiếu ở Master** (hoặc Thiếu ở Shadow)
     * **Lệch dữ liệu** (Mismatched)
   * Sử dụng Icon List (`UnorderedListOutlined`) cho nút trigger Popover xem chi tiết thay vì chỉ hiển thị nút xem thông thường nếu danh sách ID lệch quá 2 phần tử, hoặc hiển thị nút xem cho tất cả các bản ghi có chênh lệch.
