# Kế hoạch triển khai chi tiết của AI (AI Implementation Plan) - Phase Split
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal - Phase Split)

### 1. Hiện trạng và Định hướng thay đổi
Hệ thống hiện tại đang tồn tại song song cả nút "Chữa lành" (gọi `/api/reconciliation/heal` để chạy ngầm) và nút "Thực thi chữa lành" (gọi `/api/reconciliation/execute-heal` mở `ExecuteHealModal`).
Yêu cầu mới mong muốn gộp/tách biệt hoàn toàn:
- Luồng đối soát chỉ kiểm tra và ghi log chênh lệch (không tự động heal).
- Luồng chữa lành duy nhất (qua nút "Chữa lành" trên UI) sẽ là chữa lành tương tác: mở modal cho phép chọn report IDs chưa heal và tích chọn chặng sửa đổi cụ thể, gửi request lên endpoint `/api/reconciliation/heal`.
- Endpoint `/api/reconciliation/heal` sẽ gửi `ExecuteHealCommand` (Type: `"execute-heal"`) tới worker để thực hiện xử lý cụ thể, đo đạc thời gian, số lượng và cập nhật các trường granular statistic mới của report.

### 2. Kịch bản Triển khai Kỹ thuật
- **Database**:
  * Chạy file sql migration `088_recon_interactive_heal_stats.sql` trên Postgres để cập nhật schema bảng `cdc_reconciliation_report`.
- **Backend (cdc-cms-service)**:
  * Cấu hình lại handler `TriggerHeal` để unmarshal payload granular mới và gửi `ExecuteHealCommand` qua NATS.
  * Dọn dẹp route và handler `TriggerExecuteHeal` thừa.
- **Worker (centralized-data-service)**:
  * Thêm `masterDB` vào `ReconHandler` thông qua method `WithMasterDB` được gọi ở `server_setup.go`.
  * Trong `recon_execute_heal.go`, cài đặt soft-delete cho cả Segment A & B:
    * Segment A: `UPDATE <shadow_table> SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?)` trên `h.shadowDB`.
    * Segment B: `UPDATE <master_table> SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_gpay_id" IN (?)` trên `h.masterDB`.
  * Ghi nhận và cập nhật đúng thời gian thực hiện, số lượng bản ghi của từng chặng chữa lành vào DB report.
- **Frontend (cdc-cms-web)**:
  * Cập nhật mutation `useHealMutation`, xoá `useExecuteHealMutation`.
  * Di chuyển `ExecuteHealModal.tsx` thành `HealModal.tsx`, đổi tên component thành `HealModal` trỏ vào `useHealMutation`.
  * Sửa `DataIntegrity.tsx` và `ReconPipelineGrid.tsx` để nút "Chữa lành" mở `HealModal` và xoá bỏ hoàn toàn nút "Thực thi chữa lành".

### 3. Phân công vai trò
- **Brain**: Thiết kế giải pháp, tạo tài liệu, giám sát tiến độ. Không chỉnh sửa code trực tiếp.
- **Muscle**: Thực hiện chỉnh sửa mã nguồn, chạy unit test và build check sau khi kế hoạch được phê duyệt.
