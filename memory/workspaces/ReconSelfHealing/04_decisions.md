# Architecture Decisions - ReconSelfHealing

Tài liệu này ghi nhận các quyết định kiến trúc quan trọng được đưa ra trong quá trình thiết kế và thực thi tính năng tự phục hồi dữ liệu đối soát.

## ADR 1: Chốt chặn batchSize và Ràng buộc logic Orphan Master trong Transmuter

### Bối cảnh
Khi transmuter nhận tham số `onlySourceIDs` để thực hiện transmute/heal khoanh vùng, hệ thống sử dụng cơ chế phân trang (pagination) để truy vấn shadow rows từ DB.
Trong thiết kế ban đầu, logic dọn dẹp orphan master chạy trên mỗi batch. Khi trang cuối cùng được fetch và không trả về shadow rows nào, logic so khớp sẽ coi toàn bộ các ID trong `onlySourceIDs` chưa xuất hiện ở batch hiện tại là orphan và thực hiện soft-delete oan dữ liệu.

### Quyết định
1. **Chốt chặn an toàn (Safety Gate)**: Bổ sung validate `len(onlySourceIDs) > t.batchSize` ngay đầu hàm `Run` của Transmuter để ngăn ngừa xử lý lô ID quá lớn vượt quá kích thước 1 batch.
2. **Ràng buộc phân trang (Pagination Gate)**: Ràng buộc logic so khớp và dọn dẹp Orphan Master chỉ được chạy ở batch đầu tiên (`lastGpayID == 0`). Vì lô ID luôn được giới hạn nhỏ hơn `batchSize`, toàn bộ shadow rows tồn tại liên quan chắc chắn được fetch đầy đủ ở batch đầu tiên này. Các batch sau (nếu có) sẽ bỏ qua logic dọn dẹp này để triệt tiêu lỗi xóa oan dữ liệu.

### Hệ quả
- Bảo toàn tính toàn vẹn dữ liệu Master.
- Cho phép GORM phân trang shadow rows tự nhiên.
