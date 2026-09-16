# 09 Technical Solutions: CDC PostgreSQL → MongoDB & Multi-Table Aggregation

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Mục tiêu**: Hồ sơ giải pháp kỹ thuật chi tiết cho từng task, cung cấp mã nguồn mẫu chuẩn xác và cách cấu hình

---

## 1. Giải pháp cho TASK-02: Cấu hình Debezium SMT Định tuyến 1 Topic

### Vấn đề:
Khi 3 bảng đẩy vào 3 topic riêng, Kafka không đảm bảo tính tuần tự giữa các bảng, dẫn đến bảng con được xử lý trước bảng cha, gây sai lệch dữ liệu.

### Giải pháp Kỹ thuật:
Cấu hình Debezium với SMT `ByLogicalTableRouter` để hợp nhất 3 bảng thành 1 topic duy nhất `cdc.pg.unified.orders` và định vị key là `order_id`:

```json
{
  "name": "cdc-pg-orders-to-mongo",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "tasks.max": "1",
    "database.hostname": "postgres-source",
    "database.port": "5432",
    "database.user": "cdc_user",
    "database.password": "${file:/secrets.properties:pg_pass}",
    "database.dbname": "order_db",
    "topic.prefix": "cdc.pg",
    "plugin.name": "pgoutput",
    "slot.name": "cdc_pg_orders_unified_slot",
    "publication.name": "cdc_orders_pub",
    "table.include.list": "public.orders,public.order_items,public.order_payments",
    "decimal.handling.mode": "string",
    "time.precision.mode": "adaptive_time_microseconds",
    "snapshot.mode": "never",

    "transforms": "reroute,unwrap",
    
    "transforms.reroute.type": "io.debezium.transforms.ByLogicalTableRouter",
    "transforms.reroute.topic.regex": ".*orders.*|.*order_items.*|.*order_payments.*",
    "transforms.reroute.topic.replacement": "cdc.pg.unified.orders",
    "transforms.reroute.key.field.name": "order_id",

    "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
    "transforms.unwrap.drop.tombstones": "false",
    "transforms.unwrap.delete.handling.mode": "rewrite"
  }
}
```

---

## 2. Giải pháp cho TASK-04: BSON Type Converter

Tệp: `centralized-data-service/internal/sinkworker/mongo_type_converter.go`

```go
package sinkworker

import (
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseDecimal(val any) (primitive.Decimal128, error) {
	if val == nil {
		return primitive.NewDecimal128(0, 0), nil
	}
	switch v := val.(type) {
	case string:
		return primitive.ParseDecimal128(v)
	case float64:
		return primitive.ParseDecimal128(strconv.FormatFloat(v, 'f', -1, 64))
	case int64:
		return primitive.ParseDecimal128(strconv.FormatInt(v, 10))
	default:
		return primitive.ParseDecimal128(fmt.Sprintf("%v", val))
	}
}

func parseTime(val any) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		return v.UTC()
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t.UTC()
		}
		t, err = time.Parse("2006-01-02 15:04:05.999999-07", v)
		if err == nil {
			return t.UTC()
		}
	case int64: // epoch microseconds or milliseconds
		if v > 1e15 { // microseconds
			return time.UnixMicro(v).UTC()
		} else if v > 1e11 { // milliseconds
			return time.UnixMilli(v).UTC()
		}
		return time.Unix(v, 0).UTC()
	case float64:
		return time.UnixMilli(int64(v)).UTC()
	}
	return time.Time{}
}

func parseBigInt(val any) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	}
	return 0
}

func parseInt(val any) int32 {
	return int32(parseBigInt(val))
}
```

---

## 3. Giải pháp cho TASK-05: Idempotent Pull-then-Push & Zombie Prevention

### Chi tiết Kỹ thuật:
- **Ngăn chặn Zombie Document**: Khi một bản ghi trong `order_items` được insert/update, điều kiện filter BẮT BUỘC phải kèm `is_deleted: { $ne: true }` và **`Upsert: false`**.
- **Cơ chế Pull-then-Push**:
  Thay vì dùng `$push` trực tiếp gây trùng lặp phần tử khi retry message, chia thành 2 operation liên tiếp trong `BulkWrite`:
  1. `$pull`: Xóa item cũ có cùng `id`.
  2. `$push`: Đưa item mới vào mảng kèm version `_v` của item đó.
  Vì `BulkWrite` được cấu hình `SetOrdered(true)`, hai lệnh này được chạy theo đúng thứ tự trên MongoDB engine.

---

## 4. Giải pháp cho TASK-07: SQL Snapshot Tránh Cartesian Product

```sql
SELECT 
    o.id AS order_id,
    o.order_code,
    o.customer_id,
    o.status,
    o.total_amount,
    o.created_at,
    o.updated_at,
    COALESCE(i.items_json, '[]'::jsonb) AS items,
    p.payment_json AS payment
FROM orders o
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object(
        'id', id, 
        'product_id', product_id, 
        'quantity', quantity, 
        'price', unit_price,
        '_v', extract(epoch from updated_at)::bigint * 1000
    )) AS items_json
    FROM order_items 
    WHERE order_id = o.id
) i ON TRUE
LEFT JOIN LATERAL (
    SELECT jsonb_build_object(
        'id', id, 
        'payment_method', payment_method, 
        'amount', amount, 
        'status', status,
        'transaction_code', transaction_code,
        '_v', extract(epoch from updated_at)::bigint * 1000
    ) AS payment_json
    FROM order_payments
    WHERE order_id = o.id
    ORDER BY id DESC LIMIT 1
) p ON TRUE
WHERE o.id >= $1 AND o.id < $2;
```
