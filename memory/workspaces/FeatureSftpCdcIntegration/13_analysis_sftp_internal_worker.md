# 13_analysis_sftp_internal_worker.md — Phân tích Kiến trúc (AI Analysis)

## Phase: SFTP Internal Polling Worker  
**Ngày:** 2026-08-11

---

## 1. Phân tích Luồng Hiện tại

### Transport Layer (NATS vs Kafka)

Hệ thống `centralized-data-service` có **2 transport paths** song song:

**Path 1 — NATS JetStream (primary):**
```
NATS topic (cdcPrefixes.>) → ConsumerPool → EventHandler.Handle()
```
- `ConsumerPool` subcribe NATS subjects cho từng `topicPrefix`.
- Được khởi tạo tại `server_setup.go` L199-209.

**Path 2 — Kafka Direct (secondary, for Debezium topics):**
```
Kafka topics (cdc.gpaylocal.*) → KafkaConsumer → EventHandler.HandleRaw()
```
- Được khởi tạo khi `cfg.Kafka.Enabled == true` tại L457-474.
- SFTP topics cần đi qua Kafka (vì Kafka Connect SFTP là công cụ push vào Kafka).

### Quyết định: SFTP Worker → Kafka, không phải NATS

Lý do SFTP Worker push Kafka thay vì NATS:
1. Pattern của hệ thống: Kafka Connect → Kafka topic → KafkaConsumer (Debezium đang dùng).
2. `EventHandler.HandleRaw()` với `isSFTP` detection đã được viết cho Kafka path.
3. Kafka producer (kafka-go) đã sẵn sàng trong go.mod.

**Nếu cần push NATS:** Đơn giản hơn (dùng `natsClient.Conn.Publish`), nhưng cần thêm SFTP prefix vào `cdcPrefixes` để ConsumerPool subscribe. Sẽ là Phase 2 optimization nếu cần.

---

## 2. Phân tích Config Architecture

`SFTPWorkerConfig` được khai báo trong `config/config.go` (không phải trong `handler/shadow/`) vì:
- Tuân theo pattern của `KafkaConsumerConfig`, `WorkerConfig` — tất cả config nằm trong package `config`.
- `server_setup.go` đọc `cfg.SFTPWorker` và truyền vào constructor.
- `SFTPWorkerConfig` trong `handler/shadow/sftp_worker.go` là type alias hoặc struct riêng của layer handler (để tránh circular import).

**Giải pháp sạch nhất:** Khai báo `SFTPWorkerConfig` trong cả 2 nơi:
- `config/config.go`: type mapping từ YAML.
- `handler/shadow/sftp_worker.go`: internal struct dùng trong handler layer.
- Trong `server_setup.go`: convert/cast giữa 2 types.

Hoặc đơn giản hơn: Khai báo 1 lần ở `config/config.go`, import vào `server_setup.go`, truyền trực tiếp vào `NewSFTPPollingWorker(cfg config.SFTPWorkerConfig, ...)`.

---

## 3. Phân tích Rủi ro Naming

Topic name `sftp.reconcile.final` — `EventHandler.HandleRaw()` detect:
```go
isSFTP := strings.HasPrefix(subject, "sftp.") || strings.Contains(subject, ".sftp.")
```
Với topic `sftp.reconcile.final`, `strings.HasPrefix` = TRUE ✅

Parse db/table:
```go
parts := strings.Split(subject, ".") // ["sftp", "reconcile", "final"]
db = parts[0]    // "sftp"
table = strings.Join(parts[1:], "_") // "reconcile_final"
```

`ResolveSourceRoutes("sftp", "reconcile_final")` sẽ lookup registry với key `"sftp|reconcile_final"` — cần đảm bảo seed SQL đã đăng ký `source_database='sftp'` và `source_object_name='reconcile_final'` ✅ (đã có trong `sftp_reconcile_final_seed.sql`).
