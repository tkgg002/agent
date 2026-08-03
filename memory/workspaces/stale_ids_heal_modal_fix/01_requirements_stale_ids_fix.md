# Requirements: Fix Stale IDs display logic in ExecuteHealModal

## 1. Context & Problem Statement
- **Vấn đề**: Hiện tại ở `ExecuteHealModal.tsx`, khi render danh sách ID lệch từ phiên đối soát, code Frontend đối với Chặng B (`segment === 'shadow_master'`) cố gắng đọc key `parsedStale.missing_from_shadow` và `parsedStale.missing_from_master`.
- **Thực tế Backend**: Cả 2 chặng (Chặng A: Source → Shadow và Chặng B: Shadow → Master) Backend Go đều ghi ra DB và trả về JSON `stale_ids` với CÙNG 1 STRUCT (`StaleIDsPayload`):
  ```json
  {
    "missing_from_dest": [...],
    "missing_from_src": [...],
    "mismatched": [...]
  }
  ```
- **Hậu quả**: Khi xem modal Heal ở Chặng B, Frontend đọc key không tồn tại (`missing_from_shadow` / `missing_from_master`) nên danh sách ID bị rỗng (`undefined` -> `[]`), không hiển thị được ID thiếu ở Master hay thiếu ở Shadow.

## 2. Requirements & Scope
- **R1**: Sửa hàm `getDiffIDs` và logic render Popover "ID lệch" trong `ExecuteHealModal.tsx` để đọc đúng 3 key chuẩn từ `stale_ids` cho CẢ 2 CHẶNG: `missing_from_dest`, `missing_from_src`, và `mismatched`.
- **R2**: Gán nhãn (Label) linh hoạt theo đúng ngữ nghĩa (semantics) của từng chặng:
  - **Chặng A (`source_shadow`)**:
    - `missing_from_dest` -> "Thiếu ở Shadow (Missing from Dest)"
    - `missing_from_src` -> "Thiếu ở Source (Missing from Src)"
    - `mismatched` -> "Lệch dữ liệu (Mismatched)"
  - **Chặng B (`shadow_master`)**:
    - `missing_from_dest` -> "Thiếu ở Master (Missing from Dest)"
    - `missing_from_src` -> "Thiếu ở Shadow (Missing from Src)"
    - `mismatched` -> "Lệch dữ liệu (Mismatched)"
- **R3**: Vẫn giữ fallback đọc `missing_ids` cũ (nếu có) và map vào `missing_from_dest` đối với phiên legacy.
- **R4**: Đảm bảo nút "Copy tất cả" copy chính xác 100% tập ID chênh lệch không bị rỗng.
