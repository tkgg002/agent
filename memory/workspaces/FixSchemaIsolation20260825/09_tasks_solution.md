# 09_tasks_solution.md — Hồ sơ Giải pháp Kỹ thuật Chi tiết

## 1. Giải pháp Task 1: `recon_check_heal_handler.go`
```go
targetTable := payload.Table
if payload.ShadowSchema != "" && !strings.Contains(targetTable, ".") {
    targetTable = payload.ShadowSchema + "." + targetTable
}
if payload.Segment == SegmentShadowMaster {
    if payload.MasterSchema != "" && !strings.Contains(targetTable, ".") {
        targetTable = payload.MasterSchema + "." + targetTable
    }
    h.proposeHealSegmentB(ctx, msg, targetTable)
} else {
    h.proposeHealSegmentA(ctx, msg, targetTable, payload.Mode, payload.StartTime, payload.EndTime, payload.Lookback)
}
```

## 2. Giải pháp Task 3: `recon_smoke.go` & `recon_tier_b.go`
```go
tsCol := "_source_ts"
if resolver, ok := rc.registryRepo.(interface {
    GetByTargetTableAndSchema(ctx context.Context, targetTable string, schemaName string) (*modelsource.TableRegistry, error)
}); ok {
    if entry, err := resolver.GetByTargetTableAndSchema(ctx, ref.ShadowTable, ref.ShadowSchema); err == nil && entry != nil {
        _, dstTS, errTS := rc.resolveSourceAndDestTSFields(ctx, *entry, "smoke")
        if errTS == nil && dstTS != "" {
            tsCol = dstTS
        }
    }
}
```
