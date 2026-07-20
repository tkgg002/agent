# Walkthrough: Hiển thị Source/Dest Count trên Modal đối soát

## 1. Kết quả thay đổi
- Đã thêm `source_count` và `dest_count` vào interface `UnhealedReport` trong `useReconStatus.ts`.
- Đã bổ sung 2 cột hiển thị tương ứng vào `reportColumns` của modal `ExecuteHealModal.tsx`.
- Đã kích hoạt cuộn ngang `scroll.x` cho bảng "Phiên chưa xử lý" để UI co giãn tốt.

## 2. Kết quả kiểm chứng
- Đã chạy `npm run build` thành công, không gặp bất kỳ lỗi biên dịch TypeScript hay cú pháp React nào.
