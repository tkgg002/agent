# 10_gap_analysis.md — Năng lực Transform hiện tại vs Yêu cầu

> Ground: workflow `map-transmute-capabilities` (7 agent). Mọi dòng có evidence file:line.

## A. Cái GÌ engine transmute làm được hôm nay (row-level, Go)

| Khả năng | Cơ chế | Tầng | Evidence |
|----------|--------|------|----------|
| **1:1 column map** | `copy_1_to_1`: mỗi `mapping_rule_v2` → 1 cột master | Transmuter (Go, per-row) | `transmuter.go:385-433` |
| **jsonpath-extract** | `gjson.Get(_raw_data, source_path)` dot-path | Transmuter (Go) | `transmuter.go:389-407` |
| **type-cast** | `convertType()` INT/NUMERIC/BOOL/TIMESTAMP/JSONB/TEXT | Transmuter (Go) | `dynamic_mapper.go:210-235` |
| **7 transform_fn (cell-level)** | mongo_date_ms, oid_to_hex, bigint_str, numeric_cast, lowercase, jsonb_passthrough, null_if_empty | Transmuter (Go) | `transform_registry.go:31-39` |
| **mask** | hmac / aes_gcm / json_mask theo `mask_strategy` | Transmuter (Go) | `dynamic_mapper.go:143-162` |
| **flatten JSON (scalar)** | mapping rule với `source_path = after.a.b.c` | Transmuter (Go) | `transmuter.go:391-396` |
| **flatten JSON (array, 1 cấp)** | V3 `ChildExplodeService`: `explode_path` → N child **shadow** row | Shadow (Go, EventHandler) | `child_explode.go:55-132` |
| **OCC upsert + hash-dedup** | `INSERT ... ON CONFLICT(_source_id) DO UPDATE WHERE _hash IS DISTINCT` | Transmuter (SQL exec) | `transmuter.go:455-463` |
| **incremental (_source_id keyset)** | post_ingest chỉ transmute các id vừa ghi | Transmuter (Go) | `transmuter.go:333-337` |
| **soft-delete propagate** | copy `_deleted` flag (không xoá vật lý) | Transmuter (Go) | `transmuter.go:367` |

## B. Cái GÌ KHÔNG làm được (đúng các loại User hỏi)

| Loại sync | Hiện trạng | Bằng chứng |
|-----------|-----------|------------|
| **filter** (bỏ row theo điều kiện) | ❌ enum-only, KHÔNG runtime. Chỉ có keyset `_source_id` + skip khi field non-null thiếu | `notSupported` transmuter; `master_binding` enum `032:16-17` |
| **group_by** | ❌ enum-only, không có GROUP BY ở Go | `transmuter.go` không sinh aggregate SQL |
| **aggregate** (SUM/COUNT/AVG) | ❌ enum-only, không có | idem |
| **join** (cross-table) | ❌ enum-only; mỗi binding map đúng 1 shadow table | `fetchShadowBatch` 1 bảng |
| **custom_sql** | ❌ enum-only, không dispatch | `master_binding` enum |
| **conditional / expression** | ❌ `source_format='expression'` khai báo ở `033` CHECK nhưng không implement | `notSupported` U1 |
| **multi-source-row → 1 master-row** | ❌ mỗi shadow row → đúng 1 master row | `transmuter.go:385-433` |
| **flatten array đệ quy** `a[*].b[*]` | ❌ V3 MVP chỉ 1 cấp | `child_explode.go:245` |
| **auto full-document flatten** | ❌ mọi field phải khai báo mapping rule thủ công | U3 |
| **child-explode → master** | ❌ row exploded chỉ nằm ở child **shadow**, CHƯA nối lên master table BI query được | U3 reportingGap #2 |

> 🔑 **Điểm cốt lõi**: `master_binding.transform_type` CHẤP NHẬN 6 giá trị (copy_1_to_1, filter, aggregate, group_by, join, custom_sql) ở tầng CMS API/CHECK constraint, **nhưng TransmuterModule chỉ branch trên `copy_1_to_1`**. 5 loại còn lại = placeholder, **chạy ra kết quả copy 1:1 im lặng** → đây chính là "loại sync chưa có pattern để thực hiện".

## C. Tầng Master / DW (goopay_dest 5434)

- Bản sao **1:1 typed vật lý**: schema `dw_<binding>` (vd `dw_payment.payment_fact`), 11 system col + cột typed từ mapping rule. RLS bật cho master ở `public`. — `master_ddl_generator.go:61-202`
- **KHÔNG có**: VIEW, MATERIALIZED VIEW, mart, gold, aggregate table — grep 0 kết quả. — U4 `notSupported`
- → Khoảng cách tới "Metabase-ready" = **toàn bộ tầng mart/aggregate/dimensional**.

## D. Reconciliation (đối soát) hiện tại

- 3-tier **đếm row + XOR-hash fingerprint** (existence/drift), heal OCC upsert. — `recon_core.go:431-758`
- **KHÔNG** so SUM(amount), KHÔNG GROUP BY theo ngày nghiệp vụ. `ReconciliationReport` chỉ có count, **không có cột amount**. — `reconciliation_report.go:8-36`
- ✅ Thuận lợi: cột tài chính (amount/fee/balance...) ĐÃ được tạo index ở DDL (`financialIndexRe` `master_ddl_generator.go:46`) → query `SUM(amount) GROUP BY day` trên dest **khả thi**, chỉ thiếu plumbing recon.
- Để đối soát số tiền cần: (a) bảng/cột report tài chính (sum_src, sum_dst, diff, group_key), (b) `AmountReconAgent` (SUM GROUP BY day: Mongo `$group` vs PG SQL), (c) trigger/scheduler, (d) view cho Metabase. Tái dùng scaffolding window/lock/recon_runs.

## E. BI / Metabase

- **HOÀN TOÀN VẮNG MẶT**: không có container/env/config Metabase/Superset ở bất kỳ đâu. — U6
- Grafana chỉ có (Prometheus metrics, không SQL). `centralized-export-service` xuất CSV **trực tiếp từ MongoDB source**, bỏ qua dest-DW. — U6
- Để phục vụ Metabase cần: (1) read-only role trên goopay_dest, (2) schema `reporting` + view curated, (3) deploy Metabase trỏ vào **replica** (không đụng write path live).
