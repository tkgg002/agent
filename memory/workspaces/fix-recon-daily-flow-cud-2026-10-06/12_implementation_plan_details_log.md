# Kế Hoạch Triển Khai Chi Tiết: Details Log & Fix C/U/D Consistency

**Thời gian:** 2026-10-07 17:35:00 +07:00  
**Tác giả:** Brain (Architect)

---

## Bước 1: Tiếp nhận phản hồi và thẩm định vấn đề
- Thẩm định 3 yêu cầu cốt lõi của User:
  1. Thêm cột "Details log" vào 3 bảng trong Modal: Nhật ký đối soát (30 phiên), Log Kafka, Log Transmute.
  2. Giải trình ngày 2026-10-06 tại sao có 1 (thực tế 2) update mà không thấy (`5 5 0 0 | 5 5 0 0 | 5 3 2 0`).
  3. Giải trình dòng `186 163 23 0 | 168 163 23 0 | 168 139 29 0`: Tại sao Shadow Total = 168 nhưng c + u = 186; tại sao thành công ở Shadow/Master lại lấy Insert; tại sao Master 139/29 khác với Shadow.

## Bước 2: Thiết lập tài liệu quản trị theo Rule #4 và Rule #13
- Đã khởi tạo `05_progress.md` (append only).
- Đã lập `01_requirements_details_log.md`.
- Đã lập `08_tasks_details_log.md`.
- Đã lập `09_tasks_solution_details_log.md` với code demo đầy đủ.

## Bước 3: Lập phương án kỹ thuật tối ưu duy nhất (Rule #9)
- **Phương án duy nhất:**
  1. Thêm cột `Chi tiết log (Details)` vào 2 bảng `kafkaLog` và `transmuteLog` hiển thị thẻ tóm tắt + nút Popover xem raw JSON details có copy.
  2. Bổ sung nút Popover xem toàn bộ thông số phiên vào cột `Chi tiết` của `HistoryTable`.
  3. Sửa hàm tính toán `dailyStats` trong `ReconPipelineGrid.tsx`:
     - Xóa bỏ triệt để việc gán mù `row.shadow.c += written` và `row.master.inserted += rows`.
     - Phân bổ bảo toàn số dòng vật lý ở trạm Shadow: `shadowC = Math.min(c, written); rem = written - shadowC; shadowU = Math.min(u, rem); shadowD = rem - shadowU;`. Đảm bảo `shadowC + shadowU + shadowD === written`.
  4. Giữ vững kỷ luật Rule #12: Không chạy SQL UPDATE can thiệp vào bảng audit trail bất biến để "làm đẹp" số liệu cũ. Giải thích trung thực nguyên nhân do backend cũ trước 11:15 sáng nay chưa có metadata c/u/d.

## Bước 4: Trình bày Plan và xin phê duyệt (Approval Gate)
- Báo cáo rõ ràng, mạch lạc, trung thực với User kèm giải thích kỹ thuật chuẩn mực.
- Chờ User duyệt trước khi chuyển sang thực thi.
