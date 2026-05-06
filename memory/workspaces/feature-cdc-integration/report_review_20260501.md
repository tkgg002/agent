# Review Report: report_system_summary.md

**Ngày review**: 2026-05-01 22:24 (Asia/Ho_Chi_Minh)
**Đối tượng**: `report_system_summary.md` (viết ngày 2026-04-30 23:34)
**Phương pháp**: Exercise-driven verification — chạy lệnh thực tế, so sánh runtime vs report claims

---

## 1. Tổng quan đánh giá

| Tiêu chí | Đánh giá |
|---|---|
| **Độ chi tiết** | ⭐⭐⭐⭐⭐ Xuất sắc — 1298 dòng, bao phủ toàn bộ kiến trúc |
| **Chính xác kiến trúc** | ⭐⭐⭐⭐ Tốt — sơ đồ luồng, NATS subjects, API endpoints đều chính xác |
| **Chính xác runtime** | ⭐⭐⭐ Trung bình — có một số sai lệch so với thực tế hiện tại |
| **Blockers tracking** | ⭐⭐⭐⭐ Tốt — 8 blockers được liệt kê rõ ràng |
| **Verification quality** | ⭐⭐⭐ Trung bình — chỉ dừng ở health check, chưa exercise-driven |

**Verdict**: Report có giá trị cao về mặt tài liệu kiến trúc, nhưng cần sửa một số **sai lệch thực tế** được phát hiện trong lần review này.

---

## 2. Các sai lệch phát hiện (Discrepancies)

### 🔴 CRITICAL — Sai lệch nghiêm trọng

#### D1: Overall health = "critical", KHÔNG phải "healthy"

**Report claim (line 1235)**: "11/11 container Up. CMS + Worker process còn sống. Không phải báo cáo láo."

**Thực tế verify 2026-05-01 22:22**:
```json
{
  "overall": "critical",
  "alerts": [
    {
      "component": "reconciliation",
      "level": "critical",
      "message": "1 tables failed reconciliation check (source unreachable): [orders]"
    }
  ],
  "reconciliation": [
    {
      "table": "orders",
      "status": "error",
      "source_count": 0,
      "dest_count": 0,
      "last_check": "2026-04-29T15:44:22Z"
    }
  ]
}
```

**Impact**: Report cho ấn tượng hệ thống healthy, nhưng CMS self-report overall = **critical**. Recon cho bảng `orders` đang **error** (source unreachable). Consumer lag monitor bị **connection refused** (kafka_exporter port 9308 không chạy).

---

#### D2: Kafka Connect port sai trong diagram

**Report claim (line 64)**: `gpay-kafka-connect:8084 (Debezium)`

**Thực tế**:
```
$ docker port gpay-kafka-connect
8083/tcp -> 0.0.0.0:18083
```
- Port nội bộ: `8083` (không phải `8084`)
- Port host mapped: `18083`
- `localhost:8084` → **Connection refused**
- `localhost:18083` → OK, trả về `["cdc-pg-source","goopay-mongodb-cdc"]`

**Impact**: Nếu ai đó dựa vào report để debug connector, sẽ gọi sai port.

---

### 🟡 MEDIUM — Sai lệch trung bình

#### D3: Report nói "13 bảng chính" — thực tế 25 bảng

**Report claim (line 31)**: "4.1 cdc_system.* (control plane — 13 bảng chính)"

**Thực tế**:
```
cdc_system schema chứa 25 bảng (không tính partition children):
admin_actions, cdc_activity_log, cdc_alerts, cdc_mapping_rules,
cdc_reconciliation_report, cdc_table_registry, cdc_wizard_sessions,
cdc_worker_schedule, connection_registry, enum_types, failed_sync_logs,
mapping_rule_v2, master_binding, master_table_registry_legacy,
pending_fields, recon_runs, schema_changes_log, schema_proposal,
shadow_binding, source_object_registry, sources, sync_runtime_state,
table_registry_legacy, transmute_schedule, worker_registry
```

**Impact**: Undercount 12 bảng. Người đọc sẽ thiếu context về `recon_runs`, `sync_runtime_state`, `worker_registry`, `enum_types`, `schema_proposal`, v.v.

---

#### D4: Worker stats cho thấy zero throughput

**Report claim**: Worker đang xử lý pipeline, có "buffer batch 500"

**Thực tế**:
```json
{
  "buffer": { "buffer_size": 0, "last_flush": "0001-01-01T00:00:00Z" },
  "queue": { "processed": 0, "failed": 0, "active_workers": 10 }
}
```

**Impact**: Worker running (PID alive) nhưng đang **idle hoàn toàn** — chưa từng flush buffer kể từ khi start. `processed: 0, failed: 0`. Điều này consistent với blockers B4/B5 chặn ingest, nhưng report không highlight rõ là worker đang **zero-throughput**.

---

#### D5: MariaDB connector chưa tồn tại trong Kafka Connect

**Report claim (diagram line 62)**: `cdc-mariadb-source` connector hiển thị trong sơ đồ

**Thực tế**:
```
$ curl http://localhost:18083/connectors
["cdc-pg-source","goopay-mongodb-cdc"]
```

Chỉ có **2 connector**: PG và MongoDB. MariaDB connector **không tồn tại** (consistent với B8 nhưng diagram vẽ như đã có).

---

#### D6: Source object registry có nhiều entry "failed"

Report không mention rõ:
```
e2e_phaseD_auto        | failed
e2e_phaseD_auto_v2     | failed
e2e_phaseD_auto_v3     | failed
e2e_phaseD_auto_v4     | failed
mariadb_legacy_orders_v1 | failed
mongo_payment_bills_v2   | failed
```

6/22 source objects ở trạng thái **failed**. Đây là technical debt chưa được cleanup.

---

### 🟢 LOW — Sai lệch nhỏ

#### D7: Container count thực tế là 14, không phải 11

**Report claim (line 1232-1233)**: "11/11 container Up"

**Thực tế** (2026-05-01):
```
11 CDC containers:
  gpay-mariadb, gpay-redis, gpay-kafka-connect, gpay-schema-registry,
  gpay-mongo, gpay-kafka, gpay-nats, gpay-postgres-cdc,
  gpay-postgres-source, gpay-postgres, gpay-postgres-dest

3 non-CDC containers (also running):
  some-mongo, some-nats, some-redis
```

**Impact**: Nhỏ — 3 container `some-*` không liên quan CDC nhưng vẫn chạy, gây tiêu tốn tài nguyên.

---

#### D8: CMS health endpoint chỉ check 1 Debezium connector

```json
"debezium": {
  "connector": "goopay-mongodb-cdc",  // ← chỉ check MongoDB
  "status": "RUNNING"
}
```

CMS health check bỏ qua `cdc-pg-source` connector. Nếu PG connector die, health vẫn báo Debezium = RUNNING.

---

## 3. Xác nhận những gì chính xác (Confirmed Accurate)

| Claim | Verified |
|---|---|
| CMS PID 13653, Worker PID 20450 alive | ✅ Confirmed (uptime từ Wed04PM) |
| 11 CDC containers Up | ✅ Confirmed (tất cả Up 2-3 ngày) |
| Health endpoints 200 (worker, cms, auth) | ✅ Confirmed |
| CMS auth: `/api/v1/source-objects` → 401 (JWT required) | ✅ Confirmed |
| Worker ready: `{"status":"ready"}` | ✅ Confirmed |
| Frontend dev server port 5173 | ✅ Confirmed (HTTP 200) |
| Shadow tables: 11 tables across 4 schemas | ✅ Confirmed |
| Master (dw_) tables: 10 tables across 4 schemas | ✅ Confirmed |
| Debezium PG + MongoDB connectors RUNNING | ✅ Confirmed |
| Blockers B1/B2 FIXED, B3-B8 OPEN | ✅ Consistent with runtime data |
| NATS: 3 streams, 1 consumer, 4 messages | ✅ Confirmed via health |
| Infrastructure (Kafka, NATS, PG, Redis) all UP | ✅ Confirmed |
| Partitioned tables pattern (RANGE) | ✅ Confirmed (admin_actions, cdc_activity_log, failed_sync_logs) |

---

## 4. Recommendations

### Immediate (phải sửa trong report)
1. **Sửa port Kafka Connect**: `8084` → `18083` (host) / `8083` (internal)
2. **Thêm mục "Runtime Warning"**: overall health = critical, consumer lag unknown, recon error
3. **Sửa table count**: "13 bảng chính" → "25 bảng" (liệt kê đầy đủ)
4. **Ghi rõ MariaDB connector**: diagram nên dashed/dotted line + note "NOT YET DEPLOYED (B8)"

### Short-term (nên làm)
5. **Fix kafka_exporter**: Port 9308 bị refused → consumer lag monitoring mù
6. **Fix CMS health**: Nên check TẤT CẢ connector, không chỉ 1
7. **Cleanup failed source objects**: 6 entries failed cần archive hoặc retry
8. **Worker zero-throughput**: Highlight rõ trong report rằng data plane đang idle

### Long-term
9. **Automated verify script**: Viết script tự động chạy exercise-driven verification thay vì manual check
10. **Recon fix**: `orders` table recon error vì source unreachable — cần investigate root cause

---

## 5. Kết luận

Report `report_system_summary.md` là một **tài liệu kiến trúc xuất sắc** — bao phủ toàn bộ luồng từ Debezium → Kafka → Worker → Shadow → Master, NATS event bus, CMS API routes, provisioning state machine, và DLQ flow.

Tuy nhiên, report **thiếu chiều sâu về runtime health**:
- Hệ thống self-report **overall = critical** (không phải healthy)
- Worker đang **idle hoàn toàn** (0 processed, 0 failed, never flushed)
- Consumer lag monitoring **bị mù** (kafka_exporter down)
- Reconciliation **error** cho bảng orders

Những sai lệch này không phải "báo cáo láo" mà là **verify chưa đủ sâu** — chỉ check health endpoint 200 mà không đọc response body. Lesson: **"HTTP 200 ≠ Healthy. Phải đọc response body."**

---

## 6. Skills sử dụng

- **Exercise-driven verification** — curl + parse response body, không chỉ status code
- **Docker port mapping** — `docker port` để tìm actual port
- **DB schema audit** — `\dt` để đếm tables thực tế
- **Cross-reference** — so sánh report claims vs runtime facts
- **Lesson application** — áp dụng lesson "health ≠ healthy", "verify before done"
- **Governance compliance** — §3 Plan & Verify, §7 Memory, §9 Workspace-First, §11 No Overwrite, §12 Brain Code Prohibition
