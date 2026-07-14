# Kế hoạch triển khai (Plan) - Hiệu chỉnh Chữa lành tương tác (Rev.3)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

### 1. Phân tách vai trò
- **Brain**: Thiết kế luồng, viết đặc tả, giám sát tiến độ. Không chỉnh sửa code trực tiếp.
- **Muscle**: Triển khai code và test theo đặc tả.

### 2. Các bước triển khai kỹ thuật
1. **Khôi phục cấu trúc API Gateway**:
   - Khai báo lại `ExecuteHealCommand` trong `recon_async.go`.
   - Khôi phục `ReconHealCommand` gốc.
   - Đăng ký lại route `/reconciliation/execute-heal` và subject NATS `cdc.cmd.execute-heal`.
   - Khôi phục `TriggerHeal` nguyên bản và cài đặt `TriggerExecuteHeal`.
2. **Khôi phục cấu trúc Frontend**:
   - Khai báo lại `useExecuteHealMutation` và khôi phục `useHealMutation`.
   - Cập nhật `ConfirmDestructiveModal.tsx` để ẩn các checkbox quét Tier 2/thời gian khi `isHeal = true`.
   - Thêm nút "Thực thi chữa lành" bên cạnh "Chữa lành" trong `DataIntegrity.tsx` và `ReconPipelineGrid.tsx` (DrillDown).
   - Bản đồ hóa nút "Thực thi chữa lành" mở `ExecuteHealModal` chạy `useExecuteHealMutation`.
   - Bản đồ hóa nút "Chữa lành" mở `ConfirmDestructiveModal` đơn giản chạy `useHealMutation`.
3. **Hiệu chỉnh Worker**:
   - Khôi phục `HandleReconHeal` trong `recon_handler_run.go` về nguyên bản (giữ nguyên logic check Tier 2).
   - Đăng ký lại subscription `"cdc.cmd.execute-heal"` gọi `HandleExecuteHeal` trong `recon_execute_heal.go`.
   - Kiểm tra `go test` trên worker để đảm bảo các test case cũ pass 100% (vì `HandleReconHeal` đã được khôi phục về nguyên bản).
