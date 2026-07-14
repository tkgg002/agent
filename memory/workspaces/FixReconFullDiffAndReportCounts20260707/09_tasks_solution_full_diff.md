# Technical Solution: Unified Time Range Check (Remove Hot/Cold & Full Diff), Report Counts & Lookback Smoke Check

Tài liệu thiết kế chi tiết để thống nhất logic thời gian đối soát: Sử dụng duy nhất 1 quy tắc `WithReconTimeRange`, loại bỏ hoàn toàn các option lookback `hot`/`cold` ở backend và xóa bỏ check type `full_diff` khỏi toàn hệ thống.

---

## 1. Cơ sở dữ liệu (Migration)
Tạo file migration SQL mới tại `cdc-cms-service`:
`cdc-cms-service/migrations/schema/recon_dlq/090_recon_smoke_diff_time.sql`

```sql
-- 090_recon_smoke_diff_time.sql — Add diff_time column to cdc_recon_smoke_result
ALTER TABLE cdc_system.cdc_recon_smoke_result
  ADD COLUMN IF NOT EXISTS diff_time JSONB;
```

---

## 2. Thay đổi Struct Model `SmokeResult`
Thêm trường `DiffTime json.RawMessage` vào struct `SmokeResult` ở cả 2 service:

### A. Tại `centralized-data-service/internal/model/recon/recon_smoke_model.go`
```go
package recon

import (
	"encoding/json"
	"time"
)

type SmokeResult struct {
	// ... các trường cũ ...
	DiffTime     json.RawMessage `gorm:"column:diff_time;type:jsonb" json:"diff_time,omitempty"`
	Status       string          `gorm:"column:status;not null" json:"status"`
	ErrorMessage *string         `gorm:"column:error_message" json:"error_message"`
	DurationMs   *int            `gorm:"column:duration_ms" json:"duration_ms"`
	CheckedAt    time.Time       `gorm:"column:checked_at;not null" json:"checked_at"`
}
```

### B. Tại `cdc-cms-service/internal/model/recon_smoke.go`
Tương tự, import `"encoding/json"` và thêm cột:
```go
type SmokeResult struct {
	// ... các trường cũ ...
	DiffTime     json.RawMessage `gorm:"column:diff_time;type:jsonb" json:"diff_time,omitempty"`
	Status       string          `gorm:"column:status;not null" json:"status"`
	// ...
}
```

---

## 3. Thay đổi tại `internal/service/recon/`

### A. Dọn dẹp `recon_models.go`
Xóa bỏ các struct keys và functions liên quan đến `manualLookbackKey` và `coldLookbackKey` (chỉ giữ lại `WithReconTimeRange` và `GetReconTimeRange`).

### B. Cập nhật `effectiveLookback` tại `recon_engine.go`
Đơn giản hóa hàm `effectiveLookback` chỉ sử dụng cấu hình mặc định (vẫn giữ để dùng cho các smoke check):
```go
func (rc *ReconCore) effectiveLookback(ctx context.Context) time.Duration {
	if rc.cfg.WindowLookback > 0 {
		return rc.cfg.WindowLookback
	}
	return 2 * time.Hour
}
```

### C. Tính toán thông số đếm tại `recon_tier_a.go`
Cập nhật `RunHashWindowCheck` để lưu `totalSrc` / `totalDst` và query count của 2 trạm:
```go
	var missingFromDest []string
	var missingFromSrc []string
	var mismatchedFromDest []string
	var driftedWindows int
	var totalSrc, totalDst int64

	for _, w := range windows {
		// ...
		handle.docsScanned += srcRes.Count + dstRes.Count
		totalSrc += srcRes.Count
		totalDst += dstRes.Count
        // ...
	}
    // ...
	var srcEst int64
	if exact, err := rc.sourceAgent.CountDocuments(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable); err == nil {
		srcEst = exact
	}
	var dstTotal int64
	_ = rc.shadowPlane.Table(entry.QualifiedTarget()).Where(`NOT "_deleted"`).Count(&dstTotal).Error

	duration := int(time.Since(handle.started).Milliseconds())
	report := &recon.ReconciliationReport{
		TargetTable:      entry.QualifiedTarget(),
		SourceDB:         entry.SourceDB,
		SourceCount:      &totalSrc,
		DestCount:        totalDst,
		Diff:             totalSrc - totalDst,
		TotalSourceCount: &srcEst,
		TotalDestCount:   &dstTotal,
		MissingCount:     len(missingFromDest),
		MissingIDs:       missingJSON,
		StaleCount:       len(mismatchedFromDest) + len(missingFromSrc),
		StaleIDs:         staleJSON,
		OrphanCount:      len(missingFromSrc),
		CheckType:        "hash_window",
		Status:       statusStr,
		Tier:         2,
		DurationMs:       &duration,
		CheckedAt:        time.Now().UTC(),
	}
```

### D. Xây dựng lookback check ghi nhận giờ bị lệch vào `recon_smoke.go`
Import `"encoding/json"` trong `recon_smoke.go`.
Thêm các hàm `runLookbackCheckA` và `runLookbackCheckB`, và gọi chúng khi có drift:
```go
func (rc *ReconCore) runLookbackCheckA(ctx context.Context, entry source.TableRegistry) []string {
	lo, hi, _, srcTS, dstTS, err := rc.pickScanRangeWithLag(ctx, entry)
	if err != nil {
		rc.logger.Warn("smoke lookback check A: pick scan range failed", zap.Error(err))
		return nil
	}

	srcBuckets, err := rc.sourceAgent.BucketCounts(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, srcTS, lo, hi)
	if err != nil {
		return nil
	}
	dstBuckets, err := rc.destAgent.BucketCounts(ctx, entry.QualifiedTarget(), entry.PrimaryKeyField, dstTS, lo, hi)
	if err != nil {
		return nil
	}

	var driftedTimes []string
	keys := make(map[int64]struct{}, len(srcBuckets)+len(dstBuckets))
	for k := range srcBuckets {
		keys[k] = struct{}{}
	}
	for k := range dstBuckets {
		keys[k] = struct{}{}
	}
	for k := range keys {
		s, d := srcBuckets[k], dstBuckets[k].Count
		if abs(s-d) > 0 {
			driftedTimes = append(driftedTimes, time.UnixMilli(k).UTC().Format(time.RFC3339))
		}
	}
	return driftedTimes
}

func (rc *ReconCore) runLookbackCheckB(ctx context.Context, ref MasterBindingRef) []string {
	upper := time.Now().UTC()
	lower := upper.Add(-7 * 24 * time.Hour) // 7 days lookback default cho smoke

	shadowRel := ref.ShadowRel()
	masterRel := ref.MasterRel()

	shBuckets, errS := rc.destAgent.BucketCounts(ctx, shadowRel, segBPKColumn, "_source_ts", lower, upper)
	if errS != nil {
		return nil
	}
	msBuckets, errM := rc.masterAgent.BucketCounts(ctx, masterRel, segBPKColumn, "_source_ts", lower, upper)
	if errM != nil {
		return nil
	}

	var driftedTimes []string
	keys := make(map[int64]struct{}, len(shBuckets)+len(msBuckets))
	for k := range shBuckets {
		keys[k] = struct{}{}
	}
	for k := range msBuckets {
		keys[k] = struct{}{}
	}
	for k := range keys {
		s := shBuckets[k].Count
		d := msBuckets[k].Count
		if abs(s-d) > 0 {
			driftedTimes = append(driftedTimes, time.UnixMilli(k).UTC().Format(time.RFC3339))
		}
	}
	return driftedTimes
}
```
Tại `RunTotalOnlyA` và `RunTotalOnlyB`, khi phát hiện `diff != 0`:
```go
	diff := srcEst - dstActive
	statusStr := "ok"
	var diffTimeJSON []byte
	if diff != 0 {
		statusStr = "drift"
		driftTimes := rc.runLookbackCheckA(ctx, entry)
		if len(driftTimes) > 0 {
			diffTimeJSON, _ = json.Marshal(driftTimes)
		}
	}
    // ...
	result := &recon.SmokeResult{
        // ... các trường cũ ...
		Diff:         diff,
		Status:       statusStr,
		DiffTime:     json.RawMessage(diffTimeJSON),
		DurationMs:   &dur,
		CheckedAt:    time.Now().UTC(),
	}
```

---

## 4. Thay đổi tại `internal/handler/recon/`

### A. Dọn dẹp `recon_base_handler.go`
- Xóa bỏ hằng số `TypeReconFullDiff = "full_diff"`.
- Xóa bỏ hằng số `LookbackHot = "hot"` và `LookbackCold = "cold"`.

### B. Cập nhật `recon_check_handler.go`
1. Đơn giản hóa `validateAndEnrichContext` để bắt buộc `StartTime` và `EndTime` đối với `TypeReconHashWindow`, thực hiện validator tối đa 30 ngày:
```go
func (h *CheckHandler) validateAndEnrichContext(ctx context.Context, payload *reconCheckPayload) (context.Context, error) {
	if payload.TypeRecon == TypeReconHashWindow {
		if payload.StartTime == nil || payload.EndTime == nil {
			return ctx, fmt.Errorf("invalid_time_range: must provide both start_time and end_time for hash_window check")
		}
		if *payload.EndTime < *payload.StartTime {
			return ctx, fmt.Errorf("invalid_time_range: end_time must be >= start_time")
		}
		if *payload.EndTime-*payload.StartTime > 30*24*3600*1000 {
			return ctx, fmt.Errorf("invalid_time_range: max range is 30 days")
		}
		startT := time.UnixMilli(*payload.StartTime)
		endT := time.UnixMilli(*payload.EndTime)
		return servicerecon.WithReconTimeRange(ctx, startT, endT), nil
	}

	return ctx, nil
}
```
2. Trong `executeGenericCheck`:
   - Xóa bỏ hoàn toàn case `TypeReconFullDiff`.
   - Cập nhật case `TypeReconHashWindow` chạy trực tiếp `fnHash(ctx)`:
```go
	case TypeReconHashWindow:
		fallthrough
	default:
		return fnHash(ctx)
```

### C. Cập nhật `recon_check_heal_handler.go`
1. Tại `HandleReconHeal`, parse khoảng thời gian custom và nạp vào context:
```go
	if payload.StartTime != "" && payload.EndTime != "" {
		start, err1 := time.Parse(time.RFC3339, payload.StartTime)
		end, err2 := time.Parse(time.RFC3339, payload.EndTime)
		if err1 == nil && err2 == nil {
			ctx = servicerecon.WithReconTimeRange(ctx, start, end)
		}
	}
```
2. Sửa `proposeHealSegmentA` yêu cầu bắt buộc time range, chạy Hash Window check và heal đầy đủ:
```go
func (h *CheckHealHandler) proposeHealSegmentA(ctx context.Context, msg *nats.Msg, table, mode, startTimeStr, endTimeStr, lookback string) {
	startTime := time.Now()
	const op = "recon-heal-a-propose"

	h.logger.Info("[heal-a-propose] triggered", zap.String("table", table), zap.Time("triggered_at", startTime), zap.String("start_time", startTimeStr), zap.String("end_time", endTimeStr))

	entry := h.resolveTargetTableConfig(table)
	if entry == nil {
		h.handleHealError(msg, op, table, fmt.Errorf("registry not found: %s", table))
		return
	}

	if startTimeStr == "" || endTimeStr == "" {
		h.handleHealError(msg, op, table, fmt.Errorf("invalid time range: start_time and end_time are required"))
		return
	}

	start, err1 := time.Parse(time.RFC3339, startTimeStr)
	end, err2 := time.Parse(time.RFC3339, endTimeStr)
	if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > 30*24*time.Hour {
		h.handleHealError(msg, op, table, fmt.Errorf("invalid time range for heal: must be bounded within 30 days"))
		return
	}

	runCtx := servicerecon.WithReconTimeRange(ctx, start, end)
	newReport := h.reconCore.RunHashWindowCheck(runCtx, *entry)
	if newReport == nil {
		h.handleHealNoop(msg, op, table, "RunHashWindowCheck returned nil")
		return
	}

	if newReport.Status == "failed" {
		str := "failed"
		if newReport.ErrorMessage != nil {
			str = *newReport.ErrorMessage
		}
		h.handleHealError(msg, op, table, fmt.Errorf("hash window check failed: %s", str))
		return
	}

	if newReport.MissingCount == 0 && newReport.StaleCount == 0 && newReport.OrphanCount == 0 {
		h.handleHealNoop(msg, op, table, "RunHashWindowCheck window clean")
		return
	}

	missingIDs := parseMissingIDs(newReport.MissingIDs)
	staleObj := parseStaleSegmentA(newReport.StaleIDs)

	healIDs := append(append(append([]string{}, missingIDs...), staleObj.Mismatched...), staleObj.MissingFromSrc...)
	srcCount := int64(0)
	if newReport.SourceCount != nil {
		srcCount = *newReport.SourceCount
	}

	if h.healThresholdBlocked(msg, op, table, len(healIDs), srcCount, newReport.Diff) {
		return
	}

	if h.executeHeal != nil {
		opts := executeHealOpts{
			Table:           table,
			ReportIDs:       []uint64{newReport.ID},
			HealMismatched:  true,
			HealMissingDest: true,
			PruneMissingSrc: true,
			ForceHeal:       true,
		}

		processed, err := h.executeHeal.executeHeal(ctx, opts)
		if err != nil {
			h.handleHealError(msg, op, table, err)
			return
		}

		h.logActivity(op, table, "healed", int64(processed), nil)
		h.RespondJSON(msg, map[string]any{
			"status": "healed", "segment": SegmentSourceShadow, "healed_count": processed,
			"checked_at": newReport.CheckedAt, "missing_count": len(missingIDs),
			"mismatched_count": len(staleObj.Mismatched), "orphan_count": len(staleObj.MissingFromSrc),
		})
	}
}
```
- Xóa bỏ hoàn toàn 2 hàm cũ `proposeFullDiffHealA` và `proposeWindowHealA`.

---

## 5. Đồng bộ hiển thị tầng Read-side trong `cdc-cms-service`
Cập nhật SQL truy vấn để map `diff_time` vào `stale_ids` trong `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`:
```go
		UNION ALL
		SELECT
			id,
			run_id,
			segment,
			shadow_schema,
			shadow_table,
			CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END AS target_table,
			source_db,
			CASE WHEN segment = 'shadow_master' THEN COALESCE(shadow_active, 0) ELSE COALESCE(source_active, 0) END AS source_count,
			CASE WHEN segment = 'shadow_master' THEN COALESCE(master_active, 0) ELSE COALESCE(shadow_active, 0) END AS dest_count,
			diff,
			status,
			error_message,
			duration_ms,
			checked_at,
			CASE WHEN segment = 'shadow_master' THEN shadow_total ELSE source_total END AS total_source_count,
			CASE WHEN segment = 'shadow_master' THEN master_total ELSE shadow_total END AS total_dest_count,
			'smoke' AS check_type,
			1 AS tier,
			master_table,
			master_schema,
			0::integer AS missing_count,
			NULL::jsonb AS missing_ids,
			0::integer AS stale_count,
			diff_time AS stale_ids,          -- Thay đổi từ NULL::jsonb thành diff_time
			NULL::jsonb AS field_diffs,
            ...
```

---

## 6. Cập nhật giao diện Frontend trong `cdc-cms-web`

Sửa đổi `ConfirmDestructiveModal.tsx` tại [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx):
- Cập nhật state `checkMode` hỗ trợ các giá trị: `'2h' | '7d' | 'custom' | 'deep'`. Mặc định khi open modal sẽ là `'7d'`.
- Bổ sung `useEffect` cập nhật khoảng thời gian mặc định tương ứng khi đổi check mode.
- Cập nhật `handleOk` để tính toán timestamp StartTime và EndTime động tại thời điểm bấm OK cho các preset `2h` và `7d`. Truyền `typeRecon = 'hash_window'` cho cả preset `2h`, `7d` và `custom`. Đối với `deep` truyền `typeRecon = 'deep_check'`.
- Thiết kế lại các options hiển thị trên modal sử dụng component Radio Group và RangePicker của Ant Design. Vô hiệu hóa RangePicker đối với các preset `2h` và `7d` (được khóa hiển thị để làm rõ khoảng thời gian), ẩn RangePicker đối với `deep` check.

