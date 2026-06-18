# REPORT — feat `_id` opt-in (Cách 2) + fix hardcode realCols
Ngày: 2026-06-11 | Agent: claude-opus-4-8 (Muscle) | Service: centralized-data-service

## 1. Nguyên tắc đã chốt (theo user)
- `_id` **KHÔNG mặc định mọi nơi** — chỉ OPT-IN qua mapping explicit + scan do USER bấm.
- Core **KHÔNG auto-tạo/tự can thiệp**, **KHÔNG đụng source ("db cũ")**.
- **Không Mongo-specific hardcode** — source MySQL/PG PK là `id`, không bị ép `_id`.
- Bỏ kiểu trình bày "option 1/2/3" → chốt 1 hướng minimal.

## 2. Files thay đổi (task `_id`) — git diff vs HEAD
| File | LOC (±) | Nội dung | Loại |
|---|---|---|---|
| `internal/service/mongo_introspection.go` | 7 | Bỏ skip `_id` khi introspect collection → "scan field lần đầu" HIỆN `_id` | **GIỮ** (Cách 2; user confirm "đã có _id") |
| `internal/handler/command_handler.go` | 12 | Scan-show `_id` ở `HandleDiscover`/`HandleScanRawData`/`HandleScanArrayFields` (đều **request-reply = USER bấm**). `HandlePeriodicScan` đã **REVERT**. | **GIỮ** scan-show + **REVERT** auto |
| `internal/service/child_explode.go` | 4 | Cho `_id` qua guard HasPrefix khi child rule map `_id` (map-normally) | **GIỮ** (Cách 2) |
| `internal/service/schema_inspector.go` | 0 | `findNewFields` đã **REVERT về gốc** (bỏ auto-flag `_id` lúc ingest) | **REVERT** (auto) |
| `internal/service/master_ddl_generator.go` | 52* | *Hôm nay CHỈ sửa: bỏ hardcode `"_id":true` trong `realCols` (1 dòng + comment). ~45 dòng còn lại = block `spec.pk guard + parsePKFromSpec` của **task TRƯỚC** (fix lỗi 42703 ở bảng `b3`), KHÔNG thuộc task hôm nay. | **GIỮ** (realCols fix) |

> `schema_inspector.go` không còn trong diff = đã về nguyên gốc 100%.

## 3. Hai path AUTO đã revert (đúng cái user bác: "_id mọi nơi / tự can thiệp")
1. **`schema_inspector.findNewFields`** — chạy lúc INGEST event (auto). Un-skip `_id` → `_id` bị auto-flag drift/pending cho mọi bảng Mongo. → khôi phục `"_id":true` vào skip.
2. **`command_handler.HandlePeriodicScan`** — subject `cdc.cmd.periodic-scan` (**pub/sub, scheduler-triggered**) + **AUTO-CREATE mapping rule** (`new_rules_created`). Un-skip `_id` → tự tạo rule `_id` cho mọi bảng. → khôi phục `"_id":true` vào skip.

## 4. Vì sao GIỮ phần còn lại (không phải scope-creep)
- `HandleScanRawData`/`HandleScanArrayFields`/`HandleDiscover` = **request-reply** (user chủ động scan) → hiện `_id` để user CHỌN map = đúng Cách 2 ("lúc scan field cứ hiện ra cả _id").
- `master_ddl_generator.seen` **VỐN không reserve `_id`** (chỉ thêm comment) → master nhận `_id` qua mapping là **hành vi GỐC**, không phải tôi enable.
- Bằng chứng đúng plane: shadow `cdc_shadow:5436` 6/15 bảng đã có `_id`; master `goopay_dest:5434` nhiều bảng đã có `_id` — tất cả qua **mapping rule explicit** (8 rule `_id→_id`).

## 5. Verify (Rule 16)
- **G3 build**: `gofmt -l` clean; `go build ./...` **PASS**.
- **G8 bằng chứng**: grep xác nhận `findNewFields`+`HandlePeriodicScan` có lại `"_id":true` (auto off); `mongo_introspection` vẫn show `_id`; `realCols` hết hardcode `_id`.
- **CHƯA verify runtime** (re-sync) → KHÔNG claim done phần data.

## 6. Việc còn lại (user)
Rebuild + restart worker → map `_id→_id` 1 bảng đang chưa có (vd `shadow_goopay_lc_ws.wallets`) + sync → query `cdc_shadow:5436` + `goopay_dest:5434` xác nhận cột `_id` có data. Báo lại để tôi verify đúng plane.

---

## BỔ SUNG [2026-06-11] — FEAT `transform_spec.indexes` (plain index user khai báo)
Yêu cầu: "1 thôi, làm đi" → làm `transform_spec.indexes`, defer multi-pk.

### Cơ chế
`transform_spec` nhận thêm key `"indexes"` (mảng tên cột), vd:
```json
{ "pk": "_id", "indexes": ["status", "user_id"] }
```
→ mỗi cột sinh `CREATE INDEX IF NOT EXISTS ix_<table>_<col>` (NON-unique). Đọc động qua `parseIndexesFromSpec` (KHÔNG hardcode tên cột). Dùng lại guard `realCols` như spec.pk: cột chưa tồn tại (chưa map/approve) → bỏ qua (tránh 42703); approve xong → tạo ở Apply sau. Dedup: bỏ cột đã là pk (unique idx) / financial (auto) / system.

### Files & LOC
| File | ± | |
|---|---|---|
| `internal/service/master_ddl_generator.go` | +103/-1 | `parseIndexesFromSpec` + loop tạo plain index + hoist pkCol |
| `internal/service/master_ddl_indexes_test.go` (NEW) | 55 | unit test pure func |

### Verify
- gofmt OK; `go build ./...` PASS.
- `go test -run TestParseIndexesFromSpec -v` → **9 subtests + NoHardcode PASS** (nil/empty/single/multi/coexist-pk/trim/wrong-type/malformed; + guard không leak `_id`/`_gpay_id`).
- CHƯA verify runtime: cần worker rebuild + set `transform_spec.indexes` 1 binding + approve cột + apply → query `goopay_dest:5434` thấy `ix_<table>_<col>`.

---

## BỔ SUNG [2026-06-11] — COMPOSITE index (nhóm field) cho `transform_spec.indexes`
Yêu cầu: "làm đi" → indexes hỗ trợ cả composite (nhiều cột chung 1 index).

### Format (backward-compat — phần tử string=đơn | array=composite)
```json
{ "indexes": ["status", ["tenant_id", "created_at"]] }
```
→ `ix_<t>_status ON (status)` (đơn) + `ix_<t>_tenant_id_created_at ON (tenant_id, created_at)` (composite).

### Logic
- `parseIndexesFromSpec` → `[][]string` (mỗi nhóm = 1 index). Đọc động (`[]json.RawMessage` thử string rồi []string), KHÔNG hardcode.
- Mỗi nhóm: existence-check **TẤT CẢ** cột (thiếu 1 → bỏ cả nhóm, tránh 42703); guard `realCols` + approve-gated. Dedup: index đơn trùng pk/financial/system → bỏ; chống tên index trùng.

### Files & LOC (cộng dồn task indexes)
| File | ± |
|---|---|
| `internal/service/master_ddl_generator.go` | +152/-1 (parseIndexesFromSpec [][]string + loop composite + pkCol hoist) |
| `internal/service/master_ddl_indexes_test.go` | 61 (12 subtests + NoHardcode) |

### Verify
- gofmt OK; `go build ./...` PASS.
- `go test -run TestParseIndexesFromSpec -v` → **12 subtests + NoHardcode PASS** (gồm composite/mixed/empty-group/wrong-type).
- CHƯA verify runtime: worker rebuild + set `transform_spec.indexes` (có composite) + approve cột + apply → query `goopay_dest:5434` thấy `ix_<table>_<c1>_<c2>`.
