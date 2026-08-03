# Báo Cáo Thay Đổi Mã Nguồn (Overview Report) - Reset Debezium Connector Offsets

## Danh Sách Các File Đã Thay Đổi

### 1. Backend (`cdc-cms-service`)

1. **`internal/infra/http/kafka_connect.go`**:
   - **Mô tả thay đổi**: Bổ sung method `DeleteOffsets(ctx context.Context, name string) error` vào struct `KafkaConnectClient`.
   - **Số dòng thay đổi**: +6 dòng code.

2. **`internal/api/source/system_connectors_handler.go`**:
   - **Mô tả thay đổi**: Bổ sung handler method `ResetOffsets(c *fiber.Ctx) error` xử lý validate tên connector và gửi yêu cầu xóa offset tới Kafka Connect client.
   - **Số dòng thay đổi**: +13 dòng code.

3. **`internal/router/router.go`**:
   - **Mô tả thay đổi**: Đăng ký route Destructive `registerDestructive("/v1/system/connectors/:name/offsets", h.Source.SystemConnectors.ResetOffsets)`.
   - **Số dòng thay đổi**: +2 dòng code.

---

### 2. Frontend (`cdc-cms-web`)

4. **`src/pages/SourceConnectors.tsx`**:
   - **Mô tả thay đổi**:
     - Thêm `ClearOutlined` icon import.
     - Mở rộng `MutationOp` type chứa `'resetOffsets'`.
     - Cập nhật `mutationFn` điều hướng tới endpoint `/api/v1/system/connectors/${name}/offsets`.
     - Thêm nút **Xóa Offset** trên cả 2 bảng **Connections** và **Connectors**.
     - Bổ sung cảnh báo Alert Warning và placeholder trong Modal xác nhận thao tác.
   - **Số dòng thay đổi**: +28 dòng code.
