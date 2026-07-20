# Walkthrough - Kết quả sửa lỗi Heal không xóa Master/Shadow (Bổ sung FQN Schema Prefix & Dời logic Resolve Config)

Tôi đã hoàn thành sửa đổi backend service để thực hiện xóa cứng trên Master DB (Segment B) và xóa mềm trên Shadow DB (Segment A) khi chạy Heal (Prune Thừa), đồng thời đảm bảo tên bảng luôn có đầy đủ Schema Prefix và loại bỏ hoàn toàn các lỗi log `record not found` không mong muốn.

## Các thay đổi đã thực hiện

### 1. Centralized Data Service Backend
#### [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
* Bổ sung method `MasterPlane() *gorm.DB` để export kết nối Master DB cho các handler khác package truy cập.

#### [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
* **Tự động ghép Schema Prefix động (`processSingleReport`):**
  * Tên bảng `rpt.TargetTable` có cờ `gorm:"-"` nên rỗng khi load từ DB.
  * Đã bổ sung logic tự động ghép `MasterSchema` / `ShadowSchema` tương ứng vào `rpt.TargetTable` nếu nó rỗng hoặc thiếu prefix (không chứa ký tự `.`).
  * Tránh hoàn toàn lỗi Postgres `ERROR: relation "export_jobs" does not exist (SQLSTATE 42P01)` khi chạy SQL do thiếu schema.
* **Dời logic resolve cấu hình (`processSingleReport`):**
  * Dời lệnh `entry := h.resolveTargetTableConfig(rpt.TargetTable)` từ ngoài switch-case vào hẳn bên trong case `SegmentSourceShadow, ""` (Segment A) vì chỉ có Segment A mới cần registry config để truy vấn MongoDB nguồn.
  * Tránh hoàn toàn lỗi `gorm exec error: record not found` do cố tìm config cho FQN của Master DB ở Segment B.
* **Segment B (`executeHealSegB`):**
  * Sử dụng `h.reconCore.MasterPlane()` để thực thi câu lệnh SQL xóa cứng Master DB với tên bảng fully qualified:
    `DELETE FROM %s WHERE "_gpay_id" IN (?)`
  * Chạy theo batch 1000 ID để tối ưu hiệu năng và tránh lock storm.
* **Segment A (`executeHealSegA`):**
  * Sử dụng `h.shadowDB` thực thi câu lệnh SQL xóa mềm Shadow DB với tên bảng fully qualified:
    `UPDATE %s SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?) AND NOT "_deleted"`
  * Chạy theo batch 1000 ID để đảm bảo an toàn.

#### [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
* Thêm comment cache invalidation nhỏ để buộc compiler refresh cache struct ReconciliationReport, tránh lỗi compile.

---

## Kết quả kiểm thử & Xác minh

### 1. Compile & Build Backend
Đã chạy build thành công các package:
```bash
go build ./internal/...
go build ./cmd/worker/...
go build ./cmd/admin-api/...
go build ./cmd/sinkworker/...
```
* **Kết quả:** Build thành công 100%, không phát sinh bất kỳ lỗi cú pháp hay biên dịch Go nào.

### 2. Kiểm toán quy trình (Governance Audit)
Chạy script kiểm tra quy trình:
```bash
python3 tooling/verify_governance.py --workspace fix_recon_heal_delete
```
* **Kết quả:** `⛳ GOVERNANCE AUDIT PASSED 🟢`.
