# 06 — Validation Plan: Audit V4

## V1. Cách ly 2 connector trùng tên bảng (Postgres docker, CMS repo thật)
Seed:
```sql
-- 2 shadow_binding active cùng shadow_table='payment_bills', schema khác nhau: sA (id=A), sB (id=B, updated_at mới hơn)
INSERT INTO cdc_activity_log(operation,target_table,status,triggered_by,started_at,created_at,details) VALUES
 ('kafka-consumer','payment_bills','success','kafka-consumer',now(),now(),'{"c":1,"shadow_schema":"sA"}'),
 ('kafka-consumer','payment_bills','success','kafka-consumer',now(),now(),'{"c":2,"shadow_schema":"sB"}'),
 ('kafka-consumer','payment_bills','success','kafka-consumer',now(),now(),'{"c":9}'); -- legacy, mơ hồ
```
| Case | Kỳ vọng |
|---|---|
| List `shadow_schema=sA` | đúng 1 dòng (c=1) |
| List `shadow_schema=sB` | đúng 1 dòng (c=2); dòng legacy hiện với `scope_ambiguous=true` (hành vi cũ) |
| Clear sA | `rows_deleted=1`; dòng sB + legacy còn nguyên |
| Clear sB | `rows_deleted=1`; legacy còn nguyên (bị guard ambiguity) |

## V2. Handler/route (negative path)
| Request | Kỳ vọng |
|---|---|
| DELETE thiếu `shadow_schema` | 400 |
| `before=2026-10-07` (không RFC3339) | 400 |
| role `operator` | 403 (RequireOpsAdmin) |
| thiếu `Idempotency-Key` | 400 (middleware) |
| hợp lệ | 200 + 1 dòng audit trong bảng audit admin |

## V3. Snapshot marker end-to-end
- Trigger snapshot 1 bảng có post_ingest → log transmute có `triggered_by='snapshot-hook'`; luồng CDC thường → `'kafka-consumer-hook'`.
- Mẻ debounce trộn 2 trigger → 2 log transmute riêng (unit test `groupDebouncedTasks` + log thật).

## V4. Số liệu theo ngày
- Log lúc 01:00 giờ VN rơi đúng ngày local.
- Poison-pill split: tổng Ins/Upd ngày = tổng các `transmute_sub` success.

## Bằng chứng
Ghi lệnh + output vào `05_progress.md` khi thực thi (Rule #14 G8). Chưa chạy — plan đang chờ APPROVE.

## KẾT QUẢ THỰC THI — 2026-10-12 11:16 (data thật cdc_dw @ gpay-postgres-cdc:5433)
Phương pháp: harness tạm `cdc-cms-service/scratch/v4clear/main.go` gọi **chính `ActivityLogReadRepo.ClearActivity`** qua GORM trong transaction, seed thêm binding `shadow_v4test_conn2.payment_bills` (trùng tên bảng với binding 192 `shadow_traitestpbs.payment_bills`) + 7 log marker, mỗi kịch bản ROLLBACK. Harness đã xoá sau khi chạy.

| # | Kịch bản | Kỳ vọng | Kết quả |
|---|---|---|---|
| S1 | Clear A (schema A + master `payment_bills`) | xoá A_shadow, A_transmute, A_transmute_sub_legacy; giữ B, legacy ambiguous, bảng khác, master khác | PASS (rows_affected=2041 gồm 2038 transmute log thật cùng scope) |
| S2 | Clear B (không master) | chỉ xoá B_shadow | PASS (1) |
| S3 | Clear A không master_table | transmute giữ nguyên | PASS (1) |
| S4 | before = hôm qua | không xoá log hôm nay | PASS |
| S5 | master_table FQN (bug FE cũ) | transmute KHÔNG match → chứng minh BUG-P1-FE | PASS (1) → đã fix FE |
| S6 | thiếu schema | `ErrClearScopeRequired` | PASS |
| — | Leak check sau rollback | 0 dòng `v4test` | PASS (0) |

Partition pruning: `EXPLAIN DELETE ... created_at < '2026-10-08'` → plan không chứa `cdc_activity_log_20261008/20261009` (chỉ partition < bound + default).

Build/test:
- CDS: `go test ./internal/handler/master/ ./internal/handler/shadow/` → ok.
- CMS: `go build ./cmd/... ./internal/...` + `go vet` (system api/persistence/router) → OK.
- Web: `npm run build` → OK (chỉ cảnh báo chunk size có sẵn).

Security (T12): DELETE route chỉ ở `destructiveChain` (RequireOpsAdmin + Idempotency + Audit), route cũ ở `shared` đã gỡ; SQL chỉ dùng tham số bind (không nối chuỗi input); `reason` chỉ đi qua header `X-Action-Reason`; scope bắt buộc (400 + ErrClearScopeRequired) → không có wipe toàn cục.
