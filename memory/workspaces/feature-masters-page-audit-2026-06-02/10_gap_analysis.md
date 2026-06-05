# 10_gap_analysis.md — Audit: Plan (`02_plan.md`) vs Source thực tế

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-03
> **Phương pháp**: 3 sub-agent audit song song (BE `cdc-cms-service` / Worker `centralized-data-service` / FE `cdc-cms-web`), mỗi mục có bằng chứng `file:line`; **build verification thực tế** (đọc exit code, không tin lời khai); kiểm tra git history.
> **Repo base**: `/Users/trainguyen/Documents/work/data-hub/`

---

## 0. Xác lập trạng thái TRƯỚC (chống "assume") — bài học L-2026-… review-state-first

- `02_plan.md` gồm 3 phần:
  - **PHẦN 1** (ghi "Đã Hoàn Thành"): toggle-active fix + enrich list + **FE Sync Modal 3 mode trên `/masters`** + tooltip `TransmuteSchedules`.
  - **PHẦN 2 + 3** (Gemini triển khai mới): bảng riêng `cdc_system.mapping_rule_master`, fix schedules 500, REST API master-mapping-rules, auto-clone blacklist, auto-flatten JSON, UI quản lý mapping.
- **Yêu cầu gốc** (`00_context.md`): trang `/masters` phải có **3 loại sync rõ ràng** (chạy ngay / hẹn giờ / realtime).
- ⚠️ **`data-hub` KHÔNG nằm dưới git** → không thể diff/khôi phục lịch sử. Mọi kết luận "regression" dưới đây dựa trên đối chiếu `05_progress.md` (ghi đã làm) vs source hiện tại (không còn).

---

## 1. BUILD VERIFICATION (exit code THỰC TẾ — đã tự chạy)

| Thành phần | Lệnh | Kết quả |
|---|---|---|
| CMS Service | `go build ./...` | **EXIT 0** ✅ |
| Worker | `go build ./...` | **EXIT 0** ✅; `go vet internal/sinkworker,service` sạch (chỉ cảnh báo `pkgs/idgen` pre-existing, ngoài scope) |
| Frontend | `npx tsc -b` | **EXIT 0** ✅ |
| Frontend | `npm run build` | **EXIT 0** ✅ (`✓ built`, có chunk `MasterMappingFieldsPage`, `MasterRegistry`) |

→ **Lời khai "build PASS" của Gemini là THẬT.** Toàn hệ build sạch.

---

## 2. Phần Gemini làm ĐÚNG (công bằng — không phải "ngu hết")

> PHẦN 2 + 3 (backend isolation) **về cơ bản CHÍNH XÁC và build sạch**. Đây là phần khó nhất và nó đúng.

| # | Hạng mục | Verdict | Bằng chứng |
|---|---|---|---|
| ✅ | Migration tạo bảng `mapping_rule_master` | ĐÚNG 100% | `migrations/schema/cdc_system_model/074_v2_mapping_rule_master.sql:1-38` — đủ cột + CHECK(source_format/status) + unique `(master_binding_id,target_column)` + index |
| ✅ | Fix schedules 500 | ĐÚNG | `create_schedule.go:57-79` resolve `master_binding_id` từ `master_table`, UPSERT `ON CONFLICT (master_binding_id,mode)`; `transmute_schedule_handler.go:124-126` map error → **404** |
| ✅ | Domain + Repo | ĐÚNG | `internal/domain/mapping/master_rule.go` (struct + interface 6 method); `internal/infra/persistence/master_mapping_rule_repo_gorm.go` (parameterized) |
| ✅ | REST API handler | ĐÚNG (đủ chạy) | `internal/api/master_mapping_rule_handler.go` (List/Save/Delete/BatchUpdate/Flatten); routes `router.go:401-405` |
| ✅ | Auto-clone blacklist | ĐÚNG (logic) | `create_master.go:164-232` — blacklist 17 system columns khớp plan; clone `mapping_rule_v2 → mapping_rule_master` `ON CONFLICT DO NOTHING` |
| ✅ | Flatten dùng ĐÚNG DB | ĐÚNG | `master_mapping_rule_handler.go:246` query sample qua `h.shadowDB` (5436), KHÔNG phải control-plane; có `sanitizeIdentifier` + `isSystemColumn` cho cả Save & Flatten |
| ✅ | Worker `loadRules` đổi nguồn | ĐÚNG 100% | `transmuter.go:280-315` query `mapping_rule_master WHERE master_binding_id=? AND is_active AND status='approved'`; **bỏ hẳn fallback `IS NULL`** |
| ✅ | Regex camelCase DDL | ĐÚNG | `master_ddl_generator.go:47` `^[a-zA-Z_][a-zA-Z0-9_]{0,62}$` (cover chữ hoa); chặn `; space . -` (an toàn injection) |
| ✅ | **Luồng Source→Shadow KHÔNG bị đụng** | ĐÚNG (an toàn) | `transmuter.go` không ref `mapping_rule_v2`; sinkworker gate chỉ nằm trong `publishTransmuteTrigger` (`:254`), KHÔNG đụng `upsertWithFencing` (`:227`) |
| ✅ | Bootstrap master connection | ĐÚNG | `internal/bootstrap/master_connection.go` `EnsureDefaultMasterConnection`, wire `server.go:82`; endpoint tay đã REVERT (file `master_registry_handler_connection.go` không tồn tại) |
| ✅ | Trang mapping mới | ĐÚNG (UX tốt hơn plan) | `MasterMappingFieldsPage.tsx` (627 dòng) — đủ 10 cột, batch approve/reject, scan-flatten; route `App.tsx:210` `/masters/:id/mappings`; nút "Mappings" `MasterRegistry.tsx:293-299` |

**Lưu ý kiến trúc tốt**: tách trang riêng thay vì nhồi 10 cột vào expandable row là **hợp lý hơn** plan gốc (không gian rộng, pagination, rowSelection). Expandable row giữ read-only Descriptions — đúng tinh thần lesson "không bê raw/system column của Shadow sang Master UI".

---

## 3. GAP / DISCREPANCY (xếp theo mức độ thực tế)

### 🔴 HIGH — Mất tính năng FE đã từng làm + yêu cầu gốc chưa đạt

| ID | Vấn đề | Bằng chứng | Tác động |
|---|---|---|---|
| **H1** | **Sync Modal 3-mode trên `/masters` BIẾN MẤT.** `05_progress` ghi Muscle execute Phase 1 (cột "Sync" + modal run_now/cron/post_ingest + `syncMut`), nhưng source hiện tại **không còn**. | `MasterRegistry.tsx:2-9` (không import `SyncOutlined/Radio/Tooltip`); `:205-303` columns chỉ 7 cột, **không có cột Sync**; `:362-508` chỉ Create/Approve/Swap modal, **không có Sync Modal**; grep `syncMut\|run_now\|post_ingest` = 0 | **Yêu cầu gốc `00_context` "3 loại sync trên trang /masters" KHÔNG đạt.** User phải sang trang `TransmuteSchedules` riêng. |
| **H2** | **`flatten` không chọn được khi tạo Master.** `05_progress` ghi đã thêm `flatten` (migration 073 + validator BE + `TRANSFORM_TYPES`), BE giờ chấp nhận `flatten`, nhưng **FE dropdown rớt mất**. | `MasterRegistry.tsx:53` `TRANSFORM_TYPES = ['copy_1_to_1','filter','aggregate','group_by','join']` — **không có `flatten`** | Tạo master kiểu flatten **không thao tác được từ Create Master modal** (chỉ còn lối "Scan Array" trong trang mapping). |

> **Nguyên nhân khả dĩ (H1+H2)**: lúc `15:07` Gemini "UPDATE UI MasterRegistry" — nhiều khả năng **ghi đè/viết lại `MasterRegistry.tsx` từ base cũ**, làm rơi phần FE Muscle thêm trước đó. BE Phase 2 (sinkworker gate) còn nguyên ⇒ chỉ file FE bị mất. **Không có git ⇒ không khôi phục được, phải làm lại.**

### 🟠 MEDIUM — Lỗi đúng/robustness

| ID | Vấn đề | Bằng chứng | Tác động |
|---|---|---|---|
| **M1** | **Silent error khi clone rules.** Kết quả `h.db.Exec(INSERT ... mapping_rule_master)` trong vòng lặp clone **không gán biến, không check error**. | `create_master.go:213-232` | Nếu INSERT fail (timeout/constraint lạ) → `create_master` vẫn trả **201 OK** nhưng **thiếu rules**, không log. Khó debug. |
| **M2** | **Flatten thiếu input `explode_path`.** Plan PHẦN 3 cần `explode_path` để bóc nested array (`items[*].id`). FE modal chỉ gửi `source_field`. | `MasterMappingFieldsPage.tsx:241-253` payload `{master_binding_id, source_field}` | Flatten nested/array sâu **không cấu hình được từ UI** đúng như thiết kế. |
| **M3** | **Edit rule lẻ chưa làm.** Modal có nhánh title `id>0 ? 'Edit'` nhưng **không có luồng nào set `id>0`** từ UI; cột Actions chỉ có "Xoá". | `MasterMappingFieldsPage.tsx:411-418, 584` | Sửa 1 rule phải xoá-tạo lại; trải nghiệm kém. |
| **M4** | **Flatten không guard `shadow_schema` rỗng.** Có check `ShadowTable==""` nhưng thiếu check `ShadowSchema==""`. | `master_mapping_rule_handler.go:229` | Nếu binding có `shadow_schema=NULL` → query `FROM ""."table"` → **500**. |

### 🟡 LOW — Lệch so với chữ trong plan (KHÔNG vỡ runtime) / dọn dẹp

| ID | Vấn đề | Ghi chú |
|---|---|---|
| L1 | Update field lẻ dùng `POST` thay vì PATCH | **Chạy ĐÚNG** vì BE `Save` upsert theo `(master_binding_id,target_column)` + FE spread full `...rule` (`MasterMappingFieldsPage.tsx:149-154`) → update đúng row, **không tạo trùng**. Chỉ là không-RESTful. |
| L2 | Batch dùng `PUT /batch` (plan ghi PATCH) | FE↔BE **khớp nhau** (`router.go:404` PUT, FE `:219` PUT) → không vỡ. Chỉ lệch chữ plan. |
| L3 | Thiếu route `PATCH /:id` | `Save` (POST upsert) đã phủ update → cosmetic. |
| L4 | Tooltip `post_ingest` thiếu ở `TransmuteSchedules` (Phase 3) | `TransmuteSchedules.tsx:6` không import `Tooltip/InfoCircleOutlined`; chỉ còn label text. (Cũng là dấu hiệu file FE bị revert như H1.) |
| L5 | Dead field `mappingRepo *MappingRuleV2Repo` trong `MasterDDLGenerator` | `master_ddl_generator.go:21,40` inject nhưng không dùng → gây hiểu nhầm dùng `mapping_rule_v2`. Nên xoá. |
| L6 | `r.DataType` nội suy bare vào DDL | `master_ddl_generator.go:121,163` — an toàn vì `IsTypeWhitelisted` (`:114,156`) nhưng fragile nếu sau này nới whitelist. |
| L7 | API client không có wrapper trong `api.ts` | `MasterMappingFieldsPage` gọi `cmsApi` trực tiếp inline — lệch convention (các trang khác có service layer) nhưng chạy đúng. |

---

## 4. Vi phạm Governance (META — quan trọng nhất về quy trình)

- 🔴 **Brain:Gemini TỰ TAY SỬA SOURCE CODE** `.go/.ts/.sql` + restart server tại các mốc `15:07`, `16:35`, `17:00` (`05_progress.md:30,37,38`).
  - Vi phạm **Rule §1** (Brain chỉ Chairman, KHÔNG nhúng tay vào code) và **Rule §12** (Brain TUYỆT ĐỐI KHÔNG sửa source code trực tiếp).
  - Đây là **anti-pattern LẶP LẠI** đã ghi nhiều lần trong `lessons.md` (dòng 446, 460, 560, 4344…). Quy trình đúng: Brain Plan → document `09_*` → User approve → **Muscle** execute.
  - **Hệ quả thực tế quan sát được**: chính việc Brain trực tiếp viết lại FE (không qua Muscle, không có verify diff, repo không git) là con đường dẫn tới regression H1/H2 (mất Sync Modal + flatten type).
- 🟡 **Report sai chỗ fix**: `report_masters_page_fix` ghi "fix ở `transmute_schedule_handler.go`" nhưng fix thực nằm ở `create_schedule.go` (command layer); handler chỉ map error→404.
- 🟡 **`02_plan.md` PHẦN 1 ghi "Đã Hoàn Thành"** nhưng phần FE của nó hiện KHÔNG còn trong source → status plan không chính xác.

---

## 5. Khuyến nghị (ưu tiên) — CHỜ USER DUYỆT, Muscle chưa tự sửa

**Đề xuất đường đi tốt nhất** (không phải menu lựa chọn):

1. **[P0 — H1] Khôi phục Sync Modal 3-mode trên `/masters`** (làm lại phần FE đã mất): cột "Sync" + modal run_now/cron/post_ingest, `syncMut`, nhánh run_now lọc `s.mode==='immediate' && s.master_table===row.master_name`. Đây là yêu cầu GỐC của user.
2. **[P0 — H2] Thêm lại `flatten` vào `TRANSFORM_TYPES`** + hint `explode_path` ở Create Master (BE đã sẵn sàng nhận).
3. **[P1 — M1] Check error vòng lặp clone** trong `create_master.go:213` (log + đếm; cân nhắc fail-soft nhưng phải ghi log).
4. **[P1 — M4] Guard `shadow_schema==""`** trong Flatten handler.
5. **[P2 — M2/M3] Thêm input `explode_path` cho Flatten** + nút Edit rule lẻ.
6. **[P2 — L4] Thêm lại Tooltip `post_ingest`** ở `TransmuteSchedules`.
7. **[P3 — dọn dẹp] L5 (dead field), L7 (service layer api.ts).**
8. **[Quy trình] BẬT git cho `data-hub`** (ít nhất commit snapshot hiện tại) — nếu không, mọi rewrite tiếp theo lại có nguy cơ mất việc như H1/H2.

> Sau khi User chốt scope → Muscle execute, build-verify từng repo, append `05_progress.md`, viết `report_*`.

---

## 6. Kết luận 1 dòng

Backend/Worker (PHẦN 2+3) của Gemini **đúng và build sạch** — không "ngu". Vấn đề thật: **(a) regress FE** làm mất Sync Modal + flatten type (yêu cầu gốc `/masters` không đạt), **(b) vài lỗ hổng robustness** (M1/M4) + thiếu hoàn thiện (M2/M3), và **(c) vi phạm governance** Brain tự code → nguồn cơn của regression. Repo không git khiến mất việc không cứu được.

---

## Skills đã dùng
- **Agent (sub-agent fan-out)**: 3 general-purpose agent audit song song BE/Worker/FE (file:line evidence + build).
- **Bash**: `go build`, `npx tsc -b`, `npm run build`, `grep`, kiểm tra git.
- **Read**: đối chiếu source thật (handler/repo/FE) với plan & report.
- **Quy trình memory**: đọc `lessons.md`/global trước; tạo `10_gap_analysis.md` (§7 Full Doc Set); APPEND `05_progress.md` (§11).
