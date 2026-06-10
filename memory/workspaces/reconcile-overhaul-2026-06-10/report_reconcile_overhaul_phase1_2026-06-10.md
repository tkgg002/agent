# report_reconcile_overhaul_phase1_2026-06-10.md

> **Task**: Phân tích tổng quan + review + fix/nâng cấp Reconcile.
> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-10
> **Workspace**: `reconcile-overhaul-2026-06-10`

## 1. Tổng quan Reconcile (kết quả phân tích)
Hệ thống recon **thiết kế tốt** (3-tier: count_windowed 15-min / XOR hash_window / 256-bucket off-peak; 4 lớp lock: Redis leader + scheduler lock + pg advisory + unique index; circuit breaker + rate limit + audit batcher chống spam đã có từ fix L459). **NHƯNG chưa từng chạy được trên kiến trúc hybrid Path B**: `recon_runs`=0, `report`=0 rows từ trước tới nay; trigger thật trả `success tables_checked=0` (false-positive im lặng); schedule reconcile bị disable.

## 2. Root cause (3 lớp — code V1 `public@5433` vs data V2 `shadow_*@5436`)
1. `synthesizeLegacyTableRegistry` không mang `shadow_binding.shadow_schema` → entry chỉ có tên bảng trần.
2. `CheckAll` gate `GetSchema(t)` = introspect `public` trên **control-plane 5433** → nil mọi bảng V2 → skip 100% → "success 0".
3. `ReconDestAgent` nhận `db` 5433 + mọi SQL `FROM "table"` không schema-qualify → kể cả bỏ gate vẫn trỏ sai chỗ.

## 3. Giải pháp đã thực thi (1 giải pháp duy nhất, backward-compat V1)
Làm recon **shadow-plane-aware**: entry registry mang `ShadowSchema` (synthetic, `gorm:"-"`, không đổi schema DB); dest-side nhận `shadowDB` + adapter introspect riêng trên shadow plane; SQL quote dạng `"schema"."table"` qua `quoteRelation` (tên trần → hành vi cũ → V1 không đổi); visibility: 0-tables-checked = `warning` + Warn log kèm `fix_hint` (hết false-positive im lặng).
**Cô lập pipeline tuân thủ**: KHÔNG đụng adapter/batchBuffer/sinkworker của luồng source→shadow (tạo `reconSchemaAdapter` riêng).

## 4. Files THỰC TẾ đã sửa (theo `git diff --stat`)
### centralized-data-service — 7 files, **+86 / −22 dòng**
| File | +/- | Nội dung |
|---|---|---|
| `internal/model/table_registry.go` | +17 | +`ShadowSchema` (gorm:"-") + `QualifiedTarget()` |
| `internal/service/metadata_registry_service.go` | +1 | synthesize set `ShadowSchema` |
| `internal/service/recon_dest_agent.go` | +26/− | +`quoteRelation()`; 7 vị trí table quote đổi sang nó |
| `internal/service/recon_core.go` | +34/− | gate `GetSchemaInSchema(shadow|public)`; 6 call sites `QualifiedTarget()`; đếm skip + Warn 0-checked |
| `internal/server/worker_server.go` | +15/− | destAgent nhận `shadowDB`×2; `reconSchemaAdapter` riêng |
| `internal/handler/recon_handler.go` | +11/− | CheckAll 0 tables → status `warning` |
| `internal/handler/scan_array_path_test.go` | 2 dòng | **PRE-EXISTING compile error** (test 2-return vs hàm 3-return — không thuộc diff của tôi trước đó); buộc sửa để `go test ./internal/handler` chạy được |

### cdc-cms-web — 1 file, **+3 / −1**
| File | Nội dung |
|---|---|
| `src/pages/DataIntegrity.tsx` | fix invalidate key orphan `recon-status` → `recon-report` (Overview không refresh sau backfill) |

## 5. Verify (bằng chứng thật, không chế số)
| Bước | Kết quả |
|------|---------|
| `go build ./...` | PASS |
| `go vet` service/handler/server/model | PASS (sau khi sửa test pre-existing) |
| `go test ./internal/handler/... ./internal/service/...` | **ok** handler 0.714s, service 0.888s, transmute cached |
| `npm run build` FE | PASS (443ms) |
| Restart worker | kill PID 93684 → chờ port 8082 free (L1918) → `/tmp/cdc-worker-recon` PID 9699; log: `shadow_plane=...5436...cdc_shadow` connected, ReconCore initialized |
| **E2E**: NATS `cdc.cmd.recon-check {"tier":"1","table":"*"}` | `recon_runs`: **4 rows success, 672 windows/bảng**; `cdc_reconciliation_report`: `export_jobs` **drift** (src=20, dest=0, stale=1), `export_jobs_test` drift, `wallet_capsets_1/2` ok |

→ Trước fix: 0 runs, 0 report, "success tables_checked=0". Sau fix: recon chạy thật trên shadow plane và **phát hiện drift thật**.

## 6. Phát hiện thêm trong lúc verify (visibility hoạt động)
- Connection `default_master` (postgresql) không resolve được DSN → các source PG bound vào nó bị skip có WARN rõ ràng (cần khai DSN cho connection này nếu muốn recon PG source).
- `cdc_worker_schedule.reconcile` đang `is_enabled=false` — muốn recon tự động 30' cần bật lại (quyết định vận hành của Boss, tôi không tự bật).

## 7. Gap còn lại (phase sau — đã log, KHÔNG giấu)
- **Heal path**: healer/backfill/ts-detector disabled khi không có default Mongo client (cần refactor per-source như check path); `extractSourceTsFromDoc` hardcode `updated_at` → OCC bypass cho camelCase (G10, lesson L951/1875).
- Tier3→Tier2 fallback re-acquire advisory lock (G1); `missing IDs` accumulate/ListIDsInWindow không cap (G3/G5); Tier2 drill-down lỗi bị `continue` → false-negative (G8).
- CMS/FE: job_id không poll, `X-Action-Reason` không consume, TableHistory route mồ côi, filter failed-logs UI (GAP2/3/4/8).

## 8. Trạng thái services sau task
- Worker: PID 9699 (`/tmp/cdc-worker-recon`, binary mới) — RUNNING, đã thay process `go run` cũ (93684).
- CMS 8083 + FE 5173: không đổi binary, không cần restart (FE chỉ đổi source, dev server tự reload; bản build dist đã PASS).
