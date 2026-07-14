# Walkthrough: Chữa Lành Đối Soát Tương Tác (Hoàn Tất)

Chúng ta đã hoàn tất việc cấu hình, kiểm thử và phân tách hai luồng Chữa lành trên hệ thống:
1. **Background/Window Heal (`heal`)**: Quét và tự động chữa lành bằng NATS subject `cdc.cmd.recon-heal` (`ReconHealCommand`). Luồng này tự động thực hiện Tier 2 check (`RunTier2`/`RunSegmentBFor`) ở phía backend. Giao diện FE được tối giản hóa chỉ có nút xác nhận và nhập lý do.
2. **Interactive/Granular Heal (`execute-heal`)**: Chữa lành tương tác chi tiết từ các bản ghi unhealed với các checkboxes lựa chọn hành động bằng NATS subject `cdc.cmd.execute-heal` (`ExecuteHealCommand`).

---

## Các Thay Đổi Đã Thực Hiện

### 1. API Gateway (`cdc-cms-service`)
- [recon_async.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_async.go): Định nghĩa lại `ReconHealCommand` gốc và khai báo `ExecuteHealCommand` chứa mảng `report_ids` và các flag checkboxes.
- [reconciliation_handler_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_heal.go): Khôi phục handler `TriggerHeal` nguyên bản.
- [reconciliation_handler_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go): Định nghĩa `TriggerExecuteHeal` tiếp nhận payload và dispatch `ExecuteHealCommand`.
- [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go) & [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go): Khôi phục route `/api/reconciliation/execute-heal` và subject NATS `cdc.cmd.execute-heal`.

### 2. Frontend UI (`cdc-cms-web`)
- [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts): Khai báo lại `useExecuteHealMutation` và khôi phục `useHealMutation`.
- [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx): Ẩn toàn bộ form chọn time-window/lookback/quét đối soát khi `isHeal = true` (chỉ hiển thị lý do xác nhận chữa lành).
- [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx): Sử dụng `useExecuteHealMutation` để dispatch.
- [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx): Render cả 2 nút:
  * Nút **Chữa lành** -> Gọi `openHeal` mở `ConfirmDestructiveModal` tối giản.
  * Nút **Thực thi chữa lành** -> Gọi `openExecuteHeal` mở `ExecuteHealModal`.

### 3. Worker (`centralized-data-service`)
- [recon_handler_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go): Khôi phục handler `HandleReconHeal` nguyên bản (giữ nguyên logic check Tier 2).
- [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go): Đổi tên về `HandleExecuteHeal` xử lý `cdc.cmd.execute-heal`.
- [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go): Đăng ký cả 2 subscriptions cho Worker.

---

## Kết Quả Kiểm Thử & Xác Minh

- **Unit tests**: Toàn bộ unit tests tại `internal/handler/recon` chạy PASS 100%:
  ```bash
  ok  	centralized-data-service/internal/handler/recon	(cached)
  ```
- **Gateway & Worker Build**: `go build` PASS.
- **Frontend Compile**: `npx tsc --noEmit` PASS.
- Nhật ký tiến độ vật lý đã được cập nhật đầy đủ tại [05_progress.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconInteractiveHeal/05_progress.md).
