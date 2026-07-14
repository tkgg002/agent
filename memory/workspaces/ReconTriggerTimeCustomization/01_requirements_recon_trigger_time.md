# Yêu cầu - Tùy chỉnh Thời gian Đối soát (Hot Mode, Cold Lookback, Custom)

Cấu hình lại thời gian kết thúc đối soát (`date to` / `endMs`) khi kích hoạt đối soát thủ công lùi lại 5 phút so với hiện tại và làm tròn về phút (`YYYY-MM-DD HH:mm:00`).

## Yêu cầu chi tiết
1. **Thời gian kết thúc mặc định và tính toán**:
   - `endTime = dayjs().subtract(5, 'minute').second(0).millisecond(0)`
2. **Các chế độ đối soát (Hot Mode, Cold Lookback, Custom)**:
   - **Hot Mode (2 giờ)**: `date to` = `endTime`, `date from` = `endTime - 2h`.
   - **Cold Lookback (7 ngày)**: `date to` = `endTime`, `date from` = `endTime - 7d`.
   - **Tùy chỉnh khoảng thời gian (Custom)**: Mặc định của DatePicker RangePicker khi chuyển sang hoặc khi modal mở sẽ là `[endTime - 30d, endTime]`.
3. **Thực thi**:
   - Cập nhật file [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx) ở frontend.
