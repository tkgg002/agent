# Giải pháp kỹ thuật: Chuẩn hóa phân loại đối soát (Refactor type_recon)

## 1. Mục tiêu
Thay thế hoàn toàn cách phân loại đối soát dựa trên `tier` (0, 1, 2, 3) chồng chéo bằng `type_recon` tường minh:
- `smoke`: Đối soát nhanh tổng số lượng (count toàn bảng).
- `hash_window`: Đối soát XOR hash theo cửa sổ thời gian (window-based) để phát hiện lệch và tìm ID lệch chi tiết (Segment A).
- `full_diff`: Đối soát một chiều theo khoảng thời gian tùy chọn để tìm bản ghi thiếu ở Shadow DB (Segment A).
- `deep_check`: Đối soát sâu (toàn bảng băm bucket hash ở Segment A, hoặc so khớp chi tiết từng trường dữ liệu ở Segment B).
- `prune`: Dọn dẹp bản ghi mồ côi (orphan records).

---

## 2. Chi tiết thay đổi tại các Component

### A. Centralized Data Service (CDS)
Refactor file [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go):
- Cập nhật struct payload nhận từ NATS: thay `Tier string json:"tier"` bằng `TypeRecon string json:"type_recon"`.
- Tổ chức lại logic phân phối tác vụ dựa trên `type_recon` và `segment`:

```go
type ReconPayload struct {
	TypeRecon string `json:"type_recon"` // "smoke", "hash_window", "full_diff", "deep_check", "prune"
	Table     string `json:"table"`
	Segment   string `json:"segment"`      // "source_shadow", "shadow_master"
	Deep      bool   `json:"deep"`         // Giữ để tương thích ngược cho deep check Segment B
	StartTime *int64 `json:"start_time"`
	EndTime   *int64 `json:"end_time"`
	Lookback  string `json:"lookback"`
}
```

#### Cấu trúc logic xử lý mới:
1. **Nếu `TypeRecon == "prune"`**: Chạy `RunOrphanPrune` hoặc `PruneAllOrphans` (như cũ).
2. **Nếu Segment là `"shadow_master"` (Segment B)**:
   - Gọi `handleReconCheckSegmentB(ctx, msg, payload.Table, isDeep)` với `isDeep = (payload.TypeRecon == "deep_check" || payload.Deep)`.
3. **Nếu Segment là `"source_shadow"` (Segment A)**:
   - **Quét tất cả các bảng (`table == "*"` hoặc `""`)**:
     - Gọi `h.reconCore.CheckAll(ctx)` (mặc định chạy `smoke`/`hash_window` cho toàn bộ hệ thống).
   - **Quét một bảng cụ thể**:
     - Phân giải registry `entry := h.resolveTargetTableConfig(payload.Table)`.
     - Chạy switch-case theo `TypeRecon`:
       - **`"full_diff"`**: Chạy `TimeBoundedDiffMissingFromShadow` (như logic hasTimeRange trước đây).
       - **`"deep_check"`**: Gọi `h.reconCore.RunTier3(ctx, *entry)` (bucket_hash).
       - **`"smoke"`**: Gọi `h.reconCore.RunTier1(ctx, *entry)` (count_windowed).
       - **`"hash_window"` (hoặc mặc định)**: Gọi `h.reconCore.RunTier2(ctx, *entry)`.

---

### B. CDC CMS Service
1. **`recon_check.go`**:
   - Thay trường `Tier string json:"tier"` bằng `TypeRecon string json:"type_recon"`.
   - Cập nhật `Validate()` để kiểm tra `TypeRecon` bắt buộc.
2. **`reconciliation_handler_commands.go`**:
   - API `TriggerCheck` sẽ đọc `type_recon` từ query param (hoặc body): `typeRecon := c.Query("type_recon", "hash_window")`.
   - Khởi tạo `ReconCheckCommand` truyền `TypeRecon: typeRecon` thay thế cho `Tier`.
   - API `TriggerCheckAll` gửi `TypeRecon: "hash_window"` (hoặc `"smoke"`).
   - API `TriggerPrune` gửi `TypeRecon: "prune"`.

---

### C. CMS Web (Frontend)
1. **`useReconStatus.ts`**:
   - Trong `useCheckTableMutation`, thay đổi tham số `tier: string` thành `typeRecon: string`.
   - Endpoint gọi API đổi thành: `/api/reconciliation/check?type_recon=${typeRecon}`.
2. **`DataIntegrity.tsx`**:
   - Cập nhật định nghĩa `ModalAction` và `ModalPlan` để dùng `typeRecon: string` thay cho `tier: string`.
   - Trong `openCheckTable`, thiết lập mặc định `typeRecon: 'hash_window'`.
   - Truyền `isCheckTier2` sang modal xác nhận dựa trên điều kiện: `modalPlan.action.typeRecon === 'hash_window' || modalPlan.action.typeRecon === 'full_diff' || modalPlan.action.typeRecon === 'deep_check'`.
3. **`ConfirmDestructiveModal.tsx`**:
   - Đổi prop `isCheckTier2` thành một tên mang tính chất nghiệp vụ hơn hoặc giữ nguyên nhưng điều khiển hiển thị form cấu hình dựa trên việc đây là một lệnh check cấu hình được (Segment A/B, Lookback/Full/Deep).
   - Khi chọn chế độ:
     - Chế độ `"lookback"` -> gửi `typeRecon = 'hash_window'`.
     - Chế độ `"full_diff"` -> gửi `typeRecon = 'full_diff'`.
     - Chế độ `"deep"` -> gửi `typeRecon = 'deep_check'`.
     - Tự động map giá trị gửi về hàm `onConfirm` để gọi mutation API.
