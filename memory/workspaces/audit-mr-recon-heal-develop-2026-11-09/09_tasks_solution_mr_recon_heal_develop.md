---
tier: T2
mode: standard
workspace: audit-mr-recon-heal-develop-2026-11-09
suffix: mr_recon_heal_develop
---

# 09. Technical Solution & Remediation Plan — MR `recon-heal` -> `develop`

## 1. Tóm tắt Kiến trúc Thay đổi trong MR (`origin/develop..recon-heal`)
- **Repo 1 (`centralized-data-service`):** `59 commits`, `316 files changed (+49,047 / -11,549)`, `0` git merge conflict với `origin/develop` (`b0ff47c` -> `9d41783` fast-forward).
  - Bổ sung kiến trúc Multi-Database Plane (`pkgs/database/multi.go`, `multi_dialector.go`, `pgx_exec_mode.go`) hỗ trợ PostgreSQL (`pgx` simple protocol tránh lỗi prepared statement cache), MySQL/MariaDB, ClickHouse và MongoDB.
  - Nâng cấp luồng Reconciliation & Execute-Heal Segment A (`source_shadow`) và Segment B (`shadow_master`): hỗ trợ đối soát & tự động sửa chữa đa dialect (`recon_execute_heal_handler.go`, `recon_execute_heal_segb.go`, `recon_execute_heal_segb_sql_dialects.go`, `recon_heal_fetch.go`, `recon_heal_postgres.go`, `recon_heal_route.go`).
  - Nâng cấp `ShadowBatchBuffer` (`batch_buffer.go`, `batch_buffer_fanout.go`, `batch_buffer_dlq.go`) và `TransmuteHandler` (`transmute_handler.go`, `transmute_poison_pill_split.go`) với cơ chế dynamic conflict target, cô lập poison-pill và fan-out đa Shadow/Master binding.
- **Repo 2 (`cdc-cms-service`):** `34 commits`, `191 files changed (+13,654 / -1,467)`, `0` git merge conflict với `origin/develop` (`22eaa94` -> `103e708` fast-forward).
  - Mở rộng API/CQRS cho Reconciliation (`TriggerExecuteHeal`, `GetUnhealedReports`, `GetActiveReconJobs`), Daily Flow Metrics (`GetDailyFlow`), Multi-Master/Shadow Binding và hệ thống SQL migrations (`migrations/schema/**/*.sql`).

---

## 2. Ràng buộc Kỹ thuật (Technical Constraints từ Lessons)
1. **L049 (Stub/Mock & Test Hygiene):** Mọi thay đổi chữ ký interface (`ReconReader`, `ActivityLogReader`) hoặc struct trong `internal/` phải đồng bộ toàn bộ test stubs tại `test/`; không commit file nháp thiếu định nghĩa làm gãy `go test ./...`.
2. **Rule #20 (Secret & Production Package Hygiene):** Cấm commit file test/debug chứa hardcoded DSN credentials vào package production (`internal/service/source`).
3. **L023 (Idempotent & Sequential Schema Migrations):** Mọi file migration trong `migrations/schema/**/*.sql` được `internal/migrate/runner.go` sắp xếp toàn cục theo `path.Base(p)` (`001_...` -> `104_...`); do đó migration số nhỏ hơn không được truy cập trực tiếp bảng/schema chỉ mới được tạo ở migration số lớn hơn mà thiếu nhánh kiểm tra tồn tại (`to_regclass`).
4. **L065 (Phantom Drift Guard trên Segment B Prune):** Khi thực thi `PruneMissingSrc` ở Segment B (`shadow_master`), bắt buộc kiểm tra đối chiếu với bảng Shadow (`NOT COALESCE("_deleted", false)`) trước khi xóa cứng trên Master để tránh xóa nhầm các bản ghi còn sống bị báo thiếu giả do lệch cửa sổ thời gian (`_source_ts`).

---

## 3. Phương án Khắc phục Duy nhất (Ánh xạ 1-1 tới Findings `F1`–`F5`)

### 3.1. Khắc phục `F1` (`P1 - High` — `centralized-data-service`): Xóa file debug chứa credential lọt vào production package `internal/service/source/test_routing2.go`
- **Root Cause (`VERIFIED`):** File `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/test_routing2.go:1-43` khai báo `package source` (không có hậu tố `_test.go`), export hàm `TestRoutingLogic()` chứa hardcoded PostgreSQL DSN tại dòng 18 (`postgres://***:***@localhost:5433/cdc_dw?sslmode=disable`) và `log.Fatalf` tại dòng 21, 36 vào binary production.
- **Giải pháp Duy nhất:** Xóa bỏ hoàn toàn file nháp `internal/service/source/test_routing2.go` khỏi git tree (`git rm internal/service/source/test_routing2.go`).

```go
// Code Demo F1 — Xóa file /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/test_routing2.go:1-43
// Lệnh thực thi sau khi có APPROVE:
// git -C /Users/trainguyen/Documents/work/data-hub/centralized-data-service rm internal/service/source/test_routing2.go
```

---

### 3.2. Khắc phục `F2` (`P1 - High` — `cdc-cms-service`): Bọc kiểm tra tồn tại bảng/schema (`to_regclass`) cho các migration `023`..`026` và补 sung cột vào `036`/`037` để không sập Fresh DB Bootstrap
- **Root Cause (`VERIFIED`):**
  - `internal/migrate/runner.go:140-148` gom tất cả file `.sql` trong mọi thư mục con của `migrations/schema/` và sort theo `path.Base(schema[i].path)`.
  - Các file mới `migrations/schema/worker/023_add_last_trace_id_to_transmute_schedule.sql:2`, `024_enhance_worker_schedule.sql:10`, `025_create_daily_source_counts.sql:12`, `026_fix_worker_schedule_unique_scope.sql:12` chạy ở thứ tự `023..026`, nhưng bảng `cdc_system.transmute_schedule` chỉ được tạo ở `036_v2_transmute_schedule.sql:5` và bảng `public.cdc_worker_schedule` (tạo ở `007_worker_schedule.sql:21`) chỉ được chuyển sang schema `cdc_system` ở `037_move_system_tables_to_cdc_system.sql:21`.
  - Khi chạy trên DB trống (Fresh Bootstrap / CI / Staging mới), `023`..`026` chạy trước `036` và `037` gây lỗi `SQLSTATE 42P01 (relation does not exist)`.
- **Giải pháp Duy nhất:**
  1. Giữ nguyên tên file `023`..`026` để bảo toàn tương thích ngược 100% với bảng `cdc_system.schema_migrations` trên các môi trường đã chạy, nhưng bọc `DO $$ ... END $$` kiểm tra `to_regclass('cdc_system.transmute_schedule')` / fallback giữa `cdc_system.cdc_worker_schedule` và `public.cdc_worker_schedule`.
  2. Trong `036_v2_transmute_schedule.sql` bổ sung `ALTER TABLE cdc_system.transmute_schedule ADD COLUMN IF NOT EXISTS last_trace_id VARCHAR(64) DEFAULT NULL;` để khi bootstrap từ đầu (`023` bỏ qua vì bảng chưa tạo), tới bước `036` cột `last_trace_id` vẫn được tạo đầy đủ.

```sql
-- Code Demo F2.1 — /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/023_add_last_trace_id_to_transmute_schedule.sql:1-6
DO $$
BEGIN
  IF to_regclass('cdc_system.transmute_schedule') IS NOT NULL THEN
    ALTER TABLE cdc_system.transmute_schedule
      ADD COLUMN IF NOT EXISTS last_trace_id VARCHAR(64) DEFAULT NULL;
    COMMENT ON COLUMN cdc_system.transmute_schedule.last_trace_id IS
      'OpenTelemetry trace ID của lần chạy gần nhất, dùng để trace trong SigNoz/Jaeger.';
  END IF;
END $$;
```

```sql
-- Code Demo F2.2 — /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/worker/024_enhance_worker_schedule.sql:10-35
DO $$
DECLARE
  target_rel text := CASE
    WHEN to_regclass('cdc_system.cdc_worker_schedule') IS NOT NULL THEN 'cdc_system.cdc_worker_schedule'
    WHEN to_regclass('public.cdc_worker_schedule') IS NOT NULL THEN 'public.cdc_worker_schedule'
    ELSE NULL
  END;
BEGIN
  IF target_rel IS NOT NULL THEN
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS last_error TEXT DEFAULT NULL', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS last_run_duration_ms BIGINT DEFAULT NULL', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS run_count BIGINT NOT NULL DEFAULT 0', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS fail_count BIGINT NOT NULL DEFAULT 0', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMPTZ DEFAULT NULL', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS locked_by VARCHAR(100) DEFAULT NULL', target_rel);
    EXECUTE format('ALTER TABLE %s ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ DEFAULT NULL', target_rel);
  END IF;
END $$;
```

---

### 3.3. Khắc phục `F3` (`P1 - High` — `centralized-data-service`): Bổ sung Active-Shadow Phantom-Drift Check cho nhánh `effectivePruneIDType == "gpay"` trong `pruneSQLDialectMasterMissingFromSrc`
- **Root Cause (`VERIFIED`):** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb_sql_dialects.go:130-146`, khi `effectivePruneIDType == "gpay"`, hàm xóa thẳng `execDeleteByCol("`_gpay_id`", chunk, len(chunk))` mà không đối chiếu bảng Shadow (`h.shadowDB`) như nhánh PostgreSQL (`recon_execute_heal_segb.go:188-255`), MongoDB (`recon_execute_heal_segb.go:467-552`) và nhánh `id` (`recon_execute_heal_segb_sql_dialects.go:153-180`).
- **Giải pháp Duy nhất:** Kiểm tra `h.shadowDB` (`SELECT "_gpay_id"::text FROM <shadowRel> WHERE "_gpay_id"::text IN ? AND NOT COALESCE("_deleted", false)`) trước khi xóa; các `_gpay_id` còn sống trên Shadow thì gọi `h.triggerMasterTransmuteByGpayIDs(ctx, rpt, aliveGpayIDs)` để resync, chỉ xóa các `trulyOrphanGpayIDs`.

```go
// Code Demo F3 — /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_segb_sql_dialects.go:130-214
	if effectivePruneIDType == "gpay" {
		var gpayIDs []int64
		for _, s := range missingIDs {
			if val, err := strconv.ParseInt(s, 10, 64); err == nil {
				gpayIDs = append(gpayIDs, val)
			}
		}
		var resyncSourceIDs []string
		liveGpayCount := 0
		for i := 0; i < len(gpayIDs); i += batch {
			end := i + batch
			if end > len(gpayIDs) {
				end = len(gpayIDs)
			}
			chunkGpay := gpayIDs[i:end]
			toDeleteGpay := chunkGpay
			if h.shadowDB != nil && pkCol != "" && pkCol != "_gpay_id" && shadowRel != "" {
				type masterGpayPK struct {
					GpayID int64  `gorm:"column:_gpay_id"`
					PKVal  string `gorm:"column:pk_val"`
				}
				var castExpr string
				masterSelectRel := qualifiedMaster
				if isCH {
					masterSelectRel += " FINAL"
					castExpr = fmt.Sprintf("toString(%s)", quoteHealDialectIdent(pkCol))
				} else {
					castExpr = fmt.Sprintf("CAST(%s AS CHAR)", quoteHealDialectIdent(pkCol))
				}
				var mRows []masterGpayPK
				errM := masterDB.WithContext(ctx).Raw(
					fmt.Sprintf("SELECT `_gpay_id`, %s AS pk_val FROM %s WHERE `_gpay_id` IN (?)", castExpr, masterSelectRel),
					chunkGpay,
				).Scan(&mRows).Error
				if errM == nil && len(mRows) > 0 {
					pkCandidates := make([]string, 0, len(mRows))
					for _, mr := range mRows {
						if mr.PKVal != "" {
							pkCandidates = append(pkCandidates, mr.PKVal)
						}
					}
					if len(pkCandidates) > 0 {
						var activeShadowIDs []string
						errS := h.shadowDB.WithContext(ctx).Raw(
							fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE "_source_id" IN (?) AND NOT COALESCE("_deleted", false)`, qualifiedShadow),
							pkCandidates,
						).Scan(&activeShadowIDs).Error
						if errS == nil && len(activeShadowIDs) > 0 {
							activeSet := make(map[string]bool, len(activeShadowIDs))
							for _, sid := range activeShadowIDs {
								activeSet[sid] = true
							}
							liveGpaySet := make(map[int64]bool)
							for _, mr := range mRows {
								if activeSet[mr.PKVal] {
									liveGpaySet[mr.GpayID] = true
									resyncSourceIDs = append(resyncSourceIDs, mr.PKVal)
								}
							}
							if len(liveGpaySet) > 0 {
								filtered := make([]int64, 0, len(chunkGpay))
								for _, gid := range chunkGpay {
									if !liveGpaySet[gid] {
										filtered = append(filtered, gid)
									}
								}
								liveGpayCount += len(chunkGpay) - len(filtered)
								toDeleteGpay = filtered
							}
						}
					}
				}
			}
			if len(toDeleteGpay) > 0 {
				pruned += execDeleteByCol("`_gpay_id`", toDeleteGpay, len(toDeleteGpay))
			}
		}
		if len(resyncSourceIDs) > 0 {
			resyncSourceIDs = uniqueStrings(resyncSourceIDs)
			if _, err := h.publishTransmuteChunked(ctx, rpt.TargetTable, resyncSourceIDs, "execute-heal-b", masterBindingID); err == nil {
				pruned += liveGpayCount
			}
		}
		return pruned
	}
```

---

### 3.4. Khắc phục `F4` (`P1 - High` — Cả 2 Repo): Sửa lỗi biên dịch Test Suite trong `test/` (`recon_pipeline_span_test.go` và `queries_test.go`)
- **Root Cause (`VERIFIED`):**
  1. Ở `centralized-data-service`: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/test/internal/service/recon_pipeline_span_test.go:11-134` gọi `servicerecon.PipelineFlowInfo` và `FormatSpanName()`, nhưng struct `PipelineFlowInfo` chưa được định nghĩa trong `internal/service/recon/recon_models.go`.
  2. Ở `cdc-cms-service`: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/app/queries/queries_test.go:337` khai báo `GetActiveReconJobs(context.Context, string)` thay vì `(context.Context, []string)` (`recon_reader.go:136`), và `stubActivityLogReader` (`queries_test.go:624-643`) thiếu method `GetDailyFlow(context.Context, systemQueries.DailyFlowParams) (*systemQueries.DailyFlowResult, error)` (`list_activity_logs.go:44`).
- **Giải pháp Duy nhất:**
  1. Bổ sung `PipelineFlowInfo` và `FormatSpanName()` vào `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_models.go`.
  2. Cập nhật chữ ký `stubReconReader.GetActiveReconJobs` và bổ sung `stubActivityLogReader.GetDailyFlow` trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/app/queries/queries_test.go`.

```go
// Code Demo F4.1 — Bổ sung vào /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_models.go
type PipelineFlowInfo struct {
	PipelineKey         string
	SourceEngine        string
	SourceConnectionKey string
	SourceDB            string
	SourceTable         string
	ShadowSchema        string
	ShadowTable         string
	MasterConnectionKey string
	MasterSchema        string
	MasterTable         string
}

func (p *PipelineFlowInfo) FormatSpanName() string {
	if p == nil {
		return "pipeline: unknown"
	}
	var stations []string
	if p.SourceTable != "" {
		eng := strings.ToLower(strings.TrimSpace(p.SourceEngine))
		var label string
		switch eng {
		case "mongodb", "mongo":
			label = "Mongo"
		case "postgresql", "postgres":
			label = "Postgres"
		case "mysql", "mariadb":
			label = "MySQL"
		case "":
			label = "Source"
		default:
			label = p.SourceEngine
		}
		stations = append(stations, fmt.Sprintf("[%s:%s.%s.%s]", label, p.SourceConnectionKey, p.SourceDB, p.SourceTable))
	}
	if p.ShadowTable != "" {
		stations = append(stations, fmt.Sprintf("[Shadow:%s.%s]", p.ShadowSchema, p.ShadowTable))
	}
	if p.MasterTable != "" {
		stations = append(stations, fmt.Sprintf("[Master:%s:%s.%s]", p.MasterConnectionKey, p.MasterSchema, p.MasterTable))
	}
	return "pipeline: " + strings.Join(stations, " -> ")
}
```

```go
// Code Demo F4.2 — Cập nhật /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/app/queries/queries_test.go:337-339 & 641-644
func (s *stubReconReader) GetActiveReconJobs(context.Context, []string) ([]reconQueries.ActiveReconJob, error) {
	return nil, nil
}
// ...
func (s *stubActivityLogReader) GetDailyFlow(context.Context, systemQueries.DailyFlowParams) (*systemQueries.DailyFlowResult, error) {
	return &systemQueries.DailyFlowResult{}, s.err
}
```

---

### 3.5. Khắc phục `F5` (`P2 - Medium` — `centralized-data-service`): Chặn Fallthrough sai sang MongoDB khi `entry.SourceType` không phải MongoDB/PostgreSQL trong `FetchAndWriteByIDsWithAccepted`
- **Root Cause (`VERIFIED`):** Tại `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_fetch.go:33-56`, `FetchAndWriteByIDsWithAccepted` chỉ kiểm tra `if isPostgresSourceEntry(entry)` rồi mặc định coi mọi nguồn còn lại là MongoDB (`h.reconCore.GetMongoClientForURL`). Nếu pipeline nguồn là MySQL/MariaDB và `entry.SourceURL` rỗng (fallback về `rc.mongoClient` mặc định), hàm sẽ truy vấn nhầm vào MongoDB, trả về `0` bản ghi (`err == nil`), khiến `recon_execute_heal_handler.go:286-298` hiểu nhầm toàn bộ IDs bị thiếu ở Source và thực hiện **xóa mềm (soft-delete) nhầm trên Shadow & Master**.
- **Giải pháp Duy nhất:** Kiểm tra rõ `isMongoSourceEntry(entry)` trước khi rơi vào nhánh MongoDB; nếu không phải Postgres hay Mongo thì trả lỗi tường minh để `recon_execute_heal_handler.go:280` giữ nguyên dữ liệu (không prune nhầm).

```go
// Code Demo F5 — /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_fetch.go:33-38
	if isPostgresSourceEntry(entry) {
		return h.fetchAndWritePostgresByIDs(ctx, entry, ids)
	}
	if st := strings.ToLower(strings.TrimSpace(entry.SourceType)); st != "" && st != "mongo" && st != "mongodb" {
		return 0, nil, fmt.Errorf("execute-heal fetch by ids: unsupported source_type %q for table %s", entry.SourceType, entry.TargetTable)
	}
```
