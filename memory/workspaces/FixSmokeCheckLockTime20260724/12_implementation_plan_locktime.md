# Kế Hoạch Triển Khai Khắc Phục Lỗi Smoke Check (Audit & Refinement)

## 🎯 Giải Pháp Khắc Phục Triệt Để

### 1. Đếm EXACT Mongo Source (Xóa Bỏ `EstimatedCount`)
- Trong `RunTotalOnlyA` của [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go):
  Thay `rc.sourceAgent.EstimatedCount(...)` bằng `rc.sourceAgent.CountDocuments(...)` (đếm exact 100% bản ghi at runtime).
- Triệt tiêu 100% tình trạng sai số do metadata Mongo ước lượng `2,788,465` vs exact Shadow `2,788,460`.

### 2. Exact Runtime Timestamp Cho `CheckedAt` (Bỏ Mốc `:00` Tròn Phút)
- Gán `CheckedAt: time.Now().UTC()` exact tại thời điểm tạo `SmokeResult`.
- Phản ánh trung thực mốc thời gian thực thi của phiên Smoke Check.

---
## 📋 Các tệp tin sẽ sửa đổi (Target Files)
- `centralized-data-service/internal/service/recon/recon_smoke.go`: Thay EstimatedCount bằng CountDocuments exact và gán exact time.Now().UTC() cho CheckedAt.
