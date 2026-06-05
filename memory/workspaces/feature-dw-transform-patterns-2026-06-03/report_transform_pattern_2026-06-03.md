# report_transform_pattern_2026-06-03.md

> **Workspace active**: `feature-dw-transform-patterns-2026-06-03`
> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-03
> **Scope**: (1) Transform Strategy Registry (plugin pattern) + 2 type `copy_1_to_1`/`flatten`; (2) Flatten discovery `scan-array` → `pending mapping_rule_v2`; (3) Sweep & fix SQLi class `data_type → DDL`.

---

## 1. Tóm tắt
- **Pattern plugin transform**: mỗi loại sync = 1 file implement `Strategy` + `init()` register. Engine `TransmuterModule` dispatch theo `master_binding.transform_type` (trước đây bỏ qua). Thêm type mới = copy 1 file + 1 dòng register, KHÔNG sửa engine.
- **Flatten discovery**: thêm BE `scan-array` (introspect element tại `explode_path` → tạo PENDING `mapping_rule_v2`). Operator duyệt → master DDL (gate sẵn có). KHÔNG auto-tạo cột master.
- **Security sweep**: fix 4 site SQLi cùng class `data_type` nhúng thẳng vào DDL.

## 2. Files đã thay đổi

### MỚI (6 file, 533 dòng — `wc -l` thực tế)
| File | LOC | Mục đích |
|------|-----|----------|
| `centralized-data-service/internal/service/transmute/strategy.go` | 139 | Interface `Strategy` + registry + `RunContext`/`Emit`/`BuildStats` |
| `centralized-data-service/internal/service/transmute/copy_1_to_1.go` | 26 | Strategy 1:1 (tách từ `buildMasterRow`, parity) |
| `centralized-data-service/internal/service/transmute/flatten.go` | 96 | Strategy array-explode (spec `explode_path`, fan-out key `::idx::`) |
| `centralized-data-service/internal/service/transmute/strategy_test.go` | 140 | 6 unit test (registry/fallback/copy/flatten/validate/sorted) |
| `centralized-data-service/internal/service/transmute/README.md` | 71 | "Cách thêm 1 loại sync mới" |
| `centralized-data-service/internal/handler/scan_array_path_test.go` | 61 | Test `explodePathToPGPath`/`validScanIdent` (gồm injection reject) |

### SỬA (8 file, ≈ dòng thêm — ước lượng theo khối chèn)
| File | ≈ thêm | Thay đổi |
|------|--------|----------|
| `centralized-data-service/internal/service/transmuter.go` | ≈ +115 | import transmute; field `TransformSpec`; query `transform_spec`; `processBatch` dispatch; `buildMasterRow`→`extractColumns`; `toTransmuteRules`/`extractColumnsFn`/`extractArrayBytes` |
| `centralized-data-service/internal/handler/command_handler.go` | ≈ +189 | `validScanIdent`, `explodePathToPGPath`, `HandleScanArrayFields` (subject `cdc.cmd.scan-array`); **+ guard SQLi** trong `HandleCreateDefaultColumns` |
| `centralized-data-service/internal/server/worker_server.go` | +1 | Subscribe `cdc.cmd.scan-array` |
| `centralized-data-service/internal/service/child_explode.go` | +13 | **Fix SQLi**: guard `reTypeWhitelist` trước `ALTER TABLE ADD COLUMN <data_type>` |
| `centralized-data-service/internal/service/type_resolver.go` | +8 | Export `IsTypeWhitelisted(spec)` (helper dùng chung guard) |
| `centralized-data-service/internal/service/master_ddl_generator.go` | +13 | **Fix SQLi**: guard `IsTypeWhitelisted` ở CREATE col + ALTER add col |
| `cdc-cms-service/internal/api/introspection_handler.go` | ≈ +55 | Handler `ScanArray` (POST, publish `cdc.cmd.scan-array`, chờ reply) |
| `cdc-cms-service/internal/router/router.go` | +1 | Route `POST /introspection/scan-array/:table` |

## 3. Security — 4 site SQLi class `data_type → DDL` (sweep theo lesson "no whack-a-mole")
| Site | Trạng thái trước | Fix |
|------|------------------|-----|
| `child_explode.go:~227` ADD COLUMN child shadow | ❌ bare | guard `reTypeWhitelist` |
| `command_handler.go HandleCreateDefaultColumns` ALTER TYPE + ADD COLUMN | ❌ bare | guard `service.IsTypeWhitelisted` |
| `master_ddl_generator.go:107` CREATE col (master) | ❌ bare | guard `IsTypeWhitelisted` |
| `master_ddl_generator.go:144` ALTER add (master) | ❌ bare | guard `IsTypeWhitelisted` |
| `command_handler.go:2752` alter-column | ✅ đã có `isSafeType` | (không đụng) |

Validator dùng `reTypeWhitelist` (đúng cái Transmuter validate rule) → KHÔNG drop cột hợp lệ. Đã verify thực tế:
`NUMERIC(18,2)→true, VARCHAR(100)→true, TEXT/BIGINT/TIMESTAMPTZ/JSONB→true`; `"TEXT; DROP TABLE x;--"→false`, `"FOO) ; DROP"→false`.

## 4. Verification (kết quả THỰC TẾ, exit code)
- `go build ./internal/... ./cmd/worker/... ./cmd/sinkworker/...` (worker) → **EXIT 0**
- `go build ./...` (cdc-cms-service) → **EXIT 0**
- `go vet` transmute/handler/service/server → **0** (chỉ 2 cảnh báo `pkgs/idgen/sonyflake.go` sync.Once — PRE-EXISTING, không phải code này)
- `go test ./internal/service/transmute/` → **PASS** (6/6)
- `go test ./internal/handler/ -run scan_array` → **PASS** (`TestExplodePathToPGPath`, `TestValidScanIdent`)
- Security gate: 2 agent review → code mới CLEAN; SQLi class fixed toàn bộ 4 site.

## 5. Giới hạn đã biết (ghi rõ, không giấu)
- **flatten orphan**: array co lại → master row thừa (`::idx::N`) chưa auto soft-delete (deferred, ghi ở README).
- **flatten 1 cấp**: `explode_path` chỉ 1 array level (đúng giới hạn V3); `a[*].b[*]` bị reject.
- **scan-array FE**: mới có BE + endpoint; wizard FE (nút "Scan fields" + bảng review) chưa làm.
- **post_ingest realtime**: NATS plain (no JetStream) + incremental drop nếu >500 ids/batch (audit ghi nhận, chưa fix — ngoài scope đợt này).

## 6. Việc tiếp (đề xuất)
- Tổng hợp audit realtime flow vào masters-page (snapshot full + oplog incremental) — đã có dữ liệu workflow.

---

## CẬP NHẬT (2026-06-03, sau chỉ đạo User: "ko đụng db→shadow" + "thêm child_explode_master + helper" + "FE chưa làm")

### 6a. REVERT 2 edit ở luồng db→shadow (theo chỉ đạo — luồng cũ rủi ro, chỉ dùng không sửa)
| File | Trạng thái |
|------|-----------|
| `child_explode.go` (ensureChildTable, db→shadow) | **REVERT về nguyên gốc** (đã verify: 0 guard của tôi còn lại) |
| `command_handler.go HandleCreateDefaultColumns` (ALTER shadow, db→shadow) | **REVERT** guard (đã verify: 0) |
> ⚠️ Hệ quả: 2 site SQLi `data_type→DDL` ở luồng db→shadow **CHƯA fix** (theo quyết định User không đụng luồng này). Ghi nhận là **finding cần User duyệt riêng** trước khi sửa.
> GIỮ guard ở `master_ddl_generator.go` (luồng shadow→**master**, đúng vùng User muốn kiểm soát) + helper `IsTypeWhitelisted`.

### 6b. Decouple master-explode khỏi child_explode (theo "thêm child_explode_master + helper")
- MỚI `centralized-data-service/internal/service/child_explode_master.go` (74 dòng): `explodeArrayElements` + `walkToArrayMaster` — helper array-explode RIÊNG cho master, **độc lập** với `ChildExplodeService` (db→shadow).
- `transmuter.go`: `RunContext.ExtractArray` đổi `extractArrayBytes`→`explodeArrayElements`; **xóa** `extractArrayBytes` (vốn gọi `extractArrayByPath` của child_explode). Verify: `extractArrayByPath` giờ CHỈ còn dùng nội bộ trong `child_explode.go`. Master không còn coupling vào luồng shadow.

### 6c. FE flatten discovery (cdc-cms-web — trước đó CHƯA làm)
- SỬA `cdc-cms-web/src/pages/MappingFieldsPage.tsx` (716→777 dòng, ≈ +61): state `flattenOpen/explodePath/scanningArray`; handler `handleScanArray` (POST `/api/introspection/scan-array/:table` body `{explode_path}`); nút "Scan Array (Flatten)"; Modal nhập explode_path. Mirror `handleScan` — chỉ READ + reload list pending để duyệt, KHÔNG đụng db→shadow.

### Verify (THỰC TẾ)
- Worker `go build ./internal/... cmd/worker cmd/sinkworker` → **EXIT 0**; vet 0 (trừ idgen); test transmute + handler **PASS**.
- FE `npx tsc -b` → **EXIT 0**; `npm run build` → **✓ built in 486ms** (MappingFieldsPage bundled).
- Decouple verified: `extractArrayByPath` chỉ còn trong `child_explode.go`.

### Tổng files (state cuối)
- NEW: 6 file transmute/test (533) + `child_explode_master.go` (74) = 7 file.
- MODIFIED giữ: `transmuter.go`, `command_handler.go` (chỉ scan-array, KHÔNG còn guard shadow), `worker_server.go`, `type_resolver.go`, `master_ddl_generator.go`, CMS `introspection_handler.go` + `router.go`, FE `MappingFieldsPage.tsx`.
- MODIFIED rồi REVERT (net 0): `child_explode.go`.
