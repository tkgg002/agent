# Kế Hoạch & Kết Quả Kiểm Thử (Validation Log) - Reset Debezium Connector Offsets

## 1. Kiểm Thử Tự Động (Automated Build Verification)
- **Backend Go Build**:
  - Command: `go build ./cmd/server` tại `cdc-cms-service`
  - Result: **PASS 100%** (Binary biên dịch sạch, không có lỗi cú pháp hoặc missing dependencies).

- **Frontend TypeScript & Vite Build**:
  - Command 1: `npx tsc --noEmit` tại `cdc-cms-web`
  - Result: **PASS 100%** (Không có bất kỳ lỗi TypeScript type mismatch/syntax error).
  - Command 2: `npm run build` tại `cdc-cms-web`
  - Result: **PASS 100%** (Vite production build tạo thành công các bundle chunks trong `dist/`).

## 2. Kiểm Thử Luồng Giao Diện (UI Flow Verification)
- **Tuyến API Endpoint Backend**:
  - Route: `POST /api/v1/system/connectors/:name/offsets`
  - Middleware: `destructiveChain` (Yêu cầu quyền OpsAdmin, Idempotency-Key và Audit log).
  - Kết nối Kafka Connect: Gọi `DELETE /connectors/{name}/offsets`.

- **Giao diện Người Dùng (`http://localhost:5173/sources`)**:
  - Cột `Actions` tại Tab **Connections**: Hiển thị nút **Xóa Offset** (nút màu xám/xanh nhạt kèm icon `<ClearOutlined />`).
  - Cột `Actions` tại Tab **Connectors**: Hiển thị nút **Xóa Offset** bên cạnh các nút `Restart`, `Pause/Resume`, `Delete`.
  - Modal xác nhận: Trình bày tiêu đề `Xóa offset connector: {name}?`, Alert cảnh báo màu vàng về việc cần `Pause` connector trước khi xóa offset, và bắt buộc nhập lý do audit ≥ 10 ký tự.
