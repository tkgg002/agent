# 08 Tasks: Khắc phục Triệt để Lỗi Nhân bản Trùng lặp ID Recon, Lệch Số lượng Heal và Cơ chế Dừng Khẩn cấp Recon Lớn

**Mã Workspace:** `FixLogTransmuteAndReconTraceId20260909`  
**Ngày cập nhật:** 2026-09-09 16:20:00  

---

## Danh sách Task chi tiết (Đã Audit & Bổ sung Phản biện)

### Giai đoạn 1: Backend Go Engine (`centralized-data-service`)
- [ ] **Task 1.1: Cơ chế Deduplicate All ID chuẩn hóa cho Recon Engine (`recon_stream_bucket_engine.go`)**
  - Bổ sung phương thức `DeduplicateAll()` trên struct `StaleIDsPayload`.
  - Quy trình khử trùng lặp đúng thứ tự:
    1. Deduplicate `MissingFromDest`, `MissingFromSrc`, `Mismatched` trước.
    2. Chạy `deduplicatePhantomDrift()` trên tập đã làm sạch.
    3. Deduplicate lại `Mismatched` để gom các ID mới reclassify từ phantom drift.
  - Kích hoạt `res.StaleIDs.DeduplicateAll()` tại cuối `ExecuteSegment` (Segment A) và `executeSegmentB` (Segment B) trước khi tính `MissingCount` và `RecordDiff`.
- [ ] **Task 1.2: Cơ chế Dừng khẩn cấp (Cancellation Check) cho Tiến trình Recon lớn (`recon_stream_bucket_engine.go` & `recon_job_worker.go`)**
  - Trong vòng lặp chạy theo từng ngày (1 - 30 ngày) của `recon_stream_bucket_engine.go`:
    - Ở mỗi đầu chunk ngày: kiểm tra `ctx.Err() != nil` hoặc `isJobCancelled(ctx, jobID)` (kiểm tra status trong DB `cdc_system.recon_jobs` có chuyển sang `CANCELLED` hay không).
    - Nếu có tín hiệu hủy: ngắt vòng lặp ngay lập tức, trả lỗi `recon job cancelled by user` để nhường tài nguyên cho các job sau.
  - Trong `recon_job_worker.go`:
    - Trước khi bắt đầu job: kiểm tra nếu job đã bị `CANCELLED` thì bỏ qua ngay (skip).
    - Khi nhận lỗi cancelled: cập nhật trạng thái job thành `CANCELLED`.
- [ ] **Task 1.3: Chuẩn hóa đếm và giới hạn trần trong Heal Engine cho CẢ 2 Segment A & B (`recon_execute_heal_handler.go`)**
  - Khóa trần biên (Capping):
    - Segment A: `rpt.HealedMismatchedCount = min(written, len(staleA.Mismatched))`, `rpt.HealedMissingDestCount = min(written, len(missingIDs))`.
    - Segment B: `rpt.HealedMismatchedCount = min(processed, len(staleB.Mismatched))`, `rpt.HealedMissingDestCount = min(processed, len(missingGpayIDs))`.
  - Auto-realign `stale_count`, `missing_count`, `orphan_count` cho các report cũ theo số lượng unique IDs thực tế cho cả Segment A và B.
  - Cập nhật `finalizeReport` ghi đè các giá trị chuẩn hóa vào DB.

### Giai đoạn 2: CMS Backend API (`cdc-cms-service`)
- [ ] **Task 2.1: Bổ sung method `CancelReconJob` trong Repository (`recon_read_repo_gorm.go`)**
  - Cập nhật DB: `UPDATE cdc_system.recon_jobs SET status = 'CANCELLED', error_message = 'Đã hủy bởi người dùng qua CMS' WHERE job_id = ? AND status IN ('PENDING', 'RUNNING')`.
- [ ] **Task 2.2: Thêm endpoint API Hủy Job (`reconciliation_handler_reports.go` & `router.go`)**
  - Đăng ký route: `POST /api/reconciliation/jobs/:id/cancel`.

### Giai đoạn 3: Frontend UX Alignment (`cdc-cms-web`)
- [ ] **Task 3.1: Bổ sung nút [Dừng] tiến trình Recon trong Tab "Progress recon" (`ReconPipelineGrid.tsx` & `useReconStatus.ts`)**
  - Viết hook `useCancelReconJobMutation()`.
  - Hiển thị nút `[Dừng]` (icon `StopOutlined`, Popconfirm) cạnh mỗi job có status `RUNNING` hoặc `PENDING`.
- [ ] **Task 3.2: Chuẩn hóa hiển thị tiến độ cột Thiếu, Lệch, Thừa trong Tab Unhealed (`ExecuteHealModal.tsx`)**
  - Loại bỏ phân số ngược nghĩa `{remaining}/{total}` (như `0/110`).
  - Khi hoàn thành ($100\%$): Hiển thị Tag xanh `Đã xong ({healed}/{total})`.
  - Khi dở dang: Hiển thị `{healed}/{total}` (chú thích `còn lại {remaining}`).
  - Khi chưa heal: Hiển thị số lượng cần xử lý `{total}`.

### Giai đoạn 4: Kiểm định & Verification
- [ ] **Task 4.1:** Compile backend Go `go build ./...` trong `centralized-data-service` và `cdc-cms-service`.
- [ ] **Task 4.2:** Typecheck / Build Frontend `npm run build` trong `cdc-cms-web`.
- [ ] **Task 4.3:** Kiểm tra kịch bản dừng job recon lớn và xác nhận job tiếp theo được bốc ngay.
