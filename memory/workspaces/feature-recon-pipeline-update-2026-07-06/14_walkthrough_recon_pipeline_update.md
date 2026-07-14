# Walkthrough - Dọn dẹp Deep Check Payload và Segment UI Routing

Tài liệu này tổng hợp kết quả thực hiện thay đổi mã nguồn và kết quả kiểm nghiệm tĩnh/unit test.

## 1. Các file đã thay đổi (Walkthrough)

### Frontend (`cdc-cms-web`)
* **[ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx):**
    * Thêm prop `initialSegment` để modal biết chính xác segment hiện tại (Segment A hay Segment B) từ hàng được chọn trong bảng, tránh reset segment về rỗng `""`.
    * Sửa signature `onConfirm` để nhận `typeRecon: string` trực tiếp, loại bỏ cờ `deep` boolean.
* **[DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx):**
    * Chuyển `typeRecon` trực tiếp vào `checkTable.mutateAsync` và xóa bỏ `deep` khỏi payload.
    * Truyền `initialSegment={modalPlan.action.record?.segment}` cho `ConfirmDestructiveModal`.
* **[useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts):**
    * Xóa bỏ trường `deep` khỏi kiểu dữ liệu mutation `useCheckTableMutation` và request payload gửi lên API Gateway.

### API Gateway (`cdc-cms-service`)
* **[recon_check.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go):**
    * Xóa trường `Deep` khỏi struct command `ReconCheckCommand` gửi qua NATS.
* **[reconciliation_handler_commands.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go):**
    * Xóa `Deep` khỏi HTTP request body parser trong `TriggerCheck` và `TriggerCheckAll`.

### Core Engine (`centralized-data-service`)
* **[recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go):**
    * Xóa trường `Deep` khỏi struct `reconCheckPayload` nhận từ NATS.
    * Cập nhật logic: biến `isDeep` để điều hướng xử lý Segment B được gán bằng `payload.TypeRecon == TypeReconDeepCheck`.
    * Cập nhật logic validate loại trừ lẫn nhau giữa lookback window và deep check thông qua `payload.TypeRecon == TypeReconDeepCheck`.

---

## 2. Kết quả Xác minh (Verification Results)

1. **Frontend compilation & build:**
    * Chạy lệnh `npm run build` thành công, verify kiểu dữ liệu TypeScript chính xác và không có lỗi biên dịch.
2. **API Gateway tests:**
    * Chạy `go test ./internal/...` cho `cdc-cms-service` thành công 100%.
3. **Core Engine tests:**
    * Chạy `go test ./internal/... ./pkgs/...` cho `centralized-data-service` thành công 100%.
