# Hồ sơ giải pháp kỹ thuật - Loại bỏ handlePrune & Tái cấu trúc Routing

## 1. File cần thay đổi
- `internal/handler/recon/recon_check_handler.go`

## 2. Chi tiết code thay đổi

### 2.1 Tái cấu trúc logic routing trong HandleReconCheck
```go
	// 4. Routing Logic
	if payload.Table == "*" || payload.Table == "" {
		if payload.Segment == SegmentShadowMaster {
			h.handleCheckAllSegmentB(ctx, msg)
		} else {
			h.handleCheckAllSegmentA(ctx, msg)
		}
		return
	}

	if payload.Segment == SegmentShadowMaster {
		h.executeCheckSegmentB(ctx, msg, &payload)
	} else {
		h.executeCheckSegmentA(ctx, msg, &payload)
	}
```

### 2.2 Xóa hàm handlePrune
Xóa hoàn toàn hàm `handlePrune`.

### 2.3 Thêm các hàm xử lý mới cho Segment B
```go
func (h *CheckHandler) handleCheckAllSegmentB(ctx context.Context, msg *nats.Msg) {
	reports := h.reconCore.CheckAllSegmentB(ctx)
	status := "success"
	if len(reports) == 0 {
		status = "warning"
	}
	h.RespondJSON(msg, map[string]interface{}{"status": status, "segment": SegmentShadowMaster, "tables_checked": len(reports)})
	h.logActivity("recon-check-b-all", "*", status, int64(len(reports)), nil)
}

func (h *CheckHandler) executeCheckSegmentB(ctx context.Context, msg *nats.Msg, payload *reconCheckPayload) {
	isDeep := payload.TypeRecon == TypeReconDeepCheck || payload.TypeRecon == TypeReconFullDiff
	report := h.reconCore.RunSegmentBFor(ctx, payload.Table, isDeep)
	if report == nil {
		h.logActivity("recon-check-b", payload.Table, "error", 0, fmt.Errorf("master binding not found or not active: %s", payload.Table))
		h.RespondError(msg, "master_binding_not_found")
		return
	}

	h.logActivity("recon-check-b", payload.Table, report.Status, report.Diff, nil)
	h.RespondJSON(msg, report)
}
```

### 2.4 Thay thế executeStandardCheck bằng executeCheckSegmentA
```go
func (h *CheckHandler) executeCheckSegmentA(ctx context.Context, msg *nats.Msg, payload *reconCheckPayload) {
	entry := h.resolveTargetTableConfig(payload.Table)
	if entry == nil {
		h.logActivity("recon-check", payload.Table, "error", 0, fmt.Errorf("registry not found: %s", payload.Table))
		h.RespondError(msg, "registry_not_found")
		return
	}

	if payload.TypeRecon == TypeReconFullDiff {
		h.executeFullDiff(ctx, msg, payload, entry)
		return
	}

	var report *recon.ReconciliationReport
	switch payload.TypeRecon {
	case TypeReconDeepCheck:
		report = h.reconCore.RunDeepCheck(ctx, *entry)
	case TypeReconSmoke:
		report = h.reconCore.RunSmokeCheck(ctx, *entry)
	case TypeReconHashWindow:
		fallthrough
	default:
		tier2Ctx := context.WithValue(ctx, "manual_lookback", true)
		if payload.Lookback == LookbackCold {
			tier2Ctx = context.WithValue(tier2Ctx, "cold_lookback", true)
		}
		report = h.reconCore.RunHashWindowCheck(tier2Ctx, *entry)
	}

	h.logActivity("recon-check", payload.Table, report.Status, report.Diff, nil)
	h.RespondJSON(msg, report)
}
```
Xóa hàm `executeStandardCheck` và chỉnh sửa các lời gọi liên quan.
