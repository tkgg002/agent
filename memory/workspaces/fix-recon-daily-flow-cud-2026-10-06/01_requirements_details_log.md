# Yêu Cầu Chi Tiết: Details Log, Lịch Sử 2026-10-06 & Độ Lệch C/U/D 186 vs 168

**Thời điểm:** 2026-10-07 17:35:00 +07:00  
**Người yêu cầu:** User

---

## 1. Yêu cầu 1: Lấy thêm Details log ra cho 3 bảng
- **Vị trí:** Modal Pipeline DrillDown.
- **Phạm vi:**
  1. Tab **Nhật ký đối soát (30 phiên gần nhất)** (`HistoryTable`):
     - Hiển thị rõ ràng các chỉ số: `source_count -> dest_count`, `diff`, `missing_count`, `stale_count`, `healed_count`, `recon_start_time`, `recon_end_time`.
     - Thêm cột hoặc nút Popover xem toàn bộ chi tiết json của report.
  2. Tab **Log Kafka** (`kafka-consumer` activity log table):
     - Thêm cột "Chi tiết log (Details)" hiển thị trực quan các thẻ Badge `c`, `u`, `d`, `written`, `buffer_type`.
     - Kèm nút bấm xem toàn bộ chuỗi raw JSON trong Popover có copy.
  3. Tab **Log Transmute** (`transmute` activity log table):
     - Thêm cột "Chi tiết log (Details)" hiển thị trực quan các thẻ Badge `ins`, `upd`, `skip`, `scanned`.
     - Kèm nút bấm xem toàn bộ chuỗi raw JSON trong Popover có copy.

---

## 2. Yêu cầu 2: Giải trình ngày 2026-10-06 tại sao có 1 update mà không thấy (`5 5 0 0 | 5 5 0 0 | 5 3 2 0`)
- Vì sao Kafka và Shadow hiển thị `5 | 5 | 0 | 0` (5 create, 0 update), trong khi Master Transmute ghi nhận `5 | 3 | 2 | 0` (3 insert, 2 update)?
- Đâu là nguyên nhân kỹ thuật và giải pháp xử lý triệt để không cheat DB?

---

## 3. Yêu cầu 3: Độ lệch và logic "thành công ở ghi vào shadow và master lại lấy insert"
- Dòng dữ liệu: `186 163 23 0 | 168 163 23 0 | 168 139 29 0`.
- User chỉ trích:
  1. *"sao ở chỗ ghi vào master nó ko khớp với 2 cái kafka và ghi vào shadow, check lại luôn"*
  2. *"sao 'thành công' ở ghi vào shadow và master lại lấy insert. logic cơ bản mà"*
- Yêu cầu:
  - Giải thích rõ cơ chế: Tại sao Kafka nhận 186 events nhưng Shadow chỉ ghi 168 dòng vật lý và Master ghi 168 dòng vật lý?
  - Xóa bỏ triệt để fallback mù `row.shadow.c += written` và `row.master.inserted += rows` khi thiếu metadata.
  - Chuẩn hóa phân bổ `c, u, d` ở trạm Shadow: Tổng số con (`c + u + d`) BẮT BUỘC phải bằng đúng `Total` (`written = 168`).
