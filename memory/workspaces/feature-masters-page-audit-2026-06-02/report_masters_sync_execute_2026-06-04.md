# report_masters_sync_execute_2026-06-04.md — Khôi phục Sync Modal + Activity Log

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-03→04
> **Verb**: execute trọn gói (P0+P1+P2) theo `02_plan_sync_execute_2026-06-03.md`.
> **Nguyên tắc tuân thủ**: Simplicity First, đúng pattern source, core-systems, không cheat DB/config. Báo cáo dựa trên kết quả THỰC.

---

## 1. Tóm tắt
Khôi phục FE Sync Modal 3-mode trên `/masters` (đã bị Gemini ghi đè), thêm lại `flatten`, **bổ sung Activity Log cho pipeline transmute** (yêu cầu cốt lõi "có lưu lại log activity" — trước đó THIẾU hoàn toàn), và vá robustness (M1/M4) + edit rule (M3) + tooltip (L4).

Cơ chế chạy **giữ nguyên core**: Sync Modal → API schedules sẵn có → `bus.Dispatch(TransmuteRunCommand)` → NATS `cdc.cmd.transmute` → `HandleTransmute` → `TransmuterModule.Run` (đọc `mapping_rule_master`, map Shadow→Master). KHÔNG thêm endpoint mới, KHÔNG đổi config.

---

## 2. Những file đã thay đổi & số dòng (ước tính theo edit phiên này)

> Lưu ý trung thực: 3 repo đã có git nhưng đang có **công việc uncommitted từ phiên trước (Gemini)**, nên `git diff` thô sẽ trộn lẫn. Số LoC dưới đây đếm theo các edit TÔI thực hiện trong phiên này.

### WORKER `centralized-data-service`
| File | Thay đổi | ~LoC |
|------|----------|------|
| `internal/handler/transmute_handler.go` | Inject `*service.ActivityLogger`; import `model`; trong `HandleTransmute` bọc `svc.Run` bằng `activity.Start("transmute",master,triggeredBy)` → `Complete(rows=Inserted+Updated, details{stats})` / `Fail(err)`. | +27 / -3 |
| `internal/server/worker_server.go` | Khai báo `activityLogger` trước `transmuteHandler` + truyền vào `NewTransmuteHandler`; xoá khai báo trùng ở dưới. | +2 / -1 |

### CMS `cdc-cms-service`
| File | Thay đổi | ~LoC |
|------|----------|------|
| `internal/app/commands/create_master.go` | M1: bọc `Exec(INSERT mapping_rule_master)` bằng `if .Error != nil { logger.Error(...) }` (fail-soft, hết nuốt lỗi). | +6 / -1 |
| `internal/api/master_mapping_rule_handler.go` | M4: guard `shadow.ShadowSchema == ""` trong Flatten (tránh `FROM ""."tbl"` → 500). | +2 / -2 |

### FRONTEND `cdc-cms-web`
| File | Thay đổi | ~LoC |
|------|----------|------|
| `src/pages/MasterRegistry.tsx` | H1: import Radio/Tooltip/SyncOutlined/InfoCircleOutlined; state `syncRow`/`syncForm`; mutation `syncMut` (run_now=upsert immediate+find+run-now lọc theo master_table; cron/post_ingest=upsert); cột "Sync" (disable nếu chưa approve); Sync Modal Radio 3-mode + Tooltip. H2: `TRANSFORM_TYPES` thêm `flatten` + prefill spec explode_path + hint. | +152 / -2 |
| `src/pages/MasterMappingFieldsPage.tsx` | M3: state `editOriginal`; nút "Sửa" prefill modal (merge giữ status/notes/...); khoá `target_column` khi edit (tránh tạo trùng do upsert natural key); reset khi cancel/add. | +42 / -4 |
| `src/pages/TransmuteSchedules.tsx` | L4: import Tooltip/InfoCircleOutlined; option `post_ingest` kèm Tooltip giải thích realtime. | +12 / -1 |

### Infra
| File | Thay đổi | ~LoC |
|------|----------|------|
| `data-hub/.gitignore` | Tạo mới (loại node_modules/vendor/dist/*.zip/*.log/secrets). | +30 |

**Tổng ~code thay đổi phiên này: ≈ +273 / -14 dòng** (7 file source + 1 .gitignore).

---

## 3. Quyết định Simplicity (ghi rõ, không phải bỏ sót)
- **M2 (explode_path input): KHÔNG thêm.** Backend Flatten (`discoverJsonPaths/extractPaths`, `master_mapping_rule_handler.go:307-342`) **đã tự bóc nested + array `[*]`** từ `source_field`. Thêm input thủ công là thừa + dễ lệch. Auto-discovery đơn giản & đúng hơn plan gốc. (Có prefill `explode_path` ở spec Create Master cho transform_type=flatten như gợi ý.)
- **Git data-hub: KHÔNG init.** Phát hiện cả 3 service ĐÃ là git repo riêng (`cdc-cms-web/.git`, `cdc-cms-service/.git`, `centralized-data-service/.git`). Đã gỡ `data-hub/.git` lỡ tạo. Rủi ro thật = công việc đang **uncommitted** (xem §6).

---

## 4. Verification (exit code THỰC TẾ — đã tự chạy)
| Hạng mục | Lệnh | Kết quả |
|---|---|---|
| Worker build | `go build ./...` | **EXIT 0** ✅ |
| Worker vet | `go vet ./internal/handler/ ./internal/server/` | sạch (chỉ `pkgs/idgen` pre-existing) ✅ |
| Worker test | `go test ./internal/handler/ ./internal/service/` | **PASS** (handler 0.9s, service cached) ✅ |
| CMS build | `go build ./...` | **EXIT 0** ✅ |
| CMS vet | `go vet ./internal/app/commands/ ./internal/api/` | sạch ✅ |
| FE typecheck | `npx tsc -b` | **EXIT 0** ✅ |
| FE build | `npm run build` | **✓ built** (MasterRegistry 12.36→16.40 kB) ✅ |

---

## 5. Audit end-to-end "Sync Modal nhấn chạy → chạy thật + có log activity"
**Stack đang chạy** (xác nhận live): CMS `:8083`=200, NATS `:4222`, PG control `:5433`/shadow `:5436`/dest `:5434`, FE vite `:5173`, worker `go run cmd/worker/main.go`.

**Chứng minh bằng read-only DB (cdc_dw, gpay_admin) + static trace:**
1. **Chain chạy thật** (verified-by-construction + trace `a72e5e…`): `POST /schedules/:id/run-now` → `TransmuteRunCommand{master_table}` → subject `cdc.cmd.transmute` → `HandleTransmute` → `Run`. Subscriber `worker_server.go:412` đăng ký **vô điều kiện** (không black-hole). FE `syncMut` gọi đúng các endpoint này (mirror pattern `TransmuteSchedules.tsx` đã chạy).
2. **Gap activity-log là THẬT**: `SELECT operation,count(*) FROM cdc_system.cdc_activity_log GROUP BY 1` → có snapshot.v2/alter-column/transform/bridge… **KHÔNG có `transmute`**. Code mới ghi operation=`transmute` (Start→Complete/Fail).
3. **Target hợp lệ tồn tại**: master `sssss` (id=5) `approved`+`active`, `mapping_rule_master` 9 rule approved+active → run-now sẽ có rule để transmute.

**Trạng thái live của activity-log**: worker đang chạy là **binary CŨ** (trước thay đổi). Code activity-log đã build + wire đúng chokepoint nhưng **chưa load vào process đang chạy**. ⇒ **CHƯA tạo được row `transmute` LIVE** (không báo láo). Để thấy row live: **restart worker** (`go run cmd/worker/main.go`) rồi Sync `sssss` mode "Chạy ngay" → kiểm tra `cdc_activity_log` xuất hiện row operation=`transmute`, target=`sssss`, status running→success.

> Không tự restart vì stack live đa-process (`go run`/`/main` con khó phân định) — tránh sập nhầm service khác. Sẵn sàng restart + demo live khi User đồng ý.

---

## 6. Rủi ro tồn đọng (cần User quyết)
- **Uncommitted work pile**: cdc-cms-web (5), cdc-cms-service (21), centralized-data-service (10) — tất cả việc masters-page (gồm file untracked `MasterMappingFieldsPage.tsx`, `master_mapping_rule_handler.go`, migration 073/074) **chưa commit**. Đây là nguồn cơn "mất việc". **Đề xuất**: tạo restore-point commit/repo (local, KHÔNG push theo §8). Chờ User chốt (harness: chỉ commit khi User yêu cầu).
- **Migration 074 (`mapping_rule_master`) + 073 (flatten)** đã có file nhưng cần xác nhận đã apply vào `cdc_dw` (boot auto-migrate). Read-only cho thấy bảng `mapping_rule_master` đã có data ⇒ đã apply.

---

## 7. Security self-review (§8 gate)
Thay đổi phiên này: KHÔNG endpoint mới, KHÔNG SQL string-interpolation mới. Activity log ghi qua GORM `Create/Updates(map)` (parameterized) + `SanitizeFreeformText/SanitizeNestedStrings`. FE gọi API parameterized, không injection. `.gitignore` loại secrets. → Không phát sinh lỗ hổng mới.
