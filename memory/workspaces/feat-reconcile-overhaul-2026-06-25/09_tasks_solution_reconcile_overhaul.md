# Proposed Solution & Code Diffs: Reconcile Overhaul

Dưới đây là chi tiết mã nguồn dự kiến chỉnh sửa cho cụm Reconcile để giải quyết triệt để vấn đề rác log đối soát thành công (OK).

---

## 1. File `internal/service/recon/recon_engine_segment_b.go`

Thay thế hàm `stampA` và `stampB` bằng logic `stamp` deduplicate thông minh:

```diff
-func (rc *ReconCore) stampA(report *recon.ReconciliationReport, entry source.TableRegistry) *recon.ReconciliationReport {
-	report.ShadowSchema, report.ShadowTable, report.RunID = entry.ShadowSchema, entry.TargetTable, entry.RunID
-	rc.db.Create(report)
-	return report
-}
-
-func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) *recon.ReconciliationReport {
-	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
-	rc.db.Create(report)
-	return report
-}
+func (rc *ReconCore) stampA(report *recon.ReconciliationReport, entry source.TableRegistry) *recon.ReconciliationReport {
+	report.ShadowSchema, report.ShadowTable, report.RunID = entry.ShadowSchema, entry.TargetTable, entry.RunID
+	rc.stamp(report)
+	return report
+}
+
+func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) *recon.ReconciliationReport {
+	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
+	rc.stamp(report)
+	return report
+}
+
+func (rc *ReconCore) stamp(report *recon.ReconciliationReport) {
+	if report.Status == "ok" {
+		var existing recon.ReconciliationReport
+		// Tìm kiếm bản ghi OK gần nhất cho pipeline này
+		err := rc.db.Where("shadow_schema = ? AND shadow_table = ? AND segment = ? AND check_type = ? AND status = 'ok'",
+			report.ShadowSchema, report.ShadowTable, report.Segment, report.CheckType).
+			Order("checked_at DESC").Limit(1).Find(&existing).Error
+		if err == nil && existing.ID > 0 {
+			// Cập nhật mốc thời gian và counts vào bản ghi cũ thay vì INSERT
+			rc.db.Model(&existing).Updates(map[string]interface{}{
+				"checked_at":         report.CheckedAt,
+				"run_id":             report.RunID,
+				"duration_ms":        report.DurationMs,
+				"source_count":       report.SourceCount,
+				"dest_count":         report.DestCount,
+				"total_source_count": report.TotalSourceCount,
+				"total_dest_count":   report.TotalDestCount,
+			})
+			report.ID = existing.ID
+			return
+		}
+	}
+	// INSERT nếu là trạng thái lỗi/drift hoặc là mốc OK đầu tiên
+	rc.db.Create(report)
+}
```

---

## 2. File `internal/service/recon/recon_engine_run.go`

Bổ sung hàm `pruneSuccessReports` ở cuối file và gọi ở đầu hàm `CheckAll`:

```diff
@@ -146,6 +146,7 @@
 func (rc *ReconCore) CheckAll(ctx context.Context) []*recon.ReconciliationReport {
 	isLeader, release := rc.AcquireLeader(ctx)
 	defer release()
 	if !isLeader {
 		observability.Ctx(ctx, rc.logger).Info("recon CheckAll — not leader, skipping")
 		return nil
 	}
 
+	rc.pruneSuccessReports(ctx)
+
 	entries := rc.listActiveTableConfigs(ctx)
```

```go
// Tích hợp ở cuối file recon_engine_run.go
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

## 3. File `internal/service/recon/recon_tier_b.go`

Tích hợp gọi hàm `pruneSuccessReports` ở đầu hàm `CheckAllSegmentB`:

```diff
@@ -390,6 +390,7 @@
 func (rc *ReconCore) CheckAllSegmentB(ctx context.Context) []*recon.ReconciliationReport {
 	isLeader, release := rc.AcquireLeader(ctx)
 	defer release()
 	if !isLeader {
 		observability.Ctx(ctx, rc.logger).Info("recon CheckAllSegmentB — not leader, skipping")
 		return nil
 	}
+	rc.pruneSuccessReports(ctx)
+
 	refs := rc.listActiveMasterBindings(ctx)
```

---

## 4. Đề xuất Diffs để CHỈ chạy đối soát tổng số lượng (Keep ONLY count_total)

### File `internal/service/recon/recon_tier_a.go`
Thay đổi logic trả về kết quả `count_total` trực tiếp khi lệch counts (thay vì rẽ sang chạy bucket_hash):

```diff
@@ -512,23 +512,32 @@
 	estTolerance := srcEst / 1000
 	if estTolerance < 1 {
 		estTolerance = 1
 	}
-	if abs(srcEst-dstActive) <= estTolerance {
-		duration := int(time.Since(handle.started).Milliseconds())
-		report := &recon.ReconciliationReport{
-			TargetTable: entry.TargetTable, SourceDB: entry.SourceDB,
-			SourceCount: &srcEst, DestCount: dstActive, Diff: srcEst - dstActive,
-			TotalSourceCount: &srcEst, TotalDestCount: &dstTotal,
-			CheckType: "count_total", Status: "ok", Tier: 1,
-			DurationMs: &duration, CheckedAt: time.Now().UTC(),
-		}
-		rc.stampA(report, entry)
-		metrics.ReconDrift.WithLabelValues(entry.TargetTable, "1").Set(0)
-		observability.Ctx(ctx, rc.logger).Info("tier0 count_total ok",
-			zap.String("table", entry.TargetTable),
-			zap.Int64("src_est", srcEst), zap.Int64("dst_active", dstActive),
-			zap.Int64("dst_total", dstTotal))
-		return report
-	}
+
+	statusStr := "ok"
+	diffVal := srcEst - dstActive
+	if abs(diffVal) > estTolerance {
+		statusStr = "drift"
+	}
+	duration := int(time.Since(handle.started).Milliseconds())
+	report := &recon.ReconciliationReport{
+		TargetTable:      entry.TargetTable,
+		SourceDB:         entry.SourceDB,
+		SourceCount:      &srcEst,
+		DestCount:        dstActive,
+		Diff:             diffVal,
+		TotalSourceCount: &srcEst,
+		TotalDestCount:   &dstTotal,
+		CheckType:        "count_total",
+		Status:           statusStr,
+		Tier:             1,
+		DurationMs:       &duration,
+		CheckedAt:        time.Now().UTC(),
+	}
+	rc.stampA(report, entry)
+	if statusStr == "ok" {
+		metrics.ReconDrift.WithLabelValues(entry.TargetTable, "1").Set(0)
+	} else {
+		metrics.ReconDrift.WithLabelValues(entry.TargetTable, "1").Set(float64(abs(diffVal)))
+		rc.alertOnReport(ctx, "source_shadow", entry.TargetTable, statusStr, 0, diffVal)
+	}
+	return report
```

### File `internal/service/recon/recon_tier_b.go`
Thay đổi logic trả về kết quả `count_total` trực tiếp cho Segment B khi lệch hoặc watermark trễ:

```diff
@@ -87,21 +87,28 @@
-	if errSF == nil && errMF == nil && shadowActive == masterActive && transmuteLagMs == 0 {
-		// KHỚP cả count lẫn watermark → DỪNG. Không bucket, không drill-down.
-		duration := int(time.Since(handle.started).Milliseconds())
-		report := &recon.ReconciliationReport{
-			TargetTable: ref.MasterTable, SourceDB: shadowRel,
-			SourceCount: &shadowActive, DestCount: masterActive, Diff: 0,
-			TotalSourceCount: &shadowFull, TotalDestCount: &masterFull,
-			CheckType: "count_total", Status: "ok", Tier: tierSegmentB,
-			Segment: segmentShadowMaster, DurationMs: &duration, CheckedAt: time.Now().UTC(),
-		}
-		rc.stampB(report, ref)
-		rc.finishRun(ctx, handle, "success", "")
-		metrics.ReconDrift.WithLabelValues(ref.MasterTable, "4").Set(0)
-		if errMF == nil {
-			metrics.MasterTableRowCount.WithLabelValues(ref.MasterTable).Set(float64(masterFull))
-			metrics.MasterActiveRowCount.WithLabelValues(ref.MasterTable).Set(float64(masterActive))
-		}
-		observability.Ctx(ctx, rc.logger).Info("recon segment B tier0 ok",
-			zap.String("master", masterRel),
-			zap.Int64("total", masterFull), zap.Int64("active", masterActive))
-		return report
-	}
+	statusStr := "ok"
+	diffVal := shadowActive - masterActive
+	if diffVal != 0 || transmuteLagMs > 0 {
+		statusStr = "drift"
+	}
+	duration := int(time.Since(handle.started).Milliseconds())
+	report := &recon.ReconciliationReport{
+		TargetTable:      ref.MasterTable,
+		SourceDB:         shadowRel,
+		SourceCount:      &shadowActive,
+		DestCount:        masterActive,
+		Diff:             diffVal,
+		TotalSourceCount: &shadowFull,
+		TotalDestCount:   &masterFull,
+		CheckType:        "count_total",
+		Status:           statusStr,
+		Tier:             tierSegmentB,
+		Segment:          segmentShadowMaster,
+		DurationMs:       &duration,
+		CheckedAt:        time.Now().UTC(),
+	}
+	rc.stampB(report, ref)
+	if statusStr == "ok" {
+		metrics.ReconDrift.WithLabelValues(ref.MasterTable, "4").Set(0)
+	} else {
+		metrics.ReconDrift.WithLabelValues(ref.MasterTable, "4").Set(float64(abs(diffVal)))
+		rc.alertOnReport(ctx, segmentShadowMaster, ref.MasterTable, statusStr, 0, diffVal)
+	}
+	return report
```

