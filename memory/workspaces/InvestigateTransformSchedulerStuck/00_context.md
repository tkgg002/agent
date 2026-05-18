# 00_context — InvestigateTransformSchedulerStuck

## Trigger (user paste, 2026-05-13 ~13:00 local)
> kiểm tra Chuyển đổi field (Transform)
> transform | centralized-export-service.export-jobs |
> shadow_centralized_export_service.sd_export_jobs | 1 |
> 13/05/2026 09:38:22 | 13/05/2026 09:39:22 | 1 | OK
> — sao nó ko chạy

## Câu hỏi user
UI/list activity hiển thị transform của `sd_export_jobs` lúc 09:38:22 với
`status=OK rows=1`, nhưng user khẳng định worker thực sự không chạy
transform. Yêu cầu: tài liệu hoá nguyên nhân.

## Surface
- Service: `centralized-data-service` worker (port 8082).
- File code: `internal/server/worker_server.go`
  → `runTransformCycle` (line 695-716) — scheduler dispatch.
  → schedule poller goroutine (line 575-675).
  → `runScheduleTick` gating logic (line 624-633).
- Bảng DB:
  - `cdc_system.cdc_worker_schedule` (control plane 5433)
  - `cdc_system.cdc_activity_log` (partitioned)
  - `shadow_centralized_export_service.sd_export_jobs` (shadow plane 5436)

## Bối cảnh
- Sáng cùng ngày đã fix 5 bug `cmd-batch-transform` (workspace
  `FixBatchTransformV2Repo`): V1→V2 repo, h.db→h.shadowDB, quote ident,
  JSONB cast, epoch-ms timestamp.
- Sau fix, manual `nats pub cdc.cmd.batch-transform sd_export_jobs` →
  activity_log id=34 `success rows=129`. Worker handler hoạt động đúng.
- Nhưng schedule tự động (id=6 interval=1min) chỉ chạy 1 lần lúc 09:38
  rồi không tick lại suốt ~3.5 giờ kế.

## Constraint từ user
- Đọc lessons trước.
- Theo `agent/GEMINI.md` (role/skill).
- Chỉ document theo đúng yêu cầu (KHÔNG fix code lần này).
- Số liệu phải thực, query DB chứng minh.
- Service phải healthy mới báo done.
- Bắt buộc có `report_*.md`.

## Definition of Done
- Tài liệu hoá rõ 2 root cause độc lập + bằng chứng từ DB.
- Có experiment reproduce (set lastRunAt=NULL → schedule fire).
- Verify worker `{"status":"ok"}` post-investigation.
- File `report_*.md` lưu thay đổi (chỉ là DB experiment, không touch code).
