# Architectural Decisions: Hide Disabled Master Tables in Data Integrity

## Decision: Frontend-side Filtering
- **Context**: Báo cáo đối soát vẫn tồn tại trong database lịch sử (`cdc_recon_smoke_result`), ngay cả khi master config bị tắt hoặc xóa. Backend API `/api/reconciliation/report` trả về toàn bộ dữ liệu lịch sử đối soát gần nhất này.
- **Option 1**: Lọc ở backend (chỉnh sửa SQL query).
  - *Pros*: Giảm lượng data truyền tải.
  - *Cons*: Backend SQL query trở nên phức tạp, JOIN nhiều bảng. Ngoài ra, chặng Ingest (source -> shadow) của bảng đó có thể vẫn hoạt động, việc loại bỏ hoàn toàn khỏi backend API có thể che giấu dữ liệu đối soát của chặng A (source -> shadow).
- **Option 2**: Lọc ở frontend.
  - *Pros*: Đơn giản, an toàn, dễ thay đổi. Frontend có đầy đủ thông tin về master config và schedules. Giúp giữ backend API clean và không ảnh hưởng đến các service khác cũng dùng API này.
  - *Cons*: Tải thêm một chút data (không đáng kể vì số lượng bảng thường < 500).
- **Decision**: Chọn **Option 2** (Lọc ở frontend) để đảm bảo tính an toàn, dễ bảo trì, và không làm hỏng tính năng đối soát chặng A nếu được yêu cầu hiển thị độc lập sau này.
