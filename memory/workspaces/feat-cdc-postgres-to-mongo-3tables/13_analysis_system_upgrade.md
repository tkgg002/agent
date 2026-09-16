# 13 Analysis: Phân tích Kỹ thuật Nâng cấp Hệ thống CDC PostgreSQL → MongoDB

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Chủ quản**: Brain & Staff Engineer

---

## 1. Phân tích Bản chất Bài toán Denormalization trong CDC

Khi chuyển từ mô hình cơ sở dữ liệu quan hệ (RDBMS chuẩn 3NF: 3 bảng riêng biệt) sang cơ sở dữ liệu tài liệu (Document-oriented NoSQL: 1 document duy nhất), CDC phải đối mặt với **3 mâu thuẫn cốt lõi**:

1. **Mâu thuẫn về Tính Tuần tự (Ordering Anomaly)**:
   - Trong RDBMS, 1 transaction có thể ghi đồng thời cả 3 bảng `orders`, `order_items`, `order_payments`.
   - Khi Debezium đọc WAL, nó phát ra 3 CDC event nối tiếp nhau.
   - Nếu đẩy ra 3 topic khác nhau: Kafka **không có cơ chế liên-topic ordering**. Consumer có thể đọc event của `order_items` trước event của `orders`.
   - **Giải pháp**: Buộc Debezium đẩy cả 3 bảng vào **1 Topic duy nhất** qua SMT `ByLogicalTableRouter` với Partition Key là `order_id`. Khi đó tính tuần tự của transaction được bảo toàn tuyệt đối.

2. **Mâu thuẫn về Tính Độc lập của Document (Document Independence)**:
   - Trong RDBMS, foreign key `order_id` bảo vệ tính toàn vẹn.
   - Trong MongoDB, document `orders` chứa mảng `items`. Nếu không có cơ chế chặn, một event `order_items` đến trễ sau khi `orders` bị xóa sẽ tạo ra một document rác mồ côi (Zombie Document).
   - **Giải pháp**: Cấm triệt để `SetUpsert(true)` trên các bảng con và áp dụng Soft-Delete cho bảng cha.

3. **Mâu thuẫn về Hiệu năng Query Lịch sử (Cartesian Product in Snapshot)**:
   - Khi backfill hàng triệu bản ghi, phép `LEFT JOIN` nhiều bảng 1:N và 1:1 đồng thời sẽ nhân số dòng trung gian theo tích Đề-các ($M \times N$).
   - **Giải pháp**: Tận dụng `LEFT JOIN LATERAL` độc lập kết hợp `jsonb_agg`, PostgreSQL thực thi subquery qua index `order_id` mà không nhân dòng.

---

## 2. Đánh giá Mức độ Tác động tới Hạ tầng Hiện tại
- **Hạ tầng Kafka Connect**: Chỉ bổ sung 1 connector JSON độc lập `cdc-pg-orders-to-mongo`, không sửa đổi các connector hiện có.
- **Hạ tầng Go Service (`centralized-data-service`)**: Thêm 2 file xử lý (`mongo_type_converter.go`, `mongo_aggregator_worker.go`) và mở rộng hàm snapshot trong `snapshot_runner_handler.go`. Toàn bộ các handler cũ phục vụ PostgreSQL DW vẫn nguyên vẹn 100%.
- **Hạ tầng MongoDB**: Chỉ tạo mới hoặc cập nhật vào collection đích `orders` kèm index trên `_id` và `order_code`.
