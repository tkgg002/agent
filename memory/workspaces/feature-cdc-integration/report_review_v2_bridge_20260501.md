# Review Report: report_v2_bridge_e2e_verification.md

**Ngày review**: 2026-05-01 23:17 (Asia/Ho_Chi_Minh)
**Đối tượng**: `report_v2_bridge_e2e_verification.md` (viết ngày 2026-04-30, loop dynamic mode)
**Phương pháp**: Exercise-driven verification — re-run TOÀN BỘ queries trong report, so sánh output thực tế

---

## 1. Tổng quan đánh giá

| Tiêu chí | Đánh giá |
|---|---|
| **Cấu trúc** | ⭐⭐⭐⭐⭐ Rõ ràng, phân tầng control/data plane |
| **Chính xác control-plane** | ⭐⭐⭐⭐⭐ Xuất sắc — 100% claims verified |
| **Chính xác data-plane** | ⭐⭐⭐ Trung bình — có 2 sai lệch quan trọng |
| **Blockers tracking** | ⭐⭐⭐⭐ Tốt — nhưng B5 claim không khớp runtime |
| **Khuyến nghị B6** | ⭐⭐ Yếu — đề xuất đọc column không tồn tại |
| **Governance** | ⭐⭐⭐⭐⭐ Tuân thủ §12 (no code), §11 (append-only) |

**Verdict**: Report có chất lượng tốt về mặt control-plane verification. Tuy nhiên **2 sai lệch cần sửa** và **1 khuyến nghị B6 không khả thi**.

---

## 2. Các sai lệch phát hiện (Discrepancies)

### 🔴 D1: Master DW tables ĐÃ TỒN TẠI — report claim "CHƯA TỒN TẠI" SAI

**Report claim (Section 1.7, line 89-96)**:
> ```
> ERROR: relation "dw_src_local_pg_source.orders_addtest" does not exist
> ```
> "Transmuter chưa CREATE master table vì fail ngay ở fetch shadow batch step trước DDL."

**Thực tế verify 2026-05-01 23:16**:
```
dw_mariadb_legacy_default.legacy_orders_addtest     → EXISTS, 0 rows
dw_mongo_payment_bill_default.payment_bills_addtest  → EXISTS, 0 rows
dw_src_local_pg_source.orders_addtest                → EXISTS, 0 rows
```

**Impact**: Master tables **ĐÃ được tạo** (DDL đã chạy thành công bởi MasterDDLGenerator.Apply). Transmute fail ở bước `fetchShadowBatch` NHƯNG DDL creation là bước riêng (xảy ra lúc provisioning `master_pending → master_active`, không phải lúc transmute). Report nhầm lẫn 2 bước này.

**Root cause sai**: Report giả định transmuter phải CREATE table trước khi fetch — nhưng thực tế provisioning cascade đã tạo table rồi. Transmute chỉ INSERT dữ liệu.

---

### 🟡 D2: B5 claim "3 messages offset 55/56/57 stuck DLQ" — DLQ TRỐNG

**Report claim (line 189-190)**:
> "base64-encode Avro bytes trước INSERT. Hiện tại 3 message offset 55/56/57 stuck redelivery loop, burns CPU."

Và từ `report_system_summary.md` (line 1254):
> "goopay_source.public.orders còn 3 rows ids 56-58 với notes LIKE 'track-e-test-%' đang stuck DLQ redelivery loop"

**Thực tế verify 2026-05-01 23:17**:
```sql
SELECT count(*) FROM cdc_system.failed_sync_logs;
-- Result: 0
```

**3 test rows tồn tại trong source**:
```
56|track-e-test-1
57|track-e-test-2
58|track-e-test-3
```

**Impact**: DLQ **trống hoàn toàn**. Nếu B5 claim đúng thời điểm viết report (04-30), có 2 khả năng:
1. DLQ entries đã bị cleanup/purge giữa 04-30 và 05-01
2. Report ước đoán từ gap analysis cũ mà không query trực tiếp

Dù lý do gì, claim hiện tại **không reflect reality**. 3 test rows vẫn ở source nhưng KHÔNG ở DLQ — có thể Kafka consumer đã commit offset qua rồi (messages "lost", không retry, không DLQ).

---

### 🟡 D3: Khuyến nghị B6 đề xuất đọc `shadow_pk_column` — column KHÔNG TỒN TẠI

**Report claim (Section 4.1 item 1, line 180-182)**:
> "Sửa query fetch shadow batch đọc PK column từ shadow_binding.shadow_pk_column (live-resolved)"

**Thực tế**:
```
shadow_binding columns: id, binding_code, source_object_id, shadow_connection_id,
shadow_database, shadow_schema, shadow_table, physical_table_fqn,
namespace_strategy, write_mode, ddl_status, is_active, created_at, updated_at
```

**KHÔNG CÓ `shadow_pk_column`**. Nếu Muscle thực thi khuyến nghị này, sẽ fail ngay vì column chưa tồn tại — cần migration thêm column trước.

**Impact**: Khuyến nghị incomplete — thiếu bước migration prerequisite.

---

### 🟢 D4: Transmute `last_run_at` đã update — cron vẫn đang fire

**Report claim (line 71)**: `last_run_at: 2026-05-01 15:40:45`

**Thực tế 2026-05-01 23:16**: `last_run_at: 2026-05-01 16:15:45`

Không sai — chỉ là thời gian trôi. Cron `*/1 * * * *` vẫn fire mỗi phút. Mỗi lần fire đều fail với cùng error B6. Đây là **CPU waste** — cron keep firing mà biết trước sẽ fail.

---

### 🟢 D5: Column names nhỏ lệch

Report section 1.1 dùng `source_object_name` trong query SQL (line 15) — column này tồn tại, nhưng `object_code` mới là identifier chính:
- `object_code`: addtest_pg_orders (unique key, internal)
- `source_object_name`: orders_addtest (display name)

Không sai nhưng dễ gây nhầm — object_code mới là cái dùng trong code.

---

## 3. Xác nhận những gì CHÍNH XÁC (All Verified ✅)

| Report Claim | Verified |
|---|---|
| ID 29/30/31 `provisioning_state=running` | ✅ Confirmed |
| ID 29/30/31 `profile_status=active` | ✅ Confirmed |
| ID 29/30/31 `last_step_error=NULL` | ✅ Confirmed (column exists, no error) |
| shadow_binding 38/39/40 `ddl_status=created` | ✅ Confirmed |
| shadow schemas: src_local_pg_source, mariadb_legacy, mongo_payment_bill | ✅ Confirmed |
| master_binding 28/29/30 `schema_status=approved, is_active=t` | ✅ Confirmed |
| mapping_rule_v2 counts: 7/7/4 | ✅ Confirmed |
| transmute_schedule 13/14/15 `last_status=failed` | ✅ Confirmed |
| Error: `_gpay_id does not exist (SQLSTATE 42703)` | ✅ Confirmed |
| Shadow tables: 0 rows (all 3) | ✅ Confirmed |
| Source orders: 58 rows | ✅ Confirmed |
| Shadow schema: PK=id text, no `_gpay_id` | ✅ Confirmed (exact schema match) |
| B1+B2 FIXED | ✅ Confirmed (state=running, ddl=created) |
| B6 root cause: `_gpay_id` hardcode in transmuter.go | ✅ Confirmed in code |
| B8 MariaDB connector not deployed | ✅ Confirmed (only cdc-pg-source + goopay-mongodb-cdc) |
| Debezium connectors RUNNING | ✅ Confirmed |
| Worker processed=0, buffer never flushed | ✅ Confirmed |

---

## 4. Phân tích code B6 — verify blast radius

Report claim (line 182): blast radius = `transmuter.go fetchShadowBatch()`.

**Code verify** (grep kết quả):
```go
// transmuter.go:
type shadowBatchRow struct {
    GpayID int64 `gorm:"column:_gpay_id"`     // ← hardcode _gpay_id
}

qt := fmt.Sprintf(`SELECT _gpay_id, _gpay_source_id, _raw_data, _source_ts, _gpay_deleted
    WHERE _gpay_id > ?`, ...)                    // ← hardcode _gpay_id in SELECT + WHERE
qt += ` ORDER BY _gpay_id LIMIT ?`               // ← hardcode _gpay_id in ORDER BY

record["_gpay_id"] = row.GpayID                  // ← hardcode in record map
if k == "_gpay_id" || k == "_gpay_source_id" {   // ← hardcode in exclusion
case "_gpay_id", ...:                             // ← hardcode in switch

// master_ddl_generator.go:
`"_gpay_id" BIGINT PRIMARY KEY`                   // ← hardcode in DDL generation
seen := map[string]bool{"_gpay_id": true, ...}    // ← hardcode in seen map
```

**Report claim blast radius = chỉ `fetchShadowBatch`**: ❌ **INCOMPLETE**.

Thực tế `_gpay_id` hardcode ở **ÍT NHẤT 2 files, 9+ locations**:
- `transmuter.go`: 7 locations (struct, SELECT, WHERE, ORDER BY, record map, exclusion, switch)
- `master_ddl_generator.go`: 3 locations (DDL template, seen map, case statement)

---

## 5. Recommendations

### Immediate (sửa trong report)
1. **D1**: Sửa Section 1.7 — Master tables ĐÃ tồn tại (DDL created by provisioning cascade, NOT by transmuter)
2. **D2**: Sửa B5 claim — DLQ hiện TRỐNG, messages có thể đã lost (committed offset without DLQ write)
3. **D3**: Sửa khuyến nghị B6 — thêm "cần migration thêm `shadow_pk_column` vào shadow_binding trước"

### Short-term
4. **Disable cron** cho schedule 13/14/15 (is_enabled=false) — đang burn CPU mỗi phút fire fail
5. **B6 blast radius chính xác**: 9+ locations across 2 files, KHÔNG chỉ `fetchShadowBatch`
6. **Investigate B5 message loss**: 3 test rows ở source, 0 DLQ, 0 shadow — dữ liệu đi đâu? Kafka consumer commit offset nhưng không ingest?

---

## 6. Kết luận

Report `report_v2_bridge_e2e_verification.md` là một **báo cáo kiểm chứng control-plane xuất sắc** — đã walk toàn bộ provisioning cascade từ draft → running, verify dual-write mapping_rule_v2, confirm cron schedule firing.

**Sai lệch nghiêm trọng nhất**: Claim master tables "CHƯA TỒN TẠI" là **SAI** — tables đã được tạo bởi provisioning cascade (bước master_pending → master_active), không phải bởi transmuter. Report nhầm lẫn DDL creation (provisioning) với data insertion (transmute).

**B5 concern**: DLQ trống có nghĩa messages không được ghi DLQ — hoặc consumer skip messages mà không retry, hoặc SchemaValidator reject trước khi vào DLQ path. Đây là **data loss risk** cần investigate.

---

## 7. Skills sử dụng

- **Exercise-driven verification** — re-run mọi query trong report, so sánh output
- **DB schema archaeology** — `information_schema.columns` verify column existence  
- **Code grep** — verify blast radius B6 trong source code
- **Cross-reference** — so sánh report claims vs runtime facts vs code
- **Lesson application** — "health ≠ healthy", "verify before done", "response body matters"
- **Governance**: §3 (Plan & Verify), §7 (Memory Retention), §11 (Append-only), §12 (Brain Code Prohibition)
