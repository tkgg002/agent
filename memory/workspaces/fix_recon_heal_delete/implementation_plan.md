# Kế hoạch triển khai - Sửa lỗi Heal không update _deleted = true vào Master/Shadow

Kế hoạch này bổ sung logic soft-delete thực tế khi thực hiện Heal các bản ghi thừa (Prune) cho cả Segment B (Master DB) và Segment A (Shadow DB).

## Proposed Changes

### Centralized Data Service Backend

#### [MODIFY] [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
- Export kết nối `masterPlane` qua method `MasterPlane() *gorm.DB` để các package khác có thể truy cập kết nối Master DB thông qua `ReconCore`.

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- **Hàm `executeHealSegB`:** Sử dụng `h.reconCore.MasterPlane()` chạy SQL `UPDATE` đánh dấu xóa mềm `_deleted = true` trên Master DB cho danh sách ID bị thiếu ở Shadow.
- **Hàm `executeHealSegA`:** Sử dụng `h.shadowDB` chạy SQL `UPDATE` đánh dấu xóa mềm `_deleted = true` trên Shadow DB cho danh sách ID bị thiếu ở Source MongoDB.

---

## Verification Plan

### Automated Tests
Chúng ta chạy lệnh compile & test của service:
```bash
go build ./...
```
Hoặc chạy unit test liên quan:
```bash
go test -v ./internal/handler/recon/...
```

### Manual Verification
Chạy thử heal trên môi trường chạy thực tế và verify bản ghi trong Master DB xem có cập nhật `_deleted = true` hay không.
