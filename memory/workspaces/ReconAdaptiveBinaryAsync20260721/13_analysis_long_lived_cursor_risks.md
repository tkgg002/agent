# 13 — Đánh Giá Rủi Ro Hạ Tầng: Long-lived Cursor & MVCC Snapshot Pinning

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Cập nhật:** 2026-07-21  
> **Tác giả:** System Architect Audit  
> **Trạng thái:** COLD-BLOODED ARCHITECTURAL RISK EVALUATION  

---

## 1. Phản Tỉnh Về Thái Độ Thẩm Định Kiến Trúc

* **Nghiêm túc rút kinh nghiệm:** Việc đánh giá cảm tính "10/10" hay "Mô hình đỉnh cao" ở lượt trước là một sai lầm thiếu thực tế. Trong kiến trúc hệ thống, **không có giải pháp nào là hoàn hảo 100%**, mọi thiết kế đều là một chuỗi các sự đánh đổi (Trade-offs).

---

## 2. Thẩm Định Chi Tiết 2 Rủi Ro Hạ Tầng

### 💣 Rủi Ro 1: Vỡ Mộng "Đứt Cáp" (Network Instability & Zero Fault Tolerance)
* **Kết luận Thẩm định:** **RẤT THỰC TẾ VÀ DỄ XẢY RA TRONG MÔI TRƯỜNG DISTRIBUTED CLOUD.**
* **Phân tích kỹ thuật:**
  - Việc duy trì một kết nối Socket/Cursor mở liên tục trong 5–10 phút để stream hàng chục triệu bản ghi qua mạng giữa DB Node và App Node cực kỳ dễ bị đứt gãy bởi: *TCP Idle Timeout, Network Transient Blip, Proxy/Load Balancer Timeout, hoặc DB Idle Cursor Timeout*.
  - Do cơ chế tích lũy Hash trên RAM Go là dạng **Transient State (Trạng thái tạm thời)** và **Non-resumable (Không có Checkpoint ngắt giữa chừng)**, nếu đứt kết nối ở $99\%$ tiến độ, toàn bộ dữ liệu trên RAM bị xóa sạch và Job buộc phải Retry lại $100\%$ từ đầu, làm lãng phí toàn bộ tài nguyên I/O đã đọc trước đó.

### 💣 Rủi Ro 2: Chiếm Dụng Snapshot DB (MVCC Transaction Pinning & Storage Bloat)
* **Kết luận Thẩm định:** **CHÍ MẠNG ĐỐI VỚI VẬN HÀNH POSTGRESQL VÀ MONGODB PRODUCTION.**
* **Phân tích kỹ thuật:**
  - **PostgreSQL MVCC (Multi-Version Concurrency Control):** Một câu lệnh `SELECT` giữ Cursor mở kéo dài nhiều phút sẽ giữ chặt mốc `xmin` transaction horizon. Trong suốt thời gian Cursor này hoạt động, tiến trình **Autovacuum bị chặn hoàn toàn**, không thể dọn dẹp các dead tuples sinh ra bởi các lệnh UPDATE/DELETE từ các ứng dụng khác. Hệ quả: Bảng bị **Table Bloat** nghiêm trọng, dung lượng đĩa cứng bị phình to và hiệu năng đọc index bị suy giảm nặng nề.
  - **MongoDB WiredTiger Engine:** Cursor mở lâu giữ một `Read Concern` snapshot cố định. WiredTiger Storage Engine không thể Evict (giải phóng) các dirty pages cũ khỏi RAM cache, dẫn tới **WiredTiger Cache Pressure**, ép MongoDB phải Swapping đĩa cứng và làm nghẽn toàn bộ cluster.
