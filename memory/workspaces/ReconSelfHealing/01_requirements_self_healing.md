# Yêu cầu Chi tiết - ReconSelfHealing (Recon V4 P2)

## 1. Yêu cầu Nghiệp vụ
Tự động chữa lành Master bằng cách soft-delete các bản ghi mồ côi (physical orphans - missing ở shadow; logical orphans - marked deleted ở shadow) khi chạy transmuter với danh sách ID xác định (`onlySourceIDs`).

## 2. Ràng buộc Kỹ thuật & An toàn
1. **An toàn về Concurrency**: Cập nhật `_source_ts = NOW()` (epoch milliseconds) để CDC consumer đến sau không ghi đè dữ liệu cũ hơn lên bản ghi đã bị xóa.
2. **Quy mô Batching**: Hỗ trợ thực thi bulk updates hiệu quả với SQL update có bind parameters.
3. **Cơ chế Test**: Viết unit test hoàn chỉnh chạy độc lập sử dụng SQLite in-memory mà không làm ảnh hưởng/gãy cấu trúc PostgreSQL hiện tại.
