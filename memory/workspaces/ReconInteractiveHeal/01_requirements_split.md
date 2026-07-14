# Yêu cầu chi tiết (Specs) - Tách biệt Đối soát & Thực thi (Cập nhật Tách Command)
## Dự án: Chữa lành đối soát tương tác (Recon Interactive Heal - Phase Split)

### 1. Phân tách hoàn toàn Đối soát và Thực thi
- **Luồng Đối soát (Reconciliation / Check)**:
  * Chỉ chạy các lệnh check (như `ReconCheckCommand` hoặc `ReconCommand`) để đối soát dữ liệu và ghi nhận báo cáo lệch vào `cdc_system.cdc_reconciliation_report`.
  * Tuyệt đối không tự động kích hoạt heal trong luồng đối soát này.
- **Luồng Thực thi Chữa lành (Execute Heal)**:
  * Chỉ chạy `ExecuteHealCommand` mới. Command này hoàn toàn không chứa bất kỳ logic đối soát (`RunTier2`) nào bên trong.
  * Chỉ nhận danh sách các ID của report cũ (`report_ids`), trích xuất IDs lệch từ database (trường `stale_ids` và `missing_ids` JSONB) và thực thi ghi đè (heal) hoặc soft-delete (prune) theo checkboxes đã chọn.

### 2. Yêu cầu API Gateway
- Tái cấu trúc endpoint `POST /api/reconciliation/heal` (gọi handler `TriggerHeal`):
  * Nhận payload body bao gồm:
    * `table` (string): tên bảng shadow/master cần heal.
    * `segment` (string, optional): chặng dữ liệu.
    * `report_ids` ([]uint64): danh sách ID report chưa heal cần xử lý.
    * `heal_mismatched` (bool): sửa lệch thuộc tính.
    * `heal_missing_dest` (bool): bổ sung thiếu ở đích.
    * `prune_missing_src` (bool): dọn dẹp thừa ở đích (soft-delete).
    * `reason` (string): lý do heal (lưu audit log).
  * Map payload này và dispatch command `ExecuteHealCommand` gửi qua NATS.
- Xóa handler `TriggerExecuteHeal` và route `/api/reconciliation/execute-heal` dư thừa.
- Giữ nguyên endpoint `/api/reconciliation/report/:table/unhealed` để lấy danh sách các report chưa heal.

### 3. Yêu cầu Worker
- Subscribe subject NATS `"cdc.cmd.execute-heal"` (hoặc binding tương tự cho `ExecuteHealCommand`) trỏ vào handler `HandleExecuteHeal`.
- Trong `HandleExecuteHeal`, thực thi logic chữa lành theo ID report mà không chạy lại đối soát (`RunTier2`):
  * Lặp qua mảng `ReportIDs`, load report tương ứng từ table `cdc_reconciliation_report`.
  * Parse danh sách IDs bị lệch trong report:
    * Segment A (`source_shadow`):
      * Mismatched: `stale_ids.mismatched`
      * Missing from dest: `missing_ids` (mảng phẳng)
      * Missing from src: `stale_ids.missing_from_src`
    * Segment B (`shadow_master`):
      * Mismatched: `stale_ids.stale_ids`
      * Missing from dest: `missing_ids` (mảng phẳng)
      * Missing from src: `stale_ids.orphan_in_master`
  * **Thực thi Granular**:
    * Sửa mismatched & bổ sung thiếu ở đích: gọi `FetchAndWriteByIDs` (Segment A) hoặc gửi transmute chunked (Segment B).
    * Soft-delete chênh lệch thừa ở đích (prune):
      * Segment A: Chạy query update `_deleted = true` trên `h.shadowDB` cho `staleA.MissingFromSrc`.
      * Segment B: Chạy query update `_deleted = true` trên `h.masterDB` cho `staleB.OrphanInMaster`.
  * **Đo đạc thống kê**:
    * Đo thời gian thực hiện (duration) và đếm số lượng bản ghi thực tế được xử lý của từng hành động riêng biệt.
    * Cập nhật các cột thống kê mới tương ứng: `healed_mismatched_count`, `healed_mismatched_duration_ms`, `healed_missing_dest_count`, `healed_missing_dest_duration_ms`, `pruned_missing_src_count`, `pruned_missing_src_duration_ms` của report đó.
    * Đánh dấu report `healed_at = time.Now()` và `status = "healed"`.

### 4. Yêu cầu Frontend UI/UX
- Cập nhật mutation `useHealMutation` để nhận payload dạng mới. Xóa mutation `useExecuteHealMutation`.
- Đổi tên `ExecuteHealModal.tsx` thành `HealModal.tsx`, cập nhật component thành `HealModal` gọi `useHealMutation` khi submit.
- Cập nhật `DataIntegrity.tsx`:
  * Click nút "Chữa lành" sẽ hiển thị `HealModal`.
  * Loại bỏ hoàn toàn nút "Thực thi chữa lành" (Execute Heal).
- Cập nhật `ReconPipelineGrid.tsx`:
  * Loại bỏ prop `onExecuteHeal`.
  * Loại bỏ việc hiển thị nút "Thực thi chữa lành" ở cả 2 chặng.
