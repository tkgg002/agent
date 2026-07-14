# Báo Cáo Thay Đổi - Tùy chỉnh Thời gian Đối soát (v1.13)

Báo cáo chi tiết các file đã sửa đổi, số lượng dòng code và tóm tắt thay đổi.

## 1. Danh sách file thay đổi
- **File**: `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx`
- **Đường dẫn**: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx`
- **Số lượng dòng thay đổi**: +4 dòng thêm mới, ~10 dòng logic được cập nhật.

## 2. Chi tiết thay đổi (Overview)

### A. Thêm hàm helper `getRoundedEndTime`
Helper này chịu trách nhiệm:
1. Lấy thời gian hiện tại (`dayjs()`).
2. Lùi lại 5 phút (`.subtract(5, 'minute')`).
3. Làm tròn giây về 0 (`.second(0)`).
4. Làm tròn mili-giây về 0 (`.millisecond(0)`).

```typescript
const getRoundedEndTime = () => {
  return dayjs().subtract(5, 'minute').second(0).millisecond(0);
};
```

### B. Cập nhật `handleCheckModeChange`
Sử dụng `getRoundedEndTime()` làm mốc cuối (`endTime`) thay vì `dayjs()` hiện tại để tránh việc lệch giây/mili-giây và lùi đúng 5 phút:
- Chế độ `2h`: Khoảng thời gian từ `endTime - 2h` đến `endTime`.
- Chế độ `7d`: Khoảng thời gian từ `endTime - 7d` đến `endTime`.
- Chế độ `custom`: Mặc định đặt khoảng thời gian từ `endTime - 30d` đến `endTime`.

### C. Cập nhật hook `useEffect` (khi open modal)
Mặc định khởi tạo `checkMode` là `7d` và gán khoảng thời gian bắt đầu/kết thúc dựa trên `getRoundedEndTime()`.

### D. Cập nhật `handleOk`
Khi trigger đối soát thủ công (`isManualRecon = true`), các giá trị `startMs` và `endMs` được gửi lên thông qua `onConfirm` được tính toán bằng cách sử dụng `getRoundedEndTime()` thay thế cho `dayjs()` cũ, đảm bảo đồng bộ mốc thời gian làm tròn về giây `00s`.

## 3. Xác minh chất lượng
- Đã chạy thành công kiểm tra tĩnh TypeScript: `npx tsc --noEmit` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`.
- Không phát hiện bất kỳ lỗi biên dịch nào.
