# Review Report — `pilot_cds` (pilot đầu tiên của skill `/diff-review`)

- **Repo:** `data-hub/centralized-data-service` | **Range:** `HEAD~1..HEAD` (`ab37d47..e82ed6e` — commit "fix update db master")
- **Ngày:** 2026-11-17 | **Agent:** Claude Fable 5.1 | **Mode:** Propose-Only
- **Rules nạp:** L1 base(4) | L2 packs: `fintech` | L3 profile: `memory/global/project/data-hub/review_profile.md`
- **Manifest:** `centralized-data-service/.agent-project.json` (mới tạo, khoá `review`) | **Scope file:** `scope_pilot_cds.json`

## 1. Verdict
**`APPROVE`** — S0: 0 | S1: 0 | S2: 2 | S3: 0
> Diff chạm chokepoint `CP-4` (TransmuteHandler) nhưng chỉ **mở rộng** (thêm 2 trường optional `omitempty` + helper `withShadowScope`), không đổi nhãn `TriggeredBy`, không đổi chữ ký hàm công khai. Hai S2 là nợ kỹ thuật god-file có từ trước, không do diff gây ra.

## 2. Scope (từ `scope.py`, EV `EV-20261117T151447-12a7`)
| Chỉ số | Giá trị |
| :--- | :--- |
| Files changed / added / deleted | `27 / 9 / 0` |
| LOC +/- | `+1078 / -176` |
| Layers touched | `handler, server, service` |
| `chokepoints_touched` | **`CP-4`** — `transmute_handler.go`, `transmute_event_date_split.go`; symbol `TriggeredBy` |
| `god_files` (> 600) | `transmute_handler.go (793)`, `schema_ddl_handler.go (739)`, `server_setup.go (604)`, `schema_adapter.go (1163)`, `scan_service.go (759)` |
| `money_path` files | 4 (keyword `_raw_data`/`trans_his`: `schema_adapter.go`, `mongo_introspection.go`, 2 test) |
| `layer_violations` | 0 |
| `forbidden_ifelse_touched` | `internal/service/source/scan_service.go` |

## 3. Findings
### [S2] `B-DES` god-file — `transmute_handler.go` tiếp tục lớn (793 LOC, +22/−16)
- **REF:** `centralized-data-service/internal/handler/master/transmute_handler.go:457-515`
- **Quote:** 7 call-site `h.activity.Complete/FailWithDetails(..., withShadowScope(req, map[...]{...}))`
- **Vì sao:** Diff đã làm đúng hướng (helper tách ra `transmute_shadow_scope.go:7`), nhưng file vẫn > `god_file_loc`; đây là nợ cũ, diff không làm xấu thêm (`+6` net).
- **Blast-radius:** không — chỉ bổ sung key vào `details` map của activity log.
- **Gợi ý sửa:** không bắt buộc trong PR này; ghi nhận cho roadmap tách `HandleTransmute` lifecycle.

### [S2] `INV-3`/`B-DES` — sửa 1 dòng trong file `forbidden_ifelse` `scan_service.go`
- **REF:** `centralized-data-service/internal/service/source/scan_service.go:401-403`
- **Quote:** `if shouldSkipTopLevelScanField(relPath) || existingV2[relPath] { continue }`
- **Vì sao:** `scan_service.go` nằm trong danh sách "CẤM nhồi if/else" (`review_profile.md` §5, nguồn `system_overview…md` §4.5). Tuy nhiên đây **không phải thêm `case` loại engine mới** mà là mở rộng điều kiện bằng helper đã có (`scan_field_filter.go:26`, đã dùng tại `scan_service.go:144`) — đúng tinh thần Open-Closed. Giữ S2 chỉ để ghi nhận cờ, không yêu cầu đổi.
- **Gate:** test `TestScanFieldFilter_*` PASS — `EV-20261117T152002-7b10`.

## 4. Blast-radius tổng hợp (CP-4 Contract lens)
| Chokepoint / Contract | Thay đổi trong diff | Callers/Consumers phải kiểm | Trạng thái |
| :--- | :--- | :--- | :--- |
| `CP-4 TransmuteRequest` JSON | +`shadow_schema`, +`shadow_table` (`omitempty`) `transmute_handler.go:299-300` | Producer CMS (`cdc.cmd.transmute`) không bắt buộc gửi → tương thích ngược | OK |
| `CP-4 TriggeredBy` 6 nhãn | **không đổi** (`grep 'case "'` HEAD vẫn `bridge-oplog` :217; diff không có dòng `TriggeredBy` nào bị sửa) | — | OK |
| `debounceRunKey` | +`shadowSchema`, +`shadowTable` → nhóm debounce tách mịn hơn theo shadow nguồn (`transmute_event_date_split.go:11-16,125-130`) | `groupDebouncedTasks*` tests cập nhật trong diff; `go test ./internal/handler/master/` PASS | OK — lưu ý multi_shadow N→1 giờ chạy N group riêng (hành vi **có đổi**, có test) |
| `activity_logs.details` | thêm 2 key | CMS read-model đã có `shadow_schema` (`cdc-cms-service/.../activity_log_read_repo_gorm_test.go:143`) | OK |

## 5. Gate (R5)
- Mode B `laya_gate.py review --max-rounds 3` trên 3 file CP-4/forbidden (551 dòng `-U30`): `06_laya_summary_dg3review_20261117T151644.json` → **`gate=PASS`**, cả 3 file `ACCEPT`.
- Dropped: `none`.

## 6. Evidence
| EV | Kind | Lệnh | exit |
| :--- | :--- | :--- | :---: |
| `EV-20261117T151447-12a7` | runtime | `scope.py --manifest .agent-project.json --pack fintech.md` | 0 |
| `EV-20261117T151728-7dce` | unit | `go build ./...` | 0 |
| `EV-20261117T151859-9992` | unit | `go test ./internal/handler/master/ -count=1` | 0 |
| `EV-20261117T152002-7b10` | unit | `go test ./internal/service/source/ -run TestScanFieldFilter_` | 0 |
| `EV-20261117T152005-4e6e` | unit | `go test ./internal/service/source/ -run TestMergeSampleDocs_` | 0 |
| `EV-20261117T151750-57bb` | unit | `go test ./internal/handler/master/ ./internal/service/source/` → **exit 1** | 1 |

**Giải trình EV `-57bb` fail:** `TestResolveSourceRoutes_GenericConnectionFallback` (`metadata_registry_service_test.go:510`) fail — **đã fail sẵn ở base `ab37d47`** (chạy lại trong `git worktree` base, cùng kết quả `FAIL`), file test và `metadata_registry_service.go` **không nằm trong diff**. Theo `INV-5` profile → không tính là regression của PR này; ghi nhận là nợ test có sẵn trên nhánh `recon-heal`.

**Lưu ý vận hành:** `evidence_run.py` không giữ được ký tự `|` trong `-run 'A|B'` (exit 127/2) → tách thành 2 lệnh. Đề xuất (ngoài scope): ghi chú trong `SKILL.md` §7.

## 7. Không-finding đáng ghi (đã kiểm, OK)
- `B-SEC-01/02`: 2 trường mới là tên schema/bảng, không phải secret/PII; không log `_raw_data` mới.
- `B-CON-01` + `INV-1`: không đổi tên/kiểu trường cũ; chỉ thêm `omitempty`.
- `F-LOG-*`: diff không chạm phép tính tiền (money_path hit do keyword `_raw_data`/`trans_his` trong `schema_adapter.go` & test; đã đọc hunk — là xử lý kiểu dữ liệu, không phải tính toán).
- `B-DES` layer: 0 vi phạm import ngược.
