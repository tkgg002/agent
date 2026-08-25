# 10_gap_analysis_gp2_3667.md - Phân tích Lỗ hổng Kiến trúc (Gap Analysis)

## Báo cáo Rà soát Code Smell & Lỗ hổng tiềm ẩn

1. **MongoDB BSON Type Conversion Gap:**
   - Mongo ExtJSON biểu diễn BSON Date dưới dạng `{"$date": "ISO-string"}` hoặc `{"$date": {"$numberLong": "epoch"}}`.
   - Cần đảm bảo Transmuter Engine hỗ trợ parse cả 2 định dạng này sang `TIMESTAMPTZ` mà không gây panics hoặc rớt timestamp.
2. **Indexing Gap trên PostgreSQL:**
   - Dữ liệu Transaction History có thể đạt quy mô hàng chục triệu bản ghi.
   - Cần sử dụng Composite Index `(user_id, created_at DESC)` thay vì Single Column Index đơn lẻ để phục vụ các truy vấn pagination lịch sử của người dùng.
3. **Partition Strategy Consideration:**
   - Với lượng giao dịch lớn trong tương lai, cân nhắc triển khai PostgreSQL Range Partitioning theo `created_at` (Monthly/Yearly Partitions) nếu dung lượng bảng vượt quá 50GB.
