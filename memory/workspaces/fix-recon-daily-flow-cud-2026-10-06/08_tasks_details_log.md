# Danh Sách Task Triển Khai (Phase Details Log & C/U/D Consistency)

- [ ] **Task 1: Thêm cột Details Log vào Tab Log Kafka trong `ReconPipelineGrid.tsx`**
  - Thêm cột `Chi tiết log (Details)` vào `Table<ActivityLogEntry>` của Kafka.
  - Parse `details` an toàn qua `parseLogDetails(r.details)`.
  - Hiển thị các tag tóm tắt: `c` (green), `u` (blue), `d` (red), `written`, `buffer_type`.
  - Thêm nút / icon Popover hiển thị toàn bộ JSON `details` có format đẹp và nút Copy.

- [ ] **Task 2: Thêm cột Details Log vào Tab Log Transmute trong `ReconPipelineGrid.tsx`**
  - Thêm cột `Chi tiết log (Details)` vào `Table<ActivityLogEntry>` của Transmute.
  - Parse `details` an toàn qua `parseLogDetails(r.details)`.
  - Hiển thị các tag tóm tắt: `ins` (green), `upd` (blue), `skip` (default), `scanned`.
  - Thêm nút / icon Popover hiển thị toàn bộ JSON `details` có format đẹp và nút Copy.

- [ ] **Task 3: Bổ sung chi tiết kiểm toán vào `HistoryTable` (Tab Nhật ký đối soát 30 phiên gần nhất)**
  - Thêm nút / icon Popover ở cột "Chi tiết" để xem toàn bộ thông số phiên: `source_count`, `dest_count`, `diff`, `missing_count`, `stale_count`, `healed_count`, `recon_start_time`, `recon_end_time`, `error_message`, `trace_id`.

- [ ] **Task 4: Chuẩn hóa phân bổ số dòng vật lý Shadow (`shadow.total = 168`) trong `dailyStats`**
  - Xóa bỏ tình trạng lấy `c = 163` và `u = 23` của Kafka gán thẳng vào Shadow trong khi Total chỉ là 168.
  - Áp dụng công thức phân bổ bảo toàn số dòng vật lý:
    `shadowC = Math.min(c, written); remaining = written - shadowC; shadowU = Math.min(u, remaining); shadowD = remaining - shadowU;`
    Đảm bảo `shadowC + shadowU + shadowD === written`.

- [ ] **Task 5: Loại bỏ hoàn toàn fallback mù "thành công gán hết vào insert" khi thiếu metadata**
  - Xóa bỏ `row.shadow.c += written` và `row.master.inserted += rows` khi `hasExplicitOps === false` và không phải là snapshot.
  - Khi thiếu metadata c/u/d trên các batch cũ (như ngày 2026-10-06 hoặc sáng 2026-10-07):
    + Nếu là snapshot runner: mới được gán 100% vào `c` (vì snapshot là nạp mới).
    + Nếu là stream nhưng log cũ không có `c, u, d`: Phải phản ánh trung thực số lượng `written` vào Total, và phân rã nếu trạm tiếp theo (Master) đã xác định rõ bao nhiêu Insert/Update, hoặc giữ nguyên Total mà không tự tiện gán mù vào `c`.

- [ ] **Task 6: Build & Verify không lỗi**
  - Chạy `npm run build` (`tsc -b && vite build`) trong `cdc-cms-web`.
