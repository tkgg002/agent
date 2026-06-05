# 08_tasks_phaseA_by_lane.md — Phase A tách theo Lane (FE / API / CDC Worker)

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-04
> **Phase A**: Master/Transmute finalize (execute + test) · **Window**: 06-03 → 06-05
> Trạng thái: ⬜ todo · 🟡 in-progress · ✅ done · ⛔ blocked

## Nguồn hợp nhất (Phase A = union 2 workspace)
- **W1** `feature-masters-page-audit-2026-06-02` (✅ audited): P1 FE Sync Modal, P2 sinkworker gate, P3 tooltip.
- **W2** `feature-sync-shadow-master-bindings-2026-06-04` (🟡 active): fix SQL 42703 / 42P01, system-column blacklist, UI Master Mapping (In Shadow/In Master/Source Type, Create Mapping, Pending modal).
- ⚠️ **Coordination**: W2 đang do Antigravity/x2 cầm. Brain-này KHÔNG sửa code (§12) — bảng dưới là điều phối; tránh trùng lặp, lane-lock rõ ràng.

## Bản đồ component → lane
| Lane | Repo | Vùng chạm |
|------|------|-----------|
| **FE** | `cdc-cms-web` | MasterRegistry.tsx, TransmuteSchedules.tsx, MappingFieldsPage |
| **API** | `cdc-cms-service` | master_mapping_rule_handler.go, create_master.go, transmute_schedule_handler.go |
| **CDC Worker** | `centralized-data-service` | sinkworker.go, command_handler.go, transmuter.go |

---

## 🟦 Lane FE — `cdc-cms-web`
| ID | Task | Nguồn | Phụ thuộc | DoD |
|----|------|-------|-----------|-----|
| A-FE-1 | Sync Modal trên `/masters` (3 mode run_now/cron/post_ingest) + **fix bug `.find`** (lọc `s.master_table===row.master_name`, tránh run-now nhầm master) | W1-P1 | A-API-4 | modal 3 mode hiện đúng |
| A-FE-2 | Tooltip mode `post_ingest` ở TransmuteSchedules (thêm `Tooltip`+`InfoCircleOutlined`, KHÔNG re-add 3 option đã có) | W1-P3 | — | build PASS |
| A-FE-3 | MappingFieldsPage: hiển thị cột **In Shadow / In Master / Source Data Type** | W2 | A-API-3 | render đúng từ API |
| A-FE-4 | Disable checkbox Approve vào Master khi shadow rule chưa Approved / chưa "In Shadow" | W2 | A-API-3 | gate UI hoạt động |
| A-FE-5 | Nút **"Create Mapping"** workflow thủ công cho operator | W2 | A-API-3, A-WK-1 | gọi API tạo rule OK |
| A-FE-6 | **"Pending" review modal** — Approve/Reject hàng loạt + đơn lẻ rules vào Master | W2 | A-API-3 | bulk + single OK |
| A-FE-7 | Build gate: `npm run build` + `tsc` strict | — | A-FE-1..6 | PASS, 0 type error |

## 🟩 Lane API — `cdc-cms-service`
| ID | Task | Nguồn | Phụ thuộc | DoD |
|----|------|-------|-----------|-----|
| A-API-1 | **Fix SQL 42703**: `master_name` → `master_table` trong `MasterColumns` (`master_mapping_rule_handler.go`) + struct tương ứng | W2 | — | query không còn 42703 |
| A-API-2 | **System columns blacklist** chặt ở approve/`SyncFromShadow`/`create_master.go` — loại `_gpay_id,_source_id,_raw_data,_source,_source_ts,_synced_at,_version,_hash,_deleted,_created_at,_updated_at` | W2 | — | system col không lọt sang master |
| A-API-3 | Endpoint hỗ trợ FE: MasterColumns (In Shadow/In Master/Source Type), Create-Mapping, Pending list + bulk approve/reject — **verify/đóng contract** | W2 | A-API-1 | contract khớp FE |
| A-API-4 | Contract `/schedules` cho Sync Modal: body `{master_table,mode,cron_expr,is_enabled,reason}`, `/run-now`, Idempotency-Key, reason≥10 | W1-verify | — | 3 mode dispatch đúng |
| A-API-5 | Build gate: `go build ./...` + `go vet` + `go test` | — | A-API-1..4 | PASS |

## 🟧 Lane CDC Worker — `centralized-data-service`
| ID | Task | Nguồn | Phụ thuộc | DoD |
|----|------|-------|-----------|-----|
| A-WK-1 | **Fix SQL 42P01**: `HandleScanArrayFields` (`command_handler.go`) query trên `shadowDB`/`execDB` thay vì `h.db` (shadow table chỉ ở shadow DB instance) | W2 | — | scan array fields không 42P01 |
| A-WK-2 | **P2 sinkworker post_ingest gate**: thêm `hasPostIngestSchedule()` + guard trước `publishTransmuteTrigger`. **BỎ phần wire DB vào Config/worker_server** (SAI — DB đã wire ở `cmd/sinkworker/main.go`). Verify tên cột JOIN `shadow_binding`/`master_binding` | W1-P2 (revised) | A-DOC-1 | gate hoạt động, fail-open hợp lý |
| A-WK-3 | System-column exclude ở transmuter `SyncFromShadow` DDL path (đồng bộ blacklist với A-API-2) | W2 | — | DDL master không sinh system col |
| A-WK-4 | Build gate: `go build ./...` + `go vet` + `go test ./internal/sinkworker/... ./internal/handler/...` | — | A-WK-1..3 | PASS |

## 🟪 Cross-lane — Doc / Integration / Gate
| ID | Task | Phụ thuộc | DoD |
|----|------|-----------|-----|
| A-DOC-1 | **Revise doc P2** (bỏ DB-plumbing sai) trước khi code A-WK-2 | — | doc P2 revised |
| A-INT-1 | **E2E 3 mode** sync Shadow→Master: run_now → worker log `transmute complete`; cron `* * * * *` → /schedules có row; realtime → Kafka event → log `transmute-shadow`+`transmute complete` | A-FE-7, A-API-5, A-WK-4 | evidence đủ 3 mode |
| A-INT-2 | **E2E mapping flow**: Create Mapping → Pending modal Approve → rule vào master, system col KHÔNG lọt | A-FE-7, A-API-5, A-WK-4 | evidence trước/sau |
| A-INT-3 | **Security gate** `/security-agent` cho toàn diff Phase A | A-INT-1, A-INT-2 | no HIGH/CRITICAL |

---

## Thứ tự thực thi & Song song hoá
1. **Wave 1 (song song, không phụ thuộc nhau)** — mở khoá phần còn lại:
   `A-API-1`, `A-API-2`, `A-API-4`, `A-WK-1`, `A-WK-3`, `A-DOC-1`, `A-FE-2`.
2. **Wave 2**: `A-API-3` (sau A-API-1) → `A-WK-2` (sau A-DOC-1).
3. **Wave 3 (FE phụ thuộc API contract)**: `A-FE-1` (sau A-API-4), `A-FE-3/4/5/6` (sau A-API-3, A-WK-1).
4. **Wave 4 (build gate mỗi lane)**: `A-FE-7`, `A-API-5`, `A-WK-4`.
5. **Wave 5 (integration + security)**: `A-INT-1`, `A-INT-2` → `A-INT-3`.

> **Critical path**: A-API-1 → A-API-3 → A-FE-3/5/6 → A-FE-7 → A-INT-2 → A-INT-3.
> **Lane lock đề xuất** (tránh đụng W2/x2): Muscle-1 = API+Worker (Go), Muscle-2 = FE. Nếu 1 Muscle → theo Wave tuần tự.

## Rủi ro Phase A
| Risk | Mức | Giảm thiểu |
|------|-----|-----------|
| Trùng lặp với W2 (x2 đang cầm) | MED | Lane-lock + đồng bộ blacklist A-API-2 ↔ A-WK-3 chung 1 danh sách |
| A-WK-2 gate chặn nhầm realtime hot-path | MED | fail-open + test degraded; revise doc (A-DOC-1) trước |
| System-column blacklist lệch giữa API và Worker | MED | Định nghĩa 1 nguồn duy nhất (const chung) |
| FE chờ API contract → nghẽn | LOW | Wave hoá; A-API-3 ưu tiên sớm |
