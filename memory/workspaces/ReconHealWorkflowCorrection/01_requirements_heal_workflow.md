# Yêu cầu - Sửa đổi Luồng và Trạng thái Chữa lành Đối soát (Reconciliation Heal Workflow)

## Bối cảnh
Khi người dùng thực hiện chữa lành (heal) cho một phiên đối soát (Reconciliation Report), hiện tại hệ thống tự động gán `healed_at` và `status = "healed"` cho report đó dù người dùng chỉ chọn sửa 1 trong 3 loại lỗi (Thiếu ở đích, Lệch dữ liệu, Thừa ở đích - Orphan).
Điều này dẫn đến:
1. Phiên đối soát biến mất khỏi tab "Phiên chưa xử lý" (Pending) mặc dù vẫn còn lỗi chưa được sửa đổi.
2. Tab "Phiên đã xử lý" hiển thị không chính xác trạng thái thực tế của dữ liệu toàn vẹn.

## Chi tiết Yêu cầu

### 1. Logic Backend (`cdc-cms-service`)
- Trong `recon_execute_heal_handler.go`, hàm `finalizeReport` cần kiểm tra xem phiên đối soát đã được chữa lành **hoàn toàn** chưa:
  - `HealedMissingDestCount >= MissingCount` AND
  - `HealedMismatchedCount >= StaleCount` AND
  - `PrunedMissingSrcCount >= OrphanCount`
- Nếu **thỏa mãn**:
  - Gán `status = "healed"`
  - Gán `healed_at = NOW()`
- Nếu **chưa thỏa mãn (chữa lành một phần)**:
  - Gán `status = "partially_healed"`
  - Gán `healed_at = NULL` (giúp giữ lại phiên trong danh sách chưa xử lý)
- Cập nhật logic `ReleaseHealClaim` trong `reconciliation_report_repo.go` để tránh trường hợp report bị kẹt ở trạng thái `healing` khi giải phóng claim (nếu `prevStatus` là `healing` thì fallback về `drift`).

### 2. Logic Frontend (`cdc-cms-web`)
- Trong component `ExecuteHealModal.tsx`:
  - Cập nhật cách hiển thị số lượng lỗi trong bảng "Phiên chưa xử lý":
    - Nếu lỗi đã được chữa lành một phần (healed count > 0), hiển thị theo định dạng `Remaining / Original` (Ví dụ: `3/10` cho 3 lỗi còn lại trên tổng số 10 lỗi ban đầu).
    - Nếu chưa được chữa lành lần nào, hiển thị số gốc như cũ.
  - Tự động bật/tắt (enable/disable) các ô checkbox hành động chữa lành dựa trên việc có tồn tại lỗi chưa xử lý thuộc loại đó trong danh sách phiên hay không.
    - Không cho phép check các hành động đã được hoàn tất chữa lành (Remaining = 0).
