# 12 Implementation Plan: Khắc phục Triệt để Lỗi Nhân bản Trùng lặp ID Recon, Lệch Hiển thị Heal & Cơ chế Dừng Khẩn cấp Recon Lớn

**Mã Workspace:** `FixLogTransmuteAndReconTraceId20260909`  
**Ngày cập nhật:** 2026-09-09 16:20:00  

---

## 1. Mục tiêu và Bối cảnh (Đã qua Adversarial Audit)
- **Vấn đề cốt lõi:** 
  1. **Recon Engine:** Mảng `res.StaleIDs` tích lũy qua 672 sub-windows 15 phút không được deduplicate khiến `stale_count` bị nhân bản lên 5,651 (trong khi unique IDs thực tế chỉ có 29).
  2. **Heal Engine:** Bất đối xứng khi so sánh `isFullyHealed` ($29 \ge 5651$ luôn FALSE) khiến report vĩnh viễn kẹt ở `partially_healed`, cùng sự cố dôi dư $+1$ do rò rỉ từ shared `BatchBuffer` (chưa cap trần cho cả Segment A & Segment B).
  3. **Recon Job Blocking (RỦI RO CAO):** Khi trigger 1 tiến trình Recon lớn (7 ngày, 30 ngày hoặc bảng hàng chục triệu bản ghi), worker chạy vòng lặp tuần tự không thể dừng được, chiếm dụng worker làm tất cả các job tiếp theo bị kẹt ở `PENDING`.
  4. **Frontend UI:** Tab "Cần xử lý" hiển thị `{remaining}/{total}` gây hiểu nhầm ngược nghĩa (hiển thị `0/110` khi đã xong 100%, và `5622/5651`).

---

## 2. Kế hoạch Triển khai theo từng File & Chi tiết Line Code

| STT | File & Vị trí | Module | Loại | Chi tiết & Line code cần thực hiện |
| :--- | :--- | :--- | :---: | :--- |
| 1 | `centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go`<br>*(L78, L210, L232, L765, L786)* | Recon Engine | MODIFY | • Bổ sung `DeduplicateAll()` trên `StaleIDsPayload` (khử trùng lặp `MissingFromDest`, `MissingFromSrc`, `Mismatched` trước, sau đó chạy `deduplicatePhantomDrift()`).<br>• Tại đầu vòng lặp chunk ngày (`L210` Seg A & `L765` Seg B): bổ sung `ctx.Err()` check và `isJobCancelled(ctx, jobID)` ngắt vòng lặp ngay khi có lệnh hủy.<br>• Tại cuối `ExecuteSegment` (`L232`) và `executeSegmentB` (`L786`): gọi `res.StaleIDs.DeduplicateAll()`. |
| 2 | `centralized-data-service/internal/service/recon/recon_job_worker.go`<br>*(L233, L296)* | Recon Worker | MODIFY | • Tại `HandleJobEvent` (`L233`): kiểm tra nếu `job.Status == "CANCELLED"` thì skip execution ngay.<br>• Tại Step 5 (`L296`): nếu engine trả về lỗi cancelled thì mark status = `"CANCELLED"`. |
| 3 | `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`<br>*(L245-L270, L370-L425, L195-L235)* | Heal Handler | MODIFY | • Segment A (`L250`): Khóa trần `min(written, len(...))` và auto-realign `rpt.StaleCount`, `rpt.MissingCount`.<br>• Segment B (`L375`): Khóa trần `min(processed, len(...))` và auto-realign `rpt.StaleCount`, `rpt.MissingCount`.<br>• `finalizeReport` (`L205`): lưu đè counts đã chuẩn hóa vào database. |
| 4 | `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`<br>*(L597)* | CMS Repo | MODIFY | • Bổ sung `CancelReconJob(ctx, jobID)`: Update `cdc_system.recon_jobs` sang `CANCELLED`. |
| 5 | `cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go`<br>*(L145)* & `router.go` *(L297)* | CMS API | MODIFY | • Thêm handler `CancelActiveJob(c *fiber.Ctx)` và route `POST /api/reconciliation/jobs/:id/cancel`. |
| 6 | `cdc-cms-web/src/hooks/useReconStatus.ts`<br>*(L520)* | CMS Web | MODIFY | • Thêm hook `useCancelReconJobMutation()`. |
| 7 | `cdc-cms-web/src/components/ReconPipelineGrid.tsx`<br>*(L645)* | CMS Web | MODIFY | • Thêm nút `[Dừng]` (icon `StopOutlined`, Popconfirm) vào từng row trong tab "Progress recon". |
| 8 | `cdc-cms-web/src/components/ExecuteHealModal.tsx`<br>*(L420-L466)* | CMS Web | MODIFY | • Đổi render cột Thiếu, Lệch, Thừa sang Tag `Đã xong ({healed}/{total})` khi `remaining === 0`. |

---

## 3. Kế hoạch Kiểm định (Verification Plan)
1. **Kiểm tra biên dịch:**
   - `cd centralized-data-service && go test ./internal/service/recon/... ./internal/handler/recon/...`
   - `cd cdc-cms-service && go build ./...`
   - `cd cdc-cms-web && npm run build`
2. **Kiểm tra logic nghiệp vụ:**
   - **Recon không bị nhân bản:** chạy reconA 7 ngày, xác nhận `stale_count = 29` (khớp với unique IDs thật).
   - **Dừng Recon khẩn cấp:** trigger recon 30 ngày, bấm `[Dừng]` trên UI $\rightarrow$ job dừng ngay sau chunk ngày hiện tại, chuyển sang `CANCELLED`, job tiếp theo trong queue lập tức được bốc lên chạy.
   - **Heal không bị kẹt & không $+1$:** bấm Heal trên report cũ 5651 $\rightarrow$ heal 29/29, report chuyển sang `healed`, UI hiển thị Tag `Đã xong (29/29)`.
