# Kế hoạch thực thi chi tiết của AI (AI Implementation Plan) - Hiệu chỉnh Chữa lành tương tác (Rev.3)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

### 1. Hiện trạng và Kế hoạch Sửa đổi
- **API Gateway**:
  - Khai báo lại `ExecuteHealCommand` và subject NATS `cdc.cmd.execute-heal`.
  - Khôi phục `ReconHealCommand` gốc.
  - Cấu hình route `POST /api/reconciliation/execute-heal` gọi `TriggerExecuteHeal`.
  - Khôi phục `TriggerHeal` nguyên bản trỏ vào `ReconHealCommand` gốc.
- **Frontend**:
  - Khai báo lại `useExecuteHealMutation` trỏ vào `/api/reconciliation/execute-heal`.
  - Khôi phục `useHealMutation` nguyên bản.
  - Ẩn lựa chọn quét/thời gian ở `ConfirmDestructiveModal` khi `isHeal = true`.
  - Thêm nút "Thực thi chữa lành" mở `ExecuteHealModal` chạy `useExecuteHealMutation`.
  - Giữ nút "Chữa lành" mở `ConfirmDestructiveModal` chạy `useHealMutation`.
- **Worker**:
  - Khôi phục `HandleReconHeal` nguyên bản trong `recon_handler_run.go`.
  - Khôi phục `HandleExecuteHeal` trong `recon_execute_heal.go` xử lý `cdc.cmd.execute-heal`.
  - Cấu hình cả hai subscriptions trong `server_setup.go`.

### 2. Ủy quyền thực thi (Muscle Execution)
- Chuyển giao các tác vụ sửa đổi mã nguồn và kiểm thử cho subagent Muscle (`self`) để thực hiện.
- Xác minh test cases chạy pass 100%.

### 3. Kế hoạch thực thi chi tiết của AI cho Frontend (Rev.4)
- **Bước 1: Chỉnh sửa `ReconPipelineGrid.tsx`**:
  - Thêm import `ThunderboltOutlined` từ `@ant-design/icons`.
  - Cập nhật interface `ReconPipelineGridProps` để định nghĩa thuộc tính `onExecuteHeal?: (record: ReconReport) => void;`.
  - Cập nhật định nghĩa component `ReconPipelineGrid` nhận `onExecuteHeal` và truyền `onExecuteHeal={onExecuteHeal}` xuống thẻ `<DrillDown ... />`.
  - Cập nhật component `DrillDown` nhận `onExecuteHeal` trong props destructuring và kiểu dữ liệu `DrillDownProps`.
  - Thêm nút "Thực thi chữa lành" (chỉ kích hoạt nếu trạng thái là drift, dest_missing, warning) tại Segment A và Segment B trong component `DrillDown`.
- **Bước 2: Chỉnh sửa `DataIntegrity.tsx`**:
  - Loại bỏ phần cast as any `{...({ onExecuteHeal: openExecuteHeal } as any)}` và đổi thành truyền prop tường minh `onExecuteHeal={openExecuteHeal}`.
- **Bước 3: Biên dịch kiểm tra**:
  - Chạy `npx tsc --noEmit` trong thư mục frontend `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` để đảm bảo Frontend không bị lỗi compilation.
