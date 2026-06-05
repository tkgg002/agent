# 02_plan.md — Kiến trúc Transform Patterns cho DW (Report / Đối soát / Metabase)

## 1. Nguyên lý phân tầng (quyết định kiến trúc cốt lõi)

> **Transform row-level** (xác định trên TỪNG row) → ở **Transmuter (Go)**.
> **Transform set-level/quan hệ** (filter-báo-cáo, group by, aggregate, join) → ở **tầng Mart SQL** trên DW.
> KHÔNG ép engine row-by-row (batch 500, không cross-row state) làm GROUP BY/JOIN — đi ngược kiến trúc, fragile.

```
Source (Mongo/PG/Maria)
  → Debezium/Kafka
  → SHADOW  (cdc_dw 5433)      _raw_data JSONB + system cols        [BRONZE - raw]
  → Transmuter (Go): 1:1 + scalar transform + mask + flatten-scalar
  → MASTER  (goopay_dest 5434, dw_<binding>) typed 1:1 fact         [SILVER]  ← hệ thống DỪNG ở đây
  ─────────────────────────  (phần MỚI bên dưới)  ──────────────────────────
  → Mart SQL: views / materialized views / scheduled rollups
  → REPORT  (goopay_dest, schema `reporting`) filter/join/groupby/aggregate  [GOLD]
  → read-only role
  → METABASE / BI
```

## 2. Map từng loại sync → tầng thực thi + cơ chế

| Loại sync | Bản chất | Tầng thực thi | Cơ chế đề xuất | Hiện trạng |
|-----------|----------|---------------|----------------|-----------|
| **1:1 map** | 1 field → 1 cột | Transmuter (Go, row) | `mapping_rule_v2` + gjson (`copy_1_to_1`) | ✅ Đã có |
| **Scalar transform** | cast/format | Transmuter (Go, cell) | `transform_fn` (7 fn) | ✅ Đã có |
| **Mask** | PII | Transmuter (Go, cell) | `mask_strategy` | ✅ Đã có |
| **Flatten JSON (scalar)** | `a.b.c` → cột | Transmuter (Go) | mapping rule `source_path` | ✅ Đã có (thủ công) — *nên thêm auto-suggester* |
| **Flatten JSON (array)** | `items[]` → row độc lập | Shadow→Master | nối **V3 child-explode** lên transmute→master | 🟡 Một phần (chưa lên master) |
| **Filter** | bỏ row theo điều kiện | (a) Transmuter row-skip / (b) Mart `WHERE` view | predicate spec hoặc view | ❌ Chưa |
| **Group by** | gom nhóm | **Mart SQL** | `MATERIALIZED VIEW ... GROUP BY` | ❌ Chưa |
| **Aggregate** | SUM/COUNT/AVG | **Mart SQL** | materialized view + scheduled refresh | ❌ Chưa |
| **Join** | gộp nhiều dw_* | **Mart SQL** | `VIEW ... JOIN` | ❌ Chưa |
| **Custom SQL** | tuỳ ý | **Mart SQL / dbt** | dbt model / view | ❌ Chưa |

## 3. Ba option triển khai tầng Mart (cho filter/groupby/aggregate/join)

### Option A — Postgres Views + Materialized Views (nhẹ nhất, nhanh nhất) ⭐ khuyến nghị khởi đầu
- Tạo schema `reporting` trên goopay_dest.
- `v_*` (VIEW): join/filter realtime trên dw_* (vd `v_payment_enriched`).
- `mv_*` (MATERIALIZED VIEW): aggregate/groupby (vd `mv_txn_daily_summary`), refresh định kỳ.
- Refresh: tái dùng pattern `transmute_schedule` (thêm mode/loại "mart_refresh") hoặc `pg_cron`.
- Metabase đọc `reporting.*` qua read-only role.
- **Ưu**: ship nhanh, không infra mới, Metabase-native, SQL ai cũng đọc được. **Nhược**: quản lý SQL thủ công khi nhiều mart.

### Option B — dbt (data build tool)
- Mô hình SQL versioned, lineage, test, incremental — chuẩn công nghiệp cho mart.
- **Ưu**: scale tốt, có test/lineage. **Nhược**: thêm tooling/CI, learning curve.

### Option C — Mở rộng Go transmuter đọc `transform_spec`
- Wire `transform_type` (filter/aggregate/group_by/join) → sinh SQL trong worker.
- **Ưu**: tái dùng UI binding/schedule sẵn có. **Nhược**: lặp lại việc SQL/dbt đã làm tốt hơn; phức tạp, dễ vỡ; đi ngược model streaming. → chỉ nên cho **filter** (row-skip rẻ ở ingest) chứ không cho aggregate/join.

> **Khuyến nghị**: **A bây giờ** (Metabase-ready nhanh) → tiến hoá sang **B (dbt)** nếu mart phình to. Filter có thể thêm row-skip nhẹ ở transmuter nếu cần loại bớt rác ngay từ ingest.

## 4. Đối soát giao dịch (financial reconciliation)
- Thêm `AmountReconAgent`: `SUM(amount) GROUP BY business_date` — Mongo `$group` (source) vs PG SQL (dest), so khớp.
- Bảng mới `reconciliation_financial` (group_key, sum_src, sum_dst, diff, run_id) hoặc cột thêm.
- Tái dùng scaffolding window/advisory-lock/recon_runs sẵn có.
- Surface qua `reporting.mv_recon_financial_daily` cho Metabase.

## 5. Enable Metabase
- Tạo PG role read-only trên goopay_dest, `GRANT SELECT ON ALL TABLES IN SCHEMA reporting`.
- Deploy Metabase container trỏ **read-replica** (tránh write path live).
- Chỉ expose schema `reporting` (view curated) — KHÔNG cho query thẳng dw_* (tránh full-scan + lộ cột mask).

## 6. Rollout theo phase
- **P0** — Duyệt design này (chọn option Mart A/B/C).
- **P1** — `reporting` schema + read-only role + 2–3 view seed (1 join `v_*`, 1 aggregate `mv_*`) trên dw_* hiện có + deploy Metabase. → **mở khoá BI ngay**.
- **P2** — Nối V3 child-explode → master (array flatten query được) + auto-flatten suggester (introspect `_raw_data` → gợi ý mapping rule).
- **P3** — `AmountReconAgent` + `reconciliation_financial` + view đối soát.
- **P4** (tuỳ chọn) — Orchestration refresh mart (scheduled MV / dbt) + UI `transform_spec`; cân nhắc filter row-skip ở transmuter.

## 7. Quyết định cần User chọn
1. **Hướng tầng Mart**: A (views/MV) ⭐ / B (dbt) / C (mở rộng Go transmuter).
2. **Phạm vi bắt đầu**: chỉ chốt design / làm P1 (Metabase + seed views) ngay / làm cả P1+P3 (BI + đối soát).
3. **Replica cho Metabase**: dùng replica hay chấp nhận trỏ thẳng dest dev.
