# Hồ sơ giải pháp kỹ thuật (Technical Solutions) - Hiệu chỉnh Chữa lành tương tác (Rev.3)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal)

### 1. Phân biệt `recon-heal` vs `execute-heal`
- `recon-heal` (`cdc.cmd.recon-heal`): Background/automated window-based scan và heal. Nó tự động bao hàm Tier 2 (tức là tự động gọi `RunTier2`/`RunSegmentBFor` để quét ra missing IDs nếu chưa có sẵn hoặc bị stale, sau đó mới heal).
- `execute-heal` (`cdc.cmd.execute-heal`): Granular interactive execution trực tiếp từ selected report IDs từ UI.

### 2. Khôi phục thiết lập trong Worker (`centralized-data-service`)
- Trong `recon_handler_run.go`, khôi phục hàm `HandleReconHeal` về trạng thái nguyên bản. Khi nhận NATS message, nó sẽ thực hiện quét và tự động chữa lành bằng cách gọi các logic check Tier 2/Segment B khi cần thiết.
- Trong `recon_execute_heal.go`, đổi tên `HandleReconHeal` về lại `HandleExecuteHeal` để xử lý NATS subject `"cdc.cmd.execute-heal"`. Loại bỏ fallback `GetUnhealedReports` vì luồng này thuần túy thực thi từ client.
- Trong `server_setup.go`, khôi phục cả hai subscriptions:
  ```go
  natsClient.Conn.Subscribe("cdc.cmd.recon-heal", reconHandler.HandleReconHeal)
  natsClient.Conn.Subscribe("cdc.cmd.execute-heal", reconHandler.HandleExecuteHeal)
  ```

### 3. Hiệu chỉnh Frontend (`cdc-cms-web`)
- **ConfirmDestructiveModal.tsx**:
  - Khi `isHeal = true`, ẩn toàn bộ khối:
    ```tsx
    {!isCheckTier2 && isHeal && (
      <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 8 }}>
        {/* Chế độ quét & chữa lành */}
      </div>
    )}
    ```
  - Giá trị `mode`, `startTime`, `endTime`, `lookback` khi submit sẽ được gán default/undefined hoặc rỗng, vì backend handler `HandleReconHeal` sẽ tự bao hàm Tier 2.
- **useReconStatus.ts**:
  - Khai báo lại `useExecuteHealMutation` trỏ vào `/api/reconciliation/execute-heal`.
  - Khôi phục `useHealMutation` truyền các tham số ban đầu.
- **DataIntegrity.tsx & ReconPipelineGrid.tsx**:
  - Khôi phục nút "Chữa lành" mở `ConfirmDestructiveModal` với `isHeal = true` để chạy `useHealMutation`.
  - Thêm nút "Thực thi chữa lành" mở `ExecuteHealModal` để chạy `useExecuteHealMutation`.
  - Khai báo lại prop `onExecuteHeal` trong interface của `ReconPipelineGrid` và component `DrillDown` (trong `ReconPipelineGrid.tsx`). Render nút "Thực thi chữa lành" (icon `ThunderboltOutlined`) bên cạnh nút "Chữa lành" ở cả Segment A và Segment B trong DrillDown panel.
  - Loại bỏ hoàn toàn cast hook `{...({ onExecuteHeal: openExecuteHeal } as any)}` trong `DataIntegrity.tsx`, map tường minh prop `onExecuteHeal={openExecuteHeal}`.

### 4. Xác minh unit test `recon_heal_v4_test.go`
Vì `HandleReconHeal` được khôi phục về nguyên bản, toàn bộ test cases cũ kiểm thử mock SQL cho `HandleReconHeal` (bao gồm `RunTier2` lock fail, fresh scan, full diff range) sẽ hoạt động bình thường mà không cần sửa đổi lớn.
