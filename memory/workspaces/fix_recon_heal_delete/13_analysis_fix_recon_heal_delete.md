# Phân tích kỹ thuật - Sửa lỗi Heal không update/delete Master/Shadow (Bổ sung FQN Schema Prefix & Dời logic Resolve Config)

## 1. Phân tích lỗi record not found từ repository
Log lỗi:
```
{"level":"error","ts":1784111923.5318289,"caller":"source/table_registry_repo.go:32","msg":"gorm exec error","error":"record not found","elapsed":0.007030792,"rows":0,"sql":""}
```
Nguyên nhân:
* Bất kể Segment A hay Segment B, hàm `processSingleReport` đều gọi `resolveTargetTableConfig(rpt.TargetTable)`.
* Khi `rpt.Segment` là `SegmentShadowMaster` (Segment B), `rpt.TargetTable` có giá trị là `master_schema.master_table` (ví dụ `master_centrallized_export_service.export_jobs`).
* Khi query trong `cdc_table_registry` (nơi chỉ đăng ký shadow table name là `shadow_testces.export_jobs`), DB trả về `record not found`. Lỗi này được log trực tiếp từ hàm `GetByTargetTable` của GORM repository.

## 2. Giải pháp kỹ thuật dời Resolve Config
Do `entry` (kết quả của `resolveTargetTableConfig`) chỉ được truyền vào `executeHealSegA(ctx, rpt, entry, opts)` ở case `SegmentSourceShadow`, còn case `SegmentShadowMaster` gọi `executeHealSegB(ctx, rpt, opts)` hoàn toàn không dùng đến `entry` này.

Vì vậy, ta dời lệnh `entry := h.resolveTargetTableConfig(rpt.TargetTable)` vào bên trong case `SegmentSourceShadow, ""` ở switch-case:

```go
	switch rpt.Segment {
	case SegmentSourceShadow, "":
		entry := h.resolveTargetTableConfig(rpt.TargetTable)
		if entry == nil {
			h.logger.Error("[execute-heal-a] registry not found", zap.String("table", rpt.TargetTable))
			_ = h.reportRepo.ReleaseHealClaim(ctx, rpt.ID, prevStatus)
			return 0
		}
		processed = h.executeHealSegA(ctx, rpt, entry, opts)
	case SegmentShadowMaster:
		processed = h.executeHealSegB(ctx, rpt, opts)
...
```

Nhờ đó:
* Ở Segment B (`SegmentShadowMaster`), hệ thống không gọi `resolveTargetTableConfig` nữa, loại bỏ triệt để lỗi `record not found`.
* Ở Segment A (`SegmentSourceShadow`), hệ thống vẫn resolve config chính xác từ shadow table name.
