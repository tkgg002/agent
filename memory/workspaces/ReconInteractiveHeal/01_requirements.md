# Yêu cầu chi tiết (Specs) - Hiệu chỉnh Chữa lành tương tác (Rev.3)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

### 1. Phân tách rõ ràng giữa Chữa lành và Thực thi Chữa lành
- **Chữa lành thông thường (Heal)**:
  - Route: `POST /api/reconciliation/heal`
  - Command: `ReconHealCommand` (Type: `"recon.heal"`, NATS subject: `cdc.cmd.recon-heal`)
  - Chức năng: Quét và chữa lành theo khoảng thời gian/cửa sổ. 
  - Giao diện FE: Bấm nút "Chữa lành" sẽ mở `ConfirmDestructiveModal`. Bỏ hoàn toàn các lựa chọn quét Tier 2, khoảng thời gian (startTime, endTime, lookback) trên giao diện này. Người dùng chỉ cần nhập lý do xác nhận.
  - Phía Worker: Giữ nguyên logic ban đầu của `HandleReconHeal` (bao gồm cả việc tự động gọi check Tier 2 `RunTier2`/`RunSegmentBFor` nếu không có sẵn report để tìm missing IDs). Như vậy, `HandleReconHeal` tự động bao hàm Tier 2.

- **Thực thi chữa lành tương tác (Execute Heal)**:
  - Route: `POST /api/reconciliation/execute-heal`
  - Command: `ExecuteHealCommand` (Type: `"execute-heal"`, NATS subject: `cdc.cmd.execute-heal`)
  - Chức năng: Thực thi chữa lành chi tiết cho danh sách report_ids cụ thể và các hành động (mismatched, missing_dest, prune_src) được cấu hình từ UI.
  - Giao diện FE: Thêm nút "Thực thi chữa lành" (Execute Heal) bên cạnh nút "Chữa lành" ở cả màn hình chính và DrillDown panel. Bấm nút này sẽ mở `ExecuteHealModal` để người dùng tích chọn và nhập lý do.
  - Phía Worker: Đăng ký handler `HandleExecuteHeal` (NATS subject `cdc.cmd.execute-heal`) để xử lý luồng thực thi granular này.

### 2. Frontend UI/UX chi tiết
- **ConfirmDestructiveModal.tsx**: Loại bỏ hoàn toàn giao diện chọn chế độ quét & chữa lành (Window / Full-diff) và lookback (hot/cold) cho trường hợp `isHeal = true`. Giao diện lúc này chỉ hiển thị mô tả cảnh báo và ô nhập lý do (tối thiểu 10 ký tự).
- **DataIntegrity.tsx & ReconPipelineGrid.tsx**:
  - Đối với các dòng bị drift/warning: Hiển thị cả 2 nút:
    1. "Chữa lành" (Heal) -> Gọi `openHeal` mở `ConfirmDestructiveModal` đơn giản để chạy `useHealMutation`.
    2. "Thực thi chữa lành" (Execute Heal) -> Gọi `openExecuteHeal` mở `ExecuteHealModal` để chạy `useExecuteHealMutation`.

### 3. API Gateway
- Đăng ký lại route `/api/reconciliation/execute-heal` và command `ExecuteHealCommand`.
- Khôi phục `ReconHealCommand` và route `/api/reconciliation/heal/:table` về nguyên bản.

### 4. Worker
- Khôi phục `HandleReconHeal` trong `recon_handler_run.go` về nguyên bản (giữ nguyên logic check Tier 2 cũ).
- Đăng ký lại `HandleExecuteHeal` trong `recon_execute_heal.go` cho NATS subject `cdc.cmd.execute-heal`.
