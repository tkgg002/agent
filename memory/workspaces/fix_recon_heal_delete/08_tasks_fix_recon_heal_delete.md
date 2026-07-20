# Danh sách Task thực thi - Sửa lỗi Heal soft-delete Master/Shadow

- [x] Export Master DB connection qua method MasterPlane() trong ReconCore
- [x] Thực thi hard-delete trong executeHealSegB (Segment B Master DB)
- [x] Thực thi soft-delete trong executeHealSegA (Segment A Shadow DB)
- [x] Sửa hàm processSingleReport để gán TargetTable kèm Schema Prefix khi thiếu hoặc rỗng
- [x] Kiểm thử biên dịch backend và kiểm tra quy trình governance (đã chạy go build và phát hiện lỗi cache struct ReconciliationReport cũ của compiler)
- [x] Dời lệnh resolveTargetTableConfig từ ngoài switch-case vào hẳn bên trong case SegmentSourceShadow, "" ở hàm processSingleReport và verify compile



