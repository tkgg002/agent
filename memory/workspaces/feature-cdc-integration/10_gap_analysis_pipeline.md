# Gap Analysis: CDC Pipeline End-to-End (2026-04-15)

> Role: Brain
> Triggered by: User insert record vào MongoDB → data đích không có → không biết kiểm tra ở đâu

## Tình trạng pipeline hiện tại

```
MongoDB ──→ Debezium ──→ NATS JetStream ──→ CDC Worker ──→ PostgreSQL
  ✅           ✅             ✅ (2 msgs)        ❓              ❓
```

### Đã verify
- MongoDB: insert OK
- Debezium: "2 records sent" → publish lên NATS OK
- NATS: stream CDC_EVENTS có 2 messages, 1.2 KiB

### Chưa verify / Có thể hỏng
- **Worker consumer**: Có pull messages từ NATS không? Consumer có bind đúng stream?
- **EventHandler**: Có parse được Debezium JSON format không?
- **DynamicMapper**: Có map fields đúng không?
- **PostgreSQL**: Có insert/upsert thành công không?

## Monitoring THIẾU (xương sống)

| Cần monitor | Hiện có | Gap |
|:------------|:--------|:----|
| Messages vào NATS (per second) | `nats stream info` CLI | Không có dashboard realtime |
| Consumer lag (pending messages) | `nats consumer info` CLI | Không có alert khi lag tăng |
| Worker processing rate | Prometheus metrics (6 metrics) | Không expose lên UI |
| Worker errors | zap.Logger (stdout) | Không aggregate, phải đọc log thủ công |
| DB insert rate | Không có | Hoàn toàn thiếu |
| Pipeline health (end-to-end) | Không có | THIẾU NGHIÊM TRỌNG |

## Root causes cần fix

### 1. Worker consumer có thể không subscribe đúng stream
- Worker tạo stream `CDC_EVENTS` với subjects `cdc.goopay.>`
- Debezium publish lên subjects `cdc.goopay.payment-bill-service.refund-requests`
- Consumer `cdc-worker-group` bind vào stream nào? CDC_EVENTS hay DebeziumStream?
- Nếu consumer bind stream cũ (đã xoá) → không nhận messages

### 2. Debezium JSON format khác EventHandler expect
- Debezium Server gửi format khác Debezium Kafka Connect
- EventHandler parse `model.CDCEvent` — có match không?
- Cần verify: dump 1 message từ NATS, so sánh với CDCEvent struct

### 3. Không có cách nào biết pipeline healthy
- User insert MongoDB → chờ → không thấy Postgres → không biết lỗi ở đâu
- CẦN: health check endpoint kiểm tra mỗi tầng
- CẦN: Activity Log ghi mỗi event processed (hiện chỉ ghi bridge/transform)

## Khuyến nghị

### Ngắn hạn (fix ngay)
1. Dump 1 NATS message → verify format match CDCEvent struct
2. Check Worker consumer binding
3. Thêm log chi tiết vào EventHandler (nhận event, parse, map, insert)

### Trung hạn (monitoring)
1. NATS monitoring page trong CMS (stream info, consumer lag, messages/sec)
2. Pipeline health endpoint (`GET /api/pipeline/health`) — check mỗi tầng
3. Activity Log ghi mỗi CDC event processed (không chỉ bridge/transform)

### Dài hạn (production)
1. Grafana dashboard (Prometheus NATS exporter + Worker metrics)
2. Alert khi consumer lag > threshold
3. Alert khi pipeline throughput drop
