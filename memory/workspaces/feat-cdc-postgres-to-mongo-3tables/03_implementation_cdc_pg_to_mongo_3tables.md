# 03 Technical Implementation Design: CDC PostgreSQL → MongoDB & Denormalization 3 Bảng

> **Workspace**: `feat-cdc-postgres-to-mongo-3tables`  
> **Dự án**: `centralized-data-service`, `cdc-cms-service`, Kafka Connect

---

## 1. Kiến trúc Thành phần (Component Architecture)

```
[PostgreSQL Database (Source)]
   ├── orders (Root/Parent)
   ├── order_items (Child 1:N)
   └── order_payments (Child 1:1)
         │
         │ (WAL Logical Replication - pgoutput)
         ▼
[Debezium PostgreSQL Source Connector]
   ├── SMT ByLogicalTableRouter: Reroute 3 bảng -> 1 topic `cdc.pg.unified.orders`
   ├── SMT ExtractNewRecordState: delete.handling.mode = rewrite (giữ key khi delete)
   └── Key Partitioning: Hash theo `order_id`
         │
         ▼
[Kafka Cluster / 1 Topic Duy Nhất]
   └── Topic: `cdc.pg.unified.orders`
       (Mọi event của 1 order_id nằm trên CÙNG 1 PARTITION -> Strict FIFO tuyệt đối)
         │
         ▼
[Stateless CDC Worker (Go Service in centralized-data-service)]
   ├── 1. Natural Kafka Batch Fetch (Tối đa 500 records / batch)
   ├── 2. BSON Type Conversion (NUMERIC -> Decimal128, TIMESTAMPTZ -> ISODate)
   ├── 3. Build Ordered mongo.WriteModel List:
   │      - orders: Soft-delete khi 'd' / Upsert khi 'c','u'
   │      - order_items: Pull-then-Push với _v per-item (Upsert: false)
   │      - order_payments: Set payment subdocument (Upsert: false)
   ├── 4. Atomic Execution: mongoCollection.BulkWrite(ctx, models, SetOrdered(true))
   └── 5. Commit Kafka Offset
         │
         ▼
[MongoDB Replica Set (Target)]
   └── Collection: `orders`
```

---

## 2. Cấu hình Debezium SMT Định tuyến 1 Topic

Tệp: `centralized-data-service/deployments/debezium/pg-orders-to-mongo-connector.json`

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
    "publication.autocreate.mode": "filtered",
    "table.include.list": "public.orders,public.order_items,public.order_payments",
    "decimal.handling.mode": "string",
    "time.precision.mode": "adaptive_time_microseconds",
    "snapshot.mode": "never",
    "heartbeat.interval.ms": "5000",

    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": "false",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false",

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

## 3. Go Ingest Worker Logic (Stateless Ordered BulkWrite)

Tệp: `centralized-data-service/internal/sinkworker/mongo_aggregator_worker.go`

```go
package sinkworker

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type CDCEvent struct {
	Table      string
	Op         string // "c", "u", "d", "r"
	OrderID    string
	SourceTS   time.Time
	SourceTSMs int64
	Payload    map[string]any
}

func ProcessBatch(ctx context.Context, coll *mongo.Collection, events []CDCEvent, logger *zap.Logger) error {
	if len(events) == 0 {
		return nil
	}

	var writeModels []mongo.WriteModel

	for _, ev := range events {
		switch ev.Table {
		case "orders":
			if ev.Op == "d" {
				m := mongo.NewUpdateOneModel().
					SetFilter(bson.M{"_id": ev.OrderID}).
					SetUpdate(bson.M{
						"$set": bson.M{
							"is_deleted":                   true,
							"deleted_at":                   time.Now().UTC(),
							"_cdc_metadata.last_source_ts": ev.SourceTS,
						},
					})
				writeModels = append(writeModels, m)
			} else {
				totalAmountDec, _ := parseDecimal(ev.Payload["total_amount"])
				updatedAt := parseTime(ev.Payload["updated_at"])
				createdAt := parseTime(ev.Payload["created_at"])

				m := mongo.NewUpdateOneModel().
					SetFilter(bson.M{"_id": ev.OrderID}).
					SetUpdate(bson.M{
						"$set": bson.M{
							"order_code":                   ev.Payload["order_code"],
							"status":                       ev.Payload["status"],
							"total_amount":                 totalAmountDec,
							"updated_at":                   updatedAt,
							"is_deleted":                   false,
							"_cdc_metadata.versions.order": ev.SourceTSMs,
							"_cdc_metadata.last_source_ts": ev.SourceTS,
						},
						"$setOnInsert": bson.M{
							"created_at": createdAt,
							"items":      bson.A{},
						},
					}).
					SetUpsert(true)
				writeModels = append(writeModels, m)
			}

		case "order_items":
			itemID := parseBigInt(ev.Payload["id"])
			if ev.Op == "d" {
				m := mongo.NewUpdateOneModel().
					SetFilter(bson.M{
						"_id":        ev.OrderID,
						"is_deleted": bson.M{"$ne": true},
					}).
					SetUpdate(bson.M{
						"$pull": bson.M{"items": bson.M{"id": itemID}},
						"$set":  bson.M{"_cdc_metadata.last_source_ts": ev.SourceTS},
					})
				writeModels = append(writeModels, m)
			} else {
				priceDec, _ := parseDecimal(ev.Payload["price"])
				itemDoc := bson.M{
					"id":         itemID,
					"product_id": ev.Payload["product_id"],
					"quantity":   parseInt(ev.Payload["quantity"]),
					"price":      priceDec,
					"_v":         ev.SourceTSMs,
				}

				writeModels = append(writeModels,
					mongo.NewUpdateOneModel().
						SetFilter(bson.M{
							"_id":        ev.OrderID,
							"is_deleted": bson.M{"$ne": true},
						}).
						SetUpdate(bson.M{"$pull": bson.M{"items": bson.M{"id": itemID}}}),
					mongo.NewUpdateOneModel().
						SetFilter(bson.M{
							"_id":        ev.OrderID,
							"is_deleted": bson.M{"$ne": true},
						}).
						SetUpdate(bson.M{
							"$push": bson.M{"items": itemDoc},
							"$set":  bson.M{"_cdc_metadata.last_source_ts": ev.SourceTS},
						}),
				)
			}

		case "order_payments":
			if ev.Op == "d" {
				m := mongo.NewUpdateOneModel().
					SetFilter(bson.M{
						"_id":        ev.OrderID,
						"is_deleted": bson.M{"$ne": true},
					}).
					SetUpdate(bson.M{
						"$unset": bson.M{"payment": ""},
						"$set":   bson.M{"_cdc_metadata.last_source_ts": ev.SourceTS},
					})
				writeModels = append(writeModels, m)
			} else {
				amountDec, _ := parseDecimal(ev.Payload["amount"])
				paymentDoc := bson.M{
					"id":               parseBigInt(ev.Payload["id"]),
					"payment_method":   ev.Payload["payment_method"],
					"amount":           amountDec,
					"status":           ev.Payload["status"],
					"transaction_code": ev.Payload["transaction_code"],
					"_v":               ev.SourceTSMs,
				}
				m := mongo.NewUpdateOneModel().
					SetFilter(bson.M{
						"_id":        ev.OrderID,
						"is_deleted": bson.M{"$ne": true},
					}).
					SetUpdate(bson.M{
						"$set": bson.M{
							"payment":                      paymentDoc,
							"_cdc_metadata.last_source_ts": ev.SourceTS,
						},
					})
				writeModels = append(writeModels, m)
			}
		}
	}

	if len(writeModels) == 0 {
		return nil
	}

	opts := options.BulkWrite().SetOrdered(true)
	_, err := coll.BulkWrite(ctx, writeModels, opts)
	if err != nil {
		logger.Error("mongo bulk write failed", zap.Error(err))
		return fmt.Errorf("mongo bulk write: %w", err)
	}
	return nil
}
```
