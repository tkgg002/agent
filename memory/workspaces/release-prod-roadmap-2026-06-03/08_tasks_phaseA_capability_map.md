# 08_tasks_phaseA_capability_map.md — Phase A theo Capability (FE / API / CDC Worker)

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-04
> **Thay thế** `08_tasks_phaseA_by_lane.md` (bản cũ quá micro/bug-level — giữ làm audit trail).
> **Tầm nhìn**: Master/Transmute = "Operator điều khiển toàn bộ vòng đời Shadow → Master:
> định nghĩa mapping → quản lý binding → đặt lịch/cách sync → worker thực thi sync → phản hồi trạng thái."

---

## 🧭 Bức tranh tổng (1 dòng mỗi lane)
- **FE** = 3 trang quản trị: **Master Mapping** (định nghĩa cột) · **Master Registry** (binding sync) · **Sync Schedule** (action/lịch sync).
- **API** = backend phục vụ 3 trang đó + dispatch lệnh sync + đọc trạng thái.
- **CDC Worker** = engine thực thi sync (run_now / cron / realtime), scan shadow, apply DDL, close-loop.

---

## 🟦 LANE FE — `cdc-cms-web` (3 trang quản trị)

### FE-M1 · Trang **Master Mapping** — quản lý danh sách mapping cột
> Operator xem & định nghĩa mapping Shadow column → Master column.
- Bảng cột với trạng thái: **In Shadow**, **In Master**, **Source Data Type**, transform_fn, status.
- **Create Mapping**: workflow thủ công operator chủ động tạo rule (không chờ drift auto).
- **Pending Review modal**: Approve/Reject **hàng loạt + đơn lẻ** các rule trước khi vào Master.
- **Gate UI**: chặn Approve→Master nếu shadow rule chưa Approved / chưa "In Shadow".
- **DoD**: operator tạo → review → approve 1 mapping hoàn chỉnh, system column không hiển thị để chọn.

### FE-M2 · Trang **Master Registry** — quản lý các binding sync
> Vòng đời 1 master binding (đã có nền MasterRegistry.tsx, cần hoàn thiện).
- List binding + expand Spec JSON/FQN; trạng thái schema_status / is_active.
- Action lifecycle: **Create · Approve (trigger DDL) · Reject · Toggle Active · Atomic Swap**.
- **Sync trực tiếp tại row**: nút "Sync" mở modal 3 mode (Chạy ngay / Hẹn giờ / Realtime) — shortcut không phải sang trang Schedule.
- Hiển thị **last sync status** mỗi binding (cầu nối sang luồng Monitor).
- **DoD**: từ Registry approve → bật active → sync được 1 binding mà không rời trang.

### FE-M3 · Trang **Sync Schedule** — quản lý các action/lịch sync
> Quản trị "khi nào & cách nào" sync chạy.
- List schedule theo binding; mode **cron / immediate / post_ingest** (có tooltip giải thích realtime).
- Action: tạo/sửa/xoá schedule, **Run-now** thủ công, enable/disable.
- Hiển thị **last_run_at / next_run_at / last_status** (feedback close-loop).
- **DoD**: tạo đủ 3 loại schedule, run-now 1 lần, thấy trạng thái cập nhật.

### FE-M0 · Nền tảng FE
- Contract type khớp API (MasterRow, MappingRule, Schedule); error humanize; query invalidate.
- Build gate: `npm run build` + `tsc` strict, 0 type error.

---

## 🟩 LANE API — `cdc-cms-service` (năng lực phục vụ FE)

### API-G1 · **Master Binding API** (phục vụ FE-M2)
- CRUD binding + Approve/Reject/Toggle-active/Atomic-swap.
- Approve → publish `cdc.cmd.master-create` (trigger worker DDL).
- **DoD**: 6 action lifecycle trả contract đúng, đã verify ở master-audit (OK).

### API-G2 · **Master Mapping API** (phục vụ FE-M1)
- List columns với enrich: **In Shadow / In Master / Source Data Type** (fix lookup `master_table`, không `master_name`).
- Create-Mapping endpoint (operator thủ công).
- Pending list + **bulk/single Approve-Reject**.
- **System-column blacklist** (nguồn định nghĩa DUY NHẤT, share với Worker) — không cho vào Master.
- **DoD**: trang Mapping load + tạo + review chạy thật, không lọt system column.

### API-G3 · **Schedule & Sync-Dispatch API** (phục vụ FE-M3 + nút Sync FE-M2)
- CRUD schedule, mode immediate/cron/post_ingest, reason≥10, Idempotency-Key.
- **Run-now** dispatch → publish `cdc.cmd.transmute`.
- **DoD**: 3 mode dispatch đúng, run-now không nhầm binding.

### API-G4 · **Status/Feedback Read API** (phục vụ hiển thị trạng thái mọi trang)
- Đọc last_status / dispatch-status / last_run cho binding & schedule.
- **DoD**: số liệu trạng thái khớp DB thật.

### API-G0 · Nền tảng API
- Build gate: `go build ./...` + `go vet` + `go test`.

---

## 🟧 LANE CDC WORKER — `centralized-data-service` (cơ chế chạy sync)

### WK-E1 · **Transmute Engine** (lõi sync Shadow → Master)
> Cách dữ liệu thực sự chảy sang Master.
- gjson eval theo jsonpath + transform_fn → typed value.
- **Gate chain**: master active+approved ∧ shadow active+profile_active ∧ ≥1 approved rule.
- **OCC upsert** theo `_source_ts older` (không overwrite dữ liệu mới hơn).
- **System-column exclude** ở DDL/sync path (đồng bộ blacklist với API-G2).
- **DoD**: 1 record shadow → master đúng kiểu, không sinh system column.

### WK-E2 · **3 cơ chế kích hoạt Sync** (run_now / cron / realtime)
- **Cron**: scheduler tick 60s + fencing token (chống double-tick) → publish `cdc.cmd.transmute`.
- **Immediate (run-now)**: handler nhận lệnh thủ công → chạy 1 lần.
- **Post_ingest (realtime)**: SinkWorker sau ghi shadow → **gate `hasPostIngestSchedule()`** → chỉ publish khi có schedule realtime bật (tránh NATS noise).
- **DoD**: cả 3 cơ chế trigger được transmute, realtime có gate.

### WK-E3 · **Shadow Introspection** (phục vụ Create-Mapping FE-M1)
- Scan field/array của bảng shadow chạy đúng trên **shadow DB instance** (không nhầm control-plane DB).
- **DoD**: scan trả sample fields cho FE tạo mapping, không lỗi relation-missing.

### WK-E4 · **Master DDL Apply** (phục vụ Approve FE-M2)
- Approve binding → ALTER master table theo approved rule (typed columns, RLS).
- **DoD**: approve → master table có cột đúng, không system column.

### WK-E5 · **Close-loop Feedback** (cầu nối Monitor F5)
- Publish `cdc.evt.transmute.completed` sau mỗi sync → JobMonitor cập nhật `last_status`.
- **DoD**: mỗi sync để lại trạng thái đọc được ở FE.

### WK-E0 · Nền tảng Worker
- Build gate: `go build ./...` + `go vet` + `go test ./internal/{sinkworker,handler,service}/...`.

---

## 🔗 Liên kết FE ↔ API ↔ Worker (đọc theo capability)
| Capability nghiệp vụ | FE | API | Worker |
|----------------------|----|----|--------|
| Định nghĩa mapping cột | FE-M1 | API-G2 | WK-E3 (scan), WK-E1 (apply) |
| Quản lý binding + DDL | FE-M2 | API-G1 | WK-E4 |
| Đặt lịch/cách sync | FE-M3 | API-G3 | WK-E2 |
| Chạy sync data | nút Sync (M2/M3) | API-G3 dispatch | WK-E1 + WK-E2 |
| Trạng thái/feedback | mọi trang | API-G4 | WK-E5 |

## ✅ Definition of Done — Phase A (capability-level)
Operator, **chỉ trên UI**, hoàn thành trọn vòng: **tạo mapping → review/approve → approve binding (DDL) → chọn cách sync (run-now/cron/realtime) → worker chạy → thấy trạng thái cập nhật** — không system column lọt sang Master, có evidence cả 3 mode. Sau đó security gate no HIGH.

## Thứ tự & Song song
- **Worker WK-E1/E2/E3/E4** và **API-G2/G3/G4** là nền → làm trước.
- **FE-M1/M2/M3** bám contract API → làm sau từng nhóm.
- 2 Muscle: M1 = API+Worker (Go), M2 = FE. Mỗi capability đóng theo chiều dọc (vertical slice) để demo được sớm.

## Coordination
- W2 `feature-sync-shadow-master-bindings-2026-06-04` (x2 cầm) phủ phần lớn API-G2 + WK-E3 + FE-M1 → **lane-lock**, Brain-này không đụng code (§12).
- System-column blacklist: **1 const dùng chung** API-G2 ↔ WK-E1.
