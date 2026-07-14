# Phân Tích Kỹ Thuật - Technical Analysis

Tài liệu phân tích tác động kỹ thuật và tính đúng đắn của giải pháp Tùy chỉnh Thời gian Đối soát.

## 1. Tác động của thay đổi thời gian (Time alignment)
Trước đây, các mốc thời gian đối soát nóng (2h) hay lạnh (7d) được tính toán bằng cách lấy thời điểm hiện tại `dayjs()`. Điều này gây ra hai nhược điểm:
1. **Lệch mili-giây**: Thời gian được truyền đi có chứa giây và mili-giây lẻ, gây khó khăn cho việc caching hoặc phân đoạn chính xác ở worker backend.
2. **Sai số do lag**: Từ lúc mở modal, nhập lý do đến khi bấm xác nhận có độ trễ (10s - 1 phút). `dayjs()` lúc mở modal và lúc xác nhận khác nhau.

### Giải pháp tối ưu
Áp dụng `getRoundedEndTime`:
- Luôn lùi lại 5 phút so với thời điểm hiện tại.
- Đặt giây và mili-giây về `00`.
- Đảm bảo tính nhất quán giữa thời gian hiển thị trên RangePicker và thời gian thực tế gửi đi.

## 2. Rà soát chất lượng (Regression & Safety Check)
- **Tác động đến Smoke test**: Luồng smoke test (`isManualRecon = false`) không bị ảnh hưởng vì nó chỉ gửi `typeRecon = 'smoke'`.
- **Tác động đến Deep Check**: Chế độ `deep` (`typeRecon = 'deep_check'`) vẫn đặt `customRange = null` và không truyền `startMs` / `endMs`, hoàn toàn kế thừa logic cũ.
- **Tương thích kiểu dữ liệu**: `startMs` và `endMs` được gửi dưới dạng epoch milliseconds (`valueOf()`), khớp chính xác với kiểu dữ liệu `number | null` được quy định ở định nghĩa prop `onConfirm` (`ConfirmDestructiveModalProps`).

## 3. Kết luận
Thay đổi mang tính cục bộ (chỉ tác động đến logic chuẩn bị tham số thời gian gửi đi của ConfirmDestructiveModal), có tính an toàn cao và không làm ảnh hưởng đến các luồng UI khác hay các service backend.
