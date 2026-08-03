# Danh Sách Task Chi Tiết - Reset Debezium Connector Offsets

## Task Checklist

- [ ] **Task 1 (Backend - Infra Client)**: Thêm hàm `DeleteOffsets(ctx context.Context, name string) error` trong `internal/infra/http/kafka_connect.go` gọi HTTP `DELETE /connectors/{name}/offsets`.
- [ ] **Task 2 (Backend - Command & Handler)**: Thêm handler `ResetOffsets(c *fiber.Ctx) error` trong `internal/api/source/system_connectors_handler.go` thực hiện xác thực tên connector, gọi client `DeleteOffsets` và ghi log audit.
- [ ] **Task 3 (Backend - Router)**: Đăng ký route `registerDestructive("/v1/system/connectors/:name/offsets", h.Source.SystemConnectors.ResetOffsets)` trong `internal/router/router.go`.
- [ ] **Task 4 (Frontend - API Service)**: Bổ sung method `resetConnectorOffsets(name: string, reason: string)` trong `cdc-cms-web/src/services/api.ts` (hoặc gọi qua cmsApi trong component).
- [ ] **Task 5 (Frontend - SourceConnectors Page UI)**: Mở rộng `MutationOp` type, thêm nút "Xóa Offset" vào cả 2 bảng Connections và Connectors trong `SourceConnectors.tsx` kèm Modal xác nhận và lý do ≥ 10 ký tự.
- [ ] **Task 6 (Verification)**: Test biên dịch Go backend (`go build ./...`) và build Frontend React (`npm run build`).
