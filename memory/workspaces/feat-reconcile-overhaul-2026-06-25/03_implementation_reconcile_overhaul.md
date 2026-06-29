# Technical Design: Reconcile Component Overhaul

## 1. Phân tích Cấu trúc Table hiện tại
Bảng `cdc_system.cdc_reconciliation_report` hiện tại có các cột sau:
- ID, TargetTable, SourceDB, SourceCount, DestCount, Diff, MissingCount, MissingIDs, StaleCount, StaleIDs, CheckType, Status, Tier, DurationMs, ErrorMessage, ErrorCode, CheckedAt, HealedAt, HealedCount, Segment, ShadowSchema, ShadowTable, RunID, FieldDiffs, TotalSourceCount, TotalDestCount.

Các cột được gom thành các nhóm chức năng:
- **Pipeline Identity**: `(shadow_schema, shadow_table)` và `segment`.
- **Watermark/Checked Info**: `checked_at`, `run_id`, `duration_ms`.
- **Metrics/Counts**: `source_count`, `dest_count`, `diff`, `total_source_count`, `total_dest_count`.
- **Details (Drifts/Errors)**: `missing_ids`, `stale_ids`, `field_diffs`, `error_code`, `error_message`.

### Nhận xét:
Cấu trúc bảng sau migration 085 đã đủ thông tin định danh không mơ hồ (`shadow_schema`, `shadow_table`, `segment`). Việc thiết kế lại toàn bộ bảng bằng cách thay đổi tên cột hoặc cấu trúc gốc sẽ gây rủi ro phá vỡ (breaking change) rất lớn đối với dashboard UI của `cdc-cms-service` (được viết bằng Node.js/Go read-side). Do đó, giải pháp tối ưu, tuân thủ nguyên tắc **"Simplicity First, minimal impact"** là giữ nguyên cấu trúc bảng hiện tại, nhưng tối ưu hóa hoàn toàn **logic ghi dữ liệu** và **cơ chế dọn dẹp**.

---

## 2. Giải pháp Logic Ghi dữ liệu thông minh (Smart Write / Deduplication)
Thay vì chèn mới một dòng log `ok` vào DB sau mỗi chu kỳ (tạo hàng ngàn dòng rác lặp lại), ta sẽ:
1. Khi ghi nhận một report có trạng thái `status = 'ok'`:
   - Tìm kiếm bản ghi `ok` gần nhất của cùng pipeline (`shadow_schema`, `shadow_table`), `segment`, và `check_type`.
   - Nếu tìm thấy, thực hiện `UPDATE` bản ghi đó với các thông tin mới nhất (`checked_at`, `run_id`, `duration_ms`, `source_count`, `dest_count`, `total_source_count`, `total_dest_count`).
   - Nếu không tìm thấy hoặc bản ghi gần nhất có trạng thái khác (`drift` hoặc `error`), thực hiện `INSERT` bản ghi mới để đánh dấu mốc chuyển trạng thái (state transition).
2. Khi ghi nhận report có trạng thái `drift` hoặc `error`:
   - Luôn thực hiện `INSERT` mới để đảm bảo lịch sử lỗi/lệch dữ liệu được ghi nhận đầy đủ làm bằng chứng cho audit và phục vụ tự sửa lỗi (healing).

### Mã nguồn đề xuất cho `stamp` logic (trong `recon_engine_segment_b.go`):
```go
func (rc *ReconCore) stampA(report *recon.ReconciliationReport, entry source.TableRegistry) *recon.ReconciliationReport {
	report.ShadowSchema, report.ShadowTable, report.RunID = entry.ShadowSchema, entry.TargetTable, entry.RunID
	rc.stamp(report)
	return report
}

func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) *recon.ReconciliationReport {
	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
	rc.stamp(report)
	return report
}

func (rc *ReconCore) stamp(report *recon.ReconciliationReport) {
	if report.Status == "ok" {
		var existing recon.ReconciliationReport
		// Tìm bản ghi OK mới nhất của pipeline này
		err := rc.db.Where("shadow_schema = ? AND shadow_table = ? AND segment = ? AND check_type = ? AND status = 'ok'",
			report.ShadowSchema, report.ShadowTable, report.Segment, report.CheckType).
			Order("checked_at DESC").Limit(1).Find(&existing).Error
		if err == nil && existing.ID > 0 {
			// Cập nhật bản ghi OK cũ
			rc.db.Model(&existing).Updates(map[string]interface{}{
				"checked_at":         report.CheckedAt,
				"run_id":             report.RunID,
				"duration_ms":        report.DurationMs,
				"source_count":       report.SourceCount,
				"dest_count":         report.DestCount,
				"total_source_count": report.TotalSourceCount,
				"total_dest_count":   report.TotalDestCount,
			})
			report.ID = existing.ID
			return
		}
	}
	// Insert mới nếu status là drift/error hoặc chưa có bản ghi OK nào
	rc.db.Create(report)
}
```

---

## 3. Giải pháp tự động dọn dẹp rác (Self-Cleaning / Pruning Job)
Mỗi khi chạy chu kỳ đối soát chính (`CheckAll` và `CheckAllSegmentB`), hệ thống sẽ chạy một câu lệnh SQL để dọn dẹp các bản ghi `ok` cũ hơn 7 ngày.
- Câu lệnh SQL:
  ```sql
  DELETE FROM cdc_system.cdc_reconciliation_report
   WHERE status = 'ok' AND checked_at < NOW() - INTERVAL '7 days'
  ```

### Điểm tích hợp:
Thêm phương thức `pruneSuccessReports(ctx)` vào `recon_engine_run.go` và gọi ở đầu hàm `CheckAll` và `CheckAllSegmentB`.
```go
func (rc *ReconCore) pruneSuccessReports(ctx context.Context) {
	err := rc.db.WithContext(ctx).Exec(`
		DELETE FROM cdc_system.cdc_reconciliation_report
		 WHERE status = 'ok' AND checked_at < NOW() - INTERVAL '7 days'
	`).Error
	if err != nil {
		rc.logger.Warn("recon: failed to prune old success reports", zap.Error(err))
	}
}
```

---

## 4. Giải pháp Chỉ chạy và lưu duy nhất check_type = 'count_total'
Để cấu hình hệ thống chỉ đối soát tổng số lượng (Tier 0 - O(1)) mà không thực hiện bucket/hash/drill-down nặng nề ở các Tier sâu hơn:
1. **Segment A (`recon_tier_a.go`)**:
   - Thay vì rẽ nhánh sang quét bucket khi lệch counts, ta trả trực tiếp report với `check_type = "count_total"` và `status = "drift"`.
2. **Segment B (`recon_tier_b.go`)**:
   - Tương tự, nếu phát hiện lệch hoặc watermark trễ, trả trực tiếp report Segment B với `check_type = "count_total"` và `status = "drift"` thay vì chạy Bucket scan.

Điều này giảm tải 100% tài nguyên I/O cho drill-down và giữ cho bảng đối soát chỉ duy nhất chứa các bản ghi `count_total` (ở trạng thái `ok` hoặc `drift`).

