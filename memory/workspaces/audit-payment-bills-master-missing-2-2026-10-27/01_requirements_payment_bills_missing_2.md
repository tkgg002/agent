---
tier: T2
suffix: payment_bills_missing_2
workspace: audit-payment-bills-master-missing-2-2026-10-27
mode: standard
created_at: 2026-10-27T10:14:00+07:00
---

# 01. REQUIREMENTS — Audit Vì Sao `payment-bills` (`shadow_traitestpbs.payment_bills` -> `master_payment_bill_service.payment_bills`) Bị Lệch Thiếu 2 Record Ở Master (`56,108` vs `56,108` vs `56,106`)

## 1. Bối cảnh & Yêu cầu của User
User cung cấp thông tin trên giao diện đối soát (lúc `10:07:58 27/10/2026`):
- **Luồng đồng bộ:** `payment-bills`
- **Source:** `[mongodb] traitestpbs / payment-bill-service` — bảng/collection: `payment_bills` (`on`)
- **Shadow:** `shadow_traitestpbs` — bảng: `payment_bills`
- **Master:** `master_payment_bill_service` (`Sync: Realtime`) — bảng: `payment_bills`
- **Số lượng đối soát 3 tầng:**
  - Source: `56,108`
  - Shadow: `56,108`
  - Master: `56,106`
- **Độ lệch:**
  - `ingest: 0`
  - `transmute: -2 (thiếu)`
  - `10:07:58 27/10/2026` — `Lệch , ở master bị thiếu 2 record`

## 2. Mục tiêu Audit & Giải pháp (S1 -> S3, Propose-Only)
1. **Truy vết dữ liệu thực tế & mã nguồn (`VERIFIED` vs `HYPOTHESIS`):**
   - Kiểm tra xem dữ liệu/báo cáo của `payment-bills` (`traitestpbs`, `shadow_traitestpbs.payment_bills`, `master_payment_bill_service.payment_bills`) có trên DB local/dev/testing đang kết nối được hay không, và đọc mã nguồn xử lý:
     - Luồng hiển thị/đếm số lượng `Source / Shadow / Master` (`56,108 / 56,108 / 56,106` và thông báo `ingest: 0`, `transmute: -2 (thiếu)`, `Lệch , ở master bị thiếu 2 record`).
     - Luồng Transmute (`shadow_traitestpbs.payment_bills` -> `master_payment_bill_service.payment_bills`) và Recon/Heal Segment B.
     - Kiểm tra xem vì sao Shadow có `56,108` (khớp Source `56,108`) nhưng Master chỉ có `56,106` (thiếu đúng 2 record): là do 2 bản ghi bị soft-delete (`_deleted = true`), hay bị drop/rớt trong lúc Transmute (lỗi ép kiểu/mapping/PK/cursor `_gpay_id` gap/OCC), hay cách đếm giữa bảng tổng quan (Live Count / `daily_source_counts` / `confluence` / `table-overview`) ở Shadow vs Master có sự khác biệt (ví dụ lọc `_deleted` hoặc trùng khóa `conflict_target` hoặc kẹt cursor `_gpay_id`).
2. **Đề xuất giải pháp tối ưu duy nhất (`09_tasks_solution_payment_bills_missing_2.md` & `13_analysis_payment_bills_missing_2.md`):**
   - Chỉ rõ Root Cause kèm bằng chứng `file:line` và kết quả truy vấn DB thực tế (nếu truy cập được).
   - Qua cổng `DG0` và `DG1/DG2` (`laya_gate.py`).
   - Giữ nghiêm ngặt chế độ **Propose-Only** (dừng chờ User `APPROVE`, tuyệt đối không tự ý sửa code).
