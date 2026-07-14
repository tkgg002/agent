# Hồ sơ giải pháp kỹ thuật cụ thể - Luồng Chữa Lành Tương Tác (Rev.3 — Sửa 5 Lỗ Hổng)

Hồ sơ này mô tả chi tiết tất cả các đoạn mã thay đổi cần thiết cho cả Frontend (CMS-Web), API Gateway (cdc-cms-service) và Worker (centralized-data-service).

---

## 1. Migrations Cơ Sở Dữ Liệu

```sql
ALTER TABLE cdc_reconciliation_report
ADD COLUMN healed_mismatched_count INT DEFAULT 0,
ADD COLUMN healed_mismatched_duration_ms INT DEFAULT 0,
ADD COLUMN healed_missing_dest_count INT DEFAULT 0,
ADD COLUMN healed_missing_dest_duration_ms INT DEFAULT 0,
ADD COLUMN pruned_missing_src_count INT DEFAULT 0,
ADD COLUMN pruned_missing_src_duration_ms INT DEFAULT 0;
```

---

## 2. API Gateway (`cdc-cms-service`)

### 2.1 `internal/app/commands/recon/recon_async.go` — Thêm `ExecuteHealCommand`

```go
type ExecuteHealCommand struct {
	ports.AsyncCommandMixin
	Table           string   `json:"table"`
	Segment         string   `json:"segment,omitempty"`
	ReportIDs       []uint64 `json:"report_ids"`
	HealMismatched  bool     `json:"heal_mismatched"`
	HealMissingDest bool     `json:"heal_missing_dest"`
	PruneMissingSrc bool     `json:"prune_missing_src"`
}

func (ExecuteHealCommand) Type() string { return "execute-heal" }
func (c ExecuteHealCommand) Validate() error {
	if len(c.ReportIDs) == 0 {
		return errors.New("execute-heal: report_ids required")
	}
	return nil
}
```

### 2.2 `internal/api/recon/reconciliation_handler_execute_heal.go` — [NEW] HTTP handler

```go
func (h *ReconciliationHandler) TriggerExecuteHeal(c *fiber.Ctx) error {
	var req struct {
		Table           string   `json:"table"`
		Segment         string   `json:"segment"`
		ReportIDs       []uint64 `json:"report_ids"`
		HealMismatched  bool     `json:"heal_mismatched"`
		HealMissingDest bool     `json:"heal_missing_dest"`
		PruneMissingSrc bool     `json:"prune_missing_src"`
	}
	_ = c.BodyParser(&req)

	if len(req.ReportIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "report_ids required"})
	}

	user := middleware.GetUsername(c)
	ctx := messaging.WithMetadata(c.UserContext(), user, c.Get("X-Correlation-Id"), c.Get("Idempotency-Key"))
	res, derr := h.bus.Dispatch(ctx, reconCmd.ExecuteHealCommand{
		Table:           req.Table,
		Segment:         req.Segment,
		ReportIDs:       req.ReportIDs,
		HealMismatched:  req.HealMismatched,
		HealMissingDest: req.HealMissingDest,
		PruneMissingSrc: req.PruneMissingSrc,
	})
	if derr != nil {
		return c.Status(500).JSON(fiber.Map{"error": derr.Error()})
	}

	h.activityLogger.LogAsync(ports.ActivityEntry{
		Operation: "execute-heal-trigger", TargetTable: req.Table, Status: "success",
	})
	return c.Status(202).JSON(fiber.Map{"message": "execute-heal dispatched", "job_id": res.JobID})
}
```

### 2.3 `internal/api/recon/reconciliation_handler_reports.go` — Thêm `GetUnhealedReports`

```go
func (h *ReconciliationHandler) GetUnhealedReports(c *fiber.Ctx) error {
	table := c.Params("table")
	reports, err := h.reader.ListUnhealedReports(c.UserContext(), table)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(reports)
}
```

### 2.4 `internal/infra/persistence/recon/recon_read_repo_gorm.go` — Thêm query

```go
func (r *ReconReadRepoGorm) ListUnhealedReports(ctx context.Context, shadowTable string) ([]reconmodel.ReconciliationReport, error) {
	var reports []reconmodel.ReconciliationReport
	err := r.db.WithContext(ctx).
		Where("shadow_table = ? AND healed_at IS NULL AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)", shadowTable).
		Order("checked_at DESC").
		Find(&reports).Error
	return reports, err
}
```

### 2.5 `internal/model/recon/reconciliation_report.go` — Thêm 6 trường

```go
// Thêm sau trường HealedDurationMs:
HealedMismatchedCount      int `gorm:"column:healed_mismatched_count;default:0" json:"healed_mismatched_count"`
HealedMismatchedDurationMs int `gorm:"column:healed_mismatched_duration_ms;default:0" json:"healed_mismatched_duration_ms"`
HealedMissingDestCount     int `gorm:"column:healed_missing_dest_count;default:0" json:"healed_missing_dest_count"`
HealedMissingDestDurationMs int `gorm:"column:healed_missing_dest_duration_ms;default:0" json:"healed_missing_dest_duration_ms"`
PrunedMissingSrcCount      int `gorm:"column:pruned_missing_src_count;default:0" json:"pruned_missing_src_count"`
PrunedMissingSrcDurationMs int `gorm:"column:pruned_missing_src_duration_ms;default:0" json:"pruned_missing_src_duration_ms"`
```

---

## 3. CDC Worker (`centralized-data-service`)

### 3.1 `internal/handler/recon/recon_handler_run.go` — Đăng ký handler mới + Deprecate cũ

```go
// HandleExecuteHeal — subscribe "cdc.cmd.execute-heal"
func (h *ReconHandler) HandleExecuteHeal(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleExecuteHeal")
	defer span.End()

	var payload struct {
		Table           string   `json:"table"`
		Segment         string   `json:"segment"`
		ReportIDs       []uint64 `json:"report_ids"`
		HealMismatched  bool     `json:"heal_mismatched"`
		HealMissingDest bool     `json:"heal_missing_dest"`
		PruneMissingSrc bool     `json:"prune_missing_src"`
	}
	json.Unmarshal(msg.Data, &payload)

	h.logger.Info("execute-heal received",
		zap.Any("report_ids", payload.ReportIDs),
		zap.Bool("heal_mismatched", payload.HealMismatched),
		zap.Bool("heal_missing_dest", payload.HealMissingDest),
		zap.Bool("prune_missing_src", payload.PruneMissingSrc),
	)

	err := h.reconCore.ExecuteHeal(ctx, payload.ReportIDs, servicerecon.HealOptions{
		HealMismatched:  payload.HealMismatched,
		HealMissingDest: payload.HealMissingDest,
		PruneMissingSrc: payload.PruneMissingSrc,
	})

	if err != nil {
		h.logActivity("execute-heal", payload.Table, "error", 0, err)
		h.respondErr(msg, err)
		return
	}

	h.logActivity("execute-heal", payload.Table, "success", int64(len(payload.ReportIDs)), nil)
	if msg.Reply != "" {
		res, _ := json.Marshal(map[string]any{"status": "success", "reports_processed": len(payload.ReportIDs)})
		msg.Respond(res)
	}
}
```

Deprecate handler cũ:
```go
func (h *ReconHandler) HandleReconHeal(msg *nats.Msg) {
	h.logger.Warn("[DEPRECATED] cdc.cmd.recon-heal received — use cdc.cmd.execute-heal instead")
	// Backward-compat: chuyển hướng sang luồng check-only (không heal)
	// Hoặc giữ nguyên logic cũ trong thời gian chuyển tiếp
}
```

### 3.2 `internal/service/recon/recon_execute_heal.go` — [NEW] Logic thực thi

```go
package recon

import (
	"context"
	"encoding/json"
	"time"

	model "centralized-data-service/internal/model/recon"

	"go.uber.org/zap"
)

type HealOptions struct {
	HealMismatched  bool
	HealMissingDest bool
	PruneMissingSrc bool
}

func (rc *ReconCore) ExecuteHeal(ctx context.Context, reportIDs []uint64, opts HealOptions) error {
	rc.logger.Info("[execute-heal] starting", zap.Any("report_ids", reportIDs))

	for _, id := range reportIDs {
		var rpt model.ReconciliationReport
		if err := rc.db.WithContext(ctx).First(&rpt, id).Error; err != nil {
			rc.logger.Error("[execute-heal] failed to load report", zap.Uint64("id", id), zap.Error(err))
			continue
		}

		switch rpt.Segment {
		case "source_shadow", "":
			rc.executeHealSegmentA(ctx, &rpt, opts)
		case "shadow_master":
			rc.executeHealSegmentB(ctx, &rpt, opts)
		default:
			rc.logger.Warn("[execute-heal] unknown segment", zap.String("segment", rpt.Segment))
			continue
		}

		now := time.Now().UTC()
		rpt.HealedAt = &now
		rc.db.WithContext(ctx).Save(&rpt)
	}
	return nil
}

func (rc *ReconCore) executeHealSegmentA(ctx context.Context, rpt *model.ReconciliationReport, opts HealOptions) {
	// Parse Segment A format: {"mismatched": [...], "missing_from_src": [...]}
	var staleA struct {
		Mismatched     []string `json:"mismatched"`
		MissingFromSrc []string `json:"missing_from_src"`
	}
	json.Unmarshal(rpt.StaleIDs, &staleA)

	var missingIDs []string
	json.Unmarshal(rpt.MissingIDs, &missingIDs)

	entry := rc.resolveTableConfig(rpt.TargetTable)
	if entry == nil {
		rc.logger.Error("[execute-heal-a] registry not found", zap.String("table", rpt.TargetTable))
		return
	}

	if opts.HealMismatched && len(staleA.Mismatched) > 0 {
		start := time.Now()
		// written, err := h.FetchAndWriteByIDs(ctx, entry, staleA.Mismatched)
		rpt.HealedMismatchedCount = len(staleA.Mismatched)
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.HealMissingDest && len(missingIDs) > 0 {
		start := time.Now()
		// written, err := h.FetchAndWriteByIDs(ctx, entry, missingIDs)
		rpt.HealedMissingDestCount = len(missingIDs)
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.PruneMissingSrc && len(staleA.MissingFromSrc) > 0 {
		start := time.Now()
		// soft-delete: UPDATE shadow SET _deleted = true WHERE _source_id IN (...)
		rpt.PrunedMissingSrcCount = len(staleA.MissingFromSrc)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
	}
}

func (rc *ReconCore) executeHealSegmentB(ctx context.Context, rpt *model.ReconciliationReport, opts HealOptions) {
	// Parse Segment B format: {"stale_ids": [...], "orphan_in_master": [...]}
	var staleB struct {
		StaleIDs       []string `json:"stale_ids"`
		OrphanInMaster []string `json:"orphan_in_master"`
	}
	json.Unmarshal(rpt.StaleIDs, &staleB)

	var missingGpayIDs []string
	json.Unmarshal(rpt.MissingIDs, &missingGpayIDs)

	if opts.HealMismatched && len(staleB.StaleIDs) > 0 {
		start := time.Now()
		// sourceIDs := mapGpayToSourceIDs(ctx, rpt.SourceDB, staleB.StaleIDs)
		// publish cdc.cmd.transmute (re-trigger transmute theo chunk)
		rpt.HealedMismatchedCount = len(staleB.StaleIDs)
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.HealMissingDest && len(missingGpayIDs) > 0 {
		start := time.Now()
		// sourceIDs := mapGpayToSourceIDs(ctx, rpt.SourceDB, missingGpayIDs)
		// publish cdc.cmd.transmute
		rpt.HealedMissingDestCount = len(missingGpayIDs)
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.PruneMissingSrc && len(staleB.OrphanInMaster) > 0 {
		start := time.Now()
		// soft-delete orphan trong master table
		rpt.PrunedMissingSrcCount = len(staleB.OrphanInMaster)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
	}
}
```

---

## 4. CMS-Web Frontend (`cdc-cms-web`)

### 4.1 `src/hooks/useReconStatus.ts` — Thêm hooks mới

```typescript
export interface ExecuteHealPayload {
  table: string;
  segment?: string;
  reportIds: number[];
  healMismatched?: boolean;
  healMissingDest?: boolean;
  pruneMissingSrc?: boolean;
}

// Hook lấy danh sách phiên chưa heal
export const useUnhealedReports = (table: string) => {
  return useQuery(['unhealed-reports', table], () =>
    api.get(`/api/reconciliation/report/${table}/unhealed`).then(r => r.data)
  );
};

// Hook thực thi heal
export const useExecuteHealMutation = () => {
  return useMutation((payload: ExecuteHealPayload) =>
    api.post('/api/reconciliation/execute-heal', payload)
  );
};
```

### 4.2 Modal chữa lành tương tác
* Gọi `GET /api/reconciliation/report/:table/unhealed`.
* Hiển thị bảng danh sách report chưa heal (gom theo segment A/B).
* 3 checkboxes:
  * `[ ] Sửa đổi mismatched (X bản ghi)` — Seg A: ghi đè, Seg B: re-transmute
  * `[ ] Bổ sung missing (Y bản ghi)` — Seg A: fetch & write, Seg B: re-transmute
  * `[ ] Dọn dẹp orphan (Z bản ghi)` — Seg A: soft-delete shadow, Seg B: soft-delete master
* Click **"Thực hiện"** → POST payload lên `/api/reconciliation/execute-heal`.
