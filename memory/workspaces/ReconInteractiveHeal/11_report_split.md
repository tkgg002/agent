# Báo cáo Thay đổi (Report) - Chữa lành đối soát tương tác (Tách Command)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal - Phase Split)

### 1. Danh sách các tệp tin đã chỉnh sửa

| Repo | File | Số dòng thay đổi | Mô tả chi tiết thay đổi |
| :--- | :--- | :--- | :--- |
| **centralized-data-service** | `internal/handler/recon/recon_handler.go` | ~15 lines | Thêm trường `masterDB *gorm.DB` vào struct `ReconHandler` và khai báo method `WithMasterDB` |
| | `internal/server/server_setup.go` | ~5 lines | Lấy `masterDB` từ database registry và tiêm vào `reconHandler` qua `.WithMasterDB(masterDB)` |
| | `internal/handler/recon/recon_execute_heal.go` | ~60 lines | Implement logic soft-delete ở Segment A ( shadow table update `_deleted = true` trên `h.shadowDB` cho `staleA.MissingFromSrc`) và Segment B ( master table update `_deleted = true` trên `h.masterDB` cho `staleB.OrphanInMaster`). Đo thời gian và số lượng từng chặng chữa lành, ghi nhận granular statistic trước khi update DB. Thêm hai hàm helper `quoteRelation` và `quoteIdent`. |
| **cdc-cms-service** | `internal/api/recon/reconciliation_handler_heal.go` | ~30 lines | Tái cấu trúc handler `TriggerHeal` để parse payload granular mới và gửi command `ExecuteHealCommand` |
| | `internal/api/recon/reconciliation_handler_execute_heal.go` | ~45 lines | Xóa handler `TriggerExecuteHeal` cũ và dọn dẹp các thư viện import không sử dụng |
| | `internal/router/router.go` | ~1 line | Xóa bỏ đăng ký route `/reconciliation/execute-heal` |
| **cdc-cms-web** | `src/hooks/useReconStatus.ts` | ~30 lines | Sửa mutation `useHealMutation` để nhận payload của `ExecuteHealPayload & { reason: string }` trỏ về route `/api/reconciliation/heal`, xóa mutation `useExecuteHealMutation` |
| | `src/components/HealModal.tsx` | 224 lines | Tạo component mới `HealModal` (đổi tên từ `ExecuteHealModal`), gọi `useHealMutation` khi submit |
| | `src/pages/DataIntegrity.tsx` | ~40 lines | Tích hợp `HealModal` thay cho `ExecuteHealModal`. Xoá action kind `heal` của `ConfirmDestructiveModal`, sửa nút "Chữa lành" mở `HealModal` và xoá nút "Thực thi chữa lành". |
| | `src/components/ReconPipelineGrid.tsx` | ~30 lines | Loại bỏ prop `onExecuteHeal` và nút "Thực thi chữa lành" ở cả 2 chặng |

---

### 2. Tóm tắt thay đổi logic chính

#### A. Centralized Data Service (Worker)
- Struct `ReconHandler` được trang bị thêm master database instance (`masterDB`) để thực hiện các câu lệnh thao tác trực tiếp lên CSDL đích.
- Ở hàm `HandleExecuteHeal` (trong `recon_execute_heal.go`):
  * **Segment A (Source ↔ Shadow)**: Khi flag `PruneMissingSrc` bằng `true`, chạy câu lệnh update:
    ```sql
    UPDATE <shadow_table> SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?)
    ```
    trên `h.shadowDB` cho danh sách ID `staleA.MissingFromSrc`.
  * **Segment B (Shadow ↔ Master)**: Khi flag `PruneMissingSrc` bằng `true`, chạy câu lệnh update:
    ```sql
    UPDATE <master_table> SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_gpay_id" IN (?)
    ```
    trên `h.masterDB` cho danh sách ID `staleB.OrphanInMaster`.
  * Ghi nhận chính xác số lượng bản ghi thực hiện chữa lành/soft-delete và thời gian xử lý của mỗi hành động riêng biệt rồi ghi nhận vào báo cáo (`ReconciliationReport`).

#### B. CDC CMS Service (API Gateway)
- Endpoint `POST /api/reconciliation/heal` (gọi handler `TriggerHeal`) giờ đóng vai trò là endpoint duy nhất cho cả chữa lành tự động chặng và chữa lành tương tác granular.
- Route `/api/reconciliation/execute-heal` và handler `TriggerExecuteHeal` liên quan được dọn dẹp sạch sẽ để tránh dư thừa và chồng chéo logic.

#### C. CDC CMS Web (Frontend)
- Mutation `useHealMutation` được tái cấu trúc để truyền payload granular mới tới `/api/reconciliation/heal`.
- Khi người dùng click nút "Chữa lành" trên UI, thay vì hiển thị modal confirm đơn giản để chạy ngầm, hệ thống sẽ mở `HealModal` để người dùng tích chọn các hành động granular (sửa lệch, bổ sung thiếu, prune thừa) cho các phiên chưa được chữa lành. Nút "Thực thi chữa lành" dư thừa đã được gỡ bỏ khỏi bảng điều khiển pipelines.

---

### 3. Kết quả Kiểm thử & Biên dịch (Build Check)
- **centralized-data-service**: Lệnh `go build ./internal/...` chạy thành công không có lỗi compile.
- **cdc-cms-service**: Lệnh `go build ./internal/...` chạy thành công không có lỗi compile.
- **cdc-cms-web**: Chạy lệnh compile check `npx tsc --noEmit` hoàn thành xuất sắc mà không gặp bất kỳ lỗi kiểu dữ liệu (TypeScript type mismatch) nào.
- *Lưu ý*: Các lệnh chạy test bị timeout trong môi trường sandbox do cần phê duyệt tương tác từ xa (permission prompt timeout).
