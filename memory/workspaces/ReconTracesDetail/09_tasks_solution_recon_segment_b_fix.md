# Hồ sơ Giải pháp Kỹ thuật Sửa lỗi Segment B (Recon & Heal)

Hồ sơ này hướng dẫn chi tiết Muscle thực hiện sửa đổi mã nguồn Go trong dự án `centralized-data-service`.

## 1. Sửa đổi `internal/service/recon/recon_tier_b.go`

### 1.1. Hàm `RunHashWindowCheckB` (Dòng ~230-290)
Tìm đoạn khởi tạo report và staleJSON:
```go
	missJSON, _ := json.Marshal(missingFromMaster)
	staleJSON, _ := json.Marshal(map[string][]string{
		"stale_ids":         staleIDs,
		"orphan_in_master":  orphanInMaster,
	})
```
Sửa thành:
```go
	missJSON, _ := json.Marshal(missingFromMaster)
	staleJSON, _ := json.Marshal(map[string][]string{
		"missing_from_dest": missingFromMaster,
		"missing_from_src":  orphanInMaster,
		"mismatched":        staleIDs,
	})
```
Và đoạn khởi tạo `ReconciliationReport`:
```go
	report := &recon.ReconciliationReport{
		TargetTable:      masterFQN,
		SourceDB:         shadowRel,
		SourceCount:      &shadowActive,
		TotalSourceCount: totalShadowFull,
		TotalDestCount:   totalMasterFull,
		DestCount:        masterActive,
		Diff:             shadowActive - masterActive,
		MissingCount:     len(missingFromMaster),
		MissingIDs:       missJSON,
		StaleCount:       len(staleIDs),
		StaleIDs:         staleJSON,
		OrphanCount:      len(orphanInMaster),
		CheckType:        "segment_b_window",
        ...
```
Sửa thành:
```go
	report := &recon.ReconciliationReport{
		TargetTable:      masterFQN,
		SourceDB:         "", // Để trống ở Segment B
		SourceCount:      &shadowActive,
		TotalSourceCount: totalShadowFull,
		TotalDestCount:   totalMasterFull,
		DestCount:        masterActive,
		Diff:             shadowActive - masterActive,
		MissingCount:     len(missingFromMaster),
		MissingIDs:       missJSON,
		StaleCount:       len(staleIDs) + len(orphanInMaster), // Bao gồm cả mồ côi
		StaleIDs:         staleJSON,
		OrphanCount:      len(orphanInMaster),
		CheckType:        "hash_window", // Đổi CheckType chuẩn
        ...
```

### 1.2. Hàm `RunDeepCheckB` (Dòng ~450-500)
Tìm đoạn khởi tạo report và staleJSON tương tự:
```go
	missJSON, _ := json.Marshal(missingFromMaster)
	staleJSON, _ := json.Marshal(map[string][]string{
		"stale_ids":         staleIDs,
		"orphan_in_master":  orphanInMaster,
	})
```
Sửa thành:
```go
	missJSON, _ := json.Marshal(missingFromMaster)
	staleJSON, _ := json.Marshal(map[string][]string{
		"missing_from_dest": missingFromMaster,
		"missing_from_src":  orphanInMaster,
		"mismatched":        staleIDs,
	})
```
Và đoạn khởi tạo `ReconciliationReport`:
```go
	report := &recon.ReconciliationReport{
		TargetTable:      masterFQN,
		SourceDB:         shadowRel,
		SourceCount:      &shadowActive,
		TotalSourceCount: totalShadowFull,
		TotalDestCount:   totalMasterFull,
		DestCount:        masterActive,
		Diff:             shadowActive - masterActive,
		MissingCount:     len(missingFromMaster),
		MissingIDs:       missJSON,
		StaleCount:       len(staleIDs),
		StaleIDs:         staleJSON,
		OrphanCount:      len(orphanInMaster),
		FieldDiffs:       fieldDiffsJSON,
		CheckType:        "segment_b_window",
        ...
```
Sửa thành:
```go
	report := &recon.ReconciliationReport{
		TargetTable:      masterFQN,
		SourceDB:         "", // Để trống ở Segment B
		SourceCount:      &shadowActive,
		TotalSourceCount: totalShadowFull,
		TotalDestCount:   totalMasterFull,
		DestCount:        masterActive,
		Diff:             shadowActive - masterActive,
		MissingCount:     len(missingFromMaster),
		MissingIDs:       missJSON,
		StaleCount:       len(staleIDs) + len(orphanInMaster),
		StaleIDs:         staleJSON,
		OrphanCount:      len(orphanInMaster),
		FieldDiffs:       fieldDiffsJSON,
		CheckType:        "bucket_hash", // Đổi CheckType chuẩn
        ...
```

---

## 2. Sửa đổi `internal/service/recon/recon_engine_segment_b.go`

### 2.1. Hàm `stampB` (Dòng ~10-40)
Tìm đoạn:
```go
	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
	report.MasterSchema, report.MasterTable = ref.MasterSchema, ref.MasterTable
	report.SourceType = "postgresql" // Hoặc gán từ ref
	report.SourceHost = ...
	report.SourceTable = ...
	report.SourceDB = ...
```
Sửa thành loại bỏ việc gán các trường `SourceType`, `SourceHost`, `SourceTable`, `SourceDB`:
```go
	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
	report.MasterSchema, report.MasterTable = ref.MasterSchema, ref.MasterTable
	report.SourceType = ""
	report.SourceHost = ""
	report.SourceTable = ""
	report.SourceDB = ""
```

---

## 3. Sửa đổi `internal/handler/recon/recon_base_handler.go`

### 3.1. Định nghĩa struct `staleSegmentB` (Dòng ~45)
```go
type staleSegmentB struct {
	StaleIDs       []string `json:"stale_ids"`
	OrphanInMaster []string `json:"orphan_in_master"`
}
```
Sửa thành:
```go
type staleSegmentB struct {
	Mismatched      []string `json:"mismatched"`
	MissingFromSrc  []string `json:"missing_from_src"`
	MissingFromDest []string `json:"missing_from_dest"`
}
```

### 3.2. Hàm `parseStaleSegmentB` (Dòng ~230)
```go
func parseStaleSegmentB(data []byte) staleSegmentB {
	var staleB staleSegmentB
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &staleB); err != nil {
			// Fallback if data is just a flat array of IDs
			var flatIDs []string
			if err2 := json.Unmarshal(data, &flatIDs); err2 == nil {
				staleB.OrphanInMaster = flatIDs
			}
		}
	}
	return staleB
}
```
Sửa thành:
```go
func parseStaleSegmentB(data []byte) staleSegmentB {
	var staleB staleSegmentB
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &staleB); err != nil {
			// Fallback if data is just a flat array of IDs
			var flatIDs []string
			if err2 := json.Unmarshal(data, &flatIDs); err2 == nil {
				staleB.MissingFromSrc = flatIDs
			}
		}
	}
	return staleB
}
```

---

## 4. Sửa đổi `internal/handler/recon/recon_execute_heal_handler.go`

### 4.1. Hàm `executeHealSegB` (Dòng ~259)
Tìm đoạn:
```go
	staleB := parseStaleSegmentB(rpt.StaleIDs)
	missingGpayIDs := parseMissingIDs(rpt.MissingIDs)

	// Deduplicate IDs to avoid duplicate processing
	staleB.StaleIDs = uniqueStrings(staleB.StaleIDs)
	staleB.OrphanInMaster = uniqueStrings(staleB.OrphanInMaster)
	missingGpayIDs = uniqueStrings(missingGpayIDs)

	healed := 0

	if opts.HealMismatched && len(staleB.StaleIDs) > 0 {
		start := time.Now()
		if sourceIDs, err := h.mapGpayToSourceIDs(ctx, rpt.SourceDB, staleB.StaleIDs); err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs, "execute-heal-b")
		}
		rpt.HealedMismatchedCount = len(staleB.StaleIDs)
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.HealMissingDest && len(missingGpayIDs) > 0 {
		start := time.Now()
		if sourceIDs, err := h.mapGpayToSourceIDs(ctx, rpt.SourceDB, missingGpayIDs); err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs, "execute-heal-b")
		}
		rpt.HealedMissingDestCount = len(missingGpayIDs)
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.PruneMissingSrc && len(staleB.OrphanInMaster) > 0 {
		start := time.Now()
		h.logger.Info("[execute-heal-b] prune orphan_in_master", zap.String("table", rpt.TargetTable))
		rpt.PrunedMissingSrcCount = len(staleB.OrphanInMaster)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += len(staleB.OrphanInMaster)
	}
```
Sửa thành:
```go
	staleB := parseStaleSegmentB(rpt.StaleIDs)
	missingGpayIDs := parseMissingIDs(rpt.MissingIDs)

	// Deduplicate IDs to avoid duplicate processing
	staleB.Mismatched = uniqueStrings(staleB.Mismatched)
	staleB.MissingFromSrc = uniqueStrings(staleB.MissingFromSrc)
	missingGpayIDs = uniqueStrings(missingGpayIDs)

	healed := 0

	var shadowRel string
	if rpt.Segment == SegmentShadowMaster {
		shadowRel = rpt.ShadowSchema + "." + rpt.ShadowTable
	} else {
		shadowRel = rpt.SourceDB
	}

	if opts.HealMismatched && len(staleB.Mismatched) > 0 {
		start := time.Now()
		if sourceIDs, err := h.mapGpayToSourceIDs(ctx, shadowRel, staleB.Mismatched); err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs, "execute-heal-b")
		}
		rpt.HealedMismatchedCount = len(staleB.Mismatched)
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.HealMissingDest && len(missingGpayIDs) > 0 {
		start := time.Now()
		if sourceIDs, err := h.mapGpayToSourceIDs(ctx, shadowRel, missingGpayIDs); err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs, "execute-heal-b")
		}
		rpt.HealedMissingDestCount = len(missingGpayIDs)
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.PruneMissingSrc && len(staleB.MissingFromSrc) > 0 {
		start := time.Now()
		h.logger.Info("[execute-heal-b] prune orphan_in_master (missing_from_src)", zap.String("table", rpt.TargetTable))
		rpt.PrunedMissingSrcCount = len(staleB.MissingFromSrc)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += len(staleB.MissingFromSrc)
	}
```

---

## 5. Sửa đổi `internal/handler/recon/recon_check_heal_handler.go`

### 5.1. Hàm `HandleRaw` (Dòng ~191)
Tìm đoạn:
```go
	missingGpayIDs := parseMissingIDs(report.MissingIDs)
	staleObj := parseStaleSegmentB(report.StaleIDs)
	gpayIDs := append(append(append([]string{}, missingGpayIDs...), staleObj.StaleIDs...), staleObj.OrphanInMaster...)
```
Sửa thành:
```go
	missingGpayIDs := parseMissingIDs(report.MissingIDs)
	staleObj := parseStaleSegmentB(report.StaleIDs)
	gpayIDs := append(append(append([]string{}, missingGpayIDs...), staleObj.Mismatched...), staleObj.MissingFromSrc...)
```
Và tương tự ở dòng 226 (nếu có):
Hãy tìm kiếm và đổi tất cả các nơi gọi `.StaleIDs` và `.OrphanInMaster` của `staleObj` hoặc `staleB` trong file này sang `.Mismatched` và `.MissingFromSrc`.
