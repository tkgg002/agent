# Hồ sơ giải pháp kỹ thuật cụ thể - Sửa lỗi biên dịch recon_tier_b.go

Hồ sơ giải pháp sửa đổi cụ thể cho file `internal/service/recon/recon_tier_b.go`.

## 1. Xóa stampB trùng lặp
Xóa block sau từ dòng 631 đến 641:
```go
func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) {
	if report == nil {
		return
	}
	report.SourceDB = ref.SourceDB
	report.TargetSchema = ref.MasterSchema
	report.TargetTable = ref.MasterTable
	if report.TargetTable == "" {
		report.TargetTable = ref.MasterRel()
	}
}
```

## 2. Sửa lỗi errorReportB
Thay thế trường `SourceDB` bằng chuỗi rỗng:
```go
func (rc *ReconCore) errorReportB(ref MasterBindingRef, checkType string, err error) *recon.ReconciliationReport {
	observability.Ctx(context.Background(), rc.logger).Error("tierB error",
		zap.String("table", ref.runName()),
		zap.String("check", checkType),
		zap.Error(err),
	)
	return &recon.ReconciliationReport{
		TargetTable: ref.MasterTable,
		SourceDB:    "",
		CheckType:   checkType,
		Status:      "error",
		Segment:     segmentShadowMaster,
		CheckedAt:   time.Now().UTC(),
	}
}
```

## 3. Thêm RunSegmentB
Định nghĩa phương thức `RunSegmentB` trên `ReconCore` ngay trước `RunSegmentBFor`:
```go
func (rc *ReconCore) RunSegmentB(ctx context.Context, ref MasterBindingRef, deep bool) *recon.ReconciliationReport {
	if deep {
		return rc.RunDeepCheckB(ctx, ref)
	}
	return rc.RunHashWindowCheckB(ctx, ref)
}
```
