# Báo cáo Khắc phục lỗi Postgres Relation Missing trong Partition Dropper

## 1. Thông tin Chung
- **Workspace**: `bug-partition-dropper-relation-missing-2026-05-26`
- **Thời gian thực hiện**: 26/05/2026
- **Vấn đề**: Hàm `sweep` và `backfillFromDefault` của `PartitionDropper` bị lỗi `SQLSTATE 42P01: relation "failed_sync_logs_default" does not exist` khi cố gắng thực thi DDL/DML trực tiếp trên phân vùng mặc định mà không có schema-qualification `cdc_system.`.

---

## 2. Các File Đã Thay Đổi
- **[MODIFY]** [partition_dropper.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/partition_dropper.go)
  - Prefix `"cdc_system".` cho tất cả câu lệnh SQL / DDL thao tác với các bảng phân vùng (`DefaultTable`, `Parent`, `partitionName`).
- **[NEW]** [partition_dropper_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/partition_dropper_test.go)
  - Thêm integration test tự động thiết lập schema `cdc_system`, tạo bảng phân vùng, chèn dữ liệu mồ côi (orphan rows) vào phân vùng mặc định, thực thi `RunOnce` để backfill và sweep dữ liệu cũ. Test tự động dọn dẹp bảng sau khi chạy xong.

---

## 3. Chi tiết Thay Đổi Mã Nguồn (`partition_dropper.go`)

### 3.1. Lệnh DROP TABLE trong hàm `sweep`
```go
// Trước:
ident := `"` + strings.ReplaceAll(r.Tablename, `"`, `""`) + `"`

// Sau:
ident := `"cdc_system"."` + strings.ReplaceAll(r.Tablename, `"`, `""`) + `"`
```

### 3.2. Gom nhóm dữ liệu mồ côi (`bucketSQL`)
```go
// Trước:
bucketSQL := fmt.Sprintf(
    `SELECT DATE_TRUNC('%s', created_at) AS bucket, COUNT(*) AS cnt
       FROM %q
      WHERE created_at IS NOT NULL
      GROUP BY 1
      ORDER BY 1`, truncUnit, rule.DefaultTable,
)

// Sau:
bucketSQL := fmt.Sprintf(
    `SELECT DATE_TRUNC('%s', created_at) AS bucket, COUNT(*) AS cnt
       FROM "cdc_system".%q
      WHERE created_at IS NOT NULL
      GROUP BY 1
      ORDER BY 1`, truncUnit, rule.DefaultTable,
)
```

### 3.3. Trích xuất dữ liệu từ bảng mặc định vào bảng tạm (`drainSQL`)
```go
// Trước:
drainSQL := fmt.Sprintf(
    `CREATE TEMP TABLE _backfill_staging ON COMMIT DROP AS
     WITH deleted AS (
        DELETE FROM %q
         WHERE created_at >= ?
           AND created_at <  ?
        RETURNING *
     )
     SELECT * FROM deleted`,
    rule.DefaultTable,
)

// Sau:
drainSQL := fmt.Sprintf(
    `CREATE TEMP TABLE _backfill_staging ON COMMIT DROP AS
     WITH deleted AS (
        DELETE FROM "cdc_system".%q
         WHERE created_at >= ?
           AND created_at <  ?
        RETURNING *
     )
     SELECT * FROM deleted`,
    rule.DefaultTable,
)
```

### 3.4. Tạo phân vùng con tương ứng (`createSQL`)
```go
// Trước:
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS %q PARTITION OF %q FOR VALUES FROM (%s) TO (%s)`,
    partitionName, rule.Parent, ...
)

// Sau:
createSQL := fmt.Sprintf(
    `CREATE TABLE IF NOT EXISTS "cdc_system".%q PARTITION OF "cdc_system".%q FOR VALUES FROM (%s) TO (%s)`,
    partitionName, rule.Parent, ...
)
```

### 3.5. Chèn lại dữ liệu vào phân vùng con qua parent table (`insertSQL`)
```go
// Trước:
insertSQL := fmt.Sprintf(
    `INSERT INTO %q SELECT * FROM _backfill_staging`,
    rule.Parent,
)

// Sau:
insertSQL := fmt.Sprintf(
    `INSERT INTO "cdc_system".%q SELECT * FROM _backfill_staging`,
    rule.Parent,
)
```

---

## 4. Kết quả Xác Minh
- Chạy integration test `TestPartitionDropper_BackfillAndSweep` trực tiếp trên cổng PostgreSQL local `5433` (được cấu hình mặc định trong test):
  ```bash
  go test -v ./internal/service/... -run TestPartitionDropper_BackfillAndSweep
  ```
- **Kết quả**: `PASS` (Thời gian chạy `0.10s`). Phân vùng con được tạo, dữ liệu mồ côi được dịch chuyển khỏi phân vùng `_default`, và phân vùng cũ bị xóa hoàn toàn khi hết hạn mà không gây lỗi `relation does not exist`.
- Kiểm thử toàn bộ dự án (`go test ./...`): `PASS` toàn bộ các gói dịch vụ.
