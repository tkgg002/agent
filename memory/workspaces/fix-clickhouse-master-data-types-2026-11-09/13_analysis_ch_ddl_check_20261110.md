# 13. Báo cáo Kiểm tra & Đối soát ClickHouse DDL (`goopay_clickhouse.transaction_history_1_test_field`)

- **Workspace:** `fix-clickhouse-master-data-types-2026-11-09`
- **Suffix:** `ch_ddl_check_20261110`
- **Chế độ:** `Audit (Propose-Only)` — Rule #1A, Rule #9, Rule #13

---

## 1. Kết luận Tổng quan (Direct Verdict)

DDL bảng `goopay_clickhouse.transaction_history_1_test_field` (`master_binding.id = 94`) được sinh ra trên ClickHouse là **ĐÚNG CHUẨN KỸ THUẬT** cho bảng đích CDC (`ENGINE = ReplacingMergeTree(_transmuted_at) ORDER BY _gpay_id`) và **KHỚP 100% (33/33 cột nghiệp vụ)** với bảng `Mapping Rules (33)` trên giao diện:
1. **Về kiểu dữ liệu cơ sở (Base Data Type):** Cả 33 cột nghiệp vụ đều đã được chuyển đổi chính xác từ PostgreSQL sang ClickHouse native type (`TEXT` $\rightarrow$ `String`, `TIMESTAMPTZ` $\rightarrow$ `DateTime64(3, 'UTC')`, `NUMERIC` $\rightarrow$ `Decimal(38, 10)`, `BOOLEAN` $\rightarrow$ `UInt8`).
2. **Về wrapper `Nullable(...)` trong DDL ClickHouse:** Trong ClickHouse, mọi cột mặc định là `NOT NULL` (nếu không bọc `Nullable(...)` thì giá trị `NULL` sẽ bị ép thành chuỗi rỗng `""`, `0`, hoặc `1970-01-01`). Do cả 33/33 rule ở `cdc_system.mapping_rule_v2` (`shadow_binding_id = 231`) đều có `is_nullable = true`, hàm `mapToClickHouseColumnType(r.DataType, true)` tại `master_ddl_generator_clickhouse.go:102-140` bọc `Nullable(T)` cho 33 cột nghiệp vụ để bảo toàn ngữ nghĩa `NULL`, trong khi UI hiển thị kiểu cơ sở (`String`, `DateTime64(3, 'UTC')`, `Decimal(38, 10)`, `UInt8`) giống quy ước hiển thị trên PostgreSQL.
3. **Về 5 cột hệ thống & thứ tự cột:** 5 cột đầu tiên (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`, `_transmuted_at`) là cột hệ thống CDC bắt buộc của `ReplacingMergeTree(_transmuted_at)`; thứ tự 33 cột nghiệp vụ trong DDL sắp theo `ORDER BY m.id` (`master_ddl_generator.go:122`), còn trên UI sắp theo bảng chữ cái `ORDER BY m.target_column ASC` (`master_mapping_rule_repo_gorm.go:154`).
4. **Cảnh báo vận hành quan trọng (`F2`):** Bảng `goopay_clickhouse.transaction_history_1_test_field` (`binding #94`) hiện đang có **`0` bản ghi** dù đã bật lịch đồng bộ (`schedule #144`), vì khi tạo binding `#94`, `shadow_binding_id` được gắn vào `231` (`shadow_testcoretrans.transaction_history_1`, trạng thái `ddl_status = 'pending'`, bảng vật lý chưa tồn tại trong PostgreSQL) thay vì `222` (`shadow_testcoretrans.transaction_history`, bảng Shadow đang có dữ liệu thật).

---

## 2. Bảng Đối soát Chi tiết UI (`Mapping Rules 33`) vs ClickHouse DDL (`38 cột = 5 System + 33 Business`)

| Nhóm cột | Số lượng | Trên UI (`Mapping Rules`) | Trong ClickHouse DDL (`SHOW CREATE TABLE`) | Bằng chứng `file:line` & Giải thích kỹ thuật |
| :--- | :---: | :--- | :--- | :--- |
| **Cột hệ thống CDC** (`_gpay_id`, `_source_ts`, `_deleted`, `_updated_at`, `_transmuted_at`) | **5** | Ẩn khỏi bảng Mapping Rules & bộ chọn Index Manager (`isMasterSystemColumn`) | `_gpay_id Int64`, `_source_ts Nullable(Int64)`, `_deleted UInt8 DEFAULT 0`, `_updated_at DateTime64(3, 'UTC') DEFAULT now64(3)`, `_transmuted_at UInt64 DEFAULT ...` | `master_ddl_generator_clickhouse.go:116-122`: `_gpay_id` là `ORDER BY` key nên bắt buộc `NOT NULL` (`Int64`); `_transmuted_at` là version column của `ReplacingMergeTree(_transmuted_at)`. |
| **Nhóm chuỗi (`TEXT`)**: `bill_id`, `channel_id`, `id`, `idempotency_key`, `master_merchant_code`, `merchant_info_email`, `merchant_info_id`, `merchant_info_name`, `noted`, `order_id`, `original_trans_his_id`, `partner_code`, `receiver_credit`, `receiver_customer_id`, `receiver_email`, `receiver_full_name`, `receiver_merchant_code`, `receiver_merchant_id`, `receiver_phone`, `request_id`, `sender_customer_id`, `sender_full_name`, `sender_phone`, `service_code`, `status`, `trans_id`, `trans_type` | **27** | Shadow: `TEXT` $\rightarrow$ Master: `String` | `Nullable(String)` | `master_ddl_generator_clickhouse.go:99-104` & `:140`: `TEXT` $\rightarrow$ baseType `String` $\rightarrow$ bọc `Nullable(String)` để lưu được `NULL` từ nguồn. |
| **Nhóm thời gian (`TIMESTAMPTZ`)**: `created_at`, `updated_at` | **2** | Shadow: `TIMESTAMPTZ` $\rightarrow$ Master: `DateTime64(3, 'UTC')` | `Nullable(DateTime64(3, 'UTC'))` | `master_ddl_generator_clickhouse.go:83-84` & `:140`: `TIMESTAMPTZ` $\rightarrow$ `DateTime64(3, 'UTC')` $\rightarrow$ bọc `Nullable(DateTime64(3, 'UTC'))`. |
| **Nhóm số thập phân (`NUMERIC`)**: `fee`, `value` | **2** | Shadow: `NUMERIC` $\rightarrow$ Master: `Decimal(38, 10)` | `Nullable(Decimal(38, 10))` | `master_ddl_generator_clickhouse.go:77-82` & `:140`: `NUMERIC` $\rightarrow$ `Decimal(38, 10)` $\rightarrow$ bọc `Nullable(Decimal(38, 10))`. |
| **Nhóm logic (`BOOLEAN`)**: `is_delete`, `sys_trans` | **2** | Shadow: `BOOLEAN` $\rightarrow$ Master: `UInt8` | `Nullable(UInt8)` | `master_ddl_generator_clickhouse.go:69-70` & `:140`: `BOOLEAN` $\rightarrow$ `UInt8` $\rightarrow$ bọc `Nullable(UInt8)`. |
| **Skipping Indexes** | **2** | Hiển thị tại bảng Index Manager | `INDEX ix_..._source_ts _source_ts TYPE minmax GRANULARITY 4`, `INDEX ix_..._updated_at _updated_at TYPE minmax GRANULARITY 4` | `master_ddl_generator_clickhouse.go:161-166`: Tạo tự động 2 `minmax` skipping index phục vụ quét cửa sổ đối soát (Reconciliation) và增量 (Delta scan). |

---

## 3. Các Phát hiện Chi tiết (`F1`–`F3`)

### `F1` (`P2 - Design/Consistency` | `VERIFIED`): `generateClickHouseDDL` và `reconcileClickHouseColumn` luôn bọc `Nullable(...)` (`isNullable = true`) cho mọi cột nghiệp vụ
- **Bằng chứng `file:line`:**
  - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:102-105`:
    ```go
    if isNullable {
        return fmt.Sprintf("Nullable(%s)", baseType)
    }
    return baseType
    ```
  - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:140`:
    ```go
    chType := mapToClickHouseColumnType(r.DataType, true)
    ```
  - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator_clickhouse.go:262`:
    ```go
    chType := mapToClickHouseColumnType(dataType, true)
    ```
- **Đánh giá:** Trong ClickHouse, nếu không bọc `Nullable(...)`, cột sẽ là `NOT NULL` và tự động thay `NULL` bằng giá trị rỗng/0 (`""`, `0`, `1970-01-01`). Đồng thời trong `cdc_system.mapping_rule_v2` (`shadow_binding_id = 231`), cả 33 cột của `transaction_history_1` đều có `is_nullable = true`. Do đó việc DDL sinh ra `Nullable(String)`, `Nullable(DateTime64(3, 'UTC'))`, `Nullable(Decimal(38, 10))`, `Nullable(UInt8)` là đúng đắn để bảo toàn dữ liệu `NULL`.

### `F2` (`P1 - Operational/Data Sync` | `VERIFIED`): `goopay_clickhouse.transaction_history_1_test_field` (`binding #94`) đang liên kết với `shadow_binding_id = 231` (`shadow_testcoretrans.transaction_history_1`) chưa được tạo bảng vật lý
- **Bằng chứng Runtime DB (`gpay-postgres-cdc` / `cdc_dw`):**
  - `cdc_system.master_binding` (`id = 94`): `shadow_binding_id = 231`, `master_table = 'transaction_history_1_test_field'`.
  - `cdc_system.shadow_binding` (`id = 231`): `shadow_schema = 'shadow_testcoretrans'`, `shadow_table = 'transaction_history_1'`, `ddl_status = 'pending'`, `to_regclass('shadow_testcoretrans.transaction_history_1') = NULL`.
  - Trong khi đó, bảng Shadow thực tế đang chứa dữ liệu của `transaction_history` là `shadow_binding_id = 222` (`shadow_testcoretrans.transaction_history`, `ddl_status = 'created'`).
- **Hệ quả:** Bảng `goopay_clickhouse.transaction_history_1_test_field` hiện có `0` dòng dữ liệu (trong khi `goopay_clickhouse.transaction_history` có `254,372` dòng).

### `F3` (`P2 - Metadata Persistence` | `VERIFIED`): Khi clone rule từ Shadow sang Master (`cloneAllShadowRulesTx`), cột `mapping_rule_master.data_type` để `NULL` (kế thừa từ `mapping_rule_v2.data_type`)
- **Bằng chứng `file:line`:**
  - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:49-61`: `INSERT INTO cdc_system.mapping_rule_master (master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by)` không ghi đè `m.data_type`.
  - Khi đọc lên UI (`master_mapping_rule_repo_gorm.go:101`) và khi sinh DDL (`master_ddl_generator.go:112`), hệ thống lấy `COALESCE(m.data_type, v2.data_type)` và ánh xạ đồng nhất sang bộ kiểu ClickHouse (`String`, `DateTime64(3, 'UTC')`, `Decimal(38, 10)`, `UInt8`).

---

## 4. Kết quả Kiểm định Decision Gate `DG3` (`laya_gate.py`)

- **Summary File:** `06_laya_summary_dg3_20261110T103354.json`
- **Raw File:** `06_laya_raw_dg3_20261110T103354.json`
- **Gate Verdict:** `PASS` (`trusted_rounds: []`, `UNVALIDATED` — Laya không phân biệt qua ngưỡng control nên kết luận trực tiếp bằng `file:line` & truy vấn DB runtime thực tế theo Rule #10).

| Claim ID | Nội dung kiểm chứng | R1 / R2 / R3 | `evidence_error` | Kết luận |
| :--- | :--- | :--- | :--- | :--- |
| `A-F1-NULLABLE-DDL` | `generateClickHouseDDL` gọi `mapToClickHouseColumnType(r.DataType, true)` bọc `Nullable(...)` cho 33 cột nghiệp vụ | `CORRECT (0.4832)` / `CORRECT (0.4501)` / `YES (0.7985)` | `""` (Khớp `REF:`) | `VERIFIED` (Code + ClickHouse `SHOW CREATE TABLE`) |
| `A-F2-SYSTEM-COLS-AND-ORDER` | 5 cột hệ thống CDC + `ReplacingMergeTree(_transmuted_at) ORDER BY (_gpay_id)` + 2 `minmax` skipping indexes + thứ tự `m.id` vs `m.target_column` | `CORRECT (0.2570)` / `CORRECT (0.1459)` / `YES (0.7475)` | `""` (Khớp `REF:`) | `VERIFIED` (Code + ClickHouse `system.query_log`) |
| `A-F3-CLONE-NULL-DATATYPE` | `cloneAllShadowRulesTx` để `m.data_type = NULL` và kế thừa `v2.data_type` qua `COALESCE` | `CORRECT (0.1277)` / `CORRECT (0.2488)` / `YES (0.3838)` | `""` (Khớp `REF:`) | `VERIFIED` (Code + Postgres `cdc_system.mapping_rule_master`) |

