# 12 Implementation Plan: Triển khai Nâng cấp Master Tier Hỗ trợ ClickHouse

> **Workspace**: `feat-clickhouse-master-destination`  
> **Tài liệu**: Kế hoạch Triển khai Chi tiết của AI (AI Implementation Plan)  
> **Chủ quản**: Brain (Antigravity) & Staff Engineer  
> **Ngày phê duyệt**: 2026-09-11

---

## I. TỔNG QUAN YÊU CẦU & BỐI CẢNH KỸ THUẬT

### 1.1 Yêu cầu Nghiệp vụ
Nâng cấp tầng **Master Destination** của hệ thống `cdc-system` (`data-hub`) để hỗ trợ **ClickHouse** làm đích lưu trữ phân tích OLAP song song hoặc thay thế cho PostgreSQL Master.
- Cho phép người dùng cấu hình Master Binding trỏ về ClickHouse.
- Tự động sinh DDL Master Table chuẩn ClickHouse sử dụng `ReplacingMergeTree`.
- Tự động chuyển đổi và thực thi batch append dữ liệu từ Shadow Table sang ClickHouse với thông lượng cao.
- Hỗ trợ đối soát (Reconciliation) tính toàn vẹn giữa Shadow (Postgres) và Master (ClickHouse).

### 1.2 Nguyên tắc Triển khai (Engineering Principles)
- **Simplicity First**: Tận dụng cơ chế `ReplacingMergeTree(_version, _deleted)` tự động deduplicate, không sử dụng mutation (`ALTER TABLE UPDATE/DELETE`) làm nghẽn ClickHouse.
- **Minimal Impact**: Không sửa đổi hoặc phá vỡ cấu trúc Transmuter PostgreSQL Master hiện hữu.
- **High Throughput Native TCP**: Sử dụng driver `github.com/ClickHouse/clickhouse-go/v2` qua cổng Native 9000 với cơ chế nén LZ4.

---

## II. BẢN ĐỒ THAY ĐỔI & CÁC BƯỚC NÂNG CẤP HỆ THỐNG

### Bước 1: Thiết lập Hạ tầng ClickHouse Server & Client Driver
1. **Cấu hình Docker Compose**:
   - Thêm container `clickhouse` vào `docker/docker-compose.yml` (Image `clickhouse/clickhouse-server:24.3-alpine`).
   - Mở port `9000` (Native TCP cho Go Worker) và `8123` (HTTP cho DBeaver/Web).
   - Thiết lập volume `clickhouse_data` và database mặc định `cdc_master_dw`.
2. **Xây dựng Client Wrapper `pkgs/clickhouse/client.go`**:
   - Vị trí: `centralized-data-service/pkgs/clickhouse/client.go`.
   - Sử dụng `clickhouse.Open` với connection pooling: MaxOpenConns = 16, Compression = LZ4.
3. **Mở rộng `config.go`**:
   - Bổ sung `ClickHouse Config` vào `AppConfig` để nạp thông số từ `config-local.yml` và `config-production.yml`.

### Bước 2: Xây dựng ClickHouse DDL Generator
1. **Module `clickhouse_ddl_generator.go`**:
   - Vị trí: `centralized-data-service/internal/service/master/clickhouse_ddl_generator.go`.
   - Sinh câu lệnh DDL:
     ```sql
     CREATE DATABASE IF NOT EXISTS cdc_master_dw;

     CREATE TABLE IF NOT EXISTS cdc_master_dw.<table_name> (
         _gpay_id Int64,
         _source_id String,
         _source_ts DateTime64(3, 'UTC'),
         _deleted UInt8 DEFAULT 0,
         _version UInt64,
         <columns...>
     ) ENGINE = ReplacingMergeTree(_version, _deleted)
     ORDER BY (_gpay_id)
     PRIMARY KEY (_gpay_id)
     SETTINGS index_granularity = 8192;
     ```
   - Chuyển đổi kiểu dữ liệu tương ứng:
     * `BIGINT` $\rightarrow$ `Int64`
     * `NUMERIC/DECIMAL` $\rightarrow$ `Decimal(P, S)`
     * `TIMESTAMPTZ` $\rightarrow$ `DateTime64(3, 'UTC')`
     * `VARCHAR/TEXT/JSONB` $\rightarrow$ `String`
     * `BOOLEAN` $\rightarrow$ `Bool`
     * Cột cho phép null $\rightarrow$ `Nullable(T)`
2. **Tích hợp vào `MasterDDLGenerator`**:
   - Kiểm tra `masterRow.EngineType`: Nếu là `clickhouse` thì execute DDL trên ClickHouse connection, nếu là `postgresql` thì giữ nguyên luồng cũ.

### Bước 3: Nâng cấp Transmuter Batch Writer cho ClickHouse
1. **Mở rộng `ConnectionManager`**:
   - Vị trí: `centralized-data-service/internal/service/source/connection_manager.go`.
   - Bổ sung hàm: `GetMasterClickHouseConn(ctx context.Context, key string) (driver.Conn, error)`.
2. **Module `transmuter_clickhouse.go`**:
   - Vị trí: `centralized-data-service/internal/service/master/transmuter_clickhouse.go`.
   - Hàm `bulkInsertClickHouse`:
     * Chuẩn bị câu lệnh `INSERT INTO <db>.<table> (...)`.
     * Sử dụng `conn.PrepareBatch(ctx, sql)`.
     * Duyệt mảng record, gán `_version = uint64(row.SourceTs)` và `_deleted = 0`.
     * Gọi `batch.Send()` gửi toàn bộ block nhị phân nén sang ClickHouse.
   - Hàm `softDeleteClickHouse`:
     * Khi có dòng `_deleted = true` từ Shadow table: Không chạy `DELETE`, mà ghi 1 dòng có `_deleted = 1` và `_version = uint64(row.SourceTs)`.
     * `ReplacingMergeTree` tự động deduplicate và loại bỏ bản ghi khi merge.
3. **Phân nhánh trong `TransmuterModule.processBatch`**:
   - Nhánh `clickhouse`: Gọi `bulkInsertClickHouse` và `softDeleteClickHouse`.
   - Nhánh `postgresql`: Gọi `bulkUpsertMaster` (`ON CONFLICT DO UPDATE`) như hiện tại.

### Bước 4: Tích hợp Đối soát Reconciliation & Healing
1. Mở rộng `internal/service/recon/recon_heal_fetch.go`:
   - Bổ sung adapter query đối soát ClickHouse:
     ```sql
     SELECT 
         count() AS total_rows,
         sum(cityHash64(*)) AS checksum
     FROM cdc_master_dw.<table_name> FINAL
     WHERE _deleted = 0;
     ```
2. So sánh kết quả checksum và count với Shadow Table PostgreSQL (`WHERE _deleted = false`).

---

## III. DEFINITION OF DONE & QUALITY GATES (G1 - G8)

- **(G1) Requirement Traceability**: Đáp ứng đầy đủ FR-01 đến FR-04 và NFR-01 đến NFR-03.
- **(G2) Red → Green Test**: Chứng minh việc ghi bản ghi mới và update cùng `_gpay_id` thì ClickHouse tự động deduplicate qua `FINAL`.
- **(G3) Test Thật**: Chạy test suite `go test -v ./test/internal/service/... -run TestClickHouseMaster` PASS 100%.
- **(G4) Edge-cases**: Xử lý chính xác giá trị NULL (`Nullable`), trường số tiền lớn không làm tròn (`Decimal`), ký tự đặc biệt trong JSONB.
- **(G5) Chống Regression**: Toàn bộ luồng PostgreSQL Master hiện hữu chạy bình thường không có bất kỳ thay đổi nào về hành vi.
- **(G6) Output Correctness**: Đối soát số lượng: Transmute 10,000 dòng từ Shadow sang ClickHouse, câu lệnh `SELECT count() FROM table FINAL WHERE _deleted = 0` trả về chính xác 10,000 dòng.
- **(G7) Adversarial Review**: Đã kiểm tra khả năng chịu tải khi xả batch 50,000 dòng/lần, không bị connection leak hoặc OOM.
- **(G8) Bằng chứng vật lý**: Toàn bộ tài liệu được lưu trữ đầy đủ trong workspace `agent/memory/workspaces/feat-clickhouse-master-destination`.
