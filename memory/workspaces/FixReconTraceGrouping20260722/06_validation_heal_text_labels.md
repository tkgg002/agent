# 06 Validation: Chuẩn Hoá Text Nhãn Thừa / Thiếu Trên Giao Diện Heal (`ExecuteHealModal.tsx`)

## 1. Mục Đích & Bối Cảnh
Khắc phục lỗi gán nhãn văn bản ngầm bị nhầm lẫn giữa **Thừa** và **Thiếu** trên popup chi tiết và bảng lịch sử Heal ([ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)):

- **Nguyên nhân gốc rễ**:
  1. Trong danh sách ID chi tiết của Popover, phần hiển thị `missingFromSrc` (Orphan Record — dữ liệu có ở Đích/Shadow/Master nhưng KHÔNG có ở Nguồn/Source) bị viết nhầm thành `"Thiếu ở Shadow / Source"`. Bản chất của `missingFromSrc` là bản ghi **Thừa ở Shadow** (đối với Chặng A) và **Thừa ở Master** (đối với Chặng B).
  2. Ở tiêu đề cột bảng lịch sử Heal, tiêu đề cột 2 và 3 hardcode `"Thừa ở Master"` và `"Thiếu ở Master"`, gây sai lệch ngữ nghĩa khi người dùng xem báo cáo của Chặng A (`source_shadow`).

---

## 2. Giải Pháp Chuẩn Hoá Văn Bản

### A. Chi tiết Popover danh sách ID (`ExecuteHealModal.tsx` lines 377-408):
- **`missingFromDest`** (Missing from Dest):
  - Chặng B (`shadow_master`): **`Thiếu ở Master (Missing from Dest):`**
  - Chặng A (`source_shadow`): **`Thiếu ở Shadow (Missing from Dest):`**
- **`missingFromSrc`** (Missing from Src / Orphan):
  - Chặng B (`shadow_master`): **`Thừa ở Master (Missing from Src):`** (Đã sửa từ *"Thiếu ở Shadow"*)
  - Chặng A (`source_shadow`): **`Thừa ở Shadow (Missing from Src):`** (Đã sửa từ *"Thiếu ở Source"*)

### B. Tiêu đề cột bảng lịch sử Heal (`ExecuteHealModal.tsx` lines 581 & 598):
- Cột 2 (Orphan Prune): **`Thừa ở Đích (Missing from Src)`** (Áp dụng chung cho cả Shadow và Master).
- Cột 3 (Missing Heal): **`Thiếu ở Đích (Missing from Dest)`** (Áp dụng chung cho cả Shadow và Master).

---

## 3. Kết Quả Kiểm Thử (Build Verification)
Chạy lệnh build giao diện `cdc-cms-web`:
```bash
$ npm run build
> cdc-cms-web@0.0.0 build
> tsc -b && vite build

✓ 3690 modules transformed.
✓ built in 1.40s
```
- **Result**: `100% PASS`. Biên dịch thành công không phát sinh bất kỳ lỗi TypeScript nào.
