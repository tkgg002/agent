# Thiết kế Chi tiết (03_implementation_smoke_boundary)

## Chi tiết triển khai
- Hàm `CountRecentDeletedRows` sử dụng query `SELECT COUNT(*) FROM table WHERE _deleted = true AND timestamp >= lo AND timestamp < hi`.
- Cơ chế trừ bù cửa sổ sử dụng `EstimatedCount` cho source (Mongo) và trừ bù đi số bản ghi phát sinh trong cửa sổ gần đây để tránh ảnh hưởng bởi replication lag.
- Khi có lệch số lượng, HashWindow trên dải tĩnh `[lo, hi)` (với `hi = fromTime`) được gọi để kiểm tra chéo và sửa lại `diff = 0` nếu khớp hoàn toàn.
