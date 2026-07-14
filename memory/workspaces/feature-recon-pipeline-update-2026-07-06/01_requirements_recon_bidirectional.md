# Yêu cầu đối soát hai chiều (Bidirectional Reconciliation Check)

Dự án yêu cầu cải tiến logic đối soát `full_diff` ở tầng Tier A (Source -> Shadow) để phát hiện chính xác mọi lệch lạc (drift), bao gồm cả các bản ghi tồn tại ở Shadow nhưng đã bị xóa hoặc không tồn tại ở Source (bản ghi stale/orphan).

## Chi tiết yêu cầu:
1. **Đối soát hai chiều (Bidirectional Reconciliation):**
   - Không chỉ quét các bản ghi thiếu ở Shadow (missing), mà phải phát hiện các bản ghi dư thừa ở Shadow (stale).
   - Hàm `TimeBoundedDiffMissingFromShadow` cần thay đổi signature để trả về cả hai tập hợp ID: `missing` và `stale`.

2. **Cập nhật Báo cáo đối soát (Reconciliation Report):**
   - Trong `recon_check_handler.go`, khi thực hiện `full_diff`, kết quả trả về từ `TimeBoundedDiffMissingFromShadow` phải được map đầy đủ vào báo cáo `ReconciliationReport`:
     - `MissingCount`: số lượng bản ghi missing.
     - `MissingIDs`: danh sách các ID missing (dạng JSON).
     - `StaleCount`: số lượng bản ghi stale.
     - `StaleIDs`: danh sách các ID stale (dạng JSON).
     - `Diff`: tổng số lượng lệch (`len(missing) + len(stale)`).
     - `Status`: nếu `len(missing) > 0` hoặc `len(stale) > 0`, trạng thái phải là `"drift"`.

3. **Cập nhật các cuộc gọi trong hệ thống (Callsite Resolution):**
   - Tất cả các file gọi đến `TimeBoundedDiffMissingFromShadow` (bao gồm `recon_check_handler.go` và `recon_heal_handler.go`) phải được cập nhật tương ứng để tránh lỗi biên dịch.
