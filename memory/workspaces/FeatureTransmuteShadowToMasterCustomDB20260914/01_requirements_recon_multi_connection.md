# Yêu Cầu Kỹ Thuật: Hỗ Trợ Master Multi-Connection Cho Module Recon & Sửa Lỗi Fallback SQLSTATE 25P02

**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Ngày:** 2026-09-15  
**Tác giả:** Brain (Architect)  

---

## 1. Mục Tiêu
1. **Triệt tiêu lỗi `SQLSTATE 25P02` trong `CountRows`**:
   - Khi query thử `pkColumn` bị fail (ví dụ do column hoặc table không tồn tại), transaction read-only phải được `Rollback()` dứt điểm, sau đó fallback mới mở một transaction mới tinh để chạy `SELECT COUNT(*)`.
   - Trong `scanExact`, phân biệt rõ `kind == "shadow"` (dùng `_gpay_id`) và `kind == "master"` (truyền rỗng `""` để chạy thẳng `SELECT COUNT(*)`, không thử cột nội bộ CDC của Shadow).
2. **Hỗ trợ Master Multi-Connection cho Recon Engine**:
   - `MasterBindingRef` mang thông tin `MasterConnectionKey`.
   - `ListActiveMasterBindings` truy vấn và gán `COALESCE(cr_ms.connection_code, 'default') AS master_connection_key`.
   - `ReconCore` quản lý pool động các `ReconDestAgent` theo `connectionKey` thông qua `ConnectionManager`.
   - Pipeline Recon Smoke (`CheckAllUnified`, `RunTotalOnlyB`, `reconDrillDownCheckB`) định tuyến chính xác đến `ReconDestAgent` của target database tương ứng (`master_2`, `default_master`, v.v.).
3. **Phòng chống va chạm Cache và Dedup Target**:
   - `ScanTarget` và `smokeCountCache` sử dụng `TargetKey` tổ hợp (`kind + ":" + connKey + ":" + rel`) thay vì chỉ dùng `rel` đơn lẻ.

---

## 2. Tiêu Chí Nghiệm Thu (Definition of Done - DoD)
- **G1 (Traceability):** Đáp ứng đầy đủ cả 2 mục tiêu: sửa bug fallback transaction và hỗ trợ dynamic master agent trong Recon.
- **G2 (Red → Green):** Giải thích rõ ràng cơ chế lỗi `SQLSTATE 25P02` và chứng minh code mới ngăn chặn triệt để trạng thái aborted transaction.
- **G3 (Compile & Test):** Chạy `go test ./...` và `go build ./cmd/worker` trong `centralized-data-service` biên dịch thành công 100%.
- **G4 (Edge Cases):**
  - Trường hợp `master_connection_id` là NULL hoặc 'default': Tự động fallback về `RoleDestination` (`default_master`).
  - Trường hợp connection mới chưa có trong cache: Tự động khởi tạo `ReconDestAgent` an toàn qua `ConnectionManager`.
- **G5 (Chống Regression):** Không ảnh hưởng đến luồng kiểm tra Segment A (Source ↔ Shadow).
- **G6 (Không Shadow Files):** Mọi tài liệu và nhật ký tiến độ được lưu vật lý đầy đủ trong Workspace.
