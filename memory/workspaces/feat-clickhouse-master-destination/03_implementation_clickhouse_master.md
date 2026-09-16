# 03 Technical Implementation Design: ClickHouse Master Destination

> **Workspace**: `feat-clickhouse-master-destination`

---

## 1. Thiết kế DDL Generator ClickHouse

Khi `masterRow.EngineType == "clickhouse"`, hệ thống sinh câu lệnh DDL theo chuẩn:

```sql
CREATE DATABASE IF NOT EXISTS cdc_master_dw;

CREATE TABLE IF NOT EXISTS cdc_master_dw.orders (
    _gpay_id Int64,
    _source_id String,
    _source_ts DateTime64(3, 'UTC'),
    _deleted UInt8 DEFAULT 0,
    _version UInt64,
    order_code String,
    customer_id String,
    status LowCardinality(String),
    total_amount Decimal(18, 4),
    created_at DateTime64(3, 'UTC'),
    updated_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(_version, _deleted)
ORDER BY (_gpay_id)
PRIMARY KEY (_gpay_id)
SETTINGS index_granularity = 8192;
```

---

## 2. Thiết kế Module Batch Transmute cho ClickHouse

Tệp: `centralized-data-service/internal/service/master/transmuter_clickhouse.go`

```go
package master

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// bulkInsertClickHouse thực hiện append batch vào ClickHouse ReplacingMergeTree
func (t *TransmuterModule) bulkInsertClickHouse(
	ctx context.Context, 
	conn driver.Conn, 
	database, table string, 
	records []map[string]any,
) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	keys := sortedKeysAny(records[0])
	query := fmt.Sprintf("INSERT INTO %s.%s (%s)", database, table, strings.Join(keys, ", "))

	batch, err := conn.PrepareBatch(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("prepare clickhouse batch: %w", err)
	}

	for _, rec := range records {
		vals := make([]any, len(keys))
		for i, k := range keys {
			vals[i] = rec[k]
		}
		if err := batch.Append(vals...); err != nil {
			return 0, fmt.Errorf("append to clickhouse batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return 0, fmt.Errorf("send clickhouse batch: %w", err)
	}

	return int64(len(records)), nil
}

// softDeleteClickHouse ghi nhận các dòng bị xóa bằng cách gán _deleted = 1
func (t *TransmuterModule) softDeleteClickHouse(
	ctx context.Context,
	conn driver.Conn,
	database, table string,
	gpayIDs []int64,
	sourceTs int64,
) error {
	if len(gpayIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf("INSERT INTO %s.%s (_gpay_id, _deleted, _version, _source_ts)", database, table)
	batch, err := conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, id := range gpayIDs {
		if err := batch.Append(id, uint8(1), uint64(sourceTs), now); err != nil {
			return err
		}
	}

	return batch.Send()
}
```

---

## 3. Bản đồ Ánh xạ Kiểu dữ liệu (Data Type Mapping Matrix)

| MappingRule DataType / Postgres Type | ClickHouse Target Data Type | Ghi chú |
|:---|:---|:---|
| `BIGINT`, `INT8` | `Int64` | |
| `INTEGER`, `INT`, `INT4` | `Int32` | |
| `SMALLINT`, `INT2` | `Int16` | |
| `NUMERIC`, `DECIMAL(P, S)` | `Decimal(P, S)` | Ví dụ `Decimal(18, 4)` |
| `TIMESTAMPTZ`, `TIMESTAMP` | `DateTime64(3, 'UTC')` | Chính xác tới mili-giây |
| `VARCHAR`, `TEXT`, `CHAR` | `String` | ClickHouse tối ưu string nén rất cao |
| `BOOLEAN`, `BOOL` | `Bool` (hoặc `UInt8`) | |
| `FLOAT`, `REAL` | `Float32` | |
| `DOUBLE PRECISION` | `Float64` | |
| `JSONB`, `JSON` | `String` | Lưu chuỗi JSON nguyên bản |
| Status / Enum ngắn | `LowCardinality(String)` | Tăng tốc độ lọc và nén từ điển |
| Cột có `is_nullable = true` | `Nullable(T)` | Cho phép giá trị NULL |
