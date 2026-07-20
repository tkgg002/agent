# Nhật ký Quyết định (04_decisions_smoke_boundary)

## Danh sách quyết định kiến trúc
- **ADR-001: Sử dụng EstimatedCount mặc định cho MongoDB**
  - *Bối cảnh*: `CountDocuments` gây timeout trên production vì quét toàn bộ collection.
  - *Quyết định*: Dùng `EstimatedCount` và đối soát chéo HashWindow trên phạm vi tĩnh khi có lệch số lượng.
- **ADR-002: HashWindow sử dụng mốc trên là fromTime**
  - *Bối cảnh*: Nếu dùng `nowTime`, lag replication vẫn ảnh hưởng đến dải Hash.
  - *Quyết định*: Dùng `fromTime` (now - 120s làm tròn phút) làm mốc trên của dải HashWindow.
