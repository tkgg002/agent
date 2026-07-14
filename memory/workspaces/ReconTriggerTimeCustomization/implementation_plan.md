# Kế hoạch Tùy chỉnh Thời gian Đối soát (Lùi 5 Phút & Làm tròn 00s)

Tài liệu này mô tả kế hoạch cấu hình thời gian kết thúc đối soát (`date to` / `endMs`) mặc định lùi lại 5 phút so với hiện tại và làm tròn giây về `00s` ở các chế độ đối soát thủ công (Hot Mode, Cold Lookback, Custom).

---

## Proposed Changes

### 1. Frontend (cdc-cms-web)

#### [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
- Thêm hàm helper `getRoundedEndTime`:
  ```typescript
  const getRoundedEndTime = () => {
    return dayjs().subtract(5, 'minute').second(0).millisecond(0);
  };
  ```
- Cập nhật hàm `handleCheckModeChange` để sử dụng `getRoundedEndTime` làm gốc thay cho `dayjs()`.
- Cập nhật `useEffect` (khi mở modal) khởi tạo `customRange` với `getRoundedEndTime`.
- Cập nhật hàm `handleOk` để khi trigger các chế độ `2h` và `7d`, thời gian `endMs` và `startMs` được tính toán dựa trên `getRoundedEndTime`.

---

## Verification Plan

### Automated Tests
1. Chạy lệnh tsc kiểm tra frontend:
   ```bash
   npx tsc --noEmit
   ```

### Manual Verification
1. Mở modal kích hoạt đối soát thủ công.
2. Chọn "Hot Mode" -> Kiểm tra khoảng thời gian đối soát hiển thị có kết thúc ở phút hiện tại lùi 5 phút và giây là 00 không (ví dụ hiện tại là 11:15 thì kết thúc là 11:10:00).
3. Chọn "Tùy chỉnh khoảng thời gian" -> Kiểm tra thời gian mặc định của DatePicker kết thúc ở phút hiện tại lùi 5 phút, 00 giây.
4. Nhấn Xác nhận, kiểm tra payload gửi lên backend xem `endMs` có đúng giá trị mili-giây tương ứng hay không.
