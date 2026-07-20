# Hướng dẫn giải pháp kỹ thuật - Sửa lỗi Heal không update/delete Master/Shadow (FQN Schema Prefix & Dời logic Resolve Config)

Hãy sửa đổi mã nguồn backend của centralized-data-service theo hướng dẫn chi tiết dưới đây.

---

## 1. File [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)

Đảm bảo method `MasterPlane() *gorm.DB` đã được export đầy đủ:

```go
func (rc *ReconCore) MasterPlane() *gorm.DB {
	return rc.masterPlane
}
```

---

## 2. File [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)

### Sửa hàm `processSingleReport` (khoảng dòng 145)
Tìm và sửa đổi toàn bộ hàm `processSingleReport` thành cấu trúc sau:

```go
func (h *ExecuteHealHandler) processSingleReport(ctx context.Context, rpt *modelrecon.ReconciliationReport, opts executeHealOpts) int {
	prevStatus := rpt.Status
	if rpt.Segment == SegmentShadowMaster {
		if rpt.MasterSchema != "" && !strings.Contains(rpt.TargetTable, ".") {
			rpt.TargetTable = rpt.MasterSchema + "." + rpt.MasterTable
		} else if rpt.TargetTable == "" {
			rpt.TargetTable = rpt.MasterTable
		}
	} else {
		if rpt.ShadowSchema != "" && !strings.Contains(rpt.TargetTable, ".") {
			rpt.TargetTable = rpt.ShadowSchema + "." + rpt.ShadowTable
		} else if rpt.TargetTable == "" {
			rpt.TargetTable = rpt.ShadowTable
		}
	}
	
	processed := 0

	switch rpt.Segment {
	case SegmentSourceShadow, "":
		// Chỉ resolve config cho Segment A vì Segment A mới cần entry để query MongoDB nguồn
		entry := h.resolveTargetTableConfig(rpt.TargetTable)
		if entry == nil {
			h.logger.Error("[execute-heal-a] registry not found", zap.String("table", rpt.TargetTable))
			_ = h.reportRepo.ReleaseHealClaim(ctx, rpt.ID, prevStatus)
			return 0
		}
		processed = h.executeHealSegA(ctx, rpt, entry, opts)
	case SegmentShadowMaster:
		processed = h.executeHealSegB(ctx, rpt, opts)
	default:
		h.logger.Warn("[execute-heal] unknown segment", zap.String("segment", rpt.Segment))
		_ = h.reportRepo.ReleaseHealClaim(ctx, rpt.ID, prevStatus)
		return 0
	}

	h.finalizeReport(ctx, rpt)
	return processed
}
```

### Sửa hàm `executeHealSegB` (khoảng dòng 360)
Giữ nguyên logic SQL DELETE đã chạy thành công:
```go
	if opts.PruneMissingSrc && len(staleB.MissingFromSrc) > 0 {
		start := time.Now()
		h.logger.Info("[execute-heal-b] prune orphan_in_master (missing_from_src)", zap.String("table", rpt.TargetTable), zap.Int("count", len(staleB.MissingFromSrc)))
		
		masterDB := h.reconCore.MasterPlane()
		if masterDB != nil {
			// Thực hiện XÓA CỨNG trên Master DB theo yêu cầu của User
			delSQL := fmt.Sprintf(
				`DELETE FROM %s WHERE "_gpay_id" IN (?)`,
				quoteRelation(rpt.TargetTable),
			)
			const batch = 1000
			pruned := 0
			for i := 0; i < len(staleB.MissingFromSrc); i += batch {
				end := i + batch
				if end > len(staleB.MissingFromSrc) {
					end = len(staleB.MissingFromSrc)
				}
				res := masterDB.WithContext(ctx).Exec(delSQL, staleB.MissingFromSrc[i:end])
				if res.Error != nil {
					h.logger.Error("[execute-heal-b] prune master (delete) failed", zap.Error(res.Error))
				} else {
					pruned += int(res.RowsAffected)
				}
			}
			h.logger.Info("[execute-heal-b] prune master completed", zap.Int("pruned", pruned))
			rpt.PrunedMissingSrcCount = pruned
		} else {
			h.logger.Error("[execute-heal-b] masterDB not wired, cannot prune master")
		}
		
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += rpt.PrunedMissingSrcCount
	}
```

### Sửa hàm `executeHealSegA` (khoảng dòng 250)
Giữ nguyên logic SQL UPDATE đã chạy thành công:
```go
	if opts.PruneMissingSrc && len(staleA.MissingFromSrc) > 0 {
		start := time.Now()
		h.logger.Info("[execute-heal-a] prune missing_src (soft-delete pending)", zap.String("table", rpt.TargetTable), zap.Int("count", len(staleA.MissingFromSrc)))
		
		updSQL := fmt.Sprintf(
			`UPDATE %s SET "_deleted" = TRUE, "_updated_at" = NOW() WHERE "_source_id" IN (?) AND NOT "_deleted"`,
			quoteRelation(rpt.TargetTable),
		)
		const batch = 1000
		pruned := 0
		for i := 0; i < len(staleA.MissingFromSrc); i += batch {
			end := i + batch
			if end > len(staleA.MissingFromSrc) {
				end = len(staleA.MissingFromSrc)
			}
			res := h.shadowDB.WithContext(ctx).Exec(updSQL, staleA.MissingFromSrc[i:end])
			if res.Error != nil {
				h.logger.Error("[execute-heal-a] prune shadow failed", zap.Error(res.Error))
			} else {
				pruned += int(res.RowsAffected)
			}
		}
		h.logger.Info("[execute-heal-a] prune shadow completed", zap.Int("pruned", pruned))
		rpt.PrunedMissingSrcCount = pruned
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += pruned
	}
```
