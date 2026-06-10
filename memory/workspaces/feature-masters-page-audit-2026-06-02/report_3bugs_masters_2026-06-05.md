# report_3bugs_masters_2026-06-05.md — Fix 3 bug test-tay + 2 yêu cầu mở rộng

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-05 | Phạm vi: Shadow→Master (KHÔNG đụng Source→Shadow). KHÔNG commit/push.

## 1. Bối cảnh
User test tay trang `/masters` ra 3 bug + sau đó mở rộng yêu cầu cho bug2/bug3. Tất cả đã root-cause (đọc-hiểu trước, không sửa mù), fix minimal-impact, verify exercise-driven.

## 2. Root cause + Fix + Verify (từng bug)

### BUG 1 — Nhấn Approve ở /masters kéo TOÀN BỘ field shadow sang master
- **Root cause**: `approve_master.go` khi schema-approve làm 3 việc: (1) flip schema_status; (2) auto-INSERT mọi field shadow `status='approved'`; (3) bulk `UPDATE ... SET status='approved' WHERE status IN ('pending','rejected')`. ⇒ approve schema = duyệt hết field.
- **Fix (1 best)**: step2 `'approved'`→`'pending'` (chỉ populate candidate), step3 **XOÁ**. Schema-approve chỉ mở khoá master; chọn field là per-field approve ở trang mappings. `create_master` clone vốn đã tạo `pending` nên không mất candidate.
- **Verify red→green LIVE** (cdc_dw, binding 11): set is_active=false (do constraint `v2_master_active_requires_approved`)→schema pending_review→rules pending → RED 0-approved/14-pending → POST approve API [202] → GREEN schema=approved nhưng rules **vẫn 0-approved/14-pending**.

### BUG 2 — scan-array trả new_fields:null
- **Root cause**: (a) `resolveTargetSchema("export_jobs")` mơ hồ vì 3 schema cùng có `export_jobs`; (b) `params` là **object** (xác nhận cả 3 schema: 163/456/456 rows), nên `params[*]` array-explode đúng ra rỗng → null im lặng kèm status:ok ⇒ khó hiểu.
- **Fix (1 best)** `HandleScanArrayFields`: (a) khi có `master_binding_id` → resolve shadow_schema+table CHÍNH XÁC từ binding (hết ambiguity); (b) probe `jsonb_typeof(_raw_data #> path)='array'` → trả `status:"no_array"` thay vì null im lặng.
- **Verify**: scan-array binding 11 → `{"status":"no_array"}` [200].

### BUG 2 (mở rộng) — API "Scan Array All" + nút
- **Thêm**: endpoint `POST /api/introspection/scan-array-all {master_binding_id}` → lấy field shadow approved+in-shadow (mapping_rule_v2 status='approved' AND is_active) của binding → quét từng field như mảng (reuse worker `cdc.cmd.scan-array`, KHÔNG thêm worker handler) → gộp kết quả. FE thêm nút **"Scan Array All (Flatten)"** ở trang mappings.
- **Verify**: curl binding 11 → `{"scanned":2,"array_fields":0,"results":[{_id,no_array},{params,no_array}],...}` [200] (đúng: không field nào là array).

### BUG 3 — Thêm record source→shadow OK nhưng master không có (mode realtime)
- **Root cause**: chain realtime CODE đúng (kafka→`SinkWorker.HandleMessage`→upsert+`publishTransmuteTrigger`→gate `hasPostIngestSchedule`→`cdc.cmd.transmute-shadow`→per-master transmute). Triệu chứng "shadow có, master rỗng" = chính **bug ext-JSON** (transmute upsert chết `{"$date"}`/`{"$oid"}`/jsonb/epoch → 0 dòng tới master). Đã fix trước đó (`unwrapMongoExtJSON`+`coerceForColumn` trong `transmuter.go`); **bằng chứng**: export_jobs_mt synced 163=shadow, data đúng (createdAt timestamp, params/_id jsonb).
- **Fix (mở rộng)**: FE `/masters` tách Sync 1-nút thành **3 control độc lập**: Realtime (Switch→post_ingest), Chạy ngay (Button→immediate+run-now), Hẹn giờ (Switch+Input cron). Reuse API schedules (Create upsert ON CONFLICT DO UPDATE is_enabled, RunNow, List seed state).
- **Verify backend**: toggle realtime OFF→is_enabled=f, ON→t; cron toggle→created enabled cron_expr (curl [201]).

## 3. Files thực tế đã sửa + LOC (git diff --stat, KHÔNG chế số)
> Số LOC là **cộng dồn uncommitted** của cả feature (không commit theo lệnh User), đã chú thích file thuộc 3 bug turn này.

**centralized-data-service** (worker):
- `internal/handler/command_handler.go` — +241 (BUG2 scan-array scoping+no_array; cùng file có SQL ext-json batch path)
- `internal/service/transmuter.go` — +338/-… (BUG3 ext-json unwrap/coerce + SAFE-2 cache; bao gồm fix các phiên trước)
- (worker khác trong diff: transmute_handler.go, master_ddl_generator.go, sinkworker.go, worker_server.go, type_resolver.go — feature trước)

**cdc-cms-service** (CMS):
- `internal/app/commands/approve_master.go` — +73/-… (BUG1 step2→pending, step3 xoá)
- `internal/api/introspection_handler.go` — +150 (BUG2 ScanArrayAll endpoint)
- `internal/router/router.go` — +17 (route scan-array-all + các route feature trước)
- `internal/app/commands/create_mapping_rule.go` — +31 (G2 path-1 shadow-scope, phiên trước)

**cdc-cms-web** (FE):
- `src/pages/MasterRegistry.tsx` — +185 (BUG3 3-way sync: realtime/run-now/cron)
- `src/pages/MasterMappingFieldsPage.tsx` — **untracked (file mới)**, thêm ~35 dòng (BUG2 nút Scan Array All + handler + state)

## 4. Build / Deploy (G3 — không thối code)
- `go build ./...` CMS=0, worker=0; `go vet` sạch (idgen warning pre-existing, không trong diff).
- FE `npx tsc -b` = 0.
- Restart CMS (55032→…) + worker (55104→…), mỗi lần confirm PID mới + no-fatal TRƯỚC khi exercise (theo lesson zombie-process). FE Vite hot-reload.

## 5. State để lại (disclose)
- b3: is_active=true, 454 rows, schedule id=2 (từ e2e trước).
- export_jobs_mt (binding 11): is_active=true (restore sau test Bug1), schema=approved, 14 mapping_rule_master = **pending** (do test Bug1 reset — đúng hành vi mới), post_ingest+immediate schedule. Đã xoá test cron.

## 6. Ràng buộc tuân thủ
- KHÔNG đụng Source→Shadow (upsert.go/kafka_consumer write path; chỉ sửa transform master-side + config/discovery + FE).
- KHÔNG cheat DB (toggle/approve qua API thật + dev-token=auth dev có sẵn).
- KHÔNG commit/push.
