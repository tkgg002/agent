# 10_gap_analysis_phaseA.md — Blueprint vs Source THỰC TẾ (Master/Transmute)

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-04
> **Phương pháp**: 3 subagent audit song song (FE/API/Worker), read-only, đối chiếu `file:line` repo `data-hub/`.
> **Kết luận 1 dòng**: Blueprint **~85% đã có thật** trong code; phần lõi (3 trang FE, API G1-G3, Worker E1-E5) đều LIVE. Gap còn lại chủ yếu là **CRUD chưa đủ + UX hợp nhất** + **2 rủi ro chất lượng/bảo mật (RLS, OCC) phải đóng trước prod**.

---

## A. Bảng coverage tổng (verdict theo blueprint)

### FE — `cdc-cms-web` (route thật: `/masters`, `/masters/:id/mappings`, `/schedules`)
| Blueprint | Verdict | Evidence |
|-----------|---------|----------|
| MasterRegistry: list + Create Binding + Approve/Reject DDL + dual-control reason≥10 | ✅ | `MasterRegistry.tsx:413,444,348-363,215-222` |
| Sync Mode Selector 3 mode (run_now/cron/post_ingest) | ✅ | `MasterRegistry.tsx:619-649` (Sync Modal per row) |
| MasterMapping: list + In Shadow + In Master + Source Data Type | ✅ | `MasterMappingFieldsPage.tsx:351-498,445-481,365-368` |
| Sync từ Shadow (pull schema) + Scan Array Flatten | ✅ | `:558-565` (`sync-from-shadow`), `:554-557` (`scan-array`) |
| Create Mapping thủ công | 🟡 | `:593-636` — **thiếu field `transform_fn`** |
| Pending review Approve/Reject | 🟡 | `:534-540` batch OK; **thiếu modal đơn lẻ per-row** |
| Sync Schedule: CRUD + Run-now + status widget | 🟡 | `TransmuteSchedules.tsx:54,101-117,150-157` — **thiếu Edit + Delete** |
| Bonus có sẵn (ngoài blueprint) | ✅ | Atomic Swap, Toggle Active, inline data_type edit, shadow-approval gate checkbox |

### API — `cdc-cms-service` (route thật khác blueprint: `/api/v1/masters`, `/master-mapping-rules`, `/schedules`)
| Blueprint | Verdict | Evidence |
|-----------|---------|----------|
| G1 Create/List/Approve/Reject/Toggle/Swap | ✅ | `router.go:220-225,360`; param là `:name`=`master_table` |
| G1 PUT update binding | 🔴 | Không tồn tại endpoint update binding |
| G2 List mapping + in_master + source_data_type + Save + Batch + SyncFromShadow + MasterColumns | ✅ | `master_mapping_rule_handler.go:31,66,189,136,282` |
| G2 field `in_shadow` | 🟡 | Model **không có `in_shadow`**; chỉ có `in_master` + `shadow_status` (JOIN v2). FE tự suy ra |
| G2 `scan-flatten` (1 endpoint hợp nhất) | 🔴 | Không có namespace master; thực tế tách 2 bước: `introspection/scan-array/:table` (shadow) → `sync-from-shadow` |
| G3 Create/List/Toggle(PATCH)/RunNow | ✅ | `transmute_schedule_handler.go:77,60,138,168` (run-now publish `cdc.cmd.transmute`) |
| G3 PUT/DELETE schedule | 🔴 | Chỉ PATCH toggle; **không sửa cron/mode, không xoá** |
| G4 `GET /sync/status` hợp nhất | 🔴 | Rải rác 3 chỗ: `sync/health` (aggregate), `schedules.last_status`, `registry/:id/dispatch-status` |

### Worker — `centralized-data-service`
| Blueprint | Verdict | Evidence |
|-----------|---------|----------|
| E1 gjson + transform_fn + gate chain + exclude system cols | ✅ | `transmuter.go:479,501-510,143-154,582-598` |
| E1 OCC theo `_source_ts` | 🟡 | `transmuter.go:546-557` master upsert dùng `_hash IS DISTINCT FROM` **(content-dedup, KHÔNG theo `_source_ts`)** |
| E1 gate ≥1 approved rule | 🟡 | `transmuter.go:167-171` zero rules → success yên lặng (soft gate) |
| E2 cron tick 60s + fencing + FOR UPDATE SKIP LOCKED | ✅ | `transmute_scheduler.go:52,101-106,109-120` |
| E2 immediate run-now + post_ingest + gate `hasPostIngestSchedule` | ✅ | `transmute_handler.go:155`; `sinkworker.go:239,254,285-326` (cache 30s, **fail-open** khi DB lỗi) |
| E3 introspection shadowDB + scan-flatten đúng DB | ✅ | `command_handler.go:976-996,1971-2119` query trên `h.shadowDB` (không sai DB) |
| E4 Master DDL Apply + typed whitelist | ✅ | `master_ddl_generator.go:180-230,116-122` (`IsTypeWhitelisted` cả CREATE+ALTER) |
| E4 RLS PostgreSQL | 🟡 | `master_ddl_generator.go:216-219` chỉ khi `MasterSchema=="public"`; policy `USING(true)` permissive; schema khác **bỏ RLS** |
| E5 publish `transmute.completed` + JobMonitor close-loop | ✅ | `transmute_handler.go:214,252`; `job_monitor.go:85-99` (idempotent `WHERE last_status='running'`) |
| E5 close-loop cho mọi trigger | 🟡 | `job_monitor.go:75-79` `if ScheduleID==0 return` → realtime/manual **không cập nhật last_status** |

---

## B. Danh sách GAP có severity (đầu vào cho Muscle)

| ID | Gap | Lane | Severity | Vì sao | File gốc |
|----|-----|------|----------|--------|----------|
| **GAP-01** | **RLS chỉ apply cho schema `public` + policy `USING(true)` permissive** | Worker/SQL | 🔴 HIGH | Master table ở schema khác KHÔNG có RLS; policy mặc định full-access = không cô lập dữ liệu. Prod multi-tenant = lỗ hổng | `master_ddl_generator.go:216-219`, `038_*.sql:171-196` |
| **GAP-02** | **OCC master dùng `_hash` thay vì `_source_ts`** → event cũ tới sau có thể đè bản mới hơn | Worker | 🟠 MED | Cần verify thực tế: transmute đọc từ shadow (đã LWW theo `_source_ts`); rủi ro chỉ khi xử lý batch out-of-order. PHẢI kiểm chứng trước khi sửa | `transmuter.go:546-557` |
| **GAP-03** | **Close-loop bỏ qua trigger không có `schedule_id`** → realtime/run-now không cập nhật `last_status` | Worker | 🟠 MED | Trang Monitor (F5) + status widget hiển thị thiếu cho sync realtime | `job_monitor.go:75-79` |
| **GAP-04** | **`/schedules` bị comment khỏi menu** → operator không vào được từ UI | FE | 🟠 MED | Trang LIVE nhưng ẩn; chặn vận hành. Fix trivial | `App.tsx:118-122` |
| **GAP-05** | **Thiếu Edit/Delete schedule** (FE + API PUT/DELETE) | FE+API | 🟠 MED | Sai cron_expr phải xoá-tạo lại; không quản trị được vòng đời schedule | `TransmuteSchedules.tsx`, `router.go` (no PUT/DELETE `/schedules/:id`) |
| **GAP-06** | **Create Mapping thiếu chọn `transform_fn`** | FE | 🟠 MED | Operator không cấu hình transform thủ công như blueprint; phải sửa qua đường khác | `MasterMappingFieldsPage.tsx:606-635` |
| **GAP-07** | **Gate ≥1 approved rule là soft** (zero rules = success) | Worker | 🟡 LOW | False-positive success nếu rule bị xoá nhầm; cần Warn rõ | `transmuter.go:167-171` |
| **GAP-08** | **post_ingest gate fail-open khi DB lỗi** → NATS noise transient | Worker | 🟡 LOW | Chấp nhận được (backward-compat) nhưng nên log + metric | `sinkworker.go:285-326` |
| **GAP-09** | **Thiếu PUT update binding** | API | 🟡 LOW | Tạo sai phải xoá; ít dùng vì có Swap | `router.go` |
| **GAP-10** | **`/sync/status` & `scan-flatten` hợp nhất chưa có** (chức năng tồn tại nhưng tách 2-3 bước) | API+FE | 🟡 LOW | Thuần UX/DX; không chặn nghiệp vụ. Defer được | `registry_handler_read.go:52`, `introspection_handler.go:385` |
| **GAP-11** | **Pending review thiếu modal đơn lẻ per-row** | FE | 🟡 LOW | Batch đã đủ chức năng; bổ sung sau | `MasterMappingFieldsPage.tsx:534-540` |
| **GAP-12** | **Master Registry thiếu filter UI** (status/sync_mode) | FE | 🟡 LOW | Chỉ filter qua URL param; list dài khó dùng | `MasterRegistry.tsx:74-78` |

---

## C. Khuyến nghị hướng đi (best path, không option)

**Phase A KHÔNG còn là "build từ đầu" — mà là "đóng gap + siết chất lượng".** Thứ tự đề xuất cho Muscle:

1. **Đóng 1 HIGH bảo mật trước tiên — GAP-01 (RLS)**: đây là prod-blocker thật. Phương án: RLS phải apply cho MỌI master schema (không hardcode `public`), và policy không được `USING(true)` trống — tối thiểu gate theo tenant/owner column hoặc role. Cần 1 ADR riêng vì đụng bảo mật + migration.
2. **Xác minh rồi xử lý GAP-02 (OCC)**: viết test reproduce out-of-order trên master upsert TRƯỚC khi đổi `_hash`→ thêm guard `_source_ts`. Không sửa mù (lesson L99: không báo láo). **Tuyệt đối không đụng shadow-side OCC (`upsert.go`) — đó là luồng source→shadow.**
3. **GAP-03 + GAP-04 + GAP-05 + GAP-06**: cụm "hoàn thiện vận hành" — close-loop cho mọi trigger, mở lại menu `/schedules`, Edit/Delete schedule (FE+API), thêm `transform_fn` vào Create Mapping. Đây là phần làm cho operator dùng được trọn vòng.
4. **GAP-07..12**: LOW — gom vào 1 đợt polish hoặc defer kèm ticket, không chặn release big-bang.

**Quy tắc khi thực thi (note của Boss):** core-systems, không cheat DB/config; mỗi sửa đổi bám pattern source; minimal impact; có report_*.md ghi file + LOC; chạy build/test service mới báo done. Brain chỉ plan — Muscle thực thi sau khi User approve.
