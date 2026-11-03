# 09. TASKS SOLUTION — Giải Pháp Khắc Phục Lệch `-2` Record Ở Master Cho `payment-bills` & Chuẩn Hóa Cửa Sổ Đối Soát Thủ Công (`payment_bills_missing_2`)

## 1. Tóm tắt Nguyên nhân Gốc rễ (`Root Cause`)
1. **Hai bản ghi bị thiếu ở `master_payment_bill_service.payment_bills` là `_id = 57362` và `_id = 57363`** (sinh lúc `2026-10-26 09:20:03.99+00` và `09:20:10.269+00`, tức `16:20:03` và `16:20:10` ngày `26/10/2026`).
2. **Cả 2 bản ghi này ban đầu ĐÃ được Transmute thành công sang Master** lúc `09:20:07.351+00` (`cdc_activity_log.id = 132826`) và `09:20:15.298+00` (`cdc_activity_log.id = 132832`), giúp `shadow_active = master_active = 56,006` (`cdc_recon_smoke_result.id = 498596, 498607`). Bảng `master_payment_bill_service.payment_bills_1` hiện vẫn đang có đủ cả `57362` và `57363` (`56,110` dòng).
3. **Vì sao `57362` và `57363` bị mất trên `master_payment_bill_service.payment_bills` từ `16:22` hôm qua đến nay?**
   - Giữa `09:21:44 UTC` và `09:22:40 UTC` (`16:21:44` – `16:22:40` ngày `26/10/2026`), khi xóa thử dữ liệu trên `master_payment_bill_service.payment_bills` để kiểm thử tính năng Recon & Heal trước lúc commit `786db91`, 2 bản ghi mới nhất lúc đó là `57362` và `57363` đã bị xóa (`master_active` giảm từ `56,006` xuống `56,004` tại `cdc_recon_smoke_result.id = 498616`).
   - Nhưng khi bấm **Bắt đầu đối soát** lúc `09:22:46 UTC` (`id = 273`) và `09:23:49 UTC` (`id = 274`), hàm `getRoundedEndTime()` tại `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:11-13` (`dayjs().subtract(5, 'minute').second(0).millisecond(0)`) đã **lùi cứng `recon_end_time` đi 5 phút** (về `09:17:00 UTC` và `09:18:00 UTC`).
   - Do `57362` và `57363` có `lastUpdatedAt` là `09:20:03 UTC` và `09:20:10 UTC` (`> 09:18:00 UTC`), cả 2 đợt Recon `id = 273` và `id = 274` đều **loại 2 bản ghi này ra khỏi cửa sổ quét**, khiến đợt **Execute Heal** lúc `09:24:01 UTC` (`id = 132843`) chỉ khôi phục 5 bản ghi cũ `57339..57343` (`lastUpdatedAt` lúc `08:20..08:30 UTC`) mà bỏ sót `57362, 57363`.
   - Sáng nay lúc `10:07:49 27/10/2026` (`03:07:49 UTC`), đợt Recon `id = 275` (`recon_end_time = 2026-10-27 03:02:00 UTC`) đã quét qua mốc `09:20 UTC` hôm qua và **bắt trúng `missing_ids = ["57362", "57363"]`** (`missing_count = 2, healed_at = NULL`), nhưng chưa được bấm **Execute Heal**, đồng thời `ReconJobWorker.ProcessJob` (`recon_job_worker.go:410`) lại ghi `status = "COMPLETED"` thay vì `"drift"`.

---

## 2. Phương Án Tối Ưu Duy Nhất (Single Best Solution)

### Phần A — Khôi phục ngay 2 bản ghi thiếu (`57362`, `57363`) trên `master_payment_bill_service.payment_bills`
- Báo cáo đối soát `id = 275` trong `cdc_system.cdc_reconciliation_report` đã ghi nhận sẵn `missing_ids = ["57362", "57363"]` (`id_type = "id"`).
- **Thao tác:** Bấm nút **Heal** ở chặng **Transmute (Shadow $\rightarrow$ Master)** của `payment_bills` trên UI (hoặc kích hoạt `execute-heal` cho `report_ids: [275]`), worker sẽ chạy `executeHealSegB` đồng bộ lại đúng 2 bản ghi `57362, 57363` từ `shadow_traitestpbs.payment_bills` sang `master_payment_bill_service.payment_bills`, đưa số lượng 3 trạm khớp tuyệt đối (`56,110 = 56,110 = 56,110`, `transmute: 0`).

### Phần B — Sửa dứt điểm 2 lỗi logic trong Code (Khi User `APPROVE`)
1. **Tại `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx:11-13`:**
   - Bỏ việc lùi cứng 5 phút (`subtract(5, 'minute')`) và làm tròn giây về 0 trong `getRoundedEndTime()` khi mở/chạy **Manual Recon**, trả về `dayjs()` (thời điểm hiện tại) để khi người vận hành vừa kiểm tra/thao tác trên DB và bấm **Bắt đầu đối soát** (`2h` / `7d`), cửa sổ đối soát quét đến đúng thời điểm bấm nút, không bỏ lọt các bản ghi mới trong 5 phút gần nhất.
2. **Tại `centralized-data-service/internal/service/recon/recon_job_worker.go:383-415`:**
   - Tách riêng biến `reportStatus` cho bảng `cdc_system.cdc_reconciliation_report` (khác với `status = "COMPLETED" | "FAILED"` của bảng `cdc_system.recon_jobs`):
     - Nếu `engineErr != nil` $\rightarrow$ `reportStatus = "error"`
     - Nếu `diffVal != 0 || missingCount > 0 || staleCount > 0 || orphanCount > 0` $\rightarrow$ `reportStatus = "drift"`
     - Ngược lại $\rightarrow$ `reportStatus = "ok"`

---

## 3. Code Demo

### 3.1. File 1: `cdc-cms-web/src/components/ConfirmDestructiveModal.tsx` (`L11-L13`)
```tsx
// TRƯỚC KHI SỬA:
const getRoundedEndTime = () => {
  return dayjs().subtract(5, 'minute').second(0).millisecond(0);
};

// SAU KHI SỬA:
const getRoundedEndTime = () => {
  return dayjs();
};
```

### 3.2. File 2: `centralized-data-service/internal/service/recon/recon_job_worker.go` (`L383-L415`)
```go
		checkTypeVal := "reconA"
		if reqSegment == "shadow_master" {
			checkTypeVal = "reconB"
		}

		reportStatus := "ok"
		if engineErr != nil {
			reportStatus = "error"
		} else if diffVal != 0 || missingCount > 0 || staleCount > 0 || orphanCount > 0 {
			reportStatus = "drift"
		}

		report := &modelrecon.ReconciliationReport{
			// Identity — đúng như stampA: shadow_schema + shadow_table định danh pipeline
			ShadowSchema: entry.ShadowSchema,
			ShadowTable:  entry.TargetTable,
			RunID:        event.JobID,
			// Source info
			SourceDB:    entry.SourceDB,
			SourceTable: entry.SourceTable,
			SourceType:  entry.SourceType,
			SourceHost:  extractHost(entry.SourceURL),
			// Counts — SourceCount nil khi engine lỗi (phân biệt 0 thật vs lỗi)
			SourceCount:  srcCount,
			DestCount:    dstCount,
			Diff:         diffVal,
			MissingCount: missingCount,
			MissingIDs:   missingRaw,
			StaleCount:   staleCount,
			StaleIDs:     staleRaw,
			OrphanCount:  orphanCount,
			// Check metadata
			Segment:        reqSegment,
			CheckType:      checkTypeVal,
			Status:         reportStatus,
			DurationMs:     &durationMs,
			CheckedAt:      time.Now().UTC(),
			ReconStartTime: &reconStart,
			ReconEndTime:   &reconEnd,
		}
```
