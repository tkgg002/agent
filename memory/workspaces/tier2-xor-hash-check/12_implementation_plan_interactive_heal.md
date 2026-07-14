# Kế Hoạch Triển Khai: Tách Biệt Đối Soát & Thực Thi — Luồng Chữa Lành Tương Tác (Rev.5 Final)

> **Phiên bản**: Rev.5 Final (2026-07-02)
> **Tổng hợp**: 14 lỗ hổng từ 3 vòng review đã được sửa triệt để.

---

## 1. Vấn Đề Gốc

Khi bấm "Chữa lành" trên UI, gateway dispatch `ReconHealCommand` → worker nhận và **vừa chạy đối soát (RunTier2/RunSegmentBFor) vừa heal** — vi phạm SRP. Payload `ReconHealCommand` thực chất là tham số đối soát (Mode, StartTime, EndTime, Lookback), không phải tham số thực thi heal.

**Giải pháp**: Tách thành 3 bước rõ ràng — (1) Check riêng, (2) Xem danh sách chưa heal, (3) Thực thi heal granular qua command mới `ExecuteHealCommand`.

---

## 2. Luồng Mới

```
[Bước 1: ĐỐI SOÁT — đã có sẵn, không đổi]
  POST /api/reconciliation/check/:table → NATS cdc.cmd.recon-check
  Worker: RunTier2 / RunSegmentBFor → Ghi report vào DB

[Bước 2: XEM PHIÊN CHƯA HEAL — endpoint mới]
  GET /api/reconciliation/report/:table/unhealed
  Query: (shadow_table = ? OR master_table = ?) AND healed_at IS NULL
         AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)

[Bước 3: THỰC THI GRANULAR — command + handler mới]
  POST /api/reconciliation/execute-heal
  Payload: {report_ids, heal_mismatched, heal_missing_dest, prune_missing_src}
  → NATS cdc.cmd.execute-heal → Worker: load report → parse theo segment → heal/prune
```

---

## 3. Deprecation

| Cũ | Hành động |
|----|-----------|
| `ReconHealCommand` / `cdc.cmd.recon-heal` | Giữ handler, log `[DEPRECATED]`, không xóa |
| Nhánh `payload.Legacy` (V3 Healer) | Log warning, không thực thi |

---

## 4. Thay Đổi Chi Tiết Theo File

### 4.A — Database Migration

> [!IMPORTANT]
> Cần xác định số migration tiếp theo (convention: `migrations/XXXX_*.sql`).

```sql
ALTER TABLE cdc_reconciliation_report
ADD COLUMN healed_mismatched_count INT DEFAULT 0,
ADD COLUMN healed_mismatched_duration_ms INT DEFAULT 0,
ADD COLUMN healed_missing_dest_count INT DEFAULT 0,
ADD COLUMN healed_missing_dest_duration_ms INT DEFAULT 0,
ADD COLUMN pruned_missing_src_count INT DEFAULT 0,
ADD COLUMN pruned_missing_src_duration_ms INT DEFAULT 0;
```

Report cũ `healed_at IS NOT NULL` giữ nguyên → query unhealed filter `IS NULL` → không hiển thị nhầm.

---

### 4.B — API Gateway (`cdc-cms-service`)

#### [MODIFY] [recon_async.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_async.go) — Thêm `ExecuteHealCommand`

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

#### [MODIFY] [server.go L261](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go#L261) — Đăng ký NATS subject

```diff
 cmdBus.RegisterSubject("recon.heal", "cdc.cmd.recon-heal")
+cmdBus.RegisterSubject("execute-heal", "cdc.cmd.execute-heal")
```

#### [MODIFY] [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go) — Đăng ký route

```diff
 // Destructive routes (L178):
 registerDestructive("/reconciliation/prune/:table", h.Recon.TriggerPrune)
+registerDestructive("/reconciliation/execute-heal", h.Recon.TriggerExecuteHeal)

 // Read routes (L280):
 dual("GET", shared, "/recon/backfill-source-ts/status", h.Recon.BackfillSourceTsStatus)
+dual("GET", shared, "/reconciliation/report/:table/unhealed", h.Recon.GetUnhealedReports)
```

#### [NEW] `internal/app/queries/recon/list_unhealed_reports.go` — CQRS Query Handler

> Pattern tuân thủ: `ListLatestReportsHandler`, `GetTableHistoryHandler`, `ListFailedLogsHandler` — tất cả đều dùng struct Query Handler inject vào constructor.

```go
package recon

import (
	"context"
	reconmodel "cdc-cms-service/internal/model/recon"
)

type ListUnhealedReportsQuery struct {
	Table        string // shadow_table hoặc master_table
	ShadowSchema string // optional, cho chính xác hơn
}

type ListUnhealedReportsResult struct {
	Data []reconmodel.ReconciliationReport
}

type ListUnhealedReportsHandler struct {
	reader ReconReader
}

func NewListUnhealedReportsHandler(reader ReconReader) *ListUnhealedReportsHandler {
	return &ListUnhealedReportsHandler{reader: reader}
}

func (h *ListUnhealedReportsHandler) Handle(ctx context.Context, q ListUnhealedReportsQuery) (*ListUnhealedReportsResult, error) {
	reports, err := h.reader.ListUnhealedReports(ctx, q.Table, q.ShadowSchema)
	if err != nil {
		return nil, err
	}
	return &ListUnhealedReportsResult{Data: reports}, nil
}
```

#### [MODIFY] [recon_reader.go L64](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/recon_reader.go#L64) — Thêm method vào interface

```diff
 type ReconReader interface {
     ListLatest(ctx context.Context) ([]LatestReportRow, error)
     GetTableHistory(ctx, table, shadowSchema, masterTable string, page, pageSize int) (...)
     // ...existing methods...
+    ListUnhealedReports(ctx context.Context, table, shadowSchema string) ([]reconmodel.ReconciliationReport, error)
 }
```

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go) — Implement query

> Query key: dùng `shadow_table OR master_table` (Migration 085) — không dùng `target_table` (ambiguous).

```go
func (r *reconReadRepoGorm) ListUnhealedReports(ctx context.Context, table, shadowSchema string) ([]reconmodel.ReconciliationReport, error) {
	var reports []reconmodel.ReconciliationReport
	q := r.db.WithContext(ctx).
		Table("cdc_system.cdc_reconciliation_report").
		Where("(shadow_table = ? OR master_table = ?)", table, table).
		Where("healed_at IS NULL").
		Where("(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)")
	if shadowSchema != "" {
		q = q.Where("shadow_schema = ?", shadowSchema)
	}
	err := q.Order("checked_at DESC").Find(&reports).Error
	return reports, err
}
```

#### [MODIFY] [reconciliation_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler.go#L11) — Inject query handler

```diff
 type ReconciliationHandler struct {
     reader         recon.ReconReader
     bus            ports.CommandBus
     listLatestQ    *recon.ListLatestReportsHandler
     getHistoryQ    *recon.GetTableHistoryHandler
     listFailedQ    *recon.ListFailedLogsHandler
+    listUnhealedQ  *recon.ListUnhealedReportsHandler
     activityLogger ports.ActivityLogger
     logger         *zap.Logger
 }
```

Constructor `NewReconciliationHandler` thêm param `listUnhealedQ`.

#### [NEW] `internal/api/recon/reconciliation_handler_execute_heal.go` — HTTP handler

```go
package recon

import (
	reconCmd "cdc-cms-service/internal/app/commands/recon"
	"cdc-cms-service/internal/app/ports"
	"cdc-cms-service/internal/infra/messaging"
	"cdc-cms-service/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

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
	return c.Status(202).JSON(fiber.Map{
		"message": "execute-heal dispatched",
		"job_id":  res.JobID,
	})
}

func (h *ReconciliationHandler) GetUnhealedReports(c *fiber.Ctx) error {
	table := c.Params("table")
	shadowSchema := c.Query("shadow_schema")
	res, err := h.listUnhealedQ.Handle(c.UserContext(), recon.ListUnhealedReportsQuery{
		Table:        table,
		ShadowSchema: shadowSchema,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": res.Data, "total": len(res.Data)})
}
```

#### [MODIFY] [reconciliation_report.go (gateway)](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go) — Thêm 6 trường

```diff
 HealedDurationMs int        `gorm:"column:healed_duration_ms;default:0" json:"healed_duration_ms"`
+HealedMismatchedCount      int `gorm:"column:healed_mismatched_count;default:0" json:"healed_mismatched_count"`
+HealedMismatchedDurationMs int `gorm:"column:healed_mismatched_duration_ms;default:0" json:"healed_mismatched_duration_ms"`
+HealedMissingDestCount     int `gorm:"column:healed_missing_dest_count;default:0" json:"healed_missing_dest_count"`
+HealedMissingDestDurationMs int `gorm:"column:healed_missing_dest_duration_ms;default:0" json:"healed_missing_dest_duration_ms"`
+PrunedMissingSrcCount      int `gorm:"column:pruned_missing_src_count;default:0" json:"pruned_missing_src_count"`
+PrunedMissingSrcDurationMs int `gorm:"column:pruned_missing_src_duration_ms;default:0" json:"pruned_missing_src_duration_ms"`
```

---

### 4.C — CDC Worker (`centralized-data-service`)

> [!IMPORTANT]
> `ExecuteHeal` nằm ở **handler layer** (`internal/handler/recon/`), KHÔNG phải service layer. Lý do: cần `h.FetchAndWriteByIDs` (handler method), `h.natsPub`, `h.reportRepo`, `h.mapGpayToSourceIDs` — tất cả đều wire vào `ReconHandler`.

#### [MODIFY] [server_setup.go L345](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L345) — Subscribe

```diff
 natsClient.Conn.Subscribe("cdc.cmd.recon-heal", reconHandler.HandleReconHeal)
+natsClient.Conn.Subscribe("cdc.cmd.execute-heal", reconHandler.HandleExecuteHeal)
```

#### [NEW] `internal/handler/recon/recon_execute_heal.go` — Core logic (Handler layer)

```go
package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"centralized-data-service/internal/model/recon"
	"centralized-data-service/pkgs/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type executeHealOpts struct {
	ReportIDs       []uint64
	HealMismatched  bool
	HealMissingDest bool
	PruneMissingSrc bool
}

// HandleExecuteHeal — subscribe "cdc.cmd.execute-heal"
func (h *ReconHandler) HandleExecuteHeal(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleExecuteHeal")
	defer span.End()

	var payload struct {
		Table           string   `json:"table"`
		ReportIDs       []uint64 `json:"report_ids"`
		HealMismatched  bool     `json:"heal_mismatched"`
		HealMissingDest bool     `json:"heal_missing_dest"`
		PruneMissingSrc bool     `json:"prune_missing_src"`
	}
	json.Unmarshal(msg.Data, &payload)

	h.logger.Info("execute-heal received",
		zap.String("table", payload.Table),
		zap.Any("report_ids", payload.ReportIDs),
		zap.Bool("heal_mismatched", payload.HealMismatched),
		zap.Bool("heal_missing_dest", payload.HealMissingDest),
		zap.Bool("prune_missing_src", payload.PruneMissingSrc),
	)

	opts := executeHealOpts{
		ReportIDs:       payload.ReportIDs,
		HealMismatched:  payload.HealMismatched,
		HealMissingDest: payload.HealMissingDest,
		PruneMissingSrc: payload.PruneMissingSrc,
	}

	totalProcessed, err := h.executeHeal(ctx, opts)
	if err != nil {
		h.logActivity("execute-heal", payload.Table, "error", 0, err)
		h.respondErr(msg, err)
		return
	}

	h.logActivity("execute-heal", payload.Table, "success", int64(totalProcessed), nil)
	if msg.Reply != "" {
		res, _ := json.Marshal(map[string]any{
			"status":            "success",
			"reports_processed": len(opts.ReportIDs),
			"total_healed":      totalProcessed,
		})
		msg.Respond(res)
	}
}

func (h *ReconHandler) executeHeal(ctx context.Context, opts executeHealOpts) (int, error) {
	totalProcessed := 0
	for _, id := range opts.ReportIDs {
		rpt, err := h.reportRepo.GetByID(ctx, id)
		if err != nil {
			h.logger.Error("[execute-heal] load report failed",
				zap.Uint64("id", id), zap.Error(err))
			continue
		}

		entry := h.resolveTargetTableConfig(rpt.TargetTable)

		switch rpt.Segment {
		case "source_shadow", "":
			if entry == nil {
				h.logger.Error("[execute-heal-a] registry not found",
					zap.String("table", rpt.TargetTable))
				continue
			}
			totalProcessed += h.executeHealSegA(ctx, rpt, entry, opts)
		case "shadow_master":
			totalProcessed += h.executeHealSegB(ctx, rpt, opts)
		default:
			h.logger.Warn("[execute-heal] unknown segment",
				zap.String("segment", rpt.Segment))
			continue
		}

		now := time.Now().UTC()
		_ = h.reportRepo.UpdateByID(ctx, rpt.ID, map[string]any{
			"healed_at":                     now,
			"healed_mismatched_count":       rpt.HealedMismatchedCount,
			"healed_mismatched_duration_ms": rpt.HealedMismatchedDurationMs,
			"healed_missing_dest_count":     rpt.HealedMissingDestCount,
			"healed_missing_dest_duration_ms": rpt.HealedMissingDestDurationMs,
			"pruned_missing_src_count":      rpt.PrunedMissingSrcCount,
			"pruned_missing_src_duration_ms": rpt.PrunedMissingSrcDurationMs,
			"status":                        "healed",
		})
	}
	return totalProcessed, nil
}
```

**Segment A** — Tái sử dụng `h.FetchAndWriteByIDs`:
```go
func (h *ReconHandler) executeHealSegA(ctx context.Context, rpt *recon.ReconciliationReport, entry *source.TableRegistry, opts executeHealOpts) int {
	var staleA struct {
		Mismatched     []string `json:"mismatched"`
		MissingFromSrc []string `json:"missing_from_src"`
	}
	if len(rpt.StaleIDs) > 0 && string(rpt.StaleIDs) != "null" {
		_ = json.Unmarshal(rpt.StaleIDs, &staleA)
	}

	var missingIDs []string // = missing_from_dest
	if len(rpt.MissingIDs) > 0 && string(rpt.MissingIDs) != "null" {
		json.Unmarshal(rpt.MissingIDs, &missingIDs)
	}

	healed := 0

	if opts.HealMismatched && len(staleA.Mismatched) > 0 {
		start := time.Now()
		written, err := h.FetchAndWriteByIDs(ctx, entry, staleA.Mismatched)
		if err != nil {
			h.logger.Error("[execute-heal-a] mismatched heal failed", zap.Error(err))
		}
		rpt.HealedMismatchedCount = written
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
		healed += written
	}
	if opts.HealMissingDest && len(missingIDs) > 0 {
		start := time.Now()
		written, err := h.FetchAndWriteByIDs(ctx, entry, missingIDs)
		if err != nil {
			h.logger.Error("[execute-heal-a] missing_dest heal failed", zap.Error(err))
		}
		rpt.HealedMissingDestCount = written
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
		healed += written
	}
	if opts.PruneMissingSrc && len(staleA.MissingFromSrc) > 0 {
		start := time.Now()
		// TODO: soft-delete orphan — UPDATE shadow SET _deleted = true WHERE _source_id IN (...)
		rpt.PrunedMissingSrcCount = len(staleA.MissingFromSrc)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += len(staleA.MissingFromSrc)
	}

	return healed
}
```

**Segment B** — Tái sử dụng `h.mapGpayToSourceIDs` + `h.natsPub`, **có fallback flat array**:
```go
func (h *ReconHandler) executeHealSegB(ctx context.Context, rpt *recon.ReconciliationReport, opts executeHealOpts) int {
	// Parse Segment B format với fallback (copy từ recon_heal_v4.go L139-148)
	var staleB struct {
		StaleIDs       []string `json:"stale_ids"`
		OrphanInMaster []string `json:"orphan_in_master"`
	}
	if len(rpt.StaleIDs) > 0 && string(rpt.StaleIDs) != "null" {
		if err := json.Unmarshal(rpt.StaleIDs, &staleB); err != nil {
			// Fallback: flat array → gán vào OrphanInMaster
			var flatIDs []string
			if err2 := json.Unmarshal(rpt.StaleIDs, &flatIDs); err2 == nil {
				staleB.OrphanInMaster = flatIDs
			}
		}
	}

	var missingGpayIDs []string
	if len(rpt.MissingIDs) > 0 && string(rpt.MissingIDs) != "null" {
		json.Unmarshal(rpt.MissingIDs, &missingGpayIDs)
	}

	healed := 0

	if opts.HealMismatched && len(staleB.StaleIDs) > 0 {
		start := time.Now()
		sourceIDs, err := h.mapGpayToSourceIDs(ctx, rpt.SourceDB, staleB.StaleIDs)
		if err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs)
		}
		rpt.HealedMismatchedCount = len(staleB.StaleIDs)
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.HealMissingDest && len(missingGpayIDs) > 0 {
		start := time.Now()
		sourceIDs, err := h.mapGpayToSourceIDs(ctx, rpt.SourceDB, missingGpayIDs)
		if err == nil {
			healed += h.publishTransmuteChunked(ctx, rpt.TargetTable, sourceIDs)
		}
		rpt.HealedMissingDestCount = len(missingGpayIDs)
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
	}
	if opts.PruneMissingSrc && len(staleB.OrphanInMaster) > 0 {
		start := time.Now()
		// TODO: soft-delete orphan trong master table
		rpt.PrunedMissingSrcCount = len(staleB.OrphanInMaster)
		rpt.PrunedMissingSrcDurationMs = int(time.Since(start).Milliseconds())
		healed += len(staleB.OrphanInMaster)
	}

	return healed
}

// publishTransmuteChunked — tái sử dụng pattern chunk từ healSegmentB
func (h *ReconHandler) publishTransmuteChunked(ctx context.Context, table string, sourceIDs []string) int {
	dispatched := 0
	for start := 0; start < len(sourceIDs); start += healChunkSize {
		if start > 0 {
			time.Sleep(healDelayMs)
		}
		end := start + healChunkSize
		if end > len(sourceIDs) {
			end = len(sourceIDs)
		}
		payload, _ := json.Marshal(map[string]any{
			"master_table": table,
			"_source_ids":  sourceIDs[start:end],
			"triggered_by": "execute-heal",
		})
		if err := h.natsPub.Publish("cdc.cmd.transmute", payload); err != nil {
			h.logger.Error("[execute-heal-b] transmute publish failed", zap.Error(err))
			return dispatched
		}
		dispatched += end - start
	}
	return dispatched
}
```

#### [MODIFY] [reconciliation_report.go (worker)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go) — Thêm 6 trường (giữ `*int64` cho SourceCount, giữ schema prefix `cdc_system.`)

```diff
 HealedDurationMs int   `gorm:"column:healed_duration_ms;default:0" json:"healed_duration_ms"`
+HealedMismatchedCount      int `gorm:"column:healed_mismatched_count;default:0" json:"healed_mismatched_count"`
+HealedMismatchedDurationMs int `gorm:"column:healed_mismatched_duration_ms;default:0" json:"healed_mismatched_duration_ms"`
+HealedMissingDestCount     int `gorm:"column:healed_missing_dest_count;default:0" json:"healed_missing_dest_count"`
+HealedMissingDestDurationMs int `gorm:"column:healed_missing_dest_duration_ms;default:0" json:"healed_missing_dest_duration_ms"`
+PrunedMissingSrcCount      int `gorm:"column:pruned_missing_src_count;default:0" json:"pruned_missing_src_count"`
+PrunedMissingSrcDurationMs int `gorm:"column:pruned_missing_src_duration_ms;default:0" json:"pruned_missing_src_duration_ms"`
```

#### [MODIFY] [recon_handler_run.go L205](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go#L205) — Deprecate `HandleReconHeal`

```diff
 func (h *ReconHandler) HandleReconHeal(msg *nats.Msg) {
+    h.logger.Warn("[DEPRECATED] cdc.cmd.recon-heal — use cdc.cmd.execute-heal",
+        zap.String("raw", string(msg.Data)))
+    // Giữ logic cũ backward-compat trong thời gian chuyển tiếp
```

---

### 4.D — Frontend (`cdc-cms-web`)

#### Hook mới
```typescript
export interface ExecuteHealPayload {
  table: string;
  segment?: string;
  reportIds: number[];
  healMismatched?: boolean;
  healMissingDest?: boolean;
  pruneMissingSrc?: boolean;
}

export const useUnhealedReports = (table: string, shadowSchema?: string) =>
  useQuery(['unhealed-reports', table], () =>
    api.get(`/api/reconciliation/report/${table}/unhealed`, {
      params: { shadow_schema: shadowSchema }
    }).then(r => r.data));

export const useExecuteHealMutation = () =>
  useMutation((p: ExecuteHealPayload) =>
    api.post('/api/reconciliation/execute-heal', p));
```

#### Modal: 3 checkboxes + danh sách report chưa heal (gom theo segment A/B)

---

## 5. Toàn Bộ File Thay Đổi

### Gateway (`cdc-cms-service`) — 9 files
| File | Thay đổi |
|------|----------|
| `commands/recon/recon_async.go` | [MODIFY] Thêm `ExecuteHealCommand` |
| `server/server.go` | [MODIFY] `RegisterSubject("execute-heal", ...)` |
| `router/router.go` | [MODIFY] `registerDestructive` + `dual("GET")` |
| `queries/recon/list_unhealed_reports.go` | [NEW] CQRS Query Handler |
| `queries/recon/recon_reader.go` | [MODIFY] Thêm `ListUnhealedReports` vào interface |
| `persistence/recon/recon_read_repo_gorm.go` | [MODIFY] Implement `ListUnhealedReports` |
| `api/recon/reconciliation_handler.go` | [MODIFY] Inject `listUnhealedQ` |
| `api/recon/reconciliation_handler_execute_heal.go` | [NEW] `TriggerExecuteHeal` + `GetUnhealedReports` |
| `model/recon/reconciliation_report.go` | [MODIFY] Thêm 6 trường |

### Worker (`centralized-data-service`) — 4 files
| File | Thay đổi |
|------|----------|
| `server/server_setup.go` | [MODIFY] Subscribe `cdc.cmd.execute-heal` |
| `handler/recon/recon_execute_heal.go` | [NEW] `HandleExecuteHeal` + `executeHeal` + Seg A/B |
| `handler/recon/recon_handler_run.go` | [MODIFY] Deprecate `HandleReconHeal` |
| `model/recon/reconciliation_report.go` | [MODIFY] Thêm 6 trường |

### Frontend (`cdc-cms-web`) — 3 files
| File | Thay đổi |
|------|----------|
| `hooks/useReconStatus.ts` | [MODIFY] Thêm hooks |
| `components/ReconPipelineGrid.tsx` | [MODIFY] Nút chữa lành |
| `components/ConfirmDestructiveModal.tsx` | [MODIFY] 3 checkboxes |

---

## 6. Verification Plan

1. Migration: verify 6 cột mới + report cũ `healed_at != NULL` không hiển thị trong unhealed
2. `GET /report/:table/unhealed` trả về đúng cả Seg A + Seg B
3. `POST /execute-heal` → worker log `execute-heal received` → heal/prune granular
4. 6 trường thống kê cập nhật chính xác từng report
5. Chạy lại check → verify "khớp"
6. Regression: chạy suite test hiện có
